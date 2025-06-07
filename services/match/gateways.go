package match

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

//go:generate mockgen -destination=mocks/mock_gateway.go -package=mocks github.com/piresc/nebengjek/services/match MatchGW

// MatchGW defines the unified match gateway interface that includes both NATS and location operations
type MatchGW interface {
	// NATS Gateway operations
	PublishMatchFound(ctx context.Context, matchProp models.MatchProposal) error
	PublishMatchRejected(ctx context.Context, matchProp models.MatchProposal) error
	PublishMatchAccepted(ctx context.Context, matchProp models.MatchProposal) error

	// HTTP Gateway operations (Location service)
	AddAvailableDriver(ctx context.Context, driverID string, location *models.Location) error
	RemoveAvailableDriver(ctx context.Context, driverID string) error
	AddAvailablePassenger(ctx context.Context, passengerID string, location *models.Location) error
	RemoveAvailablePassenger(ctx context.Context, passengerID string) error
	FindNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.NearbyUser, error)
	GetDriverLocation(ctx context.Context, driverID string) (models.Location, error)
	GetPassengerLocation(ctx context.Context, passengerID string) (models.Location, error)
}
