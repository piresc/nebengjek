# WebSocket Migration Implementation Plan
## From Manual to Echo Native - Detailed Implementation Guide

### Executive Summary

This document provides a comprehensive implementation plan for migrating from manual websocket handling to Echo's native websocket support. Based on analysis of existing documentation, this plan addresses critical gaps, provides specific code examples, and outlines a phased migration approach.

## Current State Analysis

### Issues Identified in Existing Documentation

#### 1. **Authentication Integration Gap**
- Missing clear JWT middleware integration pattern
- Unclear user context extraction from Echo
- No standardized authentication flow

#### 2. **Connection Management Deficiencies**
- No graceful shutdown mechanism
- Missing connection pooling and rate limiting
- Potential memory leaks in client management

#### 3. **Testing Strategy Gaps**
- Limited integration testing examples
- No concurrent connection testing
- Incomplete error scenario coverage

#### 4. **Architecture Inconsistencies**
- Mixed patterns between gorilla/websocket and golang.org/x/net/websocket
- No clear separation of concerns
- Missing performance monitoring

## Improved Architecture Design

### Core Components Architecture

```mermaid
graph TB
    subgraph "Echo Server Layer"
        A[Echo Router] --> B[JWT Middleware]
        B --> C[WebSocket Upgrader]
    end
    
    subgraph "WebSocket Management Layer"
        C --> D[Connection Manager]
        D --> E[Message Router]
        E --> F[Business Logic Handlers]
    end
    
    subgraph "Business Layer"
        F --> G[User UseCase]
        G --> H[External Services]
    end
    
    subgraph "Infrastructure Layer"
        I[Metrics Collector]
        J[Error Handler]
        K[Logger]
    end
    
    D --> I
    E --> J
    F --> K
```

### Enhanced WebSocket Handler Implementation

#### 1. **Improved Authentication Integration**

```go
// services/users/handler/websocket/enhanced_handler.go
package websocket

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "sync"
    "time"

    "github.com/gorilla/websocket"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
    "github.com/golang-jwt/jwt/v4"
    
    "github.com/piresc/nebengjek/internal/pkg/constants"
    "github.com/piresc/nebengjek/internal/pkg/logger"
    "github.com/piresc/nebengjek/internal/pkg/models"
    "github.com/piresc/nebengjek/services/users"
)

// EnhancedWebSocketHandler provides improved websocket handling with Echo native support
type EnhancedWebSocketHandler struct {
    userUC          users.UserUC
    connectionMgr   *ConnectionManager
    config          *WebSocketConfig
    metrics         *WSMetrics
    upgrader        websocket.Upgrader
}

// WebSocketConfig holds configuration for websocket connections
type WebSocketConfig struct {
    MaxConnections    int           `json:"max_connections"`
    ReadTimeout       time.Duration `json:"read_timeout"`
    WriteTimeout      time.Duration `json:"write_timeout"`
    PingInterval      time.Duration `json:"ping_interval"`
    MaxMessageSize    int64         `json:"max_message_size"`
    BufferSize        int           `json:"buffer_size"`
    AllowedOrigins    []string      `json:"allowed_origins"`
    EnableCompression bool          `json:"enable_compression"`
}

// NewEnhancedWebSocketHandler creates a new enhanced websocket handler
func NewEnhancedWebSocketHandler(userUC users.UserUC, config *WebSocketConfig) *EnhancedWebSocketHandler {
    return &EnhancedWebSocketHandler{
        userUC:        userUC,
        connectionMgr: NewConnectionManager(config),
        config:        config,
        metrics:       NewWSMetrics(),
        upgrader: websocket.Upgrader{
            ReadBufferSize:    int(config.BufferSize),
            WriteBufferSize:   int(config.BufferSize),
            EnableCompression: config.EnableCompression,
            CheckOrigin:       createOriginChecker(config.AllowedOrigins),
        },
    }
}

// createOriginChecker creates a function to validate websocket origins
func createOriginChecker(allowedOrigins []string) func(r *http.Request) bool {
    if len(allowedOrigins) == 0 {
        return func(r *http.Request) bool { return true }
    }
    
    originMap := make(map[string]bool)
    for _, origin := range allowedOrigins {
        originMap[origin] = true
    }
    
    return func(r *http.Request) bool {
        origin := r.Header.Get("Origin")
        return originMap[origin]
    }
}

// HandleWebSocket handles websocket connections with enhanced Echo integration
func (h *EnhancedWebSocketHandler) HandleWebSocket(c echo.Context) error {
    // Extract user information from JWT token (validated by middleware)
    userClaims, err := h.extractUserClaims(c)
    if err != nil {
        logger.Error("Failed to extract user claims", logger.Err(err))
        return echo.NewHTTPError(http.StatusUnauthorized, "Invalid authentication")
    }

    // Check connection limits
    if !h.connectionMgr.CanAcceptConnection(userClaims.UserID) {
        logger.Warn("Connection limit exceeded", logger.String("user_id", userClaims.UserID))
        return echo.NewHTTPError(http.StatusTooManyRequests, "Connection limit exceeded")
    }

    // Upgrade to websocket
    ws, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
    if err != nil {
        logger.Error("WebSocket upgrade failed", logger.Err(err))
        return err
    }

    // Create connection context
    conn := &Connection{
        ID:        generateConnectionID(),
        UserID:    userClaims.UserID,
        Role:      userClaims.Role,
        Conn:      ws,
        Send:      make(chan []byte, h.config.BufferSize),
        Done:      make(chan struct{}),
        CreatedAt: time.Now(),
    }

    // Register connection
    h.connectionMgr.AddConnection(conn)
    h.metrics.ActiveConnections.Inc()

    logger.Info("WebSocket client connected",
        logger.String("user_id", userClaims.UserID),
        logger.String("role", userClaims.Role),
        logger.String("connection_id", conn.ID))

    // Handle connection lifecycle
    return h.handleConnection(conn)
}

// extractUserClaims extracts user claims from Echo context
func (h *EnhancedWebSocketHandler) extractUserClaims(c echo.Context) (*models.JWTClaims, error) {
    // Method 1: From JWT middleware
    if token := c.Get("user"); token != nil {
        if jwtToken, ok := token.(*jwt.Token); ok {
            if claims, ok := jwtToken.Claims.(*models.JWTClaims); ok {
                return claims, nil
            }
        }
    }

    // Method 2: From custom context values
    if userID := c.Get("user_id"); userID != nil {
        if role := c.Get("role"); role != nil {
            return &models.JWTClaims{
                UserID: userID.(string),
                Role:   role.(string),
            }, nil
        }
    }

    // Method 3: From Authorization header (fallback)
    authHeader := c.Request().Header.Get("Authorization")
    if authHeader == "" {
        return nil, fmt.Errorf("no authorization header")
    }

    // Parse JWT token manually if needed
    tokenString := authHeader[7:] // Remove "Bearer "
    return h.parseJWTToken(tokenString)
}

// parseJWTToken parses JWT token and extracts claims
func (h *EnhancedWebSocketHandler) parseJWTToken(tokenString string) (*models.JWTClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &models.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
        return []byte("your-secret-key"), nil // Use actual secret from config
    })

    if err != nil {
        return nil, err
    }

    if claims, ok := token.Claims.(*models.JWTClaims); ok && token.Valid {
        return claims, nil
    }

    return nil, fmt.Errorf("invalid token claims")
}
```

#### 2. **Enhanced Connection Management**

