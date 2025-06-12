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

// InitRideConsumers initializes JetStream consumers for ride-related events
func (h *NatsHandler) initRideConsumers() error {
	logger.Info("Initializing JetStream consumers for ride events")

	// Create JetStream consumers for ride events
	consumerConfigs := natspkg.DefaultConsumerConfigs()

	// Create ride pickup consumer - RECREATE to ensure DeliverNewPolicy is applied
	ridePickupConfig := consumerConfigs["ride_pickup_users"]
	logger.Info("Recreating ride pickup consumer for users service with DeliverNewPolicy",
		logger.String("stream", ridePickupConfig.StreamName),
		logger.String("consumer", ridePickupConfig.ConsumerName),
		logger.String("filter_subject", ridePickupConfig.FilterSubject),
		logger.String("deliver_policy", "DeliverNewPolicy"))

	if err := h.natsClient.RecreateConsumer(ridePickupConfig); err != nil {
		logger.Error("Failed to recreate ride pickup consumer for users service",
			logger.String("consumer", ridePickupConfig.ConsumerName),
			logger.ErrorField(err))
		return fmt.Errorf("failed to recreate ride pickup consumer: %w", err)
	}

	// Start consuming ride pickup events
	logger.Info("Starting to consume ride pickup events for users service",
		logger.String("stream", "RIDE_STREAM"),
		logger.String("consumer", "ride_pickup_users"))

	if err := h.natsClient.ConsumeMessages("RIDE_STREAM", "ride_pickup_users", h.handleRidePickupEventJS); err != nil {
		logger.Error("Failed to start consuming ride pickup events for users service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to start consuming ride pickup events: %w", err)
	}

	// Create ride started consumer - RECREATE to ensure DeliverNewPolicy is applied
	rideStartedConfig := consumerConfigs["ride_started_users"]
	logger.Info("Recreating ride started consumer for users service with DeliverNewPolicy",
		logger.String("stream", rideStartedConfig.StreamName),
		logger.String("consumer", rideStartedConfig.ConsumerName),
		logger.String("deliver_policy", "DeliverNewPolicy"))

	if err := h.natsClient.RecreateConsumer(rideStartedConfig); err != nil {
		logger.Error("Failed to recreate ride started consumer for users service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to recreate ride started consumer: %w", err)
	}

	// Start consuming ride started events
	if err := h.natsClient.ConsumeMessages("RIDE_STREAM", "ride_started_users", h.handleRideStartEventJS); err != nil {
		logger.Error("Failed to start consuming ride started events for users service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to start consuming ride started events: %w", err)
	}

	// Create ride completed consumer - RECREATE to ensure DeliverNewPolicy is applied
	rideCompletedConfig := consumerConfigs["ride_completed_users"]
	logger.Info("Recreating ride completed consumer for users service with DeliverNewPolicy",
		logger.String("stream", rideCompletedConfig.StreamName),
		logger.String("consumer", rideCompletedConfig.ConsumerName),
		logger.String("deliver_policy", "DeliverNewPolicy"))

	if err := h.natsClient.RecreateConsumer(rideCompletedConfig); err != nil {
		logger.Error("Failed to recreate ride completed consumer for users service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to recreate ride completed consumer: %w", err)
	}

	// Start consuming ride completed events
	if err := h.natsClient.ConsumeMessages("RIDE_STREAM", "ride_completed_users", h.handleRideCompletedEventJS); err != nil {
		logger.Error("Failed to start consuming ride completed events for users service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to start consuming ride completed events: %w", err)
	}

	logger.Info("Successfully initialized JetStream consumers for ride events")
	return nil
}

// JetStream message handlers with proper acknowledgment

// handleRidePickupEventJS processes ride pickup events from JetStream
func (h *NatsHandler) handleRidePickupEventJS(msg jetstream.Msg) error {
	logger.InfoCtx(context.Background(), "Received ride pickup event from JetStream",
		logger.String("subject", msg.Subject()))

	if err := h.handleRidePickupEvent(msg.Data()); err != nil {
		logger.ErrorCtx(context.Background(), "Error handling ride pickup event", logger.Err(err))
		return err // Return error to trigger NAK and retry
	}

	return nil // Success - message will be ACKed automatically
}

// handleRideStartEventJS processes ride start events from JetStream
func (h *NatsHandler) handleRideStartEventJS(msg jetstream.Msg) error {
	logger.InfoCtx(context.Background(), "Received ride start event from JetStream",
		logger.String("subject", msg.Subject()))

	if err := h.handleRideStartEvent(msg.Data()); err != nil {
		logger.ErrorCtx(context.Background(), "Error handling ride start event", logger.Err(err))
		return err // Return error to trigger NAK and retry
	}

	return nil // Success - message will be ACKed automatically
}

