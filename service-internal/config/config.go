package config

import (
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Config struct {
	Port                string
	Name                string
	APIVersion          string
	LogLevel            string
	LogFormat           string
	LogSampleRate       float64
	OpenFGAAPI          string
	StoreID             string
	FGACacheEnabled     bool
	FGACacheTTL         time.Duration
	FGARetryMaxAttempts int
	FGARetryBackoff     time.Duration

	RateLimitEnabled           bool
	RateLimitRequestsPerSecond float64
	RateLimitBurst             int

	CORSAllowedOrigins string
	CORSAllowedMethods string
	CORSAllowedHeaders string
	CORSMaxAge         int

	IdempotencyCacheTTL time.Duration

	FeatureRateLimiting   bool
	FeatureIdempotency    bool
	FeatureCircuitBreaker bool
	FeatureFGACache       bool

	ShutdownTimeout time.Duration

	ServiceVersion string
	GitCommit      string
	BuildTime      string
}

var (
	cfg *Config
	mu  sync.RWMutex
)

func Load() *Config {
	mu.Lock()
	defer mu.Unlock()

	cfg = &Config{
		Port:                getEnv("PORT", "8080"),
		Name:                getEnv("NAME", "service-internal-a"),
		APIVersion:          getEnv("API_VERSION", "v1"),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		LogFormat:           getEnv("LOG_FORMAT", "json"),
		LogSampleRate:       getEnvFloat("LOG_SAMPLE_RATE", 1.0),
		OpenFGAAPI:          getEnv("OPENFGA_API", ""),
		StoreID:             getEnv("STORE_ID", ""),
		FGACacheEnabled:     getEnvBool("FGA_CACHE_ENABLED", true),
		FGACacheTTL:         getEnvDuration("FGA_CACHE_TTL", 60*time.Second),
		FGARetryMaxAttempts: getEnvInt("FGA_RETRY_MAX_ATTEMPTS", 3),
		FGARetryBackoff:     getEnvDuration("FGA_RETRY_BACKOFF", 100*time.Millisecond),

		RateLimitEnabled:           getEnvBool("RATE_LIMIT_ENABLED", true),
		RateLimitRequestsPerSecond: getEnvFloat("RATE_LIMIT_REQUESTS_PER_SECOND", 100),
		RateLimitBurst:             getEnvInt("RATE_LIMIT_BURST", 200),

		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", ""),
		CORSAllowedMethods: getEnv("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS"),
		CORSAllowedHeaders: getEnv("CORS_ALLOWED_HEADERS", "Content-Type,X-Service-Name,X-Request-ID,X-Idempotency-Key"),
		CORSMaxAge:         getEnvInt("CORS_MAX_AGE", 86400),

		IdempotencyCacheTTL: getEnvDuration("IDEMPOTENCY_CACHE_TTL", 24*time.Hour),

		FeatureRateLimiting:   getEnvBool("FEATURE_RATE_LIMITING", true),
		FeatureIdempotency:    getEnvBool("FEATURE_IDEMPOTENCY", true),
		FeatureCircuitBreaker: getEnvBool("FEATURE_CIRCUIT_BREAKER", true),
		FeatureFGACache:       getEnvBool("FEATURE_FGA_CACHE", true),

		ShutdownTimeout: getEnvDuration("SHUTDOWN_TIMEOUT", 30*time.Second),

		ServiceVersion: getEnv("SERVICE_VERSION", "dev"),
		GitCommit:      getEnv("GIT_COMMIT", "local"),
		BuildTime:      getEnv("BUILD_TIME", "now"),
	}

	return cfg
}

func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()
	return cfg
}

