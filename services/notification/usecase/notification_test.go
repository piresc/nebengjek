package usecase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"log/slog"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// MockNotificationRepo is a mock implementation of the NotificationRepo interface
type MockNotificationRepo struct {
	mock.Mock
}

func (m *MockNotificationRepo) StoreNotificationHistory(ctx context.Context, notification *models.NotificationHistory) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

func (m *MockNotificationRepo) GetNotificationHistory(ctx context.Context, userID string, limit, offset int) ([]*models.NotificationHistory, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]*models.NotificationHistory), args.Error(1)
}

func (m *MockNotificationRepo) MarkNotificationDelivered(ctx context.Context, notificationID string) error {
	args := m.Called(ctx, notificationID)
	return args.Error(0)
}

func (m *MockNotificationRepo) GetUndeliveredNotifications(ctx context.Context, userID string) ([]*models.NotificationHistory, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*models.NotificationHistory), args.Error(1)
}

func TestNewNotificationUC(t *testing.T) {
	mockRepo := &MockNotificationRepo{}
	mockLogger := slog.Default()

	uc := NewNotificationUC(mockRepo, nil, mockLogger)

	assert.NotNil(t, uc)
	assert.Equal(t, mockRepo, uc.notificationRepo)
	assert.Nil(t, uc.natsClient) // NATS client can be nil for testing
	assert.Equal(t, mockLogger, uc.logger)
}

func TestNotificationUC_ProcessMatchProposal(t *testing.T) {
	mockRepo := &MockNotificationRepo{}
	mockLogger := slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelDebug}))
	uc := NewNotificationUC(mockRepo, nil, mockLogger) // NATS client is nil for testing

	ctx := context.Background()
	driverID := uuid.New()
	passengerID := uuid.New()

	matchProposal := models.MatchProposal{
		ID:             "match-123",
		DriverID:       driverID.String(),
		PassengerID:    passengerID.String(),
		UserLocation:   models.Location{Latitude: -6.2088, Longitude: 106.8456},
		DriverLocation: models.Location{Latitude: -6.1751, Longitude: 106.8650},
		TargetLocation: models.Location{Latitude: -6.2297, Longitude: 106.8295},
		MatchStatus:    models.MatchStatusPending,
	}

	eventData, err := json.Marshal(matchProposal)
	require.NoError(t, err)

	// Execute - should panic due to nil NATS client
	assert.Panics(t, func() {
		uc.ProcessMatchProposal(ctx, eventData)
	})
}

func TestNotificationUC_ProcessMatchAccepted(t *testing.T) {
	mockRepo := &MockNotificationRepo{}
	mockLogger := slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelDebug}))
	uc := NewNotificationUC(mockRepo, nil, mockLogger) // NATS client is nil for testing

	ctx := context.Background()
	driverID := uuid.New()
	passengerID := uuid.New()

	matchProposal := models.MatchProposal{
		ID:             "match-123",
		DriverID:       driverID.String(),
		PassengerID:    passengerID.String(),
		UserLocation:   models.Location{Latitude: -6.2088, Longitude: 106.8456},
		DriverLocation: models.Location{Latitude: -6.1751, Longitude: 106.8650},
		TargetLocation: models.Location{Latitude: -6.2297, Longitude: 106.8295},
		MatchStatus:    models.MatchStatusAccepted,
	}

	eventData, err := json.Marshal(matchProposal)
	require.NoError(t, err)

	// Execute - should panic due to nil NATS client
	assert.Panics(t, func() {
		uc.ProcessMatchAccepted(ctx, eventData)
	})
}

func TestNotificationUC_ProcessMatchRejected(t *testing.T) {
	mockRepo := &MockNotificationRepo{}
	mockLogger := slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelDebug}))
	uc := NewNotificationUC(mockRepo, nil, mockLogger) // NATS client is nil for testing

	ctx := context.Background()
	driverID := uuid.New()
	passengerID := uuid.New()

	matchProposal := models.MatchProposal{
		ID:             "match-123",
		DriverID:       driverID.String(),
		PassengerID:    passengerID.String(),
		UserLocation:   models.Location{Latitude: -6.2088, Longitude: 106.8456},
		DriverLocation: models.Location{Latitude: -6.1751, Longitude: 106.8650},
		TargetLocation: models.Location{Latitude: -6.2297, Longitude: 106.8295},
		MatchStatus:    models.MatchStatusRejected,
	}

	eventData, err := json.Marshal(matchProposal)
	require.NoError(t, err)

	// Execute - should panic due to nil NATS client
	assert.Panics(t, func() {
		uc.ProcessMatchRejected(ctx, eventData)
	})
}

func TestNotificationUC_ProcessRidePickup(t *testing.T) {
	mockRepo := &MockNotificationRepo{}
	mockLogger := slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelDebug}))
	uc := NewNotificationUC(mockRepo, nil, mockLogger) // NATS client is nil for testing

	ctx := context.Background()
	driverID := uuid.New()
	passengerID := uuid.New()

	rideResp := models.RideResp{
		RideID:      uuid.New().String(),
		DriverID:    driverID.String(),
		PassengerID: passengerID.String(),
		Status:      "pickup",
	}

	eventData, err := json.Marshal(rideResp)
	require.NoError(t, err)

	// Execute - should panic due to nil NATS client
	assert.Panics(t, func() {
		uc.ProcessRidePickup(ctx, eventData)
	})
}

