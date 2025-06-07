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
	cfg := &models.Config{
		Match: models.MatchConfig{
			SearchRadiusKm: 5.0,
		},
	}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	userID := uuid.New().String()
	event := models.BeaconEvent{
		UserID:   userID,
		IsActive: true,
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	// Mock active ride check - driver has no active ride
	mockRepo.EXPECT().
		GetActiveRideByDriver(gomock.Any(), userID).
		Return("", nil).
		Times(1)

	// The implementation calls AddAvailableDriver after active ride check
	mockGW.EXPECT().
		AddAvailableDriver(gomock.Any(), userID, gomock.Any()).
		DoAndReturn(func(_ context.Context, id string, loc *models.Location) error {
			assert.Equal(t, userID, id)
			assert.Equal(t, event.Location.Latitude, loc.Latitude)
			assert.Equal(t, event.Location.Longitude, loc.Longitude)
			return nil
		})

	// Act
	err := uc.HandleBeaconEvent(context.Background(), event)

	// Assert
	assert.NoError(t, err)
}

func TestHandleFinderEvent_Success_Passenger(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	cfg := &models.Config{
		Match: models.MatchConfig{
			SearchRadiusKm: 5.0,
		},
	}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	userID := uuid.New().String()
	event := models.FinderEvent{
		UserID:   userID,
		IsActive: true,
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
		TargetLocation: models.Location{
			Latitude:  -6.200000,
			Longitude: 106.816666,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	// Mock active ride check - passenger has no active ride
	mockRepo.EXPECT().
		GetActiveRideByPassenger(gomock.Any(), userID).
		Return("", nil).
		Times(1)

	// Mock required calls
	mockGW.EXPECT().
		AddAvailablePassenger(gomock.Any(), userID, gomock.Any()).
		Return(nil)

	// Need to mock FindNearbyDrivers as it's called by the handler
	mockGW.EXPECT().
		FindNearbyDrivers(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]*models.NearbyUser{}, nil) // Return empty array to avoid further processing

	// Act
	err := uc.HandleFinderEvent(context.Background(), event)

	// Assert
	assert.NoError(t, err)
}

func TestHandleBeaconEvent_Inactive(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	cfg := &models.Config{
		Match: models.MatchConfig{
			SearchRadiusKm: 5.0,
		},
	}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	userID := uuid.New().String()
	event := models.BeaconEvent{
		UserID:   userID,
		IsActive: false, // User is going offline
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	// Set up expectations
	mockGW.EXPECT().
		RemoveAvailableDriver(gomock.Any(), userID).
		Return(nil)

	// Act
	err := uc.HandleBeaconEvent(context.Background(), event)

	// Assert
	assert.NoError(t, err)
}

func TestHandleBeaconEvent_RepositoryError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	cfg := &models.Config{
		Match: models.MatchConfig{
			SearchRadiusKm: 5.0,
		},
	}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	userID := uuid.New().String()
	event := models.BeaconEvent{
		UserID:   userID,
		IsActive: true,
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	expectedError := errors.New("database error")

	// Mock active ride check - driver has no active ride
	mockRepo.EXPECT().
		GetActiveRideByDriver(gomock.Any(), userID).
		Return("", nil).
		Times(1)

	// Set up expectations
	mockGW.EXPECT().
		AddAvailableDriver(gomock.Any(), userID, gomock.Any()).
		Return(expectedError)

	// Act
	err := uc.HandleBeaconEvent(context.Background(), event)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestHandleBeaconEvent_DriverWithActiveRide(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	cfg := &models.Config{
		Match: models.MatchConfig{
			SearchRadiusKm: 5.0,
		},
	}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	userID := uuid.New().String()
	event := models.BeaconEvent{
		UserID:   userID,
		IsActive: true,
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	// Mock active ride check - driver has an active ride
	mockRepo.EXPECT().
		GetActiveRideByDriver(gomock.Any(), userID).
		Return("active-ride-123", nil).
		Times(1)

	// AddAvailableDriver should NOT be called since driver has active ride

	// Act
	err := uc.HandleBeaconEvent(context.Background(), event)

	// Assert
	assert.NoError(t, err) // Should not return error, just skip adding to pool
}

func TestHandleFinderEvent_PassengerWithActiveRide(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	cfg := &models.Config{
		Match: models.MatchConfig{
			SearchRadiusKm: 5.0,
		},
	}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	userID := uuid.New().String()
	event := models.FinderEvent{
		UserID:   userID,
		IsActive: true,
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
		TargetLocation: models.Location{
			Latitude:  -6.200000,
			Longitude: 106.816666,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	// Mock active ride check - passenger has an active ride
	mockRepo.EXPECT().
		GetActiveRideByPassenger(gomock.Any(), userID).
		Return("active-ride-456", nil).
		Times(1)

	// AddAvailablePassenger should NOT be called since passenger has active ride

	// Act
	err := uc.HandleFinderEvent(context.Background(), event)

	// Assert
	assert.NoError(t, err) // Should not return error, just skip adding to pool
}

func TestHandleBeaconEvent_ActiveRideCheckError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	cfg := &models.Config{
		Match: models.MatchConfig{
			SearchRadiusKm: 5.0,
		},
	}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	userID := uuid.New().String()
	event := models.BeaconEvent{
		UserID:   userID,
		IsActive: true,
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	// Mock active ride check error - should continue with adding to pool
	mockRepo.EXPECT().
		GetActiveRideByDriver(gomock.Any(), userID).
		Return("", errors.New("redis connection error")).
		Times(1)

	// Should still try to add to pool on error to avoid blocking the system
	mockGW.EXPECT().
		AddAvailableDriver(gomock.Any(), userID, gomock.Any()).
		Return(nil)

	// Act
	err := uc.HandleBeaconEvent(context.Background(), event)

	// Assert
	assert.NoError(t, err)
}

func TestHandleFinderEvent_ActiveRideCheckError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	cfg := &models.Config{
		Match: models.MatchConfig{
			SearchRadiusKm: 5.0,
		},
	}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	userID := uuid.New().String()
	event := models.FinderEvent{
		UserID:   userID,
		IsActive: true,
		Location: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: time.Now(),
		},
		TargetLocation: models.Location{
			Latitude:  -6.200000,
			Longitude: 106.816666,
			Timestamp: time.Now(),
		},
		Timestamp: time.Now(),
	}

	// Mock active ride check error - should continue with adding to pool
	mockRepo.EXPECT().
		GetActiveRideByPassenger(gomock.Any(), userID).
		Return("", errors.New("redis connection error")).
		Times(1)

	// Should still try to add to pool on error to avoid blocking the system
	mockGW.EXPECT().
		AddAvailablePassenger(gomock.Any(), userID, gomock.Any()).
		Return(nil)

	mockGW.EXPECT().
		FindNearbyDrivers(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]*models.NearbyUser{}, nil)

	// Act
	err := uc.HandleFinderEvent(context.Background(), event)

	// Assert
	assert.NoError(t, err)
}

func TestConfirmMatchStatus_AcceptSuccess(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	cfg := &models.Config{
		Match: models.MatchConfig{
			SearchRadiusKm: 5.0,
		},
	}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	matchID := "match-123"
	driverID := uuid.New()
	passengerID := uuid.New()
	driverIDStr := driverID.String()
	passengerIDStr := passengerID.String()

	_ = models.MatchProposal{
		ID:          matchID,
		DriverID:    driverIDStr,
		PassengerID: passengerIDStr,
		MatchStatus: models.MatchStatusAccepted,
	}

	// The usecase first gets the pending match from Redis
	mockRepo.EXPECT().
		GetMatch(gomock.Any(), matchID).
		Return(&models.Match{
			ID:          converter.StrToUUID(matchID),
			DriverID:    driverID,
			PassengerID: passengerID,
			Status:      models.MatchStatusPending,
		}, nil)

	// Then it persists the match (note: matchID gets converted to UUID.Nil due to invalid format)
	mockRepo.EXPECT().
		ConfirmMatchByUser(gomock.Any(), "00000000-0000-0000-0000-000000000000", driverIDStr, true).
		Return(&models.Match{
			ID:          converter.StrToUUID(matchID),
			DriverID:    driverID,
			PassengerID: passengerID,
			Status:      models.MatchStatusAccepted,
		}, nil)

	// Mock ListMatchesByPassenger for async auto-rejection
	mockRepo.EXPECT().
		ListMatchesByPassenger(gomock.Any(), passengerID).
		Return([]*models.Match{}, nil).AnyTimes()

	// When match is accepted, it publishes the accepted event
	mockGW.EXPECT().
		PublishMatchAccepted(gomock.Any(), gomock.Any()).
		Return(nil)

	// The auto-rejection happens asynchronously, so we can't test it synchronously
	// Removed expectations for: ListMatchesByPassenger, RemoveAvailableDriver, RemoveAvailablePassenger

	// Act
	req := &models.MatchConfirmRequest{
		ID:     matchID,
		UserID: driverIDStr,
		Role:   "driver",
		Status: string(models.MatchStatusAccepted),
	}
	_, err := uc.ConfirmMatchStatus(context.Background(), req)

	// Assert
	assert.NoError(t, err)
}

func TestConfirmMatchStatus_RejectSuccess(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	cfg := &models.Config{
		Match: models.MatchConfig{
			SearchRadiusKm: 5.0,
		},
	}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	matchID := "match-123"
	driverID := uuid.New()
	passengerID := uuid.New()
	driverIDStr := driverID.String()
	_ = passengerID.String()

	// First GetMatch is called to retrieve the match
	mockRepo.EXPECT().
		GetMatch(gomock.Any(), matchID).
		Return(&models.Match{
			ID:          converter.StrToUUID(matchID),
			DriverID:    driverID,
			PassengerID: passengerID,
			Status:      models.MatchStatusPending,
		}, nil)

	// For rejection, the test should expect status update calls (matchID becomes UUID.Nil)
	mockRepo.EXPECT().
		UpdateMatchStatus(gomock.Any(), "00000000-0000-0000-0000-000000000000", models.MatchStatusRejected).
		Return(nil)

	// Then GetMatch is called again to get the updated match (also with UUID.Nil)
	mockRepo.EXPECT().
		GetMatch(gomock.Any(), "00000000-0000-0000-0000-000000000000").
		Return(&models.Match{
			ID:          converter.StrToUUID(matchID),
			DriverID:    driverID,
			PassengerID: passengerID,
			Status:      models.MatchStatusRejected,
		}, nil)

	// Act
	req := &models.MatchConfirmRequest{
		ID:     matchID,
		UserID: driverIDStr,
		Role:   "driver",
		Status: string(models.MatchStatusRejected),
	}
	_, err := uc.ConfirmMatchStatus(context.Background(), req)

	// Assert
	assert.NoError(t, err)
}

func TestConfirmMatchStatus_GetMatchError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)

	cfg := &models.Config{
		Match: models.MatchConfig{
			SearchRadiusKm: 5.0,
		},
	}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	matchID := "match-123"
	driverID := uuid.New().String()

	expectedError := errors.New("database error")

	// Set up expectations
	mockRepo.EXPECT().
		GetMatch(gomock.Any(), matchID).
		Return(nil, expectedError)

	// Act
	req := &models.MatchConfirmRequest{
		ID:     matchID,
		UserID: driverID,
		Role:   "driver",
		Status: string(models.MatchStatusAccepted),
	}
	_, err := uc.ConfirmMatchStatus(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "match not found in database")
}

func TestCreateMatch_DatabaseError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)

	cfg := &models.Config{
		Match: models.MatchConfig{
			SearchRadiusKm: 5.0,
		},
	}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

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

