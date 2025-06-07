# Middleware Simplification Implementation Plan

## Overview
This document provides detailed instructions for simplifying the over-complicated middleware stack in the nebengjek codebase. The current middleware has grown into a complex web that creates development friction, maintenance burden, and testing complexity.

## Current Problems Analysis

### 1. Over-Engineered HTTP Clients
- **EnhancedClient** (111 lines): Wraps circuit breaker + retry + New Relic
- **APIKeyClient** (230 lines): Duplicates HTTP logic with API key auth
- **Total**: 341 lines for basic HTTP functionality

### 2. Excessive APM Integration
- New Relic embedded in every layer
- **PanicRecoveryMiddleware** (389 lines) - massively over-engineered
- APM concerns scattered throughout business logic

### 3. Context Confusion
- **RequestContext** vs standard Go context
- **RequestContextMiddleware** adds unnecessary layer
- Duplicate context management

### 4. Circuit Breaker Overkill
- **CircuitBreaker** (314 lines) for simple failure handling
- Over-engineered for microservice-to-microservice calls

### 5. Retry Mechanism Complexity
- **Retrier** (266 lines) with metrics, jitter, complex backoff
- Too sophisticated for internal service calls

## Implementation Strategy

### Phase 1: Consolidate HTTP Clients

#### Step 1.1: Create Unified HTTP Client
**File**: `internal/pkg/http/simple_client.go`

```go
package http

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

type Client struct {
    httpClient *http.Client
    apiKey     string
    baseURL    string
    timeout    time.Duration
}

type Config struct {
    APIKey  string
    BaseURL string
    Timeout time.Duration
}

func NewClient(config Config) *Client {
    if config.Timeout == 0 {
        config.Timeout = 30 * time.Second
    }
    
    return &Client{
        httpClient: &http.Client{Timeout: config.Timeout},
        apiKey:     config.APIKey,
        baseURL:    config.BaseURL,
        timeout:    config.Timeout,
    }
}

func (c *Client) Do(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
    url := c.baseURL + endpoint
    
    var reqBody io.Reader
    if body != nil {
        jsonBody, err := json.Marshal(body)
        if err != nil {
            return nil, fmt.Errorf("marshal body: %w", err)
        }
        reqBody = bytes.NewBuffer(jsonBody)
    }
    
    req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }
    
    // Set headers
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")
    
    if c.apiKey != "" {
        req.Header.Set("X-API-Key", c.apiKey)
    }
    
    // Simple retry logic (3 attempts)
    var resp *http.Response
    for attempt := 0; attempt < 3; attempt++ {
        resp, err = c.httpClient.Do(req)
        if err == nil && resp.StatusCode < 500 {
            return resp, nil
        }
        
        if resp != nil {
            resp.Body.Close()
        }
        
        if attempt < 2 {
            time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
        }
    }
    
    return resp, err
}

func (c *Client) Get(ctx context.Context, endpoint string) (*http.Response, error) {
    return c.Do(ctx, "GET", endpoint, nil)
}

func (c *Client) Post(ctx context.Context, endpoint string, body interface{}) (*http.Response, error) {
    return c.Do(ctx, "POST", endpoint, body)
}

func (c *Client) GetJSON(ctx context.Context, endpoint string, result interface{}) error {
    resp, err := c.Get(ctx, endpoint)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode >= 400 {
        return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
    }
    
    return json.NewDecoder(resp.Body).Decode(result)
}

func (c *Client) PostJSON(ctx context.Context, endpoint string, body, result interface{}) error {
    resp, err := c.Post(ctx, endpoint, body)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode >= 400 {
        return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
    }
    
    if result != nil {
        return json.NewDecoder(resp.Body).Decode(result)
    }
    
    return nil
}
```

#### Step 1.2: Replace Usage in Gateways
**Files to update**:
- `services/users/gateway/http/match.go`
- `services/users/gateway/http/rides.go`
- `services/match/gateway/http.go`

**Before**:
```go
httpclient "github.com/piresc/nebengjek/internal/pkg/http"
client := httpclient.NewAPIKeyClient(config, "match-service", baseURL)
```

**After**:
```go
simpleclient "github.com/piresc/nebengjek/internal/pkg/http"
client := simpleclient.NewClient(simpleclient.Config{
    APIKey:  config.MatchService,
    BaseURL: baseURL,
    Timeout: 30 * time.Second,
})
```

### Phase 2: Unified Middleware Stack

#### Step 2.1: Create Unified Middleware
**File**: `internal/pkg/middleware/unified.go`

