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
	// Subscribe to ride started events
	rideStartedSub, err := h.natsClient.Subscribe(constants.SubjectRideStarted, func(msg *nats.Msg) {
		if err := h.handleRideStartedEvent(msg.Data); err != nil {
			fmt.Printf("Error handling ride started event: %v\n", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to ride started events: %w", err)
	}
	h.subs = append(h.subs, rideStartedSub)

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

// handleRideStartedEvent processes ride started events
func (h *NatsHandler) handleRideStartedEvent(msg []byte) error {
	var ride models.Ride
	if err := json.Unmarshal(msg, &ride); err != nil {
		return fmt.Errorf("failed to unmarshal ride started event: %w", err)
	}

	fmt.Printf("Received ride started event: rideID=%s, driverID=%s, customerID=%s\n",
		ride.RideID, ride.DriverID, ride.CustomerID)

	// Notify both driver and passenger about the ride start
	h.wsManager.NotifyClient(ride.DriverID.String(), constants.SubjectRideStarted, ride)
	h.wsManager.NotifyClient(ride.CustomerID.String(), constants.SubjectRideStarted, ride)

	return nil
}

// handleRideCompletedEvent processes ride completed events
func (h *NatsHandler) handleRideCompletedEvent(msg []byte) error {
	var rideComplete models.RideComplete
	if err := json.Unmarshal(msg, &rideComplete); err != nil {
		return fmt.Errorf("failed to unmarshal ride completed event: %w", err)
	}

	fmt.Printf("Received ride completed event: rideID=%s, driverID=%s, customerID=%s\n",
		rideComplete.Ride.RideID, rideComplete.Ride.DriverID, rideComplete.Ride.CustomerID)

	// Notify driver and passenger about the ride completion
	h.wsManager.NotifyClient(rideComplete.Ride.DriverID.String(), constants.EventRideCompleted, rideComplete)
	h.wsManager.NotifyClient(rideComplete.Ride.CustomerID.String(), constants.EventRideCompleted, rideComplete)

	return nil
}
