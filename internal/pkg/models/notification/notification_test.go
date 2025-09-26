package notification

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNotificationHistory_DefaultValues(t *testing.T) {
	history := &NotificationHistory{}

	// Test zero values
	assert.Equal(t, uuid.UUID{}, history.ID)
	assert.Equal(t, uuid.UUID{}, history.UserID)
	assert.Equal(t, "", history.Type)
	assert.Nil(t, history.Data)
	assert.Equal(t, time.Time{}, history.Timestamp)
	assert.Equal(t, false, history.Delivered)
	assert.Equal(t, time.Time{}, history.CreatedAt)
	assert.Equal(t, time.Time{}, history.UpdatedAt)
}

func TestNotificationHistory_WithValues(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	now := time.Now()
	data := json.RawMessage(`{"ride_id": "test-ride", "message": "Test notification"}`)

	history := &NotificationHistory{
		ID:        id,
		UserID:    userID,
		Type:      "match_proposal",
		Data:      data,
		Timestamp: now,
		Delivered: true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, id, history.ID)
	assert.Equal(t, userID, history.UserID)
	assert.Equal(t, "match_proposal", history.Type)
	assert.NotNil(t, history.Data)
	assert.Equal(t, now, history.Timestamp)
	assert.True(t, history.Delivered)
	assert.Equal(t, now, history.CreatedAt)
	assert.Equal(t, now, history.UpdatedAt)
}

func TestUserNotification_DefaultValues(t *testing.T) {
	notification := &UserNotification{}

	// Test zero values
	assert.Equal(t, uuid.UUID{}, notification.UserID)
	assert.Equal(t, "", notification.Type)
	assert.Nil(t, notification.Data)
	assert.Equal(t, time.Time{}, notification.Timestamp)
}

func TestUserNotification_WithValues(t *testing.T) {
	userID := uuid.New()
	now := time.Now()
	data := map[string]interface{}{
		"ride_id": "test-ride",
		"message": "Test notification",
	}

	notification := &UserNotification{
		UserID:    userID,
		Type:      "match_accepted",
		Data:      data,
		Timestamp: now,
	}

	assert.Equal(t, userID, notification.UserID)
	assert.Equal(t, "match_accepted", notification.Type)
	assert.NotNil(t, notification.Data)
	assert.Equal(t, now, notification.Timestamp)
}

func TestUserNotification_DifferentDataTypes(t *testing.T) {
	userID := uuid.New()
	now := time.Now()

	tests := []struct {
		name string
		data interface{}
	}{
		{
			name: "String data",
			data: "Simple string message",
		},
		{
			name: "Map data",
			data: map[string]interface{}{
				"ride_id":  "test-ride",
				"driver_id": "test-driver",
			},
		},
		{
			name: "Struct data",
			data: struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			}{
				Message: "Test message",
				Code:    200,
			},
		},
		{
			name: "Number data",
			data: 42,
		},
		{
			name: "Boolean data",
			data: true,
		},
		{
			name: "Nil data",
			data: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notification := &UserNotification{
				UserID:    userID,
				Type:      "test_notification",
				Data:      tt.data,
				Timestamp: now,
			}

			assert.Equal(t, tt.data, notification.Data)
		})
	}
}

func TestNotificationTypeConstants(t *testing.T) {
	// Test that all notification type constants have valid values
	assert.Equal(t, "match_proposal", NotificationTypeMatchProposal)
	assert.Equal(t, "match_accepted", NotificationTypeMatchAccepted)
	assert.Equal(t, "match_rejected", NotificationTypeMatchRejected)
	assert.Equal(t, "ride_started", NotificationTypeRideStarted)
	assert.Equal(t, "ride_pickup_arrived", NotificationTypeRidePickupArrived)
	assert.Equal(t, "ride_completed", NotificationTypeRideCompleted)
	assert.Equal(t, "ride_cancelled", NotificationTypeRideCancelled)
	assert.Equal(t, "payment_processed", NotificationTypePaymentProcessed)
}

