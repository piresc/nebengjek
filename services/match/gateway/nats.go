package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log" // Imported log package

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/match"
)

// matchGW handles match gateway operations
type matchGW struct {
	natsClient *natspkg.Client
}

// NewMatchGW creates a new NATS gateway instance
func NewMatchGW(client *natspkg.Client) match.MatchGW {
	return &matchGW{
		natsClient: client,
	}
}

// PublishMatchFound publishes a match found event to NATS.
func (g *matchGW) PublishMatchFound(ctx context.Context, matchProp models.MatchProposal) error {
	subject := constants.SubjectMatchFound
	log.Printf("Publishing MatchFound event to NATS for MatchID %s (Driver: %s, Passenger: %s) to subject %s",
		matchProp.ID, matchProp.DriverID, matchProp.PassengerID, subject)
	log.Printf("MatchFound event details: %+v", matchProp)

	data, err := json.Marshal(matchProp)
	if err != nil {
		log.Printf("Error marshalling MatchFound event for MatchID %s: %v", matchProp.ID, err)
		return fmt.Errorf("failed to marshal MatchFound event for match %s: %w", matchProp.ID, err)
	}

	if err := g.natsClient.Publish(subject, data); err != nil {
		log.Printf("Error publishing MatchFound event for MatchID %s to NATS subject %s: %v", matchProp.ID, subject, err)
		return fmt.Errorf("failed to publish MatchFound event for match %s to subject %s: %w", matchProp.ID, subject, err)
	}

	log.Printf("Successfully published MatchFound event for MatchID %s to subject %s", matchProp.ID, subject)
	return nil
}

// PublishMatchConfirm publishes a match confirmation event to NATS.
func (g *matchGW) PublishMatchConfirm(ctx context.Context, matchProp models.MatchProposal) error {
	subject := constants.SubjectMatchConfirm
	log.Printf("Publishing MatchConfirm event to NATS for MatchID %s (Driver: %s, Passenger: %s) to subject %s",
		matchProp.ID, matchProp.DriverID, matchProp.PassengerID, subject)
	log.Printf("MatchConfirm event details: %+v", matchProp)

	data, err := json.Marshal(matchProp)
	if err != nil {
		log.Printf("Error marshalling MatchConfirm event for MatchID %s: %v", matchProp.ID, err)
		return fmt.Errorf("failed to marshal MatchConfirm event for match %s: %w", matchProp.ID, err)
	}

	if err := g.natsClient.Publish(subject, data); err != nil {
		log.Printf("Error publishing MatchConfirm event for MatchID %s to NATS subject %s: %v", matchProp.ID, subject, err)
		return fmt.Errorf("failed to publish MatchConfirm event for match %s to subject %s: %w", matchProp.ID, subject, err)
	}

	log.Printf("Successfully published MatchConfirm event for MatchID %s to subject %s", matchProp.ID, subject)
	return nil
}

// PublishMatchRejected publishes a match rejection event to NATS.
func (g *matchGW) PublishMatchRejected(ctx context.Context, matchProp models.MatchProposal) error {
	subject := constants.SubjectMatchRejected
	log.Printf("Publishing MatchRejected event to NATS for MatchID %s (Driver: %s, Passenger: %s) to subject %s",
		matchProp.ID, matchProp.DriverID, matchProp.PassengerID, subject)
	log.Printf("MatchRejected event details: %+v", matchProp)

	data, err := json.Marshal(matchProp)
	if err != nil {
		log.Printf("Error marshalling MatchRejected event for MatchID %s: %v", matchProp.ID, err)
		return fmt.Errorf("failed to marshal MatchRejected event for match %s: %w", matchProp.ID, err)
	}

	if err := g.natsClient.Publish(subject, data); err != nil {
		log.Printf("Error publishing MatchRejected event for MatchID %s to NATS subject %s: %v", matchProp.ID, subject, err)
		return fmt.Errorf("failed to publish MatchRejected event for match %s to subject %s: %w", matchProp.ID, subject, err)
	}

	log.Printf("Successfully published MatchRejected event for MatchID %s to subject %s", matchProp.ID, subject)
	return nil
}

// PublishMatchPendingCustomerConfirmation publishes a match pending customer confirmation event to NATS.
func (g *matchGW) PublishMatchPendingCustomerConfirmation(ctx context.Context, mp models.MatchProposal) error {
	subject := constants.SubjectMatchPendingCustomerConfirmation
	log.Printf("Publishing MatchPendingCustomerConfirmation event to NATS for MatchID %s (Driver: %s, Passenger: %s) to subject %s",
		mp.ID, mp.DriverID, mp.PassengerID, subject)
	log.Printf("MatchPendingCustomerConfirmation event details: %+v", mp)

	data, err := json.Marshal(mp)
	if err != nil {
		log.Printf("Error marshalling MatchPendingCustomerConfirmation event for MatchID %s: %v", mp.ID, err)
		return fmt.Errorf("failed to marshal MatchPendingCustomerConfirmation event for match %s: %w", mp.ID, err)
	}

	if err := g.natsClient.Publish(subject, data); err != nil {
		log.Printf("Error publishing MatchPendingCustomerConfirmation event for MatchID %s to NATS subject %s: %v", mp.ID, subject, err)
		return fmt.Errorf("failed to publish MatchPendingCustomerConfirmation event for match %s to subject %s: %w", mp.ID, subject, err)
	}

	log.Printf("Successfully published MatchPendingCustomerConfirmation event for MatchID %s to subject %s", mp.ID, subject)
	return nil
}
