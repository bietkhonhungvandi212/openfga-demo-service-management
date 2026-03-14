# Service Management with OpenFGA

## Overview

This project demonstrates service-to-service authorization using **OpenFGA** (Open Fine-Grained Authorization) for production-ready service management.

## Architecture

```plaintext
                          +-----------------------+
                          |       OpenFGA         |
                          |  Authorization Server |
                          |  (relationship store) |
                          |      (PostgreSQL)     |
                          +-----------+-----------+
                                      ^
                                      |
                              Permission Check
                                      |
                              +-------+--------+
                              |  Service A     |
                              |  (Internal API)|
                              |----------------|
                              | Auth Middleware|
                              |  (Caching)     |
                              +-------+--------+
                                      ^
                         HTTP Request |
                          +-----------+-----------+
                          |                       |
                 +--------+--------+     +--------+--------+
                 | Service Caller A |     | Service Caller C |
                 |  (Allowed)       |     |   (Denied)       |
                 +------------------+     +------------------+
```

## Key Features

### OpenFGA Cluster
- **PostgreSQL** backend for production-grade persistence
- **JSON logging** for structured observability
- **Metrics endpoint** on port 8081
- **Health checks** for all services
- **Graceful shutdown** handling

### Service-Internal
- **In-memory caching** for FGA check results (configurable TTL)
- **Retry logic** with exponential backoff for FGA calls
- **Structured logging** with zap
- **Request tracing** with timing information
- **Graceful shutdown** on SIGTERM/SIGINT

### Service-Caller
- **HTTP client timeout** configuration
- **Structured logging** with zap
- **Health check endpoint**
- **Graceful shutdown**

## Quick Start

### Prerequisites
- Docker & Docker Compose
- OpenFGA CLI (optional, for setup): `brew install openfga/tap/fga`

### 1. Start the Stack
```bash
docker compose up
```

### 2. Set up OpenFGA (if not already done)
```bash
# Create store
fga store create --name service-auth

# Note the store ID from response, then update docker-compose.yml
# Set STORE_ID environment variable in service-internal service

# Write authorization model
fga model write \
  --store-id=<YOUR_STORE_ID> \
  --file=model/service-model.fga

# Define permissions
fga tuple write service:service-caller-a can_call service:service-internal-a \
  --store-id=<YOUR_STORE_ID>
```

### 3. Test
```bash
# Allowed caller
curl 'http://localhost:8083/internal'

# Denied caller (if not authorized)
curl 'http://localhost:8084/internal'
```

## Production Deployment

### Environment Variables

#### OpenFGA
- `OPENFGA_API_URL`: OpenFGA server URL (default: http://openfga:8080)
- `STORE_ID`: OpenFGA store ID (required)

#### Service-Internal
- `PORT`: Service port (default: 8080)
- `NAME`: Service identifier (default: service-internal-a)
- `LOG_LEVEL`: Logging level (debug, info, warn, error)
- `LOG_FORMAT`: Logging format (json, console)
- `FGA_CACHE_ENABLED`: Enable FGA result caching (true/false)
- `FGA_CACHE_TTL`: Cache TTL duration (e.g., 60s, 5m)
- `FGA_RETRY_MAX_ATTEMPTS`: Max retry attempts for FGA calls
- `FGA_RETRY_BACKOFF`: Initial backoff duration

#### Service-Caller
- `PORT`: Service port (default: 8081)
- `NAME`: Service identifier (default: service-caller-a)
- `SERVICE_INTERNAL_A_URL`: Target internal service URL
- `HTTP_CLIENT_TIMEOUT`: HTTP client timeout (default: 5s)
- `HTTP_CLIENT_RETRY_MAX_ATTEMPTS`: Max retry attempts

### Docker Compose Settings

Update `docker-compose.yml` for production:

```yaml
openfga:
  environment:
    - OPENFGA_AUTH_METHOD=Bearer
    - OPENFGA_AUTH_BEARER_TOKEN=<your-token>
  ports:
    - "8080:8080"  # OpenFGA API
    - "8081:8081"  # Metrics
  volumes:
    - ./openfga-data:/data  # Persistent storage
```

### Logging
All services use structured JSON logging by default. Logs include:
- Timestamp
- Service name
- Request path and method
- Response status and duration
- Error messages with stack traces

### Monitoring
- **OpenFGA Metrics**: http://localhost:8081/metrics
- **Service Health**: http://localhost:8080/health
- **Service Metrics**: Available via JSON logs

### Best Practices
1. **Use environment variables** for sensitive configuration
2. **Enable caching** for production workloads
3. **Configure appropriate timeouts** for your workload
4. **Use JSON logging** for centralized log aggregation
5. **Enable health checks** in your orchestration system
6. **Configure resource limits** in Docker/Kubernetes
7. **Use TLS** for inter-service communication in production

## Troubleshooting

### Service not authorized
Check OpenFGA tuples:
```bash
fga tuple list --store-id=<STORE_ID> --filter user=service:service-caller-a
```

### OpenFGA not starting
Check PostgreSQL health:
```bash
docker compose logs postgres
```

### High latency
- Enable FGA caching: `FGA_CACHE_ENABLED=true`
- Increase cache TTL: `FGA_CACHE_TTL=5m`
- Check network latency between services

## Folder Structure

```plaintext
servicer-management/
├── docker-compose.yml          # Services orchestration
├── README.md                   # This file
├── model/
│   └── service-model.fga       # OpenFGA authorization model
├── service-internal/
│   ├── dockerfile              # Docker configuration
│   ├── go.mod                  # Go dependencies
│   └── main.go                 # Internal service with auth middleware
└── service-caller/
    ├── dockerfile              # Docker configuration
    ├── go.mod                  # Go dependencies
    └── main.go                 # Caller service
```

## Authorization Model

The `service-model.fga` defines a simple service-to-service authorization:

```fga
type service
  relations
    define can_call: [service]
```

This allows services to grant calling permissions to other services.
