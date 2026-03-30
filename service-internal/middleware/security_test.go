package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeadersApplied(t *testing.T) {
	router := gin.New()
	router.Use(SecurityHeadersMiddleware())
	router.GET("/api/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("Expected X-Content-Type-Options 'nosniff', got '%s'", w.Header().Get("X-Content-Type-Options"))
	}
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("Expected X-Frame-Options 'DENY', got '%s'", w.Header().Get("X-Frame-Options"))
	}
	if w.Header().Get("X-XSS-Protection") != "1; mode=block" {
		t.Errorf("Expected X-XSS-Protection '1; mode=block', got '%s'", w.Header().Get("X-XSS-Protection"))
	}
	if w.Header().Get("Strict-Transport-Security") != "max-age=31536000; includeSubDomains" {
		t.Errorf("Expected Strict-Transport-Security 'max-age=31536000; includeSubDomains', got '%s'", w.Header().Get("Strict-Transport-Security"))
	}
}

func TestMetricsEndpointExcludesSecurityHeaders(t *testing.T) {
	router := gin.New()
	router.Use(SecurityHeadersMiddleware())
	router.GET("/metrics", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("Content-Security-Policy") != "" {
		t.Errorf("Expected no Content-Security-Policy on metrics, got '%s'", w.Header().Get("Content-Security-Policy"))
	}
	if w.Header().Get("X-Frame-Options") != "" {
		t.Errorf("Expected no X-Frame-Options on metrics, got '%s'", w.Header().Get("X-Frame-Options"))
	}
}

func TestCORSPreflightRequest(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins: "https://example.com",
		AllowedMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowedHeaders: "Content-Type,X-Service-Name,X-Request-ID",
		MaxAge:         86400,
	}

	router := gin.New()
	router.Use(CORSMiddleware(config))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "X-Service-Name")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 204 {
		t.Errorf("Expected status 204, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("Expected Access-Control-Allow-Origin 'https://example.com', got '%s'", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Expected Access-Control-Allow-Methods header")
	}
	if w.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("Expected Access-Control-Allow-Headers header")
	}
	if w.Header().Get("Access-Control-Max-Age") == "" {
		t.Error("Expected Access-Control-Max-Age header")
	}
}

func TestCORSRejectedOrigin(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins: "https://example.com",
		AllowedMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowedHeaders: "Content-Type,X-Service-Name",
		MaxAge:         86400,
	}

	router := gin.New()
	router.Use(CORSMiddleware(config))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://malicious.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("Expected no Access-Control-Allow-Origin for rejected origin, got '%s'", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

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
