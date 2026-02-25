# Development Guide

## Tech Stack

| Category | Technology | Purpose |
|----------|-----------|---------|
| Language | Go 1.23 | All services |
| Web Framework | Echo v4 | HTTP routing, middleware |
| Database | PostgreSQL (pgx) | Persistent storage |
| Cache | Redis | Geospatial, sessions, caching |
| Message Broker | NATS JetStream | Async event-driven communication |
| WebSocket | `github.com/coder/websocket` | Real-time client communication |
| Auth | JWT + API Keys | Client auth + service-to-service |
| APM | New Relic | Distributed tracing, error tracking |
| Logging | Go `log/slog` + New Relic | Structured logging with APM forwarding |
| Testing | Testify, GoMock, Miniredis | Unit and integration testing |
| Containers | Docker, Docker Compose | Service orchestration |

## Project Structure

```
internal/pkg/           # Shared packages
├── config/             # Environment-based config loading
├── database/           # PostgreSQL + Redis clients
├── health/             # Health check system
├── http/               # Inter-service HTTP client
├── jwt/                # JWT token management
├── logger/             # slog + New Relic integration
├── middleware/          # Unified middleware + auth
│   ├── auth/           # JWT + API key middleware
│   └── tracing/        # Tracing middleware
├── models/             # Domain models
│   ├── core/           # Config, shared types
│   ├── location/       # Location models
│   ├── match/          # Match models
│   ├── notification/   # Notification models
│   ├── ride/           # Ride models
│   ├── user/           # User models
│   └── websocket/      # WebSocket models
├── nats/               # NATS JetStream helpers
├── newrelic/           # New Relic instrumentation
├── tracing/            # Tracer interface + New Relic impl
│   └── newrelic/       # New Relic tracer
└── server/             # Graceful server + shutdown manager

services/               # Microservices
├── gateway/            # Public-facing gateway (WebSocket + REST proxy)
├── location/           # Geospatial tracking
├── match/              # Driver-passenger matching
├── notification/       # Selective event → user notification
├── rides/              # Ride lifecycle + billing
└── users/              # User management + auth
```

Each service follows clean architecture: `handler/ → usecase/ → repository/ → gateway/`

## Testing

### Running Tests

```bash
go test ./...                          # All tests
go test ./... -race                    # With race detection
go test ./... -coverprofile=coverage.txt -covermode=atomic  # With coverage
go tool cover -html=coverage.txt       # HTML coverage report
go generate ./...                      # Regenerate mocks
```

### Testing Approach

- **Unit tests** with GoMock for dependency injection — all business logic in `usecase/` and `handler/` layers
- **Redis integration tests** with Miniredis — geospatial operations in `repository/` layers
- **SQL mock tests** with `sqlmock` — database operations
- **No external dependencies** required to run tests

### Mock Generation

```go
//go:generate mockgen -source=usecase.go -destination=mocks/mock_usecase.go -package=mocks
```

### Test Pattern

```go
func TestGetUser(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockUC := mocks.NewMockUserUC(ctrl)
    handler := NewUserHandler(mockUC)

    mockUC.EXPECT().GetUser(gomock.Any(), "user-123").Return(&models.User{ID: "user-123"}, nil)

    req := httptest.NewRequest(http.MethodGet, "/users/user-123", nil)
    rec := httptest.NewRecorder()
    c := echo.New().NewContext(req, rec)
    c.SetParamNames("id")
    c.SetParamValues("user-123")

    assert.NoError(t, handler.GetUser(c))
    assert.Equal(t, http.StatusOK, rec.Code)
}
```

## CI/CD Pipeline

### GitHub Actions Workflow

**File**: `.github/workflows/continuous-integration.yml`

```
Push/PR → Secret Scan (GitLeaks) → Unit Tests + Coverage → SonarCloud Analysis → PR Comment
```

**File**: `.github/workflows/continuous-delivery.yml`

```
Release → Tests → Build (linux/amd64) → Docker Build + Push → Deploy to EC2 → Health Check
```

### Quality Gates
- GitLeaks must pass (no secrets in code)
- All tests must pass
- SonarCloud quality gate must pass

### Docker Deployment

Each service has a multi-stage Dockerfile:

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o service ./cmd/<service>

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /app/service .
CMD ["./service"]
```

Docker Compose orchestrates all services with health checks:

```bash
docker-compose up              # Start everything
docker-compose up postgres redis nats  # Infrastructure only
docker-compose logs -f gateway  # Follow gateway logs
```

## Local Development

```bash
git clone <repo-url>
go mod download
docker-compose up postgres redis nats   # Start infrastructure
go run ./cmd/gateway                     # Run a service locally
```

### Environment Variables

```bash
DB_HOST=localhost DB_PORT=5432 DB_NAME=nebengjek DB_USER=nebengjek
REDIS_HOST=localhost REDIS_PORT=6379
NATS_URL=nats://localhost:4222
JWT_SECRET_KEY=dev-secret
API_KEY_MATCH_SERVICE=dev-key
LOG_LEVEL=debug
```
