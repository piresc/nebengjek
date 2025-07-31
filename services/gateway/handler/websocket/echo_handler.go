package gatewaywebsocket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/gateway"
	"github.com/piresc/nebengjek/services/users"
	"golang.org/x/net/websocket"
)

// WebSocketNotifier defines the interface for WebSocket notification functionality
type WebSocketNotifier interface {
	NotifyClientWithError(userID string, event string, data interface{}) error
}

// EchoWebSocketHandler handles websocket connections using Echo's native support
type EchoWebSocketHandler struct {
	gatewayUC    gateway.GatewayUC
	userUC       users.UserUC
	clients      map[string]*websocket.Conn
	lastActivity map[string]time.Time
	mu           sync.RWMutex
}

// NewEchoWebSocketHandler creates a new Echo-based websocket handler
func NewEchoWebSocketHandler(gatewayUC gateway.GatewayUC, userUC users.UserUC) *EchoWebSocketHandler {
	return &EchoWebSocketHandler{
		gatewayUC:    gatewayUC,
		userUC:       userUC,
		clients:      make(map[string]*websocket.Conn),
		lastActivity: make(map[string]time.Time),
	}
}

// HandleWebSocket handles websocket connections using Echo's native websocket support
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

	// Create WebSocket server with proper configuration
	wsServer := &websocket.Server{
		Handler: func(ws *websocket.Conn) {
			defer ws.Close()

			// Register client
			h.addClient(userID, ws)
			defer h.removeClient(userID)

			logger.Info("WebSocket client connected",
				logger.String("user_id", userID),
				logger.String("role", role))

			// Message handling loop
			for {
				var msg models.WSMessage
				if err := websocket.JSON.Receive(ws, &msg); err != nil {
					if err == io.EOF {
						logger.Info("WebSocket client disconnected",
							logger.String("user_id", userID))
						break
					}
					logger.Error("Error receiving websocket message",
						logger.String("user_id", userID),
						logger.ErrorField(err))
					break
				}

				if err := h.handleMessage(userID, role, ws, &msg); err != nil {
					logger.Error("Error handling message",
						logger.String("user_id", userID),
						logger.String("event", msg.Event),
						logger.ErrorField(err))
				}
			}
		},
		// Configure WebSocket to accept any origin to avoid CORS issues
		Handshake: func(config *websocket.Config, req *http.Request) error {
			config.Origin = config.Location
			return nil
		},
	}

	wsServer.ServeHTTP(c.Response(), c.Request())
	return nil
}

// addClient safely adds a client to the manager
func (h *EchoWebSocketHandler) addClient(userID string, ws *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[userID] = ws
	h.lastActivity[userID] = time.Now()
}

// removeClient safely removes a client from the manager
func (h *EchoWebSocketHandler) removeClient(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, userID)
	delete(h.lastActivity, userID)
}

// IsUserConnected checks if a user is currently connected
func (h *EchoWebSocketHandler) IsUserConnected(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, exists := h.clients[userID]
	return exists
}

