package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/user"
)

// WebSocketManager manages WebSocket connections and client state
type WebSocketManager struct {
	sync.RWMutex
	clients  map[string]*WebSocketClient
	userUC   user.UserUC
	jwtCfg   models.JWTConfig
	upgrader websocket.Upgrader
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager(userUC user.UserUC, jwtConfig models.JWTConfig) *WebSocketManager {
	return &WebSocketManager{
		clients: make(map[string]*WebSocketClient),
		userUC:  userUC,
		jwtCfg:  jwtConfig,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// HandleWebSocket handles new WebSocket connections
func (m *WebSocketManager) HandleWebSocket(c echo.Context) error {
	client, err := m.authenticateClient(c)
	if err != nil {
		return err
	}

	ws, err := m.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	return m.handleClientConnection(client, ws)
}

// authenticateClient authenticates the WebSocket client using JWT
func (m *WebSocketManager) authenticateClient(c echo.Context) (*WebSocketClient, error) {
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
		log.Printf("Token validation failed: %v", err)
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
	}

	return &WebSocketClient{
		UserID: claims.UserID,
		Role:   claims.Role,
	}, nil
}

// validateToken validates the JWT token and returns the claims
func (m *WebSocketManager) validateToken(tokenString string) (*CustomClaims, error) {
	claims := &CustomClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.jwtCfg.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// addClient safely adds a client to the manager
func (m *WebSocketManager) addClient(client *WebSocketClient) {
	m.Lock()
	defer m.Unlock()
	m.clients[client.UserID] = client
}

// removeClient safely removes a client from the manager
func (m *WebSocketManager) removeClient(userID string) {
	m.Lock()
	defer m.Unlock()
	delete(m.clients, userID)
}

// sendMessage sends a message to a WebSocket client
func (m *WebSocketManager) sendMessage(conn *websocket.Conn, event string, data interface{}) error {
	log.Printf("Sending message to client: %s", event)
	rawData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshaling message data: %v", err)
	}

	response := WSMessage{
		Event: event,
		Data:  rawData,
	}

	return conn.WriteJSON(response)
}

// sendErrorMessage sends an error message to a WebSocket client
func (m *WebSocketManager) sendErrorMessage(conn *websocket.Conn, code string, message string) error {
	return m.sendMessage(conn, constants.EventError, WSErrorMessage{
		Code:    code,
		Message: message,
	})
}

// NotifyClient sends a notification to a specific client
func (m *WebSocketManager) NotifyClient(userID string, event string, data interface{}) {
	log.Printf("Notifying client %s with event %s", userID, event)
	m.RLock()
	client, exists := m.clients[userID]
	m.RUnlock()

	if !exists {
		return
	}

	if err := m.sendMessage(client.Conn, event, data); err != nil {
		log.Printf("Error sending message to client %s: %v", userID, err)
	}
}
