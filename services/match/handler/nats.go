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

	// Subscribe to customer match confirmed events
	sub, err = h.natsClient.Subscribe(constants.SubjectCustomerMatchConfirmed, func(msg *nats.Msg) {
		log.Printf("Received NATS message on subject: %s", msg.Subject)
		if err := h.handleCustomerMatchConfirmed(msg.Data); err != nil {
			log.Printf("Error handling customer match confirmed event: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to customer match confirmed events: %w", err)
	}
	h.subs = append(h.subs, sub)

	// Subscribe to customer match rejected events
	sub, err = h.natsClient.Subscribe(constants.SubjectCustomerMatchRejected, func(msg *nats.Msg) {
		log.Printf("Received NATS message on subject: %s", msg.Subject)
		if err := h.handleCustomerMatchRejected(msg.Data); err != nil {
			log.Printf("Error handling customer match rejected event: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to customer match rejected events: %w", err)
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

// handleMatchAccept processes match acceptance events (from driver)
func (h *MatchHandler) handleMatchAccept(msg []byte) error {
	var matchProposal models.MatchProposal
	if err := json.Unmarshal(msg, &matchProposal); err != nil {
		log.Printf("Failed to unmarshal match accept event: %v", err)
		return err
	}

	// It's assumed that for a driver's acceptance, the MatchProposal status
	// is set to models.MatchStatusPendingCustomerConfirmation by the User Service before publishing.
	// Or, if this SubjectMatchAccepted is purely for driver's initial "yes",
	// then ConfirmMatchStatus usecase should handle it accordingly.
	// Based on previous steps, ConfirmMatchStatus expects mp.MatchStatus to guide its actions.
	// If this handler is for driver's first acceptance, mp.MatchStatus should be models.MatchStatusPendingCustomerConfirmation.
	log.Printf("Received match acceptance (driver): matchID=%s, driverID=%s, passengerID=%s, status=%s",
		matchProposal.ID, matchProposal.DriverID, matchProposal.PassengerID, matchProposal.MatchStatus)

	// Update match status
	// The ConfirmMatchStatus usecase (modified in Step 2) will handle this based on mp.MatchStatus.
	// If mp.MatchStatus is PENDING_CUSTOMER_CONFIRMATION, it updates status and notifies customer.
	err := h.matchUC.ConfirmMatchStatus(matchProposal.ID, matchProposal)
	if err != nil {
		log.Printf("Failed to process match acceptance (driver): %v", err)
		return err
	}

	return nil
}

// handleCustomerMatchConfirmed processes events when a customer has confirmed a match.
func (h *MatchHandler) handleCustomerMatchConfirmed(msg []byte) error {
	var matchProposal models.MatchProposal
	if err := json.Unmarshal(msg, &matchProposal); err != nil {
		log.Printf("Failed to unmarshal customer match confirmed event: %v", err)
		return err
	}

	// Log the received event details, including the status which should be 'ACCEPTED'
	log.Printf("Received customer match confirmed: matchID=%s, driverID=%s, passengerID=%s, status=%s",
		matchProposal.ID, matchProposal.DriverID, matchProposal.PassengerID, matchProposal.MatchStatus)

	if matchProposal.MatchStatus != models.MatchStatusAccepted {
		log.Printf("Warning: Customer match confirmed event received for match %s, but status is '%s', expected '%s'",
			matchProposal.ID, matchProposal.MatchStatus, models.MatchStatusAccepted)
		// Depending on strictness, could return an error here or let the usecase handle it.
	}

	// Call the use case method. ConfirmMatchStatus (modified in Step 2) expects
	// mp.MatchStatus == models.MatchStatusAccepted for this path, leading to final persistence.
	err := h.matchUC.ConfirmMatchStatus(matchProposal.ID, matchProposal)
	if err != nil {
		log.Printf("Failed to process customer match confirmed: %v", err)
		return err
	}

	log.Printf("Successfully processed customer match confirmed for matchID: %s", matchProposal.ID)
	return nil
}

// handleCustomerMatchRejected processes events when a customer has rejected a match.
func (h *MatchHandler) handleCustomerMatchRejected(msg []byte) error {
	var matchProposal models.MatchProposal
	if err := json.Unmarshal(msg, &matchProposal); err != nil {
		log.Printf("Failed to unmarshal customer match rejected event: %v", err)
		return err
	}

	// Log the received event details, including the status which should be 'REJECTED'
	log.Printf("Received customer match rejected: matchID=%s, driverID=%s, passengerID=%s, status=%s",
		matchProposal.ID, matchProposal.DriverID, matchProposal.PassengerID, matchProposal.MatchStatus)

	if matchProposal.MatchStatus != models.MatchStatusRejected {
		log.Printf("Warning: Customer match rejected event received for match %s, but status is '%s', expected '%s'",
			matchProposal.ID, matchProposal.MatchStatus, models.MatchStatusRejected)
		// Depending on strictness, could return an error here or let the usecase handle it.
	}

	// Call the use case method. ConfirmMatchStatus (modified in Step 2) handles
	// mp.MatchStatus == models.MatchStatusRejected for rejection cleanup.
	err := h.matchUC.ConfirmMatchStatus(matchProposal.ID, matchProposal)
	if err != nil {
		log.Printf("Failed to process customer match rejected: %v", err)
		return err
	}

	log.Printf("Successfully processed customer match rejected for matchID: %s", matchProposal.ID)
	return nil
}
