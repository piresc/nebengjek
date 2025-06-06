package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// handleLocationUpdate processes location updates from clients
func (m *WebSocketManager) handleLocationUpdate(driverID string, data json.RawMessage) error {
	var locationUpdate models.LocationUpdate
	if err := json.Unmarshal(data, &locationUpdate); err != nil {
		// Create a temporary client for error handling
		tempClient := &models.WebSocketClient{UserID: driverID}
		m.SendCategorizedError(tempClient, err, constants.ErrorInvalidFormat, constants.ErrorSeverityClient)
		return fmt.Errorf("invalid location format")
	}

	// logger.Info("Location update from user",
	//	logger.String("user_id", driverID),
	//	logger.Float64("latitude", locationUpdate.Location.Latitude),
	//	logger.Float64("longitude", locationUpdate.Location.Longitude),
	//	logger.String("trip_id", locationUpdate.RideID))

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
			logger.Warn("Client not found",
				logger.String("user_id", driverID))
			return fmt.Errorf("client not found")
		}
		return m.SendCategorizedError(client, err, constants.ErrorInvalidLocation, constants.ErrorSeverityServer)
	}

	return nil
}
