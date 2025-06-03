package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/users/mocks"
	"github.com/stretchr/testify/assert"
)

// Use existing mock types from location_test.go to avoid redeclaration
// MockWebSocketManager, MockMessage, MockError, MockNotification are already defined

// testWebSocketManager wrapper for ride handler methods
func (m *testWebSocketManager) handleRideStart(client *models.WebSocketClient, data json.RawMessage) error {
	var event models.RideStartRequest
	if err := json.Unmarshal(data, &event); err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid ride start trip format")
	}
	resp, err := m.userUC.RideStart(context.Background(), &event)
	if err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Failed to process ride start trip")
	}
	// Directly notify both driver and passenger about the confirmation
	m.manager.NotifyClient(resp.DriverID.String(), constants.EventRideStarted, resp)
	m.manager.NotifyClient(resp.PassengerID.String(), constants.EventRideStarted, resp)

	return nil
}

func (m *testWebSocketManager) handleRideArrived(client *models.WebSocketClient, data json.RawMessage) error {
	var event models.RideArrivalReq
	if err := json.Unmarshal(data, &event); err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid ride arrival format")
	}

	// Call the use case to process the ride arrival via HTTP
	paymentReq, err := m.userUC.RideArrived(context.Background(), &event)
	if err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Failed to process ride arrival")
	}
	m.manager.NotifyClient(paymentReq.PassengerID, constants.EventPaymentRequest, paymentReq)

	return nil
}

func (m *testWebSocketManager) handleProcessPayment(client *models.WebSocketClient, data json.RawMessage) error {
	var paymentReq models.PaymentProccessRequest
	if err := json.Unmarshal(data, &paymentReq); err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid payment request format")
	}
	if paymentReq.Status != models.PaymentStatusAccepted && paymentReq.Status != models.PaymentStatusRejected {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid payment status")
	}

	// Call the use case to process the payment
	payment, err := m.userUC.ProcessPayment(context.Background(), &paymentReq)
	if err != nil {
		return m.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Failed to process payment")
	}

	// Notify the client about the processed payment
	if err := m.manager.SendMessage(client.Conn, constants.EventPaymentProcessed, payment); err != nil {
		return err
	}

	return nil
}

func TestHandleRideStart_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	mockManager := NewMockWebSocketManager()

	wsManager := &testWebSocketManager{
		userUC:  mockUserUC,
		manager: mockManager,
	}

	client := &models.WebSocketClient{
		UserID: uuid.New().String(),
		Conn:   nil,
	}

	driverID := uuid.New()
	passengerID := uuid.New()
	rideStartReq := models.RideStartRequest{
		RideID: uuid.New().String(),
	}

	rideResp := &models.Ride{
		RideID:      uuid.MustParse(rideStartReq.RideID),
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      models.RideStatusOngoing,
	}

	msgData, err := json.Marshal(rideStartReq)
	assert.NoError(t, err)

	mockUserUC.EXPECT().
		RideStart(gomock.Any(), gomock.Any()).
		Return(rideResp, nil)

	// Act
	err = wsManager.handleRideStart(client, json.RawMessage(msgData))

	// Assert
	assert.NoError(t, err)
	assert.Len(t, mockManager.notifications, 2)
	assert.Equal(t, driverID.String(), mockManager.notifications[0].UserID)
	assert.Equal(t, constants.EventRideStarted, mockManager.notifications[0].Event)
	assert.Equal(t, passengerID.String(), mockManager.notifications[1].UserID)
	assert.Equal(t, constants.EventRideStarted, mockManager.notifications[1].Event)
}

func TestHandleRideStart_InvalidJSON(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	mockManager := NewMockWebSocketManager()

	wsManager := &testWebSocketManager{
		userUC:  mockUserUC,
		manager: mockManager,
	}

	client := &models.WebSocketClient{
		UserID: uuid.New().String(),
		Conn:   nil,
	}

	invalidJSON := json.RawMessage(`{"invalid": json}`)

	// Act
	err := wsManager.handleRideStart(client, invalidJSON)

	// Assert
	assert.NoError(t, err) // Function returns nil but sends error message
	assert.Len(t, mockManager.sentErrors, 1)
	assert.Equal(t, constants.ErrorInvalidFormat, mockManager.sentErrors[0].ErrorCode)
	assert.Contains(t, mockManager.sentErrors[0].Message, "Invalid ride start trip format")
}

func TestHandleRideStart_UsecaseError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	mockManager := NewMockWebSocketManager()

	wsManager := &testWebSocketManager{
		userUC:  mockUserUC,
		manager: mockManager,
	}

	client := &models.WebSocketClient{
		UserID: uuid.New().String(),
		Conn:   nil,
	}

	rideStartReq := models.RideStartRequest{
		RideID: uuid.New().String(),
	}

	msgData, err := json.Marshal(rideStartReq)
	assert.NoError(t, err)

	mockUserUC.EXPECT().
		RideStart(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("database error"))

	// Act
	err = wsManager.handleRideStart(client, json.RawMessage(msgData))

	// Assert
	assert.NoError(t, err) // Function returns nil but sends error message
	assert.Len(t, mockManager.sentErrors, 1)
	assert.Equal(t, constants.ErrorInvalidFormat, mockManager.sentErrors[0].ErrorCode)
	assert.Contains(t, mockManager.sentErrors[0].Message, "Failed to process ride start trip")
}

