package gateway

import (
	"context"
	"encoding/json"
	"fmt"

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

// PublishMatchFound publishes a beacon event to NATS
func (g *matchGW) PublishMatchFound(ctx context.Context, matchProp models.MatchProposal) error {
	fmt.Printf("Publishing match event to NATS driver %s, passenger %s\n", matchProp.DriverID, matchProp.PassengerID)
	fmt.Printf("Match event: %+v\n", matchProp)
	data, err := json.Marshal(matchProp)
	if err != nil {
		return err
	}
	return g.natsClient.Publish(constants.SubjectMatchFound, data)
}

// PublishMatchAccept publishes a beacon event to NATS
func (g *matchGW) PublishMatchConfirm(ctx context.Context, matchProp models.MatchProposal) error {
	fmt.Printf("Publishing match event to NATS driver %s, passenger %s\n", matchProp.DriverID, matchProp.PassengerID)
	fmt.Printf("Match event: %+v\n", matchProp)
	data, err := json.Marshal(matchProp)
	if err != nil {
		return err
	}
	return g.natsClient.Publish(constants.SubjectMatchConfirm, data)
}

// PublishMatchRejected publishes a beacon event to NATS
func (g *matchGW) PublishMatchRejected(ctx context.Context, matchProp models.MatchProposal) error {
	fmt.Printf("Publishing match event to NATS driver %s, passenger %s\n", matchProp.DriverID, matchProp.PassengerID)
	fmt.Printf("Match event: %+v\n", matchProp)
	data, err := json.Marshal(matchProp)
	if err != nil {
		return err
	}
	return g.natsClient.Publish(constants.SubjectMatchRejected, data)
}

// PublishMatchPendingCustomerConfirmation publishes a match pending customer confirmation event to NATS.
func (g *matchGW) PublishMatchPendingCustomerConfirmation(ctx context.Context, mp models.MatchProposal) error {
	fmt.Printf("Publishing match pending customer confirmation to NATS: MatchID %s, DriverID %s, PassengerID %s\n", mp.ID, mp.DriverID, mp.PassengerID)
	fmt.Printf("Match pending customer confirmation event: %+v\n", mp)

	data, err := json.Marshal(mp)
	if err != nil {
		fmt.Printf("Error marshalling match pending customer confirmation event for MatchID %s: %v\n", mp.ID, err)
		return err
	}

	if err := g.natsClient.Publish(constants.SubjectMatchPendingCustomerConfirmation, data); err != nil {
		fmt.Printf("Error publishing match pending customer confirmation event for MatchID %s to NATS: %v\n", mp.ID, err)
		return err
	}

	fmt.Printf("Successfully published match pending customer confirmation for MatchID %s\n", mp.ID)
	return nil
}
