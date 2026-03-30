# Acceptance Criteria: Middleware Test Coverage

## Gherkin Format

```gherkin
Feature: Middleware Test Coverage for service-caller

  As a Go developer
  I want comprehensive unit tests for all middleware components
  So that I can ensure reliability and prevent regressions
```

---

## 1. Validation Middleware Tests

### Scenario: RequestValidationMiddleware accepts valid Content-Types
```gherkin
Given a gin context with Content-Type "application/json"
When RequestValidationMiddleware processes the request
Then the middleware should call c.Next()
And no abort should occur
```

### Scenario: RequestValidationMiddleware rejects invalid Content-Type
```gherkin
Given a gin context with Content-Type "application/xml"
When RequestValidationMiddleware processes the request
Then the response status should be 415
And the response body should contain "unsupported media type"
And no further middleware should be executed
```

### Scenario: RequestValidationMiddleware rejects oversized body
```gherkin
Given a gin context with Content-Length > 1MB
When RequestValidationMiddleware processes the request
Then the response status should be 413
And the response body should contain "request entity too large"
```

### Scenario: isValidContentType validates correctly
```gherkin
Given the isValidContentType function
When checking "application/json"
Then it should return true
When checking "text/plain"
Then it should return true
When checking "application/xml"
Then it should return false
```

---

## 2. Security Middleware Tests

### Scenario: SecurityHeadersMiddleware adds all security headers
```gherkin
Given a gin context for a protected endpoint
When SecurityHeadersMiddleware processes the request
Then the response should include "X-Content-Type-Options: nosniff"
And the response should include "X-Frame-Options: DENY"
And the response should include "X-XSS-Protection: 1; mode=block"
And the response should include "Strict-Transport-Security"
And the response should include "Content-Security-Policy: default-src 'none'"
```

### Scenario: SecurityHeadersMiddleware skips metrics endpoint
```gherkin
Given a gin context with path "/metrics"
When SecurityHeadersMiddleware processes the request
Then no security headers should be added
And c.Next() should be called
```

---

## 3. Rate Limit Middleware Tests

### Scenario: RateLimiter allows requests when tokens available
```gherkin
Given a RateLimiter with rate 10 and burst 5
When calling Allow with key "client1"
Then it should return allowed=true
And remaining tokens should be 4
```

### Scenario: RateLimiter denies requests when no tokens
```gherkin
Given a RateLimiter with burst 1 that has been exhausted
When calling Allow with key "client1"
Then it should return allowed=false
And reset time should be in the future
```

### Scenario: RateLimiter refills tokens over time
```gherkin
Given a RateLimiter with rate 10 tokens/sec and burst 1
And one token has been consumed
When waiting 200ms
And calling Allow with key "client1"
Then it should return allowed=true
And 1-2 tokens should have been refilled
```

### Scenario: RateLimitMiddleware returns 429 when rate limited
```gherkin
Given a RateLimiter that denies requests
And the RateLimitMiddleware is applied
When a request is made
Then the response status should be 429
And the response body should contain "rate limit exceeded"
```

### Scenario: RateLimitMiddleware sets rate limit headers
```gherkin
Given a RateLimiter with burst 100
And the RateLimitMiddleware is applied
When a request is made
Then the response should include "X-RateLimit-Limit"
And the response should include "X-RateLimit-Remaining"
And the response should include "X-RateLimit-Reset"
```

### Scenario: RateLimitMiddleware passes when disabled
```gherkin
Given a RateLimiter with enabled=false
And the RateLimitMiddleware is applied
When a request is made
Then all requests should be allowed
And c.Next() should be called
```

---

## 4. Metrics Middleware Tests

### Scenario: MetricsMiddleware records request metrics
```gherkin
Given a request to "/test/path" with method "GET"
When MetricsMiddleware processes the request
Then HTTPRequestsTotal should be incremented with labels (GET, /test/path, 200)
And HTTPRequestDuration should record the request duration
```

### Scenario: MetricsMiddleware handles unregistered routes
```gherkin
Given a request to an unregistered route
When MetricsMiddleware processes the request
Then it should use c.Request.URL.Path instead of c.FullPath()
And metrics should still be recorded
```

### Scenario: MetricsMiddleware records error status codes
```gherkin
Given a request that returns 500
When MetricsMiddleware processes the request
Then HTTPRequestsTotal should be incremented with status "500"
```

---

## 5. Health Middleware Tests

### Scenario: Liveness probe returns healthy status
```gherkin
Given a HealthHandler with serviceName "test-service"
When Liveness endpoint is called
Then the response status should be 200
And the response body should have "status" = "alive"
And the response body should have "service" = "test-service"
And the response body should have "timestamp"
```

