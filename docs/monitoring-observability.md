# Monitoring and Observability

## Overview

NebengJek implements comprehensive monitoring and observability through New Relic APM and Zap structured logging, providing end-to-end visibility into application performance, errors, and business metrics.

## New Relic APM Integration

### Architecture and Components

#### 1. New Relic Middleware
**File**: [`internal/pkg/middleware/newrelic.go`](../internal/pkg/middleware/newrelic.go)

The New Relic middleware provides:
- Transaction creation for each HTTP request
- Web request/response context setting
- Transaction context propagation
- Custom attribute helpers
- Error reporting utilities

```go
// NewNewRelicMiddleware creates middleware instance
func NewNewRelicMiddleware() echo.MiddlewareFunc {
    return nrecho.Middleware(nrApp)
}

// Helper functions for custom attributes
func AddCustomAttribute(txn *newrelic.Transaction, key string, value interface{}) {
    txn.AddAttribute(key, value)
}

func NoticeError(txn *newrelic.Transaction, err error) {
    txn.NoticeError(err)
}
```

#### 2. Database Instrumentation
**File**: [`internal/pkg/newrelic/helpers.go`](../internal/pkg/newrelic/helpers.go)

Database operations are instrumented with datastore segments:

```go
// PostgreSQL instrumentation
func StartDatastoreSegment(txn *newrelic.Transaction, operation, table string) *newrelic.DatastoreSegment {
    return &newrelic.DatastoreSegment{
        StartTime:  txn.StartSegmentNow(),
        Product:    newrelic.DatastorePostgres,
        Collection: table,
        Operation:  operation,
    }
}

// Redis instrumentation  
func StartRedisSegment(txn *newrelic.Transaction, operation string) *newrelic.DatastoreSegment {
    return &newrelic.DatastoreSegment{
        StartTime:  txn.StartSegmentNow(),
        Product:    newrelic.DatastoreRedis,
        Operation:  operation,
    }
}
```

#### 3. HTTP Client Instrumentation
**File**: [`internal/pkg/http/newrelic_client.go`](../internal/pkg/http/newrelic_client.go)

External service calls are tracked with distributed tracing:

```go
func (c *APIKeyClient) doRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
    // Start external segment for distributed tracing
    txn := newrelic.FromContext(ctx)
    segment := newrelic.StartExternalSegment(txn, req)
    defer segment.End()
    
    // Add distributed tracing headers
    req.Header = http.Header{}
    txn.InsertDistributedTraceHeaders(req.Header)
    
    return c.client.Do(req)
}
```

### Service Integration

#### Handler Layer Instrumentation
**Example**: [`services/users/handler/http/user.go`](../services/users/handler/http/user.go)

```go
func (h *UserHandler) GetUser(c echo.Context) error {
    // Extract New Relic transaction
    txn := newrelic.FromContext(c.Request().Context())
    
    // Add business context
    txn.AddAttribute("operation", "get_user")
    txn.AddAttribute("user.id", userID)
    
    // Call use case with context
    user, err := h.userUC.GetUser(c.Request().Context(), userID)
    if err != nil {
        txn.NoticeError(err)
        return err
    }
    
    return c.JSON(http.StatusOK, user)
}
```

#### Use Case Layer Instrumentation
**Example**: [`services/users/usecase/user.go`](../services/users/usecase/user.go)

```go
func (uc *UserUC) GetUser(ctx context.Context, userID string) (*models.User, error) {
    // Start business logic segment
    txn := newrelic.FromContext(ctx)
    segment := txn.StartSegment("GetUser.BusinessLogic")
    defer segment.End()
    
    // Add operation-specific attributes
    txn.AddAttribute("user.lookup_method", "by_id")
    
    // Call repository with instrumented context
    user, err := uc.userRepo.GetByID(ctx, userID)
    if err != nil {
        txn.NoticeError(err)
        return nil, err
    }
    
    return user, nil
}
```

#### Repository Layer Instrumentation
**Example**: [`services/users/repository/user.go`](../services/users/repository/user.go)

```go
func (r *UserRepo) GetByID(ctx context.Context, userID string) (*models.User, error) {
    // Start database segment
    txn := newrelic.FromContext(ctx)
    segment := &newrelic.DatastoreSegment{
        StartTime:  txn.StartSegmentNow(),
        Product:    newrelic.DatastorePostgres,
        Collection: "users",
        Operation:  "SELECT",
    }
    defer segment.End()
    
    // Add query-specific attributes
    txn.AddAttribute("db.table", "users")
    txn.AddAttribute("db.operation", "select")
    
    // Execute query
    var user models.User
    err := r.db.GetContext(ctx, &user, "SELECT * FROM users WHERE id = $1", userID)
    
    return &user, err
}
```

