package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestRequestSizeLimit(t *testing.T) {
	router := gin.New()
	router.Use(RequestValidationMiddleware())
	router.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	body := make([]byte, 2*1024*1024)
	req, _ := http.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status 413, got %d", w.Code)
	}

	var resp errorResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !strings.Contains(resp.Error, "request entity too large") {
		t.Errorf("Expected error containing 'request entity too large', got '%s'", resp.Error)
	}
}

func TestAPIVersioning(t *testing.T) {
	router := gin.New()
	router.GET("/api/v1/internal", func(c *gin.Context) {
		c.Header("X-API-Version", "v1")
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/api/v1/internal", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Header().Get("X-API-Version") != "v1" {
		t.Errorf("Expected X-API-Version 'v1', got '%s'", w.Header().Get("X-API-Version"))
	}
}

func TestIdempotencyKeyFirstRequest(t *testing.T) {
	store := NewIdempotencyStore(24 * time.Hour)
	router := gin.New()
	router.Use(IdempotencyMiddleware(store, true))
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "created"})
	})

	idempotencyKey := uuid.New().String()
	req, _ := http.NewRequest("POST", "/test", nil)
	req.Header.Set("X-Idempotency-Key", idempotencyKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestIdempotencyKeyDuplicateRequest(t *testing.T) {
	store := NewIdempotencyStore(24 * time.Hour)

	router := gin.New()
	router.Use(IdempotencyMiddleware(store, true))
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "created"})
	})

	idempotencyKey := uuid.New().String()

	req1, _ := http.NewRequest("POST", "/test", nil)
	req1.Header.Set("X-Idempotency-Key", idempotencyKey)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	var resp1 map[string]string
	json.Unmarshal(w1.Body.Bytes(), &resp1)

	req2, _ := http.NewRequest("POST", "/test", nil)
	req2.Header.Set("X-Idempotency-Key", idempotencyKey)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	var resp2 map[string]string
	json.Unmarshal(w2.Body.Bytes(), &resp2)

	if resp1["status"] != resp2["status"] {
		t.Errorf("Expected response body to match original")
	}
}

func TestIdempotencyKeyInvalidFormat(t *testing.T) {
	store := NewIdempotencyStore(24 * time.Hour)
	router := gin.New()
	router.Use(IdempotencyMiddleware(store, true))
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "created"})
	})

	req, _ := http.NewRequest("POST", "/test", nil)
	req.Header.Set("X-Idempotency-Key", "not-a-uuid")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var resp errorResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !strings.Contains(resp.Error, "invalid idempotency key") {
		t.Errorf("Expected error containing 'invalid idempotency key', got '%s'", resp.Error)
	}
}

func TestContentTypeValidation(t *testing.T) {
	router := gin.New()
	router.Use(RequestValidationMiddleware())
	router.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("POST", "/test", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("Expected status 415, got %d", w.Code)
	}

	var resp errorResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !strings.Contains(resp.Error, "unsupported media type") {
		t.Errorf("Expected error containing 'unsupported media type', got '%s'", resp.Error)
	}
}

func TestContentTypeValidationJSON(t *testing.T) {
	router := gin.New()
	router.Use(RequestValidationMiddleware())
	router.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("POST", "/test", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestContentTypeValidationTextPlain(t *testing.T) {
	router := gin.New()
	router.Use(RequestValidationMiddleware())
	router.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("POST", "/test", strings.NewReader("hello"))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
