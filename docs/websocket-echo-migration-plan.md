# WebSocket Migration to Echo Native Support

## Overview

This document outlines the migration plan from the current manual websocket implementation to Echo's native websocket capabilities. The current implementation uses a custom websocket manager with gorilla/websocket, which adds unnecessary complexity and maintenance overhead.

## Current Implementation Analysis

### Current Architecture
- **Location**: `internal/pkg/websocket/` and `services/users/handler/websocket/`
- **Pattern**: "Upgrade and Delegate" - Echo handles HTTP upgrade, then delegates to custom manager
- **Dependencies**: `gorilla/websocket`, custom authentication, manual message routing
- **Complexity**: ~500+ lines of custom websocket management code

### Issues with Current Implementation
1. **Manual Connection Management**: Custom client tracking and cleanup
2. **Complex Message Routing**: Manual JSON unmarshalling and type switching
3. **Error Handling Duplication**: Custom error categorization and response formatting
4. **Authentication Complexity**: Manual JWT validation in websocket context
5. **Testing Challenges**: Complex mocking requirements for websocket connections

## Migration Strategy

### Implementation Plan

#### Direct Replacement Strategy
Since the code is not in production, we can directly replace the current implementation without complex migration strategies.

#### Echo Native Implementation

#### 2.1 Route Definition with Echo Native WebSocket
```go
// services/users/handler/routes.go
func SetupRoutes(e *echo.Echo, h *Handler) {
    // Protected websocket routes with JWT middleware
    protected := e.Group("/api/v1")
    protected.Use(middleware.JWTWithConfig(middleware.JWTConfig{
        SigningKey: []byte(config.GetJWTSecret()),
    }))
    
    // Echo native websocket endpoint
    protected.GET("/ws", h.HandleWebSocketEcho)
}
```

#### 2.2 Simplified Handler Implementation
```go
// services/users/handler/websocket/echo_handler.go
package websocket

import (
    "github.com/labstack/echo/v4"
    "golang.org/x/net/websocket"
)

func (h *Handler) HandleWebSocketEcho(c echo.Context) error {
    websocket.Handler(func(ws *websocket.Conn) {
        defer ws.Close()
        
        // Get user from JWT context (already validated by middleware)
        user := c.Get("user").(*jwt.Token)
        claims := user.Claims.(*models.JWTClaims)
        
        // Register client with simplified manager
        client := &Client{
            ID:     claims.UserID,
            Conn:   ws,
            Send:   make(chan []byte, 256),
        }
        
        h.manager.Register(client)
        defer h.manager.Unregister(client)
        
        // Handle messages with Echo's built-in support
        for {
            var msg models.WebSocketMessage
            if err := websocket.JSON.Receive(ws, &msg); err != nil {
                break
            }
            
            h.handleMessage(client, &msg)
        }
    }).ServeHTTP(c.Response(), c.Request())
    
    return nil
}
```

#### 2.3 Simplified Message Handling
```go
// Leverage Echo's JSON binding and validation
func (h *Handler) handleMessage(client *Client, msg *models.WebSocketMessage) {
    switch msg.Type {
    case "finder_update":
        var payload models.FinderUpdate
        if err := json.Unmarshal(msg.Data, &payload); err != nil {
            client.SendError("Invalid finder update format")
            return
        }
        h.usecase.HandleFinderUpdate(client.ID, &payload)
        
    case "beacon_update":
        var payload models.BeaconUpdate
        if err := json.Unmarshal(msg.Data, &payload); err != nil {
            client.SendError("Invalid beacon update format")
            return
        }
        h.usecase.HandleBeaconUpdate(client.ID, &payload)
        
    case "match_confirmation":
        var payload models.MatchConfirmation
        if err := json.Unmarshal(msg.Data, &payload); err != nil {
            client.SendError("Invalid match confirmation format")
            return
        }
        h.usecase.HandleMatchConfirmation(client.ID, &payload)
    }
}
```

### Phase 3: Simplified Client Management

#### 3.1 Streamlined Manager
```go
// internal/pkg/websocket/echo_manager.go
type EchoManager struct {
    clients    map[string]*Client
    register   chan *Client
    unregister chan *Client
    broadcast  chan []byte
    mu         sync.RWMutex
}

func (m *EchoManager) Register(client *Client) {
    m.register <- client
}

func (m *EchoManager) Unregister(client *Client) {
    m.unregister <- client
}

func (m *EchoManager) Run() {
    for {
        select {
        case client := <-m.register:
            m.mu.Lock()
            m.clients[client.ID] = client
            m.mu.Unlock()
            
        case client := <-m.unregister:
            m.mu.Lock()
            if _, ok := m.clients[client.ID]; ok {
                delete(m.clients, client.ID)
                close(client.Send)
            }
            m.mu.Unlock()
            
        case message := <-m.broadcast:
            m.mu.RLock()
            for _, client := range m.clients {
                select {
                case client.Send <- message:
                default:
                    close(client.Send)
                    delete(m.clients, client.ID)
                }
            }
            m.mu.RUnlock()
        }
    }
}
```

