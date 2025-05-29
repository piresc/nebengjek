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
