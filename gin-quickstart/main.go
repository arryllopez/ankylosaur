package main

import (
	"container/list"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// TokenBucket implements the token bucket rate limiting algorithm.
// Tokens refill at a fixed interval. Each request consumes one token.
// When the bucket is empty, requests are rejected.
type TokenBucket struct {
	tokens       int
	capacity     int
	refillRate   time.Duration
	stopRefiller chan struct{}
	mu           sync.Mutex
}

func NewTokenBucket(capacity, tokensPerInterval int, refillRate time.Duration) *TokenBucket {
	tb := &TokenBucket{
		capacity:     capacity,
		refillRate:   refillRate,
		stopRefiller: make(chan struct{}),
	}
	tb.tokens = capacity
	go tb.refillTokens(tokensPerInterval)
	return tb
}

func (tb *TokenBucket) refillTokens(tokensPerInterval int) {
	ticker := time.NewTicker(tb.refillRate)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tb.mu.Lock()
			if tb.tokens+tokensPerInterval <= tb.capacity {
				tb.tokens += tokensPerInterval
			} else {
				tb.tokens = tb.capacity
			}
			tb.mu.Unlock()
		case <-tb.stopRefiller:
			return
		}
	}
}

func (tb *TokenBucket) TakeTokens() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if tb.tokens > 0 {
		tb.tokens--
		return true
	}
	return false
}

func (tb *TokenBucket) StopRefiller() {
	close(tb.stopRefiller)
}

// SlidingWindowLimiter implements the sliding window log algorithm.
// Tracks individual request timestamps in a deque and rejects
// requests when the count within the window exceeds the limit.
type RateLimiter interface {
	Allow() bool
}

var _ RateLimiter = (*SlidingWindowLimiter)(nil)

type SlidingWindowLimiter struct {
	window int64
	limit  int
	logs   *list.List
	mutex  sync.Mutex
}

func NewSlidingWindowLimiter(window int64, limit int) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		window: window,
		limit:  limit,
		logs:   list.New(),
	}
}

func (sw *SlidingWindowLimiter) Allow() bool {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()

	now := time.Now()
	delta := now.Unix() - sw.window
	edgeTime := time.Unix(delta, 0)

	for sw.logs.Len() > 0 {
		front := sw.logs.Front()
		if front.Value.(time.Time).Before(edgeTime) {
			sw.logs.Remove(front)
		} else {
			break
		}
	}

	if sw.logs.Len() < sw.limit {
		sw.logs.PushBack(now)
		return true
	}

	return false
}

// RateLimiterMiddleware applies per-IP rate limiting using both
// a sliding window (sustained rate) and token bucket (burst control).
func RateLimiterMiddleware() gin.HandlerFunc {
	var bucketPerIp sync.Map
	var slidingWindowPerIp sync.Map

	return func(c *gin.Context) {
		ip := c.ClientIP()

		value, okBucket := bucketPerIp.Load(ip)
		window, okWindow := slidingWindowPerIp.Load(ip)

		var slidingWindow *SlidingWindowLimiter
		if okWindow {
			slidingWindow = window.(*SlidingWindowLimiter)
		} else {
			slidingWindow = NewSlidingWindowLimiter(60, 100)
			slidingWindowPerIp.Store(ip, slidingWindow)
		}
		if allowed := slidingWindow.Allow(); !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			return
		}

		var bucket *TokenBucket
		if okBucket {
			bucket = value.(*TokenBucket)
		} else {
			bucket = NewTokenBucket(10, 1, time.Second)
			bucketPerIp.Store(ip, bucket)
		}
		if allowed := bucket.TakeTokens(); !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			return
		}

		c.Next()
	}
}

func main() {
	router := gin.Default()
	router.Use(RateLimiterMiddleware())

	router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))
	router.Use(gin.Recovery())

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	router.POST("/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "login successful",
		})
	})

	router.GET("/search", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "search completed",
		})
	})

	router.POST("/purchase", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "purchase successful",
		})
	})

	router.Run()
}
