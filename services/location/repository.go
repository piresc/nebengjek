package location

import (
	"context"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// LocationRepo defines the interface for location data access operations
type LocationRepo interface {
	// Driver location operations
	UpdateDriverLocation(ctx context.Context, driverID string, location *models.Location) error
	UpdateDriverAvailability(ctx context.Context, driverID string, isAvailable bool) error
	GetNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.User, error)

	// Customer location operations
	UpdateCustomerLocation(ctx context.Context, customerID string, location *models.Location) error

	// Background location tracking operations
	StoreLocationHistory(ctx context.Context, userID string, role string, location *models.Location) error
	GetLocationHistory(ctx context.Context, userID string, startTime, endTime time.Time) ([]*models.Location, error)
}
