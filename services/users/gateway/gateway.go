package gateway

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// MatchAccept implements the UserGW interface method for match acceptance
func (g *UserGW) MatchConfirm(req *models.MatchConfirmRequest) (*models.MatchProposal, error) {
	return g.httpGateway.MatchConfirm(req)
}

// StartRide implements the UserGW interface method for starting a trip
func (g *UserGW) StartRide(req *models.RideStartRequest) (*models.Ride, error) {
	return g.httpGateway.StartRide(req)
}

// PublishBeaconEvent forwards to the NATS gateway implementation
func (g *UserGW) PublishBeaconEvent(ctx context.Context, event *models.BeaconEvent) error {
	return g.natsGateway.PublishBeaconEvent(ctx, event)
}

// PublishFinderEvent forwards to the NATS gateway implementation
func (g *UserGW) PublishFinderEvent(ctx context.Context, event *models.FinderEvent) error {
	return g.natsGateway.PublishFinderEvent(ctx, event)
}

// PublishLocationUpdate forwards to the NATS gateway implementation
func (g *UserGW) PublishLocationUpdate(ctx context.Context, locationEvent *models.LocationUpdate) error {
	return g.natsGateway.PublishLocationUpdate(ctx, locationEvent)
}

// PublishRideArrived forwards to the NATS gateway implementation
func (g *UserGW) PublishRideArrived(ctx context.Context, event *models.RideCompleteEvent) error {
	return g.natsGateway.PublishRideArrived(ctx, event)
}

// PublishRideStart forwards to the NATS gateway implementation
func (g *UserGW) PublishRideStart(ctx context.Context, event *models.RideStartTripEvent) error {
	return g.natsGateway.PublishRideStart(ctx, event)
}
