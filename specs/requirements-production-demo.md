# Production Demo Requirements: Service-Management Distributed System

## 1. Overview

This document defines requirements for enhancing the existing service-caller + service-internal distributed system to become production-ready for demo purposes. The system demonstrates practical distributed system scenarios including authorization, observability, resilience, and operational excellence patterns.

**Current System Components:**
- `service-internal`: Protected service requiring OpenFGA authorization (Go/Gin)
- `service-caller`: External-facing service that proxies requests (Go/Gin)
- OpenFGA: Authorization service with PostgreSQL backend
- PostgreSQL: Data store for OpenFGA

## 2. Functional Requirements

### 2.1 Observability

#### 2.1.1 Prometheus Metrics
- **REQ-OBS-001**: Services MUST expose `/metrics` endpoint with Prometheus-compatible format
- **REQ-OBS-002**: Custom metrics REQUIRED:
  - `http_requests_total` (counter): Total HTTP requests by method, path, status
  - `http_request_duration_seconds` (histogram): Request latency distribution
  - `fga_authorization_total` (counter): Authorization decisions by result (allowed/denied/error)
  - `fga_cache_hits_total` (counter): OpenFGA cache hit/miss ratio
  - `fga_check_duration_seconds` (histogram): OpenFGA check latency
  - `http_client_requests_total` (counter): Outbound HTTP client requests
  - `http_client_request_duration_seconds` (histogram): Outbound request latency
- **REQ-OBS-003**: OpenFGA metrics endpoint (`:8081/metrics`) MUST be accessible for scraping
- **REQ-OBS-004**: Service metrics MUST include labels: `service_name`, `instance`

#### 2.1.2 Structured Logging Enhancement
- **REQ-OBS-005**: Logs MUST include correlation ID (`X-Request-ID`) for distributed tracing
- **REQ-OBS-006**: Logs MUST include caller service name from `X-Service-Name` header
- **REQ-OBS-007**: Error logs MUST include stack traces in non-production environments
- **REQ-OBS-008**: Log sampling for high-volume endpoints (configurable rate)

#### 2.1.3 Health Checks
- **REQ-OBS-009**: Services MUST expose separate `/health/live` (liveness) and `/health/ready` (readiness) endpoints
- **REQ-OBS-010**: Liveness probe checks: HTTP server is responding, process is not deadlocked
- **REQ-OBS-011**: Readiness probe checks:
  - OpenFGA connectivity (for service-internal)
  - Dependencies reachable (for service-caller)
  - No ongoing graceful shutdown
- **REQ-OBS-012**: Health endpoints MUST be excluded from authorization middleware
- **REQ-OBS-013**: Health response MUST include timestamp, service name, version, and status of dependencies

### 2.2 Security

#### 2.2.1 Security Headers
- **REQ-SEC-001**: All HTTP responses MUST include security headers:
  - `X-Content-Type-Options: nosniff`
  - `X-Frame-Options: DENY`
  - `X-XSS-Protection: 1; mode=block`
  - `Strict-Transport-Security: max-age=31536000; includeSubDomains`
  - `Content-Security-Policy: default-src 'none'`
- **REQ-SEC-002**: Security headers MUST NOT be applied to `/metrics` endpoint (for Prometheus compatibility)

#### 2.2.2 CORS Handling
- **REQ-SEC-003**: CORS middleware MUST be configurable via environment variables
- **REQ-SEC-004**: Default CORS configuration:
  - `CORS_ALLOWED_ORIGINS`: comma-separated list of allowed origins (default: "")
  - `CORS_ALLOWED_METHODS`: GET,POST,PUT,DELETE,OPTIONS (default)
  - `CORS_ALLOWED_HEADERS`: Content-Type,X-Service-Name,X-Request-ID,X-Idempotency-Key
  - `CORS_MAX_AGE`: 86400 seconds (24 hours)
- **REQ-SEC-005**: Pre-flight requests MUST be handled without authorization check

#### 2.2.3 Rate Limiting
- **REQ-SEC-006**: Rate limiting middleware MUST be implemented using token bucket algorithm
- **REQ-SEC-007**: Rate limits MUST be configurable per service caller via header or config
- **REQ-SEC-008**: Rate limit configuration:
  - `RATE_LIMIT_ENABLED`: Enable/disable rate limiting (default: true)
  - `RATE_LIMIT_REQUESTS_PER_SECOND`: Requests per second (default: 100)
  - `RATE_LIMIT_BURST`: Burst capacity (default: 200)
- **REQ-SEC-009**: Rate limit responses MUST include headers:
  - `X-RateLimit-Limit`: Maximum requests allowed
  - `X-RateLimit-Remaining`: Remaining requests in window
  - `X-RateLimit-Reset`: Unix timestamp when limit resets
- **REQ-SEC-010**: Rate limit exceeded response: HTTP 429 Too Many Requests

### 2.3 API Quality

