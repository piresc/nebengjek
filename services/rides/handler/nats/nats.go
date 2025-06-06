package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/rides"
)

type RidesHandler struct {
	ridesUC    rides.RideUC
	natsClient *natspkg.Client
	subs       []*nats.Subscription
	cfg        *models.Config
}

// NewRidesHandler creates a new rides NATS handler
func NewRidesHandler(
	ridesUC rides.RideUC,
	client *natspkg.Client,
	cfg *models.Config,
) *RidesHandler {
	return &RidesHandler{
		ridesUC:    ridesUC,
		natsClient: client,
		subs:       make([]*nats.Subscription, 0),
		cfg:        cfg,
	}
}

// InitNATSConsumers initializes all JetStream consumers for the rides service
func (h *RidesHandler) InitNATSConsumers() error {
	logger.Info("Initializing JetStream consumers for rides service")

	// Create JetStream consumers for rides service
	consumerConfigs := natspkg.DefaultConsumerConfigs()

	// Create match accepted consumer (using service-specific naming pattern)
	matchAcceptedConfig := consumerConfigs["match_accepted_rides"]
	logger.Info("Creating match accepted consumer for rides service",
		logger.String("stream", matchAcceptedConfig.StreamName),
		logger.String("consumer", matchAcceptedConfig.ConsumerName),
		logger.String("filter_subject", matchAcceptedConfig.FilterSubject))

	if err := h.natsClient.CreateConsumer(matchAcceptedConfig); err != nil {
		logger.Error("Failed to create match accepted consumer for rides service",
			logger.String("consumer", matchAcceptedConfig.ConsumerName),
			logger.ErrorField(err))
		return fmt.Errorf("failed to create match accepted consumer: %w", err)
	}

	// Start consuming match accepted events
	logger.Info("Starting to consume match accepted events for rides service",
		logger.String("stream", "MATCH_STREAM"),
		logger.String("consumer", "match_accepted_rides"))

	if err := h.natsClient.ConsumeMessages("MATCH_STREAM", "match_accepted_rides", h.handleMatchAcceptedJS); err != nil {
		logger.Error("Failed to start consuming match accepted events for rides service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to start consuming match accepted events: %w", err)
	}

	// Create location aggregate consumer
	locationAggregateConfig := consumerConfigs["location_aggregate_rides"]
	logger.Info("Creating location aggregate consumer for rides service",
		logger.String("stream", locationAggregateConfig.StreamName),
		logger.String("consumer", locationAggregateConfig.ConsumerName))

	if err := h.natsClient.CreateConsumer(locationAggregateConfig); err != nil {
		logger.Error("Failed to create location aggregate consumer for rides service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to create location aggregate consumer: %w", err)
	}

	// Start consuming location aggregate events
	if err := h.natsClient.ConsumeMessages("LOCATION_STREAM", "location_aggregate_rides", h.handleLocationAggregateJS); err != nil {
		logger.Error("Failed to start consuming location aggregate events for rides service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to start consuming location aggregate events: %w", err)
	}

	logger.Info("Successfully initialized JetStream consumers for rides service")
	return nil
}

// JetStream message handlers with proper acknowledgment

// handleMatchAcceptedJS processes match accepted events from JetStream
func (h *RidesHandler) handleMatchAcceptedJS(msg jetstream.Msg) error {
	logger.InfoCtx(context.Background(), "Received match accepted event from JetStream",
		logger.String("subject", msg.Subject()))

	if err := h.handleMatchAccepted(msg.Data()); err != nil {
		logger.ErrorCtx(context.Background(), "Error handling match accepted event", logger.Err(err))
		return err // Return error to trigger NAK and retry
	}

	return nil // Success - message will be ACKed automatically
}

// handleLocationAggregateJS processes location aggregate events from JetStream
func (h *RidesHandler) handleLocationAggregateJS(msg jetstream.Msg) error {
	logger.InfoCtx(context.Background(), "Received location aggregate event from JetStream",
		logger.String("subject", msg.Subject()))

	if err := h.handleLocationAggregate(msg.Data()); err != nil {
		logger.ErrorCtx(context.Background(), "Error handling location aggregate event", logger.Err(err))
		return err // Return error to trigger NAK and retry
	}

	return nil // Success - message will be ACKed automatically
}

// handleMatchAccepted processes match acceptance events to create rides
func (h *RidesHandler) handleMatchAccepted(msg []byte) error {
	logger.InfoCtx(context.Background(), "Processing match accepted event from JetStream",
		logger.String("message_size", fmt.Sprintf("%d bytes", len(msg))))

	var matchProposal models.MatchProposal
	if err := json.Unmarshal(msg, &matchProposal); err != nil {
		logger.ErrorCtx(context.Background(), "Failed to unmarshal match proposal",
			logger.String("raw_message", string(msg)),
			logger.ErrorField(err))
		return err
	}

	logger.InfoCtx(context.Background(), "Successfully parsed match accepted event, creating ride",
		logger.String("match_id", matchProposal.ID),
		logger.String("driver_id", matchProposal.DriverID),
		logger.String("passenger_id", matchProposal.PassengerID))

	// Create a ride from the match proposal
	if err := h.ridesUC.CreateRide(context.Background(), matchProposal); err != nil {
		logger.ErrorCtx(context.Background(), "Failed to create ride from match proposal",
			logger.String("match_id", matchProposal.ID),
			logger.String("driver_id", matchProposal.DriverID),
			logger.String("passenger_id", matchProposal.PassengerID),
			logger.ErrorField(err))
		return err
	}

	logger.InfoCtx(context.Background(), "Successfully processed match accepted event and created ride",
		logger.String("match_id", matchProposal.ID))
	return nil
}

// handleLocationAggregate processes location aggregates for billing
func (h *RidesHandler) handleLocationAggregate(msg []byte) error {
	var update models.LocationAggregate
	if err := json.Unmarshal(msg, &update); err != nil {
		logger.ErrorCtx(context.Background(), "Failed to unmarshal location aggregate", logger.ErrorField(err))
		return err
	}

	logger.InfoCtx(context.Background(), "Received location aggregate",
		logger.String("ride_id", update.RideID),
		logger.Float64("distance_km", update.Distance))

	// Only process if distance is >= minimum configured distance
	if update.Distance >= h.cfg.Rides.MinDistanceKm {
		// Convert ride ID to UUID
		rideUUID, err := uuid.Parse(update.RideID)
		if err != nil {
			logger.ErrorCtx(context.Background(), "Invalid ride ID format",
				logger.String("ride_id", update.RideID),
				logger.ErrorField(err))
			return fmt.Errorf("invalid ride ID: %w", err)
		}

		// Calculate cost at 3000 IDR per km
		cost := int(update.Distance * 3000)

		// Create billing entry
		entry := &models.BillingLedger{
			RideID:   rideUUID,
			Distance: update.Distance,
			Cost:     cost,
		}

		// Store billing entry and update total cost
		if err := h.ridesUC.ProcessBillingUpdate(update.RideID, entry); err != nil {
			logger.ErrorCtx(context.Background(), "Failed to process billing update",
				logger.String("ride_id", update.RideID),
				logger.ErrorField(err))
			return err
		}
	}

	return nil
}
