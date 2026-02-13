package main

import (
	"context"
	"log"
	"time"

	ankylogo "github.com/arryllopez/ankyloGo"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// EXAMPLE USING REDIS
func main() {
	// Create Redis client
	// Make sure Redis is running: docker run -d -p 6379:6379 redis:latest
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis server address
		Password: "",               // No password by default
		DB:       0,                // Use default DB
	})

	// Test Redis connection
	ctx := context.Background()
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}
	log.Println("Successfully connected to Redis!")

	// Create RedisStore for rate limiting
	redisStore := ankylogo.NewRedisStore(redisClient)

	// Create custom config or use defaults
	// This config allows:
	// - Sliding window: 100 requests per 60 seconds
	// - Token bucket: 10 token capacity, refills 1 token per second
	config := ankylogo.Config{
		Window:            60,          // 60 second sliding window
		Limit:             100,         // 100 requests max in the window
		Capacity:          10,          // 10 token bucket capacity
		TokensPerInterval: 1,           // refill 1 token per interval
		RefillRate:        time.Second, // refill every second
	}

	// Or use the default config
	// config := ankylogo.DefaultConfig()

	// Create Gin router
	router := gin.Default()

	// Apply rate limiter middleware with Redis
	// This will rate limit by client IP using both sliding window and token bucket
	router.Use(ankylogo.RateLimiterMiddleware(redisStore, config))

	// Example routes
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello! You are rate limited by Redis.",
		})
	})

	router.GET("/api/data", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"data": "This endpoint is also rate limited",
		})
	})

	// Start server
	log.Println("Server starting on :8080")
	log.Println("Rate limiting with Redis on localhost:6379")
	log.Println("Try: curl http://localhost:8080")
	router.Run(":8080")
}
