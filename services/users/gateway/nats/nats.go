package gateway_nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
)

// NATSGateway implements the NATS gateway operations for the users service
type NATSGateway struct {
	client *natspkg.Client
}

// NewNATSGateway creates a new NATS gateway
func NewNATSGateway(client *natspkg.Client) *NATSGateway {
	return &NATSGateway{
		client: client,
	}
}

// PublishBeaconEvent publishes a beacon event to JetStream with delivery guarantees
func (g *NATSGateway) PublishBeaconEvent(ctx context.Context, event *models.BeaconEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal beacon event: %w", err)
	}

	// Use JetStream publish with options for reliability
	opts := natspkg.PublishOptions{
		Subject: constants.SubjectUserBeacon,
		Data:    data,
		MsgID:   fmt.Sprintf("beacon-%s-%d", event.UserID, time.Now().UnixNano()),
		Timeout: 10 * time.Second,
	}

	if err := g.client.PublishWithOptions(opts); err != nil {
		logger.ErrorCtx(ctx, "Failed to publish beacon event to JetStream",
			logger.String("user_id", event.UserID),
			logger.Err(err))
		return fmt.Errorf("failed to publish beacon event: %w", err)
	}

	logger.InfoCtx(ctx, "Successfully published beacon event to JetStream",
		logger.String("user_id", event.UserID),
		logger.Bool("is_active", event.IsActive))

	return nil
}

// PublishFinderEvent publishes a finder event to JetStream with delivery guarantees
func (g *NATSGateway) PublishFinderEvent(ctx context.Context, event *models.FinderEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal finder event: %w", err)
	}

	// Use JetStream publish with options for reliability
	opts := natspkg.PublishOptions{
		Subject: constants.SubjectUserFinder,
		Data:    data,
		MsgID:   fmt.Sprintf("finder-%s-%d", event.UserID, time.Now().UnixNano()),
		Timeout: 10 * time.Second,
	}

	if err := g.client.PublishWithOptions(opts); err != nil {
		logger.ErrorCtx(ctx, "Failed to publish finder event to JetStream",
			logger.String("user_id", event.UserID),
			logger.Err(err))
		return fmt.Errorf("failed to publish finder event: %w", err)
	}

	logger.InfoCtx(ctx, "Successfully published finder event to JetStream",
		logger.String("user_id", event.UserID),
		logger.Bool("is_active", event.IsActive))

	return nil
}
