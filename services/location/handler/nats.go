package handler

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/location"
)

type NatsHandler struct {
	locationUC location.LocationUC
	natsClient *natspkg.Client
	subs       []*nats.Subscription
	cfg        *models.Config
}

// NewNatsHandler creates a new location NATS handler
func NewNatsHandler(locationUC location.LocationUC, cfg *models.Config) (*NatsHandler, error) {
	client, err := natspkg.NewClient(cfg.NATS.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS client: %w", err)
	}

	return &NatsHandler{
		locationUC: locationUC,
		natsClient: client,
		cfg:        cfg,
		subs:       make([]*nats.Subscription, 0),
	}, nil
}

// InitNATSConsumers initializes all NATS consumers for the location service
func (h *NatsHandler) InitNATSConsumers() error {
	// Initialize location update consumer
	sub, err := h.natsClient.Subscribe(constants.SubjectLocationUpdate, func(msg *nats.Msg) {
		if err := h.handleLocationUpdate(msg.Data); err != nil {
			log.Printf("Error handling location update: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to location updates: %w", err)
	}
	h.subs = append(h.subs, sub)

	return nil
}

// handleLocationUpdate processes location update events
func (h *NatsHandler) handleLocationUpdate(msg []byte) error {
	var update models.LocationUpdate
	if err := json.Unmarshal(msg, &update); err != nil {
		log.Printf("Failed to unmarshal location update: %v", err)
		return err
	}

	log.Printf("Received location update: rideID=%s, lat=%f, long=%f",
		update.RideID, update.Location.Latitude, update.Location.Longitude)

	// Store location update
	err := h.locationUC.StoreLocation(update)
	if err != nil {
		log.Printf("Failed to store location update: %v", err)
		return err
	}

	return nil
}

// Close unsubscribes from all NATS subscriptions
func (h *NatsHandler) Close() {
	for _, sub := range h.subs {
		sub.Unsubscribe()
	}
	if h.natsClient != nil {
		h.natsClient.Close()
	}
}
