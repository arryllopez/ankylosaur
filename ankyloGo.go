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
func RateLimiterMiddleware(store RateLimiterStore, config Config) gin.HandlerFunc {

	return func(c *gin.Context) {
		ip := c.ClientIP()

		var allowedWindow bool = store.AllowedSlidingWindow(ip, config.Window, config.Limit)

		if !allowedWindow {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			return
		}

		var allowedBucket bool = store.AllowedTokenBucket(ip, config.Capacity, config.TokensPerInterval, config.RefillRate)

		if !allowedBucket {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			return
		}
		c.Next()
	}
}
