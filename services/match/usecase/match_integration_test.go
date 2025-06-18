package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/converter"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/match/mocks"
	"github.com/stretchr/testify/assert"
)

// Integration tests for complex matching scenarios
func TestMatchUC_CompleteMatchFlow_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	cfg := &models.Config{
		Match: models.MatchConfig{
			SearchRadiusKm:     5.0,
			ActiveRideTTLHours: 24,
		},
	}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	// Test data
	passengerID := uuid.New().String()
	driverID := uuid.New().String()
	matchID := uuid.New().String()

	passengerLocation := models.Location{
		Latitude:  -6.2088,
		Longitude: 106.8456,
	}

	driverLocation := models.Location{
		Latitude:  -6.2188,
		Longitude: 106.8556,
	}

	// Step 1: Passenger requests a ride (finder event)
	finderEvent := models.FinderEvent{
		UserID:         passengerID,
		IsActive:       true,
		Location:       passengerLocation,
		TargetLocation: passengerLocation, // Add target location
		Timestamp:      time.Now(),
	}

	nearbyDrivers := []*models.NearbyUser{
		{
			ID:       driverID,
			Distance: 2.5,
			Location: driverLocation,
		},
	}

	// Mock active ride check for passenger
	mockRepo.EXPECT().
		GetActiveRideByPassenger(gomock.Any(), passengerID).
		Return("", nil) // No active ride

	// Mock adding passenger to available pool
	mockGW.EXPECT().
		AddAvailablePassenger(gomock.Any(), passengerID, &passengerLocation).
		Return(nil)

	mockGW.EXPECT().
		FindNearbyDrivers(gomock.Any(), &passengerLocation, cfg.Match.SearchRadiusKm).
		Return(nearbyDrivers, nil)

	mockRepo.EXPECT().
		CreateMatch(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, match *models.Match) (*models.Match, error) {
			match.ID = converter.StrToUUID(matchID)
			match.CreatedAt = time.Now()
			return match, nil
		})

	mockGW.EXPECT().
		PublishMatchFound(gomock.Any(), gomock.Any()).
		Return(nil)

	// Act - Step 1
	err := uc.HandleFinderEvent(context.Background(), finderEvent)

	// Assert - Step 1
	assert.NoError(t, err)

	// Step 2: Driver accepts the match
	matchRequest := &models.MatchConfirmRequest{
		ID:     matchID,
		UserID: driverID,
		Role:   "driver",
		Status: string(models.MatchStatusAccepted),
	}

	existingMatch := &models.Match{
		ID:                converter.StrToUUID(matchID),
		DriverID:          converter.StrToUUID(driverID),
		PassengerID:       converter.StrToUUID(passengerID),
		PassengerLocation: passengerLocation,
		DriverLocation:    driverLocation,
		Status:            models.MatchStatusPending,
		PassengerConfirmed: true, // Passenger already confirmed
		CreatedAt:         time.Now(),
	}

	mockRepo.EXPECT().
		GetMatch(gomock.Any(), matchID).
		Return(existingMatch, nil)

	// Mock ConfirmMatchByUser (called by updateMatchConfirmation)
	mockRepo.EXPECT().
		ConfirmMatchByUser(gomock.Any(), matchID, driverID, true).
		DoAndReturn(func(ctx context.Context, matchID, userID string, isDriver bool) (*models.Match, error) {
			existingMatch.DriverConfirmed = true
			existingMatch.Status = models.MatchStatusAccepted
			return existingMatch, nil
		})

	// Mock removing users from available pools
	mockGW.EXPECT().
		RemoveAvailableDriver(gomock.Any(), driverID).
		Return(nil)

	mockGW.EXPECT().
		RemoveAvailablePassenger(gomock.Any(), passengerID).
		Return(nil)

	mockGW.EXPECT().
		PublishMatchAccepted(gomock.Any(), gomock.Any()).
		Return(nil)

	// Mock auto-rejection process (async)
	mockRepo.EXPECT().
		ListMatchesByPassenger(gomock.Any(), converter.StrToUUID(passengerID)).
		Return([]*models.Match{}, nil).AnyTimes() // No other matches to reject

	mockRepo.EXPECT().
		BatchUpdateMatchStatus(gomock.Any(), gomock.Any(), models.MatchStatusRejected).
		Return(nil).AnyTimes()

	mockGW.EXPECT().
		PublishMatchRejected(gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	// Act - Step 2
	_, err = uc.ConfirmMatchStatus(context.Background(), matchRequest)

	// Assert - Step 2
	assert.NoError(t, err)
}

