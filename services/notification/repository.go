package notification

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models/notification"
)

//go:generate mockgen -destination=mocks/mock_repository.go -package=mocks github.com/piresc/nebengjek/services/notification NotificationRepo

// NotificationRepo defines the notification repository interface for database operations
type NotificationRepo interface {
	// Store notification history in PostgreSQL
	StoreNotificationHistory(ctx context.Context, notification *notification.NotificationHistory) error
	
	// Get notification history for a user
	GetNotificationHistory(ctx context.Context, userID string, limit, offset int) ([]*notification.NotificationHistory, error)
	
	// Mark notification as delivered
	MarkNotificationDelivered(ctx context.Context, notificationID string) error
	
	// Get undelivered notifications for a user
	GetUndeliveredNotifications(ctx context.Context, userID string) ([]*notification.NotificationHistory, error)
}