// GetConnectedUsersCount returns the number of connected users
func (h *EchoWebSocketHandler) GetConnectedUsersCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
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

	response := models.WSMessage{
		Event: event,
		Data:  rawData,
	}

	if err := websocket.JSON.Send(ws, response); err != nil {
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

	response := models.WSMessage{
		Event: event,
		Data:  rawData,
	}

	if err := websocket.JSON.Send(ws, response); err != nil {
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
func (h *EchoWebSocketHandler) sendError(ws *websocket.Conn, userID string, err error, code string, severity constants.ErrorSeverity) {
	// Always log detailed error server-side
	logger.Error("WebSocket operation failed",
		logger.String("user_id", userID),
		logger.String("error_code", code),
		logger.String("severity", h.getSeverityString(severity)),
		logger.Err(err))

	var message string
	switch severity {
	case constants.ErrorSeverityClient:
		// Show detailed error to client for validation/input issues
		message = err.Error()
	case constants.ErrorSeveritySecurity:
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
	errorResponse := models.WSMessage{
		Event: constants.EventError,
		Data:  json.RawMessage(errorDataBytes),
	}

	if err := websocket.JSON.Send(ws, errorResponse); err != nil {
		logger.Error("Failed to send error message",
			logger.String("user_id", userID),
			logger.ErrorField(err))
	}
}

// getSeverityString returns string representation of error severity
func (h *EchoWebSocketHandler) getSeverityString(severity constants.ErrorSeverity) string {
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
func (h *EchoWebSocketHandler) handleMessage(userID, role string, ws *websocket.Conn, msg *models.WSMessage) error {
	switch msg.Event {
	case constants.EventBeaconUpdate:
		return h.handleBeaconUpdate(userID, ws, msg.Data)
	case constants.EventFinderUpdate:
		return h.handleFinderUpdate(userID, ws, msg.Data)
	case constants.EventMatchConfirm:
		return h.handleMatchConfirmation(userID, ws, msg.Data)
	case constants.EventLocationUpdate:
		return h.handleLocationUpdate(userID, ws, msg.Data)
	case constants.EventRideStarted:
		return h.handleRideStart(userID, ws, msg.Data)
	case constants.EventRideArrived:
		return h.handleRideArrived(userID, ws, msg.Data)
	case constants.EventPaymentProcessed:
		return h.handleProcessPayment(userID, ws, msg.Data)
	default:
		unknownEventErr := fmt.Errorf("unknown event type: %s", msg.Event)
		h.sendError(ws, userID, unknownEventErr, constants.ErrorInvalidFormat, constants.ErrorSeverityClient)
		return nil // Don't break connection for unknown events
	}
}

// Business logic handlers - preserving exact same logic as original implementation

// handleBeaconUpdate processes beacon status updates
func (h *EchoWebSocketHandler) handleBeaconUpdate(userID string, ws *websocket.Conn, data json.RawMessage) error {
	var req models.BeaconRequest
	if err := json.Unmarshal(data, &req); err != nil {
		h.sendError(ws, userID, err, constants.ErrorInvalidFormat, constants.ErrorSeverityClient)
		return nil
	}

	// Use ProxyToUsersService directly to get the actual response from the microservice
	resp, err := h.gatewayUC.ProxyToUsersService(
		context.Background(),
		"POST",
		"/beacon/update",
		req,
		nil,
		nil,
	)
	if err != nil {
		h.sendError(ws, userID, err, constants.ErrorInvalidFormat, constants.ErrorSeverityServer)
		return nil
	}

	if resp.StatusCode >= 400 {
		h.sendError(ws, userID, fmt.Errorf("users service returned error: %s", string(resp.Body)), constants.ErrorInvalidFormat, constants.ErrorSeverityServer)
		return nil
	}

	// Send success response with proper API response format
	successData := map[string]interface{}{
		"success": true,
		"message": "Beacon status updated successfully",
		"status":  resp.StatusCode,
	}
	
	successDataBytes, _ := json.Marshal(successData)
	response := models.WSMessage{
		Event: constants.EventBeaconUpdate,
		Data:  json.RawMessage(successDataBytes),
	}

	return websocket.JSON.Send(ws, response)
}

// handleFinderUpdate processes finder status updates
func (h *EchoWebSocketHandler) handleFinderUpdate(userID string, ws *websocket.Conn, data json.RawMessage) error {
	var req models.FinderRequest
	if err := json.Unmarshal(data, &req); err != nil {
		h.sendError(ws, userID, err, constants.ErrorInvalidFormat, constants.ErrorSeverityClient)
		return nil
	}

	// Use ProxyToUsersService directly to get the actual response from the microservice
	resp, err := h.gatewayUC.ProxyToUsersService(
		context.Background(),
		"POST",
		"/finder/update",
		req,
		nil,
		nil,
	)
	if err != nil {
		h.sendError(ws, userID, err, constants.ErrorInvalidFormat, constants.ErrorSeverityServer)
		return nil
	}

	if resp.StatusCode >= 400 {
		h.sendError(ws, userID, fmt.Errorf("users service returned error: %s", string(resp.Body)), constants.ErrorInvalidFormat, constants.ErrorSeverityServer)
		return nil
	}

	// Send success response with proper API response format
	successData := map[string]interface{}{
		"success": true,
		"message": "Finder status updated successfully",
		"status":  resp.StatusCode,
	}
	
	successDataBytes, _ := json.Marshal(successData)
	response := models.WSMessage{
		Event: constants.EventFinderUpdate,
		Data:  json.RawMessage(successDataBytes),
	}

	return websocket.JSON.Send(ws, response)
}

// handleMatchConfirmation processes match confirmation with dual notification
func (h *EchoWebSocketHandler) handleMatchConfirmation(userID string, ws *websocket.Conn, data json.RawMessage) error {
	var req models.MatchConfirmRequest
	if err := json.Unmarshal(data, &req); err != nil {
		h.sendError(ws, userID, err, constants.ErrorInvalidFormat, constants.ErrorSeverityClient)
		return nil
	}

	// Critical: Set UserID from client context
	req.UserID = userID

	result, err := h.gatewayUC.ConfirmMatch(context.Background(), &req)
	if err != nil {
		h.sendError(ws, userID, err, constants.ErrorInvalidFormat, constants.ErrorSeverityServer)
		return nil
	}

	// Critical: Dual notification to both driver and passenger
	h.NotifyClient(result.DriverID, constants.EventMatchConfirm, result)
	h.NotifyClient(result.PassengerID, constants.EventMatchConfirm, result)

	// Send success response to the requesting client
	successData := map[string]interface{}{
		"success": true,
		"message": "Match confirmation processed successfully",
		"match_id": result.ID,
	}
	
	successDataBytes, _ := json.Marshal(successData)
	response := models.WSMessage{
		Event: constants.EventMatchConfirm,
		Data:  json.RawMessage(successDataBytes),
	}

	return websocket.JSON.Send(ws, response)
}

// handleLocationUpdate processes location updates with timestamp addition
func (h *EchoWebSocketHandler) handleLocationUpdate(userID string, ws *websocket.Conn, data json.RawMessage) error {
	var req models.LocationUpdate
	if err := json.Unmarshal(data, &req); err != nil {
		h.sendError(ws, userID, err, constants.ErrorInvalidFormat, constants.ErrorSeverityClient)
		return nil
	}
	req.DriverID = userID // Ensure DriverID is set from client context

	// Critical: Add timestamp to location data (preserved business logic)
	// This logic is handled inside the use case, but we ensure the call is made

	if err := h.gatewayUC.UpdateUserLocation(context.Background(), &req); err != nil {
		h.sendError(ws, userID, err, constants.ErrorInvalidFormat, constants.ErrorSeverityServer)
		return nil
	}

	// Send success response for location update
	successData := map[string]interface{}{
		"success": true,
		"message": "Location updated successfully",
		"created_at": req.CreatedAt,
	}
	
	successDataBytes, _ := json.Marshal(successData)
	response := models.WSMessage{
		Event: constants.EventLocationUpdate,
		Data:  json.RawMessage(successDataBytes),
	}

	return websocket.JSON.Send(ws, response)
}

// handleRideStart processes ride start with dual notification
func (h *EchoWebSocketHandler) handleRideStart(userID string, ws *websocket.Conn, data json.RawMessage) error {
	var req models.RideStartRequest
	if err := json.Unmarshal(data, &req); err != nil {
		h.sendError(ws, userID, err, constants.ErrorInvalidFormat, constants.ErrorSeverityClient)
		return nil
	}

	resp, err := h.gatewayUC.RideStart(context.Background(), &req)
	if err != nil {
		h.sendError(ws, userID, err, constants.ErrorInvalidFormat, constants.ErrorSeverityServer)
		return nil
	}

	// Critical: Dual notification to both driver and passenger
	h.NotifyClient(resp.DriverID.String(), constants.EventRideStarted, resp)
	h.NotifyClient(resp.PassengerID.String(), constants.EventRideStarted, resp)

	// Send success response to the requesting client
	successData := map[string]interface{}{
		"success": true,
		"message": "Ride started successfully",
		"ride_id": resp.RideID,
	}
	
	successDataBytes, _ := json.Marshal(successData)
	response := models.WSMessage{
		Event: constants.EventRideStarted,
		Data:  json.RawMessage(successDataBytes),
	}

	return websocket.JSON.Send(ws, response)
}

// handleRideArrived processes ride arrival with event type transformation
func (h *EchoWebSocketHandler) handleRideArrived(userID string, ws *websocket.Conn, data json.RawMessage) error {
	var req models.RideArrivalReq
	if err := json.Unmarshal(data, &req); err != nil {
		h.sendError(ws, userID, err, constants.ErrorInvalidFormat, constants.ErrorSeverityClient)
		return nil
	}

	paymentReq, err := h.gatewayUC.RideArrived(context.Background(), &req)
	if err != nil {
		h.sendError(ws, userID, err, constants.ErrorInvalidFormat, constants.ErrorSeverityServer)
		return nil
	}

	// First: Send ride_arrived notification to BOTH driver and passenger
	rideArrivedData := map[string]interface{}{
		"ride_id":           req.RideID,
		"adjustment_factor": req.AdjustmentFactor,
	}
	h.NotifyClient(paymentReq.PassengerID, constants.EventRideArrived, rideArrivedData)
	h.NotifyClient(paymentReq.DriverID, constants.EventRideArrived, rideArrivedData)

	// Then: Send payment request to passenger only (payment is passenger's responsibility)
	h.NotifyClient(paymentReq.PassengerID, constants.EventPaymentRequest, paymentReq)

	// Send success response to the requesting client
	successData := map[string]interface{}{
		"success": true,
		"message": "Ride arrival processed successfully",
		"ride_id": req.RideID,
	}
	
	successDataBytes, _ := json.Marshal(successData)
	response := models.WSMessage{
		Event: constants.EventRideArrived,
		Data:  json.RawMessage(successDataBytes),
	}

	return websocket.JSON.Send(ws, response)
}

// handleProcessPayment processes payment with status validation
func (h *EchoWebSocketHandler) handleProcessPayment(userID string, ws *websocket.Conn, data json.RawMessage) error {
	var req models.PaymentProccessRequest
	if err := json.Unmarshal(data, &req); err != nil {
		h.sendError(ws, userID, err, constants.ErrorInvalidFormat, constants.ErrorSeverityClient)
		return nil
	}

	// Critical: Payment status validation
	if req.Status != models.PaymentStatusAccepted && req.Status != models.PaymentStatusRejected {
		validationErr := fmt.Errorf("invalid payment status: %s", req.Status)
		h.sendError(ws, userID, validationErr, constants.ErrorInvalidFormat, constants.ErrorSeverityClient)
		return nil
	}

	_, err := h.gatewayUC.ProcessPayment(context.Background(), &req)
	if err != nil {
		h.sendError(ws, userID, err, constants.ErrorInvalidFormat, constants.ErrorSeverityServer)
		return nil
	}

	// Send success response to the requesting client
	successData := map[string]interface{}{
		"success": true,
		"message": "Payment processed successfully",
		"status": req.Status,
	}
	
	successDataBytes, _ := json.Marshal(successData)
	response := models.WSMessage{
		Event: constants.EventPaymentProcessed,
		Data:  json.RawMessage(successDataBytes),
	}

	// Payment processing successful - NATS event will also handle separate notifications
	// The notification service will send payment_processed events via NATS to both users
	return websocket.JSON.Send(ws, response)
}
