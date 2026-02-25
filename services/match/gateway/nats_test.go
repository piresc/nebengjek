package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	locationmodels "github.com/piresc/nebengjek/internal/pkg/models/location"
	matchmodels "github.com/piresc/nebengjek/internal/pkg/models/match"
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

// PublishMatchFound publishes a match found event to NATS
func (g *TestableNATSGateway) PublishMatchFound(ctx context.Context, matchProp matchmodels.MatchProposal) error {
	data, err := json.Marshal(matchProp)
	if err != nil {
		return err
	}
	return g.client.Publish(natspkg.SubjectMatchFound, data)
}

// PublishMatchRejected publishes a match rejected event to NATS
func (g *TestableNATSGateway) PublishMatchRejected(ctx context.Context, matchProp matchmodels.MatchProposal) error {
	data, err := json.Marshal(matchProp)
	if err != nil {
		return err
	}
	return g.client.Publish(natspkg.SubjectMatchRejected, data)
}

// PublishMatchAccepted publishes a match accepted event to NATS
func (g *TestableNATSGateway) PublishMatchAccepted(ctx context.Context, matchProp matchmodels.MatchProposal) error {
	data, err := json.Marshal(matchProp)
	if err != nil {
		return err
	}
	return g.client.Publish(natspkg.SubjectMatchAccepted, data)
}

// Ensure natspkg.Client implements our interface
var _ NATSClientInterface = (*natspkg.Client)(nil)

// TestPublishMatchFound_Success tests successful publishing of match found events
func TestPublishMatchFound_Success(t *testing.T) {
	// Arrange
	mockClient := NewMockNATSClient()
	natsGW := NewTestableNATSGateway(mockClient)

	matchProposal := matchmodels.MatchProposal{
		ID:          uuid.New().String(),
		DriverID:    uuid.New().String(),
		PassengerID: uuid.New().String(),
		UserLocation: locationmodels.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		DriverLocation: locationmodels.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		TargetLocation: locationmodels.Location{
			Latitude:  -6.185392,
			Longitude: 106.837153,
		},
		MatchStatus: matchmodels.MatchStatusPending,
	}

	// Act
	ctx := context.Background()
	err := natsGW.PublishMatchFound(ctx, matchProposal)

	// Assert
	require.NoError(t, err)

	// Verify the message was published to the correct subject
	publishedData, exists := mockClient.GetPublishedMessage(natspkg.SubjectMatchFound)
	require.True(t, exists, "Message should be published to match found subject")

	// Verify the published data matches the original event
	var receivedProposal matchmodels.MatchProposal
	err = json.Unmarshal(publishedData, &receivedProposal)
	require.NoError(t, err)

	assert.Equal(t, matchProposal.ID, receivedProposal.ID)
	assert.Equal(t, matchProposal.DriverID, receivedProposal.DriverID)
	assert.Equal(t, matchProposal.PassengerID, receivedProposal.PassengerID)
	assert.Equal(t, matchProposal.DriverLocation.Latitude, receivedProposal.DriverLocation.Latitude)
	assert.Equal(t, matchProposal.DriverLocation.Longitude, receivedProposal.DriverLocation.Longitude)
	assert.Equal(t, matchProposal.UserLocation.Latitude, receivedProposal.UserLocation.Latitude)
	assert.Equal(t, matchProposal.UserLocation.Longitude, receivedProposal.UserLocation.Longitude)
	assert.Equal(t, matchProposal.TargetLocation.Latitude, receivedProposal.TargetLocation.Latitude)
	assert.Equal(t, matchProposal.TargetLocation.Longitude, receivedProposal.TargetLocation.Longitude)
	assert.Equal(t, matchProposal.MatchStatus, receivedProposal.MatchStatus)
}

// TestPublishMatchFound_Error tests error handling during match found publishing
func TestPublishMatchFound_Error(t *testing.T) {
	// Arrange
	mockClient := NewMockNATSClient()
	expectedError := errors.New("NATS publish failed")
	mockClient.SetPublishError(expectedError)

	natsGW := NewTestableNATSGateway(mockClient)

	matchProposal := matchmodels.MatchProposal{
		ID:          uuid.New().String(),
		DriverID:    uuid.New().String(),
		PassengerID: uuid.New().String(),
		UserLocation: locationmodels.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		DriverLocation: locationmodels.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		TargetLocation: locationmodels.Location{
			Latitude:  -6.185392,
			Longitude: 106.837153,
		},
		MatchStatus: matchmodels.MatchStatusPending,
	}

	// Act
	ctx := context.Background()
	err := natsGW.PublishMatchFound(ctx, matchProposal)

	// Assert
	require.Error(t, err)
	assert.Equal(t, expectedError, err)
}

