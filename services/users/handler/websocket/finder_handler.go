package websocket

import (
	"context"
	"encoding/json"

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
