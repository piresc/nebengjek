# Security Implementation

## Overview

NebengJek implements comprehensive security measures including API key authentication for service-to-service communication, JWT-based WebSocket authentication, panic recovery mechanisms, and security scanning tools.

## API Key Authentication

### Architecture

API key authentication secures internal service-to-service communication, ensuring only authorized services can access internal endpoints.

#### API Key Middleware
**File**: [`internal/pkg/middleware/apikey.go`](../internal/pkg/middleware/apikey.go)

```go
type APIKeyMiddleware struct {
    config *models.APIKeyConfig
}

func NewAPIKeyMiddleware(config *models.APIKeyConfig) *APIKeyMiddleware {
    return &APIKeyMiddleware{config: config}
}

func (m *APIKeyMiddleware) ValidateAPIKey(allowedServices ...string) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            apiKey := c.Request().Header.Get("X-API-Key")
            
            if apiKey == "" {
                logger.Warn("API key missing in request",
                    logger.String("path", c.Request().URL.Path),
                    logger.String("method", c.Request().Method),
                    logger.String("remote_addr", c.Request().RemoteAddr))
                return utils.ErrorResponseHandler(c, http.StatusUnauthorized, "API key is required")
            }

            // Validate API key against allowed services
            serviceAPIKeys := m.getServiceAPIKeys()
            validKey := false
            var validService string
            
            for _, service := range allowedServices {
                if serviceKey, exists := serviceAPIKeys[service]; exists && serviceKey == apiKey {
                    validKey = true
                    validService = service
                    break
                }
            }

            if !validKey {
                logger.Warn("Invalid API key provided",
                    logger.String("path", c.Request().URL.Path),
                    logger.String("method", c.Request().Method),
                    logger.String("remote_addr", c.Request().RemoteAddr),
                    logger.Strings("allowed_services", allowedServices))
                return utils.ErrorResponseHandler(c, http.StatusUnauthorized, "Invalid API key")
            }

            // Add service context to request
            c.Set("authenticated_service", validService)
            
            logger.Debug("API key validation successful",
                logger.String("service", validService),
                logger.String("path", c.Request().URL.Path))

            return next(c)
        }
    }
}
```

#### Service API Key Mapping
```go
func (m *APIKeyMiddleware) getServiceAPIKeys() map[string]string {
    return map[string]string{
        "users-service":    m.config.UsersService,
        "location-service": m.config.LocationService,
        "match-service":    m.config.MatchService,
        "rides-service":    m.config.RidesService,
    }
}
```

### Configuration

#### API Key Configuration Structure
```go
type APIKeyConfig struct {
    UsersService    string `json:"users_service"`
    LocationService string `json:"location_service"`
    MatchService    string `json:"match_service"`
    RidesService    string `json:"rides_service"`
}
```

#### Environment Variables
```bash
# API Key Configuration
API_KEY_USERS_SERVICE=users-service-key-2024
API_KEY_LOCATION_SERVICE=location-service-key-2024
API_KEY_MATCH_SERVICE=match-service-key-2024
API_KEY_RIDES_SERVICE=rides-service-key-2024
```

### HTTP Client with API Key Authentication

#### API Key HTTP Client
**File**: [`internal/pkg/http/client_with_apikey.go`](../internal/pkg/http/client_with_apikey.go)

```go
type APIKeyClient struct {
    client      *http.Client
    baseURL     string
    apiKey      string
    serviceName string
    timeout     time.Duration
}

func NewAPIKeyClient(config *models.APIKeyConfig, serviceName, baseURL string) *APIKeyClient {
    var apiKey string
    
    // Get appropriate API key based on service name
    switch serviceName {
    case "match-service":
        apiKey = config.MatchService
    case "rides-service":
        apiKey = config.RidesService
    case "location-service":
        apiKey = config.LocationService
    default:
        logger.Warn("Unknown service name for API key", logger.String("service", serviceName))
    }

    return &APIKeyClient{
        client: &http.Client{
            Timeout: DefaultTimeout,
            Transport: &http.Transport{
                MaxIdleConns:        100,
                MaxIdleConnsPerHost: 10,
                IdleConnTimeout:     90 * time.Second,
            },
        },
        baseURL:     baseURL,
        apiKey:      apiKey,
        serviceName: serviceName,
        timeout:     DefaultTimeout,
    }
}
```

