package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// PublishBeaconEvent publishes a beacon event to NATS
func (g *UserGW) PublishBeaconEvent(ctx context.Context, event *models.BeaconEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	fmt.Printf("Publishing beacon event: %s\n", string(data))
	return g.natsClient.Publish(constants.SubjectUserBeacon, data)
}

// PublishMatchAccept publishes a match acceptance event to NATS
func (g *UserGW) MatchAccept(mp *models.MatchProposal) error {
	data, err := json.Marshal(mp)
	if err != nil {
		return err
	}
	fmt.Printf("Publishing match accept: %s\n", string(data))
	return g.natsClient.Publish(constants.SubjectMatchAccepted, data)
}

// PublishLocationUpdate publishes a location update event to NATS
func (g *UserGW) PublishLocationUpdate(ctx context.Context, locationEvent *models.LocationUpdate) error {
	data, err := json.Marshal(locationEvent)
	if err != nil {
		return err
	}
	fmt.Printf("Publishing location update: %s\n", string(data))
	return g.natsClient.Publish(constants.SubjectLocationUpdate, data)
}

func (g *UserGW) PublishRideArrived(ctx context.Context, event *models.RideCompleteEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	fmt.Printf("Publishing ride arrived event: %s\n", string(data))
	return g.natsClient.Publish(constants.SubjectRideArrived, data)
}
