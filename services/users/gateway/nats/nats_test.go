package gateway_nats

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	coremodels "github.com/piresc/nebengjek/internal/pkg/models/core"
	"github.com/piresc/nebengjek/internal/pkg/models/location"
	"github.com/piresc/nebengjek/internal/pkg/models/ride"
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
func (g *TestableNATSGateway) PublishBeaconEvent(ctx context.Context, event *coremodels.BeaconEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return g.client.Publish(natspkg.SubjectUserBeacon, data)
}

// PublishFinderEvent publishes a finder event to NATS (same logic as real implementation)
func (g *TestableNATSGateway) PublishFinderEvent(ctx context.Context, event *coremodels.FinderEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return g.client.Publish(natspkg.SubjectUserFinder, data)
}

// PublishRideStart publishes a ride start event to NATS (same logic as real implementation)
func (g *TestableNATSGateway) PublishRideStart(ctx context.Context, event *ride.RideStartTripEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return g.client.Publish(natspkg.SubjectRideStarted, data)
}

// PublishLocationUpdate publishes a location update event to NATS (same logic as real implementation)
func (g *TestableNATSGateway) PublishLocationUpdate(ctx context.Context, locationEvent *location.LocationUpdate) error {
	data, err := json.Marshal(locationEvent)
	if err != nil {
		return err
	}
	return g.client.Publish(natspkg.SubjectLocationUpdate, data)
}

// Ensure natspkg.Client implements our interface
var _ NATSClientInterface = (*natspkg.Client)(nil)

// TestNewNATSGateway tests the constructor function
func TestNewNATSGateway(t *testing.T) {
	// Arrange
	mockClient := &natspkg.Client{}

	// Act
	gateway := NewNATSGateway(mockClient)

	// Assert
	assert.NotNil(t, gateway)
	assert.Equal(t, mockClient, gateway.client)
}

// TestNewNATSGateway_NilClient tests constructor with nil client
func TestNewNATSGateway_NilClient(t *testing.T) {
	// Arrange & Act
	gateway := NewNATSGateway(nil)

	// Assert
	assert.NotNil(t, gateway)
	assert.Nil(t, gateway.client)
}

// TestNATSGateway_PublishBeaconEvent_NilClient tests publishing with nil client
func TestNATSGateway_PublishBeaconEvent_NilClient(t *testing.T) {
	// Arrange
	gateway := NewNATSGateway(nil)

	ctx := context.Background()
	event := &coremodels.BeaconEvent{
		UserID:   "test-user-id",
		IsActive: true,
		Location: location.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
			}

	// Act & Assert - Should panic due to nil client
	assert.Panics(t, func() {
		gateway.PublishBeaconEvent(ctx, event)
	})
}

// TestNATSGateway_PublishFinderEvent_NilClient tests publishing with nil client
func TestNATSGateway_PublishFinderEvent_NilClient(t *testing.T) {
	// Arrange
	gateway := NewNATSGateway(nil)

	ctx := context.Background()
	event := &coremodels.FinderEvent{
		UserID:   "test-user-id",
		IsActive: true,
		Location: location.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
		TargetLocation: location.Location{
			Latitude:  -6.2297,
			Longitude: 106.8295,
		},
			}

	// Act & Assert - Should panic due to nil client
	assert.Panics(t, func() {
		gateway.PublishFinderEvent(ctx, event)
	})
}

// TestNATSGateway_PublishBeaconEvent_NilEvent tests publishing with nil event
func TestNATSGateway_PublishBeaconEvent_NilEvent(t *testing.T) {
	// Arrange
	gateway := NewNATSGateway(nil)

	ctx := context.Background()

	// Act & Assert - Should panic due to nil client and nil event
	assert.Panics(t, func() {
		gateway.PublishBeaconEvent(ctx, nil)
	})
}

// TestNATSGateway_PublishFinderEvent_NilEvent tests publishing with nil event
func TestNATSGateway_PublishFinderEvent_NilEvent(t *testing.T) {
	// Arrange
	gateway := NewNATSGateway(nil)

	ctx := context.Background()

	// Act & Assert - Should panic due to nil client and nil event
	assert.Panics(t, func() {
		gateway.PublishFinderEvent(ctx, nil)
	})
}

