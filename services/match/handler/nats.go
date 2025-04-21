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

	// Initialize match acceptance consumer
	sub, err = h.natsClient.Subscribe(constants.SubjectMatchAccepted, func(msg *nats.Msg) {
		if err := h.handleMatchAccept(msg.Data); err != nil {
			log.Printf("Error handling match acceptance: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to match acceptance: %w", err)
	}
	h.subs = append(h.subs, sub)

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