// TestPublishMatchRejected_Success tests successful publishing of match rejected events
func TestPublishMatchRejected_Success(t *testing.T) {
	// Arrange
	mockClient := NewMockNATSClient()
	natsGW := NewTestableNATSGateway(mockClient)

	matchProposal := matchmodels.MatchProposal{
		ID:          uuid.New().String(),
		DriverID:    uuid.New().String(),
		PassengerID: uuid.New().String(),
		UserLocation: locationmodels.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		DriverLocation: locationmodels.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		TargetLocation: locationmodels.Location{
			Latitude:  -6.185392,
			Longitude: 106.837153,
		},
		MatchStatus: matchmodels.MatchStatusRejected,
	}

	// Act
	ctx := context.Background()
	err := natsGW.PublishMatchRejected(ctx, matchProposal)

	// Assert
	require.NoError(t, err)

	// Verify the message was published to the correct subject
	publishedData, exists := mockClient.GetPublishedMessage(natspkg.SubjectMatchRejected)
	require.True(t, exists, "Message should be published to match rejected subject")

	// Verify the published data matches the original event
	var receivedProposal matchmodels.MatchProposal
	err = json.Unmarshal(publishedData, &receivedProposal)
	require.NoError(t, err)

	assert.Equal(t, matchProposal.ID, receivedProposal.ID)
	assert.Equal(t, matchProposal.DriverID, receivedProposal.DriverID)
	assert.Equal(t, matchProposal.PassengerID, receivedProposal.PassengerID)
	assert.Equal(t, matchProposal.DriverLocation.Latitude, receivedProposal.DriverLocation.Latitude)
	assert.Equal(t, matchProposal.DriverLocation.Longitude, receivedProposal.DriverLocation.Longitude)
	assert.Equal(t, matchProposal.UserLocation.Latitude, receivedProposal.UserLocation.Latitude)
	assert.Equal(t, matchProposal.UserLocation.Longitude, receivedProposal.UserLocation.Longitude)
	assert.Equal(t, matchProposal.TargetLocation.Latitude, receivedProposal.TargetLocation.Latitude)
	assert.Equal(t, matchProposal.TargetLocation.Longitude, receivedProposal.TargetLocation.Longitude)
	assert.Equal(t, matchProposal.MatchStatus, receivedProposal.MatchStatus)
}

// TestPublishMatchRejected_Error tests error handling during match rejected publishing
func TestPublishMatchRejected_Error(t *testing.T) {
	// Arrange
	mockClient := NewMockNATSClient()
	expectedError := errors.New("NATS publish failed")
	mockClient.SetPublishError(expectedError)

	natsGW := NewTestableNATSGateway(mockClient)

	matchProposal := matchmodels.MatchProposal{
		ID:          uuid.New().String(),
		DriverID:    uuid.New().String(),
		PassengerID: uuid.New().String(),
		UserLocation: locationmodels.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		DriverLocation: locationmodels.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		TargetLocation: locationmodels.Location{
			Latitude:  -6.185392,
			Longitude: 106.837153,
		},
		MatchStatus: matchmodels.MatchStatusRejected,
	}

	// Act
	ctx := context.Background()
	err := natsGW.PublishMatchRejected(ctx, matchProposal)

	// Assert
	require.Error(t, err)
	assert.Equal(t, expectedError, err)
}

