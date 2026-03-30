package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

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
			for key, value := range map[string]string{} {
				c.Header(key, value)
			}
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

		headers := make(map[string]string)
		for key, values := range blw.Header() {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}

		store.Set(idempotencyKey, blw.Status(), blw.body.Bytes(), headers)
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

func RespondJSON(c *gin.Context, status int, data interface{}) {
	c.JSON(status, data)
}

func RespondError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

func RespondWithBody(c *gin.Context, status int, contentType string, body []byte) {
	c.Data(status, contentType, body)
}

func CopyResponse(c *gin.Context, status int, headers map[string]string, body []byte) {
	for key, value := range headers {
		c.Header(key, value)
	}
	c.Header(HeaderIdempotencyKey, c.GetHeader(HeaderIdempotencyKey))
	c.Data(status, "application/json", body)
}

func MarshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
