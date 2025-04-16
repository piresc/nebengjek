package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// UpdateUserLocation updates a user's location and publishes it to the location service
func (uc *UserUC) UpdateUserLocation(ctx context.Context, userID string, location *models.Location) error {
	// Validate location data
	if location == nil {
		return fmt.Errorf("location cannot be nil")
	}

	if location.Latitude == 0 && location.Longitude == 0 {
		return fmt.Errorf("invalid location coordinates")
	}

	// Get user to verify existence and get role
	user, err := uc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Ensure timestamp is set
	if location.Timestamp.IsZero() {
		location.Timestamp = time.Now()
	}

	// Create location update event
	locationEvent := struct {
		UserID    string           `json:"user_id"`
		Role      string           `json:"role"`
		Location  *models.Location `json:"location"`
		Timestamp time.Time        `json:"timestamp"`
	}{
		UserID:    userID,
		Role:      user.Role,
		Location:  location,
		Timestamp: time.Now(),
	}

	// Marshal the event to JSON
	data, err := json.Marshal(locationEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal location event: %w", err)
	}

	// Publish to NATS
	fmt.Printf("Publishing location update for user %s: %s\n", userID, string(data))
	return uc.UserGW.PublishLocationUpdate(ctx, locationEvent)
}
