package ankylogo

import (
	"testing"
	"time"
)

// Test cases for token bucket

/*
Testing First Request
Initializing a token bucket limiter with a fake ip, a capacity of 3 tokens, a refill per interval of 2 tokens
and a refill rate every second
Essentially, this bucket will refill 2 tokens every second up to the capacity of 3 tokens
*/
func TestFirstRequestBucket(t *testing.T) {
	var firstBucketRequest *MemoryStore = NewMemoryStore()
	var status bool = firstBucketRequest.AllowedTokenBucket("199.999.999", 3, 2, time.Second)
	if !status {
		t.Error()
	}
}

func TestLimitBucket(t *testing.T) {
	var limitRequestBucket *MemoryStore = NewMemoryStore()
	var firstStatus bool = limitRequestBucket.AllowedTokenBucket("100.000.111", 1, 0, time.Second)
	if !firstStatus {
		t.Error()
	}
	// This is expected to fail since its exceeding the capacity of the bucket
	var secondStatus bool = limitRequestBucket.AllowedTokenBucket("100.000.111", 1, 0, time.Second)
	if secondStatus {
		t.Error()
	}
}

// Test cases for sliding window

/*
Testing First Request in Sliding Window
Initializing a sliding window limiter with a fake IP, a window of 60 seconds, and a limit of 100 requests
The first request should always be allowed since the window is empty
*/
func TestFirstRequestSlidingWindow(t *testing.T) {
	var store *MemoryStore = NewMemoryStore()
	var status bool = store.AllowedSlidingWindow("192.168.1.1", 60, 100)
	if !status {
		t.Error("First request should be allowed")
	}
}

/*
Testing Sliding Window Limit
Creating a sliding window with a limit of 3 requests in a 60 second window
Making 3 requests (should all succeed), then a 4th request (should fail)
*/
func TestLimitSlidingWindow(t *testing.T) {
	var store *MemoryStore = NewMemoryStore()
	var ip string = "10.0.0.1"
	var window int64 = 60
	var limit int = 3

	// First 3 requests should succeed
	for i := 0; i < 3; i++ {
		var status bool = store.AllowedSlidingWindow(ip, window, limit)
		if !status {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 4th request should fail (exceeded limit)
	var fourthStatus bool = store.AllowedSlidingWindow(ip, window, limit)
	if fourthStatus {
		t.Error("Fourth request should be denied (exceeded limit)")
	}
}

/*
Testing Sliding Window Expiry
Creating a sliding window with a 2 second window and a limit of 1 request
Making a request, waiting for the window to expire (2+ seconds), then making another request
The second request should succeed because the window has reset
*/
func TestSlidingWindowExpiry(t *testing.T) {
	var store *MemoryStore = NewMemoryStore()
	var ip string = "172.16.0.1"
	var window int64 = 2 // 2 second window
	var limit int = 1

	// First request should succeed
	var firstStatus bool = store.AllowedSlidingWindow(ip, window, limit)
	if !firstStatus {
		t.Error("First request should be allowed")
	}

	// Immediate second request should fail (window not expired)
	var secondStatus bool = store.AllowedSlidingWindow(ip, window, limit)
	if secondStatus {
		t.Error("Second request should be denied (window not expired)")
	}

	// Wait for window to expire
	time.Sleep(time.Duration(window+1) * time.Second)

	// Third request should succeed (window has reset)
	var thirdStatus bool = store.AllowedSlidingWindow(ip, window, limit)
	if !thirdStatus {
		t.Error("Third request should be allowed (window has reset)")
	}
}
