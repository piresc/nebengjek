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
	// natsSubject corresponds to this status. mp.MatchStatus has been set by the caller.
	var err error

	log.Printf("HandleCustomerMatchDecision: Processing customer decision for match %s, status %s, NATS subject %s",
		mp.ID, mp.MatchStatus, natsSubject)

	switch natsSubject {
	case constants.SubjectCustomerMatchConfirmed:
		if mp.MatchStatus != models.MatchStatusAccepted {
			log.Printf("HandleCustomerMatchDecision: Mismatch! natsSubject is %s but MatchStatus is %s for match %s",
				natsSubject, mp.MatchStatus, mp.ID)
			return fmt.Errorf("natsSubject and MatchStatus mismatch for confirmation")
		}
		err = uc.UserGW.PublishCustomerConfirmedEvent(ctx, mp)
	case constants.SubjectCustomerMatchRejected:
		if mp.MatchStatus != models.MatchStatusRejected {
			log.Printf("HandleCustomerMatchDecision: Mismatch! natsSubject is %s but MatchStatus is %s for match %s",
				natsSubject, mp.MatchStatus, mp.ID)
			return fmt.Errorf("natsSubject and MatchStatus mismatch for rejection")
		}
		err = uc.UserGW.PublishCustomerRejectedEvent(ctx, mp)
	default:
		log.Printf("HandleCustomerMatchDecision: Unknown NATS subject: %s for match %s", natsSubject, mp.ID)
		return fmt.Errorf("unknown NATS subject for customer match decision: %s", natsSubject)
	}

	if err != nil {
		// Error from UserGateway methods already includes context like "failed to publish..."
		log.Printf("HandleCustomerMatchDecision: Error from UserGateway for match %s, subject %s: %v", mp.ID, natsSubject, err)
		return err // Return the error from the gateway directly
	}

	log.Printf("HandleCustomerMatchDecision: Successfully published customer decision for match %s to NATS subject %s", mp.ID, natsSubject)
	return nil
}
