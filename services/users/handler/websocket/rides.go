package websocket

import (
	"context"
	"encoding/json"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// handleRideArrived processes ride arrival events from WebSocket clients
func (m *WebSocketManager) handleRideArrived(client *models.WebSocketClient, data json.RawMessage) error {
	var event models.RideCompleteEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid ride arrival format")
	}

	// Publish to NATS for rides-service to process completion
	m.userUC.RideArrived(context.Background(), &event)

	return nil
}
