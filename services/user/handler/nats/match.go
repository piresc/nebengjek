package nats

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// InitMatchConsumers initializes NATS consumers for match-related events
func (h *Handler) InitMatchConsumers() error {
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
	matchAcceptSub, err := h.natsClient.Subscribe(constants.SubjectMatchConfirm, func(msg *nats.Msg) {
		if err := h.handleMatchConfirmEvent(msg.Data); err != nil {
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

	return nil
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
func (h *Handler) handleMatchConfirmEvent(msg []byte) error {
	var event models.MatchProposal
	if err := json.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("failed to unmarshal match accepted event: %w", err)
	}

	// Notify both driver and passenger about the acceptance
	h.wsManager.NotifyClient(event.DriverID, constants.EventMatchConfirm, event)
	h.wsManager.NotifyClient(event.PassengerID, constants.EventMatchConfirm, event)
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