// TestPublishMatchAccepted_Success tests successful publishing of match accepted events
func TestPublishMatchAccepted_Success(t *testing.T) {
	// Arrange
	mockClient := NewMockNATSClient()
	natsGW := NewTestableNATSGateway(mockClient)

	matchProposal := matchmodels.MatchProposal{
		ID:          uuid.New().String(),
		DriverID:    uuid.New().String(),
		PassengerID: uuid.New().String(),
		UserLocation: locationmodels.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		DriverLocation: locationmodels.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		TargetLocation: locationmodels.Location{
			Latitude:  -6.185392,
			Longitude: 106.837153,
		},
		MatchStatus: matchmodels.MatchStatusAccepted,
	}

	// Act
	ctx := context.Background()
	err := natsGW.PublishMatchAccepted(ctx, matchProposal)

	// Assert
	require.NoError(t, err)

	// Verify the message was published to the correct subject
	publishedData, exists := mockClient.GetPublishedMessage(natspkg.SubjectMatchAccepted)
	require.True(t, exists, "Message should be published to match accepted subject")

	// Verify the published data matches the original event
	var receivedProposal matchmodels.MatchProposal
	err = json.Unmarshal(publishedData, &receivedProposal)
	require.NoError(t, err)

	assert.Equal(t, matchProposal.ID, receivedProposal.ID)
	assert.Equal(t, matchProposal.DriverID, receivedProposal.DriverID)
	assert.Equal(t, matchProposal.PassengerID, receivedProposal.PassengerID)
	assert.Equal(t, matchProposal.DriverLocation.Latitude, receivedProposal.DriverLocation.Latitude)
	assert.Equal(t, matchProposal.DriverLocation.Longitude, receivedProposal.DriverLocation.Longitude)
	assert.Equal(t, matchProposal.UserLocation.Latitude, receivedProposal.UserLocation.Latitude)
	assert.Equal(t, matchProposal.UserLocation.Longitude, receivedProposal.UserLocation.Longitude)
	assert.Equal(t, matchProposal.TargetLocation.Latitude, receivedProposal.TargetLocation.Latitude)
	assert.Equal(t, matchProposal.TargetLocation.Longitude, receivedProposal.TargetLocation.Longitude)
	assert.Equal(t, matchProposal.MatchStatus, receivedProposal.MatchStatus)
}

// TestPublishMatchAccepted_Error tests error handling during match accepted publishing
func TestPublishMatchAccepted_Error(t *testing.T) {
	// Arrange
	mockClient := NewMockNATSClient()
	expectedError := errors.New("NATS publish failed")
	mockClient.SetPublishError(expectedError)

	natsGW := NewTestableNATSGateway(mockClient)

	matchProposal := matchmodels.MatchProposal{
		ID:          uuid.New().String(),
		DriverID:    uuid.New().String(),
		PassengerID: uuid.New().String(),
		UserLocation: locationmodels.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		DriverLocation: locationmodels.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		TargetLocation: locationmodels.Location{
			Latitude:  -6.185392,
			Longitude: 106.837153,
		},
		MatchStatus: matchmodels.MatchStatusAccepted,
	}

	// Act
	ctx := context.Background()
	err := natsGW.PublishMatchAccepted(ctx, matchProposal)

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
	matchProposal := matchmodels.MatchProposal{
		ID:          uuid.New().String(),
		DriverID:    uuid.New().String(),
		PassengerID: uuid.New().String(),
		UserLocation: locationmodels.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		DriverLocation: locationmodels.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		TargetLocation: locationmodels.Location{
			Latitude:  -6.185392,
			Longitude: 106.837153,
		},
		MatchStatus: matchmodels.MatchStatusPending,
	}

	// Act
	err1 := natsGW.PublishMatchFound(ctx, matchProposal)

	// Update match status for rejected
	matchProposal.MatchStatus = matchmodels.MatchStatusRejected
	err2 := natsGW.PublishMatchRejected(ctx, matchProposal)

	// Update match status for accepted
	matchProposal.MatchStatus = matchmodels.MatchStatusAccepted
	err3 := natsGW.PublishMatchAccepted(ctx, matchProposal)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)

	// Verify all messages were published to their respective subjects
	_, foundExists := mockClient.GetPublishedMessage(natspkg.SubjectMatchFound)
	_, rejectedExists := mockClient.GetPublishedMessage(natspkg.SubjectMatchRejected)
	_, acceptedExists := mockClient.GetPublishedMessage(natspkg.SubjectMatchAccepted)

	assert.True(t, foundExists, "Match found message should be published")
	assert.True(t, rejectedExists, "Match rejected message should be published")
	assert.True(t, acceptedExists, "Match accepted message should be published")
}
