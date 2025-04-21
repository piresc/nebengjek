package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
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

	// The implementation calls FindNearbyPassengers
	mockRepo.EXPECT().
		AddAvailableDriver(gomock.Any(), userID, gomock.Any()).
		DoAndReturn(func(_ context.Context, id string, loc *models.Location) error {
			assert.Equal(t, userID, id)
			assert.Equal(t, event.Location.Latitude, loc.Latitude)
			assert.Equal(t, event.Location.Longitude, loc.Longitude)
			return nil
		})

	// Need to mock FindNearbyPassengers as it's called by the handler
	mockRepo.EXPECT().
		FindNearbyPassengers(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]*models.NearbyUser{}, nil) // Return empty array to avoid further processing

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

	// Mock required calls
	mockRepo.EXPECT().
		AddAvailablePassenger(gomock.Any(), userID, gomock.Any()).
		Return(nil)

	// Need to mock FindNearbyDrivers as it's called by the handler
	mockRepo.EXPECT().
		FindNearbyDrivers(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]*models.NearbyUser{}, nil) // Return empty array to avoid further processing

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
		IsActive: false, // User is going offline
		Role:     "driver",
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
	}

	// Set up expectations
	mockRepo.EXPECT().
		RemoveAvailableDriver(gomock.Any(), userID).
		Return(nil)

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
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
	}

	expectedError := errors.New("database error")

	// Set up expectations
	mockRepo.EXPECT().
		AddAvailableDriver(gomock.Any(), userID, gomock.Any()).
		Return(expectedError)

	// Act
	err := uc.HandleBeaconEvent(event)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestConfirmMatchStatus_AcceptSuccess(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)

	uc := NewMatchUC(mockRepo, mockGW)

	matchID := "match-123"
	driverID := uuid.New()
	passengerID := uuid.New()
	driverIDStr := driverID.String()
	passengerIDStr := passengerID.String()

	matchProposal := models.MatchProposal{
		ID:          matchID,
		DriverID:    driverIDStr,
		PassengerID: passengerIDStr,
		MatchStatus: models.MatchStatusAccepted,
	}

	// The usecase first gets the match to validate users
	mockRepo.EXPECT().
		GetMatch(gomock.Any(), matchID).
		Return(&models.Match{
			ID:          converter.StrToUUID(matchID),
			DriverID:    driverID,
			PassengerID: passengerID,
			Status:      models.MatchStatusPending,
		}, nil)

	// Then it uses atomic operations to update match status
	mockRepo.EXPECT().
		ConfirmMatchAtomically(gomock.Any(), matchID, models.MatchStatusAccepted).
		Return(nil)

	// Need to mock ListMatchesByPassenger as it's called when match is accepted
	mockRepo.EXPECT().
		ListMatchesByPassenger(gomock.Any(), passengerID).
		Return([]*models.Match{}, nil)

	// Publish confirmation
	mockRepo.EXPECT().
		RemoveAvailableDriver(gomock.Any(), driverIDStr).
		Return(nil)
	mockRepo.EXPECT().
		RemoveAvailablePassenger(gomock.Any(), passengerIDStr).
		Return(nil)

	mockGW.EXPECT().
		PublishMatchConfirm(gomock.Any(), gomock.Any()).
		Return(nil)

	// Act
	err := uc.ConfirmMatchStatus(matchID, matchProposal)

	// Assert
	assert.NoError(t, err)
}

func TestConfirmMatchStatus_RejectSuccess(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)

	uc := NewMatchUC(mockRepo, mockGW)

	matchID := "match-123"
	driverID := uuid.New()
	passengerID := uuid.New()
	driverIDStr := driverID.String()
	passengerIDStr := passengerID.String()

	matchProposal := models.MatchProposal{
		ID:          matchID,
		DriverID:    driverIDStr,
		PassengerID: passengerIDStr,
		MatchStatus: models.MatchStatusRejected,
	}

	mockRepo.EXPECT().
		GetMatch(gomock.Any(), matchID).
		Return(&models.Match{
			ID:          converter.StrToUUID(matchID),
			DriverID:    driverID,
			PassengerID: passengerID,
			Status:      models.MatchStatusPending,
		}, nil)

	mockRepo.EXPECT().
		ConfirmMatchAtomically(gomock.Any(), matchID, models.MatchStatusRejected).
		Return(nil)

	// Act
	err := uc.ConfirmMatchStatus(matchID, matchProposal)

	// Assert
	assert.NoError(t, err)
}

