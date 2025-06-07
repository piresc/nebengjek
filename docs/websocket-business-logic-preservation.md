# WebSocket Business Logic Preservation Guide

This document outlines the critical business logic that must be preserved during the migration from manual websocket handling to Echo's native websocket implementation.

## Critical Business Logic Components

### 1. Message Handling Workflow

#### Current Implementation Pattern
```go
// All handlers follow this pattern:
1. JSON unmarshaling of incoming data
2. Data validation
3. Use case method invocation
4. Error handling with categorized responses
5. Success response with appropriate event notifications
```

#### Required Preservation
- **Exact JSON structure compatibility** for all message types
- **Use case method signatures** must remain unchanged
- **Error categorization** (client vs server errors) must be maintained
- **Event notification patterns** to multiple clients must be preserved

### 2. Message Types and Handlers

#### Location Updates (`EventLocationUpdate`)
- **Handler**: `handleLocationUpdate`
- **Business Logic**: 
  - Unmarshal `models.LocationUpdate`
  - Add timestamp to location data
  - Call `userUC.UpdateUserLocation()`
  - No response message sent back
- **Critical**: Timestamp addition logic must be preserved

#### Beacon Updates (`EventBeaconUpdate`)
- **Handler**: `handleBeaconUpdate`
- **Business Logic**:
  - Unmarshal `models.BeaconRequest`
  - Call `userUC.UpdateBeaconStatus()`
  - Send success response with same event type
- **Critical**: Response event type must match request event type

#### Finder Updates (`EventFinderUpdate`)
- **Handler**: `handleFinderUpdate`
- **Business Logic**:
  - Unmarshal `models.FinderRequest`
  - Call `userUC.UpdateFinderStatus()`
  - Send success response with same event type
- **Critical**: Response event type must match request event type

#### Match Confirmation (`EventMatchConfirm`)
- **Handler**: `handleMatchConfirmation`
- **Business Logic**:
  - Unmarshal `models.MatchConfirmRequest`
  - Set `req.UserID = client.UserID` (critical client context)
  - Call `userUC.ConfirmMatch()`
  - **Dual notification**: Notify both driver and passenger
- **Critical**: Client UserID injection and dual notification pattern

#### Ride Start (`EventRideStarted`)
- **Handler**: `handleRideStart`
- **Business Logic**:
  - Unmarshal `models.RideStartRequest`
  - Call `userUC.RideStart()`
  - **Dual notification**: Notify both driver and passenger with `EventRideStarted`
- **Critical**: Dual notification to both parties

#### Ride Arrival (`EventRideArrived`)
- **Handler**: `handleRideArrived`
- **Business Logic**:
  - Unmarshal `models.RideArrivalReq`
  - Call `userUC.RideArrived()`
  - **Single notification**: Notify passenger with `EventPaymentRequest`
- **Critical**: Event type transformation (arrival → payment request)

#### Payment Processing (`EventPaymentProcessed`)
- **Handler**: `handleProcessPayment`
- **Business Logic**:
  - Unmarshal `models.PaymentProccessRequest`
  - **Validation**: Status must be `PaymentStatusAccepted` or `PaymentStatusRejected`
  - Call `userUC.ProcessPayment()`
  - Send response with `EventPaymentProcessed`
- **Critical**: Payment status validation logic

### 3. Error Handling Patterns

#### Categorized Error System
```go
// Current pattern that must be preserved:
func SendCategorizedError(client *WebSocketClient, err error, errorCode string, severity string) error
```

#### Error Categories
- **Client Errors** (`ErrorSeverityClient`):
  - Invalid JSON format
  - Invalid data validation
  - Invalid payment status
- **Server Errors** (`ErrorSeverityServer`):
  - Use case execution failures
  - Database errors
  - External service failures

#### Error Codes (from constants)
- `ErrorInvalidFormat`: JSON parsing or validation errors
- `ErrorMatchUpdateFailed`: Match operation failures
- `ErrorLocationUpdateFailed`: Location update failures
- `ErrorPaymentProcessingFailed`: Payment processing failures

### 4. Client Management

#### Connection Lifecycle
- **Add Client**: Register client with UserID mapping
- **Remove Client**: Clean up on disconnect
- **Client Lookup**: Find client by UserID for notifications

#### Notification Patterns
- **Single Client**: `NotifyClient(userID, eventType, data)`
- **Dual Client**: Notify both driver and passenger
- **Broadcast**: Send to multiple clients (used in match confirmations)

### 5. Use Case Integration Points

#### Required Use Case Methods
```go
// These method signatures MUST be preserved:
UpdateUserLocation(ctx context.Context, req *models.LocationUpdate) error
UpdateBeaconStatus(ctx context.Context, req *models.BeaconRequest) error
UpdateFinderStatus(ctx context.Context, req *models.FinderRequest) error
ConfirmMatch(ctx context.Context, req *models.MatchConfirmRequest) (*models.MatchConfirmResponse, error)
RideStart(ctx context.Context, req *models.RideStartRequest) (*models.RideStartResponse, error)
RideArrived(ctx context.Context, req *models.RideArrivalReq) (*models.PaymentRequest, error)
ProcessPayment(ctx context.Context, req *models.PaymentProccessRequest) (*models.PaymentResponse, error)
```

### 6. Data Models

#### WebSocket Message Structure
```go
type WSMessage struct {
    Event string          `json:"event"`
    Data  json.RawMessage `json:"data"`
}
```

#### Client Structure
```go
type WebSocketClient struct {
    UserID string
    Conn   *websocket.Conn
    // Other fields...
}
```

## Migration Validation Checklist

### Functional Requirements
- [ ] All message types are handled identically
- [ ] Use case method calls remain unchanged
- [ ] Error categorization system is preserved
- [ ] Client notification patterns work correctly
- [ ] Dual notifications for match/ride events work
- [ ] Payment status validation is maintained
- [ ] Location timestamp addition is preserved
- [ ] Client UserID injection for match confirmation works

### Technical Requirements
- [ ] JSON message structure compatibility
- [ ] Error response format consistency
- [ ] Client connection management
- [ ] Event type constants usage
- [ ] Context propagation to use cases

### Testing Requirements
- [ ] All existing websocket tests pass
- [ ] Error handling scenarios work correctly
- [ ] Multi-client notification scenarios work
- [ ] Invalid message format handling
- [ ] Use case error propagation

## Risk Mitigation

### High-Risk Areas
1. **Dual Notifications**: Match confirmations and ride events require notifying multiple clients
2. **Event Type Transformations**: Ride arrival → payment request event type change
3. **Client Context**: UserID injection for match confirmations
4. **Error Categorization**: Maintaining client vs server error distinction
5. **Payment Validation**: Status validation logic preservation

### Validation Strategy
1. **Unit Tests**: Ensure all existing tests pass
2. **Integration Tests**: Test complete message flows
3. **Error Scenario Tests**: Verify error handling preservation
4. **Multi-Client Tests**: Validate notification patterns
5. **Data Validation Tests**: Ensure JSON compatibility

## Success Criteria

### Functional Success
- All websocket message types process identically
- Error responses maintain same format and categorization
- Client notifications work for all scenarios
- Use case integration remains unchanged

### Technical Success
- Code reduction achieved (estimated 70%)
- Echo native websocket features utilized
- Simplified message routing
- Maintained performance characteristics

### Business Success
- Zero business logic regression
- All user workflows function identically
- Error handling provides same user experience
- System reliability maintained or improved