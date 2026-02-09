package ankylogo

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiterMiddleware returns a gin middleware that rate limits per IP
// using both a sliding window and a token bucket.
func RateLimiterMiddleware(store RateLimiterStore) gin.HandlerFunc {

	return func(c *gin.Context) {
		ip := c.ClientIP()

		var allowedWindow bool = store.AllowedSlidingWindow(ip, 60, 100)

		if !allowedWindow {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			return
		}

		var allowedBucket bool = store.AllowedTokenBucket(ip, 10, 1, time.Second)

		if !allowedBucket {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			return
		}
		c.Next()
	}
}
