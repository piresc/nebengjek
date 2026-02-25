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

	"github.com/piresc/nebengjek/internal/pkg/models/notification"
	matchmodels "github.com/piresc/nebengjek/internal/pkg/models/match"
	locationmodels "github.com/piresc/nebengjek/internal/pkg/models/location"
	ridemodels "github.com/piresc/nebengjek/internal/pkg/models/ride"
	wsconstants "github.com/piresc/nebengjek/internal/pkg/models/websocket"
)

// MockNotificationRepo is a mock implementation of the NotificationRepo interface
type MockNotificationRepo struct {
	mock.Mock
}

func (m *MockNotificationRepo) StoreNotificationHistory(ctx context.Context, notification *notification.NotificationHistory) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

func (m *MockNotificationRepo) GetNotificationHistory(ctx context.Context, userID string, limit, offset int) ([]*notification.NotificationHistory, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]*notification.NotificationHistory), args.Error(1)
}

func (m *MockNotificationRepo) MarkNotificationDelivered(ctx context.Context, notificationID string) error {
	args := m.Called(ctx, notificationID)
	return args.Error(0)
}

func (m *MockNotificationRepo) GetUndeliveredNotifications(ctx context.Context, userID string) ([]*notification.NotificationHistory, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*notification.NotificationHistory), args.Error(1)
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

	matchProposal := matchmodels.MatchProposal{
		ID:             "match-123",
		DriverID:       driverID.String(),
		PassengerID:    passengerID.String(),
		UserLocation:   locationmodels.Location{Latitude: -6.2088, Longitude: 106.8456},
		DriverLocation: locationmodels.Location{Latitude: -6.1751, Longitude: 106.8650},
		TargetLocation: locationmodels.Location{Latitude: -6.2297, Longitude: 106.8295},
		MatchStatus:    matchmodels.MatchStatusPending,
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

	matchProposal := matchmodels.MatchProposal{
		ID:             "match-123",
		DriverID:       driverID.String(),
		PassengerID:    passengerID.String(),
		UserLocation:   locationmodels.Location{Latitude: -6.2088, Longitude: 106.8456},
		DriverLocation: locationmodels.Location{Latitude: -6.1751, Longitude: 106.8650},
		TargetLocation: locationmodels.Location{Latitude: -6.2297, Longitude: 106.8295},
		MatchStatus:    matchmodels.MatchStatusAccepted,
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

	matchProposal := matchmodels.MatchProposal{
		ID:             "match-123",
		DriverID:       driverID.String(),
		PassengerID:    passengerID.String(),
		UserLocation:   locationmodels.Location{Latitude: -6.2088, Longitude: 106.8456},
		DriverLocation: locationmodels.Location{Latitude: -6.1751, Longitude: 106.8650},
		TargetLocation: locationmodels.Location{Latitude: -6.2297, Longitude: 106.8295},
		MatchStatus:    matchmodels.MatchStatusRejected,
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

	rideResp := ridemodels.RideResp{
		RideID:      uuid.New().String(),
		DriverID:    driverID.String(),
		PassengerID: passengerID.String(),
		Status:      "PICKUP",
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

	rideData := ridemodels.Ride{
		RideID:      uuid.New(),
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      "ONGOING",
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

	rideData := ridemodels.Ride{
		RideID:      uuid.New(),
		PassengerID: passengerID,
		Status:      "PICKUP",
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

	rideComplete := ridemodels.RideComplete{
		Ride: ridemodels.Ride{
			RideID:      uuid.New(),
			DriverID:    driverID,
			PassengerID: passengerID,
			Status:      "COMPLETED",
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

	rideData := ridemodels.Ride{
		RideID:      uuid.New(),
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      "PENDING",
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

	rideComplete := ridemodels.RideComplete{
		Ride: ridemodels.Ride{
			RideID:      uuid.New(),
			DriverID:    driverID,
			PassengerID: passengerID,
			Status:      "COMPLETED",
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

	expectedHistory := []*notification.NotificationHistory{
		{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Type:      notification.NotificationTypeMatchProposal,
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

	expectedNotifications := []*notification.NotificationHistory{
		{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Type:      notification.NotificationTypeMatchProposal,
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

	notification := &notification.UserNotification{
		UserID:    userID,
		Type:      notification.NotificationTypeMatchProposal,
		Data:      matchmodels.MatchProposal{ID: "match-123"},
		Timestamp: time.Now(),
	}

	// Execute - should panic due to nil NATS client
	assert.Panics(t, func() {
		uc.SendNotificationToGateway(ctx, notification)
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
	rideResp := ridemodels.RideResp{
		RideID:      uuid.New().String(),
		DriverID:    "invalid-uuid",
		PassengerID: uuid.New().String(),
		Status:      "PICKUP",
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
		{notification.NotificationTypeMatchProposal, wsconstants.EventMatchConfirm},
		{notification.NotificationTypeMatchAccepted, wsconstants.EventMatchConfirm},
		{notification.NotificationTypeMatchRejected, wsconstants.EventMatchRejected},
		{notification.NotificationTypeRideStarted, wsconstants.EventRideStarted},
		{notification.NotificationTypeRideCompleted, wsconstants.EventRideCompleted},
		{notification.NotificationTypePaymentProcessed, wsconstants.EventPaymentProcessed},
		{"unknown_type", "notification"},
	}

	for _, tc := range testCases {
		result := uc.mapNotificationToWSEvent(tc.notificationType)
		assert.Equal(t, tc.expectedEvent, result, "Failed for type: %s", tc.notificationType)
	}
}