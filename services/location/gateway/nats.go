package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/location"
)

// NATSPublisher defines the interface for NATS publishing operations
type NATSPublisher interface {
	Publish(subject string, data []byte) error
	GetConn() *nats.Conn
	Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error)
	Close()
}

type locationGW struct {
	natsClient NATSPublisher
}

// NewLocationGW creates a new location gateway
func NewLocationGW(client NATSPublisher) location.LocationGW {
	return &locationGW{
		natsClient: client,
	}
}

// NewLocationGWWithClient creates a new location gateway with a concrete NATS client
func NewLocationGWWithClient(client *natspkg.Client) location.LocationGW {
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