#### Request Implementation with API Key
```go
func (c *APIKeyClient) doRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
    url := c.baseURL + endpoint
    
    var reqBody io.Reader
    if body != nil {
        jsonBody, err := json.Marshal(body)
        if err != nil {
            return nil, fmt.Errorf("failed to marshal request body: %w", err)
        }
        reqBody = bytes.NewBuffer(jsonBody)
    }

    req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    // Set headers
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("User-Agent", "nebengjek-"+c.serviceName)
    
    // Add API key header
    if c.apiKey != "" {
        req.Header.Set(APIKeyHeader, c.apiKey)
    }

    // Add New Relic distributed tracing headers
    if txn := newrelic.FromContext(ctx); txn != nil {
        txn.InsertDistributedTraceHeaders(req.Header)
    }

    // Log request details
    logger.Debug("Making HTTP request with API key",
        logger.String("method", method),
        logger.String("url", url),
        logger.String("service", c.serviceName),
        logger.Bool("has_api_key", c.apiKey != ""))

    return c.client.Do(req)
}
```

### Route Protection

#### Internal Route Groups
**Example**: [`services/users/handler/routes.go`](../services/users/handler/routes.go)

```go
func (h *Handler) RegisterRoutes(e *echo.Echo, apiKeyMiddleware *middleware.APIKeyMiddleware) {
    // Public routes (no authentication required)
    authGroup := e.Group("/auth")
    authGroup.POST("/login", h.authHandler.Login)
    authGroup.POST("/verify", h.authHandler.VerifyOTP)

    // Protected routes (JWT required)
    protected := e.Group("/api/v1", middleware.JWTAuthMiddleware(h.config.JWT))
    protected.GET("/users/:id", h.userHandler.GetUser)
    protected.PUT("/users/:id", h.userHandler.UpdateUser)
    protected.POST("/drivers", h.userHandler.RegisterDriver)

    // WebSocket routes (JWT required)
    wsGroup := protected.Group("/ws")
    wsGroup.GET("", h.wsManager.HandleWebSocket)

    // Internal routes (API key required)
    internal := e.Group("/internal", apiKeyMiddleware.ValidateAPIKey("match-service", "rides-service"))
    internal.GET("/users/:id", h.userHandler.GetUserInternal)
    internal.GET("/drivers/nearby", h.userHandler.GetNearbyDrivers)
}
```

## JWT WebSocket Authentication

### WebSocket Authentication Flow

#### JWT Claims Structure
**File**: [`internal/pkg/models/websocket.go`](../internal/pkg/models/websocket.go)

```go
type WebSocketClaims struct {
    jwt.RegisteredClaims
    UserID string `json:"user_id"`
    Role   string `json:"role"`
}

type WebSocketClient struct {
    UserID string
    Role   string
    Conn   *websocket.Conn
}
```

#### WebSocket Manager Authentication
**File**: [`internal/pkg/websocket/manager.go`](../internal/pkg/websocket/manager.go)

