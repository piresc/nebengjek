package nats

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// InitMatchConsumers initializes NATS consumers for match-related events
func (h *NatsHandler) initMatchConsumers() error {
	// Subscribe to match found events
	matchSub, err := h.natsClient.Subscribe(constants.SubjectMatchFound, func(msg *nats.Msg) {
		log.Printf("Received match event: %s\n", msg.Data)
		if err := h.handleMatchEvent(msg.Data); err != nil {
			fmt.Printf("Error handling match event: %v\n", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to match events: %w", err)
	}
	h.subs = append(h.subs, matchSub)

	// No longer subscribe to match accepted events - handled directly via HTTP response

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
func (h *NatsHandler) handleMatchEvent(msg []byte) error {
	var event models.MatchProposal
	if err := json.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("failed to unmarshal match event: %w", err)
	}

	// Notify both driver and passenger
	h.wsManager.NotifyClient(event.DriverID, constants.SubjectMatchFound, event)
	h.wsManager.NotifyClient(event.PassengerID, constants.SubjectMatchFound, event)
	return nil
}

// handleMatchRejectedEvent processes match rejected events
func (h *NatsHandler) handleMatchRejectedEvent(msg []byte) error {
	var event models.MatchProposal
	if err := json.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("failed to unmarshal match rejected event: %w", err)
	}

	// Only notify the driver whose match was rejected
	h.wsManager.NotifyClient(event.DriverID, constants.EventMatchRejected, event)
	return nil
}
