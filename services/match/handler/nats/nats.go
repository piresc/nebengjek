package nats

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
	"github.com/piresc/nebengjek/services/match"
)

// MatchHandler handles JetStream subscriptions for the match service
type MatchHandler struct {
	matchUC    match.MatchUC
	natsClient *natspkg.Client
	subs       []*nats.Subscription
	nrApp      *newrelic.Application
}

// NewMatchHandler creates a new match NATS handler
func NewMatchHandler(matchUC match.MatchUC, client *natspkg.Client, nrApp *newrelic.Application) *MatchHandler {
	return &MatchHandler{
		matchUC:    matchUC,
		natsClient: client,
		subs:       make([]*nats.Subscription, 0),
		nrApp:      nrApp,
	}
}

// InitNATSConsumers initializes all JetStream consumers for the match service
func (h *MatchHandler) InitNATSConsumers() error {
	logger.Info("Initializing JetStream consumers for match service")

	// Create JetStream consumers for match service
	consumerConfigs := natspkg.DefaultConsumerConfigs()

	// Create user beacon consumer
	beaconConfig := consumerConfigs["user_beacon_match"]
	logger.Info("Creating user beacon consumer for match service",
		logger.String("stream", beaconConfig.StreamName),
		logger.String("consumer", beaconConfig.ConsumerName))

	if err := h.natsClient.CreateConsumer(beaconConfig); err != nil {
		logger.Error("Failed to create user beacon consumer for match service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to create user beacon consumer: %w", err)
	}

	// Start consuming beacon events
	if err := h.natsClient.ConsumeMessages("USER_STREAM", "user_beacon_match", h.handleBeaconEventJS); err != nil {
		logger.Error("Failed to start consuming beacon events for match service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to start consuming beacon events: %w", err)
	}

	// Create user finder consumer
	finderConfig := consumerConfigs["user_finder_match"]
	logger.Info("Creating user finder consumer for match service",
		logger.String("stream", finderConfig.StreamName),
		logger.String("consumer", finderConfig.ConsumerName))

	if err := h.natsClient.CreateConsumer(finderConfig); err != nil {
		logger.Error("Failed to create user finder consumer for match service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to create user finder consumer: %w", err)
	}

	// Start consuming finder events
	if err := h.natsClient.ConsumeMessages("USER_STREAM", "user_finder_match", h.handleFinderEventJS); err != nil {
		logger.Error("Failed to start consuming finder events for match service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to start consuming finder events: %w", err)
	}

	// Create ride pickup consumer - RECREATE to ensure DeliverNewPolicy is applied
	ridePickupConfig := consumerConfigs["ride_pickup_match"]
	logger.Info("Recreating ride pickup consumer for match service with DeliverNewPolicy",
		logger.String("stream", ridePickupConfig.StreamName),
		logger.String("consumer", ridePickupConfig.ConsumerName),
		logger.String("deliver_policy", "DeliverNewPolicy"))

	if err := h.natsClient.RecreateConsumer(ridePickupConfig); err != nil {
		logger.Error("Failed to recreate ride pickup consumer for match service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to recreate ride pickup consumer: %w", err)
	}

	// Start consuming ride pickup events
	if err := h.natsClient.ConsumeMessages("RIDE_STREAM", "ride_pickup_match", h.handleRidePickupJS); err != nil {
		logger.Error("Failed to start consuming ride pickup events for match service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to start consuming ride pickup events: %w", err)
	}

	// Create ride completed consumer - RECREATE to ensure DeliverNewPolicy is applied
	rideCompletedConfig := consumerConfigs["ride_completed_match"]
	logger.Info("Recreating ride completed consumer for match service with DeliverNewPolicy",
		logger.String("stream", rideCompletedConfig.StreamName),
		logger.String("consumer", rideCompletedConfig.ConsumerName),
		logger.String("deliver_policy", "DeliverNewPolicy"))

	if err := h.natsClient.RecreateConsumer(rideCompletedConfig); err != nil {
		logger.Error("Failed to recreate ride completed consumer for match service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to recreate ride completed consumer: %w", err)
	}

	// Start consuming ride completed events
	if err := h.natsClient.ConsumeMessages("RIDE_STREAM", "ride_completed_match", h.handleRideCompletedJS); err != nil {
		logger.Error("Failed to start consuming ride completed events for match service",
			logger.ErrorField(err))
		return fmt.Errorf("failed to start consuming ride completed events: %w", err)
	}

	logger.Info("Successfully initialized JetStream consumers for match service")
	return nil
}

// JetStream message handlers with proper acknowledgment

