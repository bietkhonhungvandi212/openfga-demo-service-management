package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestHTTPRequestsTotalMetric(t *testing.T) {
	gin.SetMode(gin.TestMode)
	HTTPRequestsTotal.WithLabelValues("GET", "/test", "200").Inc()
}

func TestHTTPRequestDurationMetric(t *testing.T) {
	HTTPRequestDuration.WithLabelValues("GET", "/test").Observe(0.001)
}

func TestHTTPClientRequestMetrics(t *testing.T) {
	RecordHTTPClientRequest(http.MethodGet, "http://example.com", "200", 50*time.Millisecond)
}

func TestRateLimitExceededMetric(t *testing.T) {
	RecordRateLimitExceeded()
}

func TestServiceLabels(t *testing.T) {
	SetServiceInfo("test-service", "instance-1")

	labels := getServiceLabels()
	if labels["service_name"] != "test-service" {
		t.Errorf("Expected service_name 'test-service', got '%s'", labels["service_name"])
	}
	if labels["instance"] != "instance-1" {
		t.Errorf("Expected instance 'instance-1', got '%s'", labels["instance"])
	}
}

func TestMetricsMiddlewareRecordsDuration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MetricsMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestCircuitBreakerStateMetric(t *testing.T) {
	RecordCircuitBreakerState("test-breaker", 0)
	RecordCircuitBreakerState("test-breaker", 1)
	RecordCircuitBreakerState("test-breaker", 2)
}
