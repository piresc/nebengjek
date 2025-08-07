package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// NATSPublisher interface for publishing messages
type NATSPublisher interface {
	Publish(subject string, data []byte) error
}

// NATSGateway handles NATS events and integrates with ride use cases
type NATSGateway struct {
	publisher NATSPublisher
}

// NewNATSGateway creates a new NATS gateway
func NewNATSGateway(publisher NATSPublisher) *NATSGateway {
	return &NATSGateway{
		publisher: publisher,
	}
}

// PublishRidePickupEvent publishes a ride pickup event to NATS
func (g *NATSGateway) PublishRidePickupEvent(ctx context.Context, event *models.RidePickupEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal ride pickup event: %w", err)
	}

	return g.publisher.Publish(constants.SubjectRidePickup, data)
}

// PublishRideCompleteEvent publishes a ride complete event to NATS
func (g *NATSGateway) PublishRideCompleteEvent(ctx context.Context, event *models.RideCompleteEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal ride complete event: %w", err)
	}

	return g.publisher.Publish(constants.SubjectRideCompleted, data)
}
