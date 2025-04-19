package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// UpdateUserLocation updates a user's location and publishes it to the location service
func (uc *UserUC) UpdateUserLocation(ctx context.Context, lu *models.LocationUpdate) error {
	// Validate location data
	if lu == nil {
		return fmt.Errorf("location cannot be nil")
	}

	if lu.Location.Latitude == 0 && lu.Location.Longitude == 0 {
		return fmt.Errorf("invalid location coordinates")
	}

	// Get user to verify existence and get role
	user, err := uc.userRepo.GetUserByID(ctx, lu.DriverID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user.Role != "driver" {
		return fmt.Errorf("user is not a driver")
	}

	// Ensure timestamp is set
	if lu.Location.Timestamp.IsZero() {
		lu.Location.Timestamp = time.Now()
	}

	// Publish to NATS
	fmt.Printf("Publishing location update for user %s: ride_id=%s\n", lu.DriverID, lu.RideID)
	return uc.UserGW.PublishLocationUpdate(ctx, lu)
}