### Custom Attributes Reference

#### Common Attributes
| Attribute | Description | Example |
|-----------|-------------|---------|
| `operation` | Business operation type | `create_user`, `find_nearby_drivers` |
| `user.id` | User identifier | `uuid-string` |
| `user.role` | User role | `passenger`, `driver` |
| `driver.id` | Driver identifier | `uuid-string` |
| `ride.id` | Ride identifier | `uuid-string` |
| `match.id` | Match identifier | `uuid-string` |

#### Database Attributes
| Attribute | Description | Example |
|-----------|-------------|---------|
| `db.operation` | Database operation | `select`, `insert`, `update` |
| `db.table` | Database table | `users`, `rides` |
| `redis.operation` | Redis operation | `geoadd`, `hmset` |

#### HTTP Attributes
| Attribute | Description | Example |
|-----------|-------------|---------|
| `http.method` | HTTP method | `GET`, `POST` |
| `http.url` | Request URL | `/api/users/123` |
| `http.status_code` | Response status | `200`, `404` |
| `http.service` | Target service | `location-service` |

#### Business Logic Attributes
| Attribute | Description | Example |
|-----------|-------------|---------|
| `search.radius_km` | Search radius | `5.0` |
| `drivers.found_count` | Number of drivers found | `3` |
| `validation.error` | Validation error type | `msisdn_invalid` |

### Configuration

#### Environment Variables
```bash
# New Relic Configuration
NEW_RELIC_ENABLED=true
NEW_RELIC_LICENSE_KEY=your_license_key_here
NEW_RELIC_APP_NAME=nebengjek-users-service
NEW_RELIC_LOGS_ENABLED=true
NEW_RELIC_FORWARD_LOGS=true
```

#### Service-Specific App Names
- `nebengjek-users-service`
- `nebengjek-match-service`
- `nebengjek-location-service`
- `nebengjek-rides-service`

#### Configuration Structure
```go
type NewRelicConfig struct {
    LicenseKey   string `json:"license_key"`
    AppName      string `json:"app_name"`
    Enabled      bool   `json:"enabled"`
    LogsEnabled  bool   `json:"logs_enabled"`
    LogsEndpoint string `json:"logs_endpoint"`
    LogsAPIKey   string `json:"logs_api_key"`
    ForwardLogs  bool   `json:"forward_logs"`
}
```

## Zap Logging Framework

### Architecture and Features

#### High-Performance Structured Logging
- **JSON Format**: Machine-readable log format
- **Zero Allocation**: Optimized for production performance
- **Multiple Outputs**: Console, file, and New Relic forwarding
- **Log Rotation**: Automatic file rotation with configurable size/retention

#### Logger Configuration
**File**: [`internal/pkg/logger/zap.go`](../internal/pkg/logger/zap.go)

```go
type LoggerConfig struct {
    Level      string `json:"level"`       // debug, info, warn, error
    FilePath   string `json:"file_path"`   // Log file path
    MaxSize    int64  `json:"max_size"`    // Max size in MB
    MaxAge     int    `json:"max_age"`     // Max age in days
    MaxBackups int    `json:"max_backups"` // Max backup files
    Compress   bool   `json:"compress"`    // Compress rotated files
    Type       string `json:"type"`        // file, console, hybrid, newrelic
}
```

#### Environment Variables
```bash
# Zap Logger Configuration
LOG_LEVEL=info
LOG_FILE_PATH=./logs/app.log
LOG_MAX_SIZE=100
LOG_MAX_AGE=30
LOG_MAX_BACKUPS=10
LOG_COMPRESS=true
LOG_TYPE=hybrid
```

### Logging Implementation

#### Structured Logging with Context
```go
// Initialize logger with New Relic integration
zapLogger, err := logger.InitZapLoggerFromConfig(configs, nrApp)
if err != nil {
    log.Fatalf("Failed to create Zap logger: %v", err)
}
defer zapLogger.Close()

// Structured logging with context
zapLogger.Info("User authentication successful",
    zap.String("user_id", userID),
    zap.String("method", "OTP"),
    zap.Duration("response_time", time.Since(start)),
)

// Error logging with stack trace
zapLogger.Error("Database connection failed",
    zap.Error(err),
    zap.String("database", "postgresql"),
    zap.String("operation", "user_lookup"),
)
```