func TestHandleRideArrived_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	mockManager := NewMockWebSocketManager()

	wsManager := &testWebSocketManager{
		userUC:  mockUserUC,
		manager: mockManager,
	}

	client := &models.WebSocketClient{
		UserID: uuid.New().String(),
		Conn:   nil,
	}

	passengerID := uuid.New().String()
	rideArrivalReq := models.RideArrivalReq{
		RideID:           uuid.New().String(),
		AdjustmentFactor: 1.0,
	}

	paymentReq := &models.PaymentRequest{
		RideID:      rideArrivalReq.RideID,
		PassengerID: passengerID,
		TotalCost:   50000,
	}

	msgData, err := json.Marshal(rideArrivalReq)
	assert.NoError(t, err)

	mockUserUC.EXPECT().
		RideArrived(gomock.Any(), gomock.Any()).
		Return(paymentReq, nil)

	// Act
	err = wsManager.handleRideArrived(client, json.RawMessage(msgData))

	// Assert
	assert.NoError(t, err)
	assert.Len(t, mockManager.notifications, 1)
	assert.Equal(t, passengerID, mockManager.notifications[0].UserID)
	assert.Equal(t, constants.EventPaymentRequest, mockManager.notifications[0].Event)
}

func TestHandleRideArrived_InvalidJSON(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	mockManager := NewMockWebSocketManager()

	wsManager := &testWebSocketManager{
		userUC:  mockUserUC,
		manager: mockManager,
	}

	client := &models.WebSocketClient{
		UserID: uuid.New().String(),
		Conn:   nil,
	}

	invalidJSON := json.RawMessage(`{"invalid": json}`)

	// Act
	err := wsManager.handleRideArrived(client, invalidJSON)

	// Assert
	assert.NoError(t, err) // Function returns nil but sends error message
	assert.Len(t, mockManager.sentErrors, 1)
	assert.Equal(t, constants.ErrorInvalidFormat, mockManager.sentErrors[0].ErrorCode)
	assert.Contains(t, mockManager.sentErrors[0].Message, "Invalid ride arrival format")
}

func TestHandleProcessPayment_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	mockManager := NewMockWebSocketManager()

	wsManager := &testWebSocketManager{
		userUC:  mockUserUC,
		manager: mockManager,
	}

	client := &models.WebSocketClient{
		UserID: uuid.New().String(),
		Conn:   nil,
	}

	paymentReq := models.PaymentProccessRequest{
		RideID:    uuid.New().String(),
		TotalCost: 50000,
		Status:    models.PaymentStatusAccepted,
	}

	payment := &models.Payment{
		PaymentID:    uuid.New(),
		RideID:       uuid.MustParse(paymentReq.RideID),
		AdjustedCost: paymentReq.TotalCost,
		AdminFee:     2500,
		DriverPayout: 47500,
		Status:       models.PaymentStatusProcessed,
	}

	msgData, err := json.Marshal(paymentReq)
	assert.NoError(t, err)

	mockUserUC.EXPECT().
		ProcessPayment(gomock.Any(), gomock.Any()).
		Return(payment, nil)

	// Act
	err = wsManager.handleProcessPayment(client, json.RawMessage(msgData))

	// Assert
	assert.NoError(t, err)
	assert.Len(t, mockManager.sentMessages, 1)
	assert.Equal(t, constants.EventPaymentProcessed, mockManager.sentMessages[0].EventType)
}

func TestHandleProcessPayment_InvalidStatus(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	mockManager := NewMockWebSocketManager()

	wsManager := &testWebSocketManager{
		userUC:  mockUserUC,
		manager: mockManager,
	}

	client := &models.WebSocketClient{
		UserID: uuid.New().String(),
		Conn:   nil,
	}

	paymentReq := models.PaymentProccessRequest{
		RideID:    uuid.New().String(),
		TotalCost: 50000,
		Status:    models.PaymentStatusPending, // Invalid status for processing
	}

	msgData, err := json.Marshal(paymentReq)
	assert.NoError(t, err)

	// Act
	err = wsManager.handleProcessPayment(client, json.RawMessage(msgData))

	// Assert
	assert.NoError(t, err) // Function returns nil but sends error message
	assert.Len(t, mockManager.sentErrors, 1)
	assert.Equal(t, constants.ErrorInvalidFormat, mockManager.sentErrors[0].ErrorCode)
	assert.Contains(t, mockManager.sentErrors[0].Message, "Invalid payment status")
}

func TestHandleProcessPayment_UsecaseError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	mockManager := NewMockWebSocketManager()

	wsManager := &testWebSocketManager{
		userUC:  mockUserUC,
		manager: mockManager,
	}

	client := &models.WebSocketClient{
		UserID: uuid.New().String(),
		Conn:   nil,
	}

	paymentReq := models.PaymentProccessRequest{
		RideID:    uuid.New().String(),
		TotalCost: 50000,
		Status:    models.PaymentStatusAccepted,
	}

	msgData, err := json.Marshal(paymentReq)
	assert.NoError(t, err)

	mockUserUC.EXPECT().
		ProcessPayment(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("payment processing failed"))

	// Act
	err = wsManager.handleProcessPayment(client, json.RawMessage(msgData))

	// Assert
	assert.NoError(t, err) // Function returns nil but sends error message
	assert.Len(t, mockManager.sentErrors, 1)
	assert.Equal(t, constants.ErrorInvalidFormat, mockManager.sentErrors[0].ErrorCode)
	assert.Contains(t, mockManager.sentErrors[0].Message, "Failed to process payment")
}
