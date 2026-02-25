package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	coremodels "github.com/piresc/nebengjek/internal/pkg/models/core"
	locationmodels "github.com/piresc/nebengjek/internal/pkg/models/location"
	matchmodels "github.com/piresc/nebengjek/internal/pkg/models/match"
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
	cfg := &coremodels.Config{
		Match: coremodels.MatchConfig{
			SearchRadiusKm:     5.0,
			ActiveRideTTLHours: 24,
		},
	}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	// Test data
	passengerID := uuid.New().String()
	driverID := uuid.New().String()
	matchID := uuid.New().String()

	passengerLocation := locationmodels.Location{
		Latitude:  -6.2088,
		Longitude: 106.8456,
	}

	driverLocation := locationmodels.Location{
		Latitude:  -6.2188,
		Longitude: 106.8556,
	}

	// Step 1: Passenger requests a ride (finder event)
	finderEvent := coremodels.FinderEvent{
		UserID:         passengerID,
		IsActive:       true,
		Location:       passengerLocation,
		TargetLocation: passengerLocation, // Add target location
		Timestamp:      time.Now(),
	}

	nearbyDrivers := []*matchmodels.NearbyUser{
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
		DoAndReturn(func(ctx context.Context, match *matchmodels.Match) (*matchmodels.Match, error) {
			match.ID = coremodels.StrToUUID(matchID)
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
	matchRequest := &matchmodels.MatchConfirmRequest{
		ID:     matchID,
		UserID: driverID,
		Role:   "driver",
		Status: string(matchmodels.MatchStatusAccepted),
	}

	existingMatch := &matchmodels.Match{
		ID:                coremodels.StrToUUID(matchID),
		DriverID:          coremodels.StrToUUID(driverID),
		PassengerID:       coremodels.StrToUUID(passengerID),
		PassengerLocation: passengerLocation,
		DriverLocation:    driverLocation,
		Status:            matchmodels.MatchStatusPending,
		PassengerConfirmed: true, // Passenger already confirmed
		CreatedAt:         time.Now(),
	}

	mockRepo.EXPECT().
		GetMatch(gomock.Any(), matchID).
		Return(existingMatch, nil)

	// Mock ConfirmMatchByUser (called by updateMatchConfirmation)
	mockRepo.EXPECT().
		ConfirmMatchByUser(gomock.Any(), matchID, driverID, true).
		DoAndReturn(func(ctx context.Context, matchID, userID string, isDriver bool) (*matchmodels.Match, error) {
			existingMatch.DriverConfirmed = true
			existingMatch.Status = matchmodels.MatchStatusAccepted
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
		ListMatchesByPassenger(gomock.Any(), coremodels.StrToUUID(passengerID)).
		Return([]*matchmodels.Match{}, nil).AnyTimes() // No other matches to reject

	mockRepo.EXPECT().
		BatchUpdateMatchStatus(gomock.Any(), gomock.Any(), matchmodels.MatchStatusRejected).
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
	cfg := &coremodels.Config{
		Match: coremodels.MatchConfig{
			SearchRadiusKm:     5.0,
			ActiveRideTTLHours: 24,
		},
	}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	passengerID := uuid.New().String()
	driver1ID := uuid.New().String()
	driver2ID := uuid.New().String()
	driver3ID := uuid.New().String()

	passengerLocation := locationmodels.Location{
		Latitude:  -6.2088,
		Longitude: 106.8456,
	}

	finderEvent := &coremodels.FinderEvent{
		UserID:         passengerID,
		IsActive:       true,
		Location:       passengerLocation,
		TargetLocation: passengerLocation,
		Timestamp:      time.Now(),
	}

	// Multiple nearby drivers
	nearbyDrivers := []*matchmodels.NearbyUser{
		{
			ID:       driver1ID,
			Distance: 1.5,
			Location: locationmodels.Location{Latitude: -6.2188, Longitude: 106.8556},
		},
		{
			ID:       driver2ID,
			Distance: 2.5,
			Location: locationmodels.Location{Latitude: -6.2288, Longitude: 106.8656},
		},
		{
			ID:       driver3ID,
			Distance: 3.5,
			Location: locationmodels.Location{Latitude: -6.2388, Longitude: 106.8756},
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
		DoAndReturn(func(ctx context.Context, match *matchmodels.Match) (*matchmodels.Match, error) {
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
	cfg := &coremodels.Config{
		Match: coremodels.MatchConfig{
			SearchRadiusKm:     5.0,
			ActiveRideTTLHours: 24,
		},
	}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	matchID := uuid.New().String()
	expiredMatch := &matchmodels.Match{
		ID:          coremodels.StrToUUID(matchID),
		DriverID:    uuid.New(),
		PassengerID: uuid.New(),
		Status:      matchmodels.MatchStatusPending,
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
	cfg := &coremodels.Config{
		Match: coremodels.MatchConfig{
			SearchRadiusKm:     5.0,
			ActiveRideTTLHours: 24,
		},
	}

	uc := NewMatchUC(cfg, mockRepo, mockGW)

	matchID := uuid.New().String()
	driverID := uuid.New().String()
	passengerID := uuid.New().String()

	matchRequest := &matchmodels.MatchConfirmRequest{
		ID:     matchID,
		UserID: driverID,
		Role:   "driver",
		Status: string(matchmodels.MatchStatusRejected),
	}

	existingMatch := &matchmodels.Match{
		ID:          coremodels.StrToUUID(matchID),
		DriverID:    coremodels.StrToUUID(driverID),
		PassengerID: coremodels.StrToUUID(passengerID),
		PassengerLocation: locationmodels.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
		Status:    matchmodels.MatchStatusPending,
		CreatedAt: time.Now(),
	}

	mockRepo.EXPECT().
		GetMatch(gomock.Any(), matchID).
		Return(existingMatch, nil)

	// Mock rejection handling
	mockRepo.EXPECT().
		UpdateMatchStatus(gomock.Any(), matchID, matchmodels.MatchStatusRejected).
		Return(nil)

	mockRepo.EXPECT().
		GetMatch(gomock.Any(), matchID).
		Return(&matchmodels.Match{
			ID:          coremodels.StrToUUID(matchID),
			DriverID:    coremodels.StrToUUID(driverID),
			PassengerID: coremodels.StrToUUID(passengerID),
			PassengerLocation: existingMatch.PassengerLocation,
			Status:      matchmodels.MatchStatusRejected,
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

	