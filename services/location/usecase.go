package location

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -package=mocks github.com/piresc/nebengjek/services/location LocationUC

// LocationUseCase defines the interface for location business logic
type LocationUC interface {
	StoreLocation(location models.LocationUpdate) error

	// Geo-related methods
	AddAvailableDriver(ctx context.Context, driverID string, location *models.Location) error
	RemoveAvailableDriver(ctx context.Context, driverID string) error
	AddAvailablePassenger(ctx context.Context, passengerID string, location *models.Location) error
	RemoveAvailablePassenger(ctx context.Context, passengerID string) error
	FindNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.NearbyUser, error)
	GetDriverLocation(ctx context.Context, driverID string) (models.Location, error)
	GetPassengerLocation(ctx context.Context, passengerID string) (models.Location, error)
}