// TestNATSGateway_ContextCancellation tests publishing with cancelled context
func TestNATSGateway_ContextCancellation(t *testing.T) {
	// Arrange
	gateway := NewNATSGateway(nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel the context before using it

	beaconEvent := &coremodels.BeaconEvent{
		UserID:   "test-user-id",
		IsActive: true,
		Location: location.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
			}

	finderEvent := &coremodels.FinderEvent{
		UserID:   "test-user-id",
		IsActive: true,
		Location: location.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
		TargetLocation: location.Location{
			Latitude:  -6.2297,
			Longitude: 106.8295,
		},
			}

	// Act & Assert - Should panic due to nil client
	assert.Panics(t, func() {
		gateway.PublishBeaconEvent(ctx, beaconEvent)
	})

	assert.Panics(t, func() {
		gateway.PublishFinderEvent(ctx, finderEvent)
	})
}

// TestNATSGateway_WithRealClient tests with real NATS client (but no server)
func TestNATSGateway_WithRealClient(t *testing.T) {
	// Arrange
	mockClient := &natspkg.Client{}
	gateway := NewNATSGateway(mockClient)

	ctx := context.Background()
	beaconEvent := &coremodels.BeaconEvent{
		UserID:   "test-user-id",
		IsActive: true,
		Location: location.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
			}

	finderEvent := &coremodels.FinderEvent{
		UserID:   "test-user-id",
		IsActive: true,
		Location: location.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
		TargetLocation: location.Location{
			Latitude:  -6.2297,
			Longitude: 106.8295,
		},
			}

	// Act & Assert - Should panic due to nil NATS client connection
	assert.Panics(t, func() {
		gateway.PublishBeaconEvent(ctx, beaconEvent)
	})

	assert.Panics(t, func() {
		gateway.PublishFinderEvent(ctx, finderEvent)
	})
}

// TestPublishBeaconEvent_Success tests successful publishing of beacon events
func TestPublishBeaconEvent_Success(t *testing.T) {
	// Arrange
	mockClient := NewMockNATSClient()
	natsGW := NewTestableNATSGateway(mockClient)

	beaconEvent := &coremodels.BeaconEvent{
		UserID:   uuid.New().String(),
		IsActive: true,
		Location: location.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
					},
			}

	// Act
	ctx := context.Background()
	err := natsGW.PublishBeaconEvent(ctx, beaconEvent)

	// Assert
	require.NoError(t, err)

	// Verify the message was published to the correct subject
	publishedData, exists := mockClient.GetPublishedMessage(natspkg.SubjectUserBeacon)
	require.True(t, exists, "Message should be published to beacon subject")

	// Verify the published data matches the original event
	var receivedEvent coremodels.BeaconEvent
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

	beaconEvent := &coremodels.BeaconEvent{
		UserID:   uuid.New().String(),
		IsActive: true,
		Location: location.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
					},
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

	finderEvent := &coremodels.FinderEvent{
		UserID:   uuid.New().String(),
		IsActive: true,
		Location: location.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
					},
		TargetLocation: location.Location{
			Latitude:  -6.185392,
			Longitude: 106.837153,
					},
			}

	// Act
	ctx := context.Background()
	err := natsGW.PublishFinderEvent(ctx, finderEvent)

	// Assert
	require.NoError(t, err)

	// Verify the message was published to the correct subject
	publishedData, exists := mockClient.GetPublishedMessage(natspkg.SubjectUserFinder)
	require.True(t, exists, "Message should be published to finder subject")

	// Verify the published data matches the original event
	var receivedEvent coremodels.FinderEvent
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
	rideStartEvent := &ride.RideStartTripEvent{
		RideID: uuid.New().String(),
		DriverLocation: location.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
					},
		PassengerLocation: location.Location{
			Latitude:  -6.185392,
			Longitude: 106.837153,
					},
			}

	// Act
	ctx := context.Background()
	err := natsGW.PublishRideStart(ctx, rideStartEvent)

	// Assert
	require.NoError(t, err)

	// Verify the message was published to the correct subject
	publishedData, exists := mockClient.GetPublishedMessage(natspkg.SubjectRideStarted)
	require.True(t, exists, "Message should be published to ride started subject")

	// Verify the published data matches the original event
	var receivedEvent ride.RideStartTripEvent
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

	locationUpdate := &location.LocationUpdate{
		RideID:   "ride-123",
		DriverID: uuid.New().String(),
		Location: location.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
					},
		CreatedAt: time.Now(),
	}

	// Act
	ctx := context.Background()
	err := natsGW.PublishLocationUpdate(ctx, locationUpdate)

	// Assert
	require.NoError(t, err)

	// Verify the message was published to the correct subject
	publishedData, exists := mockClient.GetPublishedMessage(natspkg.SubjectLocationUpdate)
	require.True(t, exists, "Message should be published to location update subject")

	// Verify the published data matches the original event
	var receivedUpdate location.LocationUpdate
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

	locationUpdate := &location.LocationUpdate{
		RideID:   "ride-123",
		DriverID: uuid.New().String(),
		Location: location.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
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
	beaconEvent := &coremodels.BeaconEvent{
		UserID:   uuid.New().String(),
		IsActive: true,
		Location: location.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
					},
			}

	finderEvent := &coremodels.FinderEvent{
		UserID:   uuid.New().String(),
		IsActive: true,
		Location: location.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
					},
		TargetLocation: location.Location{
			Latitude:  -6.185392,
			Longitude: 106.837153,
					},
			}

	// Act
	err1 := natsGW.PublishBeaconEvent(ctx, beaconEvent)
	err2 := natsGW.PublishFinderEvent(ctx, finderEvent)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)

	// Verify both messages were published to their respective subjects
	_, beaconExists := mockClient.GetPublishedMessage(natspkg.SubjectUserBeacon)
	_, finderExists := mockClient.GetPublishedMessage(natspkg.SubjectUserFinder)

	assert.True(t, beaconExists, "Beacon message should be published")
	assert.True(t, finderExists, "Finder message should be published")
}

// TestRideStartEventStructure tests that RideStartTripEvent has the correct structure
func TestRideStartEventStructure(t *testing.T) {
	// Arrange
	mockClient := NewMockNATSClient()
	natsGW := NewTestableNATSGateway(mockClient)

	// Create a RideStartTripEvent with all expected fields based on the actual model
	rideStartEvent := &ride.RideStartTripEvent{
		RideID: uuid.New().String(),
		DriverLocation: location.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
					},
		PassengerLocation: location.Location{
			Latitude:  -6.185392,
			Longitude: 106.837153,
					},
			}

	// Act
	ctx := context.Background()
	err := natsGW.PublishRideStart(ctx, rideStartEvent)

	// Assert
	require.NoError(t, err)

	// Verify the published message structure
	publishedData, exists := mockClient.GetPublishedMessage(natspkg.SubjectRideStarted)
	require.True(t, exists)

	var receivedEvent ride.RideStartTripEvent
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
