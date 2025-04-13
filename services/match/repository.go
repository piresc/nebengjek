package match

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// MatchRepo defines the interface for match data access operations
type MatchRepo interface {
	// Match operations
	CreateMatch(ctx context.Context, trip *models.Trip) error
	UpdateMatchStatus(ctx context.Context, tripID string, status models.TripStatus) error
	GetMatchByID(ctx context.Context, id string) (*models.Trip, error)
	GetPendingMatchesByDriverID(ctx context.Context, driverID string) ([]*models.Trip, error)
	GetPendingMatchesByPassengerID(ctx context.Context, passengerID string) ([]*models.Trip, error)

	// Driver availability operations
	AddAvailableDriver(ctx context.Context, driverID string, location *models.Location) error
	RemoveAvailableDriver(ctx context.Context, driverID string) error
	FindNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]string, error)

	// Passenger management
	AddAvailablePassenger(ctx context.Context, passengerID string, location *models.Location) error
	RemoveAvailablePassenger(ctx context.Context, passengerID string) error
	FindNearbyPassengers(ctx context.Context, location *models.Location, radiusKm float64) ([]string, error)
}
