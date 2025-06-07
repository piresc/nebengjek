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

// Mock structures and methods are defined in location_test.go to avoid redeclaration

func (h *testWebSocketManager) handleFinderUpdate(client *models.WebSocketClient, data json.RawMessage) error {
	var finderReq models.FinderRequest
	if err := json.Unmarshal(data, &finderReq); err != nil {
		return h.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid finder request format")
	}

	if err := h.userUC.UpdateFinderStatus(context.Background(), &finderReq); err != nil {
		return h.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidBeacon, "Unable to update finder status")
	}

	response := models.FinderResponse{
		Message: "Finder status updated successfully",
	}
	return h.manager.SendMessage(client.Conn, constants.EventFinderUpdate, response)
}

func (h *testWebSocketManager) handleBeaconUpdate(client *models.WebSocketClient, data json.RawMessage) error {
	var beaconReq models.BeaconRequest
	if err := json.Unmarshal(data, &beaconReq); err != nil {
		return h.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid beacon request format")
	}

	if err := h.userUC.UpdateBeaconStatus(context.Background(), &beaconReq); err != nil {
		return h.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidBeacon, "Unable to update beacon status")
	}

	response := models.BeaconResponse{
		Message: "Beacon status updated successfully",
	}
	return h.manager.SendMessage(client.Conn, constants.EventBeaconUpdate, response)
}

func (h *testWebSocketManager) handleMatchConfirmation(client *models.WebSocketClient, data json.RawMessage) error {
	var confirm models.MatchConfirmRequest
	if err := json.Unmarshal(data, &confirm); err != nil {
		return h.manager.SendErrorMessage(client.Conn, constants.ErrorInvalidFormat, "Invalid match proposal format")
	}
	confirm.UserID = client.UserID

	result, err := h.userUC.ConfirmMatch(context.Background(), &confirm)
	if err != nil {
		return h.manager.SendErrorMessage(client.Conn, constants.ErrorMatchUpdateFailed, "Unable to confirm match")
	}

	h.manager.NotifyClient(result.DriverID, constants.EventMatchConfirm, result)
	h.manager.NotifyClient(result.PassengerID, constants.EventMatchConfirm, result)

	return nil
}

func TestHandleFinderUpdate_Success(t *testing.T) {
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

	finderReq := models.FinderRequest{
		MSISDN:   "081234567890",
		IsActive: true,
	}

	msgData, err := json.Marshal(finderReq)
	assert.NoError(t, err)

	mockUserUC.EXPECT().
		UpdateFinderStatus(gomock.Any(), gomock.Any()).
		Return(nil)

	// Act
	err = wsManager.handleFinderUpdate(client, json.RawMessage(msgData))

	// Assert
	assert.NoError(t, err)
	assert.Len(t, mockManager.sentMessages, 1)
	assert.Equal(t, constants.EventFinderUpdate, mockManager.sentMessages[0].EventType)

	response, ok := mockManager.sentMessages[0].Data.(models.FinderResponse)
	assert.True(t, ok)
	assert.Equal(t, "Finder status updated successfully", response.Message)
}

func TestHandleFinderUpdate_InvalidJSON(t *testing.T) {
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
	err := wsManager.handleFinderUpdate(client, invalidJSON)

	// Assert
	assert.NoError(t, err) // Function returns nil but sends error message
	assert.Len(t, mockManager.sentErrors, 1)
	assert.Equal(t, constants.ErrorInvalidFormat, mockManager.sentErrors[0].ErrorCode)
	assert.Contains(t, mockManager.sentErrors[0].Message, "Invalid finder request format")
}

func TestHandleFinderUpdate_UsecaseError(t *testing.T) {
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

	finderReq := models.FinderRequest{
		MSISDN:   "081234567890",
		IsActive: true,
	}

	msgData, err := json.Marshal(finderReq)
	assert.NoError(t, err)

	mockUserUC.EXPECT().
		UpdateFinderStatus(gomock.Any(), gomock.Any()).
		Return(errors.New("database error"))

	// Act
	err = wsManager.handleFinderUpdate(client, json.RawMessage(msgData))

	// Assert
	assert.NoError(t, err) // Function returns nil but sends error message
	assert.Len(t, mockManager.sentErrors, 1)
	assert.Equal(t, constants.ErrorInvalidBeacon, mockManager.sentErrors[0].ErrorCode)
}

