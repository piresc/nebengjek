package websocket

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	pkgws "github.com/piresc/nebengjek/internal/pkg/websocket"
	"github.com/piresc/nebengjek/services/users"
)

// WebSocketManager extends the base WebSocket manager for user-specific functionality
type WebSocketManager struct {
	userUC  users.UserUC
	manager *pkgws.Manager
}

// NewWebSocketManager creates a new WebSocket manager for the user service
func NewWebSocketManager(
	userUC users.UserUC,
	manager *pkgws.Manager,
) *WebSocketManager {
	return &WebSocketManager{
		manager: manager,
		userUC:  userUC,
	}
}

// HandleWebSocket handles new WebSocket connections
func (m *WebSocketManager) HandleWebSocket(c echo.Context) error {
	return m.manager.HandleConnection(c, m.handleClientConnection)
}

// handleClientConnection manages the client's WebSocket connection
func (m *WebSocketManager) handleClientConnection(client *models.WebSocketClient, ws *websocket.Conn) error {
	client.Conn = ws
	m.manager.AddClient(client)
	defer m.manager.RemoveClient(client.UserID)

	return m.messageLoop(client)
}

// messageLoop handles incoming WebSocket messages
func (m *WebSocketManager) messageLoop(client *models.WebSocketClient) error {
	for {
		_, msg, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			return err
		}

		if err := m.handleMessage(client, msg); err != nil {
			log.Printf("Error handling message: %v", err)
		}
	}
}

func (m *WebSocketManager) NotifyClient(userID string, event string, data interface{}) {
	m.manager.NotifyClient(userID, event, data)
}

// handleMessage processes incoming WebSocket messages
func (m *WebSocketManager) handleMessage(client *models.WebSocketClient, msg []byte) error {
	var wsMsg models.WSMessage
	if err := json.Unmarshal(msg, &wsMsg); err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid message format")
	}

	switch wsMsg.Event {
	case constants.EventBeaconUpdate:
		return m.handleBeaconUpdate(client, wsMsg.Data)
	case constants.EventFinderUpdate:
		return m.handleFinderUpdate(client, wsMsg.Data)
	case constants.EventMatchConfirm:
		return m.handleMatchConfirmation(client, wsMsg.Data)
	case constants.EventLocationUpdate:
		return m.handleLocationUpdate(client.UserID, wsMsg.Data)
	case constants.EventRideStarted:
		return m.handleRideStart(client, wsMsg.Data)
	case constants.EventRideArrived:
		return m.handleRideArrived(client, wsMsg.Data)
	case constants.EventPaymentProcessed:
		return m.handleProcessPayment(client, wsMsg.Data)
	default:
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Unknown event type")
	}
}