```go
func (m *Manager) HandleConnection(c echo.Context, handleClient func(*models.WebSocketClient, *websocket.Conn) error) error {
    // Authenticate client using JWT
    client, err := m.authenticateClient(c)
    if err != nil {
        logger.Error("WebSocket authentication failed", logger.Err(err))
        return echo.NewHTTPError(http.StatusUnauthorized, "Authentication failed")
    }

    // Upgrade HTTP connection to WebSocket
    ws, err := m.upgrader.Upgrade(c.Response(), c.Request(), nil)
    if err != nil {
        logger.Error("WebSocket upgrade failed", logger.Err(err))
        return err
    }
    defer ws.Close()

    // Add client to manager
    m.AddClient(client)
    defer m.RemoveClient(client.UserID)

    logger.Info("WebSocket client connected",
        logger.String("user_id", client.UserID),
        logger.String("role", client.Role))

    // Handle client connection
    return handleClient(client, ws)
}

func (m *Manager) authenticateClient(c echo.Context) (*models.WebSocketClient, error) {
    // Extract JWT token from Authorization header
    authHeader := c.Request().Header.Get("Authorization")
    if authHeader == "" {
        return nil, errors.New("authorization header missing")
    }

    tokenString := strings.TrimPrefix(authHeader, "Bearer ")
    if tokenString == authHeader {
        return nil, errors.New("bearer token format invalid")
    }

    // Validate JWT token
    claims, err := m.validateToken(tokenString)
    if err != nil {
        return nil, fmt.Errorf("token validation failed: %w", err)
    }

    return &models.WebSocketClient{
        UserID: claims.UserID,
        Role:   claims.Role,
    }, nil
}
```

#### Token Validation
```go
func (m *Manager) validateToken(tokenString string) (*models.WebSocketClaims, error) {
    claims := &models.WebSocketClaims{}
    token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(m.cfg.SecretKey), nil
    })

    if err != nil {
        return nil, err
    }

    if !token.Valid {
        return nil, errors.New("token is invalid")
    }

    // Validate token expiration
    if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
        return nil, errors.New("token has expired")
    }

    return claims, nil
}
```

### WebSocket Security Features

#### Connection Management
```go
type Manager struct {
    sync.RWMutex
    clients  map[string]*models.WebSocketClient
    cfg      models.JWTConfig
    upgrader websocket.Upgrader
}

func (m *Manager) AddClient(client *models.WebSocketClient) {
    m.Lock()
    defer m.Unlock()
    
    // Remove existing connection for same user
    if existingClient, exists := m.clients[client.UserID]; exists {
        existingClient.Conn.Close()
        logger.Info("Replaced existing WebSocket connection",
            logger.String("user_id", client.UserID))
    }
    
    m.clients[client.UserID] = client
}

func (m *Manager) RemoveClient(userID string) {
    m.Lock()
    defer m.Unlock()
    delete(m.clients, userID)
    
    logger.Info("WebSocket client disconnected",
        logger.String("user_id", userID))
}
```

#### Message Security
```go
func (m *Manager) SendCategorizedError(conn *websocket.Conn, err error, code string, severity constants.ErrorSeverity, userID string) error {
    // Always log detailed error server-side
    logger.Error("WebSocket operation failed",
        logger.String("user_id", userID),
        logger.String("error_code", code),
        logger.Err(err))

    var clientMessage string
    switch severity {
    case constants.ErrorSeverityClient:
        clientMessage = err.Error() // Full error for client issues
    case constants.ErrorSeverityServer:
        clientMessage = "Internal server error" // Generic message for server issues
    case constants.ErrorSeveritySecurity:
        clientMessage = "Authentication required" // Minimal info for security issues
    }

    return m.SendErrorMessage(conn, code, clientMessage)
}
```

## Panic Recovery Mechanisms

### Enhanced Panic Recovery Middleware

#### Panic Recovery Configuration
**File**: [`internal/pkg/middleware/panic_recovery.go`](../internal/pkg/middleware/panic_recovery.go)

```go
type PanicRecoveryConfig struct {
    DisableStackAll   bool
    DisablePrintStack bool
    LogLevel         string
    SkipFrames       int
}

func DefaultPanicRecoveryConfig() PanicRecoveryConfig {
    return PanicRecoveryConfig{
        DisableStackAll:   false,
        DisablePrintStack: false,
        LogLevel:         "error",
        SkipFrames:       3,
    }
}
```

