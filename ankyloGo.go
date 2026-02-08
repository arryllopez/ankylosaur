package ankylogo

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiterMiddleware returns a gin middleware that rate limits per IP
// using both a sliding window and a token bucket.
func RateLimiterMiddleware() gin.HandlerFunc {
	// Hash map for buckets and IP (ip : buckets)
	var bucketPerIp sync.Map
	// Hash map for sliding window and IP (ip : sliding Window)
	var slidingWindowPerIp sync.Map

	return func(c *gin.Context) {
		//load the ip
		ip := c.ClientIP()
		// Look up the bucket for the ip (value = bucket, okBucket = found or not with ip)
		value, okBucket := bucketPerIp.Load(ip)
		// Look up the sliding window for the ip (value = window, okWindow = found or not with ip)
		window, okWindow := slidingWindowPerIp.Load(ip)
		// type assert both bucket and sliding window since sync maps return any
		var bucket *TokenBucket
		var slidingWindow *SlidingWindowLimiter

		if okWindow {
			slidingWindow = window.(*SlidingWindowLimiter)
		} else {
			slidingWindow = NewSlidingWindowLimiter(60, 100)
			slidingWindowPerIp.Store(ip, slidingWindow)
		}
		if allowedWindow := slidingWindow.Allow(); !allowedWindow {
			// if no tokens are available, we return 429 status code
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			return
		}
		if okBucket {
			bucket = value.(*TokenBucket)

		} else {
			bucket = NewTokenBucket(10, 1, time.Second)
			bucketPerIp.Store(ip, bucket)
		}
		if allowedBucket := bucket.TakeTokens(); !allowedBucket {
			// if no tokens are available, we return 429 status code
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			return
		}
		c.Next()
	}
}
