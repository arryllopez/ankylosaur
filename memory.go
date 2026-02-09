package ankylogo

import (
	"sync"
	"time"
)

type MemoryStore struct {
	bucketPerIp        sync.Map
	slidingWindowPerIP sync.Map
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (m *MemoryStore) AllowedSlidingWindow(ip string, window int64, limit int) bool {
	sw, okWindow := m.slidingWindowPerIP.Load(ip)
	var slideWindow *SlidingWindowLimiter
	if okWindow {
		slideWindow = sw.(*SlidingWindowLimiter)
	} else {
		slideWindow = NewSlidingWindowLimiter(window, limit)
		m.slidingWindowPerIP.Store(ip, slideWindow)
	}
	return slideWindow.Allow()
}

func (m *MemoryStore) AllowedTokenBucket(ip string, capacity, tokensPerInterval int, refillRate time.Duration) bool {
	bucket, okBucket := m.bucketPerIp.Load(ip)
	var bucketToken *TokenBucket
	if okBucket {
		bucketToken = bucket.(*TokenBucket)
	} else {
		bucketToken = NewTokenBucket(capacity, tokensPerInterval, refillRate)
		m.bucketPerIp.Store(ip, bucketToken)
	}
	return bucketToken.TakeTokens()

}
