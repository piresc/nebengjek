package notification

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -package=mocks github.com/piresc/nebengjek/services/notification NotificationUC

// NotificationUC represents the notification usecase interface
type NotificationUC interface {
	// Process incoming NATS events and create notifications
	ProcessMatchProposal(ctx context.Context, eventData []byte) error
	ProcessMatchAccepted(ctx context.Context, eventData []byte) error
	ProcessMatchRejected(ctx context.Context, eventData []byte) error
	ProcessRidePickup(ctx context.Context, eventData []byte) error
	ProcessRideStarted(ctx context.Context, eventData []byte) error
	ProcessRidePickupArrived(ctx context.Context, eventData []byte) error
	ProcessRideCompleted(ctx context.Context, eventData []byte) error
	ProcessRideCancelled(ctx context.Context, eventData []byte) error
	ProcessPaymentProcessed(ctx context.Context, eventData []byte) error

	// Notification management
	GetNotificationHistory(ctx context.Context, userID string, limit, offset int) ([]*models.NotificationHistory, error)
	GetUndeliveredNotifications(ctx context.Context, userID string) ([]*models.NotificationHistory, error)
	MarkNotificationDelivered(ctx context.Context, notificationID string) error

	// Send notification to gateway
	SendNotificationToGateway(ctx context.Context, notification *models.UserNotification) error
}