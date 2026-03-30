package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type healthResponse struct {
	Status       string            `json:"status"`
	Service      string            `json:"service"`
	Version      string            `json:"version"`
	Dependencies map[string]string `json:"dependencies"`
	Timestamp    string            `json:"timestamp"`
	Reason       string            `json:"reason"`
}

func TestLivenessProbe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	healthHandler := NewHealthHandler("test-service", "v1.0.0")

	router := gin.New()
	router.GET("/health/live", healthHandler.Liveness)

	req, _ := http.NewRequest("GET", "/health/live", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp healthResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Status != "alive" {
		t.Errorf("Expected status 'alive', got '%s'", resp.Status)
	}
}

func TestReadinessProbeHealthy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDep := &mockDependencyChecker{healthy: true, status: "healthy"}
	healthHandler := NewHealthHandler("test-service", "v1.0.0", mockDep)

	router := gin.New()
	router.GET("/health/ready", healthHandler.Readiness)

	req, _ := http.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp healthResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Status != "ready" {
		t.Errorf("Expected status 'ready', got '%s'", resp.Status)
	}
	if resp.Version == "" {
		t.Error("Expected version field in response")
	}
}

func TestReadinessProbeUnhealthy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDep := &mockDependencyChecker{healthy: false, status: "unhealthy"}
	healthHandler := NewHealthHandler("test-service", "v1.0.0", mockDep)

	router := gin.New()
	router.GET("/health/ready", healthHandler.Readiness)

	req, _ := http.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	var resp healthResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Status != "not_ready" {
		t.Errorf("Expected status 'not_ready', got '%s'", resp.Status)
	}
}

func TestLivenessResponseTime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	healthHandler := NewHealthHandler("test-service", "v1.0.0")

	router := gin.New()
	router.GET("/health/live", healthHandler.Liveness)

	start := time.Now()
	req, _ := http.NewRequest("GET", "/health/live", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	duration := time.Since(start)

	if duration > 100*time.Millisecond {
		t.Errorf("Expected response within 100ms, took %v", duration)
	}
}

func TestVersionEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	versionHandler := NewVersionHandler("v1.0.0", "abc123", "2024-01-01")

	router := gin.New()
	router.GET("/version", versionHandler.GetVersion)

	req, _ := http.NewRequest("GET", "/version", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["version"] != "v1.0.0" {
		t.Errorf("Expected version 'v1.0.0', got '%s'", resp["version"])
	}
	if resp["git_commit"] != "abc123" {
		t.Errorf("Expected git_commit 'abc123', got '%s'", resp["git_commit"])
	}
	if resp["build_time"] != "2024-01-01" {
		t.Errorf("Expected build_time '2024-01-01', got '%s'", resp["build_time"])
	}
}

func TestReadinessIncludesVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockDep := &mockDependencyChecker{healthy: true, status: "healthy"}
	healthHandler := NewHealthHandler("test-service", "v1.0.0", mockDep)

	router := gin.New()
	router.GET("/health/ready", healthHandler.Readiness)

	req, _ := http.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp healthResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Version == "" {
		t.Error("Expected version field in health/ready response")
	}
}

func TestShutdownState(t *testing.T) {
	SetShuttingDown(true)

	if !IsShuttingDown() {
		t.Error("Expected IsShuttingDown to return true")
	}

	SetShuttingDown(false)

	if IsShuttingDown() {
		t.Error("Expected IsShuttingDown to return false")
	}
}

func TestMetricsEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	HTTPRequestsTotal.WithLabelValues("GET", "/test", "200").Inc()
	HTTPRequestDuration.WithLabelValues("GET", "/test").Observe(0.001)

	router := gin.New()
	router.Use(MetricsMiddleware())
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") || !strings.Contains(contentType, "version=0.0.4") {
		t.Errorf("Expected content-type text/plain; version=0.0.4, got %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "# HELP http_requests_total") {
		t.Error("Missing # HELP http_requests_total")
	}
	if !strings.Contains(body, "# TYPE http_requests_total counter") {
		t.Error("Missing # TYPE http_requests_total counter")
	}
}

type mockDependencyChecker struct {
	healthy bool
	status  string
}

func (m *mockDependencyChecker) CheckHealth() (bool, string) {
	return m.healthy, m.status
}
