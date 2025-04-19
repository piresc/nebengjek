package location

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// LocationRepo defines the interface for location data access operations
type LocationRepo interface {
	// StoreLocation stores a location update in Redis for a ride
	StoreLocation(ctx context.Context, rideID string, location models.Location) error

	// GetLastLocation gets the last stored location for a ride
	GetLastLocation(ctx context.Context, rideID string) (*models.Location, error)
}
