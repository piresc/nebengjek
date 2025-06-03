package nats

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/stretchr/testify/assert"
)

// handleMatchAcceptedEvent mirrors the actual handler method for testing
func (h *testNatsHandler) handleMatchAcceptedEvent(data []byte) error {
	var matchProposal models.MatchProposal
	if err := json.Unmarshal(data, &matchProposal); err != nil {
		return fmt.Errorf("failed to unmarshal match accepted event: %w", err)
	}

	// Notify driver
	h.wsManager.NotifyClient(matchProposal.DriverID, constants.EventMatchConfirm, matchProposal)

	// Notify passenger
	h.wsManager.NotifyClient(matchProposal.PassengerID, constants.EventMatchConfirm, matchProposal)

	return nil
}

// handleRidePickupEvent mirrors the actual handler method for testing
func (h *testNatsHandler) handleRidePickupEvent(data []byte) error {
	var ridePickup models.RideResp
	if err := json.Unmarshal(data, &ridePickup); err != nil {
		return fmt.Errorf("failed to unmarshal match event: %w", err)
	}

	// Notify driver
	h.wsManager.NotifyClient(ridePickup.DriverID, constants.SubjectRidePickup, ridePickup)

	// Notify passenger
	h.wsManager.NotifyClient(ridePickup.PassengerID, constants.SubjectRidePickup, ridePickup)

	return nil
}

// handleRideStartEvent mirrors the actual handler method for testing
func (h *testNatsHandler) handleRideStartEvent(data []byte) error {
	var rideStarted models.RideResp
	if err := json.Unmarshal(data, &rideStarted); err != nil {
		return fmt.Errorf("failed to unmarshal ride start event: %w", err)
	}

	// Notify driver
	h.wsManager.NotifyClient(rideStarted.DriverID, constants.EventMatchConfirm, rideStarted)

	// Notify passenger
	h.wsManager.NotifyClient(rideStarted.PassengerID, constants.EventMatchConfirm, rideStarted)

	return nil
}

// handleRideCompletedEvent mirrors the actual handler method for testing
func (h *testNatsHandler) handleRideCompletedEvent(data []byte) error {
	var rideComplete models.RideComplete
	if err := json.Unmarshal(data, &rideComplete); err != nil {
		return fmt.Errorf("failed to unmarshal ride completed event: %w", err)
	}

	// Notify driver
	h.wsManager.NotifyClient(rideComplete.Ride.DriverID.String(), constants.EventRideCompleted, rideComplete)

	// Notify passenger
	h.wsManager.NotifyClient(rideComplete.Ride.PassengerID.String(), constants.EventRideCompleted, rideComplete)

	return nil
}

func TestHandleMatchAcceptedEvent_Success(t *testing.T) {
	// Arrange
	driverID := uuid.New().String()
	passengerID := uuid.New().String()
	matchProposal := models.MatchProposal{
		ID:          uuid.New().String(),
		DriverID:    driverID,
		PassengerID: passengerID,
	}

	msgData, err := json.Marshal(matchProposal)
	assert.NoError(t, err)

	mockWS := &MockWebSocketManager{}
	handler := &testNatsHandler{wsManager: mockWS}

	// Act
	err = handler.handleMatchAcceptedEvent(msgData)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, mockWS.notifications, 2)

	// Check driver notification
	assert.Equal(t, driverID, mockWS.notifications[0].UserID)
	assert.Equal(t, constants.EventMatchConfirm, mockWS.notifications[0].Event)

	// Check passenger notification
	assert.Equal(t, passengerID, mockWS.notifications[1].UserID)
	assert.Equal(t, constants.EventMatchConfirm, mockWS.notifications[1].Event)
}

func TestHandleMatchAcceptedEvent_InvalidJSON(t *testing.T) {
	// Arrange
	invalidJSON := []byte(`{invalid json}`)
	mockWS := &MockWebSocketManager{}
	handler := &testNatsHandler{wsManager: mockWS}

	// Act
	err := handler.handleMatchAcceptedEvent(invalidJSON)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal match accepted event")
	assert.Len(t, mockWS.notifications, 0)
}

