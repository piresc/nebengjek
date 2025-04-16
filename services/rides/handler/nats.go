package handler

import (
	"encoding/json"
	"fmt"
	"log"

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

	return nil
}

// handleMatchAccept processes rides acceptance events
func (h *ridesHandler) handleMatchAccept(msg []byte) error {
	var matchConfirm models.MatchProposal
	if err := json.Unmarshal(msg, &matchConfirm); err != nil {
		log.Printf("Failed to unmarshal rides accept event: %v", err)
		return err
	}

	log.Printf("Received rides acceptance: ridesID=%s, driverID=%s, passengerID=%s",
		matchConfirm.ID, matchConfirm.DriverID, matchConfirm.PassengerID)

	// Update rides status in database
	err := h.ridesUC.CreateRide(matchConfirm)
	if err != nil {
		log.Printf("Failed to update rides status: %v", err)
		return err
	}

	return nil
}
