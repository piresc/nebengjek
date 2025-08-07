package nats

import (
	"context"
	"encoding/json"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/match"
)

// MatchHandler handles JetStream subscriptions for the match service
type MatchHandler struct {
	matchUC    match.MatchUC
	natsClient *natspkg.Client
}

// NewMatchHandler creates a new match NATS handler
func NewMatchHandler(matchUC match.MatchUC, client *natspkg.Client) *MatchHandler {
	return &MatchHandler{
		matchUC:    matchUC,
		natsClient: client,
	}
}

// InitNATSConsumers initializes all JetStream consumers for the match service
func (h *MatchHandler) InitNATSConsumers() error {
	consumerConfigs := natspkg.DefaultConsumerConfigs()

	// Create and start user beacon consumer
	beaconConfig := consumerConfigs["user_beacon_match"]
	if err := h.natsClient.RecreateConsumer(beaconConfig); err != nil {
		return err
	}
	if err := h.natsClient.ConsumeMessages("USER_STREAM", "user_beacon_match", h.handleBeaconEventJS); err != nil {
		return err
	}

	// Create and start user finder consumer
	finderConfig := consumerConfigs["user_finder_match"]
	if err := h.natsClient.RecreateConsumer(finderConfig); err != nil {
		return err
	}
	if err := h.natsClient.ConsumeMessages("USER_STREAM", "user_finder_match", h.handleFinderEventJS); err != nil {
		return err
	}

	// Create and start ride pickup consumer
	ridePickupConfig := consumerConfigs["ride_pickup_match"]
	if err := h.natsClient.RecreateConsumer(ridePickupConfig); err != nil {
		return err
	}
	if err := h.natsClient.ConsumeMessages("RIDE_STREAM", "ride_pickup_match", h.handleRidePickupJS); err != nil {
		return err
	}

	// Create and start ride completed consumer
	rideCompletedConfig := consumerConfigs["ride_completed_match"]
	if err := h.natsClient.RecreateConsumer(rideCompletedConfig); err != nil {
		return err
	}
	if err := h.natsClient.ConsumeMessages("RIDE_STREAM", "ride_completed_match", h.handleRideCompletedJS); err != nil {
		return err
	}

	return nil
}

// handleBeaconEventJS processes beacon events from JetStream
func (h *MatchHandler) handleBeaconEventJS(msg jetstream.Msg) error {
	return h.handleBeaconEvent(context.Background(), msg.Data())
}

// handleFinderEventJS processes finder events from JetStream
func (h *MatchHandler) handleFinderEventJS(msg jetstream.Msg) error {
	return h.handleFinderEvent(context.Background(), msg.Data())
}

// handleRidePickupJS processes ride pickup events from JetStream
func (h *MatchHandler) handleRidePickupJS(msg jetstream.Msg) error {
	return h.handleRidePickup(context.Background(), msg.Data())
}

// handleRideCompletedJS processes ride completed events from JetStream
func (h *MatchHandler) handleRideCompletedJS(msg jetstream.Msg) error {
	return h.handleRideCompleted(context.Background(), msg.Data())
}

// handleBeaconEvent processes beacon events from the user service
func (h *MatchHandler) handleBeaconEvent(ctx context.Context, msg []byte) error {
	var event models.BeaconEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		return err
	}
	
	return h.matchUC.HandleBeaconEvent(ctx, event)
}

// handleFinderEvent processes finder events from the user service
func (h *MatchHandler) handleFinderEvent(ctx context.Context, msg []byte) error {
	var event models.FinderEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		return err
	}
	
	return h.matchUC.HandleFinderEvent(ctx, event)
}

// handleRidePickup processes ride pickup events to lock drivers
func (h *MatchHandler) handleRidePickup(ctx context.Context, msg []byte) error {
	var ridePickup models.RideResp
	if err := json.Unmarshal(msg, &ridePickup); err != nil {
		return err
	}

	// Store active ride information in Redis
	h.matchUC.SetActiveRide(ctx, ridePickup.DriverID, ridePickup.PassengerID, ridePickup.RideID)

	// Remove driver from available pool (lock them)
	h.matchUC.RemoveDriverFromPool(ctx, ridePickup.DriverID)

	// Remove passenger from available pool (lock them)
	h.matchUC.RemovePassengerFromPool(ctx, ridePickup.PassengerID)

	return nil
}

// handleRideCompleted processes ride completed events to unlock users
func (h *MatchHandler) handleRideCompleted(ctx context.Context, msg []byte) error {
	var rideComplete models.RideComplete
	if err := json.Unmarshal(msg, &rideComplete); err != nil {
		return err
	}

	// Remove active ride information from Redis
	h.matchUC.RemoveActiveRide(ctx, rideComplete.Ride.DriverID.String(), rideComplete.Ride.PassengerID.String())

	return nil
}