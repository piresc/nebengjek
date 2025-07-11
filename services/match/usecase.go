package match

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -package=mocks github.com/piresc/nebengjek/services/match MatchUC

// MatchUC defines the interface for match business logic
type MatchUC interface {
	HandleBeaconEvent(ctx context.Context, event models.BeaconEvent) error
	HandleFinderEvent(ctx context.Context, event models.FinderEvent) error
	ConfirmMatchStatus(ctx context.Context, req *models.MatchConfirmRequest) (models.MatchProposal, error)
	GetMatch(ctx context.Context, matchID string) (*models.Match, error)
	GetPendingMatch(ctx context.Context, matchID string) (*models.Match, error)
	RemoveDriverFromPool(ctx context.Context, driverID string) error
	RemovePassengerFromPool(ctx context.Context, passengerID string) error

	// Active ride management
	SetActiveRide(ctx context.Context, driverID, passengerID, rideID string) error
	RemoveActiveRide(ctx context.Context, driverID, passengerID string) error
	HasActiveRide(ctx context.Context, userID string, isDriver bool) (bool, error)
}
