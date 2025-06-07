package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/users"
	"golang.org/x/net/websocket"
)

// EchoWebSocketHandler handles websocket connections using Echo's native support
type EchoWebSocketHandler struct {
	userUC  users.UserUC
	clients map[string]*websocket.Conn
	mu      sync.RWMutex
}

// NewEchoWebSocketHandler creates a new Echo-based websocket handler
func NewEchoWebSocketHandler(userUC users.UserUC) *EchoWebSocketHandler {
	return &EchoWebSocketHandler{
		userUC:  userUC,
		clients: make(map[string]*websocket.Conn),
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
}

// removeClient safely removes a client from the manager
func (h *EchoWebSocketHandler) removeClient(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, userID)
}

// NotifyClient sends a notification to a specific client
func (h *EchoWebSocketHandler) NotifyClient(userID string, event string, data interface{}) {
	h.mu.RLock()
	ws, exists := h.clients[userID]
	h.mu.RUnlock()

	if !exists {
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
	}
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
		// Generic message for server errors
		message = "Operation failed"
	}

	errorResponse := models.WSMessage{
		Event: constants.EventError,
		Data:  json.RawMessage(fmt.Sprintf(`{"code":"%s","message":"%s"}`, code, message)),
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

	if err := h.userUC.UpdateBeaconStatus(context.Background(), &req); err != nil {
		h.sendError(ws, userID, err, constants.ErrorInvalidFormat, constants.ErrorSeverityServer)
		return nil
	}

	// Send success response with same event type
	response := models.WSMessage{
		Event: constants.EventBeaconUpdate,
		Data:  data, // Echo back the same data
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

	if err := h.userUC.UpdateFinderStatus(context.Background(), &req); err != nil {
		h.sendError(ws, userID, err, constants.ErrorInvalidFormat, constants.ErrorSeverityServer)
		return nil
	}

	// Send success response with same event type
	response := models.WSMessage{
		Event: constants.EventFinderUpdate,
		Data:  data, // Echo back the same data
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

	result, err := h.userUC.ConfirmMatch(context.Background(), &req)
	if err != nil {
		h.sendError(ws, userID, err, constants.ErrorInvalidFormat, constants.ErrorSeverityServer)
		return nil
	}

	// Critical: Dual notification to both driver and passenger
	h.NotifyClient(result.DriverID, constants.EventMatchConfirm, result)
	h.NotifyClient(result.PassengerID, constants.EventMatchConfirm, result)

	return nil
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

	if err := h.userUC.UpdateUserLocation(context.Background(), &req); err != nil {
		h.sendError(ws, userID, err, constants.ErrorInvalidFormat, constants.ErrorSeverityServer)
		return nil
	}

	// No response message sent back for location updates
	return nil
}

// handleRideStart processes ride start with dual notification
func (h *EchoWebSocketHandler) handleRideStart(userID string, ws *websocket.Conn, data json.RawMessage) error {
	var req models.RideStartRequest
	if err := json.Unmarshal(data, &req); err != nil {
		h.sendError(ws, userID, err, constants.ErrorInvalidFormat, constants.ErrorSeverityClient)
		return nil
	}

	resp, err := h.userUC.RideStart(context.Background(), &req)
	if err != nil {
		h.sendError(ws, userID, err, constants.ErrorInvalidFormat, constants.ErrorSeverityServer)
		return nil
	}

	// Critical: Dual notification to both driver and passenger
	h.NotifyClient(resp.DriverID.String(), constants.EventRideStarted, resp)
	h.NotifyClient(resp.PassengerID.String(), constants.EventRideStarted, resp)

	return nil
}

// handleRideArrived processes ride arrival with event type transformation
func (h *EchoWebSocketHandler) handleRideArrived(userID string, ws *websocket.Conn, data json.RawMessage) error {
	var req models.RideArrivalReq
	if err := json.Unmarshal(data, &req); err != nil {
		h.sendError(ws, userID, err, constants.ErrorInvalidFormat, constants.ErrorSeverityClient)
		return nil
	}

	paymentReq, err := h.userUC.RideArrived(context.Background(), &req)
	if err != nil {
		h.sendError(ws, userID, err, constants.ErrorInvalidFormat, constants.ErrorSeverityServer)
		return nil
	}

	// Critical: Event type transformation (arrival → payment request)
	h.NotifyClient(paymentReq.PassengerID, constants.EventPaymentRequest, paymentReq)

	return nil
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

	payment, err := h.userUC.ProcessPayment(context.Background(), &req)
	if err != nil {
		h.sendError(ws, userID, err, constants.ErrorInvalidFormat, constants.ErrorSeverityServer)
		return nil
	}

	// Send response with EventPaymentProcessed
	paymentData, _ := json.Marshal(payment)
	response := models.WSMessage{
		Event: constants.EventPaymentProcessed,
		Data:  paymentData,
	}

	return websocket.JSON.Send(ws, response)
}
