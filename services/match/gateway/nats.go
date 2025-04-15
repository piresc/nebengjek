package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// PublishBeaconEvent publishes a beacon event to NATS
func (g *matchGW) PublishMatchEvent(ctx context.Context, matchProp models.MatchProposal) error {
	fmt.Printf("Publishing match event to NATS driver %s, passenger %s\n", matchProp.DriverID, matchProp.PassengerID)
	fmt.Printf("Match event: %+v\n", matchProp)
	data, err := json.Marshal(matchProp)
	if err != nil {
		return err
	}
	return g.nc.Publish("match.found", data)
}
