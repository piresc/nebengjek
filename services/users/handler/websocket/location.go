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
