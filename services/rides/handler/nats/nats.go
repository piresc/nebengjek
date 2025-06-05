package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/constants"
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

// NewNatsHandler creates a new rides NATS handler
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

// InitNATSConsumers initializes all NATS consumers for the rides service
func (h *RidesHandler) InitNATSConsumers() error {
	// Initialize match accepted consumer
	sub, err := h.natsClient.Subscribe(constants.SubjectMatchAccepted, func(msg *nats.Msg) {
		if err := h.handleMatchAccepted(msg.Data); err != nil {
			logger.Error("Error handling match accepted event", logger.ErrorField(err))
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to match accepted events: %w", err)
	}
	h.subs = append(h.subs, sub)

	// Initialize location aggregate consumer
	sub, err = h.natsClient.Subscribe(constants.SubjectLocationAggregate, func(msg *nats.Msg) {
		if err := h.handleLocationAggregate(msg.Data); err != nil {
			logger.Error("Error handling location aggregate", logger.ErrorField(err))
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to location aggregates: %w", err)
	}
	h.subs = append(h.subs, sub)

	return nil
}

// handleMatchAccepted processes match acceptance events to create rides
func (h *RidesHandler) handleMatchAccepted(msg []byte) error {
	var matchProposal models.MatchProposal
	if err := json.Unmarshal(msg, &matchProposal); err != nil {
		logger.ErrorCtx(context.Background(), "Failed to unmarshal match proposal", logger.ErrorField(err))
		return err
	}

	logger.InfoCtx(context.Background(), "Received match accepted event",
		logger.String("match_id", matchProposal.ID),
		logger.String("driver_id", matchProposal.DriverID),
		logger.String("passenger_id", matchProposal.PassengerID))

	// Create a ride from the match proposal
	if err := h.ridesUC.CreateRide(context.Background(), matchProposal); err != nil {
		logger.ErrorCtx(context.Background(), "Failed to create ride from match", logger.ErrorField(err))
		return err
	}

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
