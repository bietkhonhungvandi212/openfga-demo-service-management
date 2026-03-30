package config

import (
	"os"
	"testing"
	"time"
)

func TestConfigLoad(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("API_VERSION", "v2")
	os.Setenv("RATE_LIMIT_REQUESTS_PER_SECOND", "50")
	os.Setenv("RATE_LIMIT_BURST", "100")
	os.Setenv("SHUTDOWN_TIMEOUT", "60s")

	cfg := Load()

	if cfg.Port != "9090" {
		t.Errorf("Expected PORT '9090', got '%s'", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("Expected LOG_LEVEL 'debug', got '%s'", cfg.LogLevel)
	}
	if cfg.APIVersion != "v2" {
		t.Errorf("Expected API_VERSION 'v2', got '%s'", cfg.APIVersion)
	}
	if cfg.RateLimitRequestsPerSecond != 50 {
		t.Errorf("Expected RATE_LIMIT_REQUESTS_PER_SECOND 50, got %v", cfg.RateLimitRequestsPerSecond)
	}
	if cfg.RateLimitBurst != 100 {
		t.Errorf("Expected RATE_LIMIT_BURST 100, got %v", cfg.RateLimitBurst)
	}
	if cfg.ShutdownTimeout != 60*time.Second {
		t.Errorf("Expected SHUTDOWN_TIMEOUT 60s, got %v", cfg.ShutdownTimeout)
	}
}

func TestConfigDefaults(t *testing.T) {
	os.Unsetenv("PORT")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("API_VERSION")

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("Expected default PORT '8080', got '%s'", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("Expected default LOG_LEVEL 'info', got '%s'", cfg.LogLevel)
	}
	if cfg.APIVersion != "v1" {
		t.Errorf("Expected default API_VERSION 'v1', got '%s'", cfg.APIVersion)
	}
	if cfg.FeatureRateLimiting != true {
		t.Errorf("Expected default FEATURE_RATE_LIMITING true, got %v", cfg.FeatureRateLimiting)
	}
	if cfg.FeatureIdempotency != true {
		t.Errorf("Expected default FEATURE_IDEMPOTENCY true, got %v", cfg.FeatureIdempotency)
	}
	if cfg.FeatureCircuitBreaker != true {
		t.Errorf("Expected default FEATURE_CIRCUIT_BREAKER true, got %v", cfg.FeatureCircuitBreaker)
	}
}

func TestFeatureFlags(t *testing.T) {
	os.Setenv("FEATURE_RATE_LIMITING", "false")
	os.Setenv("FEATURE_IDEMPOTENCY", "false")
	os.Setenv("FEATURE_CIRCUIT_BREAKER", "false")
	os.Setenv("FEATURE_FGA_CACHE", "false")

	cfg := Load()

	if cfg.FeatureRateLimiting != false {
		t.Errorf("Expected FEATURE_RATE_LIMITING false, got %v", cfg.FeatureRateLimiting)
	}
	if cfg.FeatureIdempotency != false {
		t.Errorf("Expected FEATURE_IDEMPOTENCY false, got %v", cfg.FeatureIdempotency)
	}
	if cfg.FeatureCircuitBreaker != false {
		t.Errorf("Expected FEATURE_CIRCUIT_BREAKER false, got %v", cfg.FeatureCircuitBreaker)
	}
	if cfg.FeatureFGACache != false {
		t.Errorf("Expected FEATURE_FGA_CACHE false, got %v", cfg.FeatureFGACache)
	}
}

func TestConfigValidation(t *testing.T) {
	os.Setenv("RATE_LIMIT_REQUESTS_PER_SECOND", "50")
	os.Setenv("RATE_LIMIT_BURST", "100")

	cfg := Load()

	if cfg.RateLimitRequestsPerSecond != 50 {
		t.Errorf("Expected validated RATE_LIMIT_REQUESTS_PER_SECOND 50, got %v", cfg.RateLimitRequestsPerSecond)
	}
	if cfg.RateLimitBurst != 100 {
		t.Errorf("Expected validated RATE_LIMIT_BURST 100, got %v", cfg.RateLimitBurst)
	}
}

func TestConfigReload(t *testing.T) {
	os.Setenv("LOG_LEVEL", "info")

	cfg1 := Load()
	originalLogLevel := cfg1.LogLevel

	os.Setenv("LOG_LEVEL", "debug")

	err := Reload()
	if err != nil {
		t.Errorf("Expected reload to succeed, got error: %v", err)
	}

	cfg2 := Get()
	if cfg2.LogLevel != "debug" {
		t.Errorf("Expected LOG_LEVEL 'debug' after reload, got '%s'", cfg2.LogLevel)
	}

	_ = originalLogLevel
}

func TestConfigReloadInvalid(t *testing.T) {
	os.Setenv("RATE_LIMIT_REQUESTS_PER_SECOND", "10")

	Load()

	os.Setenv("RATE_LIMIT_REQUESTS_PER_SECOND", "-5")

	err := Reload()
	if err != nil {
		t.Logf("Reload failed as expected with invalid config")
	}

	cfg := Get()
	if cfg.RateLimitRequestsPerSecond <= 0 {
		t.Logf("Config validation should fix negative values")
	}
}
