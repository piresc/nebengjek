package websocket

import (
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/users/mocks"
	"github.com/stretchr/testify/assert"
)

// testManagerHandler is a test wrapper that uses the mock manager from location_test.go
type testManagerHandler struct {
	userUC  *mocks.MockUserUC
	manager *MockWebSocketManager
}

func (m *testManagerHandler) handleMessage(client *models.WebSocketClient, msgData []byte) error {
	var wsMsg models.WSMessage
	if err := json.Unmarshal(msgData, &wsMsg); err != nil {
		m.manager.sentErrors = append(m.manager.sentErrors, MockError{
			ErrorCode: constants.ErrorInvalidFormat,
			Message:   "Invalid message format",
		})
		return nil // Follow the same pattern as actual implementation - return nil after sending error
	}

	switch wsMsg.Event {
	case constants.EventBeaconUpdate:
		var req models.BeaconRequest
		if err := json.Unmarshal(wsMsg.Data, &req); err != nil {
			return err
		}
		if err := m.userUC.UpdateBeaconStatus(nil, &req); err != nil {
			return err
		}
		m.manager.sentMessages = append(m.manager.sentMessages, MockMessage{
			EventType: constants.EventBeaconUpdate,
			Data:      req,
		})

	case constants.EventFinderUpdate:
		var req models.FinderRequest
		if err := json.Unmarshal(wsMsg.Data, &req); err != nil {
			return err
		}
		if err := m.userUC.UpdateFinderStatus(nil, &req); err != nil {
			return err
		}
		m.manager.sentMessages = append(m.manager.sentMessages, MockMessage{
			EventType: constants.EventFinderUpdate,
			Data:      req,
		})

	case constants.EventMatchConfirm:
		var req models.MatchConfirmRequest
		if err := json.Unmarshal(wsMsg.Data, &req); err != nil {
			return err
		}
		req.UserID = client.UserID
		result, err := m.userUC.ConfirmMatch(nil, &req)
		if err != nil {
			return err
		}
		// Notify both driver and passenger
		m.manager.NotifyClient(result.DriverID, constants.EventMatchConfirm, result)
		m.manager.NotifyClient(result.PassengerID, constants.EventMatchConfirm, result)

	case constants.EventLocationUpdate:
		var req models.LocationUpdate
		if err := json.Unmarshal(wsMsg.Data, &req); err != nil {
			return err
		}
		if err := m.userUC.UpdateUserLocation(nil, &req); err != nil {
			return err
		}

	case constants.EventRideStarted:
		var req models.RideStartRequest
		if err := json.Unmarshal(wsMsg.Data, &req); err != nil {
			return err
		}
		result, err := m.userUC.RideStart(nil, &req)
		if err != nil {
			return err
		}
		// Notify both driver and passenger
		m.manager.NotifyClient(result.DriverID.String(), constants.EventRideStarted, result)
		m.manager.NotifyClient(result.PassengerID.String(), constants.EventRideStarted, result)

	case constants.EventRideArrived:
		var req models.RideArrivalReq
		if err := json.Unmarshal(wsMsg.Data, &req); err != nil {
			return err
		}
		result, err := m.userUC.RideArrived(nil, &req)
		if err != nil {
			return err
		}
		m.manager.NotifyClient(result.PassengerID, constants.EventRideArrived, result)

	case constants.EventPaymentProcessed:
		var req models.PaymentProccessRequest
		if err := json.Unmarshal(wsMsg.Data, &req); err != nil {
			return err
		}
		result, err := m.userUC.ProcessPayment(nil, &req)
		if err != nil {
			return err
		}
		m.manager.sentMessages = append(m.manager.sentMessages, MockMessage{
			EventType: constants.EventPaymentProcessed,
			Data:      result,
		})
	default:
		m.manager.sentErrors = append(m.manager.sentErrors, MockError{
			ErrorCode: constants.ErrorInvalidFormat,
			Message:   "Unknown event type",
		})
	}

	return nil
}

