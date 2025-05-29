package usecase_test

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/converter"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/match/mocks"
	"github.com/piresc/nebengjek/services/match/usecase"
	"github.com/stretchr/testify/assert"
)

func TestTwoWayConfirmation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	matchRepo := mocks.NewMockMatchRepo(ctrl)
	matchGW := mocks.NewMockMatchGateway(ctrl)
	uc := usecase.NewMatchUC(matchRepo, matchGW)

	// Setup test data
	driverID := uuid.New()
	passengerID := uuid.New()
	matchID := uuid.New().String()
	driverLocation := models.Location{Latitude: 1.0, Longitude: 1.0}
	passengerLocation := models.Location{Latitude: 1.1, Longitude: 1.1}

	// Create a match with driver and passenger
	match := &models.Match{
		ID:                uuid.MustParse(matchID),
		DriverID:          driverID,
		PassengerID:       passengerID,
		DriverLocation:    driverLocation,
		PassengerLocation: passengerLocation,
		Status:            models.MatchStatusPending,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Test case 1: Redis-only match is confirmed by driver first
	t.Run("driver confirms redis match first", func(t *testing.T) {
		// When match is not found in DB, it should look in Redis
		matchRepo.EXPECT().GetMatch(gomock.Any(), matchID).Return(nil, assert.AnError)
		matchRepo.EXPECT().GetPendingMatchByID(gomock.Any(), matchID).Return(match, nil)

		// It should persist the Redis match to the database
		matchRepo.EXPECT().CreateMatch(gomock.Any(), match).Return(match, nil)

		// Store ID mapping for future reference
		matchRepo.EXPECT().StoreIDMapping(gomock.Any(), gomock.Any(), matchID, gomock.Any()).Return(nil)

		// Driver confirms the match
		matchWithDriverConfirmed := *match
		matchWithDriverConfirmed.DriverConfirmed = true
		matchWithDriverConfirmed.Status = models.MatchStatusDriverConfirmed

		matchRepo.EXPECT().ConfirmMatchByUser(
			gomock.Any(),
			matchID,
			converter.UUIDToStr(driverID),
			true,
		).Return(&matchWithDriverConfirmed, nil)

		// Driver confirmation happens
		result, err := uc.ConfirmMatchStatus(
			matchID,
			converter.UUIDToStr(driverID),
			true,
			models.MatchStatusAccepted,
		)

		assert.NoError(t, err)
		assert.Equal(t, models.MatchStatusDriverConfirmed, result.MatchStatus)
	})

	// Test case 2: Passenger confirms after driver, completing the match
	t.Run("passenger confirms after driver", func(t *testing.T) {
		// Match is now in the database and driver-confirmed
		matchWithDriverConfirmed := *match
		matchWithDriverConfirmed.DriverConfirmed = true
		matchWithDriverConfirmed.Status = models.MatchStatusDriverConfirmed

		matchRepo.EXPECT().GetMatch(gomock.Any(), matchID).Return(&matchWithDriverConfirmed, nil)

		// Both parties confirmed
		matchFullyConfirmed := matchWithDriverConfirmed
		matchFullyConfirmed.PassengerConfirmed = true
		matchFullyConfirmed.Status = models.MatchStatusAccepted

		matchRepo.EXPECT().ConfirmMatchByUser(
			gomock.Any(),
			matchID,
			converter.UUIDToStr(passengerID),
			false,
		).Return(&matchFullyConfirmed, nil)

		// Passenger confirmation happens
		result, err := uc.ConfirmMatchStatus(
			matchID,
			converter.UUIDToStr(passengerID),
			false,
			models.MatchStatusAccepted,
		)

		assert.NoError(t, err)
		assert.Equal(t, models.MatchStatusAccepted, result.MatchStatus)
	})

	// Test case 3: Confirm with ID mapping
	t.Run("confirm with ID mapping", func(t *testing.T) {
		// Original match ID not found in DB
		originalID := "original-id"
		mappedID := matchID

		matchRepo.EXPECT().GetMatch(gomock.Any(), originalID).Return(nil, assert.AnError)
		matchRepo.EXPECT().GetIDMapping(gomock.Any(), originalID).Return(mappedID, nil)
		matchRepo.EXPECT().GetMatch(gomock.Any(), mappedID).Return(match, nil)

		// Match is found with the mapped ID and updated
		matchRepo.EXPECT().ConfirmMatchByUser(
			gomock.Any(),
			mappedID,
			converter.UUIDToStr(driverID),
			true,
		).Return(match, nil)

		result, err := uc.ConfirmMatchStatus(
			originalID,
			converter.UUIDToStr(driverID),
			true,
			models.MatchStatusAccepted,
		)

		assert.NoError(t, err)
		assert.Equal(t, converter.UUIDToStr(match.ID), result.ID)
	})
}
