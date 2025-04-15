package nats

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/user/handler/websocket"
)

// Handler handles NATS events for the user service
type Handler struct {
	wsManager  *websocket.WebSocketManager
	natsClient *natspkg.Client
	natsURL    string
	subs       []*nats.Subscription
}

// NewHandler creates a new NATS handler
func NewHandler(wsManager *websocket.WebSocketManager, natsURL string) (*Handler, error) {
	client, err := natspkg.NewClient(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS client: %w", err)
	}

	h := &Handler{
		wsManager:  wsManager,
		natsClient: client,
		natsURL:    natsURL,
	}

	if err := h.initConsumers(); err != nil {
		return nil, fmt.Errorf("failed to initialize NATS consumers: %w", err)
	}

	return h, nil
}

// initConsumers initializes all NATS consumers
func (h *Handler) initConsumers() error {
	// Subscribe to match found events
	matchSub, err := h.natsClient.Subscribe(constants.SubjectMatchFound, func(msg *nats.Msg) {
		if err := h.handleMatchEvent(msg.Data); err != nil {
			fmt.Printf("Error handling match event: %v\n", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to match events: %w", err)
	}
	h.subs = append(h.subs, matchSub)

	// Subscribe to match accepted events
	matchAcceptSub, err := h.natsClient.Subscribe(constants.SubjectMatchAccepted, func(msg *nats.Msg) {
		if err := h.handleMatchAcceptedEvent(msg.Data); err != nil {
			fmt.Printf("Error handling match accepted event: %v\n", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to match accepted events: %w", err)
	}
	h.subs = append(h.subs, matchAcceptSub)

	// Subscribe to match rejected events
	matchRejectSub, err := h.natsClient.Subscribe(constants.SubjectMatchRejected, func(msg *nats.Msg) {
		if err := h.handleMatchRejectedEvent(msg.Data); err != nil {
			fmt.Printf("Error handling match rejected event: %v\n", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to match rejected events: %w", err)
	}
	h.subs = append(h.subs, matchRejectSub)

	// Subscribe to trip events
	tripSub, err := h.natsClient.Subscribe(constants.SubjectTripStarted, func(msg *nats.Msg) {
		if err := h.handleTripEvent(msg.Data); err != nil {
			fmt.Printf("Error handling trip event: %v\n", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to trip events: %w", err)
	}
	h.subs = append(h.subs, tripSub)

	return nil
}

// Close unsubscribes from all NATS subscriptions
func (h *Handler) Close() {
	for _, sub := range h.subs {
		sub.Unsubscribe()
	}
}

// handleMatchEvent processes match events
func (h *Handler) handleMatchEvent(msg []byte) error {
	var event models.MatchProposal
	if err := json.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("failed to unmarshal match event: %w", err)
	}

	// Notify both driver and passenger
	h.wsManager.NotifyClient(event.DriverID, constants.SubjectMatchFound, event)
	h.wsManager.NotifyClient(event.PassengerID, constants.SubjectMatchFound, event)
	return nil
}

// handleMatchAcceptedEvent processes match accepted events
func (h *Handler) handleMatchAcceptedEvent(msg []byte) error {
	var event models.MatchProposal
	if err := json.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("failed to unmarshal match accepted event: %w", err)
	}

	// Notify both driver and passenger about the acceptance
	h.wsManager.NotifyClient(event.DriverID, constants.EventMatchAccepted, event)
	h.wsManager.NotifyClient(event.PassengerID, constants.EventMatchAccepted, event)
	return nil
}

// handleMatchRejectedEvent processes match rejected events
func (h *Handler) handleMatchRejectedEvent(msg []byte) error {
	var event models.MatchProposal
	if err := json.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("failed to unmarshal match rejected event: %w", err)
	}

	// Only notify the driver whose match was rejected
	h.wsManager.NotifyClient(event.DriverID, constants.EventMatchRejected, event)
	return nil
}

// handleTripEvent processes trip-related events
func (h *Handler) handleTripEvent(msg []byte) error {
	var event map[string]interface{}
	if err := json.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("failed to unmarshal trip event: %w", err)
	}

	// Extract relevant information
	driverID, _ := event["driver_id"].(string)
	passengerID, _ := event["passenger_id"].(string)
	eventType, _ := event["type"].(string)

	// Notify relevant clients
	if driverID != "" {
		h.wsManager.NotifyClient(driverID, eventType, event)
	}
	if passengerID != "" {
		h.wsManager.NotifyClient(passengerID, eventType, event)
	}

	return nil
}
