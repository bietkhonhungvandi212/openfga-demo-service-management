# Requirements: Middleware Test Coverage for service-caller

## 1. Overview

This document defines requirements for adding comprehensive test coverage for all middleware components in the `service-caller` package.

**Scope:** `service-caller/middleware/` directory  
**Test Framework:** Go standard testing package with `net/http/httptest`  
**Target Coverage:** All middleware components and their edge cases

---

## 2. Middleware Components to Test

### 2.1 Validation Middleware (`validation.go`)

| ID | Requirement |
|----|-------------|
| REQ-VAL-001 | Test `RequestValidationMiddleware` rejects invalid Content-Type with 415 |
| REQ-VAL-002 | Test `RequestValidationMiddleware` accepts `application/json` Content-Type |
| REQ-VAL-003 | Test `RequestValidationMiddleware` accepts `text/plain` Content-Type |
| REQ-VAL-004 | Test `RequestValidationMiddleware` rejects request body > 1MB with 413 |
| REQ-VAL-005 | Test `RequestValidationMiddleware` allows empty Content-Type header |
| REQ-VAL-006 | Test `isValidContentType` helper function edge cases |

### 2.2 Security Middleware (`security.go`)

| ID | Requirement |
|----|-------------|
| REQ-SEC-001 | Test `SecurityHeadersMiddleware` adds all required security headers |
| REQ-SEC-002 | Test `SecurityHeadersMiddleware` skips `/metrics` endpoint |
| REQ-SEC-003 | Verify all 5 security headers are present |

### 2.3 Rate Limit Middleware (`ratelimit.go`)

| ID | Requirement |
|----|-------------|
| REQ-RAT-001 | Test `NewRateLimiter` creates limiter with correct defaults |
| REQ-RAT-002 | Test `RateLimiter.Allow` grants request when tokens available |
| REQ-RAT-003 | Test `RateLimiter.Allow` denies request when no tokens |
| REQ-RAT-004 | Test `RateLimiter.Allow` refills tokens over time |
| REQ-RAT-005 | Test `RateLimitMiddleware` returns 429 when rate limited |
| REQ-RAT-006 | Test `RateLimitMiddleware` sets rate limit headers |
| REQ-RAT-007 | Test `RateLimitMiddleware` passes when limiter disabled |
| REQ-RAT-008 | Test rate limiting per client IP |
| REQ-RAT-009 | Test bucket token refilling respects ratePerSec |

### 2.4 Metrics Middleware (`metrics.go`)

| ID | Requirement |
|----|-------------|
| REQ-MET-001 | Test `MetricsMiddleware` records request count |
| REQ-MET-002 | Test `MetricsMiddleware` records request duration |
| REQ-MET-003 | Test `MetricsMiddleware` handles unregistered routes |
| REQ-MET-004 | Test `SetServiceInfo` sets labels correctly |
| REQ-MET-005 | Test `RecordRateLimitExceeded` increments counter |
| REQ-MET-006 | Test `RecordHTTPClientRequest` records client metrics |
| REQ-MET-007 | Test `RecordCircuitBreakerState` sets gauge |

### 2.5 Health Middleware (`health.go`)

| ID | Requirement |
|----|-------------|
| REQ-HEALTH-001 | Test `HealthHandler.Liveness` returns 200 with correct structure |
| REQ-HEALTH-002 | Test `HealthHandler.Readiness` returns 200 when healthy |
| REQ-HEALTH-003 | Test `HealthHandler.Readiness` returns 503 when shutting down |
| REQ-HEALTH-004 | Test `HealthHandler.Readiness` returns 503 when dependencies unhealthy |
| REQ-HEALTH-005 | Test `HealthHandler.Readiness` with no dependencies |
| REQ-HEALTH-006 | Test `VersionHandler.GetVersion` returns version info |
| REQ-HEALTH-007 | Test `SetShuttingDown` and `IsShuttingDown` atomic operations |
| REQ-HEALTH-008 | Test health response includes required fields (status, service, timestamp) |

### 2.6 Idempotency Middleware (`idempotency.go`)

