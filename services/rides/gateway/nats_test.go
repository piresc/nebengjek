package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	ridemodels "github.com/piresc/nebengjek/internal/pkg/models/ride"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	natsconstants "github.com/piresc/nebengjek/internal/pkg/nats"
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

// PublishRidePickup publishes a ride pickup event to NATS
func (g *TestableNATSGateway) PublishRidePickup(ctx context.Context, ride *ridemodels.Ride) error {
	rideResponse := ridemodels.RideResp{
		RideID:      ride.RideID.String(),
		DriverID:    ride.DriverID.String(),
		PassengerID: ride.PassengerID.String(),
		Status:      string(ride.Status),
		TotalCost:   ride.TotalCost,
		CreatedAt:   ride.CreatedAt,
		UpdatedAt:   ride.UpdatedAt,
	}
	data, err := json.Marshal(rideResponse)
	if err != nil {
		return err
	}
	return g.client.Publish(natsconstants.SubjectRidePickup, data)
}

// PublishRideStarted publishes a ride started event to NATS
func (g *TestableNATSGateway) PublishRideStarted(ctx context.Context, ride *ridemodels.Ride) error {
	rideResponse := ridemodels.RideResp{
		RideID:      ride.RideID.String(),
		DriverID:    ride.DriverID.String(),
		PassengerID: ride.PassengerID.String(),
		Status:      string(ride.Status),
		TotalCost:   ride.TotalCost,
		CreatedAt:   ride.CreatedAt,
		UpdatedAt:   ride.UpdatedAt,
	}
	data, err := json.Marshal(rideResponse)
	if err != nil {
		return err
	}
	return g.client.Publish(natsconstants.SubjectRideStarted, data)
}

// PublishRideCompleted publishes a ride completed event to NATS
func (g *TestableNATSGateway) PublishRideCompleted(ctx context.Context, rideComplete ridemodels.RideComplete) error {
	data, err := json.Marshal(rideComplete)
	if err != nil {
		return err
	}
	return g.client.Publish(natsconstants.SubjectRideCompleted, data)
}

// Ensure natspkg.Client implements our interface
var _ NATSClientInterface = (*natspkg.Client)(nil)

