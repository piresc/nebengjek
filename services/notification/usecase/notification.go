package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models/match"
	"github.com/piresc/nebengjek/internal/pkg/models/ride"
	"github.com/piresc/nebengjek/internal/pkg/models/notification"
	"github.com/piresc/nebengjek/internal/pkg/models/websocket"
	"github.com/piresc/nebengjek/internal/pkg/nats"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	notificationpkg "github.com/piresc/nebengjek/services/notification"
	"github.com/piresc/nebengjek/services/notification/repository"
)

// NotificationUC implements the notification usecase interface
type NotificationUC struct {
	notificationRepo notificationpkg.NotificationRepo
	natsClient       *natspkg.Client
	logger           *slog.Logger
}

// NewNotificationUC creates a new notification usecase
func NewNotificationUC(
	notificationRepo notificationpkg.NotificationRepo,
	natsClient *natspkg.Client,
	logger *slog.Logger,
) *NotificationUC {
	return &NotificationUC{
		notificationRepo: notificationRepo,
		natsClient:       natsClient,
		logger:           logger,
	}
}

// ProcessMatchProposal processes match proposal events
func (uc *NotificationUC) ProcessMatchProposal(ctx context.Context, eventData []byte) error {
	var matchProposal match.MatchProposal
	if err := json.Unmarshal(eventData, &matchProposal); err != nil {
		return fmt.Errorf("failed to unmarshal match proposal data: %w", err)
	}

	// Create notification for driver
	driverNotification := &notification.UserNotification{
		UserID:    uuid.MustParse(matchProposal.DriverID),
		Type:      notification.NotificationTypeMatchProposal,
		Data:      matchProposal,
		Timestamp: time.Now(),
	}

	// Create notification for passenger
	passengerNotification := &notification.UserNotification{
		UserID:    uuid.MustParse(matchProposal.PassengerID),
		Type:      notification.NotificationTypeMatchProposal,
		Data:      matchProposal,
		Timestamp: time.Now(),
	}

	// Process both notifications
	if err := uc.processAndStoreNotification(ctx, driverNotification); err != nil {
		return fmt.Errorf("failed to process driver notification: %w", err)
	}

	if err := uc.processAndStoreNotification(ctx, passengerNotification); err != nil {
		return fmt.Errorf("failed to process passenger notification: %w", err)
	}

	return nil
}

// ProcessMatchAccepted processes match accepted events
func (uc *NotificationUC) ProcessMatchAccepted(ctx context.Context, eventData []byte) error {
	var matchProposal match.MatchProposal
	if err := json.Unmarshal(eventData, &matchProposal); err != nil {
		return fmt.Errorf("failed to unmarshal match accepted data: %w", err)
	}

	// Create notifications for both driver and passenger
	driverNotification := &notification.UserNotification{
		UserID:    uuid.MustParse(matchProposal.DriverID),
		Type:      notification.NotificationTypeMatchAccepted,
		Data:      matchProposal,
		Timestamp: time.Now(),
	}

	passengerNotification := &notification.UserNotification{
		UserID:    uuid.MustParse(matchProposal.PassengerID),
		Type:      notification.NotificationTypeMatchAccepted,
		Data:      matchProposal,
		Timestamp: time.Now(),
	}

	if err := uc.processAndStoreNotification(ctx, driverNotification); err != nil {
		return err
	}

	return uc.processAndStoreNotification(ctx, passengerNotification)
}

// ProcessMatchRejected processes match rejected events
func (uc *NotificationUC) ProcessMatchRejected(ctx context.Context, eventData []byte) error {
	var matchProposal match.MatchProposal
	if err := json.Unmarshal(eventData, &matchProposal); err != nil {
		return fmt.Errorf("failed to unmarshal match rejected data: %w", err)
	}

	// Create notifications for both driver and passenger
	driverNotification := &notification.UserNotification{
		UserID:    uuid.MustParse(matchProposal.DriverID),
		Type:      notification.NotificationTypeMatchRejected,
		Data:      matchProposal,
		Timestamp: time.Now(),
	}

	passengerNotification := &notification.UserNotification{
		UserID:    uuid.MustParse(matchProposal.PassengerID),
		Type:      notification.NotificationTypeMatchRejected,
		Data:      matchProposal,
		Timestamp: time.Now(),
	}

	if err := uc.processAndStoreNotification(ctx, driverNotification); err != nil {
		return err
	}

	return uc.processAndStoreNotification(ctx, passengerNotification)
}

