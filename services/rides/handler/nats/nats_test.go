package handler

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/rides"
	"github.com/piresc/nebengjek/services/rides/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NATSClientInterface defines the interface for NATS client operations
type NATSClientInterface interface {
	Subscribe(subject string, handler func([]byte) error) error
	Close()
}

// MockNATSClient simulates NATS client behavior for testing
type MockNATSClient struct {
	subscriptions  map[string]func([]byte) error
	subscribeError error
}

// NewMockNATSClient creates a new mock NATS client
func NewMockNATSClient() *MockNATSClient {
	return &MockNATSClient{
		subscriptions: make(map[string]func([]byte) error),
	}
}

// Subscribe simulates subscribing to a subject
func (m *MockNATSClient) Subscribe(subject string, handler func([]byte) error) error {
	if m.subscribeError != nil {
		return m.subscribeError
	}
	m.subscriptions[subject] = handler
	return nil
}

// SimulateMessage simulates receiving a message on a subject
func (m *MockNATSClient) SimulateMessage(subject string, data []byte) error {
	handler, exists := m.subscriptions[subject]
	if !exists {
		return errors.New("no subscription found for subject")
	}
	return handler(data)
}

// SetSubscribeError sets an error to return on subscribe
func (m *MockNATSClient) SetSubscribeError(err error) {
	m.subscribeError = err
}

// Close simulates closing the connection
func (m *MockNATSClient) Close() {
	// No-op for mock
}

// TestableRidesHandler extends RidesHandler to allow testing with mocks
type TestableRidesHandler struct {
	ridesUC rides.RideUC
	client  NATSClientInterface
	cfg     *models.Config
}

// NewTestableRidesHandler creates a handler that can work with mocks
func NewTestableRidesHandler(ridesUC rides.RideUC, client NATSClientInterface, cfg *models.Config) *TestableRidesHandler {
	return &TestableRidesHandler{
		ridesUC: ridesUC,
		client:  client,
		cfg:     cfg,
	}
}

// InitNATSConsumers initializes all NATS consumers for testing
func (h *TestableRidesHandler) InitNATSConsumers() error {
	// Initialize match accepted consumer
	err := h.client.Subscribe(constants.SubjectMatchAccepted, func(data []byte) error {
		return h.handleMatchAccepted(data)
	})
	if err != nil {
		return err
	}

	// Initialize location aggregate consumer
	err = h.client.Subscribe(constants.SubjectLocationAggregate, func(data []byte) error {
		return h.handleLocationAggregate(data)
	})
	if err != nil {
		return err
	}

	return nil
}

// handleMatchAccepted processes match acceptance events to create rides
func (h *TestableRidesHandler) handleMatchAccepted(msg []byte) error {
	var matchProposal models.MatchProposal
	if err := json.Unmarshal(msg, &matchProposal); err != nil {
		return err
	}

	// Create a ride from the match proposal
	if err := h.ridesUC.CreateRide(matchProposal); err != nil {
		return err
	}

	return nil
}

// handleLocationAggregate processes location aggregates for billing
func (h *TestableRidesHandler) handleLocationAggregate(msg []byte) error {
	var update models.LocationAggregate
	if err := json.Unmarshal(msg, &update); err != nil {
		return err
	}

	// Only process if distance is >= minimum configured distance
	if update.Distance >= h.cfg.Rides.MinDistanceKm {
		// Convert ride ID to UUID
		rideUUID, err := uuid.Parse(update.RideID)
		if err != nil {
			return err
		}

		// Calculate cost at 3000 IDR per km
		cost := int(update.Distance * 3000)

		// Create billing entry
		entry := &models.BillingLedger{
			RideID:   rideUUID,
			Distance: update.Distance,
			Cost:     cost,
		}

		// Store billing entry and update total cost
		if err := h.ridesUC.ProcessBillingUpdate(update.RideID, entry); err != nil {
			return err
		}
	}

	return nil
}