// handleBeaconEventJS processes beacon events from JetStream
func (h *MatchHandler) handleBeaconEventJS(msg jetstream.Msg) error {
	// Create background transaction for NATS message processing
	txn := h.nrApp.StartTransaction("NATS.Match.HandleBeaconEvent")
	defer txn.End()

	// Add message attributes
	nrpkg.AddTransactionAttribute(txn, "message.subject", msg.Subject())
	nrpkg.AddTransactionAttribute(txn, "message.size", len(msg.Data()))
	nrpkg.AddTransactionAttribute(txn, "service", "match")

	// Create context with transaction
	ctx := newrelic.NewContext(context.Background(), txn)

	logger.InfoCtx(ctx, "Received beacon event from JetStream",
		logger.String("subject", msg.Subject()))

	if err := h.handleBeaconEvent(ctx, msg.Data()); err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		logger.ErrorCtx(ctx, "Error handling beacon event", logger.Err(err))
		return err // Return error to trigger NAK and retry
	}

	return nil // Success - message will be ACKed automatically
}

// handleFinderEventJS processes finder events from JetStream
func (h *MatchHandler) handleFinderEventJS(msg jetstream.Msg) error {
	// Create background transaction for NATS message processing
	txn := h.nrApp.StartTransaction("NATS.Match.HandleFinderEvent")
	defer txn.End()

	// Add message attributes
	nrpkg.AddTransactionAttribute(txn, "message.subject", msg.Subject())
	nrpkg.AddTransactionAttribute(txn, "message.size", len(msg.Data()))
	nrpkg.AddTransactionAttribute(txn, "service", "match")

	// Create context with transaction
	ctx := newrelic.NewContext(context.Background(), txn)

	logger.InfoCtx(ctx, "Received finder event from JetStream",
		logger.String("subject", msg.Subject()))

	if err := h.handleFinderEvent(ctx, msg.Data()); err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		logger.ErrorCtx(ctx, "Error handling finder event", logger.Err(err))
		return err // Return error to trigger NAK and retry
	}

	return nil // Success - message will be ACKed automatically
}

// handleRidePickupJS processes ride pickup events from JetStream
func (h *MatchHandler) handleRidePickupJS(msg jetstream.Msg) error {
	// Create background transaction for NATS message processing
	txn := h.nrApp.StartTransaction("NATS.Match.HandleRidePickup")
	defer txn.End()

	// Add message attributes
	nrpkg.AddTransactionAttribute(txn, "message.subject", msg.Subject())
	nrpkg.AddTransactionAttribute(txn, "message.size", len(msg.Data()))
	nrpkg.AddTransactionAttribute(txn, "service", "match")

	// Create context with transaction
	ctx := newrelic.NewContext(context.Background(), txn)

	logger.InfoCtx(ctx, "Received ride pickup event from JetStream",
		logger.String("subject", msg.Subject()))

	if err := h.handleRidePickup(ctx, msg.Data()); err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		logger.ErrorCtx(ctx, "Error handling ride pickup event", logger.Err(err))
		return err // Return error to trigger NAK and retry
	}

	return nil // Success - message will be ACKed automatically
}

// handleRideCompletedJS processes ride completed events from JetStream
func (h *MatchHandler) handleRideCompletedJS(msg jetstream.Msg) error {
	// Create background transaction for NATS message processing
	txn := h.nrApp.StartTransaction("NATS.Match.HandleRideCompleted")
	defer txn.End()

	// Add message attributes
	nrpkg.AddTransactionAttribute(txn, "message.subject", msg.Subject())
	nrpkg.AddTransactionAttribute(txn, "message.size", len(msg.Data()))
	nrpkg.AddTransactionAttribute(txn, "service", "match")

	// Create context with transaction
	ctx := newrelic.NewContext(context.Background(), txn)

	logger.InfoCtx(ctx, "Received ride completed event from JetStream",
		logger.String("subject", msg.Subject()))

	if err := h.handleRideCompleted(ctx, msg.Data()); err != nil {
		nrpkg.NoticeTransactionError(txn, err)
		logger.ErrorCtx(ctx, "Error handling ride completed event", logger.Err(err))
		return err // Return error to trigger NAK and retry
	}

	return nil // Success - message will be ACKed automatically
}

// handleBeaconEvent processes beacon events from the user service
func (h *MatchHandler) handleBeaconEvent(ctx context.Context, msg []byte) error {
	var event models.BeaconEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		logger.ErrorCtx(ctx, "Failed to unmarshal beacon event", logger.Err(err))
		return err
	}

	// Add business attributes to transaction
	if txn := nrpkg.FromContext(ctx); txn != nil {
		nrpkg.AddTransactionAttribute(txn, "user.id", event.UserID)
		nrpkg.AddTransactionAttribute(txn, "beacon.is_active", event.IsActive)
	}

	logger.InfoCtx(ctx, "Received beacon event",
		logger.String("user_id", event.UserID),
		logger.Bool("is_active", event.IsActive))

	// Forward the event to usecase for processing
	return h.matchUC.HandleBeaconEvent(ctx, event)
}

