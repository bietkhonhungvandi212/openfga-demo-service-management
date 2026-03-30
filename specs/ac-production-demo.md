# Acceptance Criteria: Production Demo Enhancement

## Gherkin Format

```gherkin
Feature: Service Management Distributed System - Production Readiness

  As a DevOps engineer demonstrating production-ready distributed systems
  I want the system to implement observability, security, and resilience patterns
  So that I can demonstrate practical scenarios suitable for production environments
```

---

## 1. Observability

### Scenario: Prometheus Metrics Endpoint
```gherkin
Given a running service instance
When I make a GET request to "/metrics"
Then the response status should be 200
And the response content-type should be "text/plain; version=0.0.4; charset=utf-8"
And the response body should contain "# HELP http_requests_total"
And the response body should contain "# HELP http_request_duration_seconds"
And the response body should contain "# HELP fga_authorization_total"
And the response body should contain '# TYPE http_requests_total counter'
And the response body should contain '# TYPE http_request_duration_seconds histogram'
```

### Scenario: Authorization Metrics
```gherkin
Given a service-internal instance with OpenFGA
When a caller requests an authorized endpoint with valid X-Service-Name
Then metrics should include fga_authorization_total with label "result"="allowed"
When the same caller requests an unauthorized endpoint
Then metrics should include fga_authorization_total with label "result"="denied"
```

### Scenario: Cache Hit Metrics
```gherkin
Given FGA caching is enabled with 60s TTL
When a caller requests an authorized endpoint twice within TTL
Then first request should have fga_cache_hits_total incremented by 0
And second request should have fga_cache_hits_total incremented by 1
```

### Scenario: Distributed Tracing with Correlation ID
```gherkin
Given a service-caller calling service-internal
When I send a request with header "X-Request-ID": "test-123"
Then service-internal logs should include "request_id":"test-123"
And service-internal response should include header "X-Request-ID":"test-123"
And service-caller logs should include "request_id":"test-123"
```

### Scenario: Liveness Probe
```gherkin
Given a running service instance
When I make a GET request to "/health/live"
Then the response status should be 200
And the response body should have "status" equal to "alive"
And the response should complete within 100ms
And no external dependencies should be checked
```

### Scenario: Readiness Probe - Healthy
```gherkin
Given a service-internal instance with healthy OpenFGA connection
When I make a GET request to "/health/ready"
Then the response status should be 200
And the response body should have "status" equal to "ready"
And the response body should have "dependencies.openfga.status" equal to "healthy"
```

### Scenario: Readiness Probe - Unhealthy
```gherkin
Given a service-internal instance with unreachable OpenFGA
When I make a GET request to "/health/ready"
Then the response status should be 503
And the response body should have "status" equal to "not_ready"
And the response body should have "dependencies.openfga.status" equal to "unhealthy"
```

### Scenario: Health Endpoints Bypass Authorization
```gherkin
Given service-internal with authorization enabled
When I make a GET request to "/health/live" without X-Service-Name header
Then the response status should be 200
And no authorization error should be returned
```

---

## 2. Security

### Scenario: Security Headers Applied
```gherkin
Given a running service instance
When I make any GET request to a protected endpoint
Then the response should include header "X-Content-Type-Options" with value "nosniff"
And the response should include header "X-Frame-Options" with value "DENY"
And the response should include header "X-XSS-Protection" with value "1; mode=block"
And the response should include header "Strict-Transport-Security" with value "max-age=31536000; includeSubDomains"
```

### Scenario: Metrics Endpoint Excludes Security Headers
```gherkin
Given a running service instance
When I make a GET request to "/metrics"
Then the response should NOT include header "Content-Security-Policy"
And the response should NOT include header "X-Frame-Options"
And valid Prometheus scraping should succeed
```

### Scenario: CORS Pre-flight Request
```gherkin
Given CORS is configured with allowed origin "https://example.com"
When I send an OPTIONS request with headers:
  | Origin | https://example.com |
  | Access-Control-Request-Method | GET |
  | Access-Control-Request-Headers | X-Service-Name |
Then the response status should be 204
And the response should include header "Access-Control-Allow-Origin" with value "https://example.com"
And the response should include header "Access-Control-Allow-Methods" with allowed methods
And the response should include header "Access-Control-Allow-Headers" with allowed headers
And the response should include header "Access-Control-Max-Age" with value "86400"
And no authorization check should be performed
```