// ProcessRidePickup processes ride pickup events (ride.pickup subject)
func (uc *NotificationUC) ProcessRidePickup(ctx context.Context, eventData []byte) error {
	var rideResp ride.RideResp
	if err := json.Unmarshal(eventData, &rideResp); err != nil {
		return fmt.Errorf("failed to unmarshal ride pickup data: %w", err)
	}

	// Convert string IDs to UUIDs for notification processing
	driverID, err := uuid.Parse(rideResp.DriverID)
	if err != nil {
		return fmt.Errorf("failed to parse driver ID: %w", err)
	}
	
	passengerID, err := uuid.Parse(rideResp.PassengerID)
	if err != nil {
		return fmt.Errorf("failed to parse passenger ID: %w", err)
	}

	// Create notifications for both driver and passenger with ride_started type
	driverNotification := &notification.UserNotification{
		UserID:    driverID,
		Type:      notification.NotificationTypeRideStarted,
		Data:      rideResp,
		Timestamp: time.Now(),
	}

	passengerNotification := &notification.UserNotification{
		UserID:    passengerID,
		Type:      notification.NotificationTypeRideStarted,
		Data:      rideResp,
		Timestamp: time.Now(),
	}

	if err := uc.processAndStoreNotification(ctx, driverNotification); err != nil {
		return err
	}

	return uc.processAndStoreNotification(ctx, passengerNotification)
}

// ProcessRideStarted processes ride started events
func (uc *NotificationUC) ProcessRideStarted(ctx context.Context, eventData []byte) error {
	var rideData ride.Ride
	if err := json.Unmarshal(eventData, &rideData); err != nil {
		return fmt.Errorf("failed to unmarshal ride started data: %w", err)
	}

	// Create notifications for both driver and passenger
	driverNotification := &notification.UserNotification{
		UserID:    rideData.DriverID,
		Type:      notification.NotificationTypeRideStarted,
		Data:      rideData,
		Timestamp: time.Now(),
	}

	passengerNotification := &notification.UserNotification{
		UserID:    rideData.PassengerID,
		Type:      notification.NotificationTypeRideStarted,
		Data:      rideData,
		Timestamp: time.Now(),
	}

	if err := uc.processAndStoreNotification(ctx, driverNotification); err != nil {
		return err
	}

	return uc.processAndStoreNotification(ctx, passengerNotification)
}

// ProcessRidePickupArrived processes ride pickup arrived events
func (uc *NotificationUC) ProcessRidePickupArrived(ctx context.Context, eventData []byte) error {
	var rideData ride.Ride
	if err := json.Unmarshal(eventData, &rideData); err != nil {
		return fmt.Errorf("failed to unmarshal ride pickup arrived data: %w", err)
	}

	// Notify passenger that driver has arrived
	notification := &notification.UserNotification{
		UserID:    rideData.PassengerID,
		Type:      notification.NotificationTypeRidePickupArrived,
		Data:      rideData,
		Timestamp: time.Now(),
	}

	return uc.processAndStoreNotification(ctx, notification)
}

// ProcessRideCompleted processes ride completed events (sends payment_processed notifications)
func (uc *NotificationUC) ProcessRideCompleted(ctx context.Context, eventData []byte) error {
	var rideCompleteData ride.RideComplete
	if err := json.Unmarshal(eventData, &rideCompleteData); err != nil {
		return fmt.Errorf("failed to unmarshal ride completed data: %w", err)
	}

	// Create payment_processed notifications for both driver and passenger
	driverNotification := &notification.UserNotification{
		UserID:    rideCompleteData.Ride.DriverID,
		Type:      notification.NotificationTypePaymentProcessed,
		Data:      rideCompleteData,
		Timestamp: time.Now(),
	}

	passengerNotification := &notification.UserNotification{
		UserID:    rideCompleteData.Ride.PassengerID,
		Type:      notification.NotificationTypePaymentProcessed,
		Data:      rideCompleteData,
		Timestamp: time.Now(),
	}

	if err := uc.processAndStoreNotification(ctx, driverNotification); err != nil {
		return err
	}

	return uc.processAndStoreNotification(ctx, passengerNotification)
}