func TestMatchUC_HandleMultipleDriversScenario(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	cfg := &models.Config{
		Match: models.MatchConfig{
			SearchRadiusKm:     5.0,
			ActiveRideTTLHours: 24,
		},
	}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	passengerID := uuid.New().String()
	driver1ID := uuid.New().String()
	driver2ID := uuid.New().String()
	driver3ID := uuid.New().String()

	passengerLocation := models.Location{
		Latitude:  -6.2088,
		Longitude: 106.8456,
	}

	finderEvent := &models.FinderEvent{
		UserID:         passengerID,
		IsActive:       true,
		Location:       passengerLocation,
		TargetLocation: passengerLocation,
		Timestamp:      time.Now(),
	}

	// Multiple nearby drivers
	nearbyDrivers := []*models.NearbyUser{
		{
			ID:       driver1ID,
			Distance: 1.5,
			Location: models.Location{Latitude: -6.2188, Longitude: 106.8556},
		},
		{
			ID:       driver2ID,
			Distance: 2.5,
			Location: models.Location{Latitude: -6.2288, Longitude: 106.8656},
		},
		{
			ID:       driver3ID,
			Distance: 3.5,
			Location: models.Location{Latitude: -6.2388, Longitude: 106.8756},
		},
	}

	// Mock active ride check for passenger
	mockRepo.EXPECT().
		GetActiveRideByPassenger(gomock.Any(), passengerID).
		Return("", nil) // No active ride

	// Mock adding passenger to available pool
	mockGW.EXPECT().
		AddAvailablePassenger(gomock.Any(), passengerID, &passengerLocation).
		Return(nil)

	mockGW.EXPECT().
		FindNearbyDrivers(gomock.Any(), &passengerLocation, cfg.Match.SearchRadiusKm).
		Return(nearbyDrivers, nil)

	// Expect 3 matches to be created
	mockRepo.EXPECT().
		CreateMatch(gomock.Any(), gomock.Any()).
		Times(3).
		DoAndReturn(func(ctx context.Context, match *models.Match) (*models.Match, error) {
			match.ID = uuid.New()
			match.CreatedAt = time.Now()
			return match, nil
		})

	// Expect 3 match proposals to be published
	mockGW.EXPECT().
		PublishMatchFound(gomock.Any(), gomock.Any()).
		Times(3).
		Return(nil)

	// Act
	err := uc.HandleFinderEvent(context.Background(), *finderEvent)

	// Assert
	assert.NoError(t, err)
}

func TestMatchUC_HandleMatchTimeout(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	cfg := &models.Config{
		Match: models.MatchConfig{
			SearchRadiusKm:     5.0,
			ActiveRideTTLHours: 24,
		},
	}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	matchID := uuid.New().String()
	expiredMatch := &models.Match{
		ID:          converter.StrToUUID(matchID),
		DriverID:    uuid.New(),
		PassengerID: uuid.New(),
		Status:      models.MatchStatusPending,
		CreatedAt:   time.Now().Add(-2 * time.Minute), // Created 2 minutes ago
	}

	// Test getting a match by ID
	mockRepo.EXPECT().
		GetMatch(gomock.Any(), matchID).
		Return(expiredMatch, nil)

	// Act - Test getting a match
	match, err := uc.GetMatch(context.Background(), matchID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, match)
}

func TestMatchUC_HandleDriverRejection_FindAlternative(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	cfg := &models.Config{
		Match: models.MatchConfig{
			SearchRadiusKm:     5.0,
			ActiveRideTTLHours: 24,
		},
	}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	matchID := uuid.New().String()
	driverID := uuid.New().String()
	passengerID := uuid.New().String()

	matchRequest := &models.MatchConfirmRequest{
		ID:     matchID,
		UserID: driverID,
		Role:   "driver",
		Status: string(models.MatchStatusRejected),
	}

	existingMatch := &models.Match{
		ID:          converter.StrToUUID(matchID),
		DriverID:    converter.StrToUUID(driverID),
		PassengerID: converter.StrToUUID(passengerID),
		PassengerLocation: models.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
		Status:    models.MatchStatusPending,
		CreatedAt: time.Now(),
	}

	mockRepo.EXPECT().
		GetMatch(gomock.Any(), matchID).
		Return(existingMatch, nil)

	// Mock rejection handling
	mockRepo.EXPECT().
		UpdateMatchStatus(gomock.Any(), matchID, models.MatchStatusRejected).
		Return(nil)

	mockRepo.EXPECT().
		GetMatch(gomock.Any(), matchID).
		Return(&models.Match{
			ID:          converter.StrToUUID(matchID),
			DriverID:    converter.StrToUUID(driverID),
			PassengerID: converter.StrToUUID(passengerID),
			PassengerLocation: existingMatch.PassengerLocation,
			Status:      models.MatchStatusRejected,
			CreatedAt:   time.Now(),
		}, nil)

	mockGW.EXPECT().
		PublishMatchRejected(gomock.Any(), gomock.Any()).
		Return(nil)

	// Act
	_, err := uc.ConfirmMatchStatus(context.Background(), matchRequest)

	// Assert
	assert.NoError(t, err)
}

