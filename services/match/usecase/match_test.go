package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/converter"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/match/mocks"
	"github.com/stretchr/testify/assert"
)

func TestHandleBeaconEvent_Success_Driver(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)

	uc := NewMatchUC(mockRepo, mockGW)

	userID := uuid.New().String()
	event := models.BeaconEvent{
		UserID:   userID,
		IsActive: true,
		Role:     "driver",
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
	}

	mockRepo.EXPECT().AddAvailableDriver(gomock.Any(), userID, gomock.Any()).Return(nil)
	mockRepo.EXPECT().FindNearbyPassengers(gomock.Any(), gomock.Any(), gomock.Any()).Return([]*models.NearbyUser{}, nil)

	// Act
	err := uc.HandleBeaconEvent(event)

	// Assert
	assert.NoError(t, err)
}

func TestHandleBeaconEvent_Success_Passenger(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	uc := NewMatchUC(mockRepo, mockGW)

	userID := uuid.New().String()
	event := models.BeaconEvent{
		UserID:   userID,
		IsActive: true,
		Role:     "passenger",
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
	}

	mockRepo.EXPECT().AddAvailablePassenger(gomock.Any(), userID, gomock.Any()).Return(nil)
	mockRepo.EXPECT().FindNearbyDrivers(gomock.Any(), gomock.Any(), gomock.Any()).Return([]*models.NearbyUser{}, nil)

	// Act
	err := uc.HandleBeaconEvent(event)

	// Assert
	assert.NoError(t, err)
}

func TestHandleBeaconEvent_Inactive(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	uc := NewMatchUC(mockRepo, mockGW)

	userID := uuid.New().String()
	event := models.BeaconEvent{
		UserID:   userID,
		IsActive: false,
		Role:     "driver",
	}

	mockRepo.EXPECT().RemoveAvailableDriver(gomock.Any(), userID).Return(nil)

	// Act
	err := uc.HandleBeaconEvent(event)

	// Assert
	assert.NoError(t, err)
}

func TestHandleBeaconEvent_RepositoryError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	uc := NewMatchUC(mockRepo, mockGW)

	userID := uuid.New().String()
	event := models.BeaconEvent{
		UserID:   userID,
		IsActive: true,
		Role:     "driver",
		Location: models.Location{Latitude: -6.175392, Longitude: 106.827153, Timestamp: time.Now()},
	}
	expectedError := errors.New("database error")
	mockRepo.EXPECT().AddAvailableDriver(gomock.Any(), userID, gomock.Any()).Return(expectedError)

	// Act
	err := uc.HandleBeaconEvent(event)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestHandleActiveDriver_WithNearbyPassengers(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	uc := NewMatchUC(mockRepo, mockGW)

	userID := uuid.New().String()
	event := models.BeaconEvent{
		UserID:   userID,
		IsActive: true,
		Role:     "driver",
		Location: models.Location{Latitude: -6.175392, Longitude: 106.827153, Timestamp: time.Now()},
	}

	passengerID := uuid.New().String()
	nearbyPassengers := []*models.NearbyUser{{ID: passengerID, Location: models.Location{Latitude: -6.175492, Longitude: 106.827253}}}

	mockRepo.EXPECT().AddAvailableDriver(gomock.Any(), userID, gomock.Any()).Return(nil)
	mockRepo.EXPECT().FindNearbyPassengers(gomock.Any(), gomock.Any(), gomock.Any()).Return(nearbyPassengers, nil)
	mockRepo.EXPECT().CreatePendingMatch(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, match *models.Match) (string, error) {
			assert.Equal(t, userID, match.DriverID.String())
			assert.Equal(t, passengerID, match.PassengerID.String())
			assert.Equal(t, models.MatchStatusPending, match.Status)
			return match.DriverID.String() + "-" + match.PassengerID.String(), nil // Return a constructed match ID string
		})
	mockGW.EXPECT().PublishMatchFound(gomock.Any(), gomock.Any()).Return(nil)

	// Act
	err := uc.HandleBeaconEvent(event)

	// Assert
	assert.NoError(t, err)
}

