package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	client "github.com/openfga/go-sdk/client"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"service-internal/config"
	"service-internal/middleware"
)

var (
	logger *zap.Logger
)

func main() {
	cfg := config.Load()
	initLogger(cfg.LogLevel, cfg.LogFormat)
	logger = zap.L()

	apiVersion := cfg.APIVersion
	if !strings.HasPrefix(apiVersion, "v") {
		apiVersion = "v" + apiVersion
	}

	fga := NewFGA(cfg)

	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(middleware.CorrelationIDMiddleware())
	router.Use(middleware.RequestLoggerMiddleware())
	router.Use(middleware.MetricsMiddleware())
	router.Use(middleware.SecurityHeadersMiddleware())

	if cfg.FeatureRateLimiting {
		rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRequestsPerSecond, cfg.RateLimitBurst, cfg.RateLimitEnabled)
		router.Use(middleware.RateLimitMiddleware(rateLimiter))
	}

	if cfg.FeatureCircuitBreaker {
		middleware.InitCircuitBreaker(middleware.CircuitBreakerConfig{
			Enabled:          cfg.FeatureCircuitBreaker,
			FailureThreshold: 5,
			SuccessThreshold: 2,
			Timeout:          30 * time.Second,
			Name:             "fga-circuit-breaker",
		})
	}

	var idempotencyStore *middleware.IdempotencyStore
	if cfg.FeatureIdempotency {
		idempotencyStore = middleware.NewIdempotencyStore(cfg.IdempotencyCacheTTL)
		go cleanupIdempotencyStore(idempotencyStore)
		router.Use(middleware.IdempotencyMiddleware(idempotencyStore, cfg.FeatureIdempotency))
	}

	healthHandler := middleware.NewHealthHandler(cfg.Name, cfg.ServiceVersion, fga)
	versionHandler := middleware.NewVersionHandler(cfg.ServiceVersion, cfg.GitCommit, cfg.BuildTime)

	router.GET("/health/live", healthHandler.Liveness)
	router.GET("/health/ready", healthHandler.Readiness)
	router.GET("/version", versionHandler.GetVersion)

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	corsConfig := middleware.CORSConfig{
		AllowedOrigins: cfg.CORSAllowedOrigins,
		AllowedMethods: cfg.CORSAllowedMethods,
		AllowedHeaders: cfg.CORSAllowedHeaders,
		MaxAge:         cfg.CORSMaxAge,
	}
	router.Use(middleware.CORSMiddleware(corsConfig))

	api := router.Group(fmt.Sprintf("/api/%s", apiVersion))
	api.Use(func(c *gin.Context) {
		c.Header("X-API-Version", apiVersion)
		c.Next()
	})
	api.Use(middleware.RequestValidationMiddleware())
	if cfg.FeatureIdempotency {
		api.Use(middleware.IdempotencyMiddleware(idempotencyStore, cfg.FeatureIdempotency))
	}
	api.Use(Authorize(fga, cfg.Name))
	{
		api.GET("/internal", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"service":   cfg.Name,
				"status":    "ok",
				"message":   "internal service reached successfully",
				"timestamp": time.Now().Format(time.RFC3339),
			})
		})
	}

	router.Any("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": cfg.Name,
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		logger.Info("Starting server",
			zap.String("port", cfg.Port),
			zap.String("service", cfg.Name),
			zap.String("version", cfg.APIVersion),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGHUP)
		for {
			sig := <-sigChan
			if sig == syscall.SIGHUP {
				logger.Info("Received SIGHUP, reloading configuration")
				if err := config.Reload(); err != nil {
					logger.Error("Configuration reload failed", zap.Error(err))
				} else {
					logger.Info("Configuration reloaded successfully")
					newCfg := config.Get()
					applyLogLevel(newCfg.LogLevel)
				}
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server gracefully")
	middleware.SetShuttingDown(true)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited gracefully")
}

func initLogger(logLevel, logFormat string) {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

	switch strings.ToLower(logLevel) {
	case "debug":
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "warn":
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
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

func applyLogLevel(logLevel string) {
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

func NewFGA(cfg *config.Config) *FGAAuth {
	cfg2 := &client.ClientConfiguration{
		ApiUrl:  cfg.OpenFGAAPI,
		StoreId: cfg.StoreID,
	}

	fga, err := client.NewSdkClient(cfg2)
	if err != nil {
		logger.Fatal("Failed to create OpenFGA client", zap.Error(err))
	}

	return &FGAAuth{
		client:       fga,
		cache:        make(map[string]cacheEntry),
		cacheTTL:     cfg.FGACacheTTL,
		cacheEnabled: cfg.FGACacheEnabled && cfg.FeatureFGACache,
	}
}

func (f *FGAAuth) CheckWithRetry(ctx context.Context, req client.ClientCheckRequest) (bool, error) {
	maxAttempts := 3
	backoffBase := 100 * time.Millisecond

	cacheKey := fmt.Sprintf("%s:%s:%s", req.User, req.Relation, req.Object)
	if f.cacheEnabled {
		f.mu.Lock()
		entry, found := f.cache[cacheKey]
		f.mu.Unlock()
		if found && time.Now().Before(entry.expires) {
			middleware.RecordFGACacheHit(true)
			return entry.allowed, nil
		}
		middleware.RecordFGACacheHit(false)
	}

	start := time.Now()
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := f.client.Check(ctx).
			Body(req).
			Execute()

		if err == nil && resp != nil && resp.Allowed != nil {
			allowed := *resp.Allowed
			duration := time.Since(start)
			middleware.RecordFGACheckDuration(duration)

			if f.cacheEnabled {
				f.mu.Lock()
				f.cache[cacheKey] = cacheEntry{
					allowed: allowed,
					expires: time.Now().Add(f.cacheTTL),
				}
				f.mu.Unlock()
			}

			if allowed {
				middleware.RecordFGAAuthorization("allowed")
			} else {
				middleware.RecordFGAAuthorization("denied")
			}

			return allowed, nil
		}

		lastErr = err
		if attempt < maxAttempts {
			backoff := backoffBase * time.Duration(attempt)
			time.Sleep(backoff)
		}
	}

	middleware.RecordFGAAuthorization("error")
	logger.Error("FGA check failed after retries",
		zap.Int("attempts", maxAttempts),
		zap.Error(lastErr),
	)

	return false, lastErr
}

func (f *FGAAuth) CheckHealth() (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := client.ClientCheckRequest{
		User:     "service:health-check",
		Relation: "can_call",
		Object:   "service:health",
	}

	_, err := f.client.Check(ctx).Body(req).Execute()
	if err != nil {
		return false, "unhealthy"
	}
	return true, "healthy"
}

func cleanupIdempotencyStore(store *middleware.IdempotencyStore) {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		store.Cleanup()
	}
}

func Authorize(fga *FGAAuth, serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		caller := c.GetHeader("X-Service-Name")
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