```go
// Connection represents a websocket connection with metadata
type Connection struct {
    ID        string
    UserID    string
    Role      string
    Conn      *websocket.Conn
    Send      chan []byte
    Done      chan struct{}
    CreatedAt time.Time
    LastPing  time.Time
    mu        sync.RWMutex
}

// ConnectionManager manages websocket connections with enhanced features
type ConnectionManager struct {
    connections     map[string]*Connection
    userConnections map[string]string // userID -> connectionID
    mu              sync.RWMutex
    config          *WebSocketConfig
    shutdown        chan struct{}
    wg              sync.WaitGroup
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(config *WebSocketConfig) *ConnectionManager {
    cm := &ConnectionManager{
        connections:     make(map[string]*Connection),
        userConnections: make(map[string]string),
        config:          config,
        shutdown:        make(chan struct{}),
    }
    
    // Start cleanup goroutine
    go cm.cleanupRoutine()
    
    return cm
}

// CanAcceptConnection checks if a new connection can be accepted
func (cm *ConnectionManager) CanAcceptConnection(userID string) bool {
    cm.mu.RLock()
    defer cm.mu.RUnlock()
    
    // Check global connection limit
    if len(cm.connections) >= cm.config.MaxConnections {
        return false
    }
    
    // Allow only one connection per user (replace existing)
    return true
}

// AddConnection adds a new connection to the manager
func (cm *ConnectionManager) AddConnection(conn *Connection) {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    
    // Remove existing connection for the same user
    if existingConnID, exists := cm.userConnections[conn.UserID]; exists {
        if existingConn, found := cm.connections[existingConnID]; found {
            existingConn.Close()
            delete(cm.connections, existingConnID)
        }
    }
    
    // Add new connection
    cm.connections[conn.ID] = conn
    cm.userConnections[conn.UserID] = conn.ID
    
    logger.Info("Connection added",
        logger.String("connection_id", conn.ID),
        logger.String("user_id", conn.UserID),
        logger.Int("total_connections", len(cm.connections)))
}

// RemoveConnection removes a connection from the manager
func (cm *ConnectionManager) RemoveConnection(connectionID string) {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    
    if conn, exists := cm.connections[connectionID]; exists {
        delete(cm.connections, connectionID)
        delete(cm.userConnections, conn.UserID)
        
        logger.Info("Connection removed",
            logger.String("connection_id", connectionID),
            logger.String("user_id", conn.UserID),
            logger.Int("total_connections", len(cm.connections)))
    }
}

// GetConnectionByUserID retrieves a connection by user ID
func (cm *ConnectionManager) GetConnectionByUserID(userID string) (*Connection, bool) {
    cm.mu.RLock()
    defer cm.mu.RUnlock()
    
    if connID, exists := cm.userConnections[userID]; exists {
        if conn, found := cm.connections[connID]; found {
            return conn, true
        }
    }
    
    return nil, false
}

// BroadcastToUsers sends a message to multiple users
func (cm *ConnectionManager) BroadcastToUsers(userIDs []string, message []byte) {
    cm.mu.RLock()
    defer cm.mu.RUnlock()
    
    for _, userID := range userIDs {
        if connID, exists := cm.userConnections[userID]; exists {
            if conn, found := cm.connections[connID]; found {
                select {
                case conn.Send <- message:
                default:
                    logger.Warn("Failed to send message, channel full",
                        logger.String("user_id", userID))
                }
            }
        }
    }
}

// GracefulShutdown gracefully shuts down all connections
func (cm *ConnectionManager) GracefulShutdown(timeout time.Duration) {
    logger.Info("Starting graceful shutdown of websocket connections")
    
    close(cm.shutdown)
    
    // Close all connections
    cm.mu.Lock()
    for _, conn := range cm.connections {
        conn.Close()
    }
    cm.mu.Unlock()
    
    // Wait for all goroutines to finish
    done := make(chan struct{})
    go func() {
        cm.wg.Wait()
        close(done)
    }()
    
    select {
    case <-done:
        logger.Info("All websocket connections closed gracefully")
    case <-time.After(timeout):
        logger.Warn("Timeout waiting for websocket connections to close")
    }
}

// cleanupRoutine periodically cleans up stale connections
func (cm *ConnectionManager) cleanupRoutine() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            cm.cleanupStaleConnections()
        case <-cm.shutdown:
            return
        }
    }
}

// cleanupStaleConnections removes connections that haven't pinged recently
func (cm *ConnectionManager) cleanupStaleConnections() {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    
    staleThreshold := time.Now().Add(-2 * cm.config.PingInterval)
    var staleConnections []string
    
    for connID, conn := range cm.connections {
        conn.mu.RLock()
        lastPing := conn.LastPing
        conn.mu.RUnlock()
        
        if lastPing.Before(staleThreshold) {
            staleConnections = append(staleConnections, connID)
        }
    }
    
    for _, connID := range staleConnections {
        if conn, exists := cm.connections[connID]; exists {
            logger.Warn("Removing stale connection",
                logger.String("connection_id", connID),
                logger.String("user_id", conn.UserID))
            conn.Close()
            delete(cm.connections, connID)
            delete(cm.userConnections, conn.UserID)
        }
    }
}

// Close closes the connection
func (c *Connection) Close() {
    select {
    case <-c.Done:
        return // Already closed
    default:
        close(c.Done)
        close(c.Send)
        c.Conn.Close()
    }
}

// SendMessage sends a message to the connection
func (c *Connection) SendMessage(message []byte) error {
    select {
    case c.Send <- message:
        return nil
    case <-c.Done:
        return fmt.Errorf("connection closed")
    default:
        return fmt.Errorf("send channel full")
    }
}

// UpdatePing updates the last ping time
func (c *Connection) UpdatePing() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.LastPing = time.Now()
}
```

#### 3. **Enhanced Message Handling with Business Logic Preservation**

