package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func RequestValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		contentType := c.Request.Header.Get("Content-Type")
		if contentType != "" && !isValidContentType(contentType) {
			c.AbortWithStatusJSON(415, gin.H{
				"error": "unsupported media type: only application/json and text/plain are accepted",
			})
			return
		}

		if c.Request.ContentLength > 1*1024*1024 {
			c.AbortWithStatusJSON(413, gin.H{
				"error": "request entity too large: maximum size is 1MB",
			})
			return
		}

		c.Next()
	}
}

func isValidContentType(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.HasPrefix(ct, "application/json") || strings.HasPrefix(ct, "text/plain")
}

func MaxBodySizeMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

type BodyLimitMiddleware struct {
	maxBytes int64
}

func NewBodyLimitMiddleware(maxBytes int64) *BodyLimitMiddleware {
	return &BodyLimitMiddleware{maxBytes: maxBytes}
}

func (m *BodyLimitMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > m.maxBytes && c.Request.ContentLength > 0 {
			c.AbortWithStatusJSON(413, gin.H{
				"error": "request entity too large",
			})
			return
		}
		c.Next()
	}
}
