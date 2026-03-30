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

func TestIdempotencyKeyFirstRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
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
	gin.SetMode(gin.TestMode)
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
	gin.SetMode(gin.TestMode)
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

func TestIdempotencyKeyDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := NewIdempotencyStore(24 * time.Hour)
	router := gin.New()
	router.Use(IdempotencyMiddleware(store, false))
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "created"})
	})

	req, _ := http.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 when idempotency disabled, got %d", w.Code)
	}
}

func TestIdempotencyStoreSetAndGet(t *testing.T) {
	store := NewIdempotencyStore(24 * time.Hour)

	store.Set("test-key", 200, []byte(`{"status":"ok"}`))

	status, body, found := store.Get("test-key")
	if !found {
		t.Error("Expected to find stored entry")
	}
	if status != 200 {
		t.Errorf("Expected status 200, got %d", status)
	}
	if !bytes.Contains(body, []byte("ok")) {
		t.Error("Expected body to contain 'ok'")
	}
}

func TestIdempotencyStoreExpiration(t *testing.T) {
	store := NewIdempotencyStore(1 * time.Millisecond)

	store.Set("test-key", 200, []byte(`{"status":"ok"}`))

	_, _, found := store.Get("test-key")
	if !found {
		t.Error("Expected to find entry before expiration")
	}

	time.Sleep(10 * time.Millisecond)

	_, _, found = store.Get("test-key")
	if found {
		t.Error("Expected entry to be expired")
	}
}

func TestIdempotencyStoreCleanup(t *testing.T) {
	store := NewIdempotencyStore(1 * time.Millisecond)

	store.Set("key1", 200, []byte(`{}`))
	store.Set("key2", 200, []byte(`{}`))

	time.Sleep(10 * time.Millisecond)

	store.Cleanup()

	_, _, found := store.Get("key1")
	if found {
		t.Error("Expected key1 to be cleaned up")
	}
	_, _, found = store.Get("key2")
	if found {
		t.Error("Expected key2 to be cleaned up")
	}
}

func TestIdempotencyKeyNonPOSTRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := NewIdempotencyStore(24 * time.Hour)
	router := gin.New()
	router.Use(IdempotencyMiddleware(store, true))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Idempotency-Key", uuid.New().String())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for GET request, got %d", w.Code)
	}
}