// ProcessRideCancelled processes ride cancelled events
func (uc *NotificationUC) ProcessRideCancelled(ctx context.Context, eventData []byte) error {
	var rideData ride.Ride
	if err := json.Unmarshal(eventData, &rideData); err != nil {
		return fmt.Errorf("failed to unmarshal ride cancelled data: %w", err)
	}

	// Create notifications for both driver and passenger
	driverNotification := &notification.UserNotification{
		UserID:    rideData.DriverID,
		Type:      notification.NotificationTypeRideCancelled,
		Data:      rideData,
		Timestamp: time.Now(),
	}

	passengerNotification := &notification.UserNotification{
		UserID:    rideData.PassengerID,
		Type:      notification.NotificationTypeRideCancelled,
		Data:      rideData,
		Timestamp: time.Now(),
	}

	if err := uc.processAndStoreNotification(ctx, driverNotification); err != nil {
		return err
	}

	return uc.processAndStoreNotification(ctx, passengerNotification)
}

// ProcessPaymentProcessed processes payment processed events
func (uc *NotificationUC) ProcessPaymentProcessed(ctx context.Context, eventData []byte) error {
	var rideCompleteData ride.RideComplete
	if err := json.Unmarshal(eventData, &rideCompleteData); err != nil {
		return fmt.Errorf("failed to unmarshal payment processed data: %w", err)
	}

	// Create notifications for both driver and passenger using ride information
	driverNotification := &notification.UserNotification{
		UserID:    rideCompleteData.Ride.DriverID,
		Type:      notification.NotificationTypePaymentProcessed,
		Data:      rideCompleteData,
		Timestamp: time.Now(),
	}

	passengerNotification := &notification.UserNotification{
		UserID:    rideCompleteData.Ride.PassengerID,
		Type:      notification.NotificationTypePaymentProcessed,
		Data:      rideCompleteData,
		Timestamp: time.Now(),
	}

	if err := uc.processAndStoreNotification(ctx, driverNotification); err != nil {
		return err
	}

	return uc.processAndStoreNotification(ctx, passengerNotification)
}

// GetNotificationHistory retrieves notification history for a user
func (uc *NotificationUC) GetNotificationHistory(ctx context.Context, userID string, limit, offset int) ([]*notification.NotificationHistory, error) {
	return uc.notificationRepo.GetNotificationHistory(ctx, userID, limit, offset)
}

// GetUndeliveredNotifications retrieves undelivered notifications for a user
func (uc *NotificationUC) GetUndeliveredNotifications(ctx context.Context, userID string) ([]*notification.NotificationHistory, error) {
	return uc.notificationRepo.GetUndeliveredNotifications(ctx, userID)
}

// MarkNotificationDelivered marks a notification as delivered
func (uc *NotificationUC) MarkNotificationDelivered(ctx context.Context, notificationID string) error {
	return uc.notificationRepo.MarkNotificationDelivered(ctx, notificationID)
}

// SendNotificationToGateway sends a notification to all gateways via multi-gateway broadcasting
func (uc *NotificationUC) SendNotificationToGateway(ctx context.Context, notification *notification.UserNotification) error {
	// Map notification type to WebSocket event
	wsEvent := uc.mapNotificationToWSEvent(notification.Type)
	
	// Marshal notification data to JSON
	notificationData, err := json.Marshal(notification.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal notification data: %w", err)
	}
	
	// Create user-targeted broadcast message
	broadcastMsg := &websocket.WSUserBroadcast{
		WSBroadcastMessage: websocket.WSBroadcastMessage{
			MessageID: uuid.New().String(),
			Event:     wsEvent,
			Data:      json.RawMessage(notificationData),
			Timestamp: time.Now(),
			Source:    "notification-service",
			Priority:  5,
		},
		TargetUsers: []string{notification.UserID.String()},
	}
	
	// Serialize broadcast message
	data, err := json.Marshal(broadcastMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal broadcast message: %w", err)
	}
	
	// Publish to multi-gateway subject (all gateways will receive this)
	opts := natspkg.PublishOptions{
		Subject: nats.SubjectWSBroadcastUser,
		Data:    data,
		MsgID:   broadcastMsg.MessageID,
		Timeout: 10 * time.Second,
	}
	
	if err := uc.natsClient.PublishWithOptions(opts); err != nil {
		uc.logger.Error("Failed to publish multi-gateway notification",
			slog.String("user_id", notification.UserID.String()),
			slog.String("event", wsEvent),
			slog.Any("error", err))
		return fmt.Errorf("failed to publish notification: %w", err)
	}
	
	uc.logger.Info("Multi-gateway notification published successfully",
		slog.String("user_id", notification.UserID.String()),
		slog.String("event", wsEvent),
		slog.String("message_id", broadcastMsg.MessageID))
	
	return nil
}

