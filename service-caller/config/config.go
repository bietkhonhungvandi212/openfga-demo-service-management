package config

import (
	"os"
	"sync"
	"time"
)

type Config struct {
	Port               string
	Name               string
	APIVersion         string
	LogLevel           string
	LogFormat          string
	LogSampleRate      float64
	ServiceInternalURL string

	HTTPClientTimeout         time.Duration
	HTTPMaxIdleConnections    int
	HTTPMaxIdlePerHost        int
	HTTPIdleConnectionTimeout time.Duration

	CircuitBreakerEnabled          bool
	CircuitBreakerFailureThreshold int
	CircuitBreakerSuccessThreshold int
	CircuitBreakerTimeout          time.Duration

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
		Port:               getEnv("PORT", "8081"),
		Name:               getEnv("NAME", "service-caller-a"),
		APIVersion:         getEnv("API_VERSION", "v1"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		LogFormat:          getEnv("LOG_FORMAT", "json"),
		LogSampleRate:      getEnvFloat("LOG_SAMPLE_RATE", 1.0),
		ServiceInternalURL: getEnv("SERVICE_INTERNAL_A_URL", ""),

		HTTPClientTimeout:         getEnvDuration("HTTP_CLIENT_TIMEOUT", 5*time.Second),
		HTTPMaxIdleConnections:    getEnvInt("HTTP_MAX_IDLE_CONNECTIONS", 100),
		HTTPMaxIdlePerHost:        getEnvInt("HTTP_MAX_IDLE_PER_HOST", 10),
		HTTPIdleConnectionTimeout: getEnvDuration("HTTP_IDLE_CONNECTION_TIMEOUT", 90*time.Second),

		CircuitBreakerEnabled:          getEnvBool("CIRCUIT_BREAKER_ENABLED", true),
		CircuitBreakerFailureThreshold: getEnvInt("CIRCUIT_BREAKER_FAILURE_THRESHOLD", 5),
		CircuitBreakerSuccessThreshold: getEnvInt("CIRCUIT_BREAKER_SUCCESS_THRESHOLD", 2),
		CircuitBreakerTimeout:          getEnvDuration("CIRCUIT_BREAKER_TIMEOUT", 30*time.Second),

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
		Port:               getEnv("PORT", "8081"),
		Name:               getEnv("NAME", "service-caller-a"),
		APIVersion:         getEnv("API_VERSION", "v1"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		LogFormat:          getEnv("LOG_FORMAT", "json"),
		LogSampleRate:      getEnvFloat("LOG_SAMPLE_RATE", 1.0),
		ServiceInternalURL: getEnv("SERVICE_INTERNAL_A_URL", ""),

		HTTPClientTimeout:         getEnvDuration("HTTP_CLIENT_TIMEOUT", 5*time.Second),
		HTTPMaxIdleConnections:    getEnvInt("HTTP_MAX_IDLE_CONNECTIONS", 100),
		HTTPMaxIdlePerHost:        getEnvInt("HTTP_MAX_IDLE_PER_HOST", 10),
		HTTPIdleConnectionTimeout: getEnvDuration("HTTP_IDLE_CONNECTION_TIMEOUT", 90*time.Second),

		CircuitBreakerEnabled:          getEnvBool("CIRCUIT_BREAKER_ENABLED", true),
		CircuitBreakerFailureThreshold: getEnvInt("CIRCUIT_BREAKER_FAILURE_THRESHOLD", 5),
		CircuitBreakerSuccessThreshold: getEnvInt("CIRCUIT_BREAKER_SUCCESS_THRESHOLD", 2),
		CircuitBreakerTimeout:          getEnvDuration("CIRCUIT_BREAKER_TIMEOUT", 30*time.Second),

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

		ShutdownTimeout: getEnvDuration("SHUTDOWN_TIMEOUT", 30*time.Second),

		ServiceVersion: getEnv("SERVICE_VERSION", "dev"),
		GitCommit:      getEnv("GIT_COMMIT", "local"),
		BuildTime:      getEnv("BUILD_TIME", "now"),
	}

	if err := validateConfig(newCfg); err != nil {
		return err
	}

	mu.Lock()
	cfg = newCfg
	mu.Unlock()

	_ = oldCfg
	return nil
}

func validateConfig(c *Config) error {
	if c.Port == "" {
		c.Port = "8081"
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
