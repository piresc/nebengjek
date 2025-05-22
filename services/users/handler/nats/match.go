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

	// Subscribe to match pending customer confirmation events
	matchPendingCustomerSub, err := h.natsClient.Subscribe(constants.SubjectMatchPendingCustomerConfirmation, func(msg *nats.Msg) {
		log.Printf("Received match pending customer confirmation event: %s\n", msg.Data)
		if err := h.handleMatchPendingCustomerConfirmationEvent(msg.Data); err != nil {
			fmt.Printf("Error handling match pending customer confirmation event: %v\n", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to match pending customer confirmation events: %w", err)
	}
	h.subs = append(h.subs, matchPendingCustomerSub)

	return nil
}

// handleMatchEvent processes match events
func (h *NatsHandler) handleMatchEvent(msg []byte) error {
	var event models.MatchProposal
	if err := json.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("failed to unmarshal match event: %w", err)
	}

	// Notify both driver and passenger
	// Assuming "server.match.found" is the WebSocket event type clients expect for this.
	h.wsManager.NotifyClient(event.DriverID, "server.match.found", event)
	h.wsManager.NotifyClient(event.PassengerID, "server.match.found", event)
	return nil
}

// handleMatchConfirmEvent processes match accepted events from MatchService (final confirmation)
func (h *NatsHandler) handleMatchConfirmEvent(msg []byte) error {
	var event models.MatchProposal
	if err := json.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("failed to unmarshal match confirm event: %w", err)
	}

	// Notify both driver and passenger about the final confirmation
	// Assuming "server.match.confirmed" is the WebSocket event type clients expect.
	// constants.EventMatchConfirm might be an older or different constant.
	// Using a string literal for clarity as per task description's conceptual WS message types.
	h.wsManager.NotifyClient(event.DriverID, "server.match.confirmed", event)
	h.wsManager.NotifyClient(event.PassengerID, "server.match.confirmed", event)
	return nil
}

// handleMatchRejectedEvent processes match rejected events from MatchService
func (h *NatsHandler) handleMatchRejectedEvent(msg []byte) error {
	var event models.MatchProposal
	if err := json.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("failed to unmarshal match rejected event: %w", err)
	}

	// Notify relevant parties about the rejection.
	// The original code notified only the driver. Depending on who rejected and at what stage,
	// the passenger might also need notification. For now, mirroring existing logic's target.
	// Using a string literal "server.match.rejected" for WS event type.
	// constants.EventMatchRejected might be an older or different constant.
	h.wsManager.NotifyClient(event.DriverID, "server.match.rejected", event)
	// If passenger was involved and needs notification:
	// h.wsManager.NotifyClient(event.PassengerID, "server.match.rejected", event)
	return nil
}

// handleMatchPendingCustomerConfirmationEvent processes events when a driver has accepted,
// and now the customer needs to confirm.
func (h *NatsHandler) handleMatchPendingCustomerConfirmationEvent(msg []byte) error {
	var event models.MatchProposal
	if err := json.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("failed to unmarshal match pending customer confirmation event: %w", err)
	}

	// The event (models.MatchProposal) contains MatchID, DriverID, PassengerID, locations etc.
	// This will be sent as the payload for the WebSocket message.
	// The WebSocket message type is "server.customer.match_confirmation_request".
	log.Printf("Notifying passenger %s for match confirmation request: %s", event.PassengerID, event.ID)
	h.wsManager.NotifyClient(event.PassengerID, "server.customer.match_confirmation_request", event)

	// TODO: Consider caching the 'event' (MatchProposal) temporarily if the WebSocket handler
	// for 'client.customer.match_response' cannot reconstruct it or get all necessary fields
	// (like DriverID, locations) from the client's response or session.
	// For example: h.cache.Set("matchproposal:"+event.ID, event, 5*time.Minute)

	return nil
}
