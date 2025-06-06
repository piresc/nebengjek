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
	ListMatchesByPassenger(ctx context.Context, passengerID uuid.UUID) ([]*models.Match, error)
	ConfirmMatchByUser(ctx context.Context, matchID string, userID string, isDriver bool) (*models.Match, error)

	BatchUpdateMatchStatus(ctx context.Context, matchIDs []string, status models.MatchStatus) error

	// Active ride tracking operations
	SetActiveRide(ctx context.Context, driverID, passengerID, rideID string) error
	RemoveActiveRide(ctx context.Context, driverID, passengerID string) error
	GetActiveRideByDriver(ctx context.Context, driverID string) (string, error)
	GetActiveRideByPassenger(ctx context.Context, passengerID string) (string, error)
}
