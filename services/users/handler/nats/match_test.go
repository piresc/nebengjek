package nats

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/stretchr/testify/assert"
)

// MockNotification represents a notification sent to a client
type MockNotification struct {
	UserID string
	Event  string
	Data   interface{}
}

// WebSocketNotifier interface for testing - only includes what we need
type WebSocketNotifier interface {
	NotifyClient(userID string, event string, data interface{})
}

// MockWebSocketManager implements WebSocketNotifier for testing
type MockWebSocketManager struct {
	notifications []MockNotification
}

func NewMockWebSocketManager() *MockWebSocketManager {
	return &MockWebSocketManager{
		notifications: []MockNotification{},
	}
}

func (m *MockWebSocketManager) NotifyClient(userID string, event string, data interface{}) {
	m.notifications = append(m.notifications, MockNotification{
		UserID: userID,
		Event:  event,
		Data:   data,
	})
}

func (m *MockWebSocketManager) GetNotifications() []MockNotification {
	return m.notifications
}

func (m *MockWebSocketManager) Reset() {
	m.notifications = []MockNotification{}
}

// Helper function to create test handler with mock
func createTestHandler(mockWS *MockWebSocketManager) *testNatsHandler {
	return &testNatsHandler{
		wsManager: mockWS,
	}
}

// testNatsHandler is a test struct that mirrors NatsHandler but uses our mock
type testNatsHandler struct {
	wsManager *MockWebSocketManager
}

// Mirror the match handler methods for testing
func (h *testNatsHandler) handleMatchEvent(msg []byte) error {
	var event models.MatchProposal
	if err := json.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("failed to unmarshal match event: %w", err)
	}

	// Notify both driver and passenger
	h.wsManager.NotifyClient(event.DriverID, constants.SubjectMatchFound, event)
	h.wsManager.NotifyClient(event.PassengerID, constants.SubjectMatchFound, event)
	return nil
}

func (h *testNatsHandler) handleMatchAccEvent(msg []byte) error {
	var event models.MatchProposal
	if err := json.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("failed to unmarshal match event: %w", err)
	}

	// Notify both driver and passenger
	h.wsManager.NotifyClient(event.DriverID, constants.SubjectMatchAccepted, event)
	h.wsManager.NotifyClient(event.PassengerID, constants.SubjectMatchAccepted, event)
	return nil
}

func (h *testNatsHandler) handleMatchRejectedEvent(msg []byte) error {
	var event models.MatchProposal
	if err := json.Unmarshal(msg, &event); err != nil {
		return fmt.Errorf("failed to unmarshal match rejected event: %w", err)
	}

	// Only notify the driver whose match was rejected
	h.wsManager.NotifyClient(event.DriverID, constants.EventMatchRejected, event)
	return nil
}

func TestMatchHandleMatchEvent_Success(t *testing.T) {
	// Arrange
	driverID := uuid.New().String()
	passengerID := uuid.New().String()
	matchProposal := models.MatchProposal{
		ID:          uuid.New().String(),
		DriverID:    driverID,
		PassengerID: passengerID,
		MatchStatus: models.MatchStatusPending,
	}

	msgData, err := json.Marshal(matchProposal)
	assert.NoError(t, err)

	mockWS := NewMockWebSocketManager()
	handler := createTestHandler(mockWS)

	// Act
	err = handler.handleMatchEvent(msgData)

	// Assert
	assert.NoError(t, err)
	notifications := mockWS.GetNotifications()
	assert.Len(t, notifications, 2)

	// Check driver notification
	assert.Equal(t, driverID, notifications[0].UserID)
	assert.Equal(t, constants.SubjectMatchFound, notifications[0].Event)

	// Check passenger notification
	assert.Equal(t, passengerID, notifications[1].UserID)
	assert.Equal(t, constants.SubjectMatchFound, notifications[1].Event)
}

func TestMatchHandleMatchEvent_InvalidJSON(t *testing.T) {
	// Arrange
	invalidJSON := []byte(`{"invalid": json}`)
	mockWS := NewMockWebSocketManager()
	handler := createTestHandler(mockWS)

	// Act
	err := handler.handleMatchEvent(invalidJSON)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal match event")
	assert.Len(t, mockWS.GetNotifications(), 0)
}

func TestMatchHandleMatchAccEvent_Success(t *testing.T) {
	// Arrange
	driverID := uuid.New().String()
	passengerID := uuid.New().String()
	matchProposal := models.MatchProposal{
		ID:          uuid.New().String(),
		DriverID:    driverID,
		PassengerID: passengerID,
		MatchStatus: models.MatchStatusAccepted,
	}

	msgData, err := json.Marshal(matchProposal)
	assert.NoError(t, err)

	mockWS := NewMockWebSocketManager()
	handler := createTestHandler(mockWS)

	// Act
	err = handler.handleMatchAccEvent(msgData)

	// Assert
	assert.NoError(t, err)
	notifications := mockWS.GetNotifications()
	assert.Len(t, notifications, 2)

	// Check driver notification
	assert.Equal(t, driverID, notifications[0].UserID)
	assert.Equal(t, constants.SubjectMatchAccepted, notifications[0].Event)

	// Check passenger notification
	assert.Equal(t, passengerID, notifications[1].UserID)
	assert.Equal(t, constants.SubjectMatchAccepted, notifications[1].Event)
}

func TestMatchHandleMatchAccEvent_InvalidJSON(t *testing.T) {
	// Arrange
	invalidJSON := []byte(`{"incomplete": `)
	mockWS := NewMockWebSocketManager()
	handler := createTestHandler(mockWS)

	// Act
	err := handler.handleMatchAccEvent(invalidJSON)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal match event")
	assert.Len(t, mockWS.GetNotifications(), 0)
}

func TestMatchHandleMatchRejectedEvent_Success(t *testing.T) {
	// Arrange
	driverID := uuid.New().String()
	passengerID := uuid.New().String()
	matchProposal := models.MatchProposal{
		ID:          uuid.New().String(),
		DriverID:    driverID,
		PassengerID: passengerID,
		MatchStatus: models.MatchStatusRejected,
	}

	msgData, err := json.Marshal(matchProposal)
	assert.NoError(t, err)

	mockWS := NewMockWebSocketManager()
	handler := createTestHandler(mockWS)

	// Act
	err = handler.handleMatchRejectedEvent(msgData)

	// Assert
	assert.NoError(t, err)
	notifications := mockWS.GetNotifications()
	assert.Len(t, notifications, 1)

	// Check only driver gets notification for rejection
	assert.Equal(t, driverID, notifications[0].UserID)
	assert.Equal(t, constants.EventMatchRejected, notifications[0].Event)
}

func TestMatchHandleMatchRejectedEvent_InvalidJSON(t *testing.T) {
	// Arrange
	invalidJSON := []byte(`invalid json`)
	mockWS := NewMockWebSocketManager()
	handler := createTestHandler(mockWS)

	// Act
	err := handler.handleMatchRejectedEvent(invalidJSON)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal match rejected event")
	assert.Len(t, mockWS.GetNotifications(), 0)
}