// handleRideCompletedEventJS processes ride completed events from JetStream
func (h *NatsHandler) handleRideCompletedEventJS(msg jetstream.Msg) error {
	// DIAGNOSTIC: Log message metadata to understand replay behavior
	metadata, _ := msg.Metadata()
	logger.InfoCtx(context.Background(), "Received ride completed event from JetStream",
		logger.String("subject", msg.Subject()),
		logger.String("stream_sequence", fmt.Sprintf("%d", metadata.Sequence.Stream)),
		logger.String("consumer_sequence", fmt.Sprintf("%d", metadata.Sequence.Consumer)),
		logger.String("timestamp", metadata.Timestamp.Format("2006-01-02T15:04:05Z07:00")),
		logger.Int("num_delivered", int(metadata.NumDelivered)),
		logger.String("is_replay", fmt.Sprintf("%t", metadata.NumDelivered > 1)))

	if err := h.handleRideCompletedEvent(msg.Data()); err != nil {
		logger.ErrorCtx(context.Background(), "Error handling ride completed event", logger.Err(err))
		return err // Return error to trigger NAK and retry
	}

	return nil // Success - message will be ACKed automatically
}

// handleMatchAcceptedEvent processes match accepted events from NATS
func (h *NatsHandler) handleMatchAcceptedEvent(msg []byte) error {
	var matchProposal models.MatchProposal
	if err := json.Unmarshal(msg, &matchProposal); err != nil {
		return fmt.Errorf("failed to unmarshal match accepted event: %w", err)
	}

	logger.InfoCtx(context.Background(), "Received match accepted event",
		logger.String("match_id", matchProposal.ID),
		logger.String("driver_id", matchProposal.DriverID),
		logger.String("passenger_id", matchProposal.PassengerID))

	// Notify both driver and passenger that their match is confirmed and they're locked
	// Use a specific event type for match acceptance notification
	h.echoWSHandler.NotifyClient(matchProposal.DriverID, constants.EventMatchConfirm, matchProposal)
	h.echoWSHandler.NotifyClient(matchProposal.PassengerID, constants.EventMatchConfirm, matchProposal)

	return nil
}

// handleRidePickupEvent processes ride pickup events
func (h *NatsHandler) handleRidePickupEvent(msg []byte) error {
	logger.InfoCtx(context.Background(), "Processing ride pickup event from JetStream",
		logger.String("message_size", fmt.Sprintf("%d bytes", len(msg))))

	var ridePickup models.RideResp
	if err := json.Unmarshal(msg, &ridePickup); err != nil {
		logger.ErrorCtx(context.Background(), "Failed to unmarshal ride pickup event",
			logger.String("raw_message", string(msg)),
			logger.ErrorField(err))
		return fmt.Errorf("failed to unmarshal ride pickup event: %w", err)
	}

	logger.InfoCtx(context.Background(), "Successfully parsed ride pickup event",
		logger.String("ride_id", ridePickup.RideID),
		logger.String("driver_id", ridePickup.DriverID),
		logger.String("passenger_id", ridePickup.PassengerID),
		logger.String("status", ridePickup.Status))

	logger.InfoCtx(context.Background(), "Sending WebSocket notifications for ride pickup",
		logger.String("driver_id", ridePickup.DriverID),
		logger.String("passenger_id", ridePickup.PassengerID),
		logger.String("event_type", constants.EventRidePickup))

	// Notify both driver and passenger with correct WebSocket event type
	h.echoWSHandler.NotifyClient(ridePickup.DriverID, constants.EventRidePickup, ridePickup)
	h.echoWSHandler.NotifyClient(ridePickup.PassengerID, constants.EventRidePickup, ridePickup)

	logger.InfoCtx(context.Background(), "Successfully processed ride pickup event and sent WebSocket notifications",
		logger.String("ride_id", ridePickup.RideID))
	return nil
}

// handleMatchAcceptedEvent processes match accepted events from NATS
func (h *NatsHandler) handleRideStartEvent(msg []byte) error {
	var rideStarted models.RideResp
	if err := json.Unmarshal(msg, &rideStarted); err != nil {
		return fmt.Errorf("failed to unmarshal ride start event: %w", err)
	}

	logger.InfoCtx(context.Background(), "Received ride started event",
		logger.String("ride_id", rideStarted.RideID),
		logger.String("driver_id", rideStarted.DriverID),
		logger.String("passenger_id", rideStarted.PassengerID))

	// Notify both driver and passenger that their match is confirmed and they're locked
	// Use a specific event type for match acceptance notification
	h.echoWSHandler.NotifyClient(rideStarted.DriverID, constants.EventRideStarted, rideStarted)
	h.echoWSHandler.NotifyClient(rideStarted.PassengerID, constants.EventRideStarted, rideStarted)

	return nil
}

// handleRideCompletedEvent processes ride completed events
func (h *NatsHandler) handleRideCompletedEvent(msg []byte) error {
	var rideComplete models.RideComplete
	if err := json.Unmarshal(msg, &rideComplete); err != nil {
		return fmt.Errorf("failed to unmarshal ride completed event: %w", err)
	}

	logger.InfoCtx(context.Background(), "Received ride completed event",
		logger.String("ride_id", rideComplete.Ride.RideID.String()),
		logger.String("driver_id", rideComplete.Ride.DriverID.String()),
		logger.String("passenger_id", rideComplete.Ride.PassengerID.String()))

	// Notify driver and passenger about the ride completion
	h.echoWSHandler.NotifyClient(rideComplete.Ride.DriverID.String(), constants.EventRideCompleted, rideComplete)
	h.echoWSHandler.NotifyClient(rideComplete.Ride.PassengerID.String(), constants.EventRideCompleted, rideComplete)

	return nil
}