// TestInitNATSConsumers_Success tests successful initialization of NATS consumers
func TestInitNATSConsumers_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockNATSClient()
	mockRidesUC := mocks.NewMockRideUC(ctrl)
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 1.0,
		},
	}

	handler := NewTestableRidesHandler(mockRidesUC, mockClient, cfg)

	// Act
	err := handler.InitNATSConsumers()

	// Assert
	require.NoError(t, err)
	assert.Len(t, mockClient.subscriptions, 2)
	assert.Contains(t, mockClient.subscriptions, constants.SubjectMatchAccepted)
	assert.Contains(t, mockClient.subscriptions, constants.SubjectLocationAggregate)
}

// TestInitNATSConsumers_SubscribeError tests error handling during consumer initialization
func TestInitNATSConsumers_SubscribeError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockNATSClient()
	expectedError := errors.New("subscription failed")
	mockClient.SetSubscribeError(expectedError)

	mockRidesUC := mocks.NewMockRideUC(ctrl)
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 1.0,
		},
	}

	handler := NewTestableRidesHandler(mockRidesUC, mockClient, cfg)

	// Act
	err := handler.InitNATSConsumers()

	// Assert
	require.Error(t, err)
	assert.Equal(t, expectedError, err)
}

// TestHandleMatchAccepted_Success tests successful processing of match accepted events
func TestHandleMatchAccepted_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockNATSClient()
	mockRidesUC := mocks.NewMockRideUC(ctrl)
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 1.0,
		},
	}

	handler := NewTestableRidesHandler(mockRidesUC, mockClient, cfg)
	err := handler.InitNATSConsumers()
	require.NoError(t, err)

	matchProposal := models.MatchProposal{
		ID:          uuid.New().String(),
		DriverID:    uuid.New().String(),
		PassengerID: uuid.New().String(),
		MatchStatus: models.MatchStatusAccepted,
	}

	mockRidesUC.EXPECT().CreateRide(matchProposal).Return(nil)

	// Act
	matchData, err := json.Marshal(matchProposal)
	require.NoError(t, err)

	err = mockClient.SimulateMessage(constants.SubjectMatchAccepted, matchData)

	// Assert
	require.NoError(t, err)
}

// TestHandleMatchAccepted_InvalidJSON tests error handling for invalid JSON
func TestHandleMatchAccepted_InvalidJSON(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockNATSClient()
	mockRidesUC := mocks.NewMockRideUC(ctrl)
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 1.0,
		},
	}

	handler := NewTestableRidesHandler(mockRidesUC, mockClient, cfg)
	err := handler.InitNATSConsumers()
	require.NoError(t, err)

	// Act
	invalidJSON := []byte("{invalid json}")
	err = mockClient.SimulateMessage(constants.SubjectMatchAccepted, invalidJSON)

	// Assert
	require.Error(t, err)
}

// TestHandleMatchAccepted_CreateRideError tests error handling when CreateRide fails
func TestHandleMatchAccepted_CreateRideError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockNATSClient()
	mockRidesUC := mocks.NewMockRideUC(ctrl)
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 1.0,
		},
	}

	handler := NewTestableRidesHandler(mockRidesUC, mockClient, cfg)
	err := handler.InitNATSConsumers()
	require.NoError(t, err)

	matchProposal := models.MatchProposal{
		ID:          uuid.New().String(),
		DriverID:    uuid.New().String(),
		PassengerID: uuid.New().String(),
		MatchStatus: models.MatchStatusAccepted,
	}

	expectedError := errors.New("create ride failed")
	mockRidesUC.EXPECT().CreateRide(matchProposal).Return(expectedError)

	// Act
	matchData, err := json.Marshal(matchProposal)
	require.NoError(t, err)

	err = mockClient.SimulateMessage(constants.SubjectMatchAccepted, matchData)

	// Assert
	require.Error(t, err)
	assert.Equal(t, expectedError, err)
}

