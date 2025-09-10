package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	gatewaywebsocket "github.com/piresc/nebengjek/services/gateway/handler/websocket"
	"github.com/piresc/nebengjek/services/gateway/repository"
)

// GatewayNotificationHandler handles notification delivery events for the gateway
type GatewayNotificationHandler struct {
	natsClient      *natspkg.Client
	wsHandler       gatewaywebsocket.WebSocketNotifier
	sessionRepo     *repository.WSSessionRepository
	logger          *slog.Logger
	serverID        string
}

// NewGatewayNotificationHandler creates a new gateway notification handler
func NewGatewayNotificationHandler(
	natsClient *natspkg.Client,
	wsHandler gatewaywebsocket.WebSocketNotifier,
	sessionRepo *repository.WSSessionRepository,
	logger *slog.Logger,
	serverID string,
) *GatewayNotificationHandler {
	return &GatewayNotificationHandler{
		natsClient:  natsClient,
		wsHandler:   wsHandler,
		sessionRepo: sessionRepo,
		logger:      logger,
		serverID:    serverID,
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

// InitMultiGatewayConsumers initializes consumers for multi-gateway broadcasting
func (h *GatewayNotificationHandler) InitMultiGatewayConsumers() error {
	sanitizedServerID := strings.ReplaceAll(h.serverID, ".", "-")
	sanitizedServerID = strings.ReplaceAll(sanitizedServerID, "_", "-")
	
	consumers := []struct {
		subject      string
		consumerName string
		handler      func(jetstream.Msg) error
	}{
		{
			subject:      constants.SubjectWSBroadcastUser,
			consumerName: fmt.Sprintf("ws_user_broadcast_%s", sanitizedServerID),
			handler:      h.handleUserBroadcast,
		},
		{
			subject:      constants.SubjectWSBroadcastRole,
			consumerName: fmt.Sprintf("ws_role_broadcast_%s", sanitizedServerID),
			handler:      h.handleRoleBroadcast,
		},
		{
			subject:      constants.SubjectWSBroadcastAll,
			consumerName: fmt.Sprintf("ws_all_broadcast_%s", sanitizedServerID),
			handler:      h.handleAllBroadcast,
		},
		{
			subject:      constants.SubjectWSRouteUser,
			consumerName: fmt.Sprintf("ws_user_route_%s", sanitizedServerID),
			handler:      h.handleUserRoute,
		},
	}

	for _, consumer := range consumers {
		config := natspkg.ConsumerConfig{
			StreamName:    constants.StreamWebSocket,
			ConsumerName:  consumer.consumerName,
			FilterSubject: consumer.subject,
			AckPolicy:     jetstream.AckExplicitPolicy,
			DeliverPolicy: jetstream.DeliverNewPolicy,
			MaxDeliver:    3,
			AckWait:       30 * time.Second,
			MaxAckPending: 10, // Process multiple messages concurrently
		}

		if err := h.natsClient.RecreateConsumer(config); err != nil {
			return fmt.Errorf("failed to create consumer for %s: %w", consumer.subject, err)
		}

		if err := h.natsClient.ConsumeMessages(
			constants.StreamWebSocket,
			consumer.consumerName,
			consumer.handler,
		); err != nil {
			return fmt.Errorf("failed to start consuming %s: %w", consumer.subject, err)
		}

		h.logger.Info("Multi-gateway consumer initialized",
			slog.String("subject", consumer.subject),
			slog.String("consumer", consumer.consumerName))
	}

	return nil
}

// handleUserBroadcast processes user-targeted broadcasts
func (h *GatewayNotificationHandler) handleUserBroadcast(msg jetstream.Msg) error {
	var broadcast models.WSUserBroadcast
	if err := json.Unmarshal(msg.Data(), &broadcast); err != nil {
		h.logger.Error("Failed to unmarshal user broadcast", slog.String("error", err.Error()))
		return err
	}

	ctx := context.Background()
	delivered := 0

	// Get server connections (from Redis session store)
	serverConnections, err := h.sessionRepo.GetServerConnections(ctx, h.serverID)
	if err != nil {
		h.logger.Error("Failed to get server connections", slog.String("error", err.Error()))
		return err
	}

	// Create map for efficient lookup
	serverUserMap := make(map[string]bool)
	for _, userID := range serverConnections {
		serverUserMap[userID] = true
	}

	// Process target users connected to this server
	for _, userID := range broadcast.TargetUsers {
		// Skip if user is in exclude list
		if h.isUserExcluded(userID, broadcast.ExcludeUsers) {
			continue
		}

		// Skip if user is not connected to this server
		if !serverUserMap[userID] {
			continue
		}

		// Deliver to user
		var messageData interface{}
		if err := json.Unmarshal(broadcast.Data, &messageData); err != nil {
			h.logger.Error("Failed to unmarshal message data",
				slog.String("user_id", userID),
				slog.String("error", err.Error()))
			continue
		}

		if err := h.wsHandler.NotifyClientWithError(userID, broadcast.Event, messageData); err != nil {
			h.logger.Error("Failed to deliver message to user",
				slog.String("user_id", userID),
				slog.String("message_id", broadcast.MessageID),
				slog.String("error", err.Error()))
		} else {
			delivered++
		}
	}

	h.logger.Info("User broadcast processed",
		slog.String("message_id", broadcast.MessageID),
		slog.String("server_id", h.serverID),
		slog.Int("delivered", delivered))

	return nil
}

// handleRoleBroadcast processes role-targeted broadcasts
func (h *GatewayNotificationHandler) handleRoleBroadcast(msg jetstream.Msg) error {
	var broadcast models.WSRoleBroadcast
	if err := json.Unmarshal(msg.Data(), &broadcast); err != nil {
		h.logger.Error("Failed to unmarshal role broadcast", slog.String("error", err.Error()))
		return err
	}

	ctx := context.Background()
	delivered := 0

	// Get server connections
	serverConnections, err := h.sessionRepo.GetServerConnections(ctx, h.serverID)
	if err != nil {
		h.logger.Error("Failed to get server connections", slog.String("error", err.Error()))
		return err
	}

	// Process each connection and check role
	for _, userID := range serverConnections {
		// Skip if user is in exclude list
		if h.isUserExcluded(userID, broadcast.ExcludeUsers) {
			continue
		}

		// Get user session to check role
		session, err := h.sessionRepo.GetConnection(ctx, userID)
		if err != nil || session == nil {
			continue
		}

		// Check if user role matches target roles
		if h.isRoleMatched(session.Role, broadcast.TargetRoles) {
			var messageData interface{}
			if err := json.Unmarshal(broadcast.Data, &messageData); err != nil {
				continue
			}

			if err := h.wsHandler.NotifyClientWithError(userID, broadcast.Event, messageData); err != nil {
				h.logger.Error("Failed to deliver message to user",
					slog.String("user_id", userID),
					slog.String("role", session.Role),
					slog.String("error", err.Error()))
			} else {
				delivered++
			}
		}
	}

	h.logger.Info("Role broadcast processed",
		slog.String("message_id", broadcast.MessageID),
		slog.String("target_roles", fmt.Sprintf("%v", broadcast.TargetRoles)),
		slog.Int("delivered", delivered))

	return nil
}

// handleAllBroadcast processes broadcasts to all users
func (h *GatewayNotificationHandler) handleAllBroadcast(msg jetstream.Msg) error {
	var broadcast models.WSBroadcastMessage
	if err := json.Unmarshal(msg.Data(), &broadcast); err != nil {
		h.logger.Error("Failed to unmarshal all broadcast", slog.String("error", err.Error()))
		return err
	}

	ctx := context.Background()
	delivered := 0

	// Get all connections on this server
	serverConnections, err := h.sessionRepo.GetServerConnections(ctx, h.serverID)
	if err != nil {
		h.logger.Error("Failed to get server connections", slog.String("error", err.Error()))
		return err
	}

	var messageData interface{}
	if err := json.Unmarshal(broadcast.Data, &messageData); err != nil {
		h.logger.Error("Failed to unmarshal message data", slog.String("error", err.Error()))
		return err
	}

	// Broadcast to all users on this server
	for _, userID := range serverConnections {
		if err := h.wsHandler.NotifyClientWithError(userID, broadcast.Event, messageData); err != nil {
			h.logger.Error("Failed to deliver broadcast message to user",
				slog.String("user_id", userID),
				slog.String("error", err.Error()))
		} else {
			delivered++
		}
	}

	h.logger.Info("All broadcast processed",
		slog.String("message_id", broadcast.MessageID),
		slog.Int("delivered", delivered))

	return nil
}

// handleUserRoute processes cross-server user routing
func (h *GatewayNotificationHandler) handleUserRoute(msg jetstream.Msg) error {
	var route models.WSRouteMessage
	if err := json.Unmarshal(msg.Data(), &route); err != nil {
		h.logger.Error("Failed to unmarshal route message", slog.String("error", err.Error()))
		return err
	}

	ctx := context.Background()

	// Check if target user is connected to this server
	session, err := h.sessionRepo.GetConnection(ctx, route.TargetUserID)
	if err != nil {
		h.logger.Error("Failed to get user connection",
			slog.String("user_id", route.TargetUserID),
			slog.String("error", err.Error()))
		return err
	}

	if session == nil || session.ServerID != h.serverID {
		// User not connected to this server, skip
		return nil
	}

	// Deliver message to user
	var messageData interface{}
	if err := json.Unmarshal(route.Data, &messageData); err != nil {
		h.logger.Error("Failed to unmarshal route data", slog.String("error", err.Error()))
		return err
	}

	if err := h.wsHandler.NotifyClientWithError(route.TargetUserID, route.Event, messageData); err != nil {
		h.logger.Error("Failed to route message to user",
			slog.String("user_id", route.TargetUserID),
			slog.String("message_id", route.MessageID),
			slog.String("error", err.Error()))
		return err
	}

	h.logger.Info("Message routed successfully",
		slog.String("message_id", route.MessageID),
		slog.String("user_id", route.TargetUserID))

	return nil
}

// Helper methods
func (h *GatewayNotificationHandler) isUserExcluded(userID string, excludeList []string) bool {
	for _, excludeUserID := range excludeList {
		if userID == excludeUserID {
			return true
		}
	}
	return false
}

func (h *GatewayNotificationHandler) isRoleMatched(userRole string, targetRoles []string) bool {
	for _, role := range targetRoles {
		if userRole == role {
			return true
		}
	}
	return false
}