#### 2.3.1 Request/Response Validation
- **REQ-API-001**: Request body size MUST be limited (default: 1MB)
- **REQ-API-002**: Unknown request headers MUST be logged but not rejected
- **REQ-API-003**: Request Content-Type MUST be validated (application/json or text/plain)
- **REQ-API-004**: Response body MUST include proper Content-Type header

#### 2.3.2 API Versioning
- **REQ-API-005**: URL path versioning MUST be supported: `/api/v1/*`
- **REQ-API-006**: Version MUST be configurable via environment variable (`API_VERSION`, default: v1)
- **REQ-API-007**: Unversioned `/health` endpoints MUST remain accessible
- **REQ-API-008**: API version MUST be included in response headers: `X-API-Version`

#### 2.3.3 Idempotency
- **REQ-API-009**: Idempotency key support via `X-Idempotency-Key` header
- **REQ-API-010**: Idempotency keys MUST be validated (UUID format recommended)
- **REQ-API-011**: Idempotency cache TTL MUST be configurable (default: 24 hours)
- **REQ-API-012**: Repeated requests with same idempotency key MUST return cached response
- **REQ-API-013**: POST requests without idempotency key MUST be allowed (new operation)

### 2.4 Resilience

#### 2.4.1 Connection Pooling (HTTP Client)
- **REQ-RES-001**: HTTP client MUST use explicit connection pool configuration
- **REQ-RES-002**: Connection pool configuration:
  - `HTTP_MAX_IDLE_CONNECTIONS`: Maximum idle connections (default: 100)
  - `HTTP_MAX_IDLE_PER_HOST`: Maximum idle per host (default: 10)
  - `HTTP_IDLE_CONNECTION_TIMEOUT`: Idle timeout (default: 90s)
- **REQ-RES-003**: HTTP client MUST support TLS configuration (disabled by default for internal)
- **REQ-RES-004**: Connection pool metrics MUST be exposed

#### 2.4.2 Circuit Breaker
- **REQ-RES-005**: Circuit breaker pattern MUST be implemented for HTTP client calls
- **REQ-RES-006**: Circuit breaker configuration:
  - `CIRCUIT_BREAKER_ENABLED`: Enable/disable (default: true)
  - `CIRCUIT_BREAKER_FAILURE_THRESHOLD`: Failures before opening (default: 5)
  - `CIRCUIT_BREAKER_SUCCESS_THRESHOLD`: Successes to close (default: 2)
  - `CIRCUIT_BREAKER_TIMEOUT`: Time before half-open (default: 30s)
- **REQ-RES-007**: Circuit breaker states MUST be logged (open/half-open/closed)
- **REQ-RES-008**: Circuit breaker metrics MUST be exposed (state, failure count, success count)

#### 2.4.3 Timeout Management
- **REQ-RES-009**: Per-endpoint timeout configuration via middleware
- **REQ-RES-010**: Default request timeout: 5 seconds (configurable)
- **REQ-RES-011**: Timeout MUST be enforced at both request and response levels

### 2.5 Configuration

#### 2.5.1 Dynamic Configuration
- **REQ-CFG-001**: Configuration reload via SIGHUP signal
- **REQ-CFG-002**: Configuration MUST be validated on reload
- **REQ-CFG-003**: Invalid configuration reload MUST not crash the service (keep previous config)
- **REQ-CFG-004**: Configuration changes MUST be logged at INFO level

#### 2.5.2 Feature Flags
- **REQ-CFG-005**: Feature flags MUST be configurable via environment variables
- **REQ-CFG-006**: Required feature flags:
  - `FEATURE_RATE_LIMITING`: Enable/disable rate limiting
  - `FEATURE_IDEMPOTENCY`: Enable/disable idempotency
  - `FEATURE_CIRCUIT_BREAKER`: Enable/disable circuit breaker
  - `FEATURE_FGA_CACHE`: Enable/disable FGA cache

### 2.6 Operational Excellence

#### 2.6.1 Graceful Degradation
- **REQ-OPS-001**: Service MUST continue operating if OpenFGA is temporarily unavailable (fail-closed by default)
- **REQ-OPS-002**: Service MUST expose `X-Service-Unavailable` header when operating in degraded mode
- **REQ-OPS-003**: Degraded mode operations MUST be logged with WARN level

#### 2.6.2 Graceful Shutdown
- **REQ-OPS-004**: Graceful shutdown timeout MUST be configurable (default: 30 seconds)
- **REQ-OPS-005**: In-flight requests MUST be completed before shutdown
- **REQ-OPS-006**: Readiness probe MUST return unhealthy during shutdown
- **REQ-OPS-007**: Shutdown MUST be logged with INFO level including reason

#### 2.6.3 Service Metadata
- **REQ-OPS-008**: Service version MUST be exposed via `/version` endpoint and in health checks
- **REQ-OPS-009**: Build information (git commit, build time) MUST be included in version response
- **REQ-OPS-010**: Service metadata MUST be exposed as Prometheus labels