func Reload() error {
	oldCfg := cfg
	newCfg := &Config{
		Port:                getEnv("PORT", "8080"),
		Name:                getEnv("NAME", "service-internal-a"),
		APIVersion:          getEnv("API_VERSION", "v1"),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		LogFormat:           getEnv("LOG_FORMAT", "json"),
		LogSampleRate:       getEnvFloat("LOG_SAMPLE_RATE", 1.0),
		OpenFGAAPI:          getEnv("OPENFGA_API", ""),
		StoreID:             getEnv("STORE_ID", ""),
		FGACacheEnabled:     getEnvBool("FGA_CACHE_ENABLED", true),
		FGACacheTTL:         getEnvDuration("FGA_CACHE_TTL", 60*time.Second),
		FGARetryMaxAttempts: getEnvInt("FGA_RETRY_MAX_ATTEMPTS", 3),
		FGARetryBackoff:     getEnvDuration("FGA_RETRY_BACKOFF", 100*time.Millisecond),

		RateLimitEnabled:           getEnvBool("RATE_LIMIT_ENABLED", true),
		RateLimitRequestsPerSecond: getEnvFloat("RATE_LIMIT_REQUESTS_PER_SECOND", 100),
		RateLimitBurst:             getEnvInt("RATE_LIMIT_BURST", 200),

		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", ""),
		CORSAllowedMethods: getEnv("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS"),
		CORSAllowedHeaders: getEnv("CORS_ALLOWED_HEADERS", "Content-Type,X-Service-Name,X-Request-ID,X-Idempotency-Key"),
		CORSMaxAge:         getEnvInt("CORS_MAX_AGE", 86400),

		IdempotencyCacheTTL: getEnvDuration("IDEMPOTENCY_CACHE_TTL", 24*time.Hour),

		FeatureRateLimiting:   getEnvBool("FEATURE_RATE_LIMITING", true),
		FeatureIdempotency:    getEnvBool("FEATURE_IDEMPOTENCY", true),
		FeatureCircuitBreaker: getEnvBool("FEATURE_CIRCUIT_BREAKER", true),
		FeatureFGACache:       getEnvBool("FEATURE_FGA_CACHE", true),

		ShutdownTimeout: getEnvDuration("SHUTDOWN_TIMEOUT", 30*time.Second),

		ServiceVersion: getEnv("SERVICE_VERSION", "dev"),
		GitCommit:      getEnv("GIT_COMMIT", "local"),
		BuildTime:      getEnv("BUILD_TIME", "now"),
	}

	if err := validateConfig(newCfg); err != nil {
		zap.L().Error("configuration reload failed", zap.Error(err))
		return err
	}

	mu.Lock()
	cfg = newCfg
	mu.Unlock()

	zap.L().Info("configuration reloaded",
		zap.String("log_level", newCfg.LogLevel),
		zap.String("log_format", newCfg.LogFormat),
		zap.Bool("rate_limit_enabled", newCfg.RateLimitEnabled),
		zap.Bool("fga_cache_enabled", newCfg.FGACacheEnabled),
	)

	_ = oldCfg
	return nil
}

func validateConfig(c *Config) error {
	if c.Port == "" {
		c.Port = "8080"
	}
	if c.RateLimitRequestsPerSecond <= 0 {
		c.RateLimitRequestsPerSecond = 100
	}
	if c.RateLimitBurst <= 0 {
		c.RateLimitBurst = 200
	}
	return nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		return val == "true" || val == "1"
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		var result int
		if _, err := parseInt(val, &result); err == nil {
			return result
		}
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		var result float64
		if _, err := parseFloat(val, &result); err == nil {
			return result
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if parsed, err := time.ParseDuration(val); err == nil {
			return parsed
		}
	}
	return defaultVal
}

func parseInt(s string, result *int) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		n = n*10 + int(c-'0')
	}
	*result = n
	return n, nil
}

func parseFloat(s string, result *float64) (float64, error) {
	var n float64 = 0
	var decimal float64 = 1
	var inDecimal bool
	for _, c := range s {
		if c == '.' {
			inDecimal = true
			continue
		}
		if c < '0' || c > '9' {
			return 0, nil
		}
		if inDecimal {
			decimal /= 10
			n += float64(c-'0') * decimal
		} else {
			n = n*10 + float64(c-'0')
		}
	}
	*result = n
	return n, nil
}
