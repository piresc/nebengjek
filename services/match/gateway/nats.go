package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// PublishMatchFound publishes a beacon event to NATS
func (g *matchGW) PublishMatchFound(ctx context.Context, matchProp models.MatchProposal) error {
	fmt.Printf("Publishing match event to NATS driver %s, passenger %s\n", matchProp.DriverID, matchProp.PassengerID)
	fmt.Printf("Match event: %+v\n", matchProp)
	data, err := json.Marshal(matchProp)
	if err != nil {
		return err
	}
	return g.nc.Publish(constants.SubjectMatchFound, data)
}

// PublishMatchAccept publishes a beacon event to NATS
func (g *matchGW) PublishMatchConfirm(ctx context.Context, matchProp models.MatchProposal) error {
	fmt.Printf("Publishing match event to NATS driver %s, passenger %s\n", matchProp.DriverID, matchProp.PassengerID)
	fmt.Printf("Match event: %+v\n", matchProp)
	data, err := json.Marshal(matchProp)
	if err != nil {
		return err
	}
	return g.nc.Publish(constants.SubjectMatchConfirm, data)
}

// PublishMatchRejected publishes a beacon event to NATS
func (g *matchGW) PublishMatchRejected(ctx context.Context, matchProp models.MatchProposal) error {
	fmt.Printf("Publishing match event to NATS driver %s, passenger %s\n", matchProp.DriverID, matchProp.PassengerID)
	fmt.Printf("Match event: %+v\n", matchProp)
	data, err := json.Marshal(matchProp)
	if err != nil {
		return err
	}
	return g.nc.Publish(constants.SubjectMatchRejected, data)
}
