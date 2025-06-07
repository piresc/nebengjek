package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	nrpkg "github.com/piresc/nebengjek/internal/pkg/newrelic"
	"github.com/piresc/nebengjek/services/location"
)

type LocationHandler struct {
	locationUC location.LocationUC
	natsClient *natspkg.Client
	subs       []*nats.Subscription
	nrApp      *newrelic.Application
}

// NewLocationHandler creates a new location NATS handler
func NewLocationHandler(
	locationUC location.LocationUC,
	client *natspkg.Client,
	nrApp *newrelic.Application,
) *LocationHandler {
	return &LocationHandler{
		locationUC: locationUC,
		natsClient: client,
		subs:       make([]*nats.Subscription, 0),
		nrApp:      nrApp,
	}
}

// InitNATSConsumers initializes all JetStream consumers for the location service
func (h *LocationHandler) InitNATSConsumers() error {
	logger.Info("Initializing JetStream consumers for location service")

	// Create JetStream consumers for location service
	consumerConfigs := natspkg.DefaultConsumerConfigs()

	// Create location update consumer
	locationUpdateConfig := consumerConfigs["location_update_location"]
	logger.Info("Creating location update consumer for location service",
		logger.String("stream", locationUpdateConfig.StreamName),
		logger.String("consumer", locationUpdateConfig.ConsumerName))

	if err := h.natsClient.CreateConsumer(locationUpdateConfig); err != nil {
		logger.Error("Failed to create location update consumer for location service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to create location update consumer: %w", err)
	}

	// Start consuming location update events
	if err := h.natsClient.ConsumeMessages("LOCATION_STREAM", "location_update_location", h.handleLocationUpdateJS); err != nil {
		logger.Error("Failed to start consuming location update events for location service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to start consuming location update events: %w", err)
	}

	logger.Info("Successfully initialized JetStream consumers for location service")
	return nil
}

// JetStream message handlers with proper acknowledgment

// handleLocationUpdateJS processes location update events from JetStream
func (h *LocationHandler) handleLocationUpdateJS(msg jetstream.Msg) error {
	// Create background transaction for NATS message processing
	txn := h.nrApp.StartTransaction("NATS.Location.HandleLocationUpdate")
	defer txn.End()

	// Add message attributes
	nrpkg.AddTransactionAttribute(txn, "message.subject", msg.Subject())
	nrpkg.AddTransactionAttribute(txn, "message.size", len(msg.Data()))
	nrpkg.AddTransactionAttribute(txn, "service", "location")

	// Create context with transaction
	ctx := newrelic.NewContext(context.Background(), txn)

	logger.InfoCtx(ctx, "Received location update event from JetStream",
		logger.String("subject", msg.Subject()))

	if err := h.handleLocationUpdate(ctx, msg.Data()); err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		logger.ErrorCtx(ctx, "Error handling location update event", logger.Err(err))
		return err // Return error to trigger NAK and retry
	}

	return nil // Success - message will be ACKed automatically
}

// handleLocationUpdate processes location update events
func (h *LocationHandler) handleLocationUpdate(ctx context.Context, msg []byte) error {
	var update models.LocationUpdate
	if err := json.Unmarshal(msg, &update); err != nil {
		logger.ErrorCtx(ctx, "Failed to unmarshal location update", logger.Err(err))
		return err
	}

	// Add business attributes to transaction
	if txn := nrpkg.FromContext(ctx); txn != nil {
		nrpkg.AddTransactionAttribute(txn, "ride.id", update.RideID)
		nrpkg.AddTransactionAttribute(txn, "location.latitude", update.Location.Latitude)
		nrpkg.AddTransactionAttribute(txn, "location.longitude", update.Location.Longitude)
	}

	logger.InfoCtx(ctx, "Received location update",
		logger.String("ride_id", update.RideID),
		logger.Float64("latitude", update.Location.Latitude),
		logger.Float64("longitude", update.Location.Longitude))

	// Store location update
	err := h.locationUC.StoreLocation(ctx, update)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to store location update",
			logger.String("ride_id", update.RideID),
			logger.Err(err))
		return err
	}

	return nil
}