func TestHandleMessage_BeaconUpdate(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	mockManager := NewMockWebSocketManager()

	wsManager := &testManagerHandler{
		userUC:  mockUserUC,
		manager: mockManager,
	}

	client := &models.WebSocketClient{
		UserID: uuid.New().String(),
		Conn:   nil,
	}

	wsMsg := models.WSMessage{
		Event: constants.EventBeaconUpdate,
		Data:  json.RawMessage(`{"msisdn":"081234567890","is_active":true,"latitude":-6.175392,"longitude":106.827153}`),
	}

	msgData, err := json.Marshal(wsMsg)
	assert.NoError(t, err)

	mockUserUC.EXPECT().
		UpdateBeaconStatus(gomock.Any(), gomock.Any()).
		Return(nil)

	// Act
	err = wsManager.handleMessage(client, msgData)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, mockManager.sentMessages, 1)
	assert.Equal(t, constants.EventBeaconUpdate, mockManager.sentMessages[0].EventType)
}

func TestHandleMessage_FinderUpdate(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	mockManager := NewMockWebSocketManager()

	wsManager := &testManagerHandler{
		userUC:  mockUserUC,
		manager: mockManager,
	}

	client := &models.WebSocketClient{
		UserID: uuid.New().String(),
		Conn:   nil,
	}

	wsMsg := models.WSMessage{
		Event: constants.EventFinderUpdate,
		Data:  json.RawMessage(`{"msisdn":"081234567890","is_active":true}`),
	}

	msgData, err := json.Marshal(wsMsg)
	assert.NoError(t, err)

	mockUserUC.EXPECT().
		UpdateFinderStatus(gomock.Any(), gomock.Any()).
		Return(nil)

	// Act
	err = wsManager.handleMessage(client, msgData)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, mockManager.sentMessages, 1)
	assert.Equal(t, constants.EventFinderUpdate, mockManager.sentMessages[0].EventType)
}

func TestHandleMessage_MatchConfirm(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	mockManager := NewMockWebSocketManager()

	wsManager := &testManagerHandler{
		userUC:  mockUserUC,
		manager: mockManager,
	}

	client := &models.WebSocketClient{
		UserID: uuid.New().String(),
		Conn:   nil,
	}

	driverID := uuid.New().String()
	passengerID := uuid.New().String()

	matchResult := &models.MatchProposal{
		ID:          uuid.New().String(),
		DriverID:    driverID,
		PassengerID: passengerID,
		MatchStatus: models.MatchStatusAccepted,
	}

	wsMsg := models.WSMessage{
		Event: constants.EventMatchConfirm,
		Data:  json.RawMessage(`{"id":"` + matchResult.ID + `","status":"ACCEPTED"}`),
	}

	msgData, err := json.Marshal(wsMsg)
	assert.NoError(t, err)

	mockUserUC.EXPECT().
		ConfirmMatch(gomock.Any(), gomock.Any()).
		Return(matchResult, nil)

	// Act
	err = wsManager.handleMessage(client, msgData)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, mockManager.notifications, 2)
	assert.Equal(t, driverID, mockManager.notifications[0].UserID)
	assert.Equal(t, passengerID, mockManager.notifications[1].UserID)
}

func TestHandleMessage_LocationUpdate(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	mockManager := NewMockWebSocketManager()

	wsManager := &testManagerHandler{
		userUC:  mockUserUC,
		manager: mockManager,
	}

	client := &models.WebSocketClient{
		UserID: uuid.New().String(),
		Conn:   nil,
	}

	wsMsg := models.WSMessage{
		Event: constants.EventLocationUpdate,
		Data:  json.RawMessage(`{"latitude":-6.2088,"longitude":106.8456}`),
	}

	msgData, err := json.Marshal(wsMsg)
	assert.NoError(t, err)

	mockUserUC.EXPECT().
		UpdateUserLocation(gomock.Any(), gomock.Any()).
		Return(nil)

	// Act
	err = wsManager.handleMessage(client, msgData)

	// Assert
	assert.NoError(t, err)
}

func TestHandleMessage_RideStarted(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	mockManager := NewMockWebSocketManager()

	wsManager := &testManagerHandler{
		userUC:  mockUserUC,
		manager: mockManager,
	}

	client := &models.WebSocketClient{
		UserID: uuid.New().String(),
		Conn:   nil,
	}

	driverID := uuid.New().String()
	passengerID := uuid.New().String()

	rideResp := &models.Ride{
		RideID:      uuid.New(),
		DriverID:    uuid.MustParse(driverID),
		PassengerID: uuid.MustParse(passengerID),
		Status:      models.RideStatusOngoing,
	}

	wsMsg := models.WSMessage{
		Event: constants.EventRideStarted,
		Data:  json.RawMessage(`{"ride_id":"` + rideResp.RideID.String() + `"}`),
	}

	msgData, err := json.Marshal(wsMsg)
	assert.NoError(t, err)

	mockUserUC.EXPECT().
		RideStart(gomock.Any(), gomock.Any()).
		Return(rideResp, nil)

	// Act
	err = wsManager.handleMessage(client, msgData)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, mockManager.notifications, 2)
}

