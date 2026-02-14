package ankylogo

import (
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

		var allowedWindow bool = store.AllowedSlidingWindow(ip, activeConfig.Window, activeConfig.Limit)

		if !allowedWindow {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			return
		}

		var allowedBucket bool = store.AllowedTokenBucket(ip, activeConfig.Capacity, activeConfig.TokensPerInterval, activeConfig.RefillRate)

		if !allowedBucket {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			return
		}

		c.Next()
	}
}
