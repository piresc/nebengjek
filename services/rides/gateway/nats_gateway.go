package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/rides"
)

// NATSPublisher interface for publishing messages
type NATSPublisher interface {
	Publish(subject string, data []byte) error
}

// NATSGateway handles NATS events and integrates with ride use cases
type NATSGateway struct {
	rideUC    rides.RideUC
	publisher NATSPublisher
}

// NewNATSGateway creates a new NATS gateway
func NewNATSGateway(rideUC rides.RideUC, publisher NATSPublisher) *NATSGateway {
	return &NATSGateway{
		rideUC:    rideUC,
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

// handleMatchEvent handles incoming match events
func (g *NATSGateway) handleMatchEvent(ctx context.Context, matchEvent models.MatchProposal) error {
	// Create a ride from the match using the ride usecase
	return g.rideUC.CreateRide(ctx, matchEvent)
}

// handleLocationEvent handles incoming location events
func (g *NATSGateway) handleLocationEvent(ctx context.Context, locationEvent models.LocationAggregate) error {
	// Process billing update from location data
	billingUpdate := &models.BillingLedger{
		EntryID:   uuid.New(),
		Distance:  locationEvent.Distance,
		CreatedAt: time.Now(),
	}

	return g.rideUC.ProcessBillingUpdate(ctx, locationEvent.RideID, billingUpdate)
}