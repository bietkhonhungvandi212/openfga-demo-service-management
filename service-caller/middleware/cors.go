package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

type CORSConfig struct {
	AllowedOrigins string
	AllowedMethods string
	AllowedHeaders string
	MaxAge         int
}

func CORSMiddleware(cfg CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowedOrigins := parseOrigins(cfg.AllowedOrigins)

		if len(allowedOrigins) > 0 && isOriginAllowed(origin, allowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", cfg.AllowedMethods)
			c.Header("Access-Control-Allow-Headers", cfg.AllowedHeaders)
			c.Header("Access-Control-Max-Age", formatInt(cfg.MaxAge))
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func parseOrigins(origins string) []string {
	if origins == "" {
		return nil
	}
	return strings.Split(origins, ",")
}

func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		allowed = strings.TrimSpace(allowed)
		if allowed == origin || allowed == "*" {
			return true
		}
	}
	return false
}

func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	var result []byte
	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}
	return string(result)
}