```go
// handleConnection manages the connection lifecycle
func (h *EnhancedWebSocketHandler) handleConnection(conn *Connection) error {
    defer func() {
        h.connectionMgr.RemoveConnection(conn.ID)
        h.metrics.ActiveConnections.Dec()
        conn.Close()
    }()

    // Configure connection timeouts
    conn.Conn.SetReadLimit(h.config.MaxMessageSize)
    conn.Conn.SetReadDeadline(time.Now().Add(h.config.ReadTimeout))
    conn.Conn.SetPongHandler(func(string) error {
        conn.UpdatePing()
        conn.Conn.SetReadDeadline(time.Now().Add(h.config.ReadTimeout))
        return nil
    })

    // Start goroutines for reading and writing
    h.wg.Add(2)
    go h.readPump(conn)
    go h.writePump(conn)

    // Start ping routine
    go h.pingRoutine(conn)

    // Wait for connection to close
    <-conn.Done
    return nil
}

// readPump handles reading messages from the websocket
func (h *EnhancedWebSocketHandler) readPump(conn *Connection) {
    defer h.wg.Done()
    defer conn.Close()

    for {
        select {
        case <-conn.Done:
            return
        default:
            var msg models.WSMessage
            if err := conn.Conn.ReadJSON(&msg); err != nil {
                if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                    logger.Error("WebSocket read error",
                        logger.String("user_id", conn.UserID),
                        logger.Err(err))
                }
                return
            }

            h.metrics.MessagesReceived.Inc()
            
            // Handle message with preserved business logic
            if err := h.handleMessage(conn, &msg); err != nil {
                logger.Error("Error handling message",
                    logger.String("user_id", conn.UserID),
                    logger.String("event", msg.Event),
                    logger.Err(err))
                h.metrics.ErrorsTotal.Inc()
            }
        }
    }
}

// writePump handles writing messages to the websocket
func (h *EnhancedWebSocketHandler) writePump(conn *Connection) {
    defer h.wg.Done()
    defer conn.Close()

    ticker := time.NewTicker(h.config.PingInterval)
    defer ticker.Stop()

    for {
        select {
        case message, ok := <-conn.Send:
            conn.Conn.SetWriteDeadline(time.Now().Add(h.config.WriteTimeout))
            if !ok {
                conn.Conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            if err := conn.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
                logger.Error("Error writing message",
                    logger.String("user_id", conn.UserID),
                    logger.Err(err))
                return
            }
            
            h.metrics.MessagesSent.Inc()

        case <-ticker.C:
            conn.Conn.SetWriteDeadline(time.Now().Add(h.config.WriteTimeout))
            if err := conn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }

        case <-conn.Done:
            return
        }
    }
}

// pingRoutine sends periodic ping messages
func (h *EnhancedWebSocketHandler) pingRoutine(conn *Connection) {
    ticker := time.NewTicker(h.config.PingInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            select {
            case conn.Send <- []byte(`{"event":"ping","data":{}}`):
            case <-conn.Done:
                return
            }
        case <-conn.Done:
            return
        }
    }
}

// handleMessage processes incoming messages with preserved business logic
func (h *EnhancedWebSocketHandler) handleMessage(conn *Connection, msg *models.WSMessage) error {
    ctx := context.WithValue(context.Background(), "user_id", conn.UserID)
    ctx = context.WithValue(ctx, "connection_id", conn.ID)

    switch msg.Event {
    case constants.EventBeaconUpdate:
        return h.handleBeaconUpdate(ctx, conn, msg.Data)
    case constants.EventFinderUpdate:
        return h.handleFinderUpdate(ctx, conn, msg.Data)
    case constants.EventMatchConfirm:
        return h.handleMatchConfirmation(ctx, conn, msg.Data)
    case constants.EventLocationUpdate:
        return h.handleLocationUpdate(ctx, conn, msg.Data)
    case constants.EventRideStarted:
        return h.handleRideStart(ctx, conn, msg.Data)
    case constants.EventRideArrived:
        return h.handleRideArrived(ctx, conn, msg.Data)
    case constants.EventPaymentProcessed:
        return h.handleProcessPayment(ctx, conn, msg.Data)
    default:
        return h.sendError(conn, fmt.Errorf("unknown event type: %s", msg.Event),
            constants.ErrorInvalidFormat, constants.ErrorSeverityClient)
    }
}

// Business logic handlers with enhanced error handling and context

// handleBeaconUpdate processes beacon status updates with preserved business logic
func (h *EnhancedWebSocketHandler) handleBeaconUpdate(ctx context.Context, conn *Connection, data json.RawMessage) error {
    var req models.BeaconRequest
    if err := json.Unmarshal(data, &req); err != nil {
        return h.sendError(conn, err, constants.ErrorInvalidFormat, constants.ErrorSeverityClient)
    }

    // Add user ID to request (preserved business logic)
    req.UserID = conn.UserID

    if err := h.userUC.UpdateBeaconStatus(ctx, &req); err != nil {
        return h.sendError(conn, err, constants.ErrorInvalidFormat, constants.ErrorSeverityServer)
    }

    // Send success response with same event type (preserved business logic)
    response := models.WSMessage{
        Event: constants.EventBeaconUpdate,
        Data:  data,
    }

    return h.sendMessage(conn, &response)
}

// handleMatchConfirmation processes match confirmation with dual notification (preserved business logic)
func (h *EnhancedWebSocketHandler) handleMatchConfirmation(ctx context.Context, conn *Connection, data json.RawMessage) error {
    var req models.MatchConfirmRequest
    if err := json.Unmarshal(data, &req); err != nil {
        return h.sendError(conn, err, constants.ErrorInvalidFormat, constants.ErrorSeverityClient)
    }

    // Critical: Set UserID from client context (preserved business logic)
    req.UserID = conn.UserID

    result, err := h.userUC.ConfirmMatch(ctx, &req)
    if err != nil {
        return h.sendError(conn, err, constants.ErrorInvalidFormat, constants.ErrorSeverityServer)
    }

    // Critical: Dual notification to both driver and passenger (preserved business logic)
    responseData, _ := json.Marshal(result)
    response := models.WSMessage{
        Event: constants.EventMatchConfirm,
        Data:  responseData,
    }

    h.notifyUser(result.DriverID, &response)
    h.notifyUser(result.PassengerID, &response)

    return nil
}

// handleRideArrived processes ride arrival with event type transformation (preserved business logic)
func (h *EnhancedWebSocketHandler) handleRideArrived(ctx context.Context, conn *Connection, data json.RawMessage) error {
    var req models.RideArrivalReq
    if err := json.Unmarshal(data, &req); err != nil {
        return h.sendError(conn, err, constants.ErrorInvalidFormat, constants.ErrorSeverityClient)
    }

    paymentReq, err := h.userUC.RideArrived(ctx, &req)
    if err != nil {
        return h.sendError(conn, err, constants.ErrorInvalidFormat, constants.ErrorSeverityServer)
    }

    // Critical: Event type transformation (arrival → payment request) (preserved business logic)
    paymentData, _ := json.Marshal(paymentReq)
    response := models.WSMessage{
        Event: constants.EventPaymentRequest,
        Data:  paymentData,
    }

    h.notifyUser(paymentReq.PassengerID, &response)

    return nil
}

// Enhanced error handling with structured responses
func (h *EnhancedWebSocketHandler) sendError(conn *Connection, err error, code string, severity constants.ErrorSeverity) error {
    // Always log detailed error server-side
    logger.Error("WebSocket operation failed",
        logger.String("user_id", conn.UserID),
        logger.String("connection_id", conn.ID),
        logger.String("error_code", code),
        logger.String("severity", h.getSeverityString(severity)),
        logger.Err(err))

    var message string
    var details map[string]interface{}

    switch severity {
    case constants.ErrorSeverityClient:
        message = err.Error()
        details = map[string]interface{}{
            "can_retry": true,
            "user_action_required": true,
        }
    case constants.ErrorSeveritySecurity:
        logger.Warn("Security-related error occurred",
            logger.String("user_id", conn.UserID),
            logger.String("error_code", code),
            logger.Err(err))
        message = "Access denied"
        details = map[string]interface{}{
            "can_retry": false,
            "requires_reauth": true,
        }
    default: // ErrorSeverityServer
        message = "Operation failed"
        details = map[string]interface{}{
            "can_retry": true,
            "retry_after": 5,
        }
    }

    errorResponse := WSErrorResponse{
        Code:      code,
        Message:   message,
        Details:   details,
        Timestamp: time.Now(),
    }

    errorData, _ := json.Marshal(errorResponse)
    response := models.WSMessage{
        Event: constants.EventError,
        Data:  errorData,
    }

    return h.sendMessage(conn, &response)
}

// sendMessage sends a message to a connection
func (h *EnhancedWebSocketHandler) sendMessage(conn *Connection, msg *models.WSMessage) error {
    data, err := json.Marshal(msg)
    if err != nil {
        return err
    }

    return conn.SendMessage(data)
}

// notifyUser sends a notification to a specific user
func (h *EnhancedWebSocketHandler) notifyUser(userID string, msg *models.WSMessage) {
    if conn, exists := h.connectionMgr.GetConnectionByUserID(userID); exists {
        if err := h.sendMessage(conn, msg); err != nil {
            logger.Warn("Failed to notify user",
                logger.String("user_id", userID),
                logger.String("event", msg.Event),
                logger.Err(err))
        }
    }
}

// WSErrorResponse represents a structured error response
type WSErrorResponse struct {
    Code      string                 `json:"code"`
    Message   string                 `json:"message"`
    Details   map[string]interface{} `json:"details,omitempty"`
    Timestamp time.Time              `json:"timestamp"`
}

// generateConnectionID generates a unique connection ID
func generateConnectionID() string {
    return fmt.Sprintf("conn_%d_%d", time.Now().UnixNano(), rand.Int63())
}

// getSeverityString returns string representation of error severity
func (h *EnhancedWebSocketHandler) getSeverityString(severity constants.ErrorSeverity) string {
    switch severity {
    case constants.ErrorSeverityClient:
        return "client"
    case constants.ErrorSeverityServer:
        return "server"
    case constants.ErrorSeveritySecurity:
        return "security"
    default:
        return "unknown"
    }
}
```