// Test HasActiveRide functionality
func TestHasActiveRide_DriverHasActiveRide(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	cfg := &models.Config{}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	userID := "driver-123"
	rideID := "ride-456"

	// Mock repository to return active ride
	mockRepo.EXPECT().
		GetActiveRideByDriver(gomock.Any(), userID).
		Return(rideID, nil).
		Times(1)

	// Act
	hasActiveRide, err := uc.HasActiveRide(context.Background(), userID, true) // true = isDriver

	// Assert
	assert.NoError(t, err)
	assert.True(t, hasActiveRide)
}

func TestHasActiveRide_DriverNoActiveRide(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	cfg := &models.Config{}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	userID := "driver-123"

	// Mock repository to return no active ride
	mockRepo.EXPECT().
		GetActiveRideByDriver(gomock.Any(), userID).
		Return("", nil).
		Times(1)

	// Act
	hasActiveRide, err := uc.HasActiveRide(context.Background(), userID, true) // true = isDriver

	// Assert
	assert.NoError(t, err)
	assert.False(t, hasActiveRide)
}

func TestHasActiveRide_PassengerHasActiveRide(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	cfg := &models.Config{}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	userID := "passenger-123"
	rideID := "ride-456"

	// Mock repository to return active ride
	mockRepo.EXPECT().
		GetActiveRideByPassenger(gomock.Any(), userID).
		Return(rideID, nil).
		Times(1)

	// Act
	hasActiveRide, err := uc.HasActiveRide(context.Background(), userID, false) // false = isPassenger

	// Assert
	assert.NoError(t, err)
	assert.True(t, hasActiveRide)
}

