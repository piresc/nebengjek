package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	locationmodels "github.com/piresc/nebengjek/internal/pkg/models/location"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	locationsvc "github.com/piresc/nebengjek/services/location"
	)

// NATSPublisher defines the interface for NATS publishing operations
type NATSPublisher interface {
	Publish(subject string, data []byte) error
	PublishWithOptions(opts natspkg.PublishOptions) error
	GetConn() *nats.Conn
	Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error)
	Close()
}

type locationGW struct {
	natsClient NATSPublisher
}

// NewLocationGW creates a new location gateway
func NewLocationGW(client NATSPublisher) locationsvc.LocationGW {
	return &locationGW{
		natsClient: client,
	}
}


// PublishLocationAggregate publishes a location aggregate event to JetStream with delivery guarantees
func (g *locationGW) PublishLocationAggregate(ctx context.Context, aggregate locationmodels.LocationAggregate) error {
	data, err := json.Marshal(aggregate)
	if err != nil {
		return fmt.Errorf("failed to marshal location aggregate: %w", err)
	}

	// Use JetStream publish with options for reliability
	opts := natspkg.PublishOptions{
		Subject: natspkg.SubjectLocationAggregate,
		Data:    data,
		MsgID:   fmt.Sprintf("location-aggregate-%s-%d", aggregate.RideID, time.Now().UnixNano()),
		Timeout: 10 * time.Second,
	}

	if err := g.natsClient.PublishWithOptions(opts); err != nil {
		logger.ErrorCtx(ctx, "Failed to publish location aggregate to JetStream",
			logger.String("ride_id", aggregate.RideID),
			logger.Float64("distance", aggregate.Distance),
			logger.Err(err))
		return fmt.Errorf("failed to publish location aggregate: %w", err)
	}

	logger.InfoCtx(ctx, "Successfully published location aggregate to JetStream",
		logger.String("ride_id", aggregate.RideID),
		logger.Float64("distance", aggregate.Distance),
		logger.Float64("latitude", aggregate.Latitude),
		logger.Float64("longitude", aggregate.Longitude))

	return nil
}
