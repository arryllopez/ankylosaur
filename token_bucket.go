package ankylogo

import (
	"sync"
	"time"
)

// The following code block implements the Token Bucket Algoritm
type TokenBucket struct {
	tokens            int
	capacity          int
	tokensPerInterval int
	refillRate        time.Duration
	lastRefill        time.Time
	mu                sync.Mutex // handling race conditions (two processes trying to access tokens simultaneously)
}

func NewTokenBucket(capacity, tokensPerInterval int, refillRate time.Duration) *TokenBucket {
	return &TokenBucket{
		capacity:          capacity,
		tokens:            capacity,
		tokensPerInterval: tokensPerInterval,
		refillRate:        refillRate,
		lastRefill:        time.Now(),
	}
}

func (tb *TokenBucket) TakeTokens() bool {
	// handle race conditions
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// calculate how many tokens to refill based on elapsed time
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)

	if tb.refillRate > 0 && tb.tokensPerInterval > 0 {
		// how many refill intervals have passed since the last refill
		intervals := int(elapsed / tb.refillRate)
		if intervals > 0 {
			tb.tokens += intervals * tb.tokensPerInterval
			// cap tokens at capacity
			if tb.tokens > tb.capacity {
				tb.tokens = tb.capacity
			}
			tb.lastRefill = now
		}
	}

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
