package handler

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/pkg/nats"
)

// InitNATSConsumers initializes all NATS consumers for the rides service
func (h *ridesHandler) InitNATSConsumers() error {
	// Initialize rides acceptance consumer
	_, err := nats.NewConsumer(
		constants.SubjectMatchAccepted,
		"rides-service",
		h.cfg.NATS.URL,
		h.handleMatchAccept,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize rides acceptance consumer: %w", err)
	}

	// Initialize location aggregate consumer
	_, err = nats.NewConsumer(
		constants.SubjectLocationAggregate,
		"rides-service",
		h.cfg.NATS.URL,
		h.handleLocationAggregate,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize location aggregate consumer: %w", err)
	}

	// Subscribe to ride arrival events
	_, err = nats.NewConsumer(
		constants.SubjectRideArrived,
		"rides-service",
		h.cfg.NATS.URL,
		h.handleRideArrived,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize ride arrival consumer: %w", err)
	}

	return nil
}

// handleMatchAccept processes rides acceptance events
func (h *ridesHandler) handleMatchAccept(msg []byte) error {
	var match models.MatchProposal
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
func (h *ridesHandler) handleLocationAggregate(msg []byte) error {
	var update models.LocationAggregate
	if err := json.Unmarshal(msg, &update); err != nil {
		log.Printf("Failed to unmarshal location aggregate: %v", err)
		return err
	}

	log.Printf("Received location aggregate: rideID=%s, distance=%.2f km",
		update.RideID, update.Distance)

	// Only process if distance is >= 1km
	if update.Distance >= 1.0 {
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
func (h *ridesHandler) handleRideArrived(msg []byte) error {
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
