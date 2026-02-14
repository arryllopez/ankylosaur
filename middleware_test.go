package ankylogo

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// helper to create a test router with the middleware applied
func setupTestRouter(config Config, policies ...map[string]Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RateLimiterMiddleware(NewMemoryStore(), config, policies...))
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})
	return router
}

// helper to make a GET request to /ping and return the response
func makeRequest(router *gin.Engine) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ping", nil)
	router.ServeHTTP(w, req)
	return w
}

/*
Testing with both algorithms enabled (default config)
Using a small capacity of 2 for token bucket so it triggers quickly
Sliding window has a limit of 100 so token bucket will be the bottleneck
First 2 requests should pass, 3rd should be denied by the token bucket
*/
func TestMiddlewareBothAlgorithms(t *testing.T) {
	config := Config{
		Window:            60,
		Limit:             100,
		Capacity:          2,
		TokensPerInterval: 0,
		RefillRate:        time.Second,
	}
	router := setupTestRouter(config)

	// First 2 requests should succeed (token bucket capacity is 2)
	for i := 0; i < 2; i++ {
		w := makeRequest(router)
		if w.Code != http.StatusOK {
			t.Errorf("Request %d should return 200, got %d", i+1, w.Code)
		}
	}

	// 3rd request should be denied (token bucket empty, no refill)
	w := makeRequest(router)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Request 3 should return 429, got %d", w.Code)
	}
}

/*
Testing with only sliding window enabled
Capacity is 0 so token bucket is skipped entirely
Sliding window allows 3 requests in a 60 second window
4th request should be denied
*/
func TestMiddlewareSlidingWindowOnly(t *testing.T) {
	config := Config{
		Window: 60,
		Limit:  3,
		// Capacity is 0 so token bucket is skipped
	}
	router := setupTestRouter(config)

	// First 3 requests should succeed
	for i := 0; i < 3; i++ {
		w := makeRequest(router)
		if w.Code != http.StatusOK {
			t.Errorf("Request %d should return 200, got %d", i+1, w.Code)
		}
	}

	// 4th request should be denied by sliding window
	w := makeRequest(router)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Request 4 should return 429, got %d", w.Code)
	}
}

/*
Testing with only token bucket enabled
Window and Limit are 0 so sliding window is skipped entirely
Token bucket has a capacity of 2 with no refill
3rd request should be denied
*/
func TestMiddlewareTokenBucketOnly(t *testing.T) {
	config := Config{
		// Window and Limit are 0 so sliding window is skipped
		Capacity:          2,
		TokensPerInterval: 0,
		RefillRate:        time.Second,
	}
	router := setupTestRouter(config)

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		w := makeRequest(router)
		if w.Code != http.StatusOK {
			t.Errorf("Request %d should return 200, got %d", i+1, w.Code)
		}
	}

	// 3rd request should be denied by token bucket
	w := makeRequest(router)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Request 3 should return 429, got %d", w.Code)
	}
}

/*
Testing with neither algorithm configured (empty config)
All values are 0 so both algorithms are skipped
Every request should pass through with no rate limiting
*/
func TestMiddlewareNoRateLimiting(t *testing.T) {
	config := Config{
		// Everything is 0, no rate limiting
	}
	router := setupTestRouter(config)

	// All 20 requests should succeed since nothing is configured
	for i := 0; i < 20; i++ {
		w := makeRequest(router)
		if w.Code != http.StatusOK {
			t.Errorf("Request %d should return 200 (no rate limiting), got %d", i+1, w.Code)
		}
	}
}
