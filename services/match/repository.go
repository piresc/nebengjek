package match

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// MatchRepository defines the interface for match data access operations
type MatchRepo interface {
	// Match CRUD operations
	CreateMatch(ctx context.Context, match *models.Match) (*models.Match, error)
	GetMatch(ctx context.Context, matchID string) (*models.Match, error)
	UpdateMatchStatus(ctx context.Context, matchID string, status models.MatchStatus) error
	ListMatchesByDriver(ctx context.Context, driverID string) ([]*models.Match, error)
	ListMatchesByPassenger(ctx context.Context, passengerID string) ([]*models.Match, error)

	// Redis match proposal operations
	StoreMatchProposal(ctx context.Context, match *models.Match) error

	// Location-based operations
	AddAvailableDriver(ctx context.Context, driverID string, location *models.Location) error
	RemoveAvailableDriver(ctx context.Context, driverID string) error
	FindNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.NearbyUser, error)

	AddAvailablePassenger(ctx context.Context, passengerID string, location *models.Location) error
	RemoveAvailablePassenger(ctx context.Context, passengerID string) error
	FindNearbyPassengers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.NearbyUser, error)

	ProcessLocationUpdate(ctx context.Context, driverID string, location *models.Location) error
}