func TestHandleActivePassenger_WithNearbyDrivers(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	uc := NewMatchUC(mockRepo, mockGW)

	userID := uuid.New().String()
	event := models.BeaconEvent{
		UserID:   userID,
		IsActive: true,
		Role:     "passenger",
		Location: models.Location{Latitude: -6.175392, Longitude: 106.827153, Timestamp: time.Now()},
	}

	driverID := uuid.New().String()
	nearbyDrivers := []*models.NearbyUser{{ID: driverID, Location: models.Location{Latitude: -6.175492, Longitude: 106.827253}}}

	mockRepo.EXPECT().AddAvailablePassenger(gomock.Any(), userID, gomock.Any()).Return(nil)
	mockRepo.EXPECT().FindNearbyDrivers(gomock.Any(), gomock.Any(), gomock.Any()).Return(nearbyDrivers, nil)
	mockRepo.EXPECT().CreatePendingMatch(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, match *models.Match) (string, error) {
			assert.Equal(t, driverID, match.DriverID.String())
			assert.Equal(t, userID, match.PassengerID.String())
			assert.Equal(t, models.MatchStatusPending, match.Status)
			return match.DriverID.String() + "-" + match.PassengerID.String(), nil
		})
	mockGW.EXPECT().PublishMatchFound(gomock.Any(), gomock.Any()).Return(nil)

	// Act
	err := uc.HandleBeaconEvent(event)

	// Assert
	assert.NoError(t, err)
}

func TestHandleActiveDriver_FindNearbyPassengersError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	uc := NewMatchUC(mockRepo, mockGW)

	userID := uuid.New().String()
	event := models.BeaconEvent{
		UserID:   userID,
		IsActive: true,
		Role:     "driver",
		Location: models.Location{Latitude: -6.175392, Longitude: 106.827153, Timestamp: time.Now()},
	}
	expectedError := errors.New("database error")

	mockRepo.EXPECT().AddAvailableDriver(gomock.Any(), userID, gomock.Any()).Return(nil)
	mockRepo.EXPECT().FindNearbyPassengers(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, expectedError)

	// Act
	err := uc.HandleBeaconEvent(event)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestHandleActivePassenger_FindNearbyDriversError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	uc := NewMatchUC(mockRepo, mockGW)

	userID := uuid.New().String()
	event := models.BeaconEvent{
		UserID:   userID,
		IsActive: true,
		Role:     "passenger",
		Location: models.Location{Latitude: -6.175392, Longitude: 106.827153, Timestamp: time.Now()},
	}
	expectedError := errors.New("database error")

	mockRepo.EXPECT().AddAvailablePassenger(gomock.Any(), userID, gomock.Any()).Return(nil)
	mockRepo.EXPECT().FindNearbyDrivers(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, expectedError)

	// Act
	err := uc.HandleBeaconEvent(event)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestCreateMatch_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	uc := NewMatchUC(mockRepo, mockGW)

	driverID := uuid.New()
	passengerID := uuid.New()
	matchIDStr := driverID.String() + "-" + passengerID.String() // Example match ID

	match := &models.Match{DriverID: driverID, PassengerID: passengerID, Status: models.MatchStatusPending}

	mockRepo.EXPECT().CreatePendingMatch(gomock.Any(), match).Return(matchIDStr, nil)
	mockGW.EXPECT().PublishMatchFound(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, mp models.MatchProposal) error {
			assert.Equal(t, matchIDStr, mp.ID)
			assert.Equal(t, driverID.String(), mp.DriverID)
			assert.Equal(t, passengerID.String(), mp.PassengerID)
			assert.Equal(t, models.MatchStatusPending, mp.MatchStatus)
			return nil
		})

	// Act
	err := uc.CreateMatch(context.Background(), match)

	// Assert
	assert.NoError(t, err)
}

