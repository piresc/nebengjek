package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/notification"
	"github.com/piresc/nebengjek/services/notification/repository"
)

// NotificationUC implements the notification usecase interface
type NotificationUC struct {
	notificationRepo notification.NotificationRepo
	natsClient       *natspkg.Client
	logger           *slog.Logger
}

// NewNotificationUC creates a new notification usecase
func NewNotificationUC(
	notificationRepo notification.NotificationRepo,
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
	var matchProposal models.MatchProposal
	if err := json.Unmarshal(eventData, &matchProposal); err != nil {
		return fmt.Errorf("failed to unmarshal match proposal data: %w", err)
	}

	// Create notification for driver
	driverNotification := &models.UserNotification{
		UserID:    uuid.MustParse(matchProposal.DriverID),
		Type:      models.NotificationTypeMatchProposal,
		Data:      matchProposal,
		Timestamp: time.Now(),
	}

	// Create notification for passenger
	passengerNotification := &models.UserNotification{
		UserID:    uuid.MustParse(matchProposal.PassengerID),
		Type:      models.NotificationTypeMatchProposal,
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
	var matchProposal models.MatchProposal
	if err := json.Unmarshal(eventData, &matchProposal); err != nil {
		return fmt.Errorf("failed to unmarshal match accepted data: %w", err)
	}

	// Create notifications for both driver and passenger
	driverNotification := &models.UserNotification{
		UserID:    uuid.MustParse(matchProposal.DriverID),
		Type:      models.NotificationTypeMatchAccepted,
		Data:      matchProposal,
		Timestamp: time.Now(),
	}

	passengerNotification := &models.UserNotification{
		UserID:    uuid.MustParse(matchProposal.PassengerID),
		Type:      models.NotificationTypeMatchAccepted,
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
	var matchProposal models.MatchProposal
	if err := json.Unmarshal(eventData, &matchProposal); err != nil {
		return fmt.Errorf("failed to unmarshal match rejected data: %w", err)
	}

	// Create notifications for both driver and passenger
	driverNotification := &models.UserNotification{
		UserID:    uuid.MustParse(matchProposal.DriverID),
		Type:      models.NotificationTypeMatchRejected,
		Data:      matchProposal,
		Timestamp: time.Now(),
	}

	passengerNotification := &models.UserNotification{
		UserID:    uuid.MustParse(matchProposal.PassengerID),
		Type:      models.NotificationTypeMatchRejected,
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
	var rideResp models.RideResp
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
	driverNotification := &models.UserNotification{
		UserID:    driverID,
		Type:      models.NotificationTypeRideStarted,
		Data:      rideResp,
		Timestamp: time.Now(),
	}

	passengerNotification := &models.UserNotification{
		UserID:    passengerID,
		Type:      models.NotificationTypeRideStarted,
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
	var rideData models.Ride
	if err := json.Unmarshal(eventData, &rideData); err != nil {
		return fmt.Errorf("failed to unmarshal ride started data: %w", err)
	}

	// Create notifications for both driver and passenger
	driverNotification := &models.UserNotification{
		UserID:    rideData.DriverID,
		Type:      models.NotificationTypeRideStarted,
		Data:      rideData,
		Timestamp: time.Now(),
	}

	passengerNotification := &models.UserNotification{
		UserID:    rideData.PassengerID,
		Type:      models.NotificationTypeRideStarted,
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
	var rideData models.Ride
	if err := json.Unmarshal(eventData, &rideData); err != nil {
		return fmt.Errorf("failed to unmarshal ride pickup arrived data: %w", err)
	}

	// Notify passenger that driver has arrived
	notification := &models.UserNotification{
		UserID:    rideData.PassengerID,
		Type:      models.NotificationTypeRidePickupArrived,
		Data:      rideData,
		Timestamp: time.Now(),
	}

	return uc.processAndStoreNotification(ctx, notification)
}

// ProcessRideCompleted processes ride completed events (sends payment_processed notifications)
func (uc *NotificationUC) ProcessRideCompleted(ctx context.Context, eventData []byte) error {
	var rideCompleteData models.RideComplete
	if err := json.Unmarshal(eventData, &rideCompleteData); err != nil {
		return fmt.Errorf("failed to unmarshal ride completed data: %w", err)
	}

	// Create payment_processed notifications for both driver and passenger
	driverNotification := &models.UserNotification{
		UserID:    rideCompleteData.Ride.DriverID,
		Type:      models.NotificationTypePaymentProcessed,
		Data:      rideCompleteData,
		Timestamp: time.Now(),
	}

	passengerNotification := &models.UserNotification{
		UserID:    rideCompleteData.Ride.PassengerID,
		Type:      models.NotificationTypePaymentProcessed,
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
	var rideData models.Ride
	if err := json.Unmarshal(eventData, &rideData); err != nil {
		return fmt.Errorf("failed to unmarshal ride cancelled data: %w", err)
	}

	// Create notifications for both driver and passenger
	driverNotification := &models.UserNotification{
		UserID:    rideData.DriverID,
		Type:      models.NotificationTypeRideCancelled,
		Data:      rideData,
		Timestamp: time.Now(),
	}

	passengerNotification := &models.UserNotification{
		UserID:    rideData.PassengerID,
		Type:      models.NotificationTypeRideCancelled,
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
	var rideCompleteData models.RideComplete
	if err := json.Unmarshal(eventData, &rideCompleteData); err != nil {
		return fmt.Errorf("failed to unmarshal payment processed data: %w", err)
	}

	// Create notifications for both driver and passenger using ride information
	driverNotification := &models.UserNotification{
		UserID:    rideCompleteData.Ride.DriverID,
		Type:      models.NotificationTypePaymentProcessed,
		Data:      rideCompleteData,
		Timestamp: time.Now(),
	}

	passengerNotification := &models.UserNotification{
		UserID:    rideCompleteData.Ride.PassengerID,
		Type:      models.NotificationTypePaymentProcessed,
		Data:      rideCompleteData,
		Timestamp: time.Now(),
	}

	if err := uc.processAndStoreNotification(ctx, driverNotification); err != nil {
		return err
	}

	return uc.processAndStoreNotification(ctx, passengerNotification)
}

// GetNotificationHistory retrieves notification history for a user
func (uc *NotificationUC) GetNotificationHistory(ctx context.Context, userID string, limit, offset int) ([]*models.NotificationHistory, error) {
	return uc.notificationRepo.GetNotificationHistory(ctx, userID, limit, offset)
}

// GetUndeliveredNotifications retrieves undelivered notifications for a user
func (uc *NotificationUC) GetUndeliveredNotifications(ctx context.Context, userID string) ([]*models.NotificationHistory, error) {
	return uc.notificationRepo.GetUndeliveredNotifications(ctx, userID)
}

// MarkNotificationDelivered marks a notification as delivered
func (uc *NotificationUC) MarkNotificationDelivered(ctx context.Context, notificationID string) error {
	return uc.notificationRepo.MarkNotificationDelivered(ctx, notificationID)
}

// SendNotificationToGateway sends a notification to the gateway via NATS JetStream
func (uc *NotificationUC) SendNotificationToGateway(ctx context.Context, notification *models.UserNotification) error {
	deliveryEvent := &models.NotificationDeliveryEvent{
		UserID:     notification.UserID,
		Type:       notification.Type,
		Data:       notification.Data,
		Timestamp:  notification.Timestamp,
		DeliveryID: uuid.New(),
	}

	// Marshal the event data
	eventData, err := json.Marshal(deliveryEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal delivery event: %w", err)
	}

	// Use JetStream publish with options for reliability
	opts := natspkg.PublishOptions{
		Subject: "NOTIFICATION.deliver",
		Data:    eventData,
		MsgID:   fmt.Sprintf("notification-deliver-%s-%d", deliveryEvent.DeliveryID.String(), time.Now().UnixNano()),
		Timeout: 10 * time.Second,
	}

	uc.logger.Info("Publishing notification to gateway via JetStream",
		slog.String("subject", opts.Subject),
		slog.String("user_id", notification.UserID.String()),
		slog.String("type", notification.Type),
		slog.String("msg_id", opts.MsgID))

	if err := uc.natsClient.PublishWithOptions(opts); err != nil {
		uc.logger.Error("Failed to publish notification to JetStream",
			slog.String("subject", opts.Subject),
			slog.String("user_id", notification.UserID.String()),
			slog.String("msg_id", opts.MsgID),
			slog.Any("error", err))
		return fmt.Errorf("failed to publish notification to JetStream: %w", err)
	}

	uc.logger.Info("Successfully published notification to JetStream",
		slog.String("subject", opts.Subject),
		slog.String("user_id", notification.UserID.String()),
		slog.String("msg_id", opts.MsgID))

	return nil
}

// processAndStoreNotification is a helper method to process and store notifications
func (uc *NotificationUC) processAndStoreNotification(ctx context.Context, notification *models.UserNotification) error {
	// Convert to notification history  
	if repo, ok := uc.notificationRepo.(*repository.NotificationRepo); ok {
		notificationHistory, err := repo.CreateNotificationFromEvent(notification)
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
		dataBytes, err := json.Marshal(notification.Data)
		if err != nil {
			uc.logger.Error("Failed to marshal notification data", slog.Any("error", err))
			return err
		}

		notificationHistory := &models.NotificationHistory{
			UserID:    notification.UserID,
			Type:      notification.Type,
			Data:      json.RawMessage(dataBytes),
			Timestamp: notification.Timestamp,
			Delivered: false,
			CreatedAt: notification.Timestamp,
			UpdatedAt: notification.Timestamp,
		}
		
		// Store in database
		if err := uc.notificationRepo.StoreNotificationHistory(ctx, notificationHistory); err != nil {
			uc.logger.Error("Failed to store notification history", slog.Any("error", err))
			return err
		}
	}

	// Send to gateway
	if err := uc.SendNotificationToGateway(ctx, notification); err != nil {
		uc.logger.Error("Failed to send notification to gateway", slog.Any("error", err))
		// Don't return error here - notification is stored, delivery can be retried
	}

	uc.logger.Info("Notification processed successfully",
		slog.String("type", notification.Type),
		slog.String("user_id", notification.UserID.String()),
	)

	return nil
}
