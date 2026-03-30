package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSPreflightRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
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
	gin.SetMode(gin.TestMode)
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

func TestCORSSameOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := CORSConfig{
		AllowedOrigins: "*",
		AllowedMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowedHeaders: "Content-Type,X-Service-Name",
		MaxAge:         86400,
	}

	router := gin.New()
	router.Use(CORSMiddleware(config))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://any-origin.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("Expected Access-Control-Allow-Origin for wildcard origin")
	}
}

func TestCORSSpecificOriginMatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := CORSConfig{
		AllowedOrigins: "https://example.com,https://api.example.com",
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
	req.Header.Set("Origin", "https://api.example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "https://api.example.com" {
		t.Errorf("Expected Access-Control-Allow-Origin 'https://api.example.com', got '%s'", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSNoOriginsConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config := CORSConfig{
		AllowedOrigins: "",
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
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("Expected no Access-Control-Allow-Origin when no origins configured, got '%s'", w.Header().Get("Access-Control-Allow-Origin"))
	}
}