```go
package middleware

import (
    "context"
    "fmt"
    "net/http"
    "runtime/debug"
    "time"
    
    "github.com/google/uuid"
    "github.com/labstack/echo/v4"
    "github.com/newrelic/go-agent/v3/newrelic"
    "go.uber.org/zap"
)

type UnifiedConfig struct {
    Logger      *zap.Logger
    NewRelic    *newrelic.Application
    APIKeys     map[string]string
    ServiceName string
}

type UnifiedMiddleware struct {
    config UnifiedConfig
}

func NewUnifiedMiddleware(config UnifiedConfig) *UnifiedMiddleware {
    return &UnifiedMiddleware{config: config}
}

func (m *UnifiedMiddleware) Handler() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            start := time.Now()
            
            // 1. Generate Request ID
            requestID := c.Request().Header.Get("X-Request-ID")
            if requestID == "" {
                requestID = uuid.New().String()
            }
            c.Response().Header().Set("X-Request-ID", requestID)
            
            // 2. Add to context
            ctx := context.WithValue(c.Request().Context(), "request_id", requestID)
            ctx = context.WithValue(ctx, "service_name", m.config.ServiceName)
            c.SetRequest(c.Request().WithContext(ctx))
            
            // 3. Setup New Relic (if enabled)
            var txn *newrelic.Transaction
            if m.config.NewRelic != nil {
                txn = m.config.NewRelic.StartTransaction(c.Request().URL.Path)
                defer txn.End()
                txn.SetWebRequestHTTP(c.Request())
                c.Set("nr_txn", txn)
            }
            
            // 4. Panic recovery
            defer func() {
                if r := recover(); r != nil {
                    m.handlePanic(c, r, requestID, txn)
                }
            }()
            
            // 5. Execute handler
            err := next(c)
            
            // 6. Log request
            duration := time.Since(start)
            m.logRequest(c, requestID, duration, err)
            
            // 7. Set New Relic response
            if txn != nil {
                txn.SetWebResponse(c.Response().Writer)
            }
            
            return err
        }
    }
}

func (m *UnifiedMiddleware) APIKeyHandler(allowedServices ...string) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            apiKey := c.Request().Header.Get("X-API-Key")
            if apiKey == "" {
                return echo.NewHTTPError(http.StatusUnauthorized, "API key required")
            }
            
            // Validate API key
            valid := false
            for _, service := range allowedServices {
                if expectedKey, exists := m.config.APIKeys[service]; exists && expectedKey == apiKey {
                    valid = true
                    c.Set("api_service", service)
                    break
                }
            }
            
            if !valid {
                return echo.NewHTTPError(http.StatusUnauthorized, "Invalid API key")
            }
            
            return next(c)
        }
    }
}

func (m *UnifiedMiddleware) handlePanic(c echo.Context, r interface{}, requestID string, txn *newrelic.Transaction) {
    stack := debug.Stack()
    
    // Log panic
    if m.config.Logger != nil {
        m.config.Logger.Error("Panic recovered",
            zap.Any("panic", r),
            zap.String("request_id", requestID),
            zap.String("method", c.Request().Method),
            zap.String("path", c.Request().URL.Path),
            zap.String("stack", string(stack)),
        )
    }
    
    // Report to New Relic
    if txn != nil {
        txn.NoticeError(fmt.Errorf("panic: %v", r))
    }
    
    // Send error response
    if !c.Response().Committed {
        c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "error":      "Internal Server Error",
            "request_id": requestID,
        })
    }
}

func (m *UnifiedMiddleware) logRequest(c echo.Context, requestID string, duration time.Duration, err error) {
    if m.config.Logger == nil {
        return
    }
    
    fields := []zap.Field{
        zap.String("request_id", requestID),
        zap.String("method", c.Request().Method),
        zap.String("path", c.Request().URL.Path),
        zap.Int("status", c.Response().Status),
        zap.Duration("duration", duration),
        zap.String("ip", c.RealIP()),
    }
    
    if err != nil {
        fields = append(fields, zap.Error(err))
        m.config.Logger.Error("Request failed", fields...)
    } else {
        m.config.Logger.Info("Request completed", fields...)
    }
}
```

#### Step 2.2: Update Main Files
**File**: `cmd/users/main.go` (lines 117-126)

**Before**:
```go
// Add middlewares in standard order
e.Use(middleware.PanicRecoveryWithZapMiddleware(zapLogger))
e.Use(nrecho.Middleware(nrApp))
e.Use(middleware.RequestIDMiddleware())
e.Use(middleware.RequestContextMiddleware(appName))
e.Use(logger.ZapEchoMiddleware(zapLogger))

// Initialize API key middleware for internal routes only
apiKeyMiddleware := middleware.NewAPIKeyMiddleware(&configs.APIKey)
```

