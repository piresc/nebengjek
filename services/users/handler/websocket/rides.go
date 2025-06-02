package websocket

import (
	"context"
	"encoding/json"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// handleRideStartTrip processes ride start trip events from WebSocket clients
func (m *WebSocketManager) handleRideStart(client *models.WebSocketClient, data json.RawMessage) error {
	var event models.RideStartRequest
	if err := json.Unmarshal(data, &event); err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid ride start trip format")
	}
	resp, err := m.userUC.RideStart(context.Background(), &event)
	if err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Failed to process ride start trip")
	}
	// Directly notify both driver and passenger about the confirmation
	m.manager.NotifyClient(resp.DriverID.String(), constants.EventRideStarted, resp)
	m.manager.NotifyClient(resp.PassengerID.String(), constants.EventRideStarted, resp)

	return nil
}

// handleRideArrived processes ride arrival events from WebSocket clients
func (m *WebSocketManager) handleRideArrived(client *models.WebSocketClient, data json.RawMessage) error {
	var event models.RideCompleteEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid ride arrival format")
	}

	// Publish to NATS for rides-service to process completion
	if err := m.userUC.RideArrived(context.Background(), &event); err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Failed to process ride arrival")
	}

	return nil
}
