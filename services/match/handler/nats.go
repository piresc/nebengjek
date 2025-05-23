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

	// Initialize consumer for driver's initial acceptance of a match proposal.
	// This is the first step in the two-step match confirmation process.
	// The incoming NATS message payload (MatchProposal) is expected to have
	// MatchStatus = models.MatchStatusPendingCustomerConfirmation, set by the User Service.
	// Successful processing by matchUC.ConfirmMatchStatus will lead to the
	// Match Service publishing a 'match.pending_customer_confirmation' event.
	sub, err = h.natsClient.Subscribe(constants.SubjectMatchAccepted, func(msg *nats.Msg) {
		if err := h.handleMatchAccept(msg.Data); err != nil {
			// Critical errors (like wrong status) are logged and returned by handleMatchAccept.
			// This log captures such errors or other processing issues.
			log.Printf("Error processing driver's initial match acceptance (subject: %s): %v", constants.SubjectMatchAccepted, err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", constants.SubjectMatchAccepted, err)
	}
	h.subs = append(h.subs, sub)

	// Subscribe to customer's final confirmation of a match from User Service.
	// Expects MatchProposal with status MatchStatusAccepted.
	sub, err = h.natsClient.Subscribe(constants.SubjectCustomerMatchConfirmed, func(msg *nats.Msg) {
		log.Printf("Received NATS message on subject: %s for match %s", msg.Subject, getMatchIDFromMsg(msg.Data))
		if err := h.handleCustomerMatchConfirmed(msg.Data); err != nil {
			log.Printf("Error processing customer match confirmed event (subject: %s): %v", constants.SubjectCustomerMatchConfirmed, err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", constants.SubjectCustomerMatchConfirmed, err)
	}
	h.subs = append(h.subs, sub)

	// Subscribe to customer's rejection of a match from User Service.
	// Expects MatchProposal with status MatchStatusRejected.
	sub, err = h.natsClient.Subscribe(constants.SubjectCustomerMatchRejected, func(msg *nats.Msg) {
		log.Printf("Received NATS message on subject: %s for match %s", msg.Subject, getMatchIDFromMsg(msg.Data))
		if err := h.handleCustomerMatchRejected(msg.Data); err != nil {
			log.Printf("Error processing customer match rejected event (subject: %s): %v", constants.SubjectCustomerMatchRejected, err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", constants.SubjectCustomerMatchRejected, err)
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

// getMatchIDFromMsg is a helper to extract MatchID for logging, ignoring unmarshal errors for brevity.
func getMatchIDFromMsg(msgData []byte) string {
	var mp models.MatchProposal
	if json.Unmarshal(msgData, &mp) == nil {
		return mp.ID
	}
	return "unknown"
}

// handleMatchAccept processes the driver's initial provisional acceptance of a match.
// The incoming NATS message payload (MatchProposal) is expected to have
// MatchStatus = models.MatchStatusPendingCustomerConfirmation, set by the User Service.
// Successful processing via matchUC.ConfirmMatchStatus (which expects this status)
// will lead to the Match Service publishing a 'match.pending_customer_confirmation' event.
func (h *MatchHandler) handleMatchAccept(msg []byte) error {
	var matchProposal models.MatchProposal
	if err := json.Unmarshal(msg, &matchProposal); err != nil {
		log.Printf("handleMatchAccept: Failed to unmarshal MatchProposal: %v", err)
		return err
	}

	log.Printf("handleMatchAccept: Received driver's initial acceptance for matchID=%s, status=%s. Expecting %s.",
		matchProposal.ID, matchProposal.MatchStatus, models.MatchStatusPendingCustomerConfirmation)

	// CRITICAL VALIDATION: Ensure the incoming proposal has the correct status for this handler.
	if matchProposal.MatchStatus != models.MatchStatusPendingCustomerConfirmation {
		errMsg := fmt.Sprintf("handleMatchAccept: CRITICAL - Received unexpected status %s for match %s on subject %s. Expected %s. This may bypass customer confirmation.",
			matchProposal.MatchStatus, matchProposal.ID, constants.SubjectMatchAccepted, models.MatchStatusPendingCustomerConfirmation)
		log.Println(errMsg)
		return errors.New(errMsg) // Return an error to signal a problem.
	}

	// Call the use case. The ConfirmMatchStatus method (already refactored)
	// will handle proposals with MatchStatusPendingCustomerConfirmation correctly
	// by updating its state and publishing the match.pending_customer_confirmation event.
	err := h.matchUC.ConfirmMatchStatus(matchProposal) // matchID parameter was removed from usecase
	if err != nil {
		log.Printf("handleMatchAccept: Error from matchUC.ConfirmMatchStatus for matchID=%s: %v", matchProposal.ID, err)
		return err
	}

	log.Printf("handleMatchAccept: Successfully processed driver's initial acceptance for matchID=%s.", matchProposal.ID)
	return nil
}

// handleCustomerMatchConfirmed processes the customer's final confirmation of a match from User Service.
// Expects MatchProposal with status MatchStatusAccepted.
func (h *MatchHandler) handleCustomerMatchConfirmed(msg []byte) error {
	var matchProposal models.MatchProposal
	if err := json.Unmarshal(msg, &matchProposal); err != nil {
		log.Printf("handleCustomerMatchConfirmed: Failed to unmarshal MatchProposal: %v", err)
		return err
	}

	log.Printf("handleCustomerMatchConfirmed: Received customer's final confirmation for matchID=%s, status=%s.",
		matchProposal.ID, matchProposal.MatchStatus) // Simplified log

	// Retain existing warning for unexpected status, though primary validation is in use case.
	if matchProposal.MatchStatus != models.MatchStatusAccepted {
		log.Printf("handleCustomerMatchConfirmed: WARNING - Received status %s for match %s, expected %s. Forwarding to use case.",
			matchProposal.ID, matchProposal.MatchStatus, models.MatchStatusAccepted)
	}

	// The ConfirmMatchStatus use case expects mp.MatchStatus == models.MatchStatusAccepted
	// for this path, leading to final match persistence.
	err := h.matchUC.ConfirmMatchStatus(matchProposal) // matchID parameter was removed from usecase
	if err != nil {
		log.Printf("handleCustomerMatchConfirmed: Error from matchUC.ConfirmMatchStatus for matchID=%s: %v", matchProposal.ID, err)
		return err
	}

	log.Printf("handleCustomerMatchConfirmed: Successfully processed customer's final confirmation for matchID=%s.", matchProposal.ID)
	return nil
}

// handleCustomerMatchRejected processes the customer's rejection of a match from User Service.
// Expects MatchProposal with status MatchStatusRejected.
func (h *MatchHandler) handleCustomerMatchRejected(msg []byte) error {
	var matchProposal models.MatchProposal
	if err := json.Unmarshal(msg, &matchProposal); err != nil {
		log.Printf("handleCustomerMatchRejected: Failed to unmarshal MatchProposal: %v", err)
		return err
	}

	log.Printf("handleCustomerMatchRejected: Received customer's rejection for matchID=%s, status=%s.",
		matchProposal.ID, matchProposal.MatchStatus) // Simplified log

	// Retain existing warning for unexpected status.
	if matchProposal.MatchStatus != models.MatchStatusRejected {
		log.Printf("handleCustomerMatchRejected: WARNING - Received status %s for match %s, expected %s. Forwarding to use case.",
			matchProposal.ID, matchProposal.MatchStatus, models.MatchStatusRejected)
	}

	// The ConfirmMatchStatus use case handles mp.MatchStatus == models.MatchStatusRejected for cleanup.
	err := h.matchUC.ConfirmMatchStatus(matchProposal) // matchID parameter was removed from usecase
	if err != nil {
		log.Printf("handleCustomerMatchRejected: Error from matchUC.ConfirmMatchStatus for matchID=%s: %v", matchProposal.ID, err)
		return err
	}

	log.Printf("handleCustomerMatchRejected: Successfully processed customer's rejection for matchID=%s.", matchProposal.ID)
	return nil
}