func TestNotificationUC_ProcessRideStarted(t *testing.T) {
	mockRepo := &MockNotificationRepo{}
	mockLogger := slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelDebug}))
	uc := NewNotificationUC(mockRepo, nil, mockLogger) // NATS client is nil for testing

	ctx := context.Background()
	driverID := uuid.New()
	passengerID := uuid.New()

	rideData := models.Ride{
		RideID:      uuid.New(),
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      "started",
	}

	eventData, err := json.Marshal(rideData)
	require.NoError(t, err)

	// Execute - should panic due to nil NATS client
	assert.Panics(t, func() {
		uc.ProcessRideStarted(ctx, eventData)
	})
}

func TestNotificationUC_ProcessRidePickupArrived(t *testing.T) {
	mockRepo := &MockNotificationRepo{}
	mockLogger := slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelDebug}))
	uc := NewNotificationUC(mockRepo, nil, mockLogger) // NATS client is nil for testing

	ctx := context.Background()
	passengerID := uuid.New()

	rideData := models.Ride{
		RideID:      uuid.New(),
		PassengerID: passengerID,
		Status:      "arrived",
	}

	eventData, err := json.Marshal(rideData)
	require.NoError(t, err)

	// Execute - should panic due to nil NATS client
	assert.Panics(t, func() {
		uc.ProcessRidePickupArrived(ctx, eventData)
	})
}

func TestNotificationUC_ProcessRideCompleted(t *testing.T) {
	mockRepo := &MockNotificationRepo{}
	mockLogger := slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelDebug}))
	uc := NewNotificationUC(mockRepo, nil, mockLogger) // NATS client is nil for testing

	ctx := context.Background()
	driverID := uuid.New()
	passengerID := uuid.New()

	rideComplete := models.RideComplete{
		Ride: models.Ride{
			RideID:      uuid.New(),
			DriverID:    driverID,
			PassengerID: passengerID,
			Status:      "completed",
		},
	}

	eventData, err := json.Marshal(rideComplete)
	require.NoError(t, err)

	// Execute - should panic due to nil NATS client
	assert.Panics(t, func() {
		uc.ProcessRideCompleted(ctx, eventData)
	})
}

func TestNotificationUC_ProcessRideCancelled(t *testing.T) {
	mockRepo := &MockNotificationRepo{}
	mockLogger := slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelDebug}))
	uc := NewNotificationUC(mockRepo, nil, mockLogger) // NATS client is nil for testing

	ctx := context.Background()
	driverID := uuid.New()
	passengerID := uuid.New()

	rideData := models.Ride{
		RideID:      uuid.New(),
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      "cancelled",
	}

	eventData, err := json.Marshal(rideData)
	require.NoError(t, err)

	// Execute - should panic due to nil NATS client
	assert.Panics(t, func() {
		uc.ProcessRideCancelled(ctx, eventData)
	})
}

func TestNotificationUC_ProcessPaymentProcessed(t *testing.T) {
	mockRepo := &MockNotificationRepo{}
	mockLogger := slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelDebug}))
	uc := NewNotificationUC(mockRepo, nil, mockLogger) // NATS client is nil for testing

	ctx := context.Background()
	driverID := uuid.New()
	passengerID := uuid.New()

	rideComplete := models.RideComplete{
		Ride: models.Ride{
			RideID:      uuid.New(),
			DriverID:    driverID,
			PassengerID: passengerID,
			Status:      "completed",
		},
	}

	eventData, err := json.Marshal(rideComplete)
	require.NoError(t, err)

	// Execute - should panic due to nil NATS client
	assert.Panics(t, func() {
		uc.ProcessPaymentProcessed(ctx, eventData)
	})
}

func TestNotificationUC_GetNotificationHistory(t *testing.T) {
	mockRepo := &MockNotificationRepo{}
	mockLogger := slog.Default()
	uc := NewNotificationUC(mockRepo, nil, mockLogger)

	ctx := context.Background()
	userID := uuid.New().String()
	limit := 10
	offset := 0

	expectedHistory := []*models.NotificationHistory{
		{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Type:      models.NotificationTypeMatchProposal,
			Timestamp: time.Now(),
			Delivered: true,
		},
	}

	// Setup expectations
	mockRepo.On("GetNotificationHistory", ctx, userID, limit, offset).
		Return(expectedHistory, nil)

	// Execute
	result, err := uc.GetNotificationHistory(ctx, userID, limit, offset)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, expectedHistory, result)
	mockRepo.AssertExpectations(t)
}