func TestHandleBeaconUpdate_Success(t *testing.T) {
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

	beaconReq := models.BeaconRequest{
		MSISDN:    "081234567890",
		IsActive:  true,
		Latitude:  -6.2088,
		Longitude: 106.8456,
	}

	msgData, err := json.Marshal(beaconReq)
	assert.NoError(t, err)

	mockUserUC.EXPECT().
		UpdateBeaconStatus(gomock.Any(), gomock.Any()).
		Return(nil)

	// Act
	err = wsManager.handleBeaconUpdate(client, json.RawMessage(msgData))

	// Assert
	assert.NoError(t, err)
	assert.Len(t, mockManager.sentMessages, 1)
	assert.Equal(t, constants.EventBeaconUpdate, mockManager.sentMessages[0].EventType)

	response, ok := mockManager.sentMessages[0].Data.(models.BeaconResponse)
	assert.True(t, ok)
	assert.Equal(t, "Beacon status updated successfully", response.Message)
}

func TestHandleBeaconUpdate_InvalidJSON(t *testing.T) {
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

	invalidJSON := json.RawMessage(`{broken}`)

	// Act
	err := wsManager.handleBeaconUpdate(client, invalidJSON)

	// Assert
	assert.NoError(t, err) // Function returns nil but sends error message
	assert.Len(t, mockManager.sentErrors, 1)
	assert.Equal(t, constants.ErrorInvalidFormat, mockManager.sentErrors[0].ErrorCode)
	assert.Contains(t, mockManager.sentErrors[0].Message, "Invalid beacon request format")
}

func TestHandleBeaconUpdate_UsecaseError(t *testing.T) {
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

	beaconReq := models.BeaconRequest{
		MSISDN:    "081234567890",
		IsActive:  true,
		Latitude:  -6.2088,
		Longitude: 106.8456,
	}

	msgData, err := json.Marshal(beaconReq)
	assert.NoError(t, err)

	mockUserUC.EXPECT().
		UpdateBeaconStatus(gomock.Any(), gomock.Any()).
		Return(errors.New("beacon update failed"))

	// Act
	err = wsManager.handleBeaconUpdate(client, json.RawMessage(msgData))

	// Assert
	assert.NoError(t, err) // Function returns nil but sends error message
	assert.Len(t, mockManager.sentErrors, 1)
	assert.Equal(t, constants.ErrorInvalidBeacon, mockManager.sentErrors[0].ErrorCode)
}

func TestHandleMatchConfirmation_Success(t *testing.T) {
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

	driverID := uuid.New().String()
	passengerID := uuid.New().String()

	matchConfirm := models.MatchConfirmRequest{
		ID:     uuid.New().String(),
		Status: "ACCEPTED",
		UserID: client.UserID,
	}

	matchResult := &models.MatchProposal{
		ID:          matchConfirm.ID,
		DriverID:    driverID,
		PassengerID: passengerID,
		MatchStatus: models.MatchStatusAccepted,
	}

	msgData, err := json.Marshal(matchConfirm)
	assert.NoError(t, err)

	mockUserUC.EXPECT().
		ConfirmMatch(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, req *models.MatchConfirmRequest) (*models.MatchProposal, error) {
			assert.Equal(t, client.UserID, req.UserID)
			return matchResult, nil
		})

	// Act
	err = wsManager.handleMatchConfirmation(client, json.RawMessage(msgData))

	// Assert
	assert.NoError(t, err)
	assert.Len(t, mockManager.notifications, 2)
	assert.Equal(t, driverID, mockManager.notifications[0].UserID)
	assert.Equal(t, constants.EventMatchConfirm, mockManager.notifications[0].Event)
	assert.Equal(t, passengerID, mockManager.notifications[1].UserID)
	assert.Equal(t, constants.EventMatchConfirm, mockManager.notifications[1].Event)
}

func TestHandleMatchConfirmation_InvalidJSON(t *testing.T) {
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

	invalidJSON := json.RawMessage(`{"broken": json`)

	// Act
	err := wsManager.handleMatchConfirmation(client, invalidJSON)

	// Assert
	assert.NoError(t, err) // Function returns nil but sends error message
	assert.Len(t, mockManager.sentErrors, 1)
	assert.Equal(t, constants.ErrorInvalidFormat, mockManager.sentErrors[0].ErrorCode)
	assert.Contains(t, mockManager.sentErrors[0].Message, "Invalid match proposal format")
}

func TestHandleMatchConfirmation_UsecaseError(t *testing.T) {
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

	matchConfirm := models.MatchConfirmRequest{
		ID:     uuid.New().String(),
		Status: "ACCEPTED",
		UserID: client.UserID,
	}

	msgData, err := json.Marshal(matchConfirm)
	assert.NoError(t, err)

	mockUserUC.EXPECT().
		ConfirmMatch(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("match confirmation failed"))

	// Act
	err = wsManager.handleMatchConfirmation(client, json.RawMessage(msgData))

	// Assert
	assert.NoError(t, err) // Function returns nil but sends error message
	assert.Len(t, mockManager.sentErrors, 1)
	assert.Equal(t, constants.ErrorMatchUpdateFailed, mockManager.sentErrors[0].ErrorCode)
}
