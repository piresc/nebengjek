# Enhanced Health Check System

## Overview

The NebengJek Enhanced Health Check System provides comprehensive health monitoring for all service dependencies including PostgreSQL, Redis, and NATS JetStream. The system implements Kubernetes-compatible health endpoints and supports graceful degradation patterns.

## Architecture

```mermaid
graph TB
    subgraph "Health Check System"
        HS[HealthService]
        HC[HealthChecker Interface]
        
        subgraph "Health Checkers"
            PHC[PostgresHealthChecker]
            RHC[RedisHealthChecker]
            NHC[NATSHealthChecker]
        end
        
        subgraph "Health Endpoints"
            HE1[/health - Basic]
            HE2[/health/detailed - Comprehensive]
            HE3[/health/ready - Readiness Probe]
            HE4[/health/live - Liveness Probe]
        end
    end
    
    HS --> HC
    HC --> PHC
    HC --> RHC
    HC --> NHC
    
    HS --> HE1
    HS --> HE2
    HS --> HE3
    HS --> HE4
```

## Core Components

### HealthChecker Interface

The [`HealthChecker`](../internal/pkg/health/enhanced.go:16) interface provides a standardized way to check the health of different dependencies:

```go
type HealthChecker interface {
    CheckHealth(ctx context.Context) error
}
```

### Health Service

The [`HealthService`](../internal/pkg/health/enhanced.go:94) manages multiple health checkers and aggregates their results:

```go
type HealthService struct {
    checkers map[string]HealthChecker
    logger   *slog.Logger
}
```

## Health Checker Implementations

### PostgreSQL Health Checker

The [`PostgresHealthChecker`](../internal/pkg/health/enhanced.go:21) validates database connectivity:

```go
func (p *PostgresHealthChecker) CheckHealth(ctx context.Context) error {
    if p.client == nil {
        return nil // Skip if no PostgreSQL client
    }
    return p.client.GetDB().PingContext(ctx)
}
```

**Features**:
- Connection validation using `PingContext`
- Graceful handling of nil clients
- Context-aware timeout support

### Redis Health Checker

The [`RedisHealthChecker`](../internal/pkg/health/enhanced.go:39) validates Redis connectivity:

```go
func (r *RedisHealthChecker) CheckHealth(ctx context.Context) error {
    if r.client == nil {
        return nil // Skip if no Redis client
    }
    return r.client.Client.Ping(ctx).Err()
}
```

**Features**:
- Redis PING command validation
- Connection pool health verification
- Timeout handling through context

### NATS Health Checker

The [`NATSHealthChecker`](../internal/pkg/health/enhanced.go:57) validates both NATS connection and JetStream availability:

```go
func (n *NATSHealthChecker) CheckHealth(ctx context.Context) error {
    if n.client == nil {
        return nil // Skip if no NATS client
    }

    // Check basic NATS connection
    conn := n.client.GetConn()
    if conn == nil || !conn.IsConnected() {
        return echo.NewHTTPError(http.StatusServiceUnavailable, "NATS not connected")
    }

    // Check JetStream availability
    js := n.client.GetJetStream()
    if js == nil {
        return echo.NewHTTPError(http.StatusServiceUnavailable, "JetStream not available")
    }

    // Verify we can list streams (basic JetStream health check)
    _, err := n.client.ListStreams()
    if err != nil {
        return echo.NewHTTPError(http.StatusServiceUnavailable, "JetStream streams not accessible: "+err.Error())
    }

    return nil
}
```

**Features**:
- NATS connection status validation
- JetStream availability verification
- Stream accessibility testing
- Detailed error reporting

## Health Endpoints

### Basic Health Check (`/health`)

Simple health check for load balancers and basic monitoring:

```json
{
  "status": "ok",
  "service": "users-service",
  "timestamp": "2025-01-08T10:00:00Z"
}
```

**Use Cases**:
- Load balancer health checks
- Basic service availability monitoring
- Quick health verification

### Detailed Health Check (`/health/detailed`)

Comprehensive health check with dependency status:

```json
{
  "status": "healthy",
  "timestamp": "2025-01-08T10:00:00Z",
  "service": "users-service",
  "version": "1.0.0",
  "dependencies": {
    "postgres": {
      "status": "healthy"
    },
    "redis": {
      "status": "healthy"
    },
    "nats": {
      "status": "unhealthy",
      "error": "NATS not connected"
    }
  }
}
```

**Features**:
- Individual dependency status
- Error details for failed checks
- Service version information
- Overall health aggregation

### Readiness Probe (`/health/ready`)

Kubernetes readiness probe endpoint:

```json
{
  "status": "ready",
  "service": "users-service"
}
```

**Behavior**:
- Returns `200 OK` when all dependencies are healthy
- Returns `503 Service Unavailable` when any dependency is unhealthy
- Used by Kubernetes to determine if pod should receive traffic

### Liveness Probe (`/health/live`)

Kubernetes liveness probe endpoint:

```json
{
  "status": "alive",
  "service": "users-service"
}
```

**Behavior**:
- Always returns `200 OK` unless the service is completely unresponsive
- Used by Kubernetes to determine if pod should be restarted
- Does not check dependencies (only service responsiveness)

## Implementation Guide

### Basic Setup