// handleFinderEvent processes finder events from the user service
func (h *MatchHandler) handleFinderEvent(ctx context.Context, msg []byte) error {
	var event models.FinderEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		logger.ErrorCtx(ctx, "Failed to unmarshal finder event", logger.Err(err))
		return err
	}

	// Add business attributes to transaction
	if txn := nrpkg.FromContext(ctx); txn != nil {
		nrpkg.AddTransactionAttribute(txn, "user.id", event.UserID)
		nrpkg.AddTransactionAttribute(txn, "finder.is_active", event.IsActive)
		nrpkg.AddTransactionAttribute(txn, "location.latitude", event.Location.Latitude)
		nrpkg.AddTransactionAttribute(txn, "location.longitude", event.Location.Longitude)
		nrpkg.AddTransactionAttribute(txn, "target.latitude", event.TargetLocation.Latitude)
		nrpkg.AddTransactionAttribute(txn, "target.longitude", event.TargetLocation.Longitude)
	}

	logger.InfoCtx(ctx, "Received finder event",
		logger.String("user_id", event.UserID),
		logger.Bool("is_active", event.IsActive),
		logger.Float64("location_lat", event.Location.Latitude),
		logger.Float64("location_lng", event.Location.Longitude),
		logger.Float64("target_lat", event.TargetLocation.Latitude),
		logger.Float64("target_lng", event.TargetLocation.Longitude))

	// Forward the event to usecase for processing
	return h.matchUC.HandleFinderEvent(ctx, event)
}

// handleRidePickup processes ride pickup events to lock drivers
func (h *MatchHandler) handleRidePickup(ctx context.Context, msg []byte) error {
	var ridePickup models.RideResp
	if err := json.Unmarshal(msg, &ridePickup); err != nil {
		logger.ErrorCtx(ctx, "Failed to unmarshal ride pickup event", logger.Err(err))
		return err
	}

	// Add business attributes to transaction
	if txn := nrpkg.FromContext(ctx); txn != nil {
		nrpkg.AddTransactionAttribute(txn, "ride.id", ridePickup.RideID)
		nrpkg.AddTransactionAttribute(txn, "driver.id", ridePickup.DriverID)
		nrpkg.AddTransactionAttribute(txn, "passenger.id", ridePickup.PassengerID)
	}

	logger.InfoCtx(ctx, "Received ride pickup event",
		logger.String("ride_id", ridePickup.RideID),
		logger.String("driver_id", ridePickup.DriverID),
		logger.String("passenger_id", ridePickup.PassengerID))

	// Store active ride information in Redis
	if err := h.matchUC.SetActiveRide(ctx, ridePickup.DriverID, ridePickup.PassengerID, ridePickup.RideID); err != nil {
		logger.WarnCtx(ctx, "Failed to set active ride",
			logger.String("ride_id", ridePickup.RideID),
			logger.Err(err))
		// Continue even if this fails - don't block ride flow
	}

	// Remove driver from available pool (lock them)
	if err := h.matchUC.RemoveDriverFromPool(ctx, ridePickup.DriverID); err != nil {
		logger.WarnCtx(ctx, "Failed to lock driver",
			logger.String("driver_id", ridePickup.DriverID),
			logger.Err(err))
		// Continue even if this fails - don't block ride flow
	}

	// Remove passenger from available pool (lock them)
	if err := h.matchUC.RemovePassengerFromPool(ctx, ridePickup.PassengerID); err != nil {
		logger.WarnCtx(ctx, "Failed to lock passenger",
			logger.String("passenger_id", ridePickup.PassengerID),
			logger.Err(err))
		// Continue even if this fails - don't block ride flow
	}

	return nil
}

// handleRideCompleted processes ride completed events to unlock users
func (h *MatchHandler) handleRideCompleted(ctx context.Context, msg []byte) error {
	var rideComplete models.RideComplete
	if err := json.Unmarshal(msg, &rideComplete); err != nil {
		logger.ErrorCtx(ctx, "Failed to unmarshal ride completed event", logger.Err(err))
		return err
	}

	// Add business attributes to transaction
	if txn := nrpkg.FromContext(ctx); txn != nil {
		nrpkg.AddTransactionAttribute(txn, "ride.id", rideComplete.Ride.RideID.String())
		nrpkg.AddTransactionAttribute(txn, "driver.id", rideComplete.Ride.DriverID.String())
		nrpkg.AddTransactionAttribute(txn, "passenger.id", rideComplete.Ride.PassengerID.String())
	}

	logger.InfoCtx(ctx, "Received ride completed event",
		logger.String("ride_id", rideComplete.Ride.RideID.String()),
		logger.String("driver_id", rideComplete.Ride.DriverID.String()),
		logger.String("passenger_id", rideComplete.Ride.PassengerID.String()))

	// Remove active ride information from Redis
	if err := h.matchUC.RemoveActiveRide(ctx, rideComplete.Ride.DriverID.String(), rideComplete.Ride.PassengerID.String()); err != nil {
		logger.WarnCtx(ctx, "Failed to remove active ride",
			logger.String("ride_id", rideComplete.Ride.RideID.String()),
			logger.Err(err))
		// Continue even if this fails
	}

	return nil
}
