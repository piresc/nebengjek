package nats

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// InitRideConsumers initializes NATS consumers for ride-related events
func (h *Handler) InitRideConsumers() error {
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

	return nil
}

// handleRideStartedEvent processes ride started events
func (h *Handler) handleRideStartedEvent(msg []byte) error {
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
