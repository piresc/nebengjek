package gatewaywebsocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models/core"
	websocketmodels "github.com/piresc/nebengjek/internal/pkg/models/websocket"
	"github.com/piresc/nebengjek/internal/pkg/models/location"
	"github.com/piresc/nebengjek/internal/pkg/models/match"
	"github.com/piresc/nebengjek/internal/pkg/models/ride"
	"github.com/piresc/nebengjek/services/gateway"
	"github.com/piresc/nebengjek/services/users"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// WebSocketNotifier defines the interface for WebSocket notification functionality
type WebSocketNotifier interface {
	NotifyClientWithError(userID string, event string, data interface{}) error
}

// EchoWebSocketHandler handles websocket connections with heartbeat support
type EchoWebSocketHandler struct {
	gatewayUC       gateway.GatewayUC
	userUC          users.UserUC
	sessionRepo     websocketmodels.WSConnectionRegistry
	serverID        string
	clients         map[string]*websocket.Conn
	lastActivity    map[string]time.Time
	lastPing        map[string]time.Time      // Track last ping sent using conn.Ping()
	missedPings     map[string]int            // Count failed pings
	connectedSince  map[string]time.Time      // Connection start time
	heartbeatCancel map[string]context.CancelFunc // Cancel heartbeat goroutines
	mu              sync.RWMutex
}

// NewEchoWebSocketHandler creates a new WebSocket handler with heartbeat support
func NewEchoWebSocketHandler(gatewayUC gateway.GatewayUC, userUC users.UserUC, sessionRepo websocketmodels.WSConnectionRegistry, serverID string) *EchoWebSocketHandler {
	return &EchoWebSocketHandler{
		gatewayUC:       gatewayUC,
		userUC:          userUC,
		sessionRepo:     sessionRepo,
		serverID:        serverID,
		clients:         make(map[string]*websocket.Conn),
		lastActivity:    make(map[string]time.Time),
		lastPing:        make(map[string]time.Time),
		missedPings:     make(map[string]int),
		connectedSince:  make(map[string]time.Time),
		heartbeatCancel: make(map[string]context.CancelFunc),
	}
}

