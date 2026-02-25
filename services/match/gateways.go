package match

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models/location"
	"github.com/piresc/nebengjek/internal/pkg/models/match"
)

//go:generate mockgen -destination=mocks/mock_gateway.go -package=mocks github.com/piresc/nebengjek/services/match MatchGW

// MatchGW defines the  match gateway interface that includes both NATS and location operations
type MatchGW interface {
	// NATS Gateway operations
	PublishMatchFound(ctx context.Context, matchProp match.MatchProposal) error
	PublishMatchRejected(ctx context.Context, matchProp match.MatchProposal) error
	PublishMatchAccepted(ctx context.Context, matchProp match.MatchProposal) error

	// HTTP Gateway operations (Location service)
	AddAvailableDriver(ctx context.Context, driverID string, location *location.Location) error
	RemoveAvailableDriver(ctx context.Context, driverID string) error
	AddAvailablePassenger(ctx context.Context, passengerID string, location *location.Location) error
	RemoveAvailablePassenger(ctx context.Context, passengerID string) error
	FindNearbyDrivers(ctx context.Context, location *location.Location, radiusKm float64) ([]*match.NearbyUser, error)
	GetDriverLocation(ctx context.Context, driverID string) (location.Location, error)
	GetPassengerLocation(ctx context.Context, passengerID string) (location.Location, error)
}
