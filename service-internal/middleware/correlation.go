package middleware

import (
	"math/rand"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	HeaderRequestID      = "X-Request-ID"
	HeaderServiceName    = "X-Service-Name"
	HeaderIdempotencyKey = "X-Idempotency-Key"
	HeaderAPIVersion     = "X-API-Version"
)

var logSampleRate float64 = 1.0
var isProduction = false

func SetLogSampleRate(rate float64) {
	logSampleRate = rate
}

func SetProductionMode(prod bool) {
	isProduction = prod
}

func shouldSample() bool {
	if logSampleRate >= 1.0 {
		return true
	}
	return rand.Float64() < logSampleRate
}

func CorrelationIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(HeaderRequestID)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set("request_id", requestID)
		c.Header(HeaderRequestID, requestID)

		serviceName := c.GetHeader(HeaderServiceName)
		if serviceName != "" {
			c.Set("caller_service", serviceName)
		}

		c.Next()
	}
}

func RequestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		if !shouldSample() {
			return
		}

		duration := time.Since(start)
		status := c.Writer.Status()
		requestID, _ := c.Get("request_id")
		callerService, _ := c.Get("caller_service")

		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", status),
			zap.Duration("duration", duration),
			zap.String("client_ip", c.ClientIP()),
		}

		if requestID != nil {
			fields = append(fields, zap.String("request_id", requestID.(string)))
		}

		if callerService != nil {
			fields = append(fields, zap.String("caller_service", callerService.(string)))
		}

		switch {
		case status >= 500:
			zap.L().Error("Request completed", fields...)
		case status >= 400:
			zap.L().Warn("Request completed", fields...)
		default:
			zap.L().Info("Request completed", fields...)
		}
	}
}
