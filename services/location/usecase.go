package location

import (
	"context"

		"github.com/piresc/nebengjek/internal/pkg/models/location"
	matchmodels "github.com/piresc/nebengjek/internal/pkg/models/match"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -package=mocks github.com/piresc/nebengjek/services/location LocationUC

// LocationUseCase defines the interface for location business logic
type LocationUC interface {
	StoreLocation(ctx context.Context, location location.LocationUpdate) error

	// Geo-related methods
	AddAvailableDriver(ctx context.Context, driverID string, location *location.Location) error
	RemoveAvailableDriver(ctx context.Context, driverID string) error
	AddAvailablePassenger(ctx context.Context, passengerID string, location *location.Location) error
	RemoveAvailablePassenger(ctx context.Context, passengerID string) error
	FindNearbyDrivers(ctx context.Context, location *location.Location, radiusKm float64) ([]*matchmodels.NearbyUser, error)
	GetDriverLocation(ctx context.Context, driverID string) (location.Location, error)
	GetPassengerLocation(ctx context.Context, passengerID string) (location.Location, error)
}
