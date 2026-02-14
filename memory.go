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
	newWindow := NewSlidingWindowLimiter(window, limit)
	sw, _ := m.slidingWindowPerIP.LoadOrStore(ip, newWindow)
	slideWindow := sw.(*SlidingWindowLimiter)
	return slideWindow.Allow()
}

func (m *MemoryStore) AllowedTokenBucket(ip string, capacity, tokensPerInterval int, refillRate time.Duration) bool {
	newBucket := NewTokenBucket(capacity, tokensPerInterval, refillRate)
	bucket, _ := m.bucketPerIp.LoadOrStore(ip, newBucket)
	bucketToken := bucket.(*TokenBucket)
	return bucketToken.TakeTokens()
}
