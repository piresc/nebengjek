package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/rides/mocks"
	"github.com/stretchr/testify/assert"
)

// Integration tests for NATS gateway functionality
func TestNATSGateway_PublishRidePickupEvent_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	mockNATS := NewMockNATSPublisher()
	gateway := NewNATSGateway(mockRideUC, mockNATS)

	ctx := context.Background()
	rideEvent := &models.RidePickupEvent{
		RideID:      uuid.New().String(),
		DriverID:    uuid.New().String(),
		PassengerID: uuid.New().String(),
		DriverLocation: models.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
		Timestamp: time.Now(),
	}

	// Act
	err := gateway.PublishRidePickupEvent(ctx, rideEvent)

	// Assert
	assert.NoError(t, err)

	// Verify message was published
	messages := mockNATS.GetPublishedMessages()
	assert.Len(t, messages, 1)
	assert.Equal(t, constants.SubjectRidePickup, messages[0].Subject)

	// Verify message content
	var publishedEvent models.RidePickupEvent
	err = json.Unmarshal(messages[0].Data, &publishedEvent)
	assert.NoError(t, err)
	assert.Equal(t, rideEvent.RideID, publishedEvent.RideID)
	assert.Equal(t, rideEvent.DriverID, publishedEvent.DriverID)
	assert.Equal(t, rideEvent.PassengerID, publishedEvent.PassengerID)
}

func TestNATSGateway_PublishRidePickupEvent_PublishError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	mockNATS := NewMockNATSPublisher()
	mockNATS.SetPublishError(errors.New("NATS publish failed"))
	gateway := NewNATSGateway(mockRideUC, mockNATS)

	ctx := context.Background()
	rideEvent := &models.RidePickupEvent{
		RideID:      uuid.New().String(),
		DriverID:    uuid.New().String(),
		PassengerID: uuid.New().String(),
		DriverLocation: models.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
		Timestamp:  time.Now(),
	}

	// Act
	err := gateway.PublishRidePickupEvent(ctx, rideEvent)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NATS publish failed")
}

func TestNATSGateway_PublishRideCompleteEvent_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	mockNATS := NewMockNATSPublisher()
	gateway := NewNATSGateway(mockRideUC, mockNATS)

	ctx := context.Background()
	completeEvent := &models.RideCompleteEvent{
		RideID:           "ride-123",
		AdjustmentFactor: 0.9,
	}

	// Act
	err := gateway.PublishRideCompleteEvent(ctx, completeEvent)

	// Assert
	assert.NoError(t, err)

	// Verify message was published
	messages := mockNATS.GetPublishedMessages()
	assert.Len(t, messages, 1)
	assert.Equal(t, constants.SubjectRideCompleted, messages[0].Subject)

	// Verify message content
	var publishedEvent models.RideCompleteEvent
	err = json.Unmarshal(messages[0].Data, &publishedEvent)
	assert.NoError(t, err)
	assert.Equal(t, completeEvent.RideID, publishedEvent.RideID)
	assert.Equal(t, completeEvent.AdjustmentFactor, publishedEvent.AdjustmentFactor)
}

func TestNATSGateway_HandleMatchEvent_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	mockNATS := NewMockNATSPublisher()
	gateway := NewNATSGateway(mockRideUC, mockNATS)

	matchEvent := &models.MatchProposal{
		ID:          uuid.New().String(),
		DriverID:    uuid.New().String(),
		PassengerID: uuid.New().String(),
		UserLocation: models.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
		DriverLocation: models.Location{
			Latitude:  -6.2188,
			Longitude: 106.8556,
		},
		MatchStatus: models.MatchStatusAccepted,
	}

	// Set up mock expectations
	mockRideUC.EXPECT().
		CreateRide(gomock.Any(), gomock.Any()).
		Return(nil)

	// Act
	err := gateway.handleMatchEvent(context.Background(), *matchEvent)

	// Assert
	assert.NoError(t, err)
}

