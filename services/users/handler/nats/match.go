package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
)

// InitMatchConsumers initializes JetStream consumers for match-related events
func (h *NatsHandler) initMatchConsumers() error {
	logger.Info("Initializing JetStream consumers for match events in users service")

	// Create JetStream consumers for match events
	consumerConfigs := natspkg.DefaultConsumerConfigs()

	// Create match found consumer
	matchFoundConfig := consumerConfigs["match_found_users"]
	logger.Info("Creating match found consumer for users service",
		logger.String("stream", matchFoundConfig.StreamName),
		logger.String("consumer", matchFoundConfig.ConsumerName))

	if err := h.natsClient.CreateConsumer(matchFoundConfig); err != nil {
		logger.Error("Failed to create match found consumer for users service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to create match found consumer: %w", err)
	}

	// Start consuming match found events
	if err := h.natsClient.ConsumeMessages("MATCH_STREAM", "match_found_users", h.handleMatchEventJS); err != nil {
		logger.Error("Failed to start consuming match found events for users service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to start consuming match found events: %w", err)
	}

	// Create match accepted consumer
	matchAcceptedConfig := consumerConfigs["match_accepted_users"]
	logger.Info("Creating match accepted consumer for users service",
		logger.String("stream", matchAcceptedConfig.StreamName),
		logger.String("consumer", matchAcceptedConfig.ConsumerName))

	if err := h.natsClient.CreateConsumer(matchAcceptedConfig); err != nil {
		logger.Error("Failed to create match accepted consumer for users service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to create match accepted consumer: %w", err)
	}

	// Start consuming match accepted events
	if err := h.natsClient.ConsumeMessages("MATCH_STREAM", "match_accepted_users", h.handleMatchAccEventJS); err != nil {
		logger.Error("Failed to start consuming match accepted events for users service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to start consuming match accepted events: %w", err)
	}

	// Create match rejected consumer
	matchRejectedConfig := consumerConfigs["match_rejected_users"]
	logger.Info("Creating match rejected consumer for users service",
		logger.String("stream", matchRejectedConfig.StreamName),
		logger.String("consumer", matchRejectedConfig.ConsumerName))

	if err := h.natsClient.CreateConsumer(matchRejectedConfig); err != nil {
		logger.Error("Failed to create match rejected consumer for users service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to create match rejected consumer: %w", err)
	}

	// Start consuming match rejected events
	if err := h.natsClient.ConsumeMessages("MATCH_STREAM", "match_rejected_users", h.handleMatchRejectedEventJS); err != nil {
		logger.Error("Failed to start consuming match rejected events for users service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to start consuming match rejected events: %w", err)
	}

	logger.Info("Successfully initialized JetStream consumers for match events in users service")
	return nil
}

// JetStream message handlers with proper acknowledgment

// handleMatchEventJS processes match events from JetStream
func (h *NatsHandler) handleMatchEventJS(msg jetstream.Msg) error {
	logger.InfoCtx(context.Background(), "Received match event from JetStream",
		logger.String("subject", msg.Subject()),
		logger.String("data", string(msg.Data())))

	if err := h.handleMatchEvent(msg.Data()); err != nil {
		logger.ErrorCtx(context.Background(), "Error handling match event", logger.Err(err))
		return err // Return error to trigger NAK and retry
	}

	return nil // Success - message will be ACKed automatically
}

// handleMatchAccEventJS processes match accepted events from JetStream
func (h *NatsHandler) handleMatchAccEventJS(msg jetstream.Msg) error {
	logger.InfoCtx(context.Background(), "Received match accept event from JetStream",
		logger.String("subject", msg.Subject()),
		logger.String("data", string(msg.Data())))

	if err := h.handleMatchAccEvent(msg.Data()); err != nil {
		logger.ErrorCtx(context.Background(), "Error handling match accept event", logger.Err(err))
		return err // Return error to trigger NAK and retry
	}

	return nil // Success - message will be ACKed automatically
}

// handleMatchRejectedEventJS processes match rejected events from JetStream
func (h *NatsHandler) handleMatchRejectedEventJS(msg jetstream.Msg) error {
	logger.InfoCtx(context.Background(), "Received match rejected event from JetStream",
		logger.String("subject", msg.Subject()))

	if err := h.handleMatchRejectedEvent(msg.Data()); err != nil {
		logger.ErrorCtx(context.Background(), "Error handling match rejected event", logger.Err(err))
		return err // Return error to trigger NAK and retry
	}

	return nil // Success - message will be ACKed automatically
}

// handleMatchEvent processes match events
func (h *NatsHandler) handleMatchEvent(msg []byte) error {
	var event models.MatchProposal
	if err := json.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("failed to unmarshal match event: %w", err)
	}

	// Notify both driver and passenger
	h.wsManager.NotifyClient(event.DriverID, constants.SubjectMatchFound, event)
	h.wsManager.NotifyClient(event.PassengerID, constants.SubjectMatchFound, event)
	return nil
}

// handleMatchEvent processes match events
func (h *NatsHandler) handleMatchAccEvent(msg []byte) error {
	var event models.MatchProposal
	if err := json.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("failed to unmarshal match event: %w", err)
	}

	// Notify both driver and passenger
	h.wsManager.NotifyClient(event.DriverID, constants.SubjectMatchAccepted, event)
	h.wsManager.NotifyClient(event.PassengerID, constants.SubjectMatchAccepted, event)
	return nil
}

// handleMatchRejectedEvent processes match rejected events
func (h *NatsHandler) handleMatchRejectedEvent(msg []byte) error {
	var event models.MatchProposal
	if err := json.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("failed to unmarshal match rejected event: %w", err)
	}

	// Only notify the driver whose match was rejected
	h.wsManager.NotifyClient(event.DriverID, constants.EventMatchRejected, event)
	return nil
}