// TestPublishRidePickup_Success tests successful publishing of ride pickup events
func TestPublishRidePickup_Success(t *testing.T) {
	// Arrange
	mockClient := NewMockNATSClient()
	natsGW := NewTestableNATSGateway(mockClient)

	rideID := uuid.New()
	driverID := uuid.New()
	passengerID := uuid.New()
	createdAt := time.Now().Add(-10 * time.Minute)
	updatedAt := time.Now()

	ride := &ridemodels.Ride{
		RideID:      rideID,
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      ridemodels.RideStatusDriverPickup,
		TotalCost:   1000,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	// Act
	ctx := context.Background()
	err := natsGW.PublishRidePickup(ctx, ride)

	// Assert
	require.NoError(t, err)

	// Verify the message was published to the correct subject
	publishedData, exists := mockClient.GetPublishedMessage(natsconstants.SubjectRidePickup)
	require.True(t, exists, "Message should be published to ride pickup subject")

	// Verify the published data matches the original event
	var receivedRide ridemodels.RideResp
	err = json.Unmarshal(publishedData, &receivedRide)
	require.NoError(t, err)

	assert.Equal(t, ride.RideID.String(), receivedRide.RideID)
	assert.Equal(t, ride.DriverID.String(), receivedRide.DriverID)
	assert.Equal(t, ride.PassengerID.String(), receivedRide.PassengerID)
	assert.Equal(t, string(ride.Status), receivedRide.Status)
	assert.Equal(t, ride.TotalCost, receivedRide.TotalCost)
	assert.Equal(t, ride.CreatedAt.Unix(), receivedRide.CreatedAt.Unix())
	assert.Equal(t, ride.UpdatedAt.Unix(), receivedRide.UpdatedAt.Unix())
}

// TestPublishRidePickup_Error tests error handling during ride pickup publishing
func TestPublishRidePickup_Error(t *testing.T) {
	// Arrange
	mockClient := NewMockNATSClient()
	expectedError := errors.New("NATS publish failed")
	mockClient.SetPublishError(expectedError)

	natsGW := NewTestableNATSGateway(mockClient)

	rideID := uuid.New()
	driverID := uuid.New()
	passengerID := uuid.New()
	createdAt := time.Now().Add(-10 * time.Minute)
	updatedAt := time.Now()

	ride := &ridemodels.Ride{
		RideID:      rideID,
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      ridemodels.RideStatusDriverPickup,
		TotalCost:   1000,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	// Act
	ctx := context.Background()
	err := natsGW.PublishRidePickup(ctx, ride)

	// Assert
	require.Error(t, err)
	assert.Equal(t, expectedError, err)
}

// TestPublishRideStarted_Success tests successful publishing of ride started events
func TestPublishRideStarted_Success(t *testing.T) {
	// Arrange
	mockClient := NewMockNATSClient()
	natsGW := NewTestableNATSGateway(mockClient)

	rideID := uuid.New()
	driverID := uuid.New()
	passengerID := uuid.New()
	createdAt := time.Now().Add(-10 * time.Minute)
	updatedAt := time.Now()

	ride := &ridemodels.Ride{
		RideID:      rideID,
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      ridemodels.RideStatusOngoing,
		TotalCost:   1000,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	// Act
	ctx := context.Background()
	err := natsGW.PublishRideStarted(ctx, ride)

	// Assert
	require.NoError(t, err)

	// Verify the message was published to the correct subject
	publishedData, exists := mockClient.GetPublishedMessage(natsconstants.SubjectRideStarted)
	require.True(t, exists, "Message should be published to ride started subject")

	// Verify the published data matches the original event
	var receivedRide ridemodels.RideResp
	err = json.Unmarshal(publishedData, &receivedRide)
	require.NoError(t, err)

	assert.Equal(t, ride.RideID.String(), receivedRide.RideID)
	assert.Equal(t, ride.DriverID.String(), receivedRide.DriverID)
	assert.Equal(t, ride.PassengerID.String(), receivedRide.PassengerID)
	assert.Equal(t, string(ride.Status), receivedRide.Status)
	assert.Equal(t, ride.TotalCost, receivedRide.TotalCost)
	assert.Equal(t, ride.CreatedAt.Unix(), receivedRide.CreatedAt.Unix())
	assert.Equal(t, ride.UpdatedAt.Unix(), receivedRide.UpdatedAt.Unix())
}

// TestPublishRideStarted_Error tests error handling during ride started publishing
func TestPublishRideStarted_Error(t *testing.T) {
	// Arrange
	mockClient := NewMockNATSClient()
	expectedError := errors.New("NATS publish failed")
	mockClient.SetPublishError(expectedError)

	natsGW := NewTestableNATSGateway(mockClient)

	rideID := uuid.New()
	driverID := uuid.New()
	passengerID := uuid.New()
	createdAt := time.Now().Add(-10 * time.Minute)
	updatedAt := time.Now()

	ride := &ridemodels.Ride{
		RideID:      rideID,
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      ridemodels.RideStatusOngoing,
		TotalCost:   1000,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	// Act
	ctx := context.Background()
	err := natsGW.PublishRideStarted(ctx, ride)

	// Assert
	require.Error(t, err)
	assert.Equal(t, expectedError, err)
}

// TestPublishRideCompleted_Success tests successful publishing of ride completed events
func TestPublishRideCompleted_Success(t *testing.T) {
	// Arrange
	mockClient := NewMockNATSClient()
	natsGW := NewTestableNATSGateway(mockClient)

	rideID := uuid.New()
	driverID := uuid.New()
	passengerID := uuid.New()
	createdAt := time.Now().Add(-10 * time.Minute)
	updatedAt := time.Now()

	ride := ridemodels.Ride{
		RideID:      rideID,
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      ridemodels.RideStatusCompleted,
		TotalCost:   1000,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	paymentID := uuid.New()
	payment := ridemodels.Payment{
		PaymentID:    paymentID,
		RideID:       rideID,
		AdjustedCost: 1200,
		AdminFee:     200,
		DriverPayout: 1000,
		Status:       ridemodels.PaymentStatusProcessed,
		CreatedAt:    time.Now(),
	}

	rideComplete := ridemodels.RideComplete{
		Ride:    ride,
		Payment: payment,
	}

	// Act
	ctx := context.Background()
	err := natsGW.PublishRideCompleted(ctx, rideComplete)

	// Assert
	require.NoError(t, err)

	// Verify the message was published to the correct subject
	publishedData, exists := mockClient.GetPublishedMessage(natsconstants.SubjectRideCompleted)
	require.True(t, exists, "Message should be published to ride completed subject")

	// Verify the published data matches the original event
	var receivedRideComplete ridemodels.RideComplete
	err = json.Unmarshal(publishedData, &receivedRideComplete)
	require.NoError(t, err)

	assert.Equal(t, rideComplete.Ride.RideID, receivedRideComplete.Ride.RideID)
	assert.Equal(t, rideComplete.Ride.DriverID, receivedRideComplete.Ride.DriverID)
	assert.Equal(t, rideComplete.Ride.PassengerID, receivedRideComplete.Ride.PassengerID)
	assert.Equal(t, rideComplete.Ride.Status, receivedRideComplete.Ride.Status)
	assert.Equal(t, rideComplete.Ride.TotalCost, receivedRideComplete.Ride.TotalCost)
	assert.Equal(t, rideComplete.Ride.CreatedAt.Unix(), receivedRideComplete.Ride.CreatedAt.Unix())
	assert.Equal(t, rideComplete.Ride.UpdatedAt.Unix(), receivedRideComplete.Ride.UpdatedAt.Unix())

	assert.Equal(t, rideComplete.Payment.PaymentID, receivedRideComplete.Payment.PaymentID)
	assert.Equal(t, rideComplete.Payment.RideID, receivedRideComplete.Payment.RideID)
	assert.Equal(t, rideComplete.Payment.AdjustedCost, receivedRideComplete.Payment.AdjustedCost)
	assert.Equal(t, rideComplete.Payment.AdminFee, receivedRideComplete.Payment.AdminFee)
	assert.Equal(t, rideComplete.Payment.DriverPayout, receivedRideComplete.Payment.DriverPayout)
	assert.Equal(t, rideComplete.Payment.Status, receivedRideComplete.Payment.Status)
	assert.Equal(t, rideComplete.Payment.CreatedAt.Unix(), receivedRideComplete.Payment.CreatedAt.Unix())
}

// TestPublishRideCompleted_Error tests error handling during ride completed publishing
func TestPublishRideCompleted_Error(t *testing.T) {
	// Arrange
	mockClient := NewMockNATSClient()
	expectedError := errors.New("NATS publish failed")
	mockClient.SetPublishError(expectedError)

	natsGW := NewTestableNATSGateway(mockClient)

	rideID := uuid.New()
	driverID := uuid.New()
	passengerID := uuid.New()
	createdAt := time.Now().Add(-10 * time.Minute)
	updatedAt := time.Now()

	ride := ridemodels.Ride{
		RideID:      rideID,
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      ridemodels.RideStatusCompleted,
		TotalCost:   1000,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	paymentID := uuid.New()
	payment := ridemodels.Payment{
		PaymentID:    paymentID,
		RideID:       rideID,
		AdjustedCost: 1200,
		AdminFee:     200,
		DriverPayout: 1000,
		Status:       ridemodels.PaymentStatusProcessed,
		CreatedAt:    time.Now(),
	}

	rideComplete := ridemodels.RideComplete{
		Ride:    ride,
		Payment: payment,
	}

	// Act
	ctx := context.Background()
	err := natsGW.PublishRideCompleted(ctx, rideComplete)

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
	rideID := uuid.New()
	driverID := uuid.New()
	passengerID := uuid.New()
	createdAt := time.Now().Add(-10 * time.Minute)
	updatedAt := time.Now()

	// Create ride for pickup
	ridePickup := &ridemodels.Ride{
		RideID:      rideID,
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      ridemodels.RideStatusDriverPickup,
		TotalCost:   1000,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	// Create ride for started
	rideStarted := &ridemodels.Ride{
		RideID:      rideID,
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      ridemodels.RideStatusOngoing,
		TotalCost:   1000,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	// Create ride for completed
	rideCompleted := ridemodels.Ride{
		RideID:      rideID,
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      ridemodels.RideStatusCompleted,
		TotalCost:   1000,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	paymentID := uuid.New()
	payment := ridemodels.Payment{
		PaymentID:    paymentID,
		RideID:       rideID,
		AdjustedCost: 1200,
		AdminFee:     200,
		DriverPayout: 1000,
		Status:       ridemodels.PaymentStatusProcessed,
		CreatedAt:    time.Now(),
	}

	rideComplete := ridemodels.RideComplete{
		Ride:    rideCompleted,
		Payment: payment,
	}

	// Act
	err1 := natsGW.PublishRidePickup(ctx, ridePickup)
	err2 := natsGW.PublishRideStarted(ctx, rideStarted)
	err3 := natsGW.PublishRideCompleted(ctx, rideComplete)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)

	// Verify all messages were published to their respective subjects
	_, pickupExists := mockClient.GetPublishedMessage(natsconstants.SubjectRidePickup)
	_, startedExists := mockClient.GetPublishedMessage(natsconstants.SubjectRideStarted)
	_, completedExists := mockClient.GetPublishedMessage(natsconstants.SubjectRideCompleted)

	assert.True(t, pickupExists, "Ride pickup message should be published")
	assert.True(t, startedExists, "Ride started message should be published")
	assert.True(t, completedExists, "Ride completed message should be published")
}