#### Echo Middleware Integration
**File**: [`internal/pkg/middleware/logger.go`](../internal/pkg/middleware/logger.go)

```go
func ZapEchoMiddleware(logger *ZapLogger) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            start := time.Now()
            
            err := next(c)
            
            // Log HTTP request with comprehensive context
            logger.Info("HTTP Request",
                zap.String("method", c.Request().Method),
                zap.String("path", c.Request().URL.Path),
                zap.Int("status", c.Response().Status),
                zap.Duration("latency", time.Since(start)),
                zap.String("client_ip", c.RealIP()),
                zap.String("user_id", getUserID(c)),
                zap.String("request_id", getRequestID(c)),
            )
            
            return err
        }
    }
}
```

#### Log Output Format
```json
{
  "timestamp": "2024-01-15T10:30:45.123Z",
  "level": "info",
  "message": "HTTP Request",
  "method": "POST",
  "path": "/api/v1/auth/login",
  "status": 200,
  "latency_ms": 45,
  "client_ip": "192.168.1.100",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "request_id": "req_123456789",
  "service": "nebengjek-users-app",
  "caller": "handler/http/auth.go:45"
}
```

### Log Correlation and Tracing

#### Request ID Middleware
```go
func RequestIDMiddleware() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            requestID := c.Request().Header.Get("X-Request-ID")
            if requestID == "" {
                requestID = generateRequestID()
            }
            
            c.Set("request_id", requestID)
            c.Response().Header().Set("X-Request-ID", requestID)
            
            return next(c)
        }
    }
}
```

#### Context Propagation
```go
func LogWithContext(ctx context.Context, level string, message string, fields ...zap.Field) {
    // Extract correlation IDs from context
    if requestID := getRequestIDFromContext(ctx); requestID != "" {
        fields = append(fields, zap.String("request_id", requestID))
    }
    
    if userID := getUserIDFromContext(ctx); userID != "" {
        fields = append(fields, zap.String("user_id", userID))
    }
    
    // Add New Relic trace context
    if txn := newrelic.FromContext(ctx); txn != nil {
        fields = append(fields, zap.String("trace_id", txn.GetTraceMetadata().TraceID))
    }
    
    logger.Log(level, message, fields...)
}
```

## Performance Monitoring

### Key Metrics to Monitor

#### Application Performance
- **Transaction Throughput**: Requests per minute by service
- **Response Times**: P50, P95, P99 percentiles
- **Error Rates**: 4xx and 5xx error percentages
- **Apdex Score**: Application performance index

#### Database Performance
- **Query Response Times**: Database operation latency
- **Connection Pool Usage**: Active/idle connection ratios
- **Slow Query Identification**: Queries exceeding thresholds
- **Database Throughput**: Operations per second

#### External Service Dependencies
- **Service-to-Service Latency**: Inter-service call performance
- **External Service Error Rates**: Dependency failure rates
- **Dependency Failure Impact**: Cascade failure detection

#### Business Metrics
- **User Registration Success Rates**: Authentication performance
- **Driver Matching Performance**: Match success rates and timing
- **Location Update Frequency**: Real-time data freshness
- **Ride Completion Rates**: End-to-end transaction success

### Alerting and Notifications

#### Recommended Alerts
```yaml
# High Error Rate Alert
- name: "High Error Rate"
  condition: "error_rate > 5% for 5 minutes"
  severity: "critical"
  
# Slow Response Time Alert  
- name: "Slow Response Time"
  condition: "average_response_time > 2 seconds for 5 minutes"
  severity: "warning"
  
# Database Slow Queries
- name: "Database Slow Queries"
  condition: "query_time > 1 second"
  severity: "warning"
  
# External Service Failures
- name: "External Service Failures"
  condition: "external_error_rate > 10% for 3 minutes"
  severity: "critical"
```

#### Alert Channels
- **Slack Integration**: Real-time team notifications
- **Email Alerts**: Critical issue notifications
- **PagerDuty**: On-call escalation for critical alerts
- **Webhook Integration**: Custom alert handling

### Custom Dashboards