// mapNotificationToWSEvent maps notification types to WebSocket events
func (uc *NotificationUC) mapNotificationToWSEvent(notificationType string) string {
	eventMap := map[string]string{
		notification.NotificationTypeMatchProposal:  websocket.EventMatchConfirm,
		notification.NotificationTypeMatchAccepted:  websocket.EventMatchConfirm,
		notification.NotificationTypeMatchRejected:  websocket.EventMatchRejected,
		notification.NotificationTypeRideStarted:    websocket.EventRideStarted,
		notification.NotificationTypeRideCompleted:  websocket.EventRideCompleted,
		notification.NotificationTypePaymentProcessed: websocket.EventPaymentProcessed,
		// Add more mappings as needed
	}
	
	if event, exists := eventMap[notificationType]; exists {
		return event
	}
	
	// Default fallback
	return "notification"
}

// BroadcastToRole sends notifications to all users with specific roles across all gateways
func (uc *NotificationUC) BroadcastToRole(ctx context.Context, roles []string, event string, data interface{}) error {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal broadcast data: %w", err)
	}
	
	broadcastMsg := &websocket.WSRoleBroadcast{
		WSBroadcastMessage: websocket.WSBroadcastMessage{
			MessageID: uuid.New().String(),
			Event:     event,
			Data:      json.RawMessage(dataBytes),
			Timestamp: time.Now(),
			Source:    "notification-service",
			Priority:  5,
		},
		TargetRoles: roles,
	}
	
	msgData, err := json.Marshal(broadcastMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal role broadcast: %w", err)
	}
	
	opts := natspkg.PublishOptions{
		Subject: nats.SubjectWSBroadcastRole,
		Data:    msgData,
		MsgID:   broadcastMsg.MessageID,
		Timeout: 10 * time.Second,
	}
	
	return uc.natsClient.PublishWithOptions(opts)
}

// BroadcastToAll sends notifications to all connected users across all gateways
func (uc *NotificationUC) BroadcastToAll(ctx context.Context, event string, data interface{}) error {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal broadcast data: %w", err)
	}
	
	broadcastMsg := &websocket.WSBroadcastMessage{
		MessageID: uuid.New().String(),
		Event:     event,
		Data:      json.RawMessage(dataBytes),
		Timestamp: time.Now(),
		Source:    "notification-service",
		Priority:  5,
	}
	
	msgData, err := json.Marshal(broadcastMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal all broadcast: %w", err)
	}
	
	opts := natspkg.PublishOptions{
		Subject: nats.SubjectWSBroadcastAll,
		Data:    msgData,
		MsgID:   broadcastMsg.MessageID,
		Timeout: 10 * time.Second,
	}
	
	return uc.natsClient.PublishWithOptions(opts)
}

// processAndStoreNotification is a helper method to process and store notifications
func (uc *NotificationUC) processAndStoreNotification(ctx context.Context, userNotification *notification.UserNotification) error {
	// Convert to notification history  
	if repo, ok := uc.notificationRepo.(*repository.NotificationRepo); ok {
		notificationHistory, err := repo.CreateNotificationFromEvent(userNotification)
		if err != nil {
			uc.logger.Error("Failed to create notification from event", slog.Any("error", err))
			return err
		}
		
		// Store in database
		if err := uc.notificationRepo.StoreNotificationHistory(ctx, notificationHistory); err != nil {
			uc.logger.Error("Failed to store notification history", slog.Any("error", err))
			return err
		}
	} else {
		// For interface compatibility, create notification history manually
		dataBytes, err := json.Marshal(userNotification.Data)
		if err != nil {
			uc.logger.Error("Failed to marshal notification data", slog.Any("error", err))
			return err
		}

		notificationHistory := &notification.NotificationHistory{
			UserID:    userNotification.UserID,
			Type:      userNotification.Type,
			Data:      json.RawMessage(dataBytes),
			Timestamp: userNotification.Timestamp,
			Delivered: false,
			CreatedAt: userNotification.Timestamp,
			UpdatedAt: userNotification.Timestamp,
		}
		
		// Store in database
		if err := uc.notificationRepo.StoreNotificationHistory(ctx, notificationHistory); err != nil {
			uc.logger.Error("Failed to store notification history", slog.Any("error", err))
			return err
		}
	}

	// Send to gateway
	if err := uc.SendNotificationToGateway(ctx, userNotification); err != nil {
		uc.logger.Error("Failed to send notification to gateway", slog.Any("error", err))
		// Don't return error here - notification is stored, delivery can be retried
	}

	uc.logger.Info("Notification processed successfully",
		slog.String("type", userNotification.Type),
		slog.String("user_id", userNotification.UserID.String()),
	)

	return nil
}
