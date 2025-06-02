package nats

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// InitRideConsumers initializes NATS consumers for ride-related events
func (h *NatsHandler) initRideConsumers() error {

	// Subscribe to match accepted events
	matchAcceptedSub, err := h.natsClient.Subscribe(constants.SubjectMatchAccepted, func(msg *nats.Msg) {
		if err := h.handleMatchAcceptedEvent(msg.Data); err != nil {
			fmt.Printf("Error handling match accepted event: %v\n", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to match accepted events: %w", err)
	}
	h.subs = append(h.subs, matchAcceptedSub)

	// Subscribe to ride start trip events
	ridePickupSub, err := h.natsClient.Subscribe(constants.SubjectRidePickup, func(msg *nats.Msg) {
		if err := h.handleRidePickupEvent(msg.Data); err != nil {
			fmt.Printf("Error handling ride start trip event: %v\n", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to ride start trip events: %w", err)
	}
	h.subs = append(h.subs, ridePickupSub)

	// Subscribe to ride start trip events
	rideStartSub, err := h.natsClient.Subscribe(constants.SubjectRideStarted, func(msg *nats.Msg) {
		if err := h.handleRideStartEvent(msg.Data); err != nil {
			fmt.Printf("Error handling ride start trip event: %v\n", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to ride start events: %w", err)
	}
	h.subs = append(h.subs, rideStartSub)

	// Subscribe to ride completed events
	rideCompletedSub, err := h.natsClient.Subscribe(constants.SubjectRideCompleted, func(msg *nats.Msg) {
		if err := h.handleRideCompletedEvent(msg.Data); err != nil {
			fmt.Printf("Error handling ride completed event: %v\n", err)
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

	fmt.Printf("Received match accepted event: matchID=%s, driverID=%s, passengerID=%s\n",
		matchProposal.ID, matchProposal.DriverID, matchProposal.PassengerID)

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

	fmt.Printf("Received ride pickup event: rideID=%s, driverID=%s, passengerID=%s\n",
		ridePickup.RideID, ridePickup.DriverID, ridePickup.PassengerID)

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

	fmt.Printf("Received ride started event: rideID=%s, driverID=%s, passengerID=%s\n",
		rideStarted.RideID, rideStarted.DriverID, rideStarted.PassengerID)

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

	fmt.Printf("Received ride completed event: rideID=%s, driverID=%s, PassengerID=%s\n",
		rideComplete.Ride.RideID, rideComplete.Ride.DriverID, rideComplete.Ride.PassengerID)

	// Notify driver and passenger about the ride completion
	h.wsManager.NotifyClient(rideComplete.Ride.DriverID.String(), constants.EventRideCompleted, rideComplete)
	h.wsManager.NotifyClient(rideComplete.Ride.PassengerID.String(), constants.EventRideCompleted, rideComplete)

	return nil
}
