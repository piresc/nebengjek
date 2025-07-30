package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

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
		DeliverPolicy: jetstream.DeliverNewPolicy,  // Only deliver new messages after consumer creation
		MaxDeliver:    3,
		AckWait:       30 * time.Second,    // Explicit ACK timeout
		MaxAckPending: 1,                   // Process one message at a time to avoid race conditions
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
	metadata, metaErr := msg.Metadata()
	var msgSeq, pendingStr string
	if metaErr == nil {
		msgSeq = fmt.Sprintf("%d", metadata.Sequence.Stream)
		pendingStr = fmt.Sprintf("%d", metadata.NumPending)
	} else {
		msgSeq = "unknown"
		pendingStr = "unknown"
	}
	
	h.logger.Info("GATEWAY: Received notification delivery message from JetStream",
		slog.String("subject", msg.Subject()),
		slog.String("consumer", "gateway_notification_delivery"),
		slog.String("message_sequence", msgSeq),
		slog.String("pending", pendingStr))
	
	ctx := context.Background()
	err := h.handleNotificationDelivery(ctx, msg.Data())
	if err != nil {
		h.logger.Error("GATEWAY: Failed to process notification delivery - will NAK", 
			slog.Any("error", err),
			slog.String("message_sequence", msgSeq))
		// Return error to NAK the message for retry
		return err
	}
	
	h.logger.Info("GATEWAY: Successfully processed notification delivery - returning nil for ACK",
		slog.String("message_sequence", msgSeq))
	return nil // Success - message will be auto-acknowledged by ConsumeMessages
}

// handleNotificationDelivery processes notification delivery events
func (h *GatewayNotificationHandler) handleNotificationDelivery(ctx context.Context, msgData []byte) error {
	var deliveryEvent models.NotificationDeliveryEvent
	if err := json.Unmarshal(msgData, &deliveryEvent); err != nil {
		h.logger.Error("Failed to unmarshal notification delivery event", 
			slog.Any("error", err),
			slog.String("raw_data", string(msgData)))
		// This is a permanent failure - don't retry
		return nil
	}

	userID := deliveryEvent.UserID.String()
	h.logger.Info("Processing notification delivery", 
		slog.String("user_id", userID),
		slog.String("type", deliveryEvent.Type),
		slog.String("delivery_id", deliveryEvent.DeliveryID.String()))

	// Map notification types to WebSocket event names
	eventName := h.mapNotificationTypeToWSEvent(deliveryEvent.Type)
	
	// Send notification via WebSocket - this method returns errors for failed delivery
	if err := h.wsHandler.NotifyClientWithError(userID, eventName, deliveryEvent.Data); err != nil {
		h.logger.Error("Failed to deliver notification via WebSocket", 
			slog.String("user_id", userID),
			slog.String("type", deliveryEvent.Type),
			slog.String("event", eventName),
			slog.String("delivery_id", deliveryEvent.DeliveryID.String()),
			slog.Any("error", err))
		
		// Check if error is due to client not being connected
		if strings.Contains(err.Error(), "not connected") {
			h.logger.Info("Client not connected - acknowledging message to prevent retry loop", 
				slog.String("user_id", userID),
				slog.String("delivery_id", deliveryEvent.DeliveryID.String()))
			// Acknowledge the message since client being offline is not a system error
			return nil
		}
		
		// For other errors (marshaling, WebSocket send failures), retry the message
		return fmt.Errorf("failed to deliver notification via WebSocket: %w", err)
	}

	h.logger.Info("Notification delivered via WebSocket successfully", 
		slog.String("user_id", userID),
		slog.String("type", deliveryEvent.Type),
		slog.String("event", eventName),
		slog.String("delivery_id", deliveryEvent.DeliveryID.String()))

	// Return nil only when message was successfully delivered to WebSocket client
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