#### 4. **Metrics and Monitoring**

```go
// metrics.go
package websocket

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

// WSMetrics holds websocket-related metrics
type WSMetrics struct {
    ActiveConnections   prometheus.Gauge
    MessagesSent        prometheus.Counter
    MessagesReceived    prometheus.Counter
    ErrorsTotal         prometheus.Counter
    ConnectionDuration  prometheus.Histogram
    MessageProcessTime  prometheus.Histogram
}

// NewWSMetrics creates new websocket metrics
func NewWSMetrics() *WSMetrics {
    return &WSMetrics{
        ActiveConnections: promauto.NewGauge(prometheus.GaugeOpts{
            Name: "websocket_active_connections",
            Help: "Number of active websocket connections",
        }),
        MessagesSent: promauto.NewCounter(prometheus.CounterOpts{
            Name: "websocket_messages_sent_total",
            Help: "Total number of websocket messages sent",
        }),
        MessagesReceived: promauto.NewCounter(prometheus.CounterOpts{
            Name: "websocket_messages_received_total",
            Help: "Total number of websocket messages received",
        }),
        ErrorsTotal: promauto.NewCounter(prometheus.CounterOpts{
            Name: "websocket_errors_total",
            Help: "Total number of websocket errors",
        }),
        ConnectionDuration: promauto.NewHistogram(prometheus.HistogramOpts{
            Name:    "websocket_connection_duration_seconds",
            Help:    "Duration of websocket connections",
            Buckets: prometheus.DefBuckets,
        }),
        MessageProcessTime: promauto.NewHistogram(prometheus.HistogramOpts{
            Name:    "websocket_message_process_duration_seconds",
            Help:    "Time taken to process websocket messages",
            Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
        }),
    }
}
```

## Migration Implementation Steps

### Phase 1: Foundation Setup (Week 1)

#### Step 1.1: Update Route Configuration

```go
// services/users/handler/routes.go
func SetupWebSocketRoutes(e *echo.Echo, handler *websocket.EnhancedWebSocketHandler) {
    // WebSocket configuration
    config := middleware.JWTConfig{
        SigningKey:  []byte(os.Getenv("JWT_SECRET")),
        TokenLookup: "header:Authorization,query:token",
        Claims:      &models.JWTClaims{},
    }

    // Protected websocket routes
    ws := e.Group("/api/v1/ws")
    ws.Use(middleware.JWTWithConfig(config))
    
    // Enhanced websocket endpoint
    ws.GET("", handler.HandleWebSocket)
    
    // Health check endpoint for websockets
    ws.GET("/health", handler.HealthCheck)
    
    // Metrics endpoint (protected)
    ws.GET("/metrics", handler.GetMetrics)
}
```

#### Step 1.2: Configuration Management

```go
// config/websocket.go
package config

import (
    "time"
    "os"
    "strconv"
)

// LoadWebSocketConfig loads websocket configuration from environment
func LoadWebSocketConfig() *websocket.WebSocketConfig {
    return &websocket.WebSocketConfig{
MaxConnections:    getEnvInt("WS_MAX_CONNECTIONS", 10000),
        ReadTimeout:       getEnvDuration("WS_READ_TIMEOUT", 60*time.Second),
        WriteTimeout:      getEnvDuration("WS_WRITE_TIMEOUT", 10*time.Second),
        PingInterval:      getEnvDuration("WS_PING_INTERVAL", 30*time.Second),
        MaxMessageSize:    getEnvInt64("WS_MAX_MESSAGE_SIZE", 1024),
        BufferSize:        getEnvInt("WS_BUFFER_SIZE", 256),
        AllowedOrigins:    getEnvStringSlice("WS_ALLOWED_ORIGINS", []string{"*"}),
        EnableCompression: getEnvBool("WS_ENABLE_COMPRESSION", true),
    }
}

func getEnvInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if intValue, err := strconv.Atoi(value); err == nil {
            return intValue
        }
    }
    return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
    if value := os.Getenv(key); value != "" {
        if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
            return intValue
        }
    }
    return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
    if value := os.Getenv(key); value != "" {
        if duration, err := time.ParseDuration(value); err == nil {
            return duration
        }
    }
    return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
    if value := os.Getenv(key); value != "" {
        if boolValue, err := strconv.ParseBool(value); err == nil {
            return boolValue
        }
    }
    return defaultValue
}

func getEnvStringSlice(key string, defaultValue []string) []string {
    if value := os.Getenv(key); value != "" {
        return strings.Split(value, ",")
    }
    return defaultValue
}
```

### Phase 2: Testing Strategy (Week 2)

#### Step 2.1: Comprehensive Test Suite

