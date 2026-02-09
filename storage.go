package ankylogo

import (
	"time"
)

type RateLimiterStore interface {
	AllowedSlidingWindow(ip string, window int64, limit int) bool
	AllowedTokenBucket(ip string, capacity, tokensPerInterval int, refillRate time.Duration) bool
}
