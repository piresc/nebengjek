package location

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

//go:generate mockgen -destination=mocks/mock_repository.go -package=mocks github.com/piresc/nebengjek/services/location LocationRepo

// LocationRepo defines the interface for location data access operations
type LocationRepo interface {
	// StoreLocation stores a location update in Redis for a ride
	StoreLocation(ctx context.Context, rideID string, location models.Location) error

	// GetLastLocation gets the last stored location for a ride
	GetLastLocation(ctx context.Context, rideID string) (*models.Location, error)

	// Geo-related methods moved from match service
	// AddAvailableDriver adds a driver to the available drivers geo set
	AddAvailableDriver(ctx context.Context, driverID string, location *models.Location) error

	// RemoveAvailableDriver removes a driver from the available drivers sets
	RemoveAvailableDriver(ctx context.Context, driverID string) error

	// AddAvailablePassenger adds a passenger to the Redis geospatial index
	AddAvailablePassenger(ctx context.Context, passengerID string, location *models.Location) error

	// RemoveAvailablePassenger removes a passenger from the Redis geospatial index
	RemoveAvailablePassenger(ctx context.Context, passengerID string) error

	// FindNearbyDrivers finds available drivers within the specified radius
	FindNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.NearbyUser, error)

	// GetDriverLocation retrieves a driver's last known location
	GetDriverLocation(ctx context.Context, driverID string) (models.Location, error)

	// GetPassengerLocation retrieves a passenger's last known location
	GetPassengerLocation(ctx context.Context, passengerID string) (models.Location, error)
}