func TestConfirmMatchStatus_AtomicUpdateError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)

	uc := NewMatchUC(mockRepo, mockGW)

	matchID := "match-123"
	driverID := uuid.New()
	passengerID := uuid.New()
	driverIDStr := driverID.String()
	passengerIDStr := passengerID.String()

	matchProposal := models.MatchProposal{
		ID:          matchID,
		DriverID:    driverIDStr,
		PassengerID: passengerIDStr,
		MatchStatus: models.MatchStatusAccepted,
	}

	expectedError := errors.New("atomic update error")

	// Set up expectations
	mockRepo.EXPECT().
		GetMatch(gomock.Any(), matchID).
		Return(&models.Match{
			ID:          converter.StrToUUID(matchID),
			DriverID:    driverID,
			PassengerID: passengerID,
			Status:      models.MatchStatusPending,
		}, nil)

	mockRepo.EXPECT().
		ConfirmMatchAtomically(gomock.Any(), matchID, models.MatchStatusAccepted).
		Return(expectedError)

	// Act
	err := uc.ConfirmMatchStatus(matchID, matchProposal)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to confirm match")
}

func TestConfirmMatchStatus_GetMatchError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)

	uc := NewMatchUC(mockRepo, mockGW)

	matchID := "match-123"
	driverID := uuid.New().String()
	passengerID := uuid.New().String()

	matchProposal := models.MatchProposal{
		ID:          matchID,
		DriverID:    driverID,
		PassengerID: passengerID,
		MatchStatus: models.MatchStatusAccepted,
	}

	expectedError := errors.New("database error")

	// Set up expectations
	mockRepo.EXPECT().
		GetMatch(gomock.Any(), matchID).
		Return(nil, expectedError)

	// Act
	err := uc.ConfirmMatchStatus(matchID, matchProposal)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get match")
}

