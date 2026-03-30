package middleware

import (
	"github.com/gin-gonic/gin"
)

var excludedPaths = map[string]bool{
	"/metrics":      true,
	"/health/live":  true,
	"/health/ready": true,
}

func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if excludedPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'none'")

		c.Next()
	}
}

func IsExcludedFromSecurity(path string) bool {
	return excludedPaths[path]
}

func IsHealthEndpoint(path string) bool {
	return path == "/health/live" || path == "/health/ready"
}