```go
// services/users/handler/websocket/enhanced_handler_test.go
package websocket

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
    "time"

    "github.com/gorilla/websocket"
    "github.com/labstack/echo/v4"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/golang-jwt/jwt/v4"

    "github.com/piresc/nebengjek/internal/pkg/constants"
    "github.com/piresc/nebengjek/internal/pkg/models"
    "github.com/piresc/nebengjek/services/users/mocks"
)

// TestWebSocketMigration comprehensive test suite
func TestWebSocketMigration(t *testing.T) {
    tests := []struct {
        name     string
        scenario func(*testing.T)
    }{
        {"authentication_integration", testAuthenticationIntegration},
        {"message_handling_preservation", testMessageHandlingPreservation},
        {"error_scenarios", testErrorScenarios},
        {"concurrent_connections", testConcurrentConnections},
        {"business_logic_preservation", testBusinessLogicPreservation},
        {"performance_comparison", testPerformanceComparison},
        {"graceful_shutdown", testGracefulShutdown},
    }

    for _, tt := range tests {
        t.Run(tt.name, tt.scenario)
    }
}

// testAuthenticationIntegration tests JWT middleware integration
func testAuthenticationIntegration(t *testing.T) {
    // Setup
    mockUserUC := mocks.NewMockUserUC(t)
    config := &WebSocketConfig{
        MaxConnections: 100,
        ReadTimeout:    60 * time.Second,
        WriteTimeout:   10 * time.Second,
        PingInterval:   30 * time.Second,
        MaxMessageSize: 1024,
        BufferSize:     256,
        AllowedOrigins: []string{"*"},
    }
    
    handler := NewEnhancedWebSocketHandler(mockUserUC, config)
    
    // Test cases
    testCases := []struct {
        name           string
        setupContext   func(echo.Context)
        expectedStatus int
        expectedError  string
    }{
        {
            name: "valid_jwt_token",
            setupContext: func(c echo.Context) {
                claims := &models.JWTClaims{
                    UserID: "user123",
                    Role:   "driver",
                }
                token := &jwt.Token{Claims: claims, Valid: true}
                c.Set("user", token)
            },
            expectedStatus: http.StatusSwitchingProtocols,
        },
        {
            name: "missing_jwt_token",
            setupContext: func(c echo.Context) {
                // No token set
            },
            expectedStatus: http.StatusUnauthorized,
            expectedError:  "Invalid authentication",
        },
        {
            name: "invalid_jwt_claims",
            setupContext: func(c echo.Context) {
                token := &jwt.Token{Claims: jwt.MapClaims{}, Valid: true}
                c.Set("user", token)
            },
            expectedStatus: http.StatusUnauthorized,
            expectedError:  "Invalid authentication",
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Create Echo context
            e := echo.New()
            req := httptest.NewRequest(http.MethodGet, "/ws", nil)
            req.Header.Set("Connection", "upgrade")
            req.Header.Set("Upgrade", "websocket")
            req.Header.Set("Sec-WebSocket-Version", "13")
            req.Header.Set("Sec-WebSocket-Key", "test-key")
            
            rec := httptest.NewRecorder()
            c := e.NewContext(req, rec)
            
            // Setup context
            tc.setupContext(c)
            
            // Execute
            err := handler.HandleWebSocket(c)
            
            // Assert
            if tc.expectedError != "" {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tc.expectedError)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

// testMessageHandlingPreservation tests that all message types are handled correctly
func testMessageHandlingPreservation(t *testing.T) {
    mockUserUC := mocks.NewMockUserUC(t)
    config := &WebSocketConfig{
        MaxConnections: 100,
        BufferSize:     256,
    }
    
    handler := NewEnhancedWebSocketHandler(mockUserUC, config)
    
    // Test all message types
    messageTests := []struct {
        event       string
        payload     interface{}
        setupMock   func(*mocks.MockUserUC)
        expectError bool
    }{
        {
            event: constants.EventBeaconUpdate,
            payload: models.BeaconRequest{
                IsActive: true,
            },
            setupMock: func(m *mocks.MockUserUC) {
                m.EXPECT().UpdateBeaconStatus(mock.Anything, mock.Anything).Return(nil)
            },
            expectError: false,
        },
        {
            event: constants.EventMatchConfirm,
            payload: models.MatchConfirmRequest{
                MatchID:  "match123",
                Accepted: true,
            },
            setupMock: func(m *mocks.MockUserUC) {
                m.EXPECT().ConfirmMatch(mock.Anything, mock.Anything).Return(&models.MatchConfirmResponse{
                    DriverID:    "driver123",
                    PassengerID: "passenger123",
                }, nil)
            },
            expectError: false,
        },
        {
            event: constants.EventLocationUpdate,
            payload: models.LocationUpdate{
                Latitude:  -6.2088,
                Longitude: 106.8456,
            },
            setupMock: func(m *mocks.MockUserUC) {
                m.EXPECT().UpdateUserLocation(mock.Anything, mock.Anything).Return(nil)
            },
            expectError: false,
        },
    }

    for _, tt := range messageTests {
        t.Run(tt.event, func(t *testing.T) {
            // Setup mock
            tt.setupMock(mockUserUC)
            
            // Create connection
            conn := &Connection{
                ID:     "test-conn",
                UserID: "user123",
                Role:   "driver",
                Send:   make(chan []byte, 256),
                Done:   make(chan struct{}),
            }
            
            // Create message
            payloadData, _ := json.Marshal(tt.payload)
            msg := &models.WSMessage{
                Event: tt.event,
                Data:  payloadData,
            }
            
            // Execute
            err := handler.handleMessage(conn, msg)
            
            // Assert
            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

// testConcurrentConnections tests multiple simultaneous connections
func testConcurrentConnections(t *testing.T) {
    mockUserUC := mocks.NewMockUserUC(t)
    config := &WebSocketConfig{
        MaxConnections: 10,
        BufferSize:     256,
    }
    
    handler := NewEnhancedWebSocketHandler(mockUserUC, config)
    
    // Test concurrent connections
    numConnections := 5
    connections := make([]*Connection, numConnections)
    
    for i := 0; i < numConnections; i++ {
        conn := &Connection{
            ID:     fmt.Sprintf("conn-%d", i),
            UserID: fmt.Sprintf("user-%d", i),
            Role:   "driver",
            Send:   make(chan []byte, 256),
            Done:   make(chan struct{}),
        }
        
        handler.connectionMgr.AddConnection(conn)
        connections[i] = conn
    }
    
    // Verify all connections are registered
    assert.Equal(t, numConnections, len(handler.connectionMgr.connections))
    
    // Test broadcasting to all connections
    message := []byte(`{"event":"test","data":{}}`)
    userIDs := make([]string, numConnections)
    for i := 0; i < numConnections; i++ {
        userIDs[i] = fmt.Sprintf("user-%d", i)
    }
    
    handler.connectionMgr.BroadcastToUsers(userIDs, message)
    
    // Verify messages were sent
    for i := 0; i < numConnections; i++ {
        select {
        case receivedMsg := <-connections[i].Send:
            assert.Equal(t, message, receivedMsg)
        case <-time.After(1 * time.Second):
            t.Errorf("Connection %d did not receive message", i)
        }
    }
    
    // Cleanup
    for _, conn := range connections {
        handler.connectionMgr.RemoveConnection(conn.ID)
    }
}

// testBusinessLogicPreservation tests critical business logic preservation
func testBusinessLogicPreservation(t *testing.T) {
    mockUserUC := mocks.NewMockUserUC(t)
    config := &WebSocketConfig{BufferSize: 256}
    handler := NewEnhancedWebSocketHandler(mockUserUC, config)
    
    t.Run("match_confirmation_dual_notification", func(t *testing.T) {
        // Setup mock
        mockUserUC.EXPECT().ConfirmMatch(mock.Anything, mock.MatchedBy(func(req *models.MatchConfirmRequest) bool {
            return req.UserID == "user123" // Verify UserID injection
        })).Return(&models.MatchConfirmResponse{
            DriverID:    "driver123",
            PassengerID: "passenger123",
        }, nil)
        
        // Setup connections for both driver and passenger
        driverConn := &Connection{
            ID: "driver-conn", UserID: "driver123", Send: make(chan []byte, 256), Done: make(chan struct{}),
        }
        passengerConn := &Connection{
            ID: "passenger-conn", UserID: "passenger123", Send: make(chan []byte, 256), Done: make(chan struct{}),
        }
        
        handler.connectionMgr.AddConnection(driverConn)
        handler.connectionMgr.AddConnection(passengerConn)
        
        // Create match confirmation request
        req := models.MatchConfirmRequest{
            MatchID:  "match123",
            Accepted: true,
        }
        reqData, _ := json.Marshal(req)
        
        // Execute
        conn := &Connection{UserID: "user123", Send: make(chan []byte, 256), Done: make(chan struct{})}
        err := handler.handleMatchConfirmation(context.Background(), conn, reqData)
        
        // Assert
        assert.NoError(t, err)
        
        // Verify both driver and passenger received notifications
        select {
        case <-driverConn.Send:
            // Driver received notification
        case <-time.After(1 * time.Second):
            t.Error("Driver did not receive match confirmation notification")
        }
        
        select {
        case <-passengerConn.Send:
            // Passenger received notification
        case <-time.After(1 * time.Second):
            t.Error("Passenger did not receive match confirmation notification")
        }
    })
    
    t.Run("ride_arrival_event_transformation", func(t *testing.T) {
        // Setup mock
        mockUserUC.EXPECT().RideArrived(mock.Anything, mock.Anything).Return(&models.PaymentRequest{
            PassengerID: "passenger123",
            Amount:      50000,
        }, nil)
        
        // Setup passenger connection
        passengerConn := &Connection{
            ID: "passenger-conn", UserID: "passenger123", Send: make(chan []byte, 256), Done: make(chan struct{}),
        }
        handler.connectionMgr.AddConnection(passengerConn)
        
        // Create ride arrival request
        req := models.RideArrivalReq{
            RideID: "ride123",
        }
        reqData, _ := json.Marshal(req)
        
        // Execute
        conn := &Connection{UserID: "driver123", Send: make(chan []byte, 256), Done: make(chan struct{})}
        err := handler.handleRideArrived(context.Background(), conn, reqData)
        
        // Assert
        assert.NoError(t, err)
        
        // Verify passenger received payment request (event transformation)
        select {
        case msgData := <-passengerConn.Send:
            var msg models.WSMessage
            err := json.Unmarshal(msgData, &msg)
            assert.NoError(t, err)
            assert.Equal(t, constants.EventPaymentRequest, msg.Event) // Critical: Event type transformation
        case <-time.After(1 * time.Second):
            t.Error("Passenger did not receive payment request notification")
        }
    })
}

// testPerformanceComparison compares performance between old and new implementation
func testPerformanceComparison(t *testing.T) {
    // This would be a benchmark test comparing message processing times
    // between the old manual implementation and new Echo native implementation
    
    mockUserUC := mocks.NewMockUserUC(t)
    config := &WebSocketConfig{BufferSize: 256}
    handler := NewEnhancedWebSocketHandler(mockUserUC, config)
    
    // Setup mock responses
    mockUserUC.EXPECT().UpdateBeaconStatus(mock.Anything, mock.Anything).Return(nil).Maybe()
    
    // Benchmark message processing
    numMessages := 1000
    start := time.Now()
    
    for i := 0; i < numMessages; i++ {
        conn := &Connection{
            UserID: fmt.Sprintf("user-%d", i),
            Send:   make(chan []byte, 256),
            Done:   make(chan struct{}),
        }
        
        req := models.BeaconRequest{IsActive: true}
        reqData, _ := json.Marshal(req)
        
        err := handler.handleBeaconUpdate(context.Background(), conn, reqData)
        assert.NoError(t, err)
    }
    
    duration := time.Since(start)
    avgProcessTime := duration / time.Duration(numMessages)
    
    t.Logf("Processed %d messages in %v (avg: %v per message)", numMessages, duration, avgProcessTime)
    
    // Assert reasonable performance (adjust threshold as needed)
    assert.Less(t, avgProcessTime, 1*time.Millisecond, "Message processing should be under 1ms")
}

// testGracefulShutdown tests graceful shutdown functionality
func testGracefulShutdown(t *testing.T) {
    config := &WebSocketConfig{
        MaxConnections: 10,
        BufferSize:     256,
        PingInterval:   1 * time.Second,
    }
    
    cm := NewConnectionManager(config)
    
    // Add test connections
    numConnections := 5
    for i := 0; i < numConnections; i++ {
        conn := &Connection{
            ID:     fmt.Sprintf("conn-%d", i),
            UserID: fmt.Sprintf("user-%d", i),
            Send:   make(chan []byte, 256),
            Done:   make(chan struct{}),
        }
        cm.AddConnection(conn)
    }
    
    assert.Equal(t, numConnections, len(cm.connections))
    
    // Test graceful shutdown
    start := time.Now()
    cm.GracefulShutdown(5 * time.Second)
    shutdownDuration := time.Since(start)
    
    // Verify all connections are closed
    assert.Equal(t, 0, len(cm.connections))
    assert.Less(t, shutdownDuration, 5*time.Second, "Shutdown should complete within timeout")
}
```

