package location

import (
	"context"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// LocationUseCase defines the interface for location business logic
type LocationUseCase interface {
	// Driver location operations
	UpdateDriverLocation(ctx context.Context, driverID string, location *models.Location) error
	UpdateDriverAvailability(ctx context.Context, driverID string, isAvailable bool) error
	GetNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.User, error)

	// Customer location operations
	UpdateCustomerLocation(ctx context.Context, customerID string, location *models.Location) error

	// Background location tracking operations
	StartPeriodicUpdates(ctx context.Context, userID string, role string, interval time.Duration) error
	StopPeriodicUpdates(ctx context.Context, userID string) error

	// Event-based location updates
	UpdateLocationOnEvent(ctx context.Context, userID string, role string, location *models.Location, eventType string) error

	// Location history operations
	GetLocationHistory(ctx context.Context, userID string, startTime, endTime time.Time) ([]*models.Location, error)
}
