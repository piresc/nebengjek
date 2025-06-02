package match

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -package=mocks github.com/piresc/nebengjek/services/match MatchUC

// MatchUC defines the interface for match business logic
type MatchUC interface {
	HandleBeaconEvent(event models.BeaconEvent) error
	HandleFinderEvent(event models.FinderEvent) error
	ConfirmMatchStatus(req *models.MatchConfirmRequest) (models.MatchProposal, error)
	GetMatch(ctx context.Context, matchID string) (*models.Match, error)
	GetPendingMatch(ctx context.Context, matchID string) (*models.Match, error)
	ReleaseDriver(driverID string) error
	ReleasePassenger(passengerID string) error
	RemoveDriverFromPool(ctx context.Context, driverID string) error
	RemovePassengerFromPool(ctx context.Context, passengerID string) error
}