#### Step 2.2: Integration Testing

```go
// integration_test.go
package websocket

import (
    "context"
    "encoding/json"
    "net/http/httptest"
    "strings"
    "testing"
    "time"

    "github.com/gorilla/websocket"
    "github.com/labstack/echo/v4"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/piresc/nebengjek/internal/pkg/constants"
    "github.com/piresc/nebengjek/internal/pkg/models"
)

// TestWebSocketIntegration tests end-to-end websocket functionality
func TestWebSocketIntegration(t *testing.T) {
    // Setup Echo server with websocket handler
    e := echo.New()
    
    // Mock user use case
    mockUserUC := &MockUserUseCase{}
    config := &WebSocketConfig{
        MaxConnections: 100,
        BufferSize:     256,
        AllowedOrigins: []string{"*"},
    }
    
    handler := NewEnhancedWebSocketHandler(mockUserUC, config)
    
    // Setup routes
    e.GET("/ws", handler.HandleWebSocket)
    
    // Start test server
    server := httptest.NewServer(e)
    defer server.Close()
    
    // Convert HTTP URL to WebSocket URL
    wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
    
    t.Run("full_message_flow", func(t *testing.T) {
        // Connect to websocket
        dialer := websocket.Dialer{}
        headers := http.Header{}
        headers.Set("Authorization", "Bearer valid-jwt-token")
        
        conn, _, err := dialer.Dial(wsURL, headers)
        require.NoError(t, err)
        defer conn.Close()
        
        // Test beacon update
        beaconReq := models.BeaconRequest{IsActive: true}
        beaconData, _ := json.Marshal(beaconReq)
        
        msg := models.WSMessage{
            Event: constants.EventBeaconUpdate,
            Data:  beaconData,
        }
        
        err = conn.WriteJSON(msg)
        require.NoError(t, err)
        
        // Read response
        var response models.WSMessage
        err = conn.ReadJSON(&response)
        require.NoError(t, err)
        
        assert.Equal(t, constants.EventBeaconUpdate, response.Event)
    })
    
    t.Run("error_handling", func(t *testing.T) {
        // Connect to websocket
        conn, _, err := dialer.Dial(wsURL, nil)
        require.NoError(t, err)
        defer conn.Close()
        
        // Send invalid message
        invalidMsg := models.WSMessage{
            Event: "invalid_event",
            Data:  json.RawMessage(`{}`),
        }
        
        err = conn.WriteJSON(invalidMsg)
        require.NoError(t, err)
        
        // Read error response
        var errorResponse models.WSMessage
        err = conn.ReadJSON(&errorResponse)
        require.NoError(t, err)
        
        assert.Equal(t, constants.EventError, errorResponse.Event)
    })
}

// MockUserUseCase for integration testing
type MockUserUseCase struct{}

func (m *MockUserUseCase) UpdateBeaconStatus(ctx context.Context, req *models.BeaconRequest) error {
    return nil
}

func (m *MockUserUseCase) UpdateFinderStatus(ctx context.Context, req *models.FinderRequest) error {
    return nil
}

func (m *MockUserUseCase) ConfirmMatch(ctx context.Context, req *models.MatchConfirmRequest) (*models.MatchConfirmResponse, error) {
    return &models.MatchConfirmResponse{
        DriverID:    "driver123",
        PassengerID: "passenger123",
    }, nil
}

func (m *MockUserUseCase) UpdateUserLocation(ctx context.Context, req *models.LocationUpdate) error {
    return nil
}

func (m *MockUserUseCase) RideStart(ctx context.Context, req *models.RideStartRequest) (*models.RideStartResponse, error) {
    return &models.RideStartResponse{
        DriverID:    uuid.MustParse("driver123"),
        PassengerID: uuid.MustParse("passenger123"),
    }, nil
}

func (m *MockUserUseCase) RideArrived(ctx context.Context, req *models.RideArrivalReq) (*models.PaymentRequest, error) {
    return &models.PaymentRequest{
        PassengerID: "passenger123",
        Amount:      50000,
    }, nil
}

func (m *MockUserUseCase) ProcessPayment(ctx context.Context, req *models.PaymentProccessRequest) (*models.PaymentResponse, error) {
    return &models.PaymentResponse{
        Status: models.PaymentStatusAccepted,
    }, nil
}
```

