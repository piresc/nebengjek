package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// handleClientConnection manages the client's WebSocket connection
func (m *WebSocketManager) handleClientConnection(client *WebSocketClient, ws *websocket.Conn) error {
	client.Conn = ws
	m.addClient(client)
	defer m.removeClient(client.UserID)

	return m.messageLoop(client)
}

// messageLoop handles incoming WebSocket messages
func (m *WebSocketManager) messageLoop(client *WebSocketClient) error {
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

// handleMessage processes incoming WebSocket messages
func (m *WebSocketManager) handleMessage(client *WebSocketClient, msg []byte) error {
	var wsMsg WSMessage
	if err := json.Unmarshal(msg, &wsMsg); err != nil {
		return m.sendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid message format")
	}

	switch wsMsg.Event {
	case constants.EventBeaconUpdate:
		return m.handleBeaconUpdate(client, wsMsg.Data)
	case constants.EventLocationUpdate:
		return m.handleLocationUpdate(client.UserID, wsMsg.Data)
	default:
		return m.sendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Unknown event type")
	}
}

// handleBeaconUpdate processes beacon status updates from clients
func (m *WebSocketManager) handleBeaconUpdate(client *WebSocketClient, data json.RawMessage) error {
	var req models.BeaconRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return m.sendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid beacon request format")
	}

	// Update beacon status
	if err := m.userUC.UpdateBeaconStatus(context.Background(), &req); err != nil {
		return m.sendErrorMessage(client.Conn, constants.ErrorInvalidBeacon, err.Error())
	}

	return m.sendMessage(client.Conn, constants.EventBeaconUpdate, models.BeaconResponse{
		Message: "Beacon status updated successfully",
	})
}

// handleLocationUpdate processes location updates from clients
func (m *WebSocketManager) handleLocationUpdate(userID string, data json.RawMessage) error {
	var location models.Location
	if err := json.Unmarshal(data, &location); err != nil {
		log.Printf("Error parsing location update from user %s: %v", userID, err)
		return fmt.Errorf("invalid location format")
	}

	log.Printf("Location update from user %s: lat=%f, lng=%f",
		userID, location.Latitude, location.Longitude)

	return nil
}
