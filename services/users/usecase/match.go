package usecase

import (
	"context"
	"fmt"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// UpdateBeaconStatus updates a user's beacon status and location
func (uc *UserUC) ConfirmMatch(ctx context.Context, mp *models.MatchProposal, userID string) error {
	if mp.MatchStatus != models.MatchStatusAccepted {
		return fmt.Errorf("invalid match status: %s", mp.MatchStatus)
	}
	driver, err := uc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if driver.Role != "driver" {
		return fmt.Errorf("user %s is not a driver", mp.DriverID)
	}

	return uc.UserGW.MatchAccept(mp)
}

// HandleCustomerMatchDecision processes the customer's decision (accept/reject) on a match proposal
// and publishes this decision to the appropriate NATS subject via the User Gateway.
func (uc *UserUC) HandleCustomerMatchDecision(ctx context.Context, mp models.MatchProposal, natsSubject string) error {
	// Log the action
	// Note: mp.ID is the MatchID. mp.PassengerID is the customer making the decision.
	// mp.DriverID is also present. mp.MatchStatus has been set by the caller (WebSocket handler)
	// to either models.MatchStatusAccepted or models.MatchStatusRejected.
	// natsSubject corresponds to this status.

	payloadBytes, err := json.Marshal(mp)
	if err != nil {
		log.Printf("HandleCustomerMatchDecision: Error marshalling MatchProposal for match %s: %v", mp.ID, err)
		return fmt.Errorf("failed to marshal match proposal: %w", err)
	}

	// Delegate to a new method in UserGW to handle the actual NATS publication.
	// This keeps the NATS publishing logic encapsulated within the gateway layer.
	// We need to ensure UserGW interface and its implementation are updated to support this.
	// For this subtask, we are implementing the usecase method and assuming the gateway will be updated.

	log.Printf("HandleCustomerMatchDecision: Publishing customer decision for match %s to subject %s", mp.ID, natsSubject)

	// This is a conceptual call. The UserGW interface would need a method like:
	// PublishGenericEvent(ctx context.Context, subject string, payload []byte) error
	// Or more specific methods like:
	// PublishCustomerConfirmedEvent(ctx context.Context, mp models.MatchProposal) error
	// PublishCustomerRejectedEvent(ctx context.Context, mp models.MatchProposal) error
	//
	// Given the current UserGW structure, adding specific methods is more consistent.
	// Let's assume specific methods for now.

	switch natsSubject {
	case constants.SubjectCustomerMatchConfirmed:
		// Assuming UserGW will have a method like PublishCustomerConfirmedEvent
		// We will need to add this method to the UserGW interface and implement it.
		// err = uc.UserGW.PublishCustomerConfirmedEvent(ctx, mp)
		// For now, as UserGW doesn't have this, we'll log and return an error indicating missing GW method.
		// This will be addressed in the next step of modifying the gateway.
		log.Printf("TODO: Call UserGW.PublishCustomerConfirmedEvent for subject %s with payload: %s", natsSubject, string(payloadBytes))
		// To make this runnable without immediate GW changes, we'd need a generic publish on UserGW,
		// or this usecase would need direct NATS client access (which it doesn't have).
		// For the purpose of this step, we highlight what needs to be called.
		// The subtask asks to "Publish the JSON payload to the NATS subject".
		// Since uc.UserGW is the way to publish, and it doesn't have a generic publish or these specific methods yet,
		// this implementation points to the next required change (updating UserGW).

		// Let's assume, for the sake of fulfilling "publish ... to the NATS subject",
		// that UserGW needs a generic publish method for now.
		// This is a temporary assumption for this step if UserGW isn't updated simultaneously.
		// The ideal is specific methods on UserGW.

		// If UserGW had a method like `Publish(subject string, data []byte) error`
		// err = uc.UserGW.Publish(natsSubject, payloadBytes)
		// For now, we'll simulate this by acknowledging the need.
		// The previous step (websocket handler) calls this usecase method.
		// This usecase method IS the one responsible for ensuring publication.
		// If UserGW is the ONLY route to NATS, then UserGW MUST be enhanced.

		// The instruction "It needs access to the NATS producer (e.g., uc.natsProducer)" was a hint.
		// But UserUC has UserGW, not natsProducer directly.
		// So the publishing MUST go via UserGW.
		// The most straightforward way is to add specific methods to UserGW.
		// Let's assume those methods (PublishCustomerConfirmedEvent, PublishCustomerRejectedEvent)
		// will be added to UserGW and simply call them here.
		if mp.MatchStatus != models.MatchStatusAccepted {
			log.Printf("HandleCustomerMatchDecision: Mismatch! natsSubject is %s but MatchStatus is %s", natsSubject, mp.MatchStatus)
			return fmt.Errorf("natsSubject and MatchStatus mismatch for confirmation")
		}
		// This call will fail until UserGW is updated.
		err = uc.UserGW.PublishCustomerConfirmedEvent(ctx, mp) // Conceptual method
	case constants.SubjectCustomerMatchRejected:
		if mp.MatchStatus != models.MatchStatusRejected {
			log.Printf("HandleCustomerMatchDecision: Mismatch! natsSubject is %s but MatchStatus is %s", natsSubject, mp.MatchStatus)
			return fmt.Errorf("natsSubject and MatchStatus mismatch for rejection")
		}
		// This call will fail until UserGW is updated.
		err = uc.UserGW.PublishCustomerRejectedEvent(ctx, mp) // Conceptual method
	default:
		log.Printf("HandleCustomerMatchDecision: Unknown NATS subject: %s for match %s", natsSubject, mp.ID)
		return fmt.Errorf("unknown nats subject for customer match decision: %s", natsSubject)
	}

	if err != nil {
		log.Printf("HandleCustomerMatchDecision: Error publishing customer decision for match %s to NATS subject %s: %v", mp.ID, natsSubject, err)
		return fmt.Errorf("failed to publish customer match decision: %w", err)
	}

	log.Printf("HandleCustomerMatchDecision: Successfully published customer decision for match %s to NATS subject %s", mp.ID, natsSubject)
	return nil
}
