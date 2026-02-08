package ankylogo

import (
	"container/list"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// The following code block implements the Token Bucket Algoritm
type TokenBucket struct {
	tokens       int
	capacity     int
	refillRate   time.Duration
	stopRefiller chan struct{} //signal to stop refilling
	mu           sync.Mutex    // handling race conditions (two processes trying to access tokens simultaneously)
}

func NewTokenBucket(capacity, tokensPerInterval int, refillRate time.Duration) *TokenBucket {
	tb := &TokenBucket{
		capacity:     capacity,
		refillRate:   refillRate,
		stopRefiller: make(chan struct{}),
	}
	go tb.refillTokens(tokensPerInterval) // start with a full bucket
	return tb
}

func (tb *TokenBucket) refillTokens(tokensPerInterval int) {
	// ticker is a great way to do something repeatedly to know more
	ticker := time.NewTicker(tb.refillRate)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// handle race conditions
			tb.mu.Lock()
			if tb.tokens+tokensPerInterval <= tb.capacity {
				// if we won't exceed the capacity add tokensPerInterval
				// tokens into our bucket
				tb.tokens += tokensPerInterval
			} else {
				// as we cant add more than capacity tokens, set
				// current tokens to bucket's capacity
				tb.tokens = tb.capacity
			}
			tb.mu.Unlock()
		case <-tb.stopRefiller:
			// let's stop refilling
			return
		}
	}
}

func (tb *TokenBucket) TakeTokens() bool {
	// handle race conditions
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// if there are tokens available in the bucket, we take one out
	// in this case request goes through, thus we return true.
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}
	// in the case where tokens are unavailable, this request won't
	// go through, so we return false
	return false
}

func (tb *TokenBucket) StopRefiller() {
	// close the channel
	close(tb.stopRefiller)
}

// The following block implements the sliding window algorithm

type RateLimiter interface {
	Allow() bool
}

var _ RateLimiter = (*SlidingWindowLimiter)(nil)

type SlidingWindowLimiter struct {
	window int64
	limit  int
	logs   *list.List // deque // push_back, push_front -> in O(1)
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

	// Remove outdated logs
	for sw.logs.Len() > 0 {
		front := sw.logs.Front()
		if front.Value.(time.Time).Before(edgeTime) {
			sw.logs.Remove(front)
		} else {
			break
		}
	}

	// Check if we can accept the request
	if sw.logs.Len() < sw.limit {
		sw.logs.PushBack(now)
		return true
	}

	return false
}

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