#### Panic Handler Implementation
```go
func PanicRecoveryWithZapMiddleware(zapLogger *logger.ZapLogger) echo.MiddlewareFunc {
    config := DefaultPanicRecoveryConfig()
    
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            defer func() {
                if r := recover(); r != nil {
                    handlePanic(c, r, config)
                }
            }()
            return next(c)
        }
    }
}

func handlePanic(c echo.Context, r interface{}, config PanicRecoveryConfig) {
    // Capture stack trace
    stack := debug.Stack()
    
    // Extract request information
    req := c.Request()
    requestInfo := map[string]interface{}{
        "method":     req.Method,
        "url":        req.URL.String(),
        "user_agent": req.UserAgent(),
        "remote_ip":  c.RealIP(),
        "headers":    sanitizeHeaders(req.Header),
    }

    // Get caller information
    caller := getCaller(config.SkipFrames)
    
    // Create comprehensive error context
    errorContext := map[string]interface{}{
        "panic_value":    r,
        "request_info":   requestInfo,
        "caller":         caller,
        "stack_trace":    string(stack),
        "timestamp":      time.Now().UTC(),
        "service":        "nebengjek",
        "component":      "panic_recovery",
    }

    // Log panic with full context
    logger.Error("Panic recovered",
        logger.Any("panic_value", r),
        logger.Any("request_info", requestInfo),
        logger.String("caller", caller),
        logger.String("stack_trace", string(stack)))

    // Report to New Relic if available
    if txn := newrelic.FromContext(c.Request().Context()); txn != nil {
        txn.NoticeError(fmt.Errorf("panic recovered: %v", r))
        for key, value := range errorContext {
            txn.AddAttribute(key, value)
        }
    }

    // Send error response
    err := c.JSON(http.StatusInternalServerError, map[string]interface{}{
        "error":   "Internal server error",
        "code":    "INTERNAL_ERROR",
        "message": "An unexpected error occurred",
    })
    
    if err != nil {
        logger.Error("Failed to send panic recovery response", logger.Err(err))
    }
}
```

#### Header Sanitization
```go
func sanitizeHeaders(headers http.Header) map[string]string {
    sanitized := make(map[string]string)
    sensitiveHeaders := map[string]bool{
        "authorization": true,
        "cookie":        true,
        "set-cookie":    true,
        "x-api-key":     true,
    }

    for key, values := range headers {
        lowerKey := strings.ToLower(key)
        if sensitiveHeaders[lowerKey] {
            sanitized[key] = "[REDACTED]"
        } else {
            sanitized[key] = strings.Join(values, ", ")
        }
    }

    return sanitized
}
```

### Graceful Shutdown Procedures

#### Graceful Server Implementation
**File**: [`internal/pkg/server/server.go`](../internal/pkg/server/server.go)

```go
type GracefulServer struct {
    echo   *echo.Echo
    logger *logger.ZapLogger
    port   int
}

func NewGracefulServer(e *echo.Echo, zapLogger *logger.ZapLogger, port int) *GracefulServer {
    return &GracefulServer{
        echo:   e,
        logger: zapLogger,
        port:   port,
    }
}

func (s *GracefulServer) Start() error {
    // Start server in goroutine
    go func() {
        addr := fmt.Sprintf(":%d", s.port)
        s.logger.Info("Starting HTTP server", logger.String("address", addr))
        
        if err := s.echo.Start(addr); err != nil && err != http.ErrServerClosed {
            s.logger.Fatal("Failed to start server", logger.Err(err))
        }
    }()

    // Wait for interrupt signal to gracefully shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    
    sig := <-quit
    s.logger.Info("Received shutdown signal", logger.String("signal", sig.String()))

    return s.shutdown()
}

func (s *GracefulServer) shutdown() error {
    // Create shutdown context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    s.logger.Info("Starting graceful shutdown...")

    // Shutdown HTTP server
    if err := s.echo.Shutdown(ctx); err != nil {
        s.logger.Error("Server forced to shutdown", logger.Err(err))
        return err
    }

    s.logger.Info("Server exited gracefully")
    return nil
}
```

