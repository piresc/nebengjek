package handler

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
)

// NatsHandler handles NATS subscriptions for the match service
type NatsHandler struct {
	matchUC    MatchUsecase
	natsClient *natspkg.Client
	subs       []*nats.Subscription
	cfg        *models.Config
}

// NewNatsHandler creates a new match NATS handler
func NewNatsHandler(matchUC MatchUsecase, cfg *models.Config) (*NatsHandler, error) {
	client, err := natspkg.NewClient(cfg.NATS.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS client: %w", err)
	}

	return &NatsHandler{
		matchUC:    matchUC,
		natsClient: client,
		cfg:        cfg,
		subs:       make([]*nats.Subscription, 0),
	}, nil
}

// InitNATSConsumers initializes all NATS consumers for the match service
func (h *NatsHandler) InitNATSConsumers() error {
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
func (h *NatsHandler) handleBeaconEvent(msg []byte) error {
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
func (h *NatsHandler) handleMatchAccept(msg []byte) error {
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

// Close unsubscribes from all NATS subscriptions
func (h *NatsHandler) Close() {
	for _, sub := range h.subs {
		sub.Unsubscribe()
	}
	if h.natsClient != nil {
		h.natsClient.Close()
	}
}