### Scenario: Readiness probe returns healthy when all dependencies OK
```gherkin
Given a HealthHandler with healthy dependencies
When Readiness endpoint is called
Then the response status should be 200
And the response body should have "status" = "ready"
And "dependencies" should indicate healthy
```

### Scenario: Readiness probe returns unhealthy when shutting down
```gherkin
Given SetShuttingDown(true) has been called
And a HealthHandler is configured
When Readiness endpoint is called
Then the response status should be 503
And the response body should have "status" = "not_ready"
And the response body should have "reason" = "shutting_down"
```

### Scenario: Readiness probe returns unhealthy when dependency fails
```gherkin
Given a HealthHandler with an unhealthy dependency
When Readiness endpoint is called
Then the response status should be 503
And the response body should have "status" = "not_ready"
And "dependencies" should indicate unhealthy
```

### Scenario: Version endpoint returns build info
```gherkin
Given a VersionHandler with version "1.0.0", gitCommit "abc123", buildTime "2024-01-01"
When GetVersion endpoint is called
Then the response should have "version" = "1.0.0"
And the response should have "git_commit" = "abc123"
And the response should have "build_time" = "2024-01-01"
```

---

## 6. Idempotency Middleware Tests

### Scenario: IdempotencyMiddleware disabled passes through
```gherkin
Given IdempotencyMiddleware with enabled=false
When a POST request is made
Then c.Next() should be called immediately
```

### Scenario: IdempotencyMiddleware skips non-POST requests
```gherkin
Given IdempotencyMiddleware with enabled=true
When a GET request is made
Then c.Next() should be called immediately
```

### Scenario: IdempotencyMiddleware rejects invalid UUID key
```gherkin
Given IdempotencyMiddleware with enabled=true
And an IdempotencyStore
When a POST request with header "X-Idempotency-Key" = "not-a-uuid" is made
Then the response status should be 400
And the response body should contain "invalid idempotency key"
```

### Scenario: IdempotencyMiddleware returns cached response
```gherkin
Given an IdempotencyStore with stored entry for key "valid-uuid-here"
When IdempotencyMiddleware receives request with that key
Then the response status should be the stored status
And the response body should be the stored body
And c.Next() should NOT be called
```

### Scenario: IdempotencyStore stores and retrieves entries
```gherkin
Given an IdempotencyStore with TTL 1 hour
When Set is called with key "test", status 200, body "response"
Then Get should return status 200, body "response", found true
```

### Scenario: IdempotencyStore returns false for expired entries
```gherkin
Given an IdempotencyStore with TTL 1ms
And an entry that has expired
When Get is called for that entry
Then it should return found false
```

### Scenario: IdempotencyStore cleanup removes expired entries
```gherkin
Given an IdempotencyStore with mixed expired and valid entries
When Cleanup is called
Then expired entries should be removed
And valid entries should remain
```

### Scenario: isValidUUID validates correctly
```gherkin
Given isValidUUID function
When checking "550e8400-e29b-41d4-a716-446655440000"
Then it should return true
When checking "not-a-uuid"
Then it should return false
When checking "550e8400-e29b-41d4-a716"
Then it should return false
```

---

## 7. Correlation Middleware Tests

### Scenario: CorrelationIDMiddleware generates new ID
```gherkin
Given a gin context without X-Request-ID header
When CorrelationIDMiddleware processes the request
Then a new UUID should be generated
And set as X-Request-ID header
And stored in context as "request_id"
```

### Scenario: CorrelationIDMiddleware uses provided ID
```gherkin
Given a gin context with X-Request-ID "existing-id-123"
When CorrelationIDMiddleware processes the request
Then the header should remain "existing-id-123"
And context should have "request_id" = "existing-id-123"
```

### Scenario: CorrelationIDMiddleware captures caller service
```gherkin
Given a gin context with X-Service-Name "caller-service"
When CorrelationIDMiddleware processes the request
Then context should have "caller_service" = "caller-service"
```

### Scenario: RequestLoggerMiddleware logs request completion
```gherkin
Given a completed request with status 200
When RequestLoggerMiddleware processes
Then a log entry should be created with method, path, status, duration
And log should include request_id from context
```

### Scenario: RequestLoggerMiddleware uses correct log levels
```gherkin
Given a request with status 500
When RequestLoggerMiddleware processes
Then zap.Error should be called
Given a request with status 400
When RequestLoggerMiddleware processes
Then zap.Warn should be called
Given a request with status 200
When RequestLoggerMiddleware processes
Then zap.Info should be called
```

---

## 8. CORS Middleware Tests

### Scenario: CORSMiddleware allows configured origin
```gherkin
Given CORS config with allowed origin "https://example.com"
And a request with Origin "https://example.com"
When CORSMiddleware processes the request
Then response should include "Access-Control-Allow-Origin: https://example.com"
And response should include "Access-Control-Allow-Credentials: true"
```

