package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func IdempotencyMiddleware(store *IdempotencyStore, enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !enabled {
			c.Next()
			return
		}

		if c.Request.Method != "POST" {
			c.Next()
			return
		}

		idempotencyKey := c.GetHeader(HeaderIdempotencyKey)
		if idempotencyKey == "" {
			c.Next()
			return
		}

		if !isValidUUID(idempotencyKey) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid idempotency key: must be UUID format",
			})
			return
		}

		statusCode, body, found := store.Get(idempotencyKey)
		if found {
			c.Header(HeaderIdempotencyKey, idempotencyKey)
			c.Data(statusCode, "application/json", body)
			c.Abort()
			return
		}

		c.Set("idempotency_key", idempotencyKey)

		blw := &bodyLogWriter{body: bytes.NewBuffer(nil), ResponseWriter: c.Writer}
		c.Writer = blw

		c.Next()

		if c.IsAborted() {
			return
		}

		store.Set(idempotencyKey, blw.Status(), blw.body.Bytes())
	}
}

type IdempotencyStore struct {
	mu    sync.RWMutex
	store map[string]idempotencyEntry
	ttl   time.Duration
}

type idempotencyEntry struct {
	statusCode int
	body       []byte
	expires    time.Time
}

func NewIdempotencyStore(ttl time.Duration) *IdempotencyStore {
	return &IdempotencyStore{
		store: make(map[string]idempotencyEntry),
		ttl:   ttl,
	}
}

func (s *IdempotencyStore) Get(key string) (int, []byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.store[key]
	if !exists || time.Now().After(entry.expires) {
		return 0, nil, false
	}

	return entry.statusCode, entry.body, true
}

func (s *IdempotencyStore) Set(key string, statusCode int, body []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.store[key] = idempotencyEntry{
		statusCode: statusCode,
		body:       body,
		expires:    time.Now().Add(s.ttl),
	}
}

func (s *IdempotencyStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for key, entry := range s.store {
		if now.After(entry.expires) {
			delete(s.store, key)
		}
	}
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyLogWriter) Status() int {
	return w.ResponseWriter.Status()
}

func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

func ReadBody(c *gin.Context) ([]byte, error) {
	if c.Request.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	return body, nil
}

func MarshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
