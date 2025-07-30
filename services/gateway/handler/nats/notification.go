package nats

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	gatewaywebsocket "github.com/piresc/nebengjek/services/gateway/handler/websocket"
)

// GatewayNotificationHandler handles notification delivery events for the gateway
type GatewayNotificationHandler struct {
	natsClient      *natspkg.Client
	wsHandler       *gatewaywebsocket.EchoWebSocketHandler
	logger          *slog.Logger
}

// NewGatewayNotificationHandler creates a new gateway notification handler
func NewGatewayNotificationHandler(
	natsClient *natspkg.Client,
	wsHandler *gatewaywebsocket.EchoWebSocketHandler,
	logger *slog.Logger,
) *GatewayNotificationHandler {
	return &GatewayNotificationHandler{
		natsClient: natsClient,
		wsHandler:  wsHandler,
		logger:     logger,
	}
}

// InitNotificationConsumer initializes the NATS consumer for notification delivery
func (h *GatewayNotificationHandler) InitNotificationConsumer() error {
	// Create consumer config for notification delivery
	config := natspkg.ConsumerConfig{
		StreamName:    "NOTIFICATION_STREAM",
		ConsumerName:  "gateway_notification_delivery",
		FilterSubject: "NOTIFICATION.deliver",
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    3,
	}

	// Recreate consumer
	if err := h.natsClient.RecreateConsumer(config); err != nil {
		h.logger.Error("Failed to create notification delivery consumer", 
			slog.String("consumer", "gateway_notification_delivery"),
			slog.Any("error", err))
		return err
	}

	// Start consuming messages
	if err := h.natsClient.ConsumeMessages(
		"NOTIFICATION_STREAM", 
		"gateway_notification_delivery", 
		h.handleNotificationDeliveryJS,
	); err != nil {
		h.logger.Error("Failed to start consuming notification delivery messages", 
			slog.Any("error", err))
		return err
	}

	h.logger.Info("Started notification delivery consumer for gateway")
	return nil
}

// handleNotificationDeliveryJS processes notification delivery events from JetStream
func (h *GatewayNotificationHandler) handleNotificationDeliveryJS(msg jetstream.Msg) error {
	h.logger.Info("Received notification delivery message from JetStream",
		slog.String("subject", msg.Subject()),
		slog.String("consumer", "gateway_notification_delivery"))
	
	ctx := context.Background()
	err := h.handleNotificationDelivery(ctx, msg.Data())
	if err != nil {
		h.logger.Error("Failed to process notification delivery", slog.Any("error", err))
		return err
	}
	return nil // Message is auto-acknowledged by ConsumeMessages on success
}

// handleNotificationDelivery processes notification delivery events
func (h *GatewayNotificationHandler) handleNotificationDelivery(ctx context.Context, msgData []byte) error {
	var deliveryEvent models.NotificationDeliveryEvent
	if err := json.Unmarshal(msgData, &deliveryEvent); err != nil {
		h.logger.Error("Failed to unmarshal notification delivery event", slog.Any("error", err))
		return err
	}

	h.logger.Info("Processing notification delivery", 
		slog.String("user_id", deliveryEvent.UserID.String()),
		slog.String("type", deliveryEvent.Type),
		slog.String("delivery_id", deliveryEvent.DeliveryID.String()))

	// Check if user is connected to WebSocket
	userID := deliveryEvent.UserID.String()
	if !h.wsHandler.IsUserConnected(userID) {
		h.logger.Info("User not connected, notification will be queued", 
			slog.String("user_id", userID),
			slog.String("type", deliveryEvent.Type))
		return nil // Not an error - user just not connected
	}

	// Map notification types to WebSocket event names
	eventName := h.mapNotificationTypeToWSEvent(deliveryEvent.Type)
	
	// Send notification via WebSocket
	h.wsHandler.NotifyClient(userID, eventName, deliveryEvent.Data)

	h.logger.Info("Notification delivered via WebSocket", 
		slog.String("user_id", userID),
		slog.String("type", deliveryEvent.Type),
		slog.String("event", eventName))

	return nil
}

// mapNotificationTypeToWSEvent maps notification types to WebSocket event names
func (h *GatewayNotificationHandler) mapNotificationTypeToWSEvent(notificationType string) string {
	switch notificationType {
	case models.NotificationTypeMatchProposal:
		return "match_proposal"
	case models.NotificationTypeMatchAccepted:
		return "match_accepted"
	case models.NotificationTypeMatchRejected:
		return "match_rejected"
	case models.NotificationTypeRideStarted:
		return "ride_started"
	case models.NotificationTypeRidePickupArrived:
		return "ride_pickup_arrived"
	case models.NotificationTypeRideCompleted:
		return "ride_completed"
	case models.NotificationTypeRideCancelled:
		return "ride_cancelled"
	case models.NotificationTypePaymentProcessed:
		return "payment_processed"
	default:
		h.logger.Warn("Unknown notification type", slog.String("type", notificationType))
		return "notification"
	}
}