func TestCreateMatch_CreatePendingMatchError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	uc := NewMatchUC(mockRepo, mockGW)

	match := &models.Match{DriverID: uuid.New(), PassengerID: uuid.New(), Status: models.MatchStatusPending}
	expectedError := errors.New("redis error")

	mockRepo.EXPECT().CreatePendingMatch(gomock.Any(), match).Return("", expectedError)

	// Act
	err := uc.CreateMatch(context.Background(), match)

	// Assert
	assert.Error(t, err)
	assert.True(t, errors.Is(err, expectedError))
	assert.Contains(t, err.Error(), "failed to create pending match")
}

func TestCreateMatch_PublishError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	uc := NewMatchUC(mockRepo, mockGW)

	match := &models.Match{DriverID: uuid.New(), PassengerID: uuid.New(), Status: models.MatchStatusPending}
	matchIDStr := "match-id-publish-error"
	expectedError := errors.New("publish error")

	mockRepo.EXPECT().CreatePendingMatch(gomock.Any(), match).Return(matchIDStr, nil)
	mockGW.EXPECT().PublishMatchFound(gomock.Any(), gomock.Any()).Return(expectedError)

	// Act
	err := uc.CreateMatch(context.Background(), match)

	// Assert
	assert.Error(t, err)
	assert.True(t, errors.Is(err, expectedError))
	assert.Contains(t, err.Error(), "failed to publish match proposal")
}

// TestConfirmMatchStatus_PendingCustomerConfirmation tests driver initial acceptance.
func TestConfirmMatchStatus_PendingCustomerConfirmation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	uc := NewMatchUC(mockRepo, mockGW)

	mp := models.MatchProposal{
		ID:          "match-pending-cust-123",
		DriverID:    uuid.NewString(),
		PassengerID: uuid.NewString(),
		MatchStatus: models.MatchStatusPendingCustomerConfirmation,
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.EXPECT().UpdateMatchStatus(gomock.Any(), mp.ID, models.MatchStatusPendingCustomerConfirmation, mp.DriverID, mp.PassengerID).Return(nil).Times(1)
		mockGW.EXPECT().PublishMatchPendingCustomerConfirmation(gomock.Any(), mp).Return(nil).Times(1)

		err := uc.ConfirmMatchStatus(mp)
		assert.NoError(t, err)
	})

	t.Run("PublishError", func(t *testing.T) {
		expectedError := errors.New("publish pending error")
		mockRepo.EXPECT().UpdateMatchStatus(gomock.Any(), mp.ID, models.MatchStatusPendingCustomerConfirmation, mp.DriverID, mp.PassengerID).Return(nil).Times(1)
		mockGW.EXPECT().PublishMatchPendingCustomerConfirmation(gomock.Any(), mp).Return(expectedError).Times(1)

		err := uc.ConfirmMatchStatus(mp)
		assert.Error(t, err)
		assert.Equal(t, expectedError, err) 
	})
}