func TestHandleMessage_RideArrived(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	mockManager := NewMockWebSocketManager()

	wsManager := &testManagerHandler{
		userUC:  mockUserUC,
		manager: mockManager,
	}

	client := &models.WebSocketClient{
		UserID: uuid.New().String(),
		Conn:   nil,
	}

	paymentReq := &models.PaymentRequest{
		PassengerID: uuid.New().String(),
		TotalCost:   25000,
	}

	wsMsg := models.WSMessage{
		Event: constants.EventRideArrived,
		Data:  json.RawMessage(`{"ride_id":"` + uuid.New().String() + `"}`),
	}

	msgData, err := json.Marshal(wsMsg)
	assert.NoError(t, err)

	mockUserUC.EXPECT().
		RideArrived(gomock.Any(), gomock.Any()).
		Return(paymentReq, nil)

	// Act
	err = wsManager.handleMessage(client, msgData)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, mockManager.notifications, 1)
	assert.Equal(t, paymentReq.PassengerID, mockManager.notifications[0].UserID)
}

func TestHandleMessage_ProcessPayment(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	mockManager := NewMockWebSocketManager()

	wsManager := &testManagerHandler{
		userUC:  mockUserUC,
		manager: mockManager,
	}

	client := &models.WebSocketClient{
		UserID: uuid.New().String(),
		Conn:   nil,
	}

	payment := &models.Payment{
		PaymentID:    uuid.New(),
		Status:       models.PaymentStatusProcessed,
		AdjustedCost: 25000,
		AdminFee:     2500,
		DriverPayout: 22500,
	}

	wsMsg := models.WSMessage{
		Event: constants.EventPaymentProcessed,
		Data:  json.RawMessage(`{"status":"accepted","amount":25000}`),
	}

	msgData, err := json.Marshal(wsMsg)
	assert.NoError(t, err)

	mockUserUC.EXPECT().
		ProcessPayment(gomock.Any(), gomock.Any()).
		Return(payment, nil)

	// Act
	err = wsManager.handleMessage(client, msgData)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, mockManager.sentMessages, 1)
	assert.Equal(t, constants.EventPaymentProcessed, mockManager.sentMessages[0].EventType)
}

func TestHandleMessage_InvalidJSON(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	mockManager := NewMockWebSocketManager()

	wsManager := &testManagerHandler{
		userUC:  mockUserUC,
		manager: mockManager,
	}

	client := &models.WebSocketClient{
		UserID: uuid.New().String(),
		Conn:   nil,
	}

	invalidJSON := []byte(`{invalid json}`)

	// Act
	err := wsManager.handleMessage(client, invalidJSON)

	// Assert
	assert.NoError(t, err) // Function returns nil but sends error message
	assert.Len(t, mockManager.sentErrors, 1)
	assert.Equal(t, constants.ErrorInvalidFormat, mockManager.sentErrors[0].ErrorCode)
}

func TestHandleMessage_UnknownEvent(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	mockManager := NewMockWebSocketManager()

	wsManager := &testManagerHandler{
		userUC:  mockUserUC,
		manager: mockManager,
	}

	client := &models.WebSocketClient{
		UserID: uuid.New().String(),
		Conn:   nil,
	}

	wsMsg := models.WSMessage{
		Event: "unknown_event",
		Data:  json.RawMessage(`{}`),
	}

	msgData, err := json.Marshal(wsMsg)
	assert.NoError(t, err)

	// Act
	err = wsManager.handleMessage(client, msgData)

	// Assert
	assert.NoError(t, err) // Function returns nil but sends error message
	assert.Len(t, mockManager.sentErrors, 1)
	assert.Equal(t, constants.ErrorInvalidFormat, mockManager.sentErrors[0].ErrorCode)
	assert.Contains(t, mockManager.sentErrors[0].Message, "Unknown event type")
}
