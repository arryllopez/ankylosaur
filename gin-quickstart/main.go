package main

import (
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
	// https://gobyexample.com/tickers
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

// RateLimiter middleware using token bucket algorithm.
// Capacity 10, refills 1 token per second.
func RateLimiter() gin.HandlerFunc {
	tb := NewTokenBucket(10, 1, time.Second)

	return func(c *gin.Context) {
		if allowed := tb.TakeTokens(); !allowed {
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
	router.Use(RateLimiter())

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
