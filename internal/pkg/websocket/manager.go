package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// Manager manages WebSocket connections and client state
type Manager struct {
	sync.RWMutex
	clients  map[string]*models.WebSocketClient
	cfg      models.JWTConfig
	upgrader websocket.Upgrader
}

// NewManager creates a new WebSocket manager
func NewManager(jwtConfig models.JWTConfig) *Manager {
	return &Manager{
		clients: make(map[string]*models.WebSocketClient),
		cfg:     jwtConfig,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// HandleConnection authenticates and handles a new WebSocket connection
func (m *Manager) HandleConnection(c echo.Context, handleClient func(*models.WebSocketClient, *websocket.Conn) error) error {
	client, err := m.authenticateClient(c)
	if err != nil {
		return err
	}

	ws, err := m.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	return handleClient(client, ws)
}

// AuthenticateClient authenticates the WebSocket client using JWT
func (m *Manager) authenticateClient(c echo.Context) (*models.WebSocketClient, error) {
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Authorization header is required")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Invalid authorization format")
	}

	claims, err := m.validateToken(parts[1])
	if err != nil {
		logger.Warn("Token validation failed",
			logger.Err(err))
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
	}

	return &models.WebSocketClient{
		UserID: claims.UserID,
		Role:   claims.Role,
	}, nil
}

// ValidateToken validates the JWT token and returns the claims
func (m *Manager) validateToken(tokenString string) (*models.WebSocketClaims, error) {
	claims := &models.WebSocketClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.cfg.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// AddClient safely adds a client to the manager
func (m *Manager) AddClient(client *models.WebSocketClient) {
	m.Lock()
	defer m.Unlock()
	m.clients[client.UserID] = client
}

// RemoveClient safely removes a client from the manager
func (m *Manager) RemoveClient(userID string) {
	m.Lock()
	defer m.Unlock()
	delete(m.clients, userID)
}

// GetClient returns a client by ID
func (m *Manager) GetClient(userID string) (*models.WebSocketClient, bool) {
	m.RLock()
	defer m.RUnlock()
	client, exists := m.clients[userID]
	return client, exists
}

// SendMessage sends a message to a WebSocket client
func (m *Manager) SendMessage(conn *websocket.Conn, event string, data interface{}) error {
	if conn == nil {
		return nil // Handle nil connection gracefully for tests
	}

	rawData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshaling message data: %v", err)
	}

	response := models.WSMessage{
		Event: event,
		Data:  rawData,
	}

	return conn.WriteJSON(response)
}

// SendErrorMessage sends an error message to a WebSocket client
func (m *Manager) SendErrorMessage(conn *websocket.Conn, code string, message string) error {
	return m.SendMessage(conn, constants.EventError, models.WSErrorMessage{
		Code:    code,
		Message: message,
	})
}

// SendCategorizedError sends an error message based on severity level
func (m *Manager) SendCategorizedError(conn *websocket.Conn, err error, code string, severity constants.ErrorSeverity, userID string) error {
	// Always log detailed error server-side
	logger.Error("WebSocket operation failed",
		logger.String("user_id", userID),
		logger.String("error_code", code),
		logger.String("severity", m.getSeverityString(severity)),
		logger.Err(err))

	switch severity {
	case constants.ErrorSeverityClient:
		// Show detailed error to client for validation/input issues
		return m.SendErrorMessage(conn, code, err.Error())
	case constants.ErrorSeveritySecurity:
		// Minimal info to client for security issues
		logger.Warn("Security-related error occurred",
			logger.String("user_id", userID),
			logger.String("error_code", code),
			logger.Err(err))
		return m.SendErrorMessage(conn, code, "Access denied")
	default: // ErrorSeverityServer
		// Generic message for server errors
		return m.SendErrorMessage(conn, code, "Operation failed")
	}
}

// getSeverityString returns string representation of error severity
func (m *Manager) getSeverityString(severity constants.ErrorSeverity) string {
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

// NotifyClient sends a notification to a specific client
func (m *Manager) NotifyClient(userID string, event string, data interface{}) {
	logger.Debug("Notifying client",
		logger.String("user_id", userID),
		logger.String("event", event))
	m.RLock()
	client, exists := m.clients[userID]
	m.RUnlock()

	if !exists {
		return
	}

	if err := m.SendMessage(client.Conn, event, data); err != nil {
		logger.Warn("Error sending message to client",
			logger.String("user_id", userID),
			logger.Err(err))
	}
}