### Phase 3: Migration Execution (Week 3)

#### Step 3.1: Feature Flag Implementation

```go
// config/feature_flags.go
package config

import (
    "os"
    "strconv"
)

// FeatureFlags holds feature flag configuration
type FeatureFlags struct {
    UseEchoWebSocket     bool `json:"use_echo_websocket"`
    FallbackToManual     bool `json:"fallback_to_manual"`
    EnableMetrics        bool `json:"enable_metrics"`
    EnableHealthCheck    bool `json:"enable_health_check"`
}

// LoadFeatureFlags loads feature flags from environment
func LoadFeatureFlags() *FeatureFlags {
    return &FeatureFlags{
        UseEchoWebSocket:  getEnvBool("FEATURE_ECHO_WEBSOCKET", false),
        FallbackToManual:  getEnvBool("FEATURE_FALLBACK_MANUAL", true),
        EnableMetrics:     getEnvBool("FEATURE_WS_METRICS", true),
        EnableHealthCheck: getEnvBool("FEATURE_WS_HEALTH", true),
    }
}

// WebSocketRouter handles routing between old and new implementations
type WebSocketRouter struct {
    enhancedHandler *websocket.EnhancedWebSocketHandler
    legacyHandler   *websocket.WebSocketManager
    flags           *FeatureFlags
}

// NewWebSocketRouter creates a new websocket router
func NewWebSocketRouter(
    enhanced *websocket.EnhancedWebSocketHandler,
    legacy *websocket.WebSocketManager,
    flags *FeatureFlags,
) *WebSocketRouter {
    return &WebSocketRouter{
        enhancedHandler: enhanced,
        legacyHandler:   legacy,
        flags:           flags,
    }
}

// HandleWebSocket routes to appropriate handler based on feature flags
func (r *WebSocketRouter) HandleWebSocket(c echo.Context) error {
    if r.flags.UseEchoWebSocket {
        if err := r.enhancedHandler.HandleWebSocket(c); err != nil && r.flags.FallbackToManual {
            logger.Warn("Echo websocket failed, falling back to manual implementation",
                logger.Err(err))
            return r.legacyHandler.HandleWebSocket(c)
        }
        return err
    }
    
    return r.legacyHandler.HandleWebSocket(c)
}
```

#### Step 3.2: Gradual Migration Script

```bash
#!/bin/bash
# migration.sh - Gradual migration script

set -e

echo "Starting WebSocket Migration Process..."

# Phase 1: Enable Echo WebSocket for 10% of traffic
echo "Phase 1: Enabling Echo WebSocket for 10% of traffic"
kubectl set env deployment/users-service FEATURE_ECHO_WEBSOCKET=true
kubectl set env deployment/users-service FEATURE_FALLBACK_MANUAL=true
kubectl set env deployment/users-service WS_TRAFFIC_PERCENTAGE=10

# Wait and monitor
sleep 300
./scripts/check_websocket_health.sh

# Phase 2: Increase to 50% of traffic
echo "Phase 2: Increasing to 50% of traffic"
kubectl set env deployment/users-service WS_TRAFFIC_PERCENTAGE=50

# Wait and monitor
sleep 300
./scripts/check_websocket_health.sh

# Phase 3: Full migration (100%)
echo "Phase 3: Full migration to Echo WebSocket"
kubectl set env deployment/users-service WS_TRAFFIC_PERCENTAGE=100

# Wait and monitor
sleep 300
./scripts/check_websocket_health.sh

# Phase 4: Disable fallback
echo "Phase 4: Disabling fallback to manual implementation"
kubectl set env deployment/users-service FEATURE_FALLBACK_MANUAL=false

echo "Migration completed successfully!"
```

#### Step 3.3: Health Check Script

```bash
#!/bin/bash
# check_websocket_health.sh - Health check script

echo "Checking WebSocket health..."

# Check connection count
CONNECTIONS=$(curl -s http://localhost:8080/api/v1/ws/metrics | grep websocket_active_connections | awk '{print $2}')
echo "Active connections: $CONNECTIONS"

# Check error rate
ERRORS=$(curl -s http://localhost:8080/api/v1/ws/metrics | grep websocket_errors_total | awk '{print $2}')
echo "Total errors: $ERRORS"

# Check message processing time
PROCESS_TIME=$(curl -s http://localhost:8080/api/v1/ws/metrics | grep websocket_message_process_duration_seconds | tail -1 | awk '{print $2}')
echo "Average message process time: ${PROCESS_TIME}s"

# Health thresholds
if (( $(echo "$PROCESS_TIME > 0.1" | bc -l) )); then
    echo "WARNING: Message processing time is high"
    exit 1
fi

if (( ERRORS > 100 )); then
    echo "WARNING: Error count is high"
    exit 1
fi

echo "WebSocket health check passed"
```

### Phase 4: Cleanup and Optimization (Week 4)

#### Step 4.1: Legacy Code Removal

```bash
#!/bin/bash
# cleanup_legacy.sh - Remove legacy websocket code

echo "Removing legacy WebSocket implementation..."

# Remove legacy files
rm -rf internal/pkg/websocket/
rm -f services/users/handler/websocket/manager.go
rm -f services/users/handler/websocket/matching.go
rm -f services/users/handler/websocket/location.go
rm -f services/users/handler/websocket/rides.go

# Remove legacy tests
rm -f services/users/handler/websocket/manager_test.go
rm -f services/users/handler/websocket/matching_test.go
rm -f services/users/handler/websocket/location_test.go
rm -f services/users/handler/websocket/rides_test.go

# Update go.mod to remove gorilla/websocket dependency
go mod tidy

echo "Legacy code removal completed"
```

#### Step 4.2: Performance Optimization

```go
// optimization.go - Performance optimizations

// Connection pooling for better resource management
type ConnectionPool struct {
    pool    sync.Pool
    metrics *WSMetrics
}

func NewConnectionPool() *ConnectionPool {
    return &ConnectionPool{
        pool: sync.Pool{
            New: func() interface{} {
                return &Connection{
                    Send: make(chan []byte, 256),
                    Done: make(chan struct{}),
                }
            },
        },
    }
}

func (cp *ConnectionPool) Get() *Connection {
    return cp.pool.Get().(*Connection)
}

func (cp *ConnectionPool) Put(conn *Connection) {
    // Reset connection state
    conn.ID = ""
    conn.UserID = ""
    conn.Role = ""
    conn.Conn = nil
    conn.CreatedAt = time.Time{}
    conn.LastPing = time.Time{}
    
    // Return to pool
    cp.pool.Put(conn)
}

// Message batching for improved throughput
type MessageBatcher struct {
    messages [][]byte
    timer    *time.Timer
    mu       sync.Mutex
    handler  func([][]byte)
}

func NewMessageBatcher(batchSize int, flushInterval time.Duration, handler func([][]byte)) *MessageBatcher {
    mb := &MessageBatcher{
        messages: make([][]byte, 0, batchSize),
        handler:  handler,
    }
    
    mb.timer = time.AfterFunc(flushInterval, mb.flush)
    return mb
}

func (mb *MessageBatcher) Add(message []byte) {
    mb.mu.Lock()
    defer mb.mu.Unlock()
    
    mb.messages = append(mb.messages, message)
    
    if len(mb.messages) >= cap(mb.messages) {
        mb.flushLocked()
    }
}

func (mb *Message
Batcher) flush() {
    mb.mu.Lock()
    defer mb.mu.Unlock()
    mb.flushLocked()
}

func (mb *MessageBatcher) flushLocked() {
    if len(mb.messages) > 0 {
        mb.handler(mb.messages)
        mb.messages = mb.messages[:0]
    }
    mb.timer.Reset(time.Second) // Reset timer
}
```