## 3. Non-Functional Requirements

### 3.1 Performance
- **NFR-001**: P99 latency for authorization check < 50ms (with cache hit)
- **NFR-002**: P99 latency for authorization check < 500ms (with cache miss, OpenFGA healthy)
- **NFR-003**: Memory usage < 100MB per service instance under normal load
- **NFR-004**: Startup time < 5 seconds

### 3.2 Reliability
- **NFR-005**: Services MUST handle OpenFGA unavailability gracefully
- **NFR-006**: Services MUST handle PostgreSQL unavailability gracefully
- **NFR-007**: Circuit breaker MUST prevent cascade failures

### 3.3 Maintainability
- **NFR-008**: Code MUST follow Go best practices (go fmt, go vet, static analysis)
- **NFR-009**: Unit test coverage > 70% for core business logic
- **NFR-010**: All environment variables MUST have defaults or be validated at startup

## 4. Demo Scenarios

The enhanced system MUST support demonstration of the following scenarios:

### 4.1 Authorization Flow
- Multiple callers (service-caller-a, service-caller-b) with different permissions
- Denied access when caller lacks `can_call` relation
- Cached authorization decisions

### 4.2 Observability
- Prometheus metrics scraping
- Correlation ID tracking across services
- Log aggregation showing request lifecycle

### 4.3 Resilience
- Circuit breaker opening under load
- Rate limiting enforcement
- Graceful shutdown demonstration

### 4.4 Security
- Security headers inspection
- CORS pre-flight handling
- Rate limit headers demonstration

## 5. Environment Variables Summary

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8080/8081 | Service port |
| `NAME` | - | Service name |
| `API_VERSION` | v1 | API version |
| `LOG_LEVEL` | info | Log level (debug/info/warn/error) |
| `LOG_FORMAT` | json | Log format (json/console) |
| `LOG_SAMPLE_RATE` | 1.0 | Log sampling rate |
| `OPENFGA_API` | - | OpenFGA API URL |
| `STORE_ID` | - | OpenFGA store ID |
| `FGA_CACHE_ENABLED` | true | Enable FGA cache |
| `FGA_CACHE_TTL` | 60s | FGA cache TTL |
| `FGA_RETRY_MAX_ATTEMPTS` | 3 | Max retry attempts |
| `FGA_RETRY_BACKOFF` | 100ms | Retry backoff |
| `HTTP_CLIENT_TIMEOUT` | 5s | HTTP client timeout |
| `HTTP_MAX_IDLE_CONNECTIONS` | 100 | Max idle connections |
| `HTTP_MAX_IDLE_PER_HOST` | 10 | Max idle per host |
| `HTTP_IDLE_CONNECTION_TIMEOUT` | 90s | Idle connection timeout |
| `CIRCUIT_BREAKER_ENABLED` | true | Enable circuit breaker |
| `CIRCUIT_BREAKER_FAILURE_THRESHOLD` | 5 | Failures to open |
| `CIRCUIT_BREAKER_SUCCESS_THRESHOLD` | 2 | Successes to close |
| `CIRCUIT_BREAKER_TIMEOUT` | 30s | Half-open timeout |
| `RATE_LIMIT_ENABLED` | true | Enable rate limiting |
| `RATE_LIMIT_REQUESTS_PER_SECOND` | 100 | Requests per second |
| `RATE_LIMIT_BURST` | 200 | Burst capacity |
| `CORS_ALLOWED_ORIGINS` | - | Allowed CORS origins |
| `CORS_ALLOWED_METHODS` | GET,POST,PUT,DELETE,OPTIONS | Allowed methods |
| `CORS_MAX_AGE` | 86400 | Preflight cache duration |
| `IDEMPOTENCY_CACHE_TTL` | 24h | Idempotency cache TTL |
| `FEATURE_RATE_LIMITING` | true | Feature flag |
| `FEATURE_IDEMPOTENCY` | true | Feature flag |
| `FEATURE_CIRCUIT_BREAKER` | true | Feature flag |
| `SHUTDOWN_TIMEOUT` | 30s | Graceful shutdown timeout |

## 6. Dependencies

Required Go packages:
- `github.com/prometheus/client_golang/prometheus` - Prometheus metrics
- `github.com/gin-contrib/cors` - CORS middleware
- `github.com/sony/gobreaker` - Circuit breaker
- `github.com/tidwall/gjson` - JSON path for response caching
- `github.com/google/uuid` - UUID validation for idempotency keys

## 7. Out of Scope

The following are explicitly OUT OF SCOPE for this enhancement:
- Database migrations for application data (only OpenFGA schema)
- Service mesh (Istio, Linkerd)
- Kubernetes-specific configurations
- Deployment automation (Terraform, Helm)
- Multi-tenancy support
- gRPC protocol support
