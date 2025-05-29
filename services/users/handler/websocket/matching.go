package websocket

import (
	"context"
	"encoding/json"
	"log"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// handleFinderUpdate processes finder status updates from clients
func (m *WebSocketManager) handleFinderUpdate(client *models.WebSocketClient, data json.RawMessage) error {
	var req models.FinderRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid finder request format")
	}

	// Update finder status
	if err := m.userUC.UpdateFinderStatus(context.Background(), &req); err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidBeacon, err.Error())
	}

	return m.manager.SendMessage(client.Conn, constants.EventFinderUpdate, models.FinderResponse{
		Message: "Finder status updated successfully",
	})
}

// handleBeaconUpdate processes beacon status updates from clients
func (m *WebSocketManager) handleBeaconUpdate(client *models.WebSocketClient, data json.RawMessage) error {
	var req models.BeaconRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid beacon request format")
	}

	// Update beacon status
	if err := m.userUC.UpdateBeaconStatus(context.Background(), &req); err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidBeacon, err.Error())
	}

	return m.manager.SendMessage(client.Conn, constants.EventBeaconUpdate, models.BeaconResponse{
		Message: "Beacon status updated successfully",
	})
}

// handleMatchConfirmation processes match acceptance from drivers
func (m *WebSocketManager) handleMatchConfirmation(client *models.WebSocketClient, data json.RawMessage) error {

	var confirm models.MatchConfirmRequest
	if err := json.Unmarshal(data, &confirm); err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid match proposal format")
	}
	confirm.UserID = client.UserID

	// Update match status
	result, err := m.userUC.ConfirmMatch(context.Background(), &confirm)
	if err != nil {
		log.Printf("Error confirming match for user %s: %v", client.UserID, err)
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorMatchUpdateFailed, err.Error())
	}
	log.Print(result)

	// Directly notify both driver and passenger about the confirmation
	// since we now have the result directly from the HTTP call
	m.manager.NotifyClient(result.DriverID, string(confirm.Status), result)
	m.manager.NotifyClient(result.PassengerID, string(confirm.Status), result)

	return nil
}