// HandleWebSocket handles websocket connections using github.com/coder/websocket
func (h *EchoWebSocketHandler) HandleWebSocket(c echo.Context) error {
	// Extract user info from JWT token (already validated by middleware)
	userIDRaw := c.Get("user_id")
	roleRaw := c.Get("role")

	if userIDRaw == nil || roleRaw == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Missing user credentials in token")
	}

	// Convert interface{} to string safely with UUID validation
	userID := fmt.Sprintf("%v", userIDRaw)
	role := fmt.Sprintf("%v", roleRaw)

	// Validate that userID is not empty (which would cause UUID parsing errors)
	if userID == "" || userID == "<nil>" || userID == "00000000-0000-0000-0000-000000000000" {
		logger.Error("Invalid or empty user ID from JWT token",
			logger.String("user_id_raw", fmt.Sprintf("%v", userIDRaw)),
			logger.String("role", role))
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid user credentials in token")
	}

	// Upgrade to WebSocket using coder/websocket with optimized settings for gateway service
	ws, err := websocket.Accept(c.Response(), c.Request(), &websocket.AcceptOptions{
		// Enable compression for mobile clients (reduces bandwidth)
		CompressionMode: websocket.CompressionContextTakeover,
		// Accept any origin for development (configure properly for production)
		InsecureSkipVerify: true,
	})
	if err != nil {
		logger.Error("Failed to upgrade WebSocket connection",
			logger.String("user_id", userID),
			logger.ErrorField(err))
		return echo.NewHTTPError(http.StatusBadRequest, "WebSocket upgrade failed")
	}
	defer ws.Close(websocket.StatusNormalClosure, "connection closed")

	// Register client with Redis session storage
	if err := h.addClient(userID, role, ws); err != nil {
		logger.Error("Failed to register WebSocket client in Redis",
			logger.String("user_id", userID),
			logger.String("role", role),
			logger.ErrorField(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to register client")
	}
	defer h.removeClient(userID)

	logger.Info("WebSocket client connected",
		logger.String("user_id", userID),
		logger.String("role", role),
		logger.String("server_id", h.serverID))

	// Create context for connection lifecycle management
	ctx := c.Request().Context()
	
	// Message handling loop with proper context handling
	for {
		var msg websocketmodels.WSMessage
		if err := wsjson.Read(ctx, ws, &msg); err != nil {
			// Check for normal closure
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
			   websocket.CloseStatus(err) == websocket.StatusGoingAway {
				logger.Info("WebSocket client disconnected normally",
					logger.String("user_id", userID))
				break
			}
			logger.Error("Error receiving websocket message",
				logger.String("user_id", userID),
				logger.ErrorField(err))
			break
		}

		if err := h.handleMessage(ctx, userID, role, ws, &msg); err != nil {
			logger.Error("Error handling message",
				logger.String("user_id", userID),
				logger.String("event", msg.Event),
				logger.ErrorField(err))
		}
	}

	return nil
}

// addClient safely adds a client with heartbeat monitoring and Redis session storage
func (h *EchoWebSocketHandler) addClient(userID string, role string, ws *websocket.Conn) error {
	h.mu.Lock()
	now := time.Now()
	h.clients[userID] = ws
	h.lastActivity[userID] = now
	h.lastPing[userID] = now
	h.missedPings[userID] = 0
	h.connectedSince[userID] = now
	h.mu.Unlock()
	
	// Create session data for Redis storage
	session := &websocketmodels.WSSessionData{
		UserID:       userID,
		Role:         role,
		ServerID:     h.serverID,
		ConnectedAt:  now,
		LastActivity: now,
		LastPing:     now,
		LastPong:     now,
		MissedPings:  0,
		IsHealthy:    true,
	}
	
	// Store session in Redis
	if err := h.sessionRepo.RegisterConnection(context.Background(), userID, session); err != nil {
		h.mu.Lock()
		delete(h.clients, userID)
		delete(h.lastActivity, userID)
		delete(h.lastPing, userID)
		delete(h.missedPings, userID)
		delete(h.connectedSince, userID)
		h.mu.Unlock()
		return fmt.Errorf("failed to register session in Redis: %w", err)
	}
	
	// Start heartbeat routine using coder/websocket's built-in ping
	h.startHeartbeat(userID, ws)
	
	return nil
}

// removeClient safely removes a client and stops heartbeat
func (h *EchoWebSocketHandler) removeClient(userID string) {
	h.mu.Lock()
	// Cancel heartbeat routine
	if cancel, exists := h.heartbeatCancel[userID]; exists {
		cancel()
		delete(h.heartbeatCancel, userID)
	}
	// Clean up all tracking data
	delete(h.clients, userID)
	delete(h.lastActivity, userID)
	delete(h.lastPing, userID)
	delete(h.missedPings, userID)
	delete(h.connectedSince, userID)
	h.mu.Unlock()
	
	// Remove from Redis session storage
	if err := h.sessionRepo.UnregisterConnection(context.Background(), userID); err != nil {
		logger.Error("Failed to unregister session from Redis",
			logger.String("user_id", userID),
			logger.String("server_id", h.serverID),
			logger.ErrorField(err))
	}
}

// IsUserConnected checks if a user is currently connected
func (h *EchoWebSocketHandler) IsUserConnected(userID string) bool {
	connected, err := h.sessionRepo.IsUserConnected(context.Background(), userID)
	if err != nil {
		logger.Error("Failed to check connection status in Redis",
			logger.String("user_id", userID),
			logger.String("server_id", h.serverID),
			logger.ErrorField(err))
		return false
	}
	return connected
}

// GetConnectedUsersCount returns the number of connected users
func (h *EchoWebSocketHandler) GetConnectedUsersCount() int {
	count, err := h.sessionRepo.GetConnectionCount(context.Background())
	if err != nil {
		logger.Error("Failed to get connection count from Redis",
			logger.String("server_id", h.serverID),
			logger.ErrorField(err))
		return 0
	}
	return count
}

// GetUserLastActivity returns the last activity time for a user
func (h *EchoWebSocketHandler) GetUserLastActivity(userID string) time.Time {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if lastActivity, exists := h.lastActivity[userID]; exists {
		return lastActivity
	}
	return time.Time{}
}

// NotifyClient sends a notification to a specific client
func (h *EchoWebSocketHandler) NotifyClient(userID string, event string, data interface{}) {
	h.mu.RLock()
	ws, exists := h.clients[userID]
	h.mu.RUnlock()

	if !exists {
		logger.Warn("Client not connected for notification delivery",
			logger.String("user_id", userID),
			logger.String("event", event))
		return
	}

	rawData, err := json.Marshal(data)
	if err != nil {
		logger.Error("Error marshaling notification data",
			logger.String("user_id", userID),
			logger.String("event", event),
			logger.ErrorField(err))
		return
	}

	response := websocketmodels.WSMessage{
		Event: event,
		Data:  rawData,
	}

	// Use context with timeout for notification delivery
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := wsjson.Write(timeoutCtx, ws, response); err != nil {
		logger.Warn("Error sending message to client",
			logger.String("user_id", userID),
			logger.String("event", event),
			logger.ErrorField(err))
	} else {
		logger.Info("WebSocket message sent successfully",
			logger.String("user_id", userID),
			logger.String("event", event))
	}
}

// NotifyClientWithError sends a notification to a specific client and returns an error if delivery fails
func (h *EchoWebSocketHandler) NotifyClientWithError(userID string, event string, data interface{}) error {
	h.mu.RLock()
	ws, exists := h.clients[userID]
	h.mu.RUnlock()

	if !exists {
		logger.Warn("Client not connected for notification delivery",
			logger.String("user_id", userID),
			logger.String("event", event))
		return fmt.Errorf("client %s not connected", userID)
	}

	rawData, err := json.Marshal(data)
	if err != nil {
		logger.Error("Error marshaling notification data",
			logger.String("user_id", userID),
			logger.String("event", event),
			logger.ErrorField(err))
		return fmt.Errorf("failed to marshal notification data: %w", err)
	}

	response := websocketmodels.WSMessage{
		Event: event,
		Data:  rawData,
	}

	// Use context with timeout for notification delivery with error handling
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := wsjson.Write(timeoutCtx, ws, response); err != nil {
		logger.Warn("Error sending message to client",
			logger.String("user_id", userID),
			logger.String("event", event),
			logger.ErrorField(err))
		return fmt.Errorf("failed to send WebSocket message: %w", err)
	}

	logger.Info("WebSocket message sent successfully",
		logger.String("user_id", userID),
		logger.String("event", event))
	return nil
}

// sendError sends an error message to the client
func (h *EchoWebSocketHandler) sendError(ctx context.Context, ws *websocket.Conn, userID string, err error, code string, severity websocketmodels.ErrorSeverity) {
	// Always log detailed error server-side
	logger.Error("WebSocket operation failed",
		logger.String("user_id", userID),
		logger.String("error_code", code),
		logger.String("severity", h.getSeverityString(severity)),
		logger.Err(err))

	var message string
	switch severity {
	case websocketmodels.ErrorSeverityClient:
		// Show detailed error to client for validation/input issues
		message = err.Error()
	case websocketmodels.ErrorSeveritySecurity:
		// Minimal info to client for security issues
		logger.Warn("Security-related error occurred",
			logger.String("user_id", userID),
			logger.String("error_code", code),
			logger.Err(err))
		message = "Access denied"
	default: // ErrorSeverityServer
		// Extract clean error message from service responses
		message = h.extractCleanErrorMessage(err.Error())
	}

	// Create properly formatted error response
	errorData := map[string]string{
		"code":     code,
		"message":  message,
		"severity": h.getSeverityString(severity),
	}

	errorDataBytes, _ := json.Marshal(errorData)
	errorResponse := websocketmodels.WSMessage{
		Event: websocketmodels.EventError,
		Data:  json.RawMessage(errorDataBytes),
	}

	if err := wsjson.Write(ctx, ws, errorResponse); err != nil {
		logger.Error("Failed to send error message",
			logger.String("user_id", userID),
			logger.ErrorField(err))
	}
}

// getSeverityString returns string representation of error severity
func (h *EchoWebSocketHandler) getSeverityString(severity websocketmodels.ErrorSeverity) string {
	switch severity {
	case websocketmodels.ErrorSeverityClient:
		return "client"
	case websocketmodels.ErrorSeverityServer:
		return "server"
	case websocketmodels.ErrorSeveritySecurity:
		return "security"
	default:
		return "unknown"
	}
}

// extractCleanErrorMessage extracts clean error message from service responses
func (h *EchoWebSocketHandler) extractCleanErrorMessage(errorStr string) string {
	// Handle service error responses like: 'rides service returned error: {"success":false,"error":"actual message","code":500}'
	if strings.Contains(errorStr, "service returned error:") {
		// Try to extract JSON part
		jsonStart := strings.Index(errorStr, "{")
		if jsonStart > 0 {
			jsonPart := errorStr[jsonStart:]
			// Remove trailing newline
			jsonPart = strings.TrimSpace(jsonPart)

			// Try to parse JSON and extract error field
			var errorResponse struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal([]byte(jsonPart), &errorResponse); err == nil && errorResponse.Error != "" {
				return errorResponse.Error
			}
		}
	}

	// Fallback to original error message
	return errorStr
}

// handleMessage processes incoming WebSocket messages with preserved business logic
func (h *EchoWebSocketHandler) handleMessage(ctx context.Context, userID, role string, ws *websocket.Conn, msg *websocketmodels.WSMessage) error {
	switch msg.Event {
	case websocketmodels.EventBeaconUpdate:
		return h.handleBeaconUpdate(ctx, userID, ws, msg.Data)
	case websocketmodels.EventFinderUpdate:
		return h.handleFinderUpdate(ctx, userID, ws, msg.Data)
	case websocketmodels.EventMatchConfirm:
		return h.handleMatchConfirmation(ctx, userID, ws, msg.Data)
	case websocketmodels.EventLocationUpdate:
		return h.handleLocationUpdate(ctx, userID, ws, msg.Data)
	case websocketmodels.EventRideStarted:
		return h.handleRideStart(ctx, userID, ws, msg.Data)
	case websocketmodels.EventRideArrived:
		return h.handleRideArrived(ctx, userID, ws, msg.Data)
	case websocketmodels.EventPaymentProcessed:
		return h.handleProcessPayment(ctx, userID, ws, msg.Data)
	default:
		unknownEventErr := fmt.Errorf("unknown event type: %s", msg.Event)
		h.sendError(ctx, ws, userID, unknownEventErr, websocketmodels.ErrorInvalidFormat, websocketmodels.ErrorSeverityClient)
		return nil // Don't break connection for unknown events
	}
}

// Business logic handlers - preserving exact same logic as original implementation

// handleBeaconUpdate processes beacon status updates
func (h *EchoWebSocketHandler) handleBeaconUpdate(ctx context.Context, userID string, ws *websocket.Conn, data json.RawMessage) error {
	var req core.BeaconRequest
	if err := json.Unmarshal(data, &req); err != nil {
		h.sendError(ctx, ws, userID, err, websocketmodels.ErrorInvalidFormat, websocketmodels.ErrorSeverityClient)
		return nil
	}

	// Use ProxyToUsersService directly to get the actual response from the microservice
	resp, err := h.gatewayUC.ProxyToUsersService(
		ctx,
		"POST",
		"/beacon/update",
		req,
		nil,
		nil,
	)
	if err != nil {
		h.sendError(ctx, ws, userID, err, websocketmodels.ErrorInvalidFormat, websocketmodels.ErrorSeverityServer)
		return nil
	}

	if resp.StatusCode >= 400 {
		h.sendError(ctx, ws, userID, fmt.Errorf("users service returned error: %s", string(resp.Body)), websocketmodels.ErrorInvalidFormat, websocketmodels.ErrorSeverityServer)
		return nil
	}

	// Send success response with proper API response format
	successData := map[string]interface{}{
		"success": true,
		"message": "Beacon status updated successfully",
		"status":  resp.StatusCode,
	}
	
	successDataBytes, _ := json.Marshal(successData)
	response := websocketmodels.WSMessage{
		Event: websocketmodels.EventBeaconUpdate,
		Data:  json.RawMessage(successDataBytes),
	}

	return wsjson.Write(ctx, ws, response)
}

// handleFinderUpdate processes finder status updates
func (h *EchoWebSocketHandler) handleFinderUpdate(ctx context.Context, userID string, ws *websocket.Conn, data json.RawMessage) error {
	var req core.FinderRequest
	if err := json.Unmarshal(data, &req); err != nil {
		h.sendError(ctx, ws, userID, err, websocketmodels.ErrorInvalidFormat, websocketmodels.ErrorSeverityClient)
		return nil
	}

	// Use ProxyToUsersService directly to get the actual response from the microservice
	resp, err := h.gatewayUC.ProxyToUsersService(
		ctx,
		"POST",
		"/finder/update",
		req,
		nil,
		nil,
	)
	if err != nil {
		h.sendError(ctx, ws, userID, err, websocketmodels.ErrorInvalidFormat, websocketmodels.ErrorSeverityServer)
		return nil
	}

	if resp.StatusCode >= 400 {
		h.sendError(ctx, ws, userID, fmt.Errorf("users service returned error: %s", string(resp.Body)), websocketmodels.ErrorInvalidFormat, websocketmodels.ErrorSeverityServer)
		return nil
	}

	// Send success response with proper API response format
	successData := map[string]interface{}{
		"success": true,
		"message": "Finder status updated successfully",
		"status":  resp.StatusCode,
	}
	
	successDataBytes, _ := json.Marshal(successData)
	response := websocketmodels.WSMessage{
		Event: websocketmodels.EventFinderUpdate,
		Data:  json.RawMessage(successDataBytes),
	}

	return wsjson.Write(ctx, ws, response)
}

// handleMatchConfirmation processes match confirmation with dual notification
func (h *EchoWebSocketHandler) handleMatchConfirmation(ctx context.Context, userID string, ws *websocket.Conn, data json.RawMessage) error {
	var req match.MatchConfirmRequest
	if err := json.Unmarshal(data, &req); err != nil {
		h.sendError(ctx, ws, userID, err, websocketmodels.ErrorInvalidFormat, websocketmodels.ErrorSeverityClient)
		return nil
	}

	// Critical: Set UserID from client context
	req.UserID = userID

	result, err := h.gatewayUC.ConfirmMatch(ctx, &req)
	if err != nil {
		h.sendError(ctx, ws, userID, err, websocketmodels.ErrorInvalidFormat, websocketmodels.ErrorSeverityServer)
		return nil
	}

	// Critical: Dual notification to both driver and passenger
	h.NotifyClient(result.DriverID, websocketmodels.EventMatchConfirm, result)
	h.NotifyClient(result.PassengerID, websocketmodels.EventMatchConfirm, result)

	// Send success response to the requesting client
	successData := map[string]interface{}{
		"success": true,
		"message": "Match confirmation processed successfully",
		"match_id": result.ID,
	}
	
	successDataBytes, _ := json.Marshal(successData)
	response := websocketmodels.WSMessage{
		Event: websocketmodels.EventMatchConfirm,
		Data:  json.RawMessage(successDataBytes),
	}

	return wsjson.Write(ctx, ws, response)
}

// handleLocationUpdate processes location updates with timestamp addition
func (h *EchoWebSocketHandler) handleLocationUpdate(ctx context.Context, userID string, ws *websocket.Conn, data json.RawMessage) error {
	var req location.LocationUpdate
	if err := json.Unmarshal(data, &req); err != nil {
		h.sendError(ctx, ws, userID, err, websocketmodels.ErrorInvalidFormat, websocketmodels.ErrorSeverityClient)
		return nil
	}
	req.DriverID = userID // Ensure DriverID is set from client context

	// Critical: Add timestamp to location data (preserved business logic)
	// This logic is handled inside the use case, but we ensure the call is made

	if err := h.gatewayUC.UpdateUserLocation(ctx, &req); err != nil {
		h.sendError(ctx, ws, userID, err, websocketmodels.ErrorInvalidFormat, websocketmodels.ErrorSeverityServer)
		return nil
	}

	// Send success response for location update
	successData := map[string]interface{}{
		"success": true,
		"message": "Location updated successfully",
		"created_at": req.CreatedAt,
	}
	
	successDataBytes, _ := json.Marshal(successData)
	response := websocketmodels.WSMessage{
		Event: websocketmodels.EventLocationUpdate,
		Data:  json.RawMessage(successDataBytes),
	}

	return wsjson.Write(ctx, ws, response)
}

// handleRideStart processes ride start with dual notification
func (h *EchoWebSocketHandler) handleRideStart(ctx context.Context, userID string, ws *websocket.Conn, data json.RawMessage) error {
	var req ride.RideStartRequest
	if err := json.Unmarshal(data, &req); err != nil {
		h.sendError(ctx, ws, userID, err, websocketmodels.ErrorInvalidFormat, websocketmodels.ErrorSeverityClient)
		return nil
	}

	resp, err := h.gatewayUC.RideStart(ctx, &req)
	if err != nil {
		h.sendError(ctx, ws, userID, err, websocketmodels.ErrorInvalidFormat, websocketmodels.ErrorSeverityServer)
		return nil
	}

	// Critical: Dual notification to both driver and passenger
	h.NotifyClient(resp.DriverID.String(), websocketmodels.EventRideStarted, resp)
	h.NotifyClient(resp.PassengerID.String(), websocketmodels.EventRideStarted, resp)

	// Send success response to the requesting client
	successData := map[string]interface{}{
		"success": true,
		"message": "Ride started successfully",
		"ride_id": resp.RideID,
	}
	
	successDataBytes, _ := json.Marshal(successData)
	response := websocketmodels.WSMessage{
		Event: websocketmodels.EventRideStarted,
		Data:  json.RawMessage(successDataBytes),
	}

	return wsjson.Write(ctx, ws, response)
}

// handleRideArrived processes ride arrival with event type transformation
func (h *EchoWebSocketHandler) handleRideArrived(ctx context.Context, userID string, ws *websocket.Conn, data json.RawMessage) error {
	var req ride.RideArrivalReq
	if err := json.Unmarshal(data, &req); err != nil {
		h.sendError(ctx, ws, userID, err, websocketmodels.ErrorInvalidFormat, websocketmodels.ErrorSeverityClient)
		return nil
	}

	paymentReq, err := h.gatewayUC.RideArrived(ctx, &req)
	if err != nil {
		h.sendError(ctx, ws, userID, err, websocketmodels.ErrorInvalidFormat, websocketmodels.ErrorSeverityServer)
		return nil
	}

	// First: Send ride_arrived notification to BOTH driver and passenger
	rideArrivedData := map[string]interface{}{
		"ride_id":           req.RideID,
		"adjustment_factor": req.AdjustmentFactor,
	}
	h.NotifyClient(paymentReq.PassengerID, websocketmodels.EventRideArrived, rideArrivedData)
	h.NotifyClient(paymentReq.DriverID, websocketmodels.EventRideArrived, rideArrivedData)

	// Then: Send payment request to passenger only (payment is passenger's responsibility)
	h.NotifyClient(paymentReq.PassengerID, websocketmodels.EventPaymentRequest, paymentReq)

	// Send success response to the requesting client
	successData := map[string]interface{}{
		"success": true,
		"message": "Ride arrival processed successfully",
		"ride_id": req.RideID,
	}
	
	successDataBytes, _ := json.Marshal(successData)
	response := websocketmodels.WSMessage{
		Event: websocketmodels.EventRideArrived,
		Data:  json.RawMessage(successDataBytes),
	}

	return wsjson.Write(ctx, ws, response)
}

// handleProcessPayment processes payment with status validation
func (h *EchoWebSocketHandler) handleProcessPayment(ctx context.Context, userID string, ws *websocket.Conn, data json.RawMessage) error {
	var req ride.PaymentProccessRequest
	if err := json.Unmarshal(data, &req); err != nil {
		h.sendError(ctx, ws, userID, err, websocketmodels.ErrorInvalidFormat, websocketmodels.ErrorSeverityClient)
		return nil
	}

	// Critical: Payment status validation
	if req.Status != ride.PaymentStatusAccepted && req.Status != ride.PaymentStatusRejected {
		validationErr := fmt.Errorf("invalid payment status: %s", req.Status)
		h.sendError(ctx, ws, userID, validationErr, websocketmodels.ErrorInvalidFormat, websocketmodels.ErrorSeverityClient)
		return nil
	}

	_, err := h.gatewayUC.ProcessPayment(ctx, &req)
	if err != nil {
		h.sendError(ctx, ws, userID, err, websocketmodels.ErrorInvalidFormat, websocketmodels.ErrorSeverityServer)
		return nil
	}

	// Send success response to the requesting client
	successData := map[string]interface{}{
		"success": true,
		"message": "Payment processed successfully",
		"status": req.Status,
	}
	
	successDataBytes, _ := json.Marshal(successData)
	response := websocketmodels.WSMessage{
		Event: websocketmodels.EventPaymentProcessed,
		Data:  json.RawMessage(successDataBytes),
	}

	// Payment processing successful - NATS event will also handle separate notifications
	// The notification service will send payment_processed events via NATS to both users
	return wsjson.Write(ctx, ws, response)
}

// Heartbeat implementation using coder/websocket built-in capabilities

// startHeartbeat starts heartbeat monitoring using coder/websocket's built-in ping
func (h *EchoWebSocketHandler) startHeartbeat(userID string, ws *websocket.Conn) {
	ctx, cancel := context.WithCancel(context.Background())
	
	h.mu.Lock()
	h.heartbeatCancel[userID] = cancel
	h.mu.Unlock()

	go func() {
		ticker := time.NewTicker(websocketmodels.HeartbeatInterval)
		defer ticker.Stop()
		sequence := int64(0)

		for {
			select {
			case <-ctx.Done():
				logger.Debug("Heartbeat routine stopped",
					logger.String("user_id", userID))
				return
			case <-ticker.C:
				sequence++
				if err := h.sendPing(userID, ws, sequence); err != nil {
					logger.Error("Ping failed, connection may be dead",
						logger.String("user_id", userID),
						logger.ErrorField(err))
					
					h.mu.Lock()
					h.missedPings[userID]++
					missedCount := h.missedPings[userID]
					h.mu.Unlock()

					if missedCount >= websocketmodels.MaxMissedPings {
						logger.Warn("Connection appears dead, closing",
							logger.String("user_id", userID),
							logger.Int("missed_pings", missedCount))
						
						// Close connection and clean up
						ws.Close(websocket.StatusInternalError, "heartbeat timeout")
						h.removeClient(userID)
						return
					}
				} else {
					// Reset missed pings on successful ping (coder/websocket handles pong automatically)
					h.mu.Lock()
					h.missedPings[userID] = 0
					h.mu.Unlock()
				}
			}
		}
	}()
}

// sendPing sends a ping using coder/websocket's built-in ping method
func (h *EchoWebSocketHandler) sendPing(userID string, ws *websocket.Conn, sequence int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use coder/websocket's built-in ping method (pong is handled automatically)
	if err := ws.Ping(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	// Update ping tracking
	h.mu.Lock()
	h.lastPing[userID] = time.Now()
	h.mu.Unlock()

	logger.Debug("Ping sent successfully",
		logger.String("user_id", userID),
		logger.Int64("sequence", sequence))
	
	return nil
}

// checkConnectionHealth evaluates connection health using heartbeat data
func (h *EchoWebSocketHandler) checkConnectionHealth(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	lastPing, exists := h.lastPing[userID]
	if !exists {
		return false
	}

	missedCount := h.missedPings[userID]
	timeSinceLastPing := time.Since(lastPing)

	// Connection is unhealthy if:
	// 1. Too many missed pings, OR
	// 2. No ping response within timeout period
	isUnhealthy := missedCount >= websocketmodels.MaxMissedPings ||
				   timeSinceLastPing > websocketmodels.HeartbeatTimeout

	return !isUnhealthy
}

// Health monitoring endpoints for heartbeat status

// GetConnectionHealth returns detailed health status for a specific user
func (h *EchoWebSocketHandler) GetConnectionHealth(userID string) *websocketmodels.WSConnectionHealth {
	h.mu.RLock()
	defer h.mu.RUnlock()

	health := &websocketmodels.WSConnectionHealth{
		UserID:    userID,
		IsHealthy: false,
	}

	if lastActivity, exists := h.lastActivity[userID]; exists {
		health.LastActivity = lastActivity
		health.ConnectedSince = h.connectedSince[userID]
		health.LastPing = h.lastPing[userID]
		health.MissedPings = h.missedPings[userID]
		health.IsHealthy = h.checkConnectionHealth(userID)
	}

	return health
}

// GetAllConnectionsHealth returns health status for all connected users
func (h *EchoWebSocketHandler) GetAllConnectionsHealth() map[string]*websocketmodels.WSConnectionHealth {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make(map[string]*websocketmodels.WSConnectionHealth)
	for userID := range h.clients {
		result[userID] = &websocketmodels.WSConnectionHealth{
			UserID:         userID,
			LastActivity:   h.lastActivity[userID],
			LastPing:       h.lastPing[userID],
			MissedPings:    h.missedPings[userID],
			ConnectedSince: h.connectedSince[userID],
			IsHealthy:      h.checkConnectionHealth(userID),
		}
	}

	return result
}

// GetHealthyConnectionsCount returns count of healthy connections
func (h *EchoWebSocketHandler) GetHealthyConnectionsCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	healthyCount := 0
	for userID := range h.clients {
		if h.checkConnectionHealth(userID) {
			healthyCount++
		}
	}

	return healthyCount
}

// GetHeartbeatStats returns overall heartbeat statistics
func (h *EchoWebSocketHandler) GetHeartbeatStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	totalConnections := len(h.clients)
	healthyConnections := 0
	totalMissedPings := 0

	for userID := range h.clients {
		if h.checkConnectionHealth(userID) {
			healthyConnections++
		}
		totalMissedPings += h.missedPings[userID]
	}

	return map[string]interface{}{
		"total_connections":   totalConnections,
		"healthy_connections": healthyConnections,
		"unhealthy_connections": totalConnections - healthyConnections,
		"total_missed_pings":  totalMissedPings,
		"heartbeat_interval":  websocketmodels.HeartbeatInterval.String(),
		"heartbeat_timeout":   websocketmodels.HeartbeatTimeout.String(),
		"max_missed_pings":    websocketmodels.MaxMissedPings,
	}
}
