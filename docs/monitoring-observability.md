# Monitoring and Observability

## Overview

NebengJek implements comprehensive monitoring and observability through New Relic APM and Go's native slog structured logging, providing end-to-end visibility into application performance, errors, and business metrics.

## Current Architecture

### Unified Middleware Integration

**Implementation**: [`internal/pkg/middleware/unified.go`](../internal/pkg/middleware/unified.go)

The unified middleware provides integrated observability:
- Automatic New Relic transaction creation
- Request ID generation and propagation
- Structured logging with context correlation
- Error tracking and panic recovery
- Performance metrics collection

```go
func (m *Middleware) Handler() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            start := time.Now()
            
            // Request ID generation
            requestID := generateOrExtractRequestID(c)
            
            // APM transaction setup
            var txn observability.Transaction
            if m.config.Tracer != nil {
                txn = m.config.Tracer.StartTransaction(c.Request().URL.Path)
                defer txn.End()
                txn.SetWebRequest(c.Request())
                c.Set("nr_txn", txn)
            }
            
            // Execute with monitoring
            err := next(c)
            
            // Log request with context
            duration := time.Since(start)
            m.logRequest(c, requestID, duration, err)
            
            return err
        }
    }
}
```

## New Relic APM Integration

### Current Implementation

**Location**: [`internal/pkg/newrelic/newrelic.go`](../internal/pkg/newrelic/newrelic.go)

New Relic integration provides:
- Automatic transaction tracking
- Database operation monitoring
- External service call tracing
- Error reporting and alerting
- Custom business metrics

### Service Configuration

```go
// Service-specific New Relic setup
nrApp, err := newrelic.NewApplication(
    newrelic.ConfigAppName("nebengjek-users-service"),
    newrelic.ConfigLicense(licenseKey),
    newrelic.ConfigEnabled(true),
    newrelic.ConfigDistributedTracerEnabled(true),
)
```

### APM Transaction Tracking

**Handler Level Integration**:
```go
func (h *UserHandler) GetUser(c echo.Context) error {
    // Transaction automatically created by unified middleware
    txn := c.Get("nr_txn").(observability.Transaction)
    
    // Add business context
    txn.AddAttribute("operation", "get_user")
    txn.AddAttribute("user.id", userID)
    
    user, err := h.userUC.GetUser(c.Request().Context(), userID)
    if err != nil {
        txn.NoticeError(err)
        return err
    }
    
    return c.JSON(http.StatusOK, user)
}
```

### Database Instrumentation

**PostgreSQL Monitoring**:
```go
// Automatic database monitoring via pgx integration
func (r *UserRepo) GetByID(ctx context.Context, userID string) (*models.User, error) {
    // Database operations automatically instrumented
    var user models.User
    err := r.db.GetContext(ctx, &user, "SELECT * FROM users WHERE id = $1", userID)
    return &user, err
}
```

## Structured Logging with slog

### Current Implementation

**Location**: [`internal/pkg/logger/slog.go`](../internal/pkg/logger/slog.go)

Go's native slog provides:
- High-performance structured logging
- New Relic log forwarding
- Context-aware log correlation
- Configurable output formats

### Logger Configuration

```go
type SlogConfig struct {
    Level       slog.Level
    ServiceName string
    NewRelic    *newrelic.Application
    Format      string // "json" or "text"
}

func NewSlogLogger(config SlogConfig) *slog.Logger {
    var handler slog.Handler
    
    // Base handler
    opts := &slog.HandlerOptions{
        Level:     config.Level,
        AddSource: true,
    }
    
    switch config.Format {
    case "json":
        handler = slog.NewJSONHandler(os.Stdout, opts)
    default:
        handler = slog.NewTextHandler(os.Stdout, opts)
    }
    
    // New Relic integration
    if config.NewRelic != nil {
        handler = nrslog.WrapHandler(config.NewRelic, handler)
        handler = &NewRelicLogForwarder{
            handler: handler,
            app:     config.NewRelic,
        }
    }
    
    return slog.New(handler)
}
```

### New Relic Log Forwarding

**Automatic Error Log Forwarding**:
```go
type NewRelicLogForwarder struct {
    handler slog.Handler
    app     *newrelic.Application
}

func (h *NewRelicLogForwarder) Handle(ctx context.Context, record slog.Record) error {
    // Forward ERROR level and above to New Relic
    if record.Level >= slog.LevelError && h.app != nil {
        logData := map[string]interface{}{
            "message":   record.Message,
            "level":     record.Level.String(),
            "timestamp": record.Time.UnixMilli(),
        }
        
        // Add all log attributes
        record.Attrs(func(attr slog.Attr) bool {
            logData[attr.Key] = attr.Value.Any()
            return true
        })
        
        // Send to New Relic Application Logs
        h.app.RecordLog(newrelic.LogData{
            Message:    record.Message,
            Severity:   record.Level.String(),
            Timestamp:  record.Time.UnixMilli(),
            Attributes: logData,
        })
    }
    
    return h.handler.Handle(ctx, record)
}
```

