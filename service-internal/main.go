package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	client "github.com/openfga/go-sdk/client"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger *zap.Logger
)

func main() {
	// Initialize logger
	initLogger()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	name := os.Getenv("NAME")
	if name == "" {
		name = "service-internal-a"
	}

	// Initialize OpenFGA client with retry
	fga := NewFGA()

	// Create router
	router := gin.New()

	// Add middlewares
	router.Use(gin.Recovery())
	router.Use(loggingMiddleware())
	router.Use(Authorize(fga, name))

	// Health endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": name,
			"time":    time.Now().Format(time.RFC3339),
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

type FGAAuth struct {
	client       *client.OpenFgaClient
	cache        map[string]cacheEntry
	cacheTTL     time.Duration
	cacheEnabled bool
	mu           sync.RWMutex
}

type cacheEntry struct {
	allowed bool
	expires time.Time
}

func NewFGA() *FGAAuth {
	cfg := &client.ClientConfiguration{
		ApiUrl:  os.Getenv("OPENFGA_API"),
		StoreId: os.Getenv("STORE_ID"),
	}

	fga, err := client.NewSdkClient(cfg)
	if err != nil {
		logger.Fatal("Failed to create OpenFGA client", zap.Error(err))
	}

	// Parse cache settings
	cacheEnabled := os.Getenv("FGA_CACHE_ENABLED") == "true"
	cacheTTL := 60 * time.Second
	if ttlStr := os.Getenv("FGA_CACHE_TTL"); ttlStr != "" {
		if parsed, err := time.ParseDuration(ttlStr); err == nil {
			cacheTTL = parsed
		}
	}

	return &FGAAuth{
		client:       fga,
		cache:        make(map[string]cacheEntry),
		cacheTTL:     cacheTTL,
		cacheEnabled: cacheEnabled,
	}
}

func (f *FGAAuth) CheckWithRetry(ctx context.Context, req client.ClientCheckRequest) (bool, error) {
	maxAttempts := 3
	if maxStr := os.Getenv("FGA_RETRY_MAX_ATTEMPTS"); maxStr != "" {
		if parsed, err := fmt.Sscanf(maxStr, "%d", &maxAttempts); err == nil && parsed > 0 {
			maxAttempts = parsed
		}
	}

	backoffBase := 100 * time.Millisecond
	if backoffStr := os.Getenv("FGA_RETRY_BACKOFF"); backoffStr != "" {
		if parsed, err := time.ParseDuration(backoffStr); err == nil {
			backoffBase = parsed
		}
	}

	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s:%s", req.User, req.Relation, req.Object)
	if f.cacheEnabled {
		f.mu.Lock()
		entry, found := f.cache[cacheKey]
		f.mu.Unlock()
		if found && time.Now().Before(entry.expires) {
			return entry.allowed, nil
		}
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := f.client.Check(ctx).
			Body(req).
			Execute()

		if err == nil && resp != nil && resp.Allowed != nil {
			allowed := *resp.Allowed

			// Cache the result
			if f.cacheEnabled {
				f.mu.Lock()
				f.cache[cacheKey] = cacheEntry{
					allowed: allowed,
					expires: time.Now().Add(f.cacheTTL),
				}
				f.mu.Unlock()
			}

			return allowed, nil
		}

		lastErr = err
		if attempt < maxAttempts {
			backoff := backoffBase * time.Duration(attempt)
			time.Sleep(backoff)
		}
	}

	logger.Error("FGA check failed after retries",
		zap.Int("attempts", maxAttempts),
		zap.Error(lastErr),
	)

	return false, lastErr
}

func Authorize(fga *FGAAuth, serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		caller := c.Request.Header.Get("X-Service-Name")
		if caller == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Missing X-Service-Name header",
			})
			c.Abort()
			return
		}

		bodyOptions := client.ClientCheckRequest{
			User:     "service:" + caller,
			Relation: "can_call",
			Object:   "service:" + serviceName,
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		allowed, err := fga.CheckWithRetry(ctx, bodyOptions)

		if err != nil {
			logger.Error("Authorization check failed",
				zap.String("caller", caller),
				zap.String("service", serviceName),
				zap.Error(err),
			)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Authorization service unavailable",
			})
			c.Abort()
			return
		}

		if !allowed {
			logger.Warn("Authorization denied",
				zap.String("caller", caller),
				zap.String("service", serviceName),
			)
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Forbidden",
			})
			c.Abort()
			return
		}

		logger.Debug("Authorization granted",
			zap.String("caller", caller),
			zap.String("service", serviceName),
		)

		c.Next()
	}
}