### Scenario: CORS Rejected Origin
```gherkin
Given CORS is configured with allowed origin "https://example.com"
When I send an OPTIONS request with header "Origin" as "https://malicious.com"
Then the response should NOT include "Access-Control-Allow-Origin" header
```

### Scenario: Rate Limiting Enforced
```gherkin
Given rate limiting is enabled with 10 requests per second
When I make more than 10 requests within 1 second
Then after the 10th request, response status should be 429
And the response body should have "error" containing "rate limit"
And responses should include header "X-RateLimit-Limit"
And responses should include header "X-RateLimit-Remaining"
And responses should include header "X-RateLimit-Reset"
```

### Scenario: Rate Limit Headers Values
```gherkin
Given rate limiting is enabled with 100 requests per second and burst 200
When I make a request to a rate-limited endpoint
Then "X-RateLimit-Limit" should be 100
And "X-RateLimit-Remaining" should be between 0 and 99
And "X-RateLimit-Reset" should be a valid Unix timestamp
```

---

## 3. API Quality

### Scenario: Request Size Limit
```gherkin
Given request body size limit is 1MB
When I send a POST request with body larger than 1MB
Then the response status should be 413
And the response body should have "error" containing "request entity too large"
```

### Scenario: API Versioning
```gherkin
Given API version is configured as "v1"
When I make a GET request to "/api/v1/internal"
Then the response status should be 200
And the response should include header "X-API-Version" with value "v1"
And the request should be routed correctly
```

### Scenario: Idempotency Key - First Request
```gherkin
Given idempotency support is enabled
When I send a POST request with header "X-Idempotency-Key" as "unique-key-123"
Then the response status should be 200
And the idempotency key should be stored
```

### Scenario: Idempotency Key - Duplicate Request
```gherkin
Given a previous request with idempotency key "unique-key-123" succeeded
When I send another request with the same idempotency key "unique-key-123"
Then the response status should be 200
And the response body should match the original response
And the backend service should NOT be called again
```

### Scenario: Idempotency Key - Invalid Format
```gherkin
Given idempotency support is enabled
When I send a request with header "X-Idempotency-Key" as "not-a-uuid"
Then the response status should be 400
And the response body should have "error" containing "invalid idempotency key"
```

### Scenario: Content-Type Validation
```gherkin
Given service expects application/json or text/plain
When I send a request with header "Content-Type" as "application/xml"
Then the response status should be 415
And the response body should have "error" containing "unsupported media type"
```

---

## 4. Resilience

### Scenario: Circuit Breaker Opens After Failures
```gherkin
Given circuit breaker is enabled with failure threshold 5
And the downstream service is unreachable
When I make 6 requests that all fail
Then the circuit breaker should transition to "open" state
And subsequent requests should fail immediately without calling downstream
And response should be 503 Service Unavailable
And logs should include "circuit breaker opened"
```

### Scenario: Circuit Breaker Half-Open
```gherkin
Given circuit breaker is in "open" state
And timeout period has elapsed (30 seconds)
When I make a request
Then the circuit breaker should transition to "half-open" state
And logs should include "circuit breaker half-open"
```

### Scenario: Circuit Breaker Closes After Recovery
```gherkin
Given circuit breaker is in "half-open" state
And the downstream service is now healthy
When I make 2 successful requests
Then the circuit breaker should transition to "closed" state
And logs should include "circuit breaker closed"
```

### Scenario: Connection Pool Configuration
```gherkin
Given HTTP client is configured with max idle connections 100
When multiple requests are made
Then the client should maintain up to 100 idle connections
And idle connections should timeout after 90 seconds
```

### Scenario: Timeout Enforcement
```gherkin
Given HTTP client timeout is configured as 5 seconds
When calling a slow downstream service (> 5 seconds)
Then the request should be cancelled after 5 seconds
And response should be 504 Gateway Timeout
And the slow connection should be closed
```

---

## 5. Configuration

### Scenario: Configuration Reload on SIGHUP
```gherkin
Given a running service with log level "info"
When I send SIGHUP signal to the process
And configuration is reloaded with log level "debug"
Then new requests should use debug logging
And logs should include "configuration reloaded"
```

### Scenario: Invalid Configuration on Reload
```gherkin
Given a running service
When I send SIGHUP signal with invalid configuration
Then the service should keep previous configuration
And logs should include "configuration reload failed"
And the service should continue running
```