| ID | Requirement |
|----|-------------|
| REQ-IDEM-001 | Test `IdempotencyMiddleware` passes through when disabled |
| REQ-IDEM-002 | Test `IdempotencyMiddleware` passes through non-POST requests |
| REQ-IDEM-003 | Test `IdempotencyMiddleware` passes through POST without idempotency key |
| REQ-IDEM-004 | Test `IdempotencyMiddleware` returns 400 for invalid UUID key |
| REQ-IDEM-005 | Test `IdempotencyMiddleware` returns cached response for duplicate key |
| REQ-IDEM-006 | Test `IdempotencyStore.Get` returns stored entry |
| REQ-IDEM-007 | Test `IdempotencyStore.Get` returns false for expired entries |
| REQ-IDEM-008 | Test `IdempotencyStore.Set` stores entry with correct TTL |
| REQ-IDEM-009 | Test `IdempotencyStore.Cleanup` removes expired entries |
| REQ-IDEM-010 | Test `bodyLogWriter.Write` captures response body |
| REQ-IDEM-011 | Test `isValidUUID` validates UUID format correctly |
| REQ-IDEM-012 | Test `ReadBody` reads and restores request body |

### 2.7 Correlation Middleware (`correlation.go`)

| ID | Requirement |
|----|-------------|
| REQ-CORR-001 | Test `CorrelationIDMiddleware` generates new ID when none provided |
| REQ-CORR-002 | Test `CorrelationIDMiddleware` uses provided X-Request-ID |
| REQ-CORR-003 | Test `CorrelationIDMiddleware` sets request_id in context |
| REQ-CORR-004 | Test `CorrelationIDMiddleware` sets caller_service when header present |
| REQ-CORR-005 | Test `RequestLoggerMiddleware` logs request completion |
| REQ-CORR-006 | Test `RequestLoggerMiddleware` logs include all required fields |
| REQ-CORR-007 | Test logging level based on status code (error/warn/info) |

### 2.8 CORS Middleware (`cors.go`)

| ID | Requirement |
|----|-------------|
| REQ-CORS-001 | Test `CORSMiddleware` sets CORS headers for allowed origin |
| REQ-CORS-002 | Test `CORSMiddleware` rejects disallowed origin |
| REQ-CORS-003 | Test `CORSMiddleware` handles wildcard origin `*` |
| REQ-CORS-004 | Test `CORSMiddleware` returns 204 for OPTIONS preflight |
| REQ-CORS-005 | Test `parseOrigins` splits comma-separated origins |
| REQ-CORS-006 | Test `isOriginAllowed` matches exact and wildcard origins |
| REQ-CORS-007 | Test `formatInt` converts integers to string correctly |

### 2.9 Circuit Breaker Middleware (`circuitbreaker.go`)

| ID | Requirement |
|----|-------------|
| REQ-CB-001 | Test `InitCircuitBreaker` initializes with correct config |
| REQ-CB-002 | Test `GetCircuitBreaker` returns initialized breaker |
| REQ-CB-003 | Test `CircuitBreakerExecute` executes function when closed |
| REQ-CB-004 | Test `CircuitBreakerExecute` returns error when open |
| REQ-CB-005 | Test `CircuitBreakerHTTPExecute` executes HTTP function |
| REQ-CB-006 | Test circuit breaker returns zero value when breaker nil |
| REQ-CB-007 | Test `HTTPResult` struct correctly holds response data |
| REQ-CB-008 | Test circuit breaker state transitions (open/half-open/closed) |

---

## 3. Test File Structure

| Test File | Package to Test |
|-----------|-----------------|
| `validation_test.go` | `validation.go` |
| `security_test.go` | `security.go` |
| `ratelimit_test.go` | `ratelimit.go` |
| `metrics_test.go` | `metrics.go` |
| `health_test.go` | `health.go` |
| `idempotency_test.go` | `idempotency.go` |
| `correlation_test.go` | `correlation.go` |
| `cors_test.go` | `cors.go` |
| `circuitbreaker_test.go` | `circuitbreaker.go` |

---

## 4. Test Naming Convention

- Format: `Test<Component>_<Scenario>`
- Examples:
  - `TestRequestValidationMiddleware_RejectsInvalidContentType`
  - `TestRateLimiter_AllowsRequestWhenTokensAvailable`
  - `TestIdempotencyStore_ReturnsCachedResponse`

---

## 5. Testing Patterns

### 5.1 HTTP Handler Testing
- Use `httptest.NewRecorder` and `httptest.NewRequest`
- Test middleware chain with `gin.CreateTestContext`

### 5.2 Table-Driven Tests
- Use table-driven tests for multiple input scenarios
- Include subtests for granular test reporting

### 5.3 Mock Dependencies
- Mock `DependencyChecker` interface for health checks
- Avoid external dependencies (network calls)

---

## 6. Dependencies

Required test packages:
- `net/http/httptest` - HTTP testing utilities
- `github.com/gin-gonic/gin` - Web framework (already in dependencies)
- `github.com/google/uuid` - UUID generation (already in dependencies)

---

## 7. Out of Scope

- Integration tests with real HTTP servers
- End-to-end tests across services
- Load/performance testing
- Database or external service tests
