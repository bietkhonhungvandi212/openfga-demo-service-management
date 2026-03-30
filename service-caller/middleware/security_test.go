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
	if w.Header().Get("Content-Security-Policy") != "default-src 'none'" {
		t.Errorf("Expected Content-Security-Policy 'default-src 'none'', got '%s'", w.Header().Get("Content-Security-Policy"))
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
	if w.Header().Get("X-Content-Type-Options") != "" {
		t.Errorf("Expected no X-Content-Type-Options on metrics, got '%s'", w.Header().Get("X-Content-Type-Options"))
	}
}