```go
package main

import (
    "log/slog"
    "github.com/labstack/echo/v4"
    "github.com/piresc/nebengjek/internal/pkg/health"
    "github.com/piresc/nebengjek/internal/pkg/database"
)

func setupHealthChecks(e *echo.Echo, logger *slog.Logger, pgClient *database.PostgresClient, redisClient *database.RedisClient, natsClient *nats.Client) {
    // Create health service
    healthService := health.NewHealthService(logger)
    
    // Register health checkers
    healthService.AddChecker("postgres", health.NewPostgresHealthChecker(pgClient))
    healthService.AddChecker("redis", health.NewRedisHealthChecker(redisClient))
    healthService.AddChecker("nats", health.NewNATSHealthChecker(natsClient))
    
    // Register health endpoints
    health.RegisterEnhancedHealthEndpoints(e, "users-service", "1.0.0", healthService)
}
```

### Service Integration

Each service should integrate health checks in their main function:

```go
// cmd/users/main.go
func main() {
    // ... service initialization ...
    
    // Setup health checks
    setupHealthChecks(e, logger, pgClient, redisClient, natsClient)
    
    // Start server
    server := server.NewGracefulServer(e, logger, cfg.Server.Port)
    if err := server.Start(); err != nil {
        logger.Error("Server failed", logger.Err(err))
    }
}
```

## Kubernetes Integration

### Deployment Configuration

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: users-service
spec:
  template:
    spec:
      containers:
      - name: users-service
        image: nebengjek/users-service:latest
        ports:
        - containerPort: 9990
        livenessProbe:
          httpGet:
            path: /health/live
            port: 9990
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 9990
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
```

### Service Configuration

```yaml
apiVersion: v1
kind: Service
metadata:
  name: users-service
spec:
  selector:
    app: users-service
  ports:
  - port: 9990
    targetPort: 9990
  type: ClusterIP
```

## Docker Compose Integration

```yaml
version: '3.8'
services:
  users-service:
    build: ./cmd/users
    ports:
      - "9990:9990"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9990/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      nats:
        condition: service_healthy
```

## Monitoring Integration

### Prometheus Metrics

Health check results can be exposed as Prometheus metrics:

```go
// Example metric collection
var (
    healthCheckDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "health_check_duration_seconds",
            Help: "Duration of health checks",
        },
        []string{"service", "dependency", "status"},
    )
)

func (h *HealthService) CheckAllHealthWithMetrics(ctx context.Context) HealthResponse {
    start := time.Now()
    response := h.CheckAllHealth(ctx)
    
    for name, dep := range response.Dependencies {
        healthCheckDuration.WithLabelValues(
            response.Service,
            name,
            dep.Status,
        ).Observe(time.Since(start).Seconds())
    }
    
    return response
}
```

### New Relic Integration

Health check failures are automatically logged and can be monitored in New Relic:

```go
func (h *HealthService) CheckAllHealth(ctx context.Context) HealthResponse {
    // ... health check logic ...
    
    for name, checker := range h.checkers {
        if err := checker.CheckHealth(ctx); err != nil {
            if h.logger != nil {
                h.logger.Error("Health check failed",
                    logger.String("dependency", name),
                    logger.Err(err))
            }
            // Error is automatically forwarded to New Relic via slog integration
        }
    }
    
    return response
}
```

## Best Practices

### Health Check Design

1. **Fail Fast**: Health checks should complete quickly (< 5 seconds)
2. **Graceful Degradation**: Handle nil clients gracefully
3. **Meaningful Errors**: Provide detailed error messages for debugging
4. **Context Awareness**: Respect context timeouts and cancellation

### Kubernetes Considerations

1. **Separate Concerns**: Use different endpoints for liveness vs readiness
2. **Appropriate Timeouts**: Configure realistic timeout values
3. **Failure Thresholds**: Set appropriate failure thresholds to avoid flapping
4. **Startup Time**: Account for service startup time in initial delays

### Monitoring Strategy

1. **Alert on Failures**: Set up alerts for health check failures
2. **Track Trends**: Monitor health check duration trends
3. **Dependency Mapping**: Use health checks to understand service dependencies
4. **Capacity Planning**: Use health metrics for capacity planning

## Troubleshooting

### Common Issues

#### PostgreSQL Connection Failures
```
Error: "connection refused"
Solution: Verify database connectivity and credentials
```

#### Redis Connection Timeouts
```
Error: "i/o timeout"
Solution: Check Redis server status and network connectivity
```

#### NATS JetStream Unavailable
```
Error: "JetStream not available"
Solution: Verify NATS server has JetStream enabled
```

### Debugging Health Checks

Enable debug logging to troubleshoot health check issues:

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

healthService := health.NewHealthService(logger)
```

### Health Check Testing

Test health checks in isolation:

```go
func TestPostgresHealthChecker(t *testing.T) {
    // Setup test database
    pgClient := setupTestDatabase()
    checker := health.NewPostgresHealthChecker(pgClient)
    
    // Test healthy case
    err := checker.CheckHealth(context.Background())
    assert.NoError(t, err)
    
    // Test unhealthy case
    pgClient.Close()
    err = checker.CheckHealth(context.Background())
    assert.Error(t, err)
}
```

## Related Documentation

- [System Architecture](system-architecture.md) - Overall system design
- [Monitoring & Observability](monitoring-observability.md) - Monitoring setup
- [CI/CD & Deployment](cicd-deployment.md) - Deployment configuration
- [Database Architecture](database-architecture.md) - Database setup