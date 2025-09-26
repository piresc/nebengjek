package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	ridemodels "github.com/piresc/nebengjek/internal/pkg/models/ride"
	locationmodels "github.com/piresc/nebengjek/internal/pkg/models/location"
	natsconstants "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/stretchr/testify/assert"
)

// Integration tests for NATS gateway functionality
func TestNATSGateway_PublishRidePickupEvent_Success(t *testing.T) {
	// Arrange
	mockNATS := NewMockNATSPublisher()
	gateway := NewNATSGateway(mockNATS)

	ctx := context.Background()
	rideEvent := &ridemodels.RidePickupEvent{
		RideID:      uuid.New().String(),
		DriverID:    uuid.New().String(),
		PassengerID: uuid.New().String(),
		DriverLocation: locationmodels.Location{
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
	assert.Equal(t, natsconstants.SubjectRidePickup, messages[0].Subject)

	// Verify message content
	var publishedEvent ridemodels.RidePickupEvent
	err = json.Unmarshal(messages[0].Data, &publishedEvent)
	assert.NoError(t, err)
	assert.Equal(t, rideEvent.RideID, publishedEvent.RideID)
	assert.Equal(t, rideEvent.DriverID, publishedEvent.DriverID)
	assert.Equal(t, rideEvent.PassengerID, publishedEvent.PassengerID)
}

func TestNATSGateway_PublishRidePickupEvent_PublishError(t *testing.T) {
	// Arrange
	mockNATS := NewMockNATSPublisher()
	mockNATS.SetPublishError(errors.New("NATS publish failed"))
	gateway := NewNATSGateway(mockNATS)

	ctx := context.Background()
	rideEvent := &ridemodels.RidePickupEvent{
		RideID:      uuid.New().String(),
		DriverID:    uuid.New().String(),
		PassengerID: uuid.New().String(),
		DriverLocation: locationmodels.Location{
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
	mockNATS := NewMockNATSPublisher()
	gateway := NewNATSGateway(mockNATS)

	ctx := context.Background()
	completeEvent := &ridemodels.RideCompleteEvent{
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
	assert.Equal(t, natsconstants.SubjectRideCompleted, messages[0].Subject)

	// Verify message content
	var publishedEvent ridemodels.RideCompleteEvent
	err = json.Unmarshal(messages[0].Data, &publishedEvent)
	assert.NoError(t, err)
	assert.Equal(t, completeEvent.RideID, publishedEvent.RideID)
	assert.Equal(t, completeEvent.AdjustmentFactor, publishedEvent.AdjustmentFactor)
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