**After**:
```go
// Single unified middleware
unifiedMW := middleware.NewUnifiedMiddleware(middleware.UnifiedConfig{
    Logger:      zapLogger.Logger, // Extract zap.Logger from wrapper
    NewRelic:    nrApp,
    APIKeys: map[string]string{
        "user-service":     configs.APIKey.UserService,
        "match-service":    configs.APIKey.MatchService,
        "rides-service":    configs.APIKey.RidesService,
        "location-service": configs.APIKey.LocationService,
    },
    ServiceName: appName,
})

e.Use(unifiedMW.Handler())
```

### Phase 3: Context Simplification

#### Step 3.1: Create Simple Context Helpers
**File**: `internal/pkg/context/helpers.go`

```go
package context

import (
    "context"
)

type contextKey string

const (
    RequestIDKey  contextKey = "request_id"
    UserIDKey     contextKey = "user_id"
    ServiceKey    contextKey = "service_name"
    TraceIDKey    contextKey = "trace_id"
)

func WithRequestID(ctx context.Context, requestID string) context.Context {
    return context.WithValue(ctx, RequestIDKey, requestID)
}

func GetRequestID(ctx context.Context) string {
    if id, ok := ctx.Value(RequestIDKey).(string); ok {
        return id
    }
    return ""
}

func WithUserID(ctx context.Context, userID string) context.Context {
    return context.WithValue(ctx, UserIDKey, userID)
}

func GetUserID(ctx context.Context) string {
    if id, ok := ctx.Value(UserIDKey).(string); ok {
        return id
    }
    return ""
}

func WithServiceName(ctx context.Context, service string) context.Context {
    return context.WithValue(ctx, ServiceKey, service)
}

func GetServiceName(ctx context.Context) string {
    if name, ok := ctx.Value(ServiceKey).(string); ok {
        return name
    }
    return ""
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
    return context.WithValue(ctx, TraceIDKey, traceID)
}

func GetTraceID(ctx context.Context) string {
    if id, ok := ctx.Value(TraceIDKey).(string); ok {
        return id
    }
    return ""
}
```

#### Step 3.2: Remove RequestContext Package
**Files to delete**:
- `internal/pkg/requestcontext/request.go`
- `internal/pkg/middleware/request_context.go`

**Files to update**: Replace all imports and usage

### Phase 4: Optional APM Integration

#### Step 4.1: Create APM Abstraction
**File**: `internal/pkg/observability/tracer.go`

```go
package observability

import (
    "context"
    "net/http"
    
    "github.com/newrelic/go-agent/v3/newrelic"
)

type Tracer interface {
    StartTransaction(name string) Transaction
    StartSegment(ctx context.Context, name string) (context.Context, func())
}

type Transaction interface {
    End()
    SetWebRequest(*http.Request)
    SetWebResponse(http.ResponseWriter)
    NoticeError(error)
    AddAttribute(key string, value interface{})
}

// NoOp implementation for testing
type NoOpTracer struct{}
type NoOpTransaction struct{}

func (NoOpTracer) StartTransaction(name string) Transaction { return NoOpTransaction{} }
func (NoOpTracer) StartSegment(ctx context.Context, name string) (context.Context, func()) {
    return ctx, func() {}
}

func (NoOpTransaction) End() {}
func (NoOpTransaction) SetWebRequest(*http.Request) {}
func (NoOpTransaction) SetWebResponse(http.ResponseWriter) {}
func (NoOpTransaction) NoticeError(error) {}
func (NoOpTransaction) AddAttribute(key string, value interface{}) {}

// New Relic implementation
type NewRelicTracer struct {
    app *newrelic.Application
}

func NewNewRelicTracer(app *newrelic.Application) *NewRelicTracer {
    return &NewRelicTracer{app: app}
}

func (t *NewRelicTracer) StartTransaction(name string) Transaction {
    return &NewRelicTransaction{txn: t.app.StartTransaction(name)}
}

func (t *NewRelicTracer) StartSegment(ctx context.Context, name string) (context.Context, func()) {
    if txn := newrelic.FromContext(ctx); txn != nil {
        segment := txn.StartSegment(name)
        return ctx, segment.End
    }
    return ctx, func() {}
}

type NewRelicTransaction struct {
    txn *newrelic.Transaction
}

func (t *NewRelicTransaction) End() { t.txn.End() }
func (t *NewRelicTransaction) SetWebRequest(r *http.Request) { t.txn.SetWebRequestHTTP(r) }
func (t *NewRelicTransaction) SetWebResponse(w http.ResponseWriter) { t.txn.SetWebResponse(w) }
func (t *NewRelicTransaction) NoticeError(err error) { t.txn.NoticeError(err) }
func (t *NewRelicTransaction) AddAttribute(key string, value interface{}) { t.txn.AddAttribute(key, value) }
```

