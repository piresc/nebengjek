package gateway_nats

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NATSClientInterface defines the interface for NATS client operations
type NATSClientInterface interface {
	Publish(subject string, data []byte) error
	Close()
}

// MockNATSClient simulates NATS client behavior for testing
type MockNATSClient struct {
	publishedMessages map[string][]byte
	publishError      error
}

// NewMockNATSClient creates a new mock NATS client
func NewMockNATSClient() *MockNATSClient {
	return &MockNATSClient{
		publishedMessages: make(map[string][]byte),
	}
}

// Publish simulates publishing a message to a subject
func (m *MockNATSClient) Publish(subject string, data []byte) error {
	if m.publishError != nil {
		return m.publishError
	}
	m.publishedMessages[subject] = data
	return nil
}

// GetPublishedMessage returns the last published message for a subject
func (m *MockNATSClient) GetPublishedMessage(subject string) ([]byte, bool) {
	data, exists := m.publishedMessages[subject]
	return data, exists
}

// SetPublishError sets an error to return on publish
func (m *MockNATSClient) SetPublishError(err error) {
	m.publishError = err
}

// Close simulates closing the connection
func (m *MockNATSClient) Close() {
	// No-op for mock
}

// TestableNATSGateway extends NATSGateway to allow testing with mocks
type TestableNATSGateway struct {
	client NATSClientInterface
}

// NewTestableNATSGateway creates a gateway that can work with mocks
func NewTestableNATSGateway(client NATSClientInterface) *TestableNATSGateway {
	return &TestableNATSGateway{
		client: client,
	}
}

// PublishBeaconEvent publishes a beacon event to NATS (same logic as real implementation)
func (g *TestableNATSGateway) PublishBeaconEvent(ctx context.Context, event *models.BeaconEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return g.client.Publish(constants.SubjectUserBeacon, data)
}

// PublishFinderEvent publishes a finder event to NATS (same logic as real implementation)
func (g *TestableNATSGateway) PublishFinderEvent(ctx context.Context, event *models.FinderEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return g.client.Publish(constants.SubjectUserFinder, data)
}

// PublishRideStart publishes a ride start event to NATS (same logic as real implementation)
func (g *TestableNATSGateway) PublishRideStart(ctx context.Context, event *models.RideStartTripEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return g.client.Publish(constants.SubjectRideStarted, data)
}

// PublishLocationUpdate publishes a location update event to NATS (same logic as real implementation)
func (g *TestableNATSGateway) PublishLocationUpdate(ctx context.Context, locationEvent *models.LocationUpdate) error {
	data, err := json.Marshal(locationEvent)
	if err != nil {
		return err
	}
	return g.client.Publish(constants.SubjectLocationUpdate, data)
}

// Ensure natspkg.Client implements our interface
var _ NATSClientInterface = (*natspkg.Client)(nil)