func TestConfirmMatchStatus_PublishError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)

	uc := NewMatchUC(mockRepo, mockGW)

	matchID := "match-123"

	// Create UUIDs with fixed values to avoid mismatches in the mock expectations
	driverID := uuid.MustParse("93646c59-ce17-4b07-a845-fc6562adaf83")
	passengerID := uuid.MustParse("1cb19a6d-ad61-4d84-ab94-084f11676d0e")
	driverIDStr := driverID.String()
	passengerIDStr := passengerID.String()

	matchProposal := models.MatchProposal{
		ID:          matchID,
		DriverID:    driverIDStr,
		PassengerID: passengerIDStr,
		MatchStatus: models.MatchStatusAccepted,
	}

	matchObject := &models.Match{
		ID:          converter.StrToUUID(matchID),
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      models.MatchStatusPending,
	}

	expectedError := errors.New("publish error")

	// Set up expectations in the order they'll be called

	// First, GetMatch is called to validate the users
	mockRepo.EXPECT().
		GetMatch(gomock.Any(), matchID).
		Return(matchObject, nil)

	// Then ConfirmMatchAtomically is called to update the match status
	mockRepo.EXPECT().
		ConfirmMatchAtomically(gomock.Any(), matchID, models.MatchStatusAccepted).
		Return(nil)

	// The publish call will fail with our expected error
	mockGW.EXPECT().
		PublishMatchConfirm(gomock.Any(), gomock.Any()).
		Return(expectedError)

	// Act
	err := uc.ConfirmMatchStatus(matchID, matchProposal)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish match")
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
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
	}

	// Mock adding the driver to available pool
	mockRepo.EXPECT().
		AddAvailableDriver(gomock.Any(), userID, gomock.Any()).
		Return(nil)

	// Mock finding nearby passengers - return some passengers
	passengerID := uuid.New().String()
	nearbyPassengers := []*models.NearbyUser{
		{
			ID: passengerID,
			Location: models.Location{
				Latitude:  -6.175492,
				Longitude: 106.827253,
			},
		},
	}
	mockRepo.EXPECT().
		FindNearbyPassengers(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nearbyPassengers, nil)

	// Mock creating a match for the nearby passenger
	mockRepo.EXPECT().
		CreateMatch(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, match *models.Match) (*models.Match, error) {
			assert.Equal(t, converter.StrToUUID(userID), match.DriverID)
			assert.Equal(t, converter.StrToUUID(passengerID), match.PassengerID)
			assert.Equal(t, models.MatchStatusPending, match.Status)

			// Add an ID to the match
			match.ID = uuid.New()
			return match, nil
		})

	// Mock storing match proposal in Redis
	mockRepo.EXPECT().
		StoreMatchProposal(gomock.Any(), gomock.Any()).
		Return(nil)

	// Mock publishing match found event
	mockGW.EXPECT().
		PublishMatchFound(gomock.Any(), gomock.Any()).
		Return(nil)

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
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
	}

	// Mock adding the passenger to available pool
	mockRepo.EXPECT().
		AddAvailablePassenger(gomock.Any(), userID, gomock.Any()).
		Return(nil)

	// Mock finding nearby drivers - return some drivers
	driverID := uuid.New().String()
	nearbyDrivers := []*models.NearbyUser{
		{
			ID: driverID,
			Location: models.Location{
				Latitude:  -6.175492,
				Longitude: 106.827253,
			},
		},
	}
	mockRepo.EXPECT().
		FindNearbyDrivers(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nearbyDrivers, nil)

	// Mock creating a match for the nearby driver
	mockRepo.EXPECT().
		CreateMatch(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, match *models.Match) (*models.Match, error) {
			assert.Equal(t, converter.StrToUUID(driverID), match.DriverID)
			assert.Equal(t, converter.StrToUUID(userID), match.PassengerID)
			assert.Equal(t, models.MatchStatusPending, match.Status)

			// Add an ID to the match
			match.ID = uuid.New()
			return match, nil
		})

	// Mock storing match proposal in Redis
	mockRepo.EXPECT().
		StoreMatchProposal(gomock.Any(), gomock.Any()).
		Return(nil)

	// Mock publishing match found event
	mockGW.EXPECT().
		PublishMatchFound(gomock.Any(), gomock.Any()).
		Return(nil)

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
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
	}

	// Mock adding the driver to available pool
	mockRepo.EXPECT().
		AddAvailableDriver(gomock.Any(), userID, gomock.Any()).
		Return(nil)

	// Mock finding nearby passengers with error
	expectedError := errors.New("database error")
	mockRepo.EXPECT().
		FindNearbyPassengers(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, expectedError)

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
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
	}

	// Mock adding the passenger to available pool
	mockRepo.EXPECT().
		AddAvailablePassenger(gomock.Any(), userID, gomock.Any()).
		Return(nil)

	// Mock finding nearby drivers with error
	expectedError := errors.New("database error")
	mockRepo.EXPECT().
		FindNearbyDrivers(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, expectedError)

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
	matchID := uuid.New()

	match := &models.Match{
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      models.MatchStatusPending,
	}

	// Mock creating match in database
	mockRepo.EXPECT().
		CreateMatch(gomock.Any(), match).
		DoAndReturn(func(_ context.Context, m *models.Match) (*models.Match, error) {
			m.ID = matchID
			return m, nil
		})

	// Mock storing match proposal in Redis
	mockRepo.EXPECT().
		StoreMatchProposal(gomock.Any(), gomock.Any()).
		Return(nil)

	// Mock publishing match found event
	mockGW.EXPECT().
		PublishMatchFound(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, mp models.MatchProposal) error {
			assert.Equal(t, matchID.String(), mp.ID)
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

func TestCreateMatch_DatabaseError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)

	uc := NewMatchUC(mockRepo, mockGW)

	driverID := uuid.New()
	passengerID := uuid.New()

	match := &models.Match{
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      models.MatchStatusPending,
	}

	expectedError := errors.New("database error")

	// Mock creating match in database with error
	mockRepo.EXPECT().
		CreateMatch(gomock.Any(), match).
		Return(nil, expectedError)

	// Act
	err := uc.CreateMatch(context.Background(), match)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create match")
}

