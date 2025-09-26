package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/piresc/nebengjek/internal/pkg/models/notification"
)

// NotificationRepo implements the notification repository interface
type NotificationRepo struct {
	db *sqlx.DB
}

// NewNotificationRepo creates a new notification repository
func NewNotificationRepo(db *sqlx.DB) *NotificationRepo {
	return &NotificationRepo{db: db}
}

// StoreNotificationHistory stores a notification in the database
func (r *NotificationRepo) StoreNotificationHistory(ctx context.Context, notification *notification.NotificationHistory) error {
	txn := newrelic.FromContext(ctx)
	dbCtx := newrelic.NewContext(ctx, txn)

	notification.ID = uuid.New()

	query := `
		INSERT INTO notification_history (id, user_id, type, data, timestamp, delivered, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(dbCtx, query,
		notification.ID,
		notification.UserID,
		notification.Type,
		notification.Data,
		notification.Timestamp,
		notification.Delivered,
		notification.CreatedAt,
		notification.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store notification history: %w", err)
	}

	return nil
}

// GetNotificationHistory retrieves notification history for a user
func (r *NotificationRepo) GetNotificationHistory(ctx context.Context, userID string, limit, offset int) ([]*notification.NotificationHistory, error) {
	txn := newrelic.FromContext(ctx)
	dbCtx := newrelic.NewContext(ctx, txn)

	query := `
		SELECT id, user_id, type, data, timestamp, delivered, created_at, updated_at
		FROM notification_history
		WHERE user_id = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(dbCtx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification history: %w", err)
	}
	defer rows.Close()

	var notifications []*notification.NotificationHistory
	for rows.Next() {
		var notification notification.NotificationHistory
		err := rows.Scan(
			&notification.ID,
			&notification.UserID,
			&notification.Type,
			&notification.Data,
			&notification.Timestamp,
			&notification.Delivered,
			&notification.CreatedAt,
			&notification.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, &notification)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading notification rows: %w", err)
	}

	return notifications, nil
}

// MarkNotificationDelivered marks a notification as delivered
func (r *NotificationRepo) MarkNotificationDelivered(ctx context.Context, notificationID string) error {
	txn := newrelic.FromContext(ctx)
	dbCtx := newrelic.NewContext(ctx, txn)

	query := `
		UPDATE notification_history 
		SET delivered = true, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(dbCtx, query, notificationID)
	if err != nil {
		return fmt.Errorf("failed to mark notification as delivered: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("notification not found")
	}

	return nil
}

// GetUndeliveredNotifications retrieves undelivered notifications for a user
func (r *NotificationRepo) GetUndeliveredNotifications(ctx context.Context, userID string) ([]*notification.NotificationHistory, error) {
	txn := newrelic.FromContext(ctx)
	dbCtx := newrelic.NewContext(ctx, txn)

	query := `
		SELECT id, user_id, type, data, timestamp, delivered, created_at, updated_at
		FROM notification_history
		WHERE user_id = $1 AND delivered = false
		ORDER BY timestamp ASC
	`

	rows, err := r.db.QueryContext(dbCtx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get undelivered notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*notification.NotificationHistory
	for rows.Next() {
		var notification notification.NotificationHistory
		err := rows.Scan(
			&notification.ID,
			&notification.UserID,
			&notification.Type,
			&notification.Data,
			&notification.Timestamp,
			&notification.Delivered,
			&notification.CreatedAt,
			&notification.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, &notification)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading notification rows: %w", err)
	}

	return notifications, nil
}

// CreateNotificationFromEvent creates a notification from a user notification event
func (r *NotificationRepo) CreateNotificationFromEvent(userNotification *notification.UserNotification) (*notification.NotificationHistory, error) {
	// Convert data to JSON
	dataBytes, err := json.Marshal(userNotification.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal notification data: %w", err)
	}

	notification := &notification.NotificationHistory{
		UserID:    userNotification.UserID,
		Type:      userNotification.Type,
		Data:      json.RawMessage(dataBytes),
		Timestamp: userNotification.Timestamp,
		Delivered: false,
		CreatedAt: userNotification.Timestamp,
		UpdatedAt: userNotification.Timestamp,
	}

	return notification, nil
}