func TestNotificationUC_GetUndeliveredNotifications(t *testing.T) {
	mockRepo := &MockNotificationRepo{}
	mockLogger := slog.Default()
	uc := NewNotificationUC(mockRepo, nil, mockLogger)

	ctx := context.Background()
	userID := uuid.New().String()

	expectedNotifications := []*models.NotificationHistory{
		{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Type:      models.NotificationTypeMatchProposal,
			Timestamp: time.Now(),
			Delivered: false,
		},
	}

	// Setup expectations
	mockRepo.On("GetUndeliveredNotifications", ctx, userID).
		Return(expectedNotifications, nil)

	// Execute
	result, err := uc.GetUndeliveredNotifications(ctx, userID)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, expectedNotifications, result)
	mockRepo.AssertExpectations(t)
}

func TestNotificationUC_MarkNotificationDelivered(t *testing.T) {
	mockRepo := &MockNotificationRepo{}
	mockLogger := slog.Default()
	uc := NewNotificationUC(mockRepo, nil, mockLogger)

	ctx := context.Background()
	notificationID := uuid.New().String()

	// Setup expectations
	mockRepo.On("MarkNotificationDelivered", ctx, notificationID).
		Return(nil)

	// Execute
	err := uc.MarkNotificationDelivered(ctx, notificationID)

	// Verify
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestNotificationUC_SendNotificationToGateway_NilNATS(t *testing.T) {
	mockRepo := &MockNotificationRepo{}
	mockLogger := slog.Default()
	uc := NewNotificationUC(mockRepo, nil, mockLogger) // NATS client is nil

	ctx := context.Background()
	userID := uuid.New()

	notification := &models.UserNotification{
		UserID:    userID,
		Type:      models.NotificationTypeMatchProposal,
		Data:      models.MatchProposal{ID: "match-123"},
		Timestamp: time.Now(),
	}

	// Execute - should panic due to nil NATS client
	assert.Panics(t, func() {
		uc.SendNotificationToGateway(ctx, notification)
	})
}

func TestNotificationUC_BroadcastToRole_NilNATS(t *testing.T) {
	mockRepo := &MockNotificationRepo{}
	mockLogger := slog.Default()
	uc := NewNotificationUC(mockRepo, nil, mockLogger) // NATS client is nil

	ctx := context.Background()
	roles := []string{"driver", "passenger"}
	event := "test_event"
	data := map[string]string{"message": "test"}

	// Execute - should panic due to nil NATS client
	assert.Panics(t, func() {
		uc.BroadcastToRole(ctx, roles, event, data)
	})
}

func TestNotificationUC_BroadcastToAll_NilNATS(t *testing.T) {
	mockRepo := &MockNotificationRepo{}
	mockLogger := slog.Default()
	uc := NewNotificationUC(mockRepo, nil, mockLogger) // NATS client is nil

	ctx := context.Background()
	event := "test_event"
	data := map[string]string{"message": "test"}

	// Execute - should panic due to nil NATS client
	assert.Panics(t, func() {
		uc.BroadcastToAll(ctx, event, data)
	})
}

func TestNotificationUC_ProcessMatchProposal_InvalidJSON(t *testing.T) {
	mockRepo := &MockNotificationRepo{}
	mockLogger := slog.Default()
	uc := NewNotificationUC(mockRepo, nil, mockLogger)

	ctx := context.Background()
	invalidJSON := []byte("invalid json")

	// Execute
	err := uc.ProcessMatchProposal(ctx, invalidJSON)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal match proposal data")
}

func TestNotificationUC_ProcessRidePickup_InvalidUUID(t *testing.T) {
	mockRepo := &MockNotificationRepo{}
	mockLogger := slog.Default()
	uc := NewNotificationUC(mockRepo, nil, mockLogger)

	ctx := context.Background()
	rideResp := models.RideResp{
		RideID:      uuid.New().String(),
		DriverID:    "invalid-uuid",
		PassengerID: uuid.New().String(),
		Status:      "pickup",
	}

	eventData, err := json.Marshal(rideResp)
	require.NoError(t, err)

	// Execute
	err = uc.ProcessRidePickup(ctx, eventData)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse driver ID")
}

func TestNotificationUC_mapNotificationToWSEvent(t *testing.T) {
	mockRepo := &MockNotificationRepo{}
	mockLogger := slog.Default()
	uc := NewNotificationUC(mockRepo, nil, mockLogger)

	testCases := []struct {
		notificationType string
		expectedEvent    string
	}{
		{models.NotificationTypeMatchProposal, constants.EventMatchConfirm},
		{models.NotificationTypeMatchAccepted, constants.EventMatchConfirm},
		{models.NotificationTypeMatchRejected, constants.EventMatchRejected},
		{models.NotificationTypeRideStarted, constants.EventRideStarted},
		{models.NotificationTypeRideCompleted, constants.EventRideCompleted},
		{models.NotificationTypePaymentProcessed, constants.EventPaymentProcessed},
		{"unknown_type", "notification"},
	}

	for _, tc := range testCases {
		result := uc.mapNotificationToWSEvent(tc.notificationType)
		assert.Equal(t, tc.expectedEvent, result, "Failed for type: %s", tc.notificationType)
	}
}