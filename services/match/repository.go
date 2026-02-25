package match

import (
	"context"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models/match"
)

//go:generate mockgen -destination=mocks/mock_repository.go -package=mocks github.com/piresc/nebengjek/services/match MatchRepo

// MatchRepository defines the interface for match data access operations
type MatchRepo interface {
	// Match CRUD operations
	CreateMatch(ctx context.Context, match *match.Match) (*match.Match, error)
	GetMatch(ctx context.Context, matchID string) (*match.Match, error)
	UpdateMatchStatus(ctx context.Context, matchID string, status match.MatchStatus) error
	ListMatchesByPassenger(ctx context.Context, passengerID uuid.UUID) ([]*match.Match, error)
	ConfirmMatchByUser(ctx context.Context, matchID string, userID string, isDriver bool) (*match.Match, error)

	BatchUpdateMatchStatus(ctx context.Context, matchIDs []string, status match.MatchStatus) error

	// Active ride tracking operations
	SetActiveRide(ctx context.Context, driverID, passengerID, rideID string) error
	RemoveActiveRide(ctx context.Context, driverID, passengerID string) error
	GetActiveRideByDriver(ctx context.Context, driverID string) (string, error)
	GetActiveRideByPassenger(ctx context.Context, passengerID string) (string, error)
}
