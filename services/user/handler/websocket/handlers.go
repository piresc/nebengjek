package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

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
	case constants.EventMatchAccept:
		return m.handleMatchAccept(client, wsMsg.Data)
	case constants.EventLocationUpdate:
		return m.handleLocationUpdate(client.UserID, wsMsg.Data)
	case constants.EventRideArrived:
		return m.handleRideArrived(client, wsMsg.Data)
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

// handleMatchAccept processes match acceptance from drivers
func (m *WebSocketManager) handleMatchAccept(client *WebSocketClient, data json.RawMessage) error {
	UserID := client.UserID

	var matchProposalAccept models.MatchProposal
	if err := json.Unmarshal(data, &matchProposalAccept); err != nil {
		return m.sendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid match proposal format")
	}

	// Update beacon status
	err := m.userUC.ConfirmMatch(context.Background(), &matchProposalAccept, UserID)
	if err != nil {
		log.Printf("Error confirming match for driver %s: %v", client.UserID, err)
		return m.sendErrorMessage(client.Conn, constants.ErrorMatchUpdateFailed, err.Error())
	}
	return nil
}

// handleLocationUpdate processes location updates from clients
func (m *WebSocketManager) handleLocationUpdate(DriverID string, data json.RawMessage) error {
	var locationUpdate models.LocationUpdate
	if err := json.Unmarshal(data, &locationUpdate); err != nil {
		log.Printf("Error parsing location update from user %s: %v", DriverID, err)
		return fmt.Errorf("invalid location format")
	}

	log.Printf("Location update from user %s: lat=%f, lng=%f, tripID=%s",
		DriverID, locationUpdate.Location.Latitude, locationUpdate.Location.Longitude, locationUpdate.RideID)

	// Set timestamp if not provided
	if locationUpdate.Location.Timestamp.IsZero() {
		locationUpdate.Location.Timestamp = time.Now()
	}
	locationUpdate.DriverID = DriverID
	locationUpdate.CreatedAt = time.Now()

	// Forward location update to the user usecase
	if err := m.userUC.UpdateUserLocation(context.Background(), &locationUpdate); err != nil {
		log.Printf("Error updating location for user %s: %v", DriverID, err)
		return m.sendErrorMessage(m.clients[DriverID].Conn, constants.ErrorInvalidLocation, err.Error())
	}

	return nil
}

// handleRideArrived processes ride arrival events from WebSocket clients
func (m *WebSocketManager) handleRideArrived(client *WebSocketClient, data json.RawMessage) error {
	var event models.RideCompleteEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return m.sendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid ride arrival format")
	}

	// Publish to NATS for rides-service to process completion
	m.userUC.RideArrived(context.Background(), &event)

	return nil
}
