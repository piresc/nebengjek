package nats

import (
	"fmt"

	"github.com/nats-io/nats.go"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/user/handler/websocket"
)

// Handler handles NATS events for the user service
type Handler struct {
	wsManager  *websocket.WebSocketManager
	natsClient *natspkg.Client
	natsURL    string
	subs       []*nats.Subscription
}

// NewHandler creates a new NATS handler
func NewHandler(wsManager *websocket.WebSocketManager, natsURL string) (*Handler, error) {
	client, err := natspkg.NewClient(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS client: %w", err)
	}

	h := &Handler{
		wsManager:  wsManager,
		natsClient: client,
		natsURL:    natsURL,
	}

	if err := h.initConsumers(); err != nil {
		return nil, fmt.Errorf("failed to initialize NATS consumers: %w", err)
	}

	return h, nil
}

// initConsumers initializes all NATS consumers
func (h *Handler) initConsumers() error {
	// Initialize match-related consumers
	if err := h.InitMatchConsumers(); err != nil {
		return fmt.Errorf("failed to initialize match consumers: %w", err)
	}

	// Initialize ride-related consumers
	if err := h.InitRideConsumers(); err != nil {
		return fmt.Errorf("failed to initialize ride consumers: %w", err)
	}

	return nil
}

// Close unsubscribes from all NATS subscriptions
func (h *Handler) Close() {
	for _, sub := range h.subs {
		sub.Unsubscribe()
	}
}
