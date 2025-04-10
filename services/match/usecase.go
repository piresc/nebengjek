package match

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// MatchUseCase defines the interface for match business logic
type MatchUseCase interface {
	// Match operations
	CreateMatchRequest(ctx context.Context, trip *models.Trip) error
	ProcessLocationUpdate(ctx context.Context, driverID string, location *models.Location) error
	FindMatchForPassenger(ctx context.Context, passengerID string, location *models.Location) (*models.Trip, error)
	AcceptMatch(ctx context.Context, tripID string, driverID string) error
	RejectMatch(ctx context.Context, tripID string, driverID string) error
	CancelMatch(ctx context.Context, tripID string, userID string) error
	GetPendingMatchesForDriver(ctx context.Context, driverID string) ([]*models.Trip, error)
	GetPendingMatchesForPassenger(ctx context.Context, passengerID string) ([]*models.Trip, error)
}
