package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCorrelationID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CorrelationIDMiddleware())
	router.GET("/test", func(c *gin.Context) {
		requestID, _ := c.Get("request_id")
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "test-123")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") != "test-123" {
		t.Errorf("Expected X-Request-ID header 'test-123', got '%s'", w.Header().Get("X-Request-ID"))
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["request_id"] != "test-123" {
		t.Errorf("Expected request_id 'test-123', got '%s'", resp["request_id"])
	}
}

func TestCorrelationIDGenerated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CorrelationIDMiddleware())
	router.GET("/test", func(c *gin.Context) {
		requestID, _ := c.Get("request_id")
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") == "" {
		t.Error("Expected X-Request-ID header to be generated")
	}
}

func TestCorrelationIDCallerService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CorrelationIDMiddleware())
	router.GET("/test", func(c *gin.Context) {
		callerService, _ := c.Get("caller_service")
		c.JSON(http.StatusOK, gin.H{"caller_service": callerService})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Service-Name", "service-caller-a")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["caller_service"] != "service-caller-a" {
		t.Errorf("Expected caller_service 'service-caller-a', got '%v'", resp["caller_service"])
	}
}

func TestRequestLoggerMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CorrelationIDMiddleware())
	router.Use(RequestLoggerMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
}

func TestRequestLoggerWithCorrelationID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CorrelationIDMiddleware())
	router.Use(RequestLoggerMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "test-correlation-id")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
}
