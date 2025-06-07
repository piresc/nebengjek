package gateway

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// NATS Gateway delegation methods

// PublishMatchFound forwards to the NATS gateway implementation
func (g *MatchGW) PublishMatchFound(ctx context.Context, matchProp models.MatchProposal) error {
	return g.natsGateway.PublishMatchFound(ctx, matchProp)
}

// PublishMatchRejected forwards to the NATS gateway implementation
func (g *MatchGW) PublishMatchRejected(ctx context.Context, matchProp models.MatchProposal) error {
	return g.natsGateway.PublishMatchRejected(ctx, matchProp)
}

// PublishMatchAccepted forwards to the NATS gateway implementation
func (g *MatchGW) PublishMatchAccepted(ctx context.Context, matchProp models.MatchProposal) error {
	return g.natsGateway.PublishMatchAccepted(ctx, matchProp)
}

// HTTP Gateway delegation methods

// AddAvailableDriver forwards to the HTTP gateway implementation
func (g *MatchGW) AddAvailableDriver(ctx context.Context, driverID string, location *models.Location) error {
	return g.httpGateway.AddAvailableDriver(ctx, driverID, location)
}

// RemoveAvailableDriver forwards to the HTTP gateway implementation
func (g *MatchGW) RemoveAvailableDriver(ctx context.Context, driverID string) error {
	return g.httpGateway.RemoveAvailableDriver(ctx, driverID)
}

// AddAvailablePassenger forwards to the HTTP gateway implementation
func (g *MatchGW) AddAvailablePassenger(ctx context.Context, passengerID string, location *models.Location) error {
	return g.httpGateway.AddAvailablePassenger(ctx, passengerID, location)
}

// RemoveAvailablePassenger forwards to the HTTP gateway implementation
func (g *MatchGW) RemoveAvailablePassenger(ctx context.Context, passengerID string) error {
	return g.httpGateway.RemoveAvailablePassenger(ctx, passengerID)
}

// FindNearbyDrivers forwards to the HTTP gateway implementation
func (g *MatchGW) FindNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.NearbyUser, error) {
	return g.httpGateway.FindNearbyDrivers(ctx, location, radiusKm)
}

// GetDriverLocation forwards to the HTTP gateway implementation
func (g *MatchGW) GetDriverLocation(ctx context.Context, driverID string) (models.Location, error) {
	return g.httpGateway.GetDriverLocation(ctx, driverID)
}

// GetPassengerLocation forwards to the HTTP gateway implementation
func (g *MatchGW) GetPassengerLocation(ctx context.Context, passengerID string) (models.Location, error) {
	return g.httpGateway.GetPassengerLocation(ctx, passengerID)
}
