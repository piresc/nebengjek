package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/location"
)

type locationGW struct {
	natsClient *natspkg.Client
}

// NewLocationGW creates a new location gateway
func NewLocationGW(client *natspkg.Client) location.LocationGW {
	return &locationGW{
		natsClient: client,
	}
}

// PublishLocationAggregate publishes a location aggregate event to NATS
func (g *locationGW) PublishLocationAggregate(ctx context.Context, aggregate models.LocationAggregate) error {
	data, err := json.Marshal(aggregate)
	if err != nil {
		return fmt.Errorf("failed to marshal location aggregate: %w", err)
	}

	return g.natsClient.Publish(constants.SubjectLocationAggregate, data)
}