func TestCreateMatch_StoreProposalError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)

	uc := NewMatchUC(mockRepo, mockGW)

	driverID := uuid.New()
	passengerID := uuid.New()
	matchID := uuid.New()

	match := &models.Match{
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      models.MatchStatusPending,
	}

	expectedError := errors.New("redis error")

	// Mock creating match in database
	mockRepo.EXPECT().
		CreateMatch(gomock.Any(), match).
		DoAndReturn(func(_ context.Context, m *models.Match) (*models.Match, error) {
			m.ID = matchID
			return m, nil
		})

	// Mock storing match proposal in Redis with error
	mockRepo.EXPECT().
		StoreMatchProposal(gomock.Any(), gomock.Any()).
		Return(expectedError)

	// Act
	err := uc.CreateMatch(context.Background(), match)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store match proposal")
}

func TestCreateMatch_PublishError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)

	uc := NewMatchUC(mockRepo, mockGW)

	driverID := uuid.New()
	passengerID := uuid.New()
	matchID := uuid.New()

	match := &models.Match{
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      models.MatchStatusPending,
	}

	expectedError := errors.New("publish error")

	// Mock creating match in database
	mockRepo.EXPECT().
		CreateMatch(gomock.Any(), match).
		DoAndReturn(func(_ context.Context, m *models.Match) (*models.Match, error) {
			m.ID = matchID
			return m, nil
		})

	// Mock storing match proposal in Redis
	mockRepo.EXPECT().
		StoreMatchProposal(gomock.Any(), gomock.Any()).
		Return(nil)

	// Mock publishing match found event with error
	mockGW.EXPECT().
		PublishMatchFound(gomock.Any(), gomock.Any()).
		Return(expectedError)

	// Act
	err := uc.CreateMatch(context.Background(), match)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish match proposal")
}