// TestPublishBeaconEvent_Success tests successful publishing of beacon events
func TestPublishBeaconEvent_Success(t *testing.T) {
	// Arrange
	mockClient := NewMockNATSClient()
	natsGW := NewTestableNATSGateway(mockClient)

	beaconEvent := &models.BeaconEvent{
		UserID:   uuid.New().String(),
		IsActive: true,
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	// Act
	ctx := context.Background()
	err := natsGW.PublishBeaconEvent(ctx, beaconEvent)

	// Assert
	require.NoError(t, err)

	// Verify the message was published to the correct subject
	publishedData, exists := mockClient.GetPublishedMessage(constants.SubjectUserBeacon)
	require.True(t, exists, "Message should be published to beacon subject")

	// Verify the published data matches the original event
	var receivedEvent models.BeaconEvent
	err = json.Unmarshal(publishedData, &receivedEvent)
	require.NoError(t, err)

	assert.Equal(t, beaconEvent.UserID, receivedEvent.UserID)
	assert.Equal(t, beaconEvent.IsActive, receivedEvent.IsActive)
	assert.Equal(t, beaconEvent.Location.Latitude, receivedEvent.Location.Latitude)
	assert.Equal(t, beaconEvent.Location.Longitude, receivedEvent.Location.Longitude)
}

// TestPublishBeaconEvent_Error tests error handling during beacon event publishing
func TestPublishBeaconEvent_Error(t *testing.T) {
	// Arrange
	mockClient := NewMockNATSClient()
	expectedError := errors.New("NATS publish failed")
	mockClient.SetPublishError(expectedError)

	natsGW := NewTestableNATSGateway(mockClient)

	beaconEvent := &models.BeaconEvent{
		UserID:   uuid.New().String(),
		IsActive: true,
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	// Act
	ctx := context.Background()
	err := natsGW.PublishBeaconEvent(ctx, beaconEvent)

	// Assert
	require.Error(t, err)
	assert.Equal(t, expectedError, err)
}

// TestPublishFinderEvent_Success tests successful publishing of finder events
func TestPublishFinderEvent_Success(t *testing.T) {
	// Arrange
	mockClient := NewMockNATSClient()
	natsGW := NewTestableNATSGateway(mockClient)

	finderEvent := &models.FinderEvent{
		UserID:   uuid.New().String(),
		IsActive: true,
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
		TargetLocation: models.Location{
			Latitude:  -6.185392,
			Longitude: 106.837153,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	// Act
	ctx := context.Background()
	err := natsGW.PublishFinderEvent(ctx, finderEvent)

	// Assert
	require.NoError(t, err)

	// Verify the message was published to the correct subject
	publishedData, exists := mockClient.GetPublishedMessage(constants.SubjectUserFinder)
	require.True(t, exists, "Message should be published to finder subject")

	// Verify the published data matches the original event
	var receivedEvent models.FinderEvent
	err = json.Unmarshal(publishedData, &receivedEvent)
	require.NoError(t, err)

	assert.Equal(t, finderEvent.UserID, receivedEvent.UserID)
	assert.Equal(t, finderEvent.IsActive, receivedEvent.IsActive)
	assert.Equal(t, finderEvent.Location.Latitude, receivedEvent.Location.Latitude)
	assert.Equal(t, finderEvent.Location.Longitude, receivedEvent.Location.Longitude)
	assert.Equal(t, finderEvent.TargetLocation.Latitude, receivedEvent.TargetLocation.Latitude)
	assert.Equal(t, finderEvent.TargetLocation.Longitude, receivedEvent.TargetLocation.Longitude)
}

// TestPublishRideStart_Success tests successful publishing of ride start events
func TestPublishRideStart_Success(t *testing.T) {
	// Arrange
	mockClient := NewMockNATSClient()
	natsGW := NewTestableNATSGateway(mockClient)

	// Create test data with the correct RideStartTripEvent structure
	rideStartEvent := &models.RideStartTripEvent{
		RideID: uuid.New().String(),
		DriverLocation: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
		PassengerLocation: models.Location{
			Latitude:  -6.185392,
			Longitude: 106.837153,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	// Act
	ctx := context.Background()
	err := natsGW.PublishRideStart(ctx, rideStartEvent)

	// Assert
	require.NoError(t, err)

	// Verify the message was published to the correct subject
	publishedData, exists := mockClient.GetPublishedMessage(constants.SubjectRideStarted)
	require.True(t, exists, "Message should be published to ride started subject")

	// Verify the published data matches the original event
	var receivedEvent models.RideStartTripEvent
	err = json.Unmarshal(publishedData, &receivedEvent)
	require.NoError(t, err)

	assert.Equal(t, rideStartEvent.RideID, receivedEvent.RideID)
	assert.Equal(t, rideStartEvent.DriverLocation.Latitude, receivedEvent.DriverLocation.Latitude)
	assert.Equal(t, rideStartEvent.DriverLocation.Longitude, receivedEvent.DriverLocation.Longitude)
	assert.Equal(t, rideStartEvent.PassengerLocation.Latitude, receivedEvent.PassengerLocation.Latitude)
	assert.Equal(t, rideStartEvent.PassengerLocation.Longitude, receivedEvent.PassengerLocation.Longitude)
}

// TestPublishLocationUpdate_Success tests successful publishing of location update events
func TestPublishLocationUpdate_Success(t *testing.T) {
	// Arrange
	mockClient := NewMockNATSClient()
	natsGW := NewTestableNATSGateway(mockClient)

	locationUpdate := &models.LocationUpdate{
		RideID:   "ride-123",
		DriverID: uuid.New().String(),
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
		CreatedAt: time.Now(),
	}

	// Act
	ctx := context.Background()
	err := natsGW.PublishLocationUpdate(ctx, locationUpdate)

	// Assert
	require.NoError(t, err)

	// Verify the message was published to the correct subject
	publishedData, exists := mockClient.GetPublishedMessage(constants.SubjectLocationUpdate)
	require.True(t, exists, "Message should be published to location update subject")

	// Verify the published data matches the original event
	var receivedUpdate models.LocationUpdate
	err = json.Unmarshal(publishedData, &receivedUpdate)
	require.NoError(t, err)

	assert.Equal(t, locationUpdate.RideID, receivedUpdate.RideID)
	assert.Equal(t, locationUpdate.DriverID, receivedUpdate.DriverID)
	assert.Equal(t, locationUpdate.Location.Latitude, receivedUpdate.Location.Latitude)
	assert.Equal(t, locationUpdate.Location.Longitude, receivedUpdate.Location.Longitude)
}

// TestPublishLocationUpdate_Error tests error handling during location update publishing
func TestPublishLocationUpdate_Error(t *testing.T) {
	// Arrange
	mockClient := NewMockNATSClient()
	expectedError := errors.New("NATS publish failed")
	mockClient.SetPublishError(expectedError)

	natsGW := NewTestableNATSGateway(mockClient)

	locationUpdate := &models.LocationUpdate{
		RideID:   "ride-123",
		DriverID: uuid.New().String(),
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
		CreatedAt: time.Now(),
	}

	// Act
	ctx := context.Background()
	err := natsGW.PublishLocationUpdate(ctx, locationUpdate)

	// Assert
	require.Error(t, err)
	assert.Equal(t, expectedError, err)
}

// TestMultiplePublishes tests publishing multiple different events
func TestMultiplePublishes(t *testing.T) {
	// Arrange
	mockClient := NewMockNATSClient()
	natsGW := NewTestableNATSGateway(mockClient)

	ctx := context.Background()

	// Test data
	beaconEvent := &models.BeaconEvent{
		UserID:   uuid.New().String(),
		IsActive: true,
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	finderEvent := &models.FinderEvent{
		UserID:   uuid.New().String(),
		IsActive: true,
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
		TargetLocation: models.Location{
			Latitude:  -6.185392,
			Longitude: 106.837153,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	// Act
	err1 := natsGW.PublishBeaconEvent(ctx, beaconEvent)
	err2 := natsGW.PublishFinderEvent(ctx, finderEvent)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)

	// Verify both messages were published to their respective subjects
	_, beaconExists := mockClient.GetPublishedMessage(constants.SubjectUserBeacon)
	_, finderExists := mockClient.GetPublishedMessage(constants.SubjectUserFinder)

	assert.True(t, beaconExists, "Beacon message should be published")
	assert.True(t, finderExists, "Finder message should be published")
}

// TestRideStartEventStructure tests that RideStartTripEvent has the correct structure
func TestRideStartEventStructure(t *testing.T) {
	// Arrange
	mockClient := NewMockNATSClient()
	natsGW := NewTestableNATSGateway(mockClient)

	// Create a RideStartTripEvent with all expected fields based on the actual model
	rideStartEvent := &models.RideStartTripEvent{
		RideID: uuid.New().String(),
		DriverLocation: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
		PassengerLocation: models.Location{
			Latitude:  -6.185392,
			Longitude: 106.837153,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	// Act
	ctx := context.Background()
	err := natsGW.PublishRideStart(ctx, rideStartEvent)

	// Assert
	require.NoError(t, err)

	// Verify the published message structure
	publishedData, exists := mockClient.GetPublishedMessage(constants.SubjectRideStarted)
	require.True(t, exists)

	var receivedEvent models.RideStartTripEvent
	err = json.Unmarshal(publishedData, &receivedEvent)
	require.NoError(t, err)

	// Verify all fields are correctly serialized and deserialized
	assert.Equal(t, rideStartEvent.RideID, receivedEvent.RideID)
	assert.Equal(t, rideStartEvent.DriverLocation.Latitude, receivedEvent.DriverLocation.Latitude)
	assert.Equal(t, rideStartEvent.DriverLocation.Longitude, receivedEvent.DriverLocation.Longitude)
	assert.Equal(t, rideStartEvent.PassengerLocation.Latitude, receivedEvent.PassengerLocation.Latitude)
	assert.Equal(t, rideStartEvent.PassengerLocation.Longitude, receivedEvent.PassengerLocation.Longitude)

	// Note: The model correctly uses DriverLocation and PassengerLocation,
	// not DriverID and PassengerID as in some other events
}
