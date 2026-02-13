# AnkyloGo Examples

This directory contains examples showing how to use the ankyloGo rate limiting middleware.

## Examples

### 1. `main.go` - In-Memory Rate Limiting
Uses `MemoryStore` to store rate limit data in application memory.

**Pros:**
- Simple setup, no external dependencies
- Fast performance

**Cons:**
- Data lost on server restart
- Not suitable for distributed/multi-server deployments

**Run:**
```bash
go run main.go
```

### 2. `redis/main.go` - Redis-Based Rate Limiting
Uses `RedisStore` to store rate limit data in Redis.

**Pros:**
- Persistent across server restarts
- Works with multiple servers (distributed rate limiting)
- Data is centralized in Redis

**Cons:**
- Requires Redis server running
- Slightly more complex setup

**Prerequisites:**
```bash
# Start Redis using Docker
docker run -d -p 6379:6379 redis:latest
```

**Run:**
```bash
cd redis
go run main.go
```

## Testing the Rate Limiter

Once either example is running, test the rate limiter:

```bash
# Make a single request
curl http://localhost:8080

# Spam requests to trigger rate limiting (should get 429 Too Many Requests)
for i in {1..200}; do curl http://localhost:8080; done
```

## Configuration

Both examples use either `DefaultConfig()` or a custom `Config` struct:

```go
config := ankylogo.Config{
    Window:            60,          // 60 second sliding window
    Limit:             100,         // 100 requests max in the window
    Capacity:          10,          // 10 token bucket capacity
    TokensPerInterval: 1,           // refill 1 token per interval
    RefillRate:        time.Second, // refill every second
}
```

The middleware combines two algorithms:
- **Sliding Window**: Limits total requests in a time window
- **Token Bucket**: Controls burst traffic and enforces gradual consumption
