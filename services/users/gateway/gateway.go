package gateway

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// MatchAccept implements the UserGW interface method for match acceptance
func (g *UserGW) MatchConfirm(req *models.MatchConfirmRequest) (*models.MatchProposal, error) {
	return g.httpGateway.MatchConfirm(req)
}

// PublishBeaconEvent forwards to the NATS gateway implementation
func (g *UserGW) PublishBeaconEvent(ctx context.Context, event *models.BeaconEvent) error {
	return g.natsGateway.PublishBeaconEvent(ctx, event)
}

// PublishLocationUpdate forwards to the NATS gateway implementation
func (g *UserGW) PublishLocationUpdate(ctx context.Context, locationEvent *models.LocationUpdate) error {
	return g.natsGateway.PublishLocationUpdate(ctx, locationEvent)
}

// PublishRideArrived forwards to the NATS gateway implementation
func (g *UserGW) PublishRideArrived(ctx context.Context, event *models.RideCompleteEvent) error {
	return g.natsGateway.PublishRideArrived(ctx, event)
}
