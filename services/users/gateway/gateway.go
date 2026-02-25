package gateway

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models/core"
)

// PublishBeaconEvent forwards to the NATS gateway implementation
func (g *UserGW) PublishBeaconEvent(ctx context.Context, event *core.BeaconEvent) error {
	return g.natsGateway.PublishBeaconEvent(ctx, event)
}

// PublishFinderEvent forwards to the NATS gateway implementation
func (g *UserGW) PublishFinderEvent(ctx context.Context, event *core.FinderEvent) error {
	return g.natsGateway.PublishFinderEvent(ctx, event)
}
