package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger *zap.Logger
)

func main() {
	// Initialize logger
	initLogger()

	serviceURL := os.Getenv("SERVICE_INTERNAL_A_URL")
	port := os.Getenv("PORT")
	name := os.Getenv("NAME")

	if port == "" {
		port = "8081"
	}

	if name == "" {
		name = "service-caller-a"
	}

	if serviceURL == "" {
		logger.Fatal("SERVICE_INTERNAL_A_URL environment variable is required")
	}

	// Create HTTP client with timeout
	timeoutStr := os.Getenv("HTTP_CLIENT_TIMEOUT")
	if timeoutStr == "" {
		timeoutStr = "5s"
	}
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		timeout = 5 * time.Second
	}

	client := &http.Client{
		Timeout: timeout,
	}

	// Create router
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(loggingMiddleware())

	// Health endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": name,
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// Internal endpoint that calls service-internal
	router.GET("/internal", func(c *gin.Context) {
		req, err := http.NewRequest(http.MethodGet, serviceURL+"/health", nil)
		if err != nil {
			logger.Error("Failed to create request", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "cannot create request",
			})
			return
		}

		req.Header.Add("X-Service-Name", name)

		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		req = req.WithContext(ctx)

		resp, err := client.Do(req)
		if err != nil {
			logger.Error("Failed to call internal service", zap.Error(err))
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "cannot reach internal service",
			})
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Error("Failed to read response body", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "cannot read response",
			})
			return
		}

		logger.Info("Call to internal service successful",
			zap.String("caller", name),
			zap.String("status", fmt.Sprintf("%d", resp.StatusCode)),
		)

		c.JSON(http.StatusOK, gin.H{
			"service":  "service-internal",
			"response": string(body),
			"status":   resp.StatusCode,
		})
	})

	// Create server
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting server", zap.String("port", port), zap.String("service", name))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

func initLogger() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	logFormat := os.Getenv("LOG_FORMAT")
	if logFormat == "" {
		logFormat = "json"
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

	if logLevel == "debug" {
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	if logFormat == "console" {
		config.Encoding = "console"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	var err error
	logger, err = config.Build()
	if err != nil {
		panic(err)
	}
}

func loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		duration := time.Since(start)
		logger.Info("Request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", duration),
			zap.String("client_ip", c.ClientIP()),
		)
	}
}