// TestHandleLocationAggregate_Success tests successful processing of location aggregates
func TestHandleLocationAggregate_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockNATSClient()
	mockRidesUC := mocks.NewMockRideUC(ctrl)
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 1.0,
		},
	}

	handler := NewTestableRidesHandler(mockRidesUC, mockClient, cfg)
	err := handler.InitNATSConsumers()
	require.NoError(t, err)

	rideID := uuid.New()
	locationAggregate := models.LocationAggregate{
		RideID:   rideID.String(),
		Distance: 2.5, // Above minimum distance
	}

	expectedCost := int(2.5 * 3000) // 7500
	expectedEntry := &models.BillingLedger{
		RideID:   rideID,
		Distance: 2.5,
		Cost:     expectedCost,
	}

	mockRidesUC.EXPECT().ProcessBillingUpdate(rideID.String(), expectedEntry).Return(nil)

	// Act
	locationData, err := json.Marshal(locationAggregate)
	require.NoError(t, err)

	err = mockClient.SimulateMessage(constants.SubjectLocationAggregate, locationData)

	// Assert
	require.NoError(t, err)
}

// TestHandleLocationAggregate_BelowMinDistance tests skipping processing when distance is below minimum
func TestHandleLocationAggregate_BelowMinDistance(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockNATSClient()
	mockRidesUC := mocks.NewMockRideUC(ctrl)
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 2.0,
		},
	}

	handler := NewTestableRidesHandler(mockRidesUC, mockClient, cfg)
	err := handler.InitNATSConsumers()
	require.NoError(t, err)

	rideID := uuid.New()
	locationAggregate := models.LocationAggregate{
		RideID:   rideID.String(),
		Distance: 1.5, // Below minimum distance
	}

	// No expectation on ProcessBillingUpdate since it should be skipped

	// Act
	locationData, err := json.Marshal(locationAggregate)
	require.NoError(t, err)

	err = mockClient.SimulateMessage(constants.SubjectLocationAggregate, locationData)

	// Assert
	require.NoError(t, err)
}

// TestHandleLocationAggregate_InvalidJSON tests error handling for invalid JSON
func TestHandleLocationAggregate_InvalidJSON(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockNATSClient()
	mockRidesUC := mocks.NewMockRideUC(ctrl)
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 1.0,
		},
	}

	handler := NewTestableRidesHandler(mockRidesUC, mockClient, cfg)
	err := handler.InitNATSConsumers()
	require.NoError(t, err)

	// Act
	invalidJSON := []byte("{invalid json}")
	err = mockClient.SimulateMessage(constants.SubjectLocationAggregate, invalidJSON)

	// Assert
	require.Error(t, err)
}

// TestHandleLocationAggregate_InvalidRideID tests error handling for invalid ride ID
func TestHandleLocationAggregate_InvalidRideID(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockNATSClient()
	mockRidesUC := mocks.NewMockRideUC(ctrl)
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 1.0,
		},
	}

	handler := NewTestableRidesHandler(mockRidesUC, mockClient, cfg)
	err := handler.InitNATSConsumers()
	require.NoError(t, err)

	locationAggregate := models.LocationAggregate{
		RideID:   "invalid-uuid",
		Distance: 2.5,
	}

	// Act
	locationData, err := json.Marshal(locationAggregate)
	require.NoError(t, err)

	err = mockClient.SimulateMessage(constants.SubjectLocationAggregate, locationData)

	// Assert
	require.Error(t, err)
}

// TestHandleLocationAggregate_ProcessBillingError tests error handling when ProcessBillingUpdate fails
func TestHandleLocationAggregate_ProcessBillingError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockNATSClient()
	mockRidesUC := mocks.NewMockRideUC(ctrl)
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 1.0,
		},
	}

	handler := NewTestableRidesHandler(mockRidesUC, mockClient, cfg)
	err := handler.InitNATSConsumers()
	require.NoError(t, err)

	rideID := uuid.New()
	locationAggregate := models.LocationAggregate{
		RideID:   rideID.String(),
		Distance: 2.5,
	}

	expectedCost := int(2.5 * 3000)
	expectedEntry := &models.BillingLedger{
		RideID:   rideID,
		Distance: 2.5,
		Cost:     expectedCost,
	}

	expectedError := errors.New("billing update failed")
	mockRidesUC.EXPECT().ProcessBillingUpdate(rideID.String(), expectedEntry).Return(expectedError)

	// Act
	locationData, err := json.Marshal(locationAggregate)
	require.NoError(t, err)

	err = mockClient.SimulateMessage(constants.SubjectLocationAggregate, locationData)

	// Assert
	require.Error(t, err)
	assert.Equal(t, expectedError, err)
}
