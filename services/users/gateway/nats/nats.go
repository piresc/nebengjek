package gateway_nats

import (
	"context"
	"encoding/json"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
)

// NATSGateway implements the NATS gateway operations for the users service
type NATSGateway struct {
	client *natspkg.Client
}

// NewNATSGateway creates a new NATS gateway
func NewNATSGateway(client *natspkg.Client) *NATSGateway {
	return &NATSGateway{
		client: client,
	}
}

// PublishBeaconEvent publishes a beacon event to NATS
func (g *NATSGateway) PublishBeaconEvent(ctx context.Context, event *models.BeaconEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return g.client.Publish(constants.SubjectUserBeacon, data)
}

// PublishFinderEvent publishes a finder event to NATS
func (g *NATSGateway) PublishFinderEvent(ctx context.Context, event *models.FinderEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return g.client.Publish(constants.SubjectUserFinder, data)
}

// PublishRideStartTrip publishes a ride start event to NATS
func (g *NATSGateway) PublishRideStart(ctx context.Context, event *models.RideStartTripEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return g.client.Publish(constants.SubjectRideStarted, data)
}

// PublishLocationUpdate publishes a location update event to NATS
func (g *NATSGateway) PublishLocationUpdate(ctx context.Context, locationEvent *models.LocationUpdate) error {
	data, err := json.Marshal(locationEvent)
	if err != nil {
		return err
	}
	return g.client.Publish(constants.SubjectLocationUpdate, data)
}

// PublishRideArrived publishes a ride arrived event to NATS
func (g *NATSGateway) PublishRideArrived(ctx context.Context, event *models.RideCompleteEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return g.client.Publish(constants.SubjectRideArrived, data)
}
