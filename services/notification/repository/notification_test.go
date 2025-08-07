package repository

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestCreateNotificationFromEvent(t *testing.T) {
	repo := &NotificationRepo{}
	
	userID := uuid.New()
	now := time.Now()
	
	testData := map[string]interface{}{
		"message": "Test notification",
		"type":    "test",
	}
	
	userNotification := &models.UserNotification{
		UserID:    userID,
		Type:      models.NotificationTypeMatchProposal,
		Data:      testData,
		Timestamp: now,
	}
	
	notification, err := repo.CreateNotificationFromEvent(userNotification)
	
	assert.NoError(t, err)
	assert.Equal(t, userID, notification.UserID)
	assert.Equal(t, models.NotificationTypeMatchProposal, notification.Type)
	assert.Equal(t, now, notification.Timestamp)
	assert.False(t, notification.Delivered)
	
	// Verify data marshaling
	var unmarshaledData map[string]interface{}
	err = json.Unmarshal(notification.Data, &unmarshaledData)
	assert.NoError(t, err)
	assert.Equal(t, "Test notification", unmarshaledData["message"])
	assert.Equal(t, "test", unmarshaledData["type"])
}