#### Service Health Overview
```json
{
  "dashboard": "Service Health Overview",
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
      "title": "Throughput by Service",
      "type": "line_chart", 
      "metrics": ["throughput"],
      "facet": "appName"
    }
  ]
}
```

#### Business Operations Dashboard
```json
{
  "dashboard": "Business Operations",
  "widgets": [
    {
      "title": "User Registrations per Hour",
      "type": "line_chart",
      "nrql": "SELECT count(*) FROM Transaction WHERE name = 'POST /auth/register' TIMESERIES 1 hour"
    },
    {
      "title": "Driver Availability Metrics",
      "type": "billboard",
      "nrql": "SELECT average(drivers.found_count) FROM Transaction WHERE operation = 'find_nearby_drivers'"
    },
    {
      "title": "Ride Matching Success Rates",
      "type": "pie_chart", 
      "nrql": "SELECT count(*) FROM Transaction WHERE operation = 'match_request' FACET success"
    }
  ]
}
```

## Health Checks and Monitoring

### Enhanced Health Service
**File**: [`internal/pkg/health/health.go`](../internal/pkg/health/health.go)

```go
type HealthService struct {
    checkers map[string]HealthChecker
    logger   *logger.ZapLogger
}

func (hs *HealthService) AddChecker(name string, checker HealthChecker) {
    hs.checkers[name] = checker
}

func (hs *HealthService) CheckHealth() map[string]HealthStatus {
    results := make(map[string]HealthStatus)
    
    for name, checker := range hs.checkers {
        start := time.Now()
        err := checker.Check()
        duration := time.Since(start)
        
        status := HealthStatus{
            Status:   "healthy",
            Duration: duration,
        }
        
        if err != nil {
            status.Status = "unhealthy"
            status.Error = err.Error()
        }
        
        results[name] = status
        
        // Log health check results
        hs.logger.Info("Health check completed",
            zap.String("component", name),
            zap.String("status", status.Status),
            zap.Duration("duration", duration),
        )
    }
    
    return results
}
```

### Component Health Checkers

#### PostgreSQL Health Checker
```go
func NewPostgresHealthChecker(client *database.PostgresClient) HealthChecker {
    return &PostgresHealthChecker{client: client}
}

func (c *PostgresHealthChecker) Check() error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    return c.client.GetDB().PingContext(ctx)
}
```

#### Redis Health Checker
```go
func NewRedisHealthChecker(client *database.RedisClient) HealthChecker {
    return &RedisHealthChecker{client: client}
}

func (c *RedisHealthChecker) Check() error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    return c.client.GetClient().Ping(ctx).Err()
}
```

#### NATS Health Checker
```go
func NewNATSHealthChecker(client *nats.Client) HealthChecker {
    return &NATSHealthChecker{client: client}
}

func (c *NATSHealthChecker) Check() error {
    if !c.client.IsConnected() {
        return errors.New("NATS client not connected")
    }
    
    // Check JetStream availability
    js, err := c.client.JetStream()
    if err != nil {
        return fmt.Errorf("JetStream not available: %w", err)
    }
    
    // Verify stream health
    _, err = js.StreamInfo("LOCATION")
    return err
}
```

## Troubleshooting and Debugging

### Common Monitoring Issues

#### Missing Transactions
**Symptoms**: No data in New Relic APM
**Solutions**:
- Verify middleware registration order
- Check New Relic license key configuration
- Ensure `NEW_RELIC_ENABLED=true`

#### Missing Custom Attributes
**Symptoms**: Attributes not appearing in transaction traces
**Solutions**:
- Verify transaction context propagation
- Check attribute naming conventions
- Ensure attributes added before transaction ends

#### Log Correlation Issues
**Symptoms**: Logs not correlated with traces
**Solutions**:
- Verify request ID middleware
- Check context propagation
- Ensure trace ID extraction

### Debug Configuration
```bash
# Enable debug logging
NEW_RELIC_DEBUG=true
LOG_LEVEL=debug

# Increase log verbosity
NEW_RELIC_LOG_LEVEL=debug
```

### Performance Impact
- **New Relic Overhead**: < 1ms per transaction
- **Zap Logging Overhead**: Minimal memory footprint
- **Asynchronous Transmission**: Non-blocking data collection
- **Configurable Sampling**: Adjustable data collection rates

## See Also
- [Database Architecture](database-architecture.md)
- [NATS Messaging System](nats-messaging.md)
- [Security Implementation](security-implementation.md)
- [Testing Strategies](testing-strategies.md)