### Scenario: Feature Flags Control Features
```gherkin
Given feature flag FEATURE_RATE_LIMITING is set to "false"
When I make more than 100 requests per second
Then all requests should succeed
And no rate limiting should be applied
```

---

## 6. Operational Excellence

### Scenario: Graceful Shutdown
```gherkin
Given a service receiving requests
When shutdown signal is received
Then the service should stop accepting new connections
And in-flight requests should be completed
And readiness probe should return unhealthy
And after 30 seconds, remaining requests should be terminated
And logs should include "shutting down gracefully"
```

### Scenario: Service Version Endpoint
```gherkin
Given a running service
When I make a GET request to "/version"
Then the response status should be 200
And the response body should have "version" field
And the response body should have "git_commit" field
And the response body should have "build_time" field
```

### Scenario: Service Version in Health Check
```gherkin
Given a running service
When I make a GET request to "/health/ready"
Then the response body should have "version" field
```

---

## 7. End-to-End Scenarios

### Scenario: Complete Authorization Flow with Observability
```gherkin
Given service-caller-a has "can_call" relation to service-internal
And service-caller-b does NOT have "can_call" relation to service-internal
When service-caller-a calls /internal endpoint with request ID "req-001"
Then service-caller-a should receive 200 response
And Prometheus metrics should show authorization_success_total incremented
And logs should show correlation ID "req-001"
When service-caller-b calls /internal endpoint with request ID "req-002"
Then service-caller-b should receive 403 Forbidden
And Prometheus metrics should show authorization_failure_total incremented
And logs should show correlation ID "req-002"
```

### Scenario: Rate Limiting with Metrics
```gherkin
Given rate limiting is configured at 10 requests per second
When I make 15 rapid requests
Then first 10 should return 200
And last 5 should return 429
And Prometheus metrics should show rate_limit_exceeded_total incremented
And each response should include X-RateLimit-* headers
```

### Scenario: Resilience Under OpenFGA Failure
```gherkin
Given service-internal is running with healthy OpenFGA
And OpenFGA becomes unreachable
When a new request arrives
Then the service should fail-closed (return 503)
And readiness probe should show OpenFGA as unhealthy
And logs should show OpenFGA connection error
When OpenFGA becomes reachable again
Then after 2 successful health checks, readiness should be "ready"
And subsequent requests should succeed
```

---

## Checklist Format

### Observability
- [ ] `/metrics` endpoint returns Prometheus-formatted metrics
- [ ] HTTP request metrics include method, path, status labels
- [ ] FGA authorization metrics track allowed/denied/errors
- [ ] Cache hit/miss ratio is measurable via metrics
- [ ] `X-Request-ID` correlation is propagated through services
- [ ] All logs include service name and request ID
- [ ] `/health/live` returns 200 for live process
- [ ] `/health/ready` checks all dependencies
- [ ] Health endpoints bypass authorization

### Security
- [ ] `X-Content-Type-Options: nosniff` on all responses
- [ ] `X-Frame-Options: DENY` on all responses
- [ ] `Strict-Transport-Security` on all responses
- [ ] CORS pre-flight returns correct headers
- [ ] CORS respects allowed origins configuration
- [ ] Rate limiting returns 429 when exceeded
- [ ] Rate limit headers included in all responses
- [ ] `/metrics` endpoint compatible with Prometheus (no CSP)

### API Quality
- [ ] Request body > 1MB returns 413
- [ ] Invalid Content-Type returns 415
- [ ] API versioning via `/api/v1/` path prefix
- [ ] `X-API-Version` header on API responses
- [ ] Idempotency key stored and reused
- [ ] Duplicate idempotency key returns cached response
- [ ] Invalid idempotency key returns 400

### Resilience
- [ ] Circuit breaker opens after configured failures
- [ ] Circuit breaker half-opens after timeout
- [ ] Circuit breaker closes after recovery threshold
- [ ] Connection pool respects configured limits
- [ ] Request timeout enforced
- [ ] Slow requests cancelled after timeout

### Configuration
- [ ] SIGHUP triggers configuration reload
- [ ] Invalid configuration on reload is rejected
- [ ] Feature flags enable/disable features
- [ ] All configuration validated at startup

### Operational Excellence
- [ ] Graceful shutdown completes in-flight requests
- [ ] Readiness returns unhealthy during shutdown
- [ ] `/version` endpoint returns build info
- [ ] Version included in health check response
- [ ] All dependencies documented in health check
