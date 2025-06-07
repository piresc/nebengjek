package nats

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

// NATSGateway handles NATS gateway operations
type NATSGateway struct {
	natsClient *natspkg.Client
}

// NewNATSGateway creates a new NATS gateway instance
func NewNATSGateway(client *natspkg.Client) *NATSGateway {
	return &NATSGateway{
		natsClient: client,
	}
}

// PublishMatchFound publishes a match found event to JetStream with delivery guarantees
func (g *NATSGateway) PublishMatchFound(ctx context.Context, matchProp models.MatchProposal) error {
	data, err := json.Marshal(matchProp)
	if err != nil {
		return fmt.Errorf("failed to marshal match proposal: %w", err)
	}

	// Use JetStream publish with options for reliability
	opts := natspkg.PublishOptions{
		Subject: constants.SubjectMatchFound,
		Data:    data,
		MsgID:   fmt.Sprintf("match-found-%s-%d", matchProp.ID, time.Now().UnixNano()),
		Timeout: 10 * time.Second,
	}

	if err := g.natsClient.PublishWithOptions(opts); err != nil {
		logger.ErrorCtx(ctx, "Failed to publish match found event to JetStream",
			logger.String("match_id", matchProp.ID),
			logger.String("driver_id", matchProp.DriverID),
			logger.String("passenger_id", matchProp.PassengerID),
			logger.Err(err))
		return fmt.Errorf("failed to publish match found event: %w", err)
	}

	logger.InfoCtx(ctx, "Successfully published match found event to JetStream",
		logger.String("match_id", matchProp.ID),
		logger.String("driver_id", matchProp.DriverID),
		logger.String("passenger_id", matchProp.PassengerID))

	return nil
}

// PublishMatchRejected publishes a match rejected event to JetStream with delivery guarantees
func (g *NATSGateway) PublishMatchRejected(ctx context.Context, matchProp models.MatchProposal) error {
	data, err := json.Marshal(matchProp)
	if err != nil {
		return fmt.Errorf("failed to marshal match proposal: %w", err)
	}

	// Use JetStream publish with options for reliability
	opts := natspkg.PublishOptions{
		Subject: constants.SubjectMatchRejected,
		Data:    data,
		MsgID:   fmt.Sprintf("match-rejected-%s-%d", matchProp.ID, time.Now().UnixNano()),
		Timeout: 10 * time.Second,
	}

	if err := g.natsClient.PublishWithOptions(opts); err != nil {
		logger.ErrorCtx(ctx, "Failed to publish match rejected event to JetStream",
			logger.String("match_id", matchProp.ID),
			logger.String("driver_id", matchProp.DriverID),
			logger.String("passenger_id", matchProp.PassengerID),
			logger.Err(err))
		return fmt.Errorf("failed to publish match rejected event: %w", err)
	}

	logger.InfoCtx(ctx, "Successfully published match rejected event to JetStream",
		logger.String("match_id", matchProp.ID),
		logger.String("driver_id", matchProp.DriverID),
		logger.String("passenger_id", matchProp.PassengerID))

	return nil
}

// PublishMatchAccepted publishes a match accepted event to JetStream with delivery guarantees
func (g *NATSGateway) PublishMatchAccepted(ctx context.Context, matchProp models.MatchProposal) error {
	logger.InfoCtx(ctx, "Preparing to publish match accepted event to JetStream",
		logger.String("match_id", matchProp.ID),
		logger.String("driver_id", matchProp.DriverID),
		logger.String("passenger_id", matchProp.PassengerID))

	data, err := json.Marshal(matchProp)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to marshal match proposal for JetStream",
			logger.String("match_id", matchProp.ID),
			logger.ErrorField(err))
		return fmt.Errorf("failed to marshal match proposal: %w", err)
	}

	// Use JetStream publish with options for reliability - higher retry for critical match events
	opts := natspkg.PublishOptions{
		Subject: constants.SubjectMatchAccepted,
		Data:    data,
		MsgID:   fmt.Sprintf("match-accepted-%s-%d", matchProp.ID, time.Now().UnixNano()),
		Timeout: 15 * time.Second, // Longer timeout for critical match acceptance
	}

	logger.InfoCtx(ctx, "Publishing match accepted event to JetStream",
		logger.String("subject", opts.Subject),
		logger.String("msg_id", opts.MsgID),
		logger.String("message_size", fmt.Sprintf("%d bytes", len(data))))

	if err := g.natsClient.PublishWithOptions(opts); err != nil {
		logger.ErrorCtx(ctx, "Failed to publish match accepted event to JetStream",
			logger.String("match_id", matchProp.ID),
			logger.String("driver_id", matchProp.DriverID),
			logger.String("passenger_id", matchProp.PassengerID),
			logger.String("subject", opts.Subject),
			logger.String("msg_id", opts.MsgID),
			logger.Err(err))
		return fmt.Errorf("failed to publish match accepted event: %w", err)
	}

	logger.InfoCtx(ctx, "Successfully published match accepted event to JetStream",
		logger.String("match_id", matchProp.ID),
		logger.String("driver_id", matchProp.DriverID),
		logger.String("passenger_id", matchProp.PassengerID),
		logger.String("subject", opts.Subject),
		logger.String("msg_id", opts.MsgID))

	return nil
}
