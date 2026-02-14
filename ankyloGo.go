package ankylogo

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Config struct {
	// sliding wzndow
	Window int64
	Limit  int
	// token bucket
	Capacity          int
	TokensPerInterval int
	RefillRate        time.Duration
	// kafka
	EventPublisher EventPublisher
}

func DefaultConfig() Config {
	return Config{
		Window:            60,
		Limit:             100,
		Capacity:          10,
		TokensPerInterval: 1,
		RefillRate:        time.Second,
	}
}

// RateLimiterMiddleware returns a gin middleware that rate limits per IP
// using both a sliding window and a token bucket.
func RateLimiterMiddleware(store RateLimiterStore, config Config, endpointPolicies ...map[string]Config) gin.HandlerFunc {
	if config.Window == 0 && config.Limit == 0 && config.Capacity == 0 {
		log.Println("warning: no rate limiting configured, all requests will pass through")
	}

	return func(c *gin.Context) {
		ip := c.ClientIP()

		// Build key from method + path: "POST /login", "GET /search"
		key := c.Request.Method + " " + c.FullPath()

		// Check if this endpoint has a specific policy
		activeConfig := config // default fallback
		var policies map[string]Config
		if len(endpointPolicies) > 0 {
			policies = endpointPolicies[0]
		}
		if policies != nil {
			if policy, exists := policies[key]; exists {
				activeConfig = policy
			}
		}

		// Then use activeConfig instead of config for the rate limit checks

		if activeConfig.Window > 0 && activeConfig.Limit > 0 {
			var allowedWindow bool = store.AllowedSlidingWindow(ip, activeConfig.Window, activeConfig.Limit)

			if !allowedWindow {
				if config.EventPublisher != nil {
					config.EventPublisher.Publish(RateLimitEvent{
						IP:        ip,
						Endpoint:  key,
						Action:    "DENIED_WINDOW",
						Timestamp: time.Now().UnixNano(),
					})
				}

				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error": "Too many requests. Please try again later.",
				})
				return
			}
		}

		if activeConfig.Capacity > 0 {
			var allowedBucket bool = store.AllowedTokenBucket(ip, activeConfig.Capacity, activeConfig.TokensPerInterval, activeConfig.RefillRate)

			if !allowedBucket {
				if config.EventPublisher != nil {
					config.EventPublisher.Publish(RateLimitEvent{
						IP:        ip,
						Endpoint:  key,
						Action:    "DENIED_BUCKET",
						Timestamp: time.Now().UnixNano(),
					})
				}

				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error": "Too many requests. Please try again later.",
				})
				return
			}
		}

		c.Next()
	}
}