### Context-Aware Logging

**ContextLogger Implementation**:
```go
type ContextLogger struct {
    logger *slog.Logger
}

func (cl *ContextLogger) WithContext(ctx context.Context) *slog.Logger {
    attrs := []slog.Attr{}
    
    // Extract context values
    if requestID := ctx.Value("request_id"); requestID != nil {
        attrs = append(attrs, slog.String("request_id", requestID.(string)))
    }
    
    if userID := ctx.Value("user_id"); userID != nil {
        attrs = append(attrs, slog.String("user_id", userID.(string)))
    }
    
    if serviceName := ctx.Value("service_name"); serviceName != nil {
        attrs = append(attrs, slog.String("service_name", serviceName.(string)))
    }
    
    if len(attrs) > 0 {
        args := make([]any, len(attrs))
        for i, attr := range attrs {
            args[i] = attr
        }
        return cl.logger.With(args...)
    }
    
    return cl.logger
}
```

### Structured Logging Examples

**Business Operation Logging**:
```go
// User authentication
logger.InfoContext(ctx, "User authentication successful",
    slog.String("user_id", userID),
    slog.String("method", "JWT"),
    slog.Duration("auth_time", authDuration),
    slog.Bool("first_login", isFirstLogin),
)

// Database operation error
logger.ErrorContext(ctx, "Database operation failed",
    slog.String("operation", "user_create"),
    slog.String("table", "users"),
    slog.Any("error", err),
    slog.String("query_id", queryID),
)

// API request performance
logger.InfoContext(ctx, "API request completed",
    slog.String("endpoint", "/api/v1/users"),
    slog.String("method", "POST"),
    slog.Int("status_code", 201),
    slog.Duration("response_time", duration),
    slog.Int64("response_size", responseSize),
)
```

## HTTP Client Monitoring

### Unified HTTP Client Integration

**Location**: [`internal/pkg/http/client.go`](../internal/pkg/http/client.go)

The unified HTTP client provides:
- Request ID propagation
- Retry logic monitoring
- Error classification
- Performance tracking

```go
func (c *Client) Do(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
    // Request ID propagation
    if requestID := ctx.Value("request_id"); requestID != nil {
        req.Header.Set("X-Request-ID", fmt.Sprintf("%v", requestID))
    }
    
    // Retry logic with monitoring
    for attempt := 0; attempt < 3; attempt++ {
        resp, err = c.httpClient.Do(req)
        if err == nil && resp.StatusCode < 500 {
            return resp, nil
        }
        
        // Log retry attempts
        if attempt < 2 {
            logger.WarnContext(ctx, "HTTP request retry",
                slog.String("url", url),
                slog.Int("attempt", attempt+1),
                slog.Int("status", resp.StatusCode),
            )
        }
    }
    
    return resp, err
}
```

## WebSocket Monitoring

### Echo WebSocket Integration

**Location**: [`services/users/handler/websocket/echo_handler.go`](../services/users/handler/websocket/echo_handler.go)

WebSocket monitoring includes:
- Connection lifecycle tracking
- Message processing metrics
- Error classification and logging
- Business event monitoring

```go
// Connection logging
logger.Info("WebSocket client connected",
    slog.String("user_id", userID),
    slog.String("role", role),
    slog.String("connection_id", connectionID),
)

// Message processing
logger.InfoContext(ctx, "WebSocket message processed",
    slog.String("user_id", userID),
    slog.String("event", msg.Event),
    slog.Duration("processing_time", processingTime),
)

// Error logging with severity
logger.Error("WebSocket operation failed",
    slog.String("user_id", userID),
    slog.String("event", eventType),
    slog.String("error_code", errorCode),
    slog.String("severity", severity),
    slog.Any("error", err),
)
```

## Performance Monitoring

### Key Metrics

#### Application Performance
- **Transaction Throughput**: Requests per minute by service
- **Response Times**: P50, P95, P99 percentiles
- **Error Rates**: 4xx and 5xx error percentages
- **Apdex Score**: Application performance index

#### Database Performance
- **Query Response Times**: Database operation latency
- **Connection Pool Usage**: Active/idle connection ratios
- **Slow Query Identification**: Queries exceeding thresholds

#### Business Metrics
- **User Registration Success Rates**: Authentication performance
- **Driver Matching Performance**: Match success rates and timing
- **WebSocket Connection Health**: Real-time communication metrics
- **Ride Completion Rates**: End-to-end transaction success

### Custom Dashboards

