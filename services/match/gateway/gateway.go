package gateway

import (
	"context"

		matchmodels "github.com/piresc/nebengjek/internal/pkg/models/match"
	"github.com/piresc/nebengjek/internal/pkg/models/location"
)

// NATS Gateway delegation methods

// PublishMatchFound forwards to the NATS gateway implementation
func (g *MatchGW) PublishMatchFound(ctx context.Context, matchProp matchmodels.MatchProposal) error {
	return g.natsGateway.PublishMatchFound(ctx, matchProp)
}

// PublishMatchRejected forwards to the NATS gateway implementation
func (g *MatchGW) PublishMatchRejected(ctx context.Context, matchProp matchmodels.MatchProposal) error {
	return g.natsGateway.PublishMatchRejected(ctx, matchProp)
}

// PublishMatchAccepted forwards to the NATS gateway implementation
func (g *MatchGW) PublishMatchAccepted(ctx context.Context, matchProp matchmodels.MatchProposal) error {
	return g.natsGateway.PublishMatchAccepted(ctx, matchProp)
}

// Redis Gateway delegation methods

// AddAvailableDriver forwards to the Redis gateway implementation
func (g *MatchGW) AddAvailableDriver(ctx context.Context, driverID string, location *location.Location) error {
	return g.redisGateway.AddAvailableDriver(ctx, driverID, location)
}

// RemoveAvailableDriver forwards to the Redis gateway implementation
func (g *MatchGW) RemoveAvailableDriver(ctx context.Context, driverID string) error {
	return g.redisGateway.RemoveAvailableDriver(ctx, driverID)
}

// AddAvailablePassenger forwards to the Redis gateway implementation
func (g *MatchGW) AddAvailablePassenger(ctx context.Context, passengerID string, location *location.Location) error {
	return g.redisGateway.AddAvailablePassenger(ctx, passengerID, location)
}

// RemoveAvailablePassenger forwards to the Redis gateway implementation
func (g *MatchGW) RemoveAvailablePassenger(ctx context.Context, passengerID string) error {
	return g.redisGateway.RemoveAvailablePassenger(ctx, passengerID)
}

// FindNearbyDrivers forwards to the Redis gateway implementation
func (g *MatchGW) FindNearbyDrivers(ctx context.Context, location *location.Location, radiusKm float64) ([]*matchmodels.NearbyUser, error) {
	return g.redisGateway.FindNearbyDrivers(ctx, location, radiusKm)
}

// GetDriverLocation forwards to the Redis gateway implementation
func (g *MatchGW) GetDriverLocation(ctx context.Context, driverID string) (location.Location, error) {
	return g.redisGateway.GetDriverLocation(ctx, driverID)
}

// GetPassengerLocation forwards to the Redis gateway implementation
func (g *MatchGW) GetPassengerLocation(ctx context.Context, passengerID string) (location.Location, error) {
	return g.redisGateway.GetPassengerLocation(ctx, passengerID)
}
