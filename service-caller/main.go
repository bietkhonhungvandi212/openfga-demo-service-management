package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"service-caller/config"
	"service-caller/middleware"
)

var (
	logger *zap.Logger
)

type DependencyChecker struct {
	url string
}

func (d *DependencyChecker) CheckHealth() (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.url+"/health/live", nil)
	if err != nil {
		return false, "unhealthy"
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, "unhealthy"
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, "healthy"
	}
	return false, "unhealthy"
}

func main() {
	cfg := config.Load()
	initLogger(cfg.LogLevel, cfg.LogFormat)
	logger = zap.L()

	middleware.ServiceName = cfg.Name
	middleware.SetServiceInfo(cfg.Name, "")

	apiVersion := cfg.APIVersion
	if !strings.HasPrefix(apiVersion, "v") {
		apiVersion = "v" + apiVersion
	}

	httpClient := createHTTPClient(cfg)

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
			Name:             "internal-service-breaker",
		})
	}

	var idempotencyStore *middleware.IdempotencyStore
	if cfg.FeatureIdempotency {
		idempotencyStore = middleware.NewIdempotencyStore(cfg.IdempotencyCacheTTL)
		go cleanupIdempotencyStore(idempotencyStore)
		router.Use(middleware.IdempotencyMiddleware(idempotencyStore, cfg.FeatureIdempotency))
	}

	corsConfig := middleware.CORSConfig{
		AllowedOrigins: cfg.CORSAllowedOrigins,
		AllowedMethods: cfg.CORSAllowedMethods,
		AllowedHeaders: cfg.CORSAllowedHeaders,
		MaxAge:         cfg.CORSMaxAge,
	}
	router.Use(middleware.CORSMiddleware(corsConfig))

	healthHandler := middleware.NewHealthHandler(cfg.Name, cfg.ServiceVersion, &DependencyChecker{url: cfg.ServiceInternalURL})
	versionHandler := middleware.NewVersionHandler(cfg.ServiceVersion, cfg.GitCommit, cfg.BuildTime)

	router.GET("/health/live", healthHandler.Liveness)
	router.GET("/health/ready", healthHandler.Readiness)
	router.GET("/version", versionHandler.GetVersion)

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	api := router.Group(fmt.Sprintf("/api/%s", apiVersion))
	api.Use(middleware.RequestValidationMiddleware())
	if cfg.FeatureIdempotency {
		api.Use(middleware.IdempotencyMiddleware(idempotencyStore, cfg.FeatureIdempotency))
	}
	{
		api.GET("/internal", func(c *gin.Context) {
			requestID, _ := c.Get("request_id")

			req, reqErr := http.NewRequest(http.MethodGet, cfg.ServiceInternalURL+"/api/v1/internal", nil)
			if reqErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create request"})
				return
			}

			req.Header.Add("X-Service-Name", cfg.Name)
			req.Header.Add("X-Request-ID", requestID.(string))

			ctx, cancel := context.WithTimeout(c.Request.Context(), cfg.HTTPClientTimeout)
			defer cancel()
			req = req.WithContext(ctx)

			start := time.Now()
			resp, doErr := httpClient.Do(req)
			duration := time.Since(start)

			if doErr != nil {
				middleware.RecordHTTPClientRequest(http.MethodGet, cfg.ServiceInternalURL, "error", duration)
				logger.Error("Failed to call internal service",
					zap.Error(doErr),
					zap.String("request_id", requestID.(string)),
				)
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error": "cannot reach internal service",
				})
				return
			}
			defer resp.Body.Close()

			middleware.RecordHTTPClientRequest(http.MethodGet, cfg.ServiceInternalURL, fmt.Sprintf("%d", resp.StatusCode), duration)

			body, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot read response"})
				return
			}

			c.Header("X-API-Version", apiVersion)
			c.JSON(resp.StatusCode, gin.H{
				"service":   "service-internal",
				"response":  string(body),
				"status":    resp.StatusCode,
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
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

	switch strings.ToLower(logLevel) {
	case "debug":
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "warn":
		cfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		cfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	}

	if logFormat == "console" {
		cfg.Encoding = "console"
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	var err error
	logger, err = cfg.Build()
	if err != nil {
		panic(err)
	}
}

func createHTTPClient(cfg *config.Config) *http.Client {
	transport := &http.Transport{
		MaxIdleConns:          cfg.HTTPMaxIdleConnections,
		MaxIdleConnsPerHost:   cfg.HTTPMaxIdlePerHost,
		IdleConnTimeout:       cfg.HTTPIdleConnectionTimeout,
		ResponseHeaderTimeout: cfg.HTTPClientTimeout,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   cfg.HTTPClientTimeout,
	}
}

func cleanupIdempotencyStore(store *middleware.IdempotencyStore) {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		store.Cleanup()
	}
}