func TestConfirmMatchStatus_WithOtherMatches(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)

	uc := NewMatchUC(mockRepo, mockGW)

	// Create match IDs and user IDs
	mainMatchID := uuid.New()
	otherMatchID := uuid.New()
	mainMatchIDStr := mainMatchID.String()

	driverID := uuid.New()
	passengerID := uuid.New()
	otherDriverID := uuid.New()

	driverIDStr := driverID.String()
	passengerIDStr := passengerID.String()

	matchProposal := models.MatchProposal{
		ID:          mainMatchIDStr,
		DriverID:    driverIDStr,
		PassengerID: passengerIDStr,
		MatchStatus: models.MatchStatusAccepted,
	}

	// Mock getting the match to validate users
	mockRepo.EXPECT().
		GetMatch(gomock.Any(), mainMatchIDStr).
		Return(&models.Match{
			ID:          mainMatchID,
			DriverID:    driverID,
			PassengerID: passengerID,
			Status:      models.MatchStatusPending,
		}, nil)

	// Mock atomic confirmation
	mockRepo.EXPECT().
		ConfirmMatchAtomically(gomock.Any(), mainMatchIDStr, models.MatchStatusAccepted).
		Return(nil)

	// Mock listing other matches for the passenger
	mockRepo.EXPECT().
		ListMatchesByPassenger(gomock.Any(), passengerID).
		Return([]*models.Match{
			{
				ID:          mainMatchID,
				DriverID:    driverID,
				PassengerID: passengerID,
				Status:      models.MatchStatusAccepted,
			},
			{
				ID:          otherMatchID,
				DriverID:    otherDriverID,
				PassengerID: passengerID,
				Status:      models.MatchStatusPending,
			},
		}, nil)

	// For the other pending match, test that UpdateMatchStatus is called with any string ID
	mockRepo.EXPECT().
		UpdateMatchStatus(gomock.Any(), gomock.Any(), models.MatchStatusRejected).
		DoAndReturn(func(_ context.Context, id string, status models.MatchStatus) error {
			// Verify that the ID is the string representation of otherMatchID
			assert.Equal(t, otherMatchID.String(), id)
			assert.Equal(t, models.MatchStatusRejected, status)
			return nil
		})

	// Mock publishing rejection for the other match
	mockGW.EXPECT().
		PublishMatchRejected(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, mp models.MatchProposal) error {
			// Verify that the match proposal has the correct IDs
			assert.Equal(t, otherMatchID.String(), mp.ID)
			assert.Equal(t, otherDriverID.String(), mp.DriverID)
			assert.Equal(t, passengerID.String(), mp.PassengerID)
			assert.Equal(t, models.MatchStatusRejected, mp.MatchStatus)
			return nil
		})

	// Mock publishing match confirmation
	mockGW.EXPECT().
		PublishMatchConfirm(gomock.Any(), gomock.Any()).
		Return(nil)

	// Mock removing users from available pools
	mockRepo.EXPECT().RemoveAvailableDriver(gomock.Any(), driverIDStr).Return(nil)
	mockRepo.EXPECT().RemoveAvailablePassenger(gomock.Any(), passengerIDStr).Return(nil)

	// Act
	err := uc.ConfirmMatchStatus(mainMatchIDStr, matchProposal)

	// Assert
	assert.NoError(t, err)
}

func TestConfirmMatchStatus_UpdateRejectedMatch_WithError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)

	uc := NewMatchUC(mockRepo, mockGW)

	// Create match IDs and user IDs
	mainMatchID := uuid.New()
	otherMatchID := uuid.New()
	mainMatchIDStr := mainMatchID.String()

	driverID := uuid.New()
	passengerID := uuid.New()
	otherDriverID := uuid.New()

	driverIDStr := driverID.String()
	passengerIDStr := passengerID.String()

	matchProposal := models.MatchProposal{
		ID:          mainMatchIDStr,
		DriverID:    driverIDStr,
		PassengerID: passengerIDStr,
		MatchStatus: models.MatchStatusAccepted,
	}

	expectedError := errors.New("update error")

	// Mock getting the match to validate users
	mockRepo.EXPECT().
		GetMatch(gomock.Any(), mainMatchIDStr).
		Return(&models.Match{
			ID:          mainMatchID,
			DriverID:    driverID,
			PassengerID: passengerID,
			Status:      models.MatchStatusPending,
		}, nil)

	// Mock atomic confirmation
	mockRepo.EXPECT().
		ConfirmMatchAtomically(gomock.Any(), mainMatchIDStr, models.MatchStatusAccepted).
		Return(nil)

	// Mock listing other matches for the passenger
	mockRepo.EXPECT().
		ListMatchesByPassenger(gomock.Any(), passengerID).
		Return([]*models.Match{
			{
				ID:          mainMatchID,
				DriverID:    driverID,
				PassengerID: passengerID,
				Status:      models.MatchStatusAccepted,
			},
			{
				ID:          otherMatchID,
				DriverID:    otherDriverID,
				PassengerID: passengerID,
				Status:      models.MatchStatusPending,
			},
		}, nil)

	// For the other pending match, test that UpdateMatchStatus is called with error
	mockRepo.EXPECT().
		UpdateMatchStatus(gomock.Any(), gomock.Any(), models.MatchStatusRejected).
		DoAndReturn(func(_ context.Context, id string, status models.MatchStatus) error {
			// Verify that the ID is the string representation of otherMatchID
			assert.Equal(t, otherMatchID.String(), id)
			assert.Equal(t, models.MatchStatusRejected, status)
			return expectedError
		})

	// Mock publishing match confirmation
	mockGW.EXPECT().
		PublishMatchConfirm(gomock.Any(), gomock.Any()).
		Return(nil)

	// Mock removing users from available pools
	mockRepo.EXPECT().RemoveAvailableDriver(gomock.Any(), driverIDStr).Return(nil)
	mockRepo.EXPECT().RemoveAvailablePassenger(gomock.Any(), passengerIDStr).Return(nil)

	// Act
	err := uc.ConfirmMatchStatus(mainMatchIDStr, matchProposal)

	// Assert
	assert.NoError(t, err) // The function still succeeds even if rejecting other matches fails
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
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
	}

	expectedError := errors.New("database error")

	// Mock removing passenger with error
	mockRepo.EXPECT().
		RemoveAvailablePassenger(gomock.Any(), userID).
		Return(expectedError)

	// Act
	err := uc.HandleBeaconEvent(event)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestConfirmMatchStatus_InvalidUser(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)

	uc := NewMatchUC(mockRepo, mockGW)

	matchID := "match-123"
	driverID := uuid.New()
	passengerID := uuid.New()
	invalidUserID := uuid.New().String()

	matchProposal := models.MatchProposal{
		ID:          matchID,
		DriverID:    invalidUserID, // User not part of the match
		PassengerID: invalidUserID,
		MatchStatus: models.MatchStatusAccepted,
	}

	// Mock getting the match to validate users
	mockRepo.EXPECT().
		GetMatch(gomock.Any(), matchID).
		Return(&models.Match{
			ID:          converter.StrToUUID(matchID),
			DriverID:    driverID,
			PassengerID: passengerID,
			Status:      models.MatchStatusPending,
		}, nil)

	// Act
	err := uc.ConfirmMatchStatus(matchID, matchProposal)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user is not part of this match")
}