#### Service Health Overview
```json
{
  "dashboard": "NebengJek Service Health",
  "widgets": [
    {
      "title": "Response Times by Service",
      "type": "line_chart",
      "metrics": ["average_response_time"],
      "facet": "appName"
    },
    {
      "title": "Error Rates by Service", 
      "type": "line_chart",
      "metrics": ["error_rate"],
      "facet": "appName"
    },
    {
      "title": "WebSocket Connections",
      "type": "billboard",
      "nrql": "SELECT count(*) FROM Log WHERE message = 'WebSocket client connected' SINCE 1 hour ago"
    }
  ]
}
```

## Health Checks

### Enhanced Health Monitoring

**Location**: [`internal/pkg/health/enhanced.go`](../internal/pkg/health/enhanced.go)

Comprehensive health checking:
- PostgreSQL connection health
- Redis connection health
- NATS JetStream health
- Service dependency health

```go
// PostgreSQL health checker
type PostgresHealthChecker struct {
    client *database.PostgresClient
}

func (p *PostgresHealthChecker) CheckHealth(ctx context.Context) error {
    if p.client == nil {
        return nil // Skip if no PostgreSQL client
    }
    return p.client.GetDB().PingContext(ctx)
}

// Redis health checker
type RedisHealthChecker struct {
    client *database.RedisClient
}

func (r *RedisHealthChecker) CheckHealth(ctx context.Context) error {
    if r.client == nil {
        return nil // Skip if no Redis client
    }
    return r.client.Client.Ping(ctx).Err()
}

// NATS health checker
type NATSHealthChecker struct {
    client *nats.Client
}

func (n *NATSHealthChecker) CheckHealth(ctx context.Context) error {
    if n.client == nil {
        return nil // Skip if no NATS client
    }
    
    // Check basic NATS connection
    conn := n.client.GetConn()
    if conn == nil || !conn.IsConnected() {
        return errors.New("NATS client not connected")
    }
    
    // Check JetStream availability
    js := n.client.GetJetStream()
    if js == nil {
        return errors.New("JetStream not available")
    }
    
    // Verify we can list streams
    _, err := n.client.ListStreams()
    return err
}
```

## Configuration

### Environment Variables

```bash
# New Relic Configuration
NEW_RELIC_ENABLED=true
NEW_RELIC_LICENSE_KEY=your_license_key_here
NEW_RELIC_APP_NAME=nebengjek-users-service

# Logging Configuration
LOG_LEVEL=info
LOG_FORMAT=json
SERVICE_NAME=nebengjek-users

# Health Check Configuration
HEALTH_CHECK_INTERVAL=30s
HEALTH_CHECK_TIMEOUT=5s
```

### Service-Specific Configuration

```go
// Service configuration
type ObservabilityConfig struct {
    NewRelic struct {
        LicenseKey string
        AppName    string
        Enabled    bool
    }
    Logging struct {
        Level  string
        Format string
    }
    Health struct {
        CheckInterval time.Duration
        Timeout       time.Duration
    }
}
```

## Alerting

### Recommended Alerts

```yaml
# High Error Rate Alert
- name: "High Error Rate"
  condition: "error_rate > 5% for 5 minutes"
  severity: "critical"
  
# Slow Response Time Alert  
- name: "Slow Response Time"
  condition: "average_response_time > 2 seconds for 5 minutes"
  severity: "warning"
  
# WebSocket Connection Issues
- name: "WebSocket Connection Failures"
  condition: "websocket_error_rate > 10% for 3 minutes"
  severity: "warning"
  
# Database Health Issues
- name: "Database Connection Issues"
  condition: "database_error_rate > 5% for 2 minutes"
  severity: "critical"
```

## Troubleshooting

### Common Issues

#### Missing Transactions
**Symptoms**: No data in New Relic APM
**Solutions**:
- Verify unified middleware registration
- Check New Relic license key configuration
- Ensure `NEW_RELIC_ENABLED=true`

#### Log Correlation Issues
**Symptoms**: Logs not correlated with traces
**Solutions**:
- Verify request ID middleware in unified middleware
- Check context propagation in handlers
- Ensure slog context logger usage

#### WebSocket Monitoring Gaps
**Symptoms**: Missing WebSocket metrics
**Solutions**:
- Verify WebSocket handler logging
- Check connection lifecycle tracking
- Ensure error severity classification

### Debug Configuration

```bash
# Enable debug logging
NEW_RELIC_DEBUG=true
LOG_LEVEL=debug

# Increase log verbosity
NEW_RELIC_LOG_LEVEL=debug
```

## Performance Impact

- **New Relic Overhead**: < 1ms per transaction
- **slog Logging Overhead**: Minimal memory footprint with native Go implementation
- **Unified Middleware**: Single-pass processing reduces overhead
- **Asynchronous Log Forwarding**: Non-blocking New Relic integration

## See Also

- [Tech Stack Rationale](tech-stack-rationale.md)
- [Unified Middleware Guide](unified-middleware-guide.md)
- [Structured Logging Guide](structured-logging-guide.md)
- [Database Architecture](database-architecture.md)
- [WebSocket Events Specification](websocket-events-specification.md)