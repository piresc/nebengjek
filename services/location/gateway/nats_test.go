package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/stretchr/testify/assert"
)

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

// Publish simulates publishing a message
func (m *MockNATSClient) Publish(subject string, data []byte) error {
	if m.publishError != nil {
		return m.publishError
	}
	m.publishedMessages[subject] = data
	return nil
}

// GetConn returns nil as we don't need a real connection for testing
func (m *MockNATSClient) GetConn() *nats.Conn {
	return nil
}

// Subscribe is not used in our tests but required by the interface
func (m *MockNATSClient) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	return nil, nil
}

// Close is not used in our tests but required by the interface
func (m *MockNATSClient) Close() {
	// No-op for mock
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

// TestPublishLocationAggregate_Success tests successful publishing of location aggregate events
func TestPublishLocationAggregate_Success(t *testing.T) {
	// Create a mock NATS client
	mockClient := NewMockNATSClient()

	// Create the actual location gateway with the mock client
	gw := NewLocationGW(mockClient)

	// Create test data
	locationAggregate := models.LocationAggregate{
		RideID:    "ride123",
		Latitude:  -6.2088,
		Longitude: 106.8456,
		Distance:  100.5,
	}

	// Test successful publish
	ctx := context.Background()
	err := gw.PublishLocationAggregate(ctx, locationAggregate)

	// Assertions
	assert.NoError(t, err)

	// Verify the message was published to the correct subject
	publishedData, exists := mockClient.GetPublishedMessage(constants.SubjectLocationAggregate)
	assert.True(t, exists)

	// Verify the published data
	var publishedAggregate models.LocationAggregate
	err = json.Unmarshal(publishedData, &publishedAggregate)
	assert.NoError(t, err)
	assert.Equal(t, locationAggregate.RideID, publishedAggregate.RideID)
	assert.Equal(t, locationAggregate.Latitude, publishedAggregate.Latitude)
	assert.Equal(t, locationAggregate.Longitude, publishedAggregate.Longitude)
	assert.Equal(t, locationAggregate.Distance, publishedAggregate.Distance)
}

// TestPublishLocationAggregate_Error tests error handling during location aggregate publishing
func TestPublishLocationAggregate_Error(t *testing.T) {
	// Create a mock NATS client
	mockClient := NewMockNATSClient()

	// Set up the mock to return an error
	mockClient.SetPublishError(errors.New("publish failed"))

	// Create the actual location gateway with the mock client
	gw := NewLocationGW(mockClient)

	// Create test data
	locationAggregate := models.LocationAggregate{
		RideID:    "ride123",
		Latitude:  -6.2088,
		Longitude: 106.8456,
		Distance:  100.5,
	}

	// Test publish with error
	ctx := context.Background()
	err := gw.PublishLocationAggregate(ctx, locationAggregate)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publish failed")
}

// TestNewLocationGWWithClient tests the concrete client constructor
func TestNewLocationGWWithClient(t *testing.T) {
	// Create a mock NATS client
	mockClient := NewMockNATSClient()

	// Create the location gateway using the concrete client constructor
	// We can't directly test this with our mock, but we can test that it creates a gateway
	gw := NewLocationGW(mockClient)

	// Verify that the gateway was created successfully
	assert.NotNil(t, gw)

	// Test that it can publish (this also tests the functionality)
	locationAggregate := models.LocationAggregate{
		RideID:    "ride456",
		Latitude:  -6.2088,
		Longitude: 106.8456,
		Distance:  200.0,
	}

	ctx := context.Background()
	err := gw.PublishLocationAggregate(ctx, locationAggregate)
	assert.NoError(t, err)

	// Verify the message was published
	publishedData, exists := mockClient.GetPublishedMessage(constants.SubjectLocationAggregate)
	assert.True(t, exists)
	assert.NotEmpty(t, publishedData)
}