func TestHandleRidePickupEvent_Success(t *testing.T) {
	// Arrange
	driverID := uuid.New()
	passengerID := uuid.New()
	ridePickup := models.RideResp{
		RideID:      uuid.New().String(),
		DriverID:    driverID.String(),
		PassengerID: passengerID.String(),
		Status:      "pickup",
		CreatedAt:   time.Now(),
	}

	msgData, err := json.Marshal(ridePickup)
	assert.NoError(t, err)

	mockWS := &MockWebSocketManager{}
	handler := &testNatsHandler{wsManager: mockWS}

	// Act
	err = handler.handleRidePickupEvent(msgData)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, mockWS.notifications, 2)

	// Check driver notification
	assert.Equal(t, driverID.String(), mockWS.notifications[0].UserID)
	assert.Equal(t, constants.SubjectRidePickup, mockWS.notifications[0].Event)

	// Check passenger notification
	assert.Equal(t, passengerID.String(), mockWS.notifications[1].UserID)
	assert.Equal(t, constants.SubjectRidePickup, mockWS.notifications[1].Event)
}

func TestHandleRidePickupEvent_InvalidJSON(t *testing.T) {
	// Arrange
	invalidJSON := []byte(`{"incomplete":`)
	mockWS := &MockWebSocketManager{}
	handler := &testNatsHandler{wsManager: mockWS}

	// Act
	err := handler.handleRidePickupEvent(invalidJSON)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal match event")
	assert.Len(t, mockWS.notifications, 0)
}

func TestHandleRideStartEvent_Success(t *testing.T) {
	// Arrange
	driverID := uuid.New()
	passengerID := uuid.New()
	rideStarted := models.RideResp{
		RideID:      uuid.New().String(),
		DriverID:    driverID.String(),
		PassengerID: passengerID.String(),
		Status:      "started",
		CreatedAt:   time.Now(),
	}

	msgData, err := json.Marshal(rideStarted)
	assert.NoError(t, err)

	mockWS := &MockWebSocketManager{}
	handler := &testNatsHandler{wsManager: mockWS}

	// Act
	err = handler.handleRideStartEvent(msgData)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, mockWS.notifications, 2)

	// Check driver notification
	assert.Equal(t, driverID.String(), mockWS.notifications[0].UserID)
	assert.Equal(t, constants.EventMatchConfirm, mockWS.notifications[0].Event)

	// Check passenger notification
	assert.Equal(t, passengerID.String(), mockWS.notifications[1].UserID)
	assert.Equal(t, constants.EventMatchConfirm, mockWS.notifications[1].Event)
}

func TestHandleRideStartEvent_InvalidJSON(t *testing.T) {
	// Arrange
	invalidJSON := []byte(`{malformed}`)
	mockWS := &MockWebSocketManager{}
	handler := &testNatsHandler{wsManager: mockWS}

	// Act
	err := handler.handleRideStartEvent(invalidJSON)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal ride start event")
	assert.Len(t, mockWS.notifications, 0)
}

func TestHandleRideCompletedEvent_Success(t *testing.T) {
	// Arrange
	driverID := uuid.New()
	passengerID := uuid.New()
	rideComplete := models.RideComplete{
		Ride: models.Ride{
			RideID:      uuid.New(),
			DriverID:    driverID,
			PassengerID: passengerID,
			Status:      "completed",
			CreatedAt:   time.Now(),
		},
		Payment: models.Payment{
			PaymentID:    uuid.New(),
			RideID:       uuid.New(),
			AdjustedCost: 25000,
			AdminFee:     1250,
			DriverPayout: 23750,
			Status:       models.PaymentStatusPending,
			CreatedAt:    time.Now(),
		},
	}

	msgData, err := json.Marshal(rideComplete)
	assert.NoError(t, err)

	mockWS := &MockWebSocketManager{}
	handler := &testNatsHandler{wsManager: mockWS}

	// Act
	err = handler.handleRideCompletedEvent(msgData)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, mockWS.notifications, 2)

	// Check driver notification
	assert.Equal(t, driverID.String(), mockWS.notifications[0].UserID)
	assert.Equal(t, constants.EventRideCompleted, mockWS.notifications[0].Event)

	// Check passenger notification
	assert.Equal(t, passengerID.String(), mockWS.notifications[1].UserID)
	assert.Equal(t, constants.EventRideCompleted, mockWS.notifications[1].Event)
}

func TestHandleRideCompletedEvent_InvalidJSON(t *testing.T) {
	// Arrange
	invalidJSON := []byte(`{"broken": json`)
	mockWS := &MockWebSocketManager{}
	handler := &testNatsHandler{wsManager: mockWS}

	// Act
	err := handler.handleRideCompletedEvent(invalidJSON)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal ride completed event")
	assert.Len(t, mockWS.notifications, 0)
}