func TestHasActiveRide_PassengerNoActiveRide(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	cfg := &models.Config{}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	userID := "passenger-123"

	// Mock repository to return no active ride
	mockRepo.EXPECT().
		GetActiveRideByPassenger(gomock.Any(), userID).
		Return("", nil).
		Times(1)

	// Act
	hasActiveRide, err := uc.HasActiveRide(context.Background(), userID, false) // false = isPassenger

	// Assert
	assert.NoError(t, err)
	assert.False(t, hasActiveRide)
}

func TestSetActiveRide_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	cfg := &models.Config{}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	rideID := "ride-123"
	driverID := "driver-456"
	passengerID := "passenger-789"

	// Mock repository calls
	mockRepo.EXPECT().
		SetActiveRide(gomock.Any(), rideID, driverID, passengerID).
		Return(nil).
		Times(1)

	// Act
	err := uc.SetActiveRide(context.Background(), rideID, driverID, passengerID)

	// Assert
	assert.NoError(t, err)
}

func TestRemoveActiveRide_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	cfg := &models.Config{}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	driverID := "driver-456"
	passengerID := "passenger-789"

	// Mock repository calls
	mockRepo.EXPECT().
		RemoveActiveRide(gomock.Any(), driverID, passengerID).
		Return(nil).
		Times(1)

	// Act
	err := uc.RemoveActiveRide(context.Background(), driverID, passengerID)

	// Assert
	assert.NoError(t, err)
}