#### Shutdown Manager for Multiple Components
```go
type ShutdownManager struct {
    functions []func(context.Context) error
    logger    *logger.ZapLogger
}

func NewShutdownManager(logger *logger.ZapLogger) *ShutdownManager {
    return &ShutdownManager{
        functions: make([]func(context.Context) error, 0),
        logger:    logger,
    }
}

func (sm *ShutdownManager) Add(fn func(context.Context) error) {
    sm.functions = append(sm.functions, fn)
}

func (sm *ShutdownManager) Shutdown(ctx context.Context) error {
    sm.logger.Info("Starting graceful shutdown of components", logger.Int("components", len(sm.functions)))

    for i, fn := range sm.functions {
        sm.logger.Info("Shutting down component", logger.Int("component", i+1))
        
        if err := fn(ctx); err != nil {
            sm.logger.Error("Component shutdown failed", 
                logger.Int("component", i+1), 
                logger.Err(err))
            return err
        }
    }

    sm.logger.Info("All components shut down successfully")
    return nil
}
```

## Security Scanning and Tools

### GitLeaks Secret Scanning

#### GitLeaks Configuration
**File**: [`.gitleaks.toml`](../.gitleaks.toml)

```toml
title = "NebengJek Security Configuration"

[extend]
useDefault = true

[[rules]]
description = "API Key"
id = "api-key"
regex = '''(?i)(api[_-]?key|apikey)['"]*\s*[:=]\s*['"][a-zA-Z0-9]{20,}['"]'''
tags = ["key", "API"]

[[rules]]
description = "Database URL"
id = "database-url"
regex = '''(?i)(database[_-]?url|db[_-]?url)['"]*\s*[:=]\s*['"][^'"]+['"]'''
tags = ["database", "url"]

[[rules]]
description = "JWT Secret"
id = "jwt-secret"
regex = '''(?i)(jwt[_-]?secret|secret[_-]?key)['"]*\s*[:=]\s*['"][a-zA-Z0-9]{20,}['"]'''
tags = ["jwt", "secret"]

[allowlist]
description = "Allowlist for test files"
files = [
    '''.*_test\.go$''',
    '''.*\.example$''',
    '''.*\.template$'''
]
```

#### CI/CD Secret Scanning
**File**: [`.github/workflows/continuous-integration.yml`](../.github/workflows/continuous-integration.yml)

```yaml
secret-scan:
  name: Secret Scanning
  runs-on: ubuntu-latest
  steps:
    - name: Checkout code
      uses: actions/checkout@v3
      with:
        fetch-depth: 0  # Fetch full history for comprehensive scanning

    - name: Run GitLeaks Secret Scan
      uses: gitleaks/gitleaks-action@v2
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        GITLEAKS_LICENSE: ${{ secrets.GITLEAKS_LICENSE }}
```

### SonarCloud Integration

#### SonarCloud Configuration
**File**: [`sonar-project.properties`](../sonar-project.properties)

```properties
# Quality Gate Settings
sonar.projectKey=piresc_nebengjek
sonar.organization=nebengjek-prod
sonar.host.url=https://sonarcloud.io

# Coverage Analysis
sonar.go.coverage.reportPaths=coverage.txt
sonar.coverage.exclusions=**/cmd/**/main.go,**/mocks/**

# Code Exclusions
sonar.exclusions=**/*_test.go,**/vendor/**,**/bin/**

# Security Analysis
sonar.security.hotspots.inheritFromParent=true
sonar.security.review.category=security
```

#### Security Hotspot Detection
```yaml
# SonarCloud security analysis in CI
- name: SonarCloud Scan
  uses: SonarSource/sonarqube-scan-action@v5.0.0
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    SONAR_TOKEN: ${{ secrets.SONAR_TOKEN }}
```

### Input Validation and Sanitization

