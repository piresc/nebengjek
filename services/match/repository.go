package match

import (
	"context"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

//go:generate mockgen -destination=mocks/mock_repository.go -package=mocks github.com/piresc/nebengjek/services/match MatchRepo

// MatchRepository defines the interface for match data access operations
type MatchRepo interface {
	// Match CRUD operations
	CreateMatch(ctx context.Context, match *models.Match) (*models.Match, error)
	GetMatch(ctx context.Context, matchID string) (*models.Match, error)
	UpdateMatchStatus(ctx context.Context, matchID string, status models.MatchStatus) error
	ListMatchesByDriver(ctx context.Context, driverID string) ([]*models.Match, error)
	ListMatchesByPassenger(ctx context.Context, passengerID uuid.UUID) ([]*models.Match, error)
	ConfirmMatchByUser(ctx context.Context, matchID string, userID string, isDriver bool) (*models.Match, error)

	// Location-based operations
	AddAvailableDriver(ctx context.Context, driverID string, location *models.Location) error
	RemoveAvailableDriver(ctx context.Context, driverID string) error
	FindNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.NearbyUser, error)

	AddAvailablePassenger(ctx context.Context, passengerID string, location *models.Location) error
	RemoveAvailablePassenger(ctx context.Context, passengerID string) error
	FindNearbyPassengers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.NearbyUser, error)

	ProcessLocationUpdate(ctx context.Context, driverID string, location *models.Location) error
}