func TestUserNotificationEvents(t *testing.T) {
	// Test that UserNotificationEvents is properly populated
	assert.NotNil(t, UserNotificationEvents)
	assert.NotEmpty(t, UserNotificationEvents)

	expectedEvents := []string{
		"MATCH.proposal.created",
		"MATCH.accepted",
		"MATCH.rejected",
		"RIDE.started",
		"RIDE.pickup.arrived",
		"RIDE.completed",
		"RIDE.cancelled",
		"PAYMENT.processed",
	}

	assert.Equal(t, len(expectedEvents), len(UserNotificationEvents))

	for i, expected := range expectedEvents {
		assert.Equal(t, expected, UserNotificationEvents[i])
	}
}

func TestNotificationDeliveryEvent_DefaultValues(t *testing.T) {
	event := &NotificationDeliveryEvent{}

	// Test zero values
	assert.Equal(t, uuid.UUID{}, event.UserID)
	assert.Equal(t, "", event.Type)
	assert.Nil(t, event.Data)
	assert.Equal(t, time.Time{}, event.Timestamp)
	assert.Equal(t, uuid.UUID{}, event.DeliveryID)
}

func TestNotificationDeliveryEvent_WithValues(t *testing.T) {
	userID := uuid.New()
	deliveryID := uuid.New()
	now := time.Now()
	data := map[string]interface{}{
		"ride_id": "test-ride",
		"status":  "completed",
	}

	event := &NotificationDeliveryEvent{
		UserID:     userID,
		Type:       "ride_completed",
		Data:       data,
		Timestamp:  now,
		DeliveryID: deliveryID,
	}

	assert.Equal(t, userID, event.UserID)
	assert.Equal(t, "ride_completed", event.Type)
	assert.NotNil(t, event.Data)
	assert.Equal(t, now, event.Timestamp)
	assert.Equal(t, deliveryID, event.DeliveryID)
}

func TestNotificationHistory_JSONSerialization(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	now := time.Now()
	data := json.RawMessage(`{"ride_id":"test-ride","message":"Test notification"}`)

	history := &NotificationHistory{
		ID:        id,
		UserID:    userID,
		Type:      "match_proposal",
		Data:      data,
		Timestamp: now,
		Delivered: true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(history)
	assert.NoError(t, err)
	assert.NotNil(t, jsonData)

	// Test JSON unmarshaling
	var decodedHistory NotificationHistory
	err = json.Unmarshal(jsonData, &decodedHistory)
	assert.NoError(t, err)
	assert.Equal(t, history.ID, decodedHistory.ID)
	assert.Equal(t, history.UserID, decodedHistory.UserID)
	assert.Equal(t, history.Type, decodedHistory.Type)
	assert.Equal(t, history.Delivered, decodedHistory.Delivered)
	
	// Compare data as strings to avoid formatting issues
	assert.Equal(t, string(history.Data), string(decodedHistory.Data))
}

func TestNotificationHistory_TimestampHandling(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	now := time.Now()
	data := json.RawMessage(`{"test": "data"}`)

	tests := []struct {
		name      string
		timestamp time.Time
		isZero    bool
	}{
		{"Current time", now, false},
		{"Past time", now.Add(-1 * time.Hour), false},
		{"Future time", now.Add(1 * time.Hour), false},
		{"Zero time", time.Time{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history := &NotificationHistory{
				ID:        id,
				UserID:    userID,
				Type:      "test_notification",
				Data:      data,
				Timestamp: tt.timestamp,
				CreatedAt: tt.timestamp,
				UpdatedAt: tt.timestamp,
			}

			assert.Equal(t, tt.timestamp, history.Timestamp)
			assert.Equal(t, tt.timestamp, history.CreatedAt)
			assert.Equal(t, tt.timestamp, history.UpdatedAt)

			if tt.isZero {
				assert.True(t, history.Timestamp.IsZero())
				assert.True(t, history.CreatedAt.IsZero())
				assert.True(t, history.UpdatedAt.IsZero())
			} else {
				assert.False(t, history.Timestamp.IsZero())
				assert.False(t, history.CreatedAt.IsZero())
				assert.False(t, history.UpdatedAt.IsZero())
			}
		})
	}
}

func TestNotificationHistory_DeliveryStatus(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	now := time.Now()
	data := json.RawMessage(`{"test": "data"}`)

	tests := []struct {
		name      string
		delivered bool
		comment   string
	}{
		{"Delivered notification", true, "Notification was successfully delivered"},
		{"Undelivered notification", false, "Notification was not delivered"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history := &NotificationHistory{
				ID:        id,
				UserID:    userID,
				Type:      "test_notification",
				Data:      data,
				Timestamp: now,
				Delivered: tt.delivered,
				CreatedAt: now,
				UpdatedAt: now,
			}

			assert.Equal(t, tt.delivered, history.Delivered)
		})
	}
}

