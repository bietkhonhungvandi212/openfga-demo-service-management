package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRateLimitingEnforced(t *testing.T) {
	limiter := NewRateLimiter(10, 10, true)

	router := gin.New()
	router.Use(RateLimitMiddleware(limiter))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	successCount := 0
	for i := 0; i < 15; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			successCount++
		} else if w.Code == http.StatusTooManyRequests {
			if w.Header().Get("X-RateLimit-Limit") == "" {
				t.Error("Expected X-RateLimit-Limit header")
			}
			if w.Header().Get("X-RateLimit-Remaining") == "" {
				t.Error("Expected X-RateLimit-Remaining header")
			}
			if w.Header().Get("X-RateLimit-Reset") == "" {
				t.Error("Expected X-RateLimit-Reset header")
			}
		}
	}

	if successCount != 10 {
		t.Errorf("Expected 10 successful requests, got %d", successCount)
	}
}

func TestRateLimitHeadersValues(t *testing.T) {
	limiter := NewRateLimiter(100, 200, true)

	router := gin.New()
	router.Use(RateLimitMiddleware(limiter))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	limit := w.Header().Get("X-RateLimit-Limit")
	remaining := w.Header().Get("X-RateLimit-Remaining")
	reset := w.Header().Get("X-RateLimit-Reset")

	if limit != "200" {
		t.Errorf("Expected X-RateLimit-Limit '200', got '%s'", limit)
	}
	if remaining == "" {
		t.Error("Expected X-RateLimit-Remaining header")
	}
	if reset == "" {
		t.Error("Expected X-RateLimit-Reset header")
	}

	remainingInt, _ := strconv.Atoi(remaining)
	if remainingInt < 0 || remainingInt > 199 {
		t.Errorf("Expected X-RateLimit-Remaining between 0 and 199, got %d", remainingInt)
	}
}

func TestRateLimitingDisabledByFeatureFlag(t *testing.T) {
	limiter := NewRateLimiter(10, 10, false)

	router := gin.New()
	router.Use(RateLimitMiddleware(limiter))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for i := 0; i < 15; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected all requests to succeed when rate limiting is disabled, got %d on request %d", w.Code, i+1)
		}
	}
}

func TestRateLimitExceededErrorMessage(t *testing.T) {
	limiter := NewRateLimiter(1, 1, true)

	router := gin.New()
	router.Use(RateLimitMiddleware(limiter))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req1, _ := http.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	req2, _ := http.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", w2.Code)
	}
}

func TestRateLimitTokenBucketRefill(t *testing.T) {
	limiter := NewRateLimiter(100, 100, true)

	allowed, _, _ := limiter.Allow("test-key")
	if !allowed {
		t.Error("Expected first request to be allowed")
	}

	for i := 0; i < 99; i++ {
		limiter.Allow("test-key")
	}

	allowed, _, _ = limiter.Allow("test-key")
	if allowed {
		t.Error("Expected request to be denied after bucket exhausted")
	}
}