func TestNATSGateway_HandleMatchEvent_InvalidJSON(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	mockNATS := NewMockNATSPublisher()
	gateway := NewNATSGateway(mockRideUC, mockNATS)

	invalidMatchEvent := models.MatchProposal{}

	// Set up mock expectations - even invalid events will try to create ride
	mockRideUC.EXPECT().
		CreateRide(gomock.Any(), gomock.Any()).
		Return(errors.New("validation error"))

	// Act
	err := gateway.handleMatchEvent(context.Background(), invalidMatchEvent)

	// Assert
	assert.Error(t, err)
}

func TestNATSGateway_HandleMatchEvent_CreateRideError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	mockNATS := NewMockNATSPublisher()
	gateway := NewNATSGateway(mockRideUC, mockNATS)

	matchEvent := models.MatchProposal{
		ID:          uuid.New().String(),
		DriverID:    uuid.New().String(),
		PassengerID: uuid.New().String(),
		MatchStatus: models.MatchStatusAccepted,
	}

	// Set up mock expectations
	mockRideUC.EXPECT().
		CreateRide(gomock.Any(), gomock.Any()).
		Return(errors.New("database error"))

	// Act
	err := gateway.handleMatchEvent(context.Background(), matchEvent)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestNATSGateway_HandleLocationEvent_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	mockNATS := NewMockNATSPublisher()
	gateway := NewNATSGateway(mockRideUC, mockNATS)

	locationEvent := models.LocationAggregate{
		RideID:    uuid.New().String(),
		Distance:  15.5,
		Latitude:  -6.2088,
		Longitude: 106.8456,
	}

	// Set up mock expectations
	mockRideUC.EXPECT().
		ProcessBillingUpdate(gomock.Any(), locationEvent.RideID, gomock.Any()).
		Return(nil)

	// Act
	err := gateway.handleLocationEvent(context.Background(), locationEvent)

	// Assert
	assert.NoError(t, err)
}

func TestNATSGateway_HandleLocationEvent_InvalidJSON(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	mockNATS := NewMockNATSPublisher()
	gateway := NewNATSGateway(mockRideUC, mockNATS)

	invalidLocationEvent := models.LocationAggregate{}

	// Set up mock expectations - even invalid events will try to process billing
	mockRideUC.EXPECT().
		ProcessBillingUpdate(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errors.New("validation error"))

	// Act
	err := gateway.handleLocationEvent(context.Background(), invalidLocationEvent)

	// Assert
	assert.Error(t, err)
}

func TestNATSGateway_HandleLocationEvent_UpdateError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	mockNATS := NewMockNATSPublisher()
	gateway := NewNATSGateway(mockRideUC, mockNATS)

	locationEvent := models.LocationAggregate{
		RideID:    uuid.New().String(),
		Distance:  15.5,
		Latitude:  -6.2088,
		Longitude: 106.8456,
	}

	// Set up mock expectations
	mockRideUC.EXPECT().
		ProcessBillingUpdate(gomock.Any(), locationEvent.RideID, gomock.Any()).
		Return(errors.New("update failed"))

	// Act
	err := gateway.handleLocationEvent(context.Background(), locationEvent)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update failed")
}

// MockNATSPublisher is a mock implementation for testing
type MockNATSPublisher struct {
	publishedMessages []MockMessage
	publishError      error
}

type MockMessage struct {
	Subject string
	Data    []byte
}

// Publish implements NATSPublisher interface
func (m *MockNATSPublisher) Publish(subject string, data []byte) error {
	if m.publishError != nil {
		return m.publishError
	}
	m.publishedMessages = append(m.publishedMessages, MockMessage{
		Subject: subject,
		Data:    data,
	})
	return nil
}

// GetPublishedMessages returns all published messages
func (m *MockNATSPublisher) GetPublishedMessages() []MockMessage {
	return m.publishedMessages
}

func NewMockNATSPublisher() *MockNATSPublisher {
	return &MockNATSPublisher{
		publishedMessages: make([]MockMessage, 0),
	}
}

func (m *MockNATSPublisher) SetPublishError(err error) {
	m.publishError = err
}