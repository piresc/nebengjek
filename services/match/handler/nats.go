package handler

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/constants"
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
			log.Printf("Error handling beacon event: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to beacon events: %w", err)
	}
	h.subs = append(h.subs, sub)

	// Note: Match confirmations are now handled directly via HTTP responses

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

// Note: Match confirmations are now handled directly via HTTP responses