#### Request Validation
```go
type Validator struct {
    validator *validator.Validate
}

func NewValidator() *Validator {
    v := validator.New()
    
    // Register custom validators
    v.RegisterValidation("msisdn", validateMSISDN)
    v.RegisterValidation("uuid", validateUUID)
    
    return &Validator{validator: v}
}

func (v *Validator) Validate(i interface{}) error {
    if err := v.validator.Struct(i); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    return nil
}

func validateMSISDN(fl validator.FieldLevel) bool {
    msisdn := fl.Field().String()
    // Validate Indonesian mobile number format
    matched, _ := regexp.MatchString(`^(\+62|62|0)8[1-9][0-9]{6,9}$`, msisdn)
    return matched
}
```

#### SQL Injection Prevention
```go
// Use parameterized queries
func (r *UserRepo) GetByMSISDN(ctx context.Context, msisdn string) (*models.User, error) {
    var user models.User
    query := "SELECT id, msisdn, fullname, role FROM users WHERE msisdn = $1"
    
    err := r.db.GetContext(ctx, &user, query, msisdn)
    if err != nil {
        return nil, fmt.Errorf("failed to get user by MSISDN: %w", err)
    }
    
    return &user, nil
}
```

## Security Best Practices

### Environment Variable Management
```bash
# Use strong, unique API keys
API_KEY_MATCH_SERVICE=$(openssl rand -hex 32)
API_KEY_RIDES_SERVICE=$(openssl rand -hex 32)

# Rotate JWT secrets regularly
JWT_SECRET_KEY=$(openssl rand -hex 64)

# Use secure database credentials
DB_PASSWORD=$(openssl rand -base64 32)
```

### Rate Limiting
```go
func RateLimitMiddleware(requests int, window time.Duration) echo.MiddlewareFunc {
    limiter := rate.NewLimiter(rate.Every(window/time.Duration(requests)), requests)
    
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            if !limiter.Allow() {
                return echo.NewHTTPError(http.StatusTooManyRequests, "Rate limit exceeded")
            }
            return next(c)
        }
    }
}
```

### CORS Configuration
```go
func CORSConfig() echo.MiddlewareFunc {
    return middleware.CORSWithConfig(middleware.CORSConfig{
        AllowOrigins:     []string{"https://app.nebengjek.com"},
        AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
        AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
        AllowCredentials: true,
        MaxAge:           86400, // 24 hours
    })
}
```

## Security Monitoring

### Security Event Logging
```go
func LogSecurityEvent(event string, userID string, details map[string]interface{}) {
    logger.Warn("Security event detected",
        logger.String("event", event),
        logger.String("user_id", userID),
        logger.Any("details", details),
        logger.String("timestamp", time.Now().UTC().Format(time.RFC3339)))
}

// Usage examples
LogSecurityEvent("invalid_api_key", "", map[string]interface{}{
    "remote_ip": c.RealIP(),
    "path":      c.Request().URL.Path,
    "method":    c.Request().Method,
})

LogSecurityEvent("jwt_validation_failed", userID, map[string]interface{}{
    "error":     err.Error(),
    "remote_ip": c.RealIP(),
})
```

### Intrusion Detection
```go
func DetectSuspiciousActivity(userID string, activity string) {
    // Implement rate limiting per user
    key := fmt.Sprintf("activity:%s:%s", userID, activity)
    count, _ := redisClient.Incr(ctx, key).Result()
    redisClient.Expire(ctx, key, time.Hour)
    
    if count > 100 { // Threshold for suspicious activity
        LogSecurityEvent("suspicious_activity", userID, map[string]interface{}{
            "activity": activity,
            "count":    count,
            "window":   "1 hour",
        })
        
        // Implement temporary blocking or additional verification
    }
}
```

## See Also
- [Database Architecture](database-architecture.md)
- [WebSocket Implementation](websocket-implementation.md)
- [Monitoring and Observability](monitoring-observability.md)
- [Testing Strategies](testing-strategies.md)