### Phase 5: Remove Over-Engineering

#### Step 5.1: Delete Unnecessary Files
**Files to delete**:
- `internal/pkg/circuitbreaker/breaker.go` (314 lines)
- `internal/pkg/retry/exponential.go` (266 lines)
- `internal/pkg/http/enhanced_client.go` (111 lines)
- `internal/pkg/http/client_with_apikey.go` (230 lines)
- `internal/pkg/middleware/panic_recovery.go` (389 lines)

**Total lines removed**: ~1,310 lines

#### Step 5.2: Update All Service Main Files
**Files to update**:
- `cmd/users/main.go`
- `cmd/match/main.go`
- `cmd/rides/main.go`
- `cmd/location/main.go`

## Migration Checklist

### Pre-Migration
- [ ] Create feature branch
- [ ] Backup current middleware configurations
- [ ] Document current API contracts
- [ ] Set up monitoring for migration

### Phase 1: HTTP Client Migration
- [ ] Create `internal/pkg/http/simple_client.go`
- [ ] Update gateway files to use new client
- [ ] Test HTTP calls work correctly
- [ ] Remove old HTTP client files

### Phase 2: Middleware Migration
- [ ] Create `internal/pkg/middleware/unified.go`
- [ ] Update one service main file (start with least critical)
- [ ] Test middleware functionality
- [ ] Migrate remaining services
- [ ] Remove old middleware files

### Phase 3: Context Migration
- [ ] Create `internal/pkg/context/helpers.go`
- [ ] Replace requestcontext usage across codebase
- [ ] Update tests
- [ ] Remove requestcontext package

### Phase 4: APM Migration
- [ ] Create `internal/pkg/observability/tracer.go`
- [ ] Update middleware to use tracer interface
- [ ] Make APM optional in configuration
- [ ] Update tests to use NoOp tracer

### Phase 5: Cleanup
- [ ] Delete over-engineered components
- [ ] Update documentation
- [ ] Run full test suite
- [ ] Performance testing

## Testing Strategy

### Unit Tests
```go
// Example test for unified middleware
func TestUnifiedMiddleware(t *testing.T) {
    config := middleware.UnifiedConfig{
        Logger:      zap.NewNop(),
        ServiceName: "test-service",
    }
    
    mw := middleware.NewUnifiedMiddleware(config)
    
    e := echo.New()
    e.Use(mw.Handler())
    
    req := httptest.NewRequest(http.MethodGet, "/test", nil)
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)
    
    handler := func(c echo.Context) error {
        return c.String(http.StatusOK, "test")
    }
    
    err := mw.Handler()(handler)(c)
    assert.NoError(t, err)
    assert.Equal(t, http.StatusOK, rec.Code)
    assert.NotEmpty(t, rec.Header().Get("X-Request-ID"))
}
```

### Integration Tests
```go
// Example integration test
func TestServiceIntegration(t *testing.T) {
    // Use NoOp tracer for testing
    tracer := &observability.NoOpTracer{}
    
    // Test service with simplified middleware
    // Verify all functionality works without APM overhead
}
```

## Expected Results

### Code Reduction
- **HTTP Clients**: 571 lines → ~100 lines (82% reduction)
- **Middleware**: 6 separate middleware → 1 unified (90% reduction)
- **Context Management**: 126 lines → ~50 lines (60% reduction)
- **Circuit Breaker**: 314 lines → 0 lines (100% reduction)
- **Retry Logic**: 266 lines → ~20 lines (92% reduction)
- **Total**: ~1,300 lines removed

### Development Benefits
- Single middleware configuration
- Consistent HTTP client across services
- Standard Go context patterns
- Optional APM integration
- Easier testing with mocks

### Maintenance Benefits
- Single point of failure analysis
- Unified logging format
- Simplified dependency management
- Easier debugging
- Clear separation of concerns

## Rollback Plan

If issues arise during migration:

1. **Immediate Rollback**: Revert to previous middleware configuration
2. **Partial Rollback**: Keep new HTTP client, revert middleware
3. **Service-by-Service**: Rollback individual services while keeping others on new system

## Post-Migration Tasks

1. Update team documentation
2. Create coding standards for new patterns
3. Set up monitoring for new middleware
4. Performance benchmarking
5. Team training on simplified architecture