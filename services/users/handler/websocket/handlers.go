package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

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

// handleMatchAccept processes match acceptance from drivers
func (m *WebSocketManager) handleMatchAccept(client *models.WebSocketClient, data json.RawMessage) error {
	UserID := client.UserID

	var matchProposalAccept models.MatchProposal
	if err := json.Unmarshal(data, &matchProposalAccept); err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid match proposal format")
	}

	// Update match status
	result, err := m.userUC.ConfirmMatch(context.Background(), &matchProposalAccept, UserID)
	if err != nil {
		log.Printf("Error confirming match for driver %s: %v", client.UserID, err)
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorMatchUpdateFailed, err.Error())
	}

	// Directly notify both driver and passenger about the confirmation
	// since we now have the result directly from the HTTP call
	m.manager.NotifyClient(result.DriverID, constants.EventMatchConfirm, result)
	m.manager.NotifyClient(result.PassengerID, constants.EventMatchConfirm, result)

	return nil
}

// handleLocationUpdate processes location updates from clients
func (m *WebSocketManager) handleLocationUpdate(driverID string, data json.RawMessage) error {
	var locationUpdate models.LocationUpdate
	if err := json.Unmarshal(data, &locationUpdate); err != nil {
		log.Printf("Error parsing location update from user %s: %v", driverID, err)
		return fmt.Errorf("invalid location format")
	}

	log.Printf("Location update from user %s: lat=%f, lng=%f, tripID=%s",
		driverID, locationUpdate.Location.Latitude, locationUpdate.Location.Longitude, locationUpdate.RideID)

	// Set timestamp if not provided
	if locationUpdate.Location.Timestamp.IsZero() {
		locationUpdate.Location.Timestamp = time.Now()
	}
	locationUpdate.DriverID = driverID
	locationUpdate.CreatedAt = time.Now()

	// Forward location update to the user usecase
	if err := m.userUC.UpdateUserLocation(context.Background(), &locationUpdate); err != nil {
		client, exists := m.manager.GetClient(driverID)
		if !exists {
			log.Printf("Client with ID %s not found", driverID)
			return fmt.Errorf("client not found")
		}
		log.Printf("Error updating location for user %s: %v", driverID, err)
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidLocation, err.Error())
	}

	return nil
}

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
