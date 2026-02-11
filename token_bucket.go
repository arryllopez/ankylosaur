package ankylogo

import (
	"sync"
	"time"
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
		tokens:       capacity,
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
