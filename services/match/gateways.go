package match

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

//go:generate mockgen -destination=mocks/mock_gateway.go -package=mocks github.com/piresc/nebengjek/services/match MatchGW,LocationGW

// MatchGW defines the match gateaways interface
type MatchGW interface {
	PublishMatchFound(ctx context.Context, matchProp models.MatchProposal) error
	PublishMatchRejected(ctx context.Context, matchProp models.MatchProposal) error
	PublishMatchAccepted(ctx context.Context, matchProp models.MatchProposal) error
}

// LocationGW defines the location service gateway interface
type LocationGW interface {
	AddAvailableDriver(ctx context.Context, driverID string, location *models.Location) error
	RemoveAvailableDriver(ctx context.Context, driverID string) error
	AddAvailablePassenger(ctx context.Context, passengerID string, location *models.Location) error
	RemoveAvailablePassenger(ctx context.Context, passengerID string) error
	FindNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.NearbyUser, error)
	GetDriverLocation(ctx context.Context, driverID string) (models.Location, error)
	GetPassengerLocation(ctx context.Context, passengerID string) (models.Location, error)
}