### Scenario: CORSMiddleware rejects disallowed origin
```gherkin
Given CORS config with allowed origin "https://example.com"
And a request with Origin "https://other.com"
When CORSMiddleware processes the request
Then response should NOT include "Access-Control-Allow-Origin"
```

### Scenario: CORSMiddleware allows wildcard origin
```gherkin
Given CORS config with allowed origin "*"
And a request with Origin "https://any.com"
When CORSMiddleware processes the request
Then response should include "Access-Control-Allow-Origin: *"
```

### Scenario: CORSMiddleware handles OPTIONS preflight
```gherkin
Given an OPTIONS request
When CORSMiddleware processes the request
Then the response status should be 204
And c.Next() should NOT be called
```

### Scenario: isOriginAllowed validates correctly
```gherkin
Given isOriginAllowed with allowed ["https://example.com", "*"]
When checking "https://example.com"
Then it should return true
When checking "https://any.com"
Then it should return true (wildcard)
When checking "https://other.com"
Then it should return false
```

---

## 9. Circuit Breaker Middleware Tests

### Scenario: CircuitBreakerExecute runs function when closed
```gherkin
Given CircuitBreaker initialized and in closed state
When CircuitBreakerExecute is called with a function
Then the function should be executed
And the result should be returned
```

### Scenario: CircuitBreakerExecute returns error when open
```gherkin
Given CircuitBreaker initialized and in open state
When CircuitBreakerExecute is called with a function
Then the function should NOT be executed
And an error should be returned
```

### Scenario: CircuitBreakerHTTPExecute handles HTTP functions
```gherkin
Given CircuitBreaker initialized
When CircuitBreakerHTTPExecute is called
Then it should execute the HTTP function
And return response, body, error
```

### Scenario: CircuitBreakerExecute returns zero value when nil
```gherkin
Given CircuitBreaker is nil (not initialized)
When CircuitBreakerExecute is called with a function
Then the function should be executed directly
And result should be returned without circuit breaker
```

---

## Checklist Format

### General
- [ ] All middleware files have corresponding test files
- [ ] Test files use `_test.go` suffix
- [ ] Tests follow naming convention `Test<Component>_<Scenario>`
- [ ] Tests use table-driven pattern for multiple scenarios

### Validation Middleware
- [ ] Test accepts application/json Content-Type
- [ ] Test accepts text/plain Content-Type
- [ ] Test rejects invalid Content-Type (415)
- [ ] Test rejects oversized body (413)
- [ ] Test isValidContentType edge cases

### Security Middleware
- [ ] Test all 5 security headers present
- [ ] Test /metrics endpoint bypass

### Rate Limit Middleware
- [ ] Test TokenBucket allows when tokens available
- [ ] Test TokenBucket denies when no tokens
- [ ] Test TokenBucket refills over time
- [ ] Test RateLimitMiddleware returns 429
- [ ] Test rate limit headers present
- [ ] Test disabled limiter passes through

### Metrics Middleware
- [ ] Test request counter increments
- [ ] Test duration histogram records
- [ ] Test unregistered route handling
- [ ] Test RecordRateLimitExceeded
- [ ] Test RecordCircuitBreakerState

### Health Middleware
- [ ] Test Liveness returns 200
- [ ] Test Readiness returns 200 when healthy
- [ ] Test Readiness returns 503 when shutting down
- [ ] Test Readiness returns 503 when dependency unhealthy
- [ ] Test Version endpoint returns info
- [ ] Test atomic shutdown flag

### Idempotency Middleware
- [ ] Test disabled passes through
- [ ] Test non-POST passes through
- [ ] Test missing key passes through
- [ ] Test invalid UUID returns 400
- [ ] Test cached response returned
- [ ] Test IdempotencyStore.Get/Set
- [ ] Test expired entry handling
- [ ] Test Cleanup removes expired

### Correlation Middleware
- [ ] Test new ID generation
- [ ] Test existing ID preserved
- [ ] Test caller service captured
- [ ] Test logger called on completion
- [ ] Test log level based on status

### CORS Middleware
- [ ] Test allowed origin gets headers
- [ ] Test disallowed origin rejected
- [ ] Test wildcard origin
- [ ] Test OPTIONS preflight returns 204
- [ ] Test parseOrigins splitting
- [ ] Test isOriginAllowed matching

### Circuit Breaker Middleware
- [ ] Test InitCircuitBreaker configuration
- [ ] Test Execute runs function (closed)
- [ ] Test Execute returns error (open)
- [ ] Test HTTPExecute handles response
- [ ] Test nil breaker executes directly