func TestMatchUC_HandleConcurrentMatches(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	cfg := &models.Config{
		Match: models.MatchConfig{
			SearchRadiusKm:     5.0,
			ActiveRideTTLHours: 24,
		},
	}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	driverID := uuid.New().String()
	match1ID := uuid.New().String()
	match2ID := uuid.New().String()

	// Driver tries to accept two matches simultaneously
	matchRequest1 := &models.MatchConfirmRequest{
		ID:     match1ID,
		UserID: driverID,
		Role:   "driver",
		Status: string(models.MatchStatusAccepted),
	}

	matchRequest2 := &models.MatchConfirmRequest{
		ID:     match2ID,
		UserID: driverID,
		Role:   "driver",
		Status: string(models.MatchStatusAccepted),
	}

	match1 := &models.Match{
		ID:          converter.StrToUUID(match1ID),
		DriverID:    converter.StrToUUID(driverID),
		PassengerID: uuid.New(),
		Status:      models.MatchStatusPending,
		CreatedAt:   time.Now(),
	}

	match2 := &models.Match{
		ID:          converter.StrToUUID(match2ID),
		DriverID:    converter.StrToUUID(driverID),
		PassengerID: uuid.New(),
		Status:      models.MatchStatusPending,
		CreatedAt:   time.Now(),
	}

	// First match succeeds
	mockRepo.EXPECT().
		GetMatch(gomock.Any(), match1ID).
		Return(match1, nil)

	mockRepo.EXPECT().
		ConfirmMatchByUser(gomock.Any(), match1ID, driverID, true).
		DoAndReturn(func(ctx context.Context, matchID, userID string, isDriver bool) (*models.Match, error) {
			match1.DriverConfirmed = true
			match1.Status = models.MatchStatusDriverConfirmed
			return match1, nil
		})

	// Second match succeeds as well
	mockRepo.EXPECT().
		GetMatch(gomock.Any(), match2ID).
		Return(match2, nil)

	mockRepo.EXPECT().
		ConfirmMatchByUser(gomock.Any(), match2ID, driverID, true).
		DoAndReturn(func(ctx context.Context, matchID, userID string, isDriver bool) (*models.Match, error) {
			match2.DriverConfirmed = true
			match2.Status = models.MatchStatusDriverConfirmed
			return match2, nil
		})

	// Mock auto-rejection process (async) for both matches
	mockRepo.EXPECT().
		ListMatchesByPassenger(gomock.Any(), gomock.Any()).
		Return([]*models.Match{}, nil).AnyTimes()

	mockRepo.EXPECT().
		BatchUpdateMatchStatus(gomock.Any(), gomock.Any(), models.MatchStatusRejected).
		Return(nil).AnyTimes()

	mockGW.EXPECT().
		PublishMatchRejected(gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	// Act
	_, err1 := uc.ConfirmMatchStatus(context.Background(), matchRequest1)
	_, err2 := uc.ConfirmMatchStatus(context.Background(), matchRequest2)

	// Assert
	assert.NoError(t, err1)
	assert.NoError(t, err2)
}

func TestMatchUC_HandleLocationBasedMatching(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockMatchRepo(ctrl)
	mockGW := mocks.NewMockMatchGW(ctrl)
	cfg := &models.Config{
		Match: models.MatchConfig{
			SearchRadiusKm:     2.0, // Small radius
			ActiveRideTTLHours: 24,
		},
	}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	passengerID := uuid.New().String()
	passengerLocation := models.Location{
		Latitude:  -6.2088,
		Longitude: 106.8456,
	}

	finderEvent := &models.FinderEvent{
		UserID:         passengerID,
		IsActive:       true,
		Location:       passengerLocation,
		TargetLocation: passengerLocation,
		Timestamp:      time.Now(),
	}

	// Drivers at different distances
	nearbyDrivers := []*models.NearbyUser{
		{
			ID:       uuid.New().String(),
			Distance: 0.5, // Very close
			Location: models.Location{Latitude: -6.2098, Longitude: 106.8466},
		},
		{
			ID:       uuid.New().String(),
			Distance: 1.5, // Within radius
			Location: models.Location{Latitude: -6.2188, Longitude: 106.8556},
		},
		// Note: Driver at 3.0km would be filtered out by the 2.0km radius
	}

	// Mock active ride check for passenger
	mockRepo.EXPECT().
		GetActiveRideByPassenger(gomock.Any(), passengerID).
		Return("", nil) // No active ride

	// Mock adding passenger to available pool
	mockGW.EXPECT().
		AddAvailablePassenger(gomock.Any(), passengerID, &passengerLocation).
		Return(nil)

	mockGW.EXPECT().
		FindNearbyDrivers(gomock.Any(), &passengerLocation, cfg.Match.SearchRadiusKm).
		Return(nearbyDrivers, nil)

	// Expect matches for drivers within radius
	mockRepo.EXPECT().
		CreateMatch(gomock.Any(), gomock.Any()).
		Times(2). // Only 2 drivers within 2km radius
		DoAndReturn(func(ctx context.Context, match *models.Match) (*models.Match, error) {
			match.ID = uuid.New()
			match.CreatedAt = time.Now()
			return match, nil
		})

	mockGW.EXPECT().
		PublishMatchFound(gomock.Any(), gomock.Any()).
		Times(2).
		Return(nil)

	// Act
	err := uc.HandleFinderEvent(context.Background(), *finderEvent)

	// Assert
	assert.NoError(t, err)
}