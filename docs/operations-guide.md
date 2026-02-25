# Operations Guide

## Overview

This guide covers all operational concerns for NebengJek: middleware, tracing, logging, health checks, graceful shutdown, security, and inter-service HTTP communication.

## Unified Middleware

**Location**: [`internal/pkg/middleware/unified.go`](../internal/pkg/middleware/unified.go)

The unified middleware handles all cross-cutting concerns in a single pass:

1. **Request ID** — generates UUID or reads `X-Request-ID` from incoming header
2. **APM transaction** — starts a New Relic transaction (if tracer is configured)
3. **Panic recovery** — catches panics with stack trace and WebSocket diagnostics
4. **Request logging** — structured log with method, path, status, duration, IP
5. **Response body capture** — captures first 1KB for error analysis

```go
unifiedMW := middleware.NewMiddleware(middleware.Config{
    Logger:      slogLogger,
    Tracer:      tracer,
    APIKeys:     apiKeyMap,
    ServiceName: "gateway-service",
})
e.Use(unifiedMW.Handler())
```

### Auth Middleware

**Location**: [`internal/pkg/middleware/auth/auth.go`](../internal/pkg/middleware/auth/auth.go)

Two authentication mechanisms:

- **JWT** — `AuthMiddleware.JWTHandler()` for client-facing endpoints. Extracts `user_id` and `role` from claims.
- **API Key** — `AuthMiddleware.APIKeyHandler(service)` for service-to-service calls. Validates `X-API-Key` header against per-service keys in config.

```go
authMW := auth.NewAuthMiddleware(config)

// JWT-protected routes
protected := e.Group("/api/v1", authMW.JWTHandler())

// API-key-protected internal routes
internal := e.Group("/internal", authMW.APIKeyHandler("match-service"))
```

## Tracing

**Location**: [`internal/pkg/tracing/`](../internal/pkg/tracing/)

The `tracing.Tracer` interface provides unified APM instrumentation:

```go
type Tracer interface {
    StartHTTPRequest(r *http.Request) (context.Context, HTTPTransaction)
    StartExternalCall(ctx context.Context, host, method string) (context.Context, func())
    StartMessage(ctx context.Context, topic, operation string) (context.Context, func())
    StartDatabase(ctx context.Context, operation, table string) (context.Context, func())
    StartSegment(ctx context.Context, name string) (context.Context, func())
    InjectContext(ctx context.Context, carrier interface{})
    ExtractContext(ctx context.Context, carrier interface{}) context.Context
    IsEnabled() bool
    ShouldTrace(path string) bool
}
```

**Implementations** (in `tracing/newrelic/`):
- `NewRelicTracer` — production, wraps New Relic SDK
- `NoOpTracer` — zero-overhead for testing

The middleware automatically creates HTTP transactions. Handlers access the transaction via `c.Get("nr_txn")`.

### Configuration

```go
tracing.Config{
    Enabled:      true,
    ServiceName:  "gateway-service",
    SampleRate:   1.0,
    ExcludePaths: []string{"/health", "/metrics"},
    NoOpForTesting: false,
}
```

## Structured Logging

**Location**: [`internal/pkg/logger/slog.go`](../internal/pkg/logger/slog.go)

Built on Go's native `log/slog`:

```go
logger.NewSlogLogger(logger.SlogConfig{
    Level:       slog.LevelInfo,
    ServiceName: "gateway-service",
    NewRelic:    nrApp,    // nil for no APM
    Format:      "json",   // or "text" for dev
})
```

Key features:
- **New Relic log forwarding** — ERROR+ logs automatically sent to New Relic via `NewRelicLogForwarder`
- **Context-aware** — `ContextLogger.WithContext(ctx)` extracts request_id, user_id, trace_id
- **Middleware integration** — all requests logged with method, path, status, duration, IP

## Health Checks

**Location**: [`internal/pkg/health/enhanced.go`](../internal/pkg/health/enhanced.go)

`HealthChecker` interface with implementations for PostgreSQL, Redis, and NATS:

```go
healthService := health.NewHealthService(slogLogger)
healthService.AddChecker("postgres", health.NewPostgresHealthChecker(pgClient))
healthService.AddChecker("redis", health.NewRedisHealthChecker(redisClient))
healthService.AddChecker("nats", health.NewNATSHealthChecker(natsClient))

health.RegisterEnhancedHealthEndpoints(e, appName, version, healthService)
```

Endpoints:
| Path | Purpose | K8s Probe |
|------|---------|-----------|
| `/health` | Basic status for load balancers | — |
| `/health/detailed` | Per-dependency status with errors | — |
| `/health/ready` | 200 if all deps healthy, 503 otherwise | readinessProbe |
| `/health/live` | Always 200 (service is responsive) | livenessProbe |

## Graceful Shutdown

**Location**: [`internal/pkg/server/server.go`](../internal/pkg/server/server.go)

`GracefulServer` handles SIGINT/SIGTERM with a 30-second timeout:

```go
gracefulServer := server.NewGracefulServer(e, logger, port)
gracefulServer.Start() // blocks until signal received
```

`ShutdownManager` coordinates cleanup of multiple components:

```go
sm := server.NewShutdownManager(logger)
sm.Register(func(ctx context.Context) error { return pgClient.Close() })
sm.Register(func(ctx context.Context) error { return redisClient.Close() })
sm.Register(func(ctx context.Context) error { natsClient.Close(); return nil })
// After graceful server returns:
sm.Shutdown(ctx)
```

### Container Integration

```yaml
# Kubernetes
terminationGracePeriodSeconds: 35
lifecycle:
  preStop:
    exec:
      command: ["/bin/sh", "-c", "sleep 5"]

# Docker Compose
stop_grace_period: 35s
```

## HTTP Client

**Location**: [`internal/pkg/http/client.go`](../internal/pkg/http/client.go)

Inter-service HTTP client with:
- Automatic `X-API-Key` header injection
- `X-Request-ID` propagation from context
- Retry with exponential backoff (3 attempts, 100ms/200ms/400ms) for 5xx errors only
- `GetJSON` / `PostJSON` helpers that handle both structured (`{success, data, error}`) and direct JSON responses

```go
client := httpclient.NewClient(httpclient.Config{
    APIKey:  config.APIKey.MatchService,
    BaseURL: "http://match-service:9993",
    Timeout: 30 * time.Second,
})

var result MatchProposal
err := client.PostJSON(ctx, "/internal/matches/confirm", req, &result)
```

## Security

### Authentication
- **JWT** for client-facing endpoints (MSISDN-based OTP → JWT token)
- **API keys** for service-to-service communication (per-service keys in config)

### Secret Scanning
- **GitLeaks** runs in CI via `.gitleaks.toml` — blocks PRs with detected secrets
- Test files and `.example` files are allowlisted

### SonarCloud
- Quality gate in CI via `sonar-project.properties`
- Coverage exclusions: `**/cmd/**/main.go`, `**/mocks/**`

### Best Practices
- Parameterized SQL queries (no string interpolation)
- Sensitive headers (`Authorization`, `X-API-Key`, `Cookie`) redacted in panic logs
- Environment variables for all secrets — never hardcoded
