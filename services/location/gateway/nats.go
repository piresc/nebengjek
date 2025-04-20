package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/location"
)

type locationGW struct {
	nc *nats.Conn
}

// NewLocationGW creates a new location gateway
func NewLocationGW(nc *nats.Conn) location.LocationGW {
	return &locationGW{
		nc: nc,
	}
}

// PublishLocationAggregate publishes a location aggregate event to NATS
func (g *locationGW) PublishLocationAggregate(ctx context.Context, aggregate models.LocationAggregate) error {
	data, err := json.Marshal(aggregate)
	if err != nil {
		return fmt.Errorf("failed to marshal location aggregate: %w", err)
	}

	return g.nc.Publish(constants.SubjectLocationAggregate, data)
}
