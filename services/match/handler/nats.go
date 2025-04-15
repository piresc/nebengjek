package handler

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/pkg/nats"
)

// Topics for NATS messaging
const (
	BeaconEventTopic = "user.beacon"
)

// InitNATSConsumers initializes all NATS consumers for the match service
func (h *MatchHandler) InitNATSConsumers() error {
	// Initialize beacon events consumer
	_, err := nats.NewConsumer(
		BeaconEventTopic,
		"match-service",
		h.cfg.NATS.URL,
		h.handleBeaconEvent,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize beacon events consumer: %w", err)
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
