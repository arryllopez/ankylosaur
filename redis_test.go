package ankylogo

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

/*
Helper function to create a Redis client for testing
Connects to Redis on localhost:6379 (default Docker setup)
Returns nil if Redis is not available (tests will be skipped)
*/
func setupRedisClient() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	// Ping Redis to check if it's available
	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil
	}

	return client
}

// Test cases for Redis Token Bucket

/*
Testing First Request with Redis Token Bucket
Initializing a Redis store and making the first request
The first request should always be allowed since the bucket starts at full capacity
*/
func TestRedisFirstRequestBucket(t *testing.T) {
	client := setupRedisClient()
	if client == nil {
		t.Skip("Redis not available, skipping test")
	}
	defer client.Close()

	var store *RedisStore = NewRedisStore(client)
	var ip string = "test-bucket-first-request"
	var status bool = store.AllowedTokenBucket(ip, 3, 2, time.Second)

	if !status {
		t.Error("First request should be allowed")
	}

	// Cleanup
	ctx := context.Background()
	client.Del(ctx, "bucket:"+ip)
}

/*
Testing Redis Token Bucket Limit
Creating a bucket with capacity of 1 and no refill
Making 1 request (should succeed), then a 2nd request (should fail)
*/
func TestRedisLimitBucket(t *testing.T) {
	client := setupRedisClient()
	if client == nil {
		t.Skip("Redis not available, skipping test")
	}
	defer client.Close()

	var store *RedisStore = NewRedisStore(client)
	var ip string = "test-bucket-limit"

	// First request should succeed (bucket starts at capacity 1)
	var firstStatus bool = store.AllowedTokenBucket(ip, 1, 0, time.Second)
	if !firstStatus {
		t.Error("First request should be allowed")
	}

	// Second request should fail (no tokens left, no refill)
	var secondStatus bool = store.AllowedTokenBucket(ip, 1, 0, time.Second)
	if secondStatus {
		t.Error("Second request should be denied (bucket empty)")
	}

	// Cleanup
	ctx := context.Background()
	client.Del(ctx, "bucket:"+ip)
}

/*
Testing Redis Token Bucket Refill
Creating a bucket with capacity 2, refilling 1 token per second
Using 2 tokens, waiting 1 second for refill, then checking if 1 token is available
*/
func TestRedisTokenBucketRefill(t *testing.T) {
	client := setupRedisClient()
	if client == nil {
		t.Skip("Redis not available, skipping test")
	}
	defer client.Close()

	var store *RedisStore = NewRedisStore(client)
	var ip string = "test-bucket-refill"

	// Use 2 tokens (bucket capacity is 2)
	store.AllowedTokenBucket(ip, 2, 1, time.Second)
	store.AllowedTokenBucket(ip, 2, 1, time.Second)

	// Third request should fail (no tokens left)
	var thirdStatus bool = store.AllowedTokenBucket(ip, 2, 1, time.Second)
	if thirdStatus {
		t.Error("Third request should be denied (bucket empty)")
	}

	// Wait 1 second for refill (should add 1 token)
	time.Sleep(1 * time.Second)

	// Fourth request should succeed (1 token refilled)
	var fourthStatus bool = store.AllowedTokenBucket(ip, 2, 1, time.Second)
	if !fourthStatus {
		t.Error("Fourth request should be allowed (bucket refilled)")
	}

	// Cleanup
	ctx := context.Background()
	client.Del(ctx, "bucket:"+ip)
}

// Test cases for Redis Sliding Window

/*
Testing First Request with Redis Sliding Window
Initializing a Redis store and making the first request
The first request should always be allowed since the window is empty
*/
func TestRedisFirstRequestSlidingWindow(t *testing.T) {
	client := setupRedisClient()
	if client == nil {
		t.Skip("Redis not available, skipping test")
	}
	defer client.Close()

	var store *RedisStore = NewRedisStore(client)
	var ip string = "test-sliding-first"
	var status bool = store.AllowedSlidingWindow(ip, 60, 100)

	if !status {
		t.Error("First request should be allowed")
	}

	// Cleanup
	ctx := context.Background()
	client.Del(ctx, "sliding:"+ip)
}

/*
Testing Redis Sliding Window Limit
Creating a sliding window with a limit of 3 requests in a 60 second window
Making 3 requests (should all succeed), then a 4th request (should fail)
*/
func TestRedisLimitSlidingWindow(t *testing.T) {
	client := setupRedisClient()
	if client == nil {
		t.Skip("Redis not available, skipping test")
	}
	defer client.Close()

	ctx := context.Background()
	var store *RedisStore = NewRedisStore(client)
	var ip string = "test-sliding-limit"
	var window int64 = 60
	var limit int = 3
	var status bool

	// Cleanup any existing data before test and wait for it to take effect
	client.Del(ctx, "sliding:"+ip)
	time.Sleep(100 * time.Millisecond)

	// First 3 requests should succeed
	for i := 0; i < 3; i++ {
		status = store.AllowedSlidingWindow(ip, window, limit)
		if !status {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 4th request should fail (exceeded limit)
	var fourthStatus bool = store.AllowedSlidingWindow(ip, window, limit)
	if fourthStatus {
		t.Error("Fourth request should be denied (exceeded limit)")
	}

	// Cleanup after test
	client.Del(ctx, "sliding:"+ip)
}

/*
Testing Redis Sliding Window Expiry
Creating a sliding window with a 2 second window and a limit of 1 request
Making a request, waiting for the window to expire (2+ seconds), then making another request
The second request should succeed because the window has reset
*/
func TestRedisSlidingWindowExpiry(t *testing.T) {
	client := setupRedisClient()
	if client == nil {
		t.Skip("Redis not available, skipping test")
	}
	defer client.Close()

	var store *RedisStore = NewRedisStore(client)
	var ip string = "test-sliding-expiry"
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

	// Cleanup
	ctx := context.Background()
	client.Del(ctx, "sliding:"+ip)
}
