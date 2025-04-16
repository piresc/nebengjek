package handler

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/pkg/nats"
)

// InitNATSConsumers initializes all NATS consumers for the match service
func (h *MatchHandler) InitNATSConsumers() error {
	// Initialize beacon events consumer
	_, err := nats.NewConsumer(
		constants.SubjectUserBeacon,
		"match-service",
		h.cfg.NATS.URL,
		h.handleBeaconEvent,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize beacon events consumer: %w", err)
	}

	// Initialize match acceptance consumer
	_, err = nats.NewConsumer(
		constants.SubjectMatchAccepted,
		"match-service",
		h.cfg.NATS.URL,
		h.handleMatchAccept,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize match acceptance consumer: %w", err)
	}

	return nil
}

// handleBeaconEvent processes beacon events from the user service
func (h *MatchHandler) handleBeaconEvent(msg []byte) error {
	var event models.BeaconEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		log.Printf("Failed to unmarshal beacon event: %v", err)
		return err
	}

	log.Printf("Received beacon event: userID=%s, Role=%s, IsActive=%v",
		event.UserID, event.Role, event.IsActive)

	// Forward the event to usecase for processing
	return h.matchUC.HandleBeaconEvent(event)
}

// handleMatchAccept processes match acceptance events
func (h *MatchHandler) handleMatchAccept(msg []byte) error {
	var matchAccept models.MatchProposal
	if err := json.Unmarshal(msg, &matchAccept); err != nil {
		log.Printf("Failed to unmarshal match accept event: %v", err)
		return err
	}

	log.Printf("Received match acceptance: matchID=%s, driverID=%s, passengerID=%s",
		matchAccept.ID, matchAccept.DriverID, matchAccept.PassengerID)

	// Update match status in database
	err := h.matchUC.ConfirmMatchStatus(matchAccept.ID, matchAccept)
	if err != nil {
		log.Printf("Failed to update match status: %v", err)
		return err
	}

	return nil
}
