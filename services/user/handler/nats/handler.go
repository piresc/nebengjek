package nats

import (
	"fmt"

	"github.com/nats-io/nats.go"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/user/handler/websocket"
)

// Handler handles NATS events for the user service
type NatsHandler struct {
	wsManager  *websocket.WebSocketManager
	natsClient *natspkg.Client
	subs       []*nats.Subscription
}

// NewHandler creates a new NATS handler
func NewNatsHandler(
	wsManager *websocket.WebSocketManager,
	natsClient *natspkg.Client,
) *NatsHandler {
	return &NatsHandler{
		wsManager:  wsManager,
		natsClient: natsClient,
	}
}

// initConsumers initializes all NATS consumers
func (h *NatsHandler) InitConsumers() error {
	// Initialize match-related consumers
	if err := h.initMatchConsumers(); err != nil {
		return fmt.Errorf("failed to initialize match consumers: %w", err)
	}

	// Initialize ride-related consumers
	if err := h.initRideConsumers(); err != nil {
		return fmt.Errorf("failed to initialize ride consumers: %w", err)
	}

	return nil
}