func TestConfirmMatchStatus_RemoveAvailableUsersError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)

	uc := NewMatchUC(mockRepo, mockGW)

	matchIDStr := "match-123"
	matchID := converter.StrToUUID(matchIDStr)
	driverID := uuid.New()
	passengerID := uuid.New()
	driverIDStr := driverID.String()
	passengerIDStr := passengerID.String()

	matchProposal := models.MatchProposal{
		ID:          matchIDStr,
		DriverID:    driverIDStr,
		PassengerID: passengerIDStr,
		MatchStatus: models.MatchStatusAccepted,
	}

	expectedError := errors.New("remove error")

	// Mock getting the match to validate users
	mockRepo.EXPECT().
		GetMatch(gomock.Any(), matchIDStr).
		Return(&models.Match{
			ID:          matchID,
			DriverID:    driverID,
			PassengerID: passengerID,
			Status:      models.MatchStatusPending,
		}, nil)

	// Mock atomic confirmation
	mockRepo.EXPECT().
		ConfirmMatchAtomically(gomock.Any(), matchIDStr, models.MatchStatusAccepted).
		Return(nil)

	// Mock listing other matches for the passenger - empty for simplicity
	mockRepo.EXPECT().
		ListMatchesByPassenger(gomock.Any(), passengerID).
		Return([]*models.Match{}, nil)

	// Mock publishing match confirmation
	mockGW.EXPECT().
		PublishMatchConfirm(gomock.Any(), gomock.Any()).
		Return(nil)

	// Mock removing driver from available pool with error
	mockRepo.EXPECT().
		RemoveAvailableDriver(gomock.Any(), driverIDStr).
		Return(expectedError)

	// Mock removing passenger from available pool
	mockRepo.EXPECT().
		RemoveAvailablePassenger(gomock.Any(), passengerIDStr).
		Return(nil)

	// Act
	err := uc.ConfirmMatchStatus(matchIDStr, matchProposal)

	// Assert
	assert.NoError(t, err) // Function continues despite error
}
