package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// InitRideConsumers initializes NATS consumers for ride-related events
func (h *NatsHandler) initRideConsumers() error {

	// Subscribe to match accepted events
	matchAcceptedSub, err := h.natsClient.Subscribe(constants.SubjectMatchAccepted, func(msg *nats.Msg) {
		if err := h.handleMatchAcceptedEvent(msg.Data); err != nil {
			logger.ErrorCtx(context.Background(), "Error handling match accepted event", logger.Err(err))
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to match accepted events: %w", err)
	}
	h.subs = append(h.subs, matchAcceptedSub)

	// Subscribe to ride start trip events
	ridePickupSub, err := h.natsClient.Subscribe(constants.SubjectRidePickup, func(msg *nats.Msg) {
		if err := h.handleRidePickupEvent(msg.Data); err != nil {
			logger.ErrorCtx(context.Background(), "Error handling ride pickup event", logger.Err(err))
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to ride start trip events: %w", err)
	}
	h.subs = append(h.subs, ridePickupSub)

	// Subscribe to ride start trip events
	rideStartSub, err := h.natsClient.Subscribe(constants.SubjectRideStarted, func(msg *nats.Msg) {
		if err := h.handleRideStartEvent(msg.Data); err != nil {
			logger.ErrorCtx(context.Background(), "Error handling ride start event", logger.Err(err))
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to ride start events: %w", err)
	}
	h.subs = append(h.subs, rideStartSub)

	// Subscribe to ride completed events
	rideCompletedSub, err := h.natsClient.Subscribe(constants.SubjectRideCompleted, func(msg *nats.Msg) {
		if err := h.handleRideCompletedEvent(msg.Data); err != nil {
			logger.ErrorCtx(context.Background(), "Error handling ride completed event", logger.Err(err))
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to ride completed events: %w", err)
	}
	h.subs = append(h.subs, rideCompletedSub)

	return nil
}

// handleMatchAcceptedEvent processes match accepted events from NATS
func (h *NatsHandler) handleMatchAcceptedEvent(msg []byte) error {
	var matchProposal models.MatchProposal
	if err := json.Unmarshal(msg, &matchProposal); err != nil {
		return fmt.Errorf("failed to unmarshal match accepted event: %w", err)
	}

	logger.InfoCtx(context.Background(), "Received match accepted event",
		logger.String("match_id", matchProposal.ID),
		logger.String("driver_id", matchProposal.DriverID),
		logger.String("passenger_id", matchProposal.PassengerID))

	// Notify both driver and passenger that their match is confirmed and they're locked
	// Use a specific event type for match acceptance notification
	h.wsManager.NotifyClient(matchProposal.DriverID, constants.EventMatchConfirm, matchProposal)
	h.wsManager.NotifyClient(matchProposal.PassengerID, constants.EventMatchConfirm, matchProposal)

	return nil
}

// handleMatchEvent processes match events
func (h *NatsHandler) handleRidePickupEvent(msg []byte) error {
	var ridePickup models.RideResp
	if err := json.Unmarshal(msg, &ridePickup); err != nil {
		return fmt.Errorf("failed to unmarshal match event: %w", err)
	}

	logger.InfoCtx(context.Background(), "Received ride pickup event",
		logger.String("ride_id", ridePickup.RideID),
		logger.String("driver_id", ridePickup.DriverID),
		logger.String("passenger_id", ridePickup.PassengerID))

	// Notify both driver and passenger
	h.wsManager.NotifyClient(ridePickup.DriverID, constants.SubjectRidePickup, ridePickup)
	h.wsManager.NotifyClient(ridePickup.PassengerID, constants.SubjectRidePickup, ridePickup)
	return nil
}

// handleMatchAcceptedEvent processes match accepted events from NATS
func (h *NatsHandler) handleRideStartEvent(msg []byte) error {
	var rideStarted models.RideResp
	if err := json.Unmarshal(msg, &rideStarted); err != nil {
		return fmt.Errorf("failed to unmarshal ride start event: %w", err)
	}

	logger.InfoCtx(context.Background(), "Received ride started event",
		logger.String("ride_id", rideStarted.RideID),
		logger.String("driver_id", rideStarted.DriverID),
		logger.String("passenger_id", rideStarted.PassengerID))

	// Notify both driver and passenger that their match is confirmed and they're locked
	// Use a specific event type for match acceptance notification
	h.wsManager.NotifyClient(rideStarted.DriverID, constants.EventMatchConfirm, rideStarted)
	h.wsManager.NotifyClient(rideStarted.PassengerID, constants.EventMatchConfirm, rideStarted)

	return nil
}

// handleRideCompletedEvent processes ride completed events
func (h *NatsHandler) handleRideCompletedEvent(msg []byte) error {
	var rideComplete models.RideComplete
	if err := json.Unmarshal(msg, &rideComplete); err != nil {
		return fmt.Errorf("failed to unmarshal ride completed event: %w", err)
	}

	logger.InfoCtx(context.Background(), "Received ride completed event",
		logger.String("ride_id", rideComplete.Ride.RideID.String()),
		logger.String("driver_id", rideComplete.Ride.DriverID.String()),
		logger.String("passenger_id", rideComplete.Ride.PassengerID.String()))

	// Notify driver and passenger about the ride completion
	h.wsManager.NotifyClient(rideComplete.Ride.DriverID.String(), constants.EventRideCompleted, rideComplete)
	h.wsManager.NotifyClient(rideComplete.Ride.PassengerID.String(), constants.EventRideCompleted, rideComplete)

	return nil
}