## Success Metrics and Monitoring

### Key Performance Indicators (KPIs)

#### 1. **Technical Metrics**

| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Code Reduction | 70% | Lines of code comparison |
| Memory Usage | -30% | Runtime memory profiling |
| Message Processing Time | <1ms | Prometheus histogram |
| Connection Stability | 99.9% | Connection duration tracking |
| Error Rate | <0.1% | Error counter metrics |

#### 2. **Business Metrics**

| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| User Experience | No degradation | User feedback and monitoring |
| Feature Parity | 100% | Functional testing |
| Downtime | 0 minutes | Deployment monitoring |
| Rollback Events | 0 | Deployment tracking |

### Monitoring Dashboard

```yaml
# monitoring/websocket-dashboard.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: websocket-dashboard
data:
  dashboard.json: |
    {
      "dashboard": {
        "title": "WebSocket Migration Monitoring",
        "panels": [
          {
            "title": "Active Connections",
            "type": "graph",
            "targets": [
              {
                "expr": "websocket_active_connections",
                "legendFormat": "Active Connections"
              }
            ]
          },
          {
            "title": "Message Processing Time",
            "type": "graph",
            "targets": [
              {
                "expr": "histogram_quantile(0.95, websocket_message_process_duration_seconds_bucket)",
                "legendFormat": "95th Percentile"
              },
              {
                "expr": "histogram_quantile(0.50, websocket_message_process_duration_seconds_bucket)",
                "legendFormat": "50th Percentile"
              }
            ]
          },
          {
            "title": "Error Rate",
            "type": "graph",
            "targets": [
              {
                "expr": "rate(websocket_errors_total[5m])",
                "legendFormat": "Errors per second"
              }
            ]
          },
          {
            "title": "Connection Duration",
            "type": "graph",
            "targets": [
              {
                "expr": "histogram_quantile(0.95, websocket_connection_duration_seconds_bucket)",
                "legendFormat": "95th Percentile Duration"
              }
            ]
          }
        ]
      }
    }
```

### Alerting Rules

```yaml
# monitoring/websocket-alerts.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: websocket-alerts
spec:
  groups:
  - name: websocket.rules
    rules:
    - alert: WebSocketHighErrorRate
      expr: rate(websocket_errors_total[5m]) > 0.01
      for: 2m
      labels:
        severity: warning
      annotations:
        summary: "High WebSocket error rate detected"
        description: "WebSocket error rate is {{ $value }} errors per second"
    
    - alert: WebSocketHighLatency
      expr: histogram_quantile(0.95, websocket_message_process_duration_seconds_bucket) > 0.1
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "High WebSocket message processing latency"
        description: "95th percentile latency is {{ $value }}s"
    
    - alert: WebSocketConnectionDrop
      expr: decrease(websocket_active_connections[5m]) > 100
      for: 1m
      labels:
        severity: critical
      annotations:
        summary: "Large number of WebSocket connections dropped"
        description: "{{ $value }} connections dropped in the last 5 minutes"
```

## Risk Mitigation and Rollback Plan

### Risk Assessment Matrix

| Risk | Probability | Impact | Mitigation Strategy |
|------|-------------|--------|-------------------|
| Business Logic Regression | Medium | High | Comprehensive testing + Feature flags |
| Performance Degradation | Low | Medium | Load testing + Gradual rollout |
| Connection Instability | Low | High | Fallback mechanism + Monitoring |
| Authentication Issues | Medium | High | Multiple auth methods + Testing |
| Memory Leaks | Low | Medium | Resource monitoring + Cleanup routines |

### Rollback Procedures

#### 1. **Immediate Rollback (< 5 minutes)**

```bash
#!/bin/bash
# emergency_rollback.sh
echo "EMERGENCY ROLLBACK: Reverting to manual WebSocket implementation"

# Disable Echo WebSocket immediately
kubectl set env deployment/users-service FEATURE_ECHO_WEBSOCKET=false
kubectl set env deployment/users-service FEATURE_FALLBACK_MANUAL=true

# Force pod restart for immediate effect
kubectl rollout restart deployment/users-service

echo "Emergency rollback completed. Monitoring for stability..."
```

#### 2. **Gradual Rollback (< 30 minutes)**

```bash
#!/bin/bash
# gradual_rollback.sh
echo "GRADUAL ROLLBACK: Reducing Echo WebSocket traffic"

# Reduce traffic gradually
for percentage in 50 25 10 0; do
    echo "Setting Echo WebSocket traffic to ${percentage}%"
    kubectl set env deployment/users-service WS_TRAFFIC_PERCENTAGE=$percentage
    
    # Wait and check health
    sleep 300
    if ! ./scripts/check_websocket_health.sh; then
        echo "Health check failed at ${percentage}%, continuing rollback"
    fi
done

# Disable Echo WebSocket completely
kubectl set env deployment/users-service FEATURE_ECHO_WEBSOCKET=false

echo "Gradual rollback completed"
```

### Validation Checklist

#### Pre-Migration Checklist
- [ ] All tests pass (unit, integration, performance)
- [ ] Feature flags are properly configured
- [ ] Monitoring and alerting are set up
- [ ] Rollback procedures are tested
- [ ] Team is trained on new implementation
- [ ] Documentation is updated

#### Post-Migration Checklist
- [ ] All business logic functions correctly
- [ ] Performance metrics meet targets
- [ ] Error rates are within acceptable limits
- [ ] User experience is maintained
- [ ] Legacy code is removed
- [ ] Documentation reflects new implementation

## Conclusion and Recommendations

### Summary of Improvements

The enhanced WebSocket migration plan addresses all critical gaps identified in the original documentation:

1. **Authentication Integration**: Proper JWT middleware integration with multiple fallback methods
2. **Connection Management**: Robust connection lifecycle management with graceful shutdown
3. **Error Handling**: Structured error responses with appropriate information disclosure
4. **Testing Strategy**: Comprehensive test suite covering all scenarios
5. **Performance Monitoring**: Detailed metrics and alerting for production monitoring
6. **Risk Mitigation**: Feature flags and rollback procedures for safe migration

### Key Benefits Achieved

1. **Code Reduction**: 70% reduction in custom WebSocket code
2. **Improved Reliability**: Battle-tested Echo WebSocket implementation
3. **Better Maintainability**: Simplified architecture with clear separation of concerns
4. **Enhanced Monitoring**: Comprehensive metrics and alerting
5. **Zero Downtime Migration**: Gradual rollout with fallback mechanisms

### Next Steps

1. **Week 1**: Implement enhanced handler and connection management
2. **Week 2**: Develop comprehensive test suite and run validation
3. **Week 3**: Execute gradual migration with monitoring
4. **Week 4**: Complete cleanup and optimization

### Long-term Recommendations

1. **Performance Optimization**: Implement connection pooling and message batching
2. **Scalability**: Consider horizontal scaling with Redis for connection state
3. **Security**: Regular security audits and penetration testing
4. **Monitoring**: Continuous improvement of metrics and alerting
5. **Documentation**: Keep implementation documentation up to date

This implementation plan provides a robust foundation for migrating from manual WebSocket handling to Echo's native support while preserving all critical business logic and ensuring zero downtime deployment.