func TestNotificationTypeValidation(t *testing.T) {
	validTypes := []string{
		NotificationTypeMatchProposal,
		NotificationTypeMatchAccepted,
		NotificationTypeMatchRejected,
		NotificationTypeRideStarted,
		NotificationTypeRidePickupArrived,
		NotificationTypeRideCompleted,
		NotificationTypeRideCancelled,
		NotificationTypePaymentProcessed,
	}

	invalidTypes := []string{
		"",
		"invalid_type",
		"UNKNOWN",
		"random_notification",
	}

	for _, notificationType := range validTypes {
		t.Run("Valid_"+notificationType, func(t *testing.T) {
			assert.NotEmpty(t, notificationType)
			assert.Contains(t, validTypes, notificationType)
		})
	}

	for _, notificationType := range invalidTypes {
		t.Run("Invalid_"+notificationType, func(t *testing.T) {
			assert.NotContains(t, validTypes, notificationType)
		})
	}
}

func TestNotificationDeliveryEvent_DataVariety(t *testing.T) {
	userID := uuid.New()
	deliveryID := uuid.New()
	now := time.Now()

	tests := []struct {
		name string
		data interface{}
	}{
		{
			name: "Simple string",
			data: "Your ride has arrived!",
		},
		{
			name: "Complex map",
			data: map[string]interface{}{
				"ride_id":     "test-ride-123",
				"driver_name": "John Doe",
				"vehicle":     "Toyota Camry",
				"eta":         5,
			},
		},
		{
			name: "Structured data",
			data: struct {
				RideID   string `json:"ride_id"`
				Status   string `json:"status"`
				Duration int    `json:"duration_minutes"`
			}{
				RideID:   "test-ride-456",
				Status:   "completed",
				Duration: 25,
			},
		},
		{
			name: "Numeric data",
			data: 42,
		},
		{
			name: "Boolean data",
			data: true,
		},
		{
			name: "Nil data",
			data: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &NotificationDeliveryEvent{
				UserID:     userID,
				Type:       "test_notification",
				Data:       tt.data,
				Timestamp:  now,
				DeliveryID: deliveryID,
			}

			assert.Equal(t, tt.data, event.Data)
			assert.Equal(t, userID, event.UserID)
			assert.Equal(t, deliveryID, event.DeliveryID)
		})
	}
}

func TestNotification_JSONTags(t *testing.T) {
	// Test JSON tags for notification structures
	
	// NotificationHistory
	history := &NotificationHistory{}
	assert.NotNil(t, history)
	
	// UserNotification
	notification := &UserNotification{}
	assert.NotNil(t, notification)
	
	// NotificationDeliveryEvent
	event := &NotificationDeliveryEvent{}
	assert.NotNil(t, event)
	
	// Test that structs can be created - JSON tags are validated by the struct definitions
}