// TestConfirmMatchStatus_Accepted tests customer's final acceptance.
func TestConfirmMatchStatus_Accepted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	uc := NewMatchUC(mockRepo, mockGW)

	matchID := uuid.New()
	driverID := uuid.NewString()
	passengerID := uuid.NewString()

	mp := models.MatchProposal{
		ID:          matchID.String(),
		DriverID:    driverID,
		PassengerID: passengerID,
		MatchStatus: models.MatchStatusAccepted,
	}

	persistedMatch := &models.Match{
		ID:          matchID,
		DriverID:    converter.StrToUUID(driverID),
		PassengerID: converter.StrToUUID(passengerID),
		Status:      models.MatchStatusAccepted,
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.EXPECT().ConfirmAndPersistMatch(gomock.Any(), driverID, passengerID).Return(persistedMatch, nil).Times(1)
		mockGW.EXPECT().PublishMatchConfirm(gomock.Any(), gomock.Any()).
										DoAndReturn(func(_ context.Context, event models.MatchProposal) error {
			assert.Equal(t, persistedMatch.ID.String(), event.ID)
			return nil
		}).Times(1)
		mockRepo.EXPECT().RemoveAvailableDriver(gomock.Any(), driverID).Return(nil).Times(1)
		mockRepo.EXPECT().RemoveAvailablePassenger(gomock.Any(), passengerID).Return(nil).Times(1)

		err := uc.ConfirmMatchStatus(mp)
		assert.NoError(t, err)
	})

	t.Run("PublishError", func(t *testing.T) {
		expectedError := errors.New("publish accepted error")
		mockRepo.EXPECT().ConfirmAndPersistMatch(gomock.Any(), driverID, passengerID).Return(persistedMatch, nil).Times(1)
		mockGW.EXPECT().PublishMatchConfirm(gomock.Any(), gomock.Any()).Return(expectedError).Times(1)
		mockRepo.EXPECT().RemoveAvailableDriver(gomock.Any(), gomock.Any()).Times(0) 
		mockRepo.EXPECT().RemoveAvailablePassenger(gomock.Any(), gomock.Any()).Times(0)


		err := uc.ConfirmMatchStatus(mp)
		assert.Error(t, err)
		assert.Equal(t, expectedError, err) 
	})
}

// TestConfirmMatchStatus_Rejected tests match rejection.
func TestConfirmMatchStatus_Rejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	uc := NewMatchUC(mockRepo, mockGW)

	mp := models.MatchProposal{
		ID:          "match-rejected-123",
		DriverID:    uuid.NewString(),
		PassengerID: uuid.NewString(),
		MatchStatus: models.MatchStatusRejected,
	}

	t.Run("Success", func(t *testing.T) {
		mockRepo.EXPECT().DeleteRedisKey(gomock.Any(), fmt.Sprintf("driver_match:%s", mp.DriverID)).Return(nil).Times(1)
		mockRepo.EXPECT().DeleteRedisKey(gomock.Any(), fmt.Sprintf("passenger_match:%s", mp.PassengerID)).Return(nil).Times(1)
		mockRepo.EXPECT().DeleteRedisKey(gomock.Any(), fmt.Sprintf("pending_match_pair:%s:%s", mp.DriverID, mp.PassengerID)).Return(nil).Times(1)
		mockGW.EXPECT().PublishMatchRejected(gomock.Any(), mp).Return(nil).Times(1)

		err := uc.ConfirmMatchStatus(mp)
		assert.NoError(t, err)
	})

	t.Run("PublishError", func(t *testing.T) {
		expectedError := errors.New("publish rejected error")
		mockRepo.EXPECT().DeleteRedisKey(gomock.Any(), gomock.Any()).Return(nil).AnyTimes() // Using nil for Return() as it's best-effort
		mockGW.EXPECT().PublishMatchRejected(gomock.Any(), mp).Return(expectedError).Times(1)

		err := uc.ConfirmMatchStatus(mp)
		assert.Error(t, err)
		assert.Equal(t, expectedError, err) 
	})
}

// TestConfirmMatchStatus_UnknownStatus tests error handling for an unknown match status.
func TestConfirmMatchStatus_UnknownStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	uc := NewMatchUC(mockRepo, mockGW)

	mp := models.MatchProposal{
		ID:          "match-unknown-123",
		DriverID:    uuid.NewString(),
		PassengerID: uuid.NewString(),
		MatchStatus: "SOME_INVALID_STATUS",
	}

	err := uc.ConfirmMatchStatus(mp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown match status: SOME_INVALID_STATUS")
}

func TestHandleInactiveUser_Error(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	uc := NewMatchUC(mockRepo, mockGW)

	userID := uuid.New().String()
	event := models.BeaconEvent{
		UserID:   userID,
		IsActive: false,
		Role:     "passenger",
	}
	expectedError := errors.New("database error")
	mockRepo.EXPECT().RemoveAvailablePassenger(gomock.Any(), userID).Return(expectedError)

	// Act
	err := uc.HandleBeaconEvent(event)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}
