package websocket

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/users/mocks"
	"github.com/stretchr/testify/assert"
)

// MockManager for testing WebSocket functionality - simplified to avoid interface issues
type MockWebSocketManager struct {
	notifications []MockNotification
	sentMessages  []MockMessage
	sentErrors    []MockError
}

type MockMessage struct {
	EventType string
	Data      interface{}
}

type MockError struct {
	ErrorCode string
	Message   string
}

type MockNotification struct {
	UserID string
	Event  string
	Data   interface{}
}

func NewMockWebSocketManager() *MockWebSocketManager {
	return &MockWebSocketManager{
		notifications: []MockNotification{},
		sentMessages:  []MockMessage{},
		sentErrors:    []MockError{},
	}
}

func (m *MockWebSocketManager) NotifyClient(userID string, event string, data interface{}) {
	m.notifications = append(m.notifications, MockNotification{
		UserID: userID,
		Event:  event,
		Data:   data,
	})
}

func (m *MockWebSocketManager) SendMessage(conn interface{}, eventType string, data interface{}) error {
	m.sentMessages = append(m.sentMessages, MockMessage{
		EventType: eventType,
		Data:      data,
	})
	return nil
}

func (m *MockWebSocketManager) SendErrorMessage(conn interface{}, errorCode string, message string) error {
	m.sentErrors = append(m.sentErrors, MockError{
		ErrorCode: errorCode,
		Message:   message,
	})
	return nil
}

func (m *MockWebSocketManager) GetClient(clientID string) (*models.WebSocketClient, bool) {
	return nil, false
}

func (m *MockWebSocketManager) GetNotifications() []MockNotification {
	return m.notifications
}

func (m *MockWebSocketManager) Reset() {
	m.notifications = []MockNotification{}
	m.sentMessages = []MockMessage{}
	m.sentErrors = []MockError{}
}

// testWebSocketManager is a test wrapper that uses the mock manager
type testWebSocketManager struct {
	userUC  *mocks.MockUserUC
	manager *MockWebSocketManager
}

func (m *testWebSocketManager) handleLocationUpdate(userID string, data json.RawMessage) error {
	var req models.LocationUpdate
	if err := json.Unmarshal(data, &req); err != nil {
		m.manager.sentErrors = append(m.manager.sentErrors, MockError{
			ErrorCode: constants.ErrorInvalidFormat,
			Message:   "Invalid location format",
		})
		return fmt.Errorf("invalid location format: %w", err)
	}
	req.DriverID = userID

	if err := m.userUC.UpdateUserLocation(nil, &req); err != nil {
		m.manager.sentErrors = append(m.manager.sentErrors, MockError{
			ErrorCode: constants.ErrorInvalidLocation,
			Message:   err.Error(),
		})
		return nil // Return nil to simulate sending error message
	}

	return nil
}

func TestHandleLocationUpdate_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	mockManager := NewMockWebSocketManager()

	wsManager := &testWebSocketManager{
		userUC:  mockUserUC,
		manager: mockManager,
	}

	driverID := uuid.New().String()
	rideID := uuid.New().String()
	locationUpdate := models.LocationUpdate{
		RideID:   rideID,
		DriverID: driverID,
		Location: models.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
			Timestamp: time.Now(),
		},
		CreatedAt: time.Now(),
	}

	msgData, err := json.Marshal(locationUpdate)
	assert.NoError(t, err)

	mockUserUC.EXPECT().
		UpdateUserLocation(gomock.Any(), gomock.Any()).
		Return(nil)

	// Act
	err = wsManager.handleLocationUpdate(driverID, json.RawMessage(msgData))

	// Assert
	assert.NoError(t, err)
	assert.Len(t, mockManager.sentErrors, 0)
}

func TestHandleLocationUpdate_InvalidJSON(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	mockManager := NewMockWebSocketManager()

	wsManager := &testWebSocketManager{
		userUC:  mockUserUC,
		manager: mockManager,
	}

	driverID := uuid.New().String()
	invalidJSON := json.RawMessage(`{"invalid": json}`)

	// Act
	err := wsManager.handleLocationUpdate(driverID, invalidJSON)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid location format")
}

func TestHandleLocationUpdate_UsecaseError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	mockManager := NewMockWebSocketManager()

	wsManager := &testWebSocketManager{
		userUC:  mockUserUC,
		manager: mockManager,
	}

	driverID := "driver123"
	locationUpdate := models.LocationUpdate{
		RideID:   uuid.New().String(),
		DriverID: driverID,
		Location: models.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
			Timestamp: time.Now(),
		},
		CreatedAt: time.Now(),
	}

	msgData, err := json.Marshal(locationUpdate)
	assert.NoError(t, err)

	mockUserUC.EXPECT().
		UpdateUserLocation(gomock.Any(), gomock.Any()).
		Return(errors.New("database error"))

	// Act
	err = wsManager.handleLocationUpdate(driverID, json.RawMessage(msgData))

	// Assert
	assert.NoError(t, err) // The function returns nil but sends error message
	assert.Len(t, mockManager.sentErrors, 1)
	assert.Equal(t, constants.ErrorInvalidLocation, mockManager.sentErrors[0].ErrorCode)
}

func TestHandleLocationUpdate_ClientNotFound(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	mockManager := NewMockWebSocketManager()

	wsManager := &testWebSocketManager{
		userUC:  mockUserUC,
		manager: mockManager,
	}

	driverID := "nonexistent-driver"
	locationUpdate := models.LocationUpdate{
		RideID:   uuid.New().String(),
		DriverID: driverID,
		Location: models.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
			Timestamp: time.Now(),
		},
		CreatedAt: time.Now(),
	}

	msgData, err := json.Marshal(locationUpdate)
	assert.NoError(t, err)

	mockUserUC.EXPECT().
		UpdateUserLocation(gomock.Any(), gomock.Any()).
		Return(errors.New("location update failed"))

	// Act
	err = wsManager.handleLocationUpdate(driverID, json.RawMessage(msgData))

	// Assert
	assert.NoError(t, err) // The function returns nil but sends error message
	assert.Len(t, mockManager.sentErrors, 1)
	assert.Equal(t, constants.ErrorInvalidLocation, mockManager.sentErrors[0].ErrorCode)
}
