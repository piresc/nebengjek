package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/match"
)

// NatsHandler handles NATS subscriptions for the match service
type MatchHandler struct {
	matchUC    match.MatchUC
	natsClient *natspkg.Client
	subs       []*nats.Subscription
}

// NewNatsHandler creates a new match NATS handler
func NewMatchHandler(matchUC match.MatchUC, client *natspkg.Client) *MatchHandler {
	return &MatchHandler{
		matchUC:    matchUC,
		natsClient: client,
		subs:       make([]*nats.Subscription, 0),
	}
}

// InitNATSConsumers initializes all NATS consumers for the match service
func (h *MatchHandler) InitNATSConsumers() error {
	// Initialize beacon events consumer
	sub, err := h.natsClient.Subscribe(constants.SubjectUserBeacon, func(msg *nats.Msg) {
		if err := h.handleBeaconEvent(msg.Data); err != nil {
			logger.Error("Error handling beacon event", logger.Err(err))
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to beacon events: %w", err)
	}
	h.subs = append(h.subs, sub)

	// Initialize finder events consumer
	sub, err = h.natsClient.Subscribe(constants.SubjectUserFinder, func(msg *nats.Msg) {
		if err := h.handleFinderEvent(msg.Data); err != nil {
			logger.Error("Error handling finder event", logger.Err(err))
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to finder events: %w", err)
	}
	h.subs = append(h.subs, sub)

	// Subscribe to ride pickup events to lock drivers
	sub, err = h.natsClient.Subscribe(constants.SubjectRidePickup, func(msg *nats.Msg) {
		if err := h.handleRidePickup(msg.Data); err != nil {
			logger.Error("Error handling ride pickup event", logger.Err(err))
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to ride pickup events: %w", err)
	}
	h.subs = append(h.subs, sub)

	// Subscribe to ride completed events to unlock users
	sub, err = h.natsClient.Subscribe(constants.SubjectRideCompleted, func(msg *nats.Msg) {
		if err := h.handleRideCompleted(msg.Data); err != nil {
			logger.Error("Error handling ride completed event", logger.Err(err))
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to ride completed events: %w", err)
	}
	h.subs = append(h.subs, sub)

	// Note: Match confirmations are now handled directly via HTTP responses

	return nil
}

// handleBeaconEvent processes beacon events from the user service
func (h *MatchHandler) handleBeaconEvent(msg []byte) error {
	var event models.BeaconEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		logger.ErrorCtx(context.Background(), "Failed to unmarshal beacon event", logger.Err(err))
		return err
	}

	logger.InfoCtx(context.Background(), "Received beacon event",
		logger.String("user_id", event.UserID),
		logger.Bool("is_active", event.IsActive))

	// Forward the event to usecase for processing
	return h.matchUC.HandleBeaconEvent(event)
}

// handleFinderEvent processes finder events from the user service
func (h *MatchHandler) handleFinderEvent(msg []byte) error {
	var event models.FinderEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		logger.ErrorCtx(context.Background(), "Failed to unmarshal finder event", logger.Err(err))
		return err
	}

	logger.InfoCtx(context.Background(), "Received finder event",
		logger.String("user_id", event.UserID),
		logger.Bool("is_active", event.IsActive),
		logger.Float64("location_lat", event.Location.Latitude),
		logger.Float64("location_lng", event.Location.Longitude),
		logger.Float64("target_lat", event.TargetLocation.Latitude),
		logger.Float64("target_lng", event.TargetLocation.Longitude))

	// Forward the event to usecase for processing
	return h.matchUC.HandleFinderEvent(event)
}

// handleRidePickup processes ride pickup events to lock drivers
func (h *MatchHandler) handleRidePickup(msg []byte) error {
	var ridePickup models.RideResp
	if err := json.Unmarshal(msg, &ridePickup); err != nil {
		logger.ErrorCtx(context.Background(), "Failed to unmarshal ride pickup event", logger.Err(err))
		return err
	}

	logger.InfoCtx(context.Background(), "Received ride pickup event",
		logger.String("ride_id", ridePickup.RideID),
		logger.String("driver_id", ridePickup.DriverID),
		logger.String("passenger_id", ridePickup.PassengerID))

	ctx := context.Background()

	// Store active ride information in Redis
	if err := h.matchUC.SetActiveRide(ctx, ridePickup.DriverID, ridePickup.PassengerID, ridePickup.RideID); err != nil {
		logger.WarnCtx(ctx, "Failed to set active ride",
			logger.String("ride_id", ridePickup.RideID),
			logger.Err(err))
		// Continue even if this fails - don't block ride flow
	}

	// Remove driver from available pool (lock them)
	if err := h.matchUC.RemoveDriverFromPool(context.Background(), ridePickup.DriverID); err != nil {
		logger.WarnCtx(ctx, "Failed to lock driver",
			logger.String("driver_id", ridePickup.DriverID),
			logger.Err(err))
		// Continue even if this fails - don't block ride flow
	}

	// Remove passenger from available pool (lock them)
	if err := h.matchUC.RemovePassengerFromPool(context.Background(), ridePickup.PassengerID); err != nil {
		logger.WarnCtx(ctx, "Failed to lock passenger",
			logger.String("passenger_id", ridePickup.PassengerID),
			logger.Err(err))
		// Continue even if this fails - don't block ride flow
	}

	return nil
}

// handleRideCompleted processes ride completed events to unlock users
func (h *MatchHandler) handleRideCompleted(msg []byte) error {
	var rideComplete models.RideComplete
	if err := json.Unmarshal(msg, &rideComplete); err != nil {
		logger.ErrorCtx(context.Background(), "Failed to unmarshal ride completed event", logger.Err(err))
		return err
	}

	logger.InfoCtx(context.Background(), "Received ride completed event",
		logger.String("ride_id", rideComplete.Ride.RideID.String()),
		logger.String("driver_id", rideComplete.Ride.DriverID.String()),
		logger.String("passenger_id", rideComplete.Ride.PassengerID.String()))

	ctx := context.Background()

	// Remove active ride information from Redis
	if err := h.matchUC.RemoveActiveRide(ctx, rideComplete.Ride.DriverID.String(), rideComplete.Ride.PassengerID.String()); err != nil {
		logger.WarnCtx(ctx, "Failed to remove active ride",
			logger.String("ride_id", rideComplete.Ride.RideID.String()),
			logger.Err(err))
		// Continue even if this fails
	}

	// Add driver back to available pool
	if err := h.matchUC.ReleaseDriver(rideComplete.Ride.DriverID.String()); err != nil {
		logger.WarnCtx(ctx, "Failed to release driver",
			logger.String("driver_id", rideComplete.Ride.DriverID.String()),
			logger.Err(err))
		// Continue even if this fails
	}

	// Add passenger back to available pool
	if err := h.matchUC.ReleasePassenger(rideComplete.Ride.PassengerID.String()); err != nil {
		logger.WarnCtx(ctx, "Failed to release passenger",
			logger.String("passenger_id", rideComplete.Ride.PassengerID.String()),
			logger.Err(err))
		// Continue even if this fails
	}

	return nil
}
