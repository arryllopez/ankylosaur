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
