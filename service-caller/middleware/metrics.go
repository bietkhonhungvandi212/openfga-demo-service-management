package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ServiceName string
	InstanceID  string

	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "http_requests_total",
			Help:        "Total number of HTTP requests",
			ConstLabels: getServiceLabels(),
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "http_request_duration_seconds",
			Help:        "HTTP request latency distribution",
			Buckets:     []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
			ConstLabels: getServiceLabels(),
		},
		[]string{"method", "path"},
	)

	HTTPClientRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "http_client_requests_total",
			Help:        "Total number of outbound HTTP client requests",
			ConstLabels: getServiceLabels(),
		},
		[]string{"method", "host", "status"},
	)

	HTTPClientRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "http_client_request_duration_seconds",
			Help:        "Outbound HTTP request latency distribution",
			Buckets:     []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
			ConstLabels: getServiceLabels(),
		},
		[]string{"method", "host"},
	)

	RateLimitExceededTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "rate_limit_exceeded_total",
			Help:        "Total number of rate limit exceeded events",
			ConstLabels: getServiceLabels(),
		},
		[]string{},
	)

	CircuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:        "circuit_breaker_state",
			Help:        "Circuit breaker state (0=closed, 1=half-open, 2=open)",
			ConstLabels: getServiceLabels(),
		},
		[]string{"name"},
	)
)

func getServiceLabels() map[string]string {
	labels := make(map[string]string)
	if ServiceName != "" {
		labels["service_name"] = ServiceName
	}
	if InstanceID != "" {
		labels["instance"] = InstanceID
	}
	return labels
}

func SetServiceInfo(name, instance string) {
	ServiceName = name
	InstanceID = instance
}

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		method := c.Request.Method

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
		HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
	}
}

func RecordRateLimitExceeded() {
	RateLimitExceededTotal.WithLabelValues().Inc()
}

func RecordHTTPClientRequest(method, host, status string, duration time.Duration) {
	HTTPClientRequestsTotal.WithLabelValues(method, host, status).Inc()
	HTTPClientRequestDuration.WithLabelValues(method, host).Observe(duration.Seconds())
}

func RecordCircuitBreakerState(name string, state float64) {
	CircuitBreakerState.WithLabelValues(name).Set(state)
}
