package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// NotificationHistory represents a notification record in the database
type NotificationHistory struct {
	ID        uuid.UUID       `json:"id" db:"id"`
	UserID    uuid.UUID       `json:"user_id" db:"user_id"`
	Type      string          `json:"type" db:"type"`
	Data      json.RawMessage `json:"data" db:"data"`
	Timestamp time.Time       `json:"timestamp" db:"timestamp"`
	Delivered bool            `json:"delivered" db:"delivered"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt time.Time       `json:"updated_at" db:"updated_at"`
}

// UserNotification represents a notification to be sent to a user
type UserNotification struct {
	UserID    uuid.UUID   `json:"user_id"`
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// Notification event types
const (
	NotificationTypeMatchProposal    = "match_proposal"
	NotificationTypeMatchAccepted    = "match_accepted"
	NotificationTypeMatchRejected    = "match_rejected"
	NotificationTypeRideStarted      = "ride_started"
	NotificationTypeRidePickupArrived = "ride_pickup_arrived"
	NotificationTypeRideCompleted    = "ride_completed"
	NotificationTypeRideCancelled    = "ride_cancelled"
	NotificationTypePaymentProcessed = "payment_processed"
)

// NATS subjects for selective notification consumption
var UserNotificationEvents = []string{
	"MATCH.proposal.created",
	"MATCH.accepted",
	"MATCH.rejected",
	"RIDE.started",
	"RIDE.pickup.arrived",
	"RIDE.completed",
	"RIDE.cancelled",
	"PAYMENT.processed",
}

// NotificationDeliveryEvent represents the event sent to gateway for WebSocket delivery
type NotificationDeliveryEvent struct {
	UserID       uuid.UUID   `json:"user_id"`
	Type         string      `json:"type"`
	Data         interface{} `json:"data"`
	Timestamp    time.Time   `json:"timestamp"`
	DeliveryID   uuid.UUID   `json:"delivery_id"`
}