### Phase 4: Testing Improvements

#### 4.1 Echo Test Helpers
```go
// services/users/handler/websocket/echo_handler_test.go
func TestWebSocketEcho(t *testing.T) {
    e := echo.New()
    req := httptest.NewRequest(http.MethodGet, "/ws", nil)
    req.Header.Set(echo.HeaderUpgrade, "websocket")
    req.Header.Set(echo.HeaderConnection, "upgrade")
    
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)
    
    // Test with Echo's built-in test utilities
    handler := &Handler{manager: NewEchoManager()}
    err := handler.HandleWebSocketEcho(c)
    
    assert.NoError(t, err)
}
```

## Benefits of Migration

### 1. Reduced Complexity
- **Before**: ~500+ lines of custom websocket code
- **After**: ~150 lines leveraging Echo's built-in support
- **Reduction**: 70% less custom code to maintain

### 2. Better Integration
- Native Echo middleware support
- Consistent error handling with REST endpoints
- Unified logging and monitoring
- Built-in request/response helpers

### 3. Improved Reliability
- Echo's battle-tested websocket implementation
- Automatic connection cleanup
- Better error recovery
- Standardized message formats

### 4. Enhanced Developer Experience
- Familiar Echo patterns
- Better debugging tools
- Simplified testing
- Consistent API design

## Code Removal Plan

### Files to Remove After Migration

#### 1. Custom WebSocket Manager
```
/internal/pkg/websocket/
├── manager.go              # Remove: Custom connection management
├── client.go               # Remove: Custom client struct
├── message_types.go        # Remove: Custom message handling
└── auth.go                 # Remove: Custom authentication
```

#### 2. Service-Level WebSocket Handlers
```
/services/users/handler/websocket/
├── manager.go              # Remove: Service-specific manager
├── matching.go             # Remove: Manual message routing
├── client.go               # Remove: Custom client management
└── auth.go                 # Remove: Duplicate authentication
```

#### 3. Test Files
```
/services/users/handler/websocket/
├── manager_test.go         # Remove: Tests for custom manager
├── matching_test.go        # Remove: Tests for manual routing
└── client_test.go          # Remove: Tests for custom client
```

### Dependencies to Remove
```go
// go.mod - Remove after migration
"github.com/gorilla/websocket" // No longer needed with Echo native
```

### Configuration Cleanup
```yaml
# Remove websocket-specific configuration
websocket:
  read_buffer_size: 1024     # Remove: Echo handles internally
  write_buffer_size: 1024    # Remove: Echo handles internally
  check_origin: true         # Remove: Use Echo middleware
```

## Implementation Steps

### Step 1: Remove Current Implementation
- [ ] Delete custom websocket manager files
- [ ] Remove gorilla/websocket dependency
- [ ] Clean up custom authentication code

### Step 2: Implement Echo Native WebSocket
- [ ] Create new Echo websocket handlers
- [ ] Implement simplified message routing
- [ ] Update routes to use Echo native support

### Step 3: Update Tests
- [ ] Replace custom websocket tests
- [ ] Use Echo's testing utilities
- [ ] Verify all functionality works correctly

## Success Metrics

### Code Quality
- [ ] 70% reduction in websocket-related code
- [ ] 50% improvement in test coverage
- [ ] Elimination of custom websocket bugs

### Performance
- [ ] Maintain or improve connection handling speed
- [ ] Reduce memory usage by 30%
- [ ] Improve error recovery time

### Developer Experience
- [ ] Faster development of new websocket features
- [ ] Simplified debugging process
- [ ] Consistent patterns with REST API

## Conclusion

Migrating to Echo's native websocket support will significantly simplify the codebase while improving reliability and maintainability. The migration plan ensures a smooth transition with minimal risk and maximum benefit.

The key advantages include:
- **Simplified Architecture**: Leverage Echo's proven websocket implementation
- **Reduced Maintenance**: 70% less custom code to maintain
- **Better Integration**: Consistent patterns with existing Echo-based REST API
- **Improved Testing**: Use Echo's built-in testing utilities
- **Enhanced Reliability**: Battle-tested websocket handling

This migration aligns with the principle of using framework capabilities effectively rather than reinventing the wheel with custom implementations.