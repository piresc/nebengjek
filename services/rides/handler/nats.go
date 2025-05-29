package handler

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/rides"
)

type RidesHandler struct {
	ridesUC    rides.RideUC
	natsClient *natspkg.Client
	subs       []*nats.Subscription
	cfg        *models.Config
}

// NewNatsHandler creates a new rides NATS handler
func NewRidesHandler(
	ridesUC rides.RideUC,
	client *natspkg.Client,
	cfg *models.Config,
) *RidesHandler {
	return &RidesHandler{
		ridesUC:    ridesUC,
		natsClient: client,
		subs:       make([]*nats.Subscription, 0),
		cfg:        cfg,
	}
}

// InitNATSConsumers initializes all NATS consumers for the rides service
func (h *RidesHandler) InitNATSConsumers() error {
	// Initialize rides acceptance consumer
	sub, err := h.natsClient.Subscribe(constants.SubjectMatchAccepted, func(msg *nats.Msg) {
		if err := h.handleMatchConfirmation(msg.Data); err != nil {
			log.Printf("Error handling match acceptance: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to match acceptance: %w", err)
	}
	h.subs = append(h.subs, sub)

	// Initialize location aggregate consumer
	sub, err = h.natsClient.Subscribe(constants.SubjectLocationAggregate, func(msg *nats.Msg) {
		if err := h.handleLocationAggregate(msg.Data); err != nil {
			log.Printf("Error handling location aggregate: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to location aggregates: %w", err)
	}
	h.subs = append(h.subs, sub)

	// Subscribe to ride arrival events
	sub, err = h.natsClient.Subscribe(constants.SubjectRideArrived, func(msg *nats.Msg) {
		if err := h.handleRideArrived(msg.Data); err != nil {
			log.Printf("Error handling ride arrival: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to ride arrivals: %w", err)
	}
	h.subs = append(h.subs, sub)

	return nil
}

// handleMatchConfirmation processes rides acceptance events
func (h *RidesHandler) handleMatchConfirmation(msg []byte) error {
	var match models.MatchConfirmRequest
	if err := json.Unmarshal(msg, &match); err != nil {
		log.Printf("Failed to unmarshal match proposal: %v", err)
		return err
	}

	// Process match acceptance
	if err := h.ridesUC.CreateRide(match); err != nil {
		log.Printf("Failed to process match acceptance: %v", err)
		return err
	}

	return nil
}

// handleLocationAggregate processes location aggregates for billing
func (h *RidesHandler) handleLocationAggregate(msg []byte) error {
	var update models.LocationAggregate
	if err := json.Unmarshal(msg, &update); err != nil {
		log.Printf("Failed to unmarshal location aggregate: %v", err)
		return err
	}

	log.Printf("Received location aggregate: rideID=%s, distance=%.2f km",
		update.RideID, update.Distance)

	// Only process if distance is >= minimum configured distance
	if update.Distance >= h.cfg.Rides.MinDistanceKm {
		// Convert ride ID to UUID
		rideUUID, err := uuid.Parse(update.RideID)
		if err != nil {
			log.Printf("Invalid ride ID format: %v", err)
			return fmt.Errorf("invalid ride ID: %w", err)
		}

		// Calculate cost at 3000 IDR per km
		cost := int(update.Distance * 3000)

		// Create billing entry
		entry := &models.BillingLedger{
			RideID:   rideUUID,
			Distance: update.Distance,
			Cost:     cost,
		}

		// Store billing entry and update total cost
		if err := h.ridesUC.ProcessBillingUpdate(update.RideID, entry); err != nil {
			log.Printf("Failed to process billing update: %v", err)
			return err
		}
	}

	return nil
}

// handleRideArrived processes ride arrival events and completes the ride
func (h *RidesHandler) handleRideArrived(msg []byte) error {
	var event models.RideCompleteEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		log.Printf("Failed to unmarshal ride arrival event: %v", err)
		return err
	}

	log.Printf("Ride arrived event received: rideID=%s, adjustmentFactor=%.2f", event.RideID, event.AdjustmentFactor)

	// Complete ride which will calculate payment and publish ride.completed
	if _, err := h.ridesUC.CompleteRide(event.RideID, event.AdjustmentFactor); err != nil {
		log.Printf("Error completing ride on arrival: %v", err)
		return err
	}

	return nil
}
