package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/rides/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRide_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)

	cfg := &models.Config{}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	driverID := uuid.New().String()
	passengerID := uuid.New().String()

	matchProposal := models.MatchProposal{
		ID:          "match-123",
		DriverID:    driverID,
		PassengerID: passengerID,
		UserLocation: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		DriverLocation: models.Location{
			Latitude:  -6.185392,
			Longitude: 106.837153,
		},
		MatchStatus: models.MatchStatusAccepted,
	}

	// Set up expectations
	mockRepo.EXPECT().
		CreateRide(gomock.Any()).
		DoAndReturn(func(ride *models.Ride) (*models.Ride, error) {
			assert.Equal(t, uuid.MustParse(driverID), ride.DriverID)
			assert.Equal(t, uuid.MustParse(passengerID), ride.CustomerID)

			// Add ride ID to simulate DB creation
			ride.RideID = uuid.New()
			return ride, nil
		})

	mockGW.EXPECT().
		PublishRideStarted(gomock.Any(), gomock.Any()).
		Return(nil)

	// Act
	err = uc.CreateRide(matchProposal)

	// Assert
	assert.NoError(t, err)
}

func TestCreateRide_RepositoryError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)

	cfg := &models.Config{}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	driverID := uuid.New().String()
	passengerID := uuid.New().String()

	matchProposal := models.MatchProposal{
		ID:          "match-123",
		DriverID:    driverID,
		PassengerID: passengerID,
		MatchStatus: models.MatchStatusAccepted,
	}

	expectedError := errors.New("database error")

	// Set up expectations
	mockRepo.EXPECT().
		CreateRide(gomock.Any()).
		Return(nil, expectedError)

	// Act
	err = uc.CreateRide(matchProposal)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestCreateRide_PublishError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)

	cfg := &models.Config{}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	driverID := uuid.New().String()
	passengerID := uuid.New().String()

	matchProposal := models.MatchProposal{
		ID:          "match-123",
		DriverID:    driverID,
		PassengerID: passengerID,
		MatchStatus: models.MatchStatusAccepted,
	}

	expectedError := errors.New("publish error")

	// Set up expectations
	mockRepo.EXPECT().
		CreateRide(gomock.Any()).
		DoAndReturn(func(ride *models.Ride) (*models.Ride, error) {
			ride.RideID = uuid.New()
			return ride, nil
		})

	mockGW.EXPECT().
		PublishRideStarted(gomock.Any(), gomock.Any()).
		Return(expectedError)

	// Act
	err = uc.CreateRide(matchProposal)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestProcessBillingUpdate_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)

	cfg := &models.Config{}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)

	entry := &models.BillingLedger{
		RideID:   rideUUID,
		Distance: 2.5,
		Cost:     7500,
	}

	ride := &models.Ride{
		RideID:    rideUUID,
		Status:    models.RideStatusOngoing,
		TotalCost: 10000,
	}

	// Set up expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID).
		Return(ride, nil)

	mockRepo.EXPECT().
		AddBillingEntry(gomock.Any(), entry).
		Return(nil)

	mockRepo.EXPECT().
		UpdateTotalCost(gomock.Any(), rideID, entry.Cost).
		Return(nil)

	// Act
	err = uc.ProcessBillingUpdate(rideID, entry)

	// Assert
	assert.NoError(t, err)
}

func TestProcessBillingUpdate_GetRideError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)

	cfg := &models.Config{}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)

	entry := &models.BillingLedger{
		RideID:   rideUUID,
		Distance: 2.5,
		Cost:     7500,
	}

	expectedError := errors.New("database error")

	// Set up expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID).
		Return(nil, expectedError)

	// Act
	err = uc.ProcessBillingUpdate(rideID, entry)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get ride")
}

func TestProcessBillingUpdate_InvalidRideStatus(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)

	cfg := &models.Config{}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)

	entry := &models.BillingLedger{
		RideID:   rideUUID,
		Distance: 2.5,
		Cost:     7500,
	}

	ride := &models.Ride{
		RideID:    rideUUID,
		Status:    models.RideStatusCompleted, // Ride is already completed
		TotalCost: 10000,
	}

	// Set up expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID).
		Return(ride, nil)

	// Act
	err = uc.ProcessBillingUpdate(rideID, entry)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot update billing for non-active ride")
}

func TestCompleteRide_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)

	cfg := &models.Config{}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)
	adjustmentFactor := 0.8 // 80% of original price

	totalCost := 10000

	ride := &models.Ride{
		RideID:    rideUUID,
		Status:    models.RideStatusOngoing,
		TotalCost: totalCost,
	}

	// Expected values
	adjustedCost := int(float64(totalCost) * adjustmentFactor)
	adminFee := int(float64(adjustedCost) * 0.05)
	driverPayout := adjustedCost - adminFee

	// Set up expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID).
		Return(ride, nil)

	mockRepo.EXPECT().
		GetBillingLedgerSum(gomock.Any(), rideID).
		Return(totalCost, nil)

	mockRepo.EXPECT().
		CreatePayment(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, payment *models.Payment) error {
			assert.Equal(t, rideUUID, payment.RideID)
			assert.Equal(t, adjustedCost, payment.AdjustedCost)
			assert.Equal(t, adminFee, payment.AdminFee)
			assert.Equal(t, driverPayout, payment.DriverPayout)
			return nil
		})

	mockRepo.EXPECT().
		CompleteRide(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, updatedRide *models.Ride) error {
			assert.Equal(t, rideUUID, updatedRide.RideID)
			assert.Equal(t, models.RideStatusCompleted, updatedRide.Status)
			return nil
		})

	mockGW.EXPECT().
		PublishRideCompleted(gomock.Any(), gomock.Any()).
		Return(nil)

	// Act
	payment, err := uc.CompleteRide(rideID, adjustmentFactor)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, payment)
	assert.Equal(t, rideUUID, payment.RideID)
	assert.Equal(t, adjustedCost, payment.AdjustedCost)
	assert.Equal(t, adminFee, payment.AdminFee)
	assert.Equal(t, driverPayout, payment.DriverPayout)
}

func TestCompleteRide_InvalidAdjustmentFactor(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)

	cfg := &models.Config{}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)
	invalidAdjustmentFactor := 1.5 // Should be reset to 1.0

	totalCost := 10000

	ride := &models.Ride{
		RideID:    rideUUID,
		Status:    models.RideStatusOngoing,
		TotalCost: totalCost,
	}

	// Expected values with adjustment factor of 1.0
	adjustedCost := totalCost
	adminFee := int(float64(adjustedCost) * 0.05)
	driverPayout := adjustedCost - adminFee

	// Set up expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID).
		Return(ride, nil)

	mockRepo.EXPECT().
		GetBillingLedgerSum(gomock.Any(), rideID).
		Return(totalCost, nil)

	mockRepo.EXPECT().
		CreatePayment(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, payment *models.Payment) error {
			assert.Equal(t, rideUUID, payment.RideID)
			assert.Equal(t, adjustedCost, payment.AdjustedCost)
			assert.Equal(t, adminFee, payment.AdminFee)
			assert.Equal(t, driverPayout, payment.DriverPayout)
			return nil
		})

	mockRepo.EXPECT().
		CompleteRide(gomock.Any(), gomock.Any()).
		Return(nil)

	mockGW.EXPECT().
		PublishRideCompleted(gomock.Any(), gomock.Any()).
		Return(nil)

	// Act
	payment, err := uc.CompleteRide(rideID, invalidAdjustmentFactor)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, payment)
	assert.Equal(t, totalCost, payment.AdjustedCost) // Should be reset to full cost
}

func TestCompleteRide_PublishError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)

	cfg := &models.Config{}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)
	adjustmentFactor := 0.8

	totalCost := 10000

	ride := &models.Ride{
		RideID:    rideUUID,
		Status:    models.RideStatusOngoing,
		TotalCost: totalCost,
	}

	expectedError := errors.New("publish error")

	// Set up expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID).
		Return(ride, nil)

	mockRepo.EXPECT().
		GetBillingLedgerSum(gomock.Any(), rideID).
		Return(totalCost, nil)

	mockRepo.EXPECT().
		CreatePayment(gomock.Any(), gomock.Any()).
		Return(nil)

	mockRepo.EXPECT().
		CompleteRide(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, updatedRide *models.Ride) error {
			updatedRide.Status = models.RideStatusCompleted
			return nil
		})

	mockGW.EXPECT().
		PublishRideCompleted(gomock.Any(), gomock.Any()).
		Return(expectedError)

	// Act
	payment, err := uc.CompleteRide(rideID, adjustmentFactor)

	// Assert
	assert.NoError(t, err) // Should not fail the operation
	assert.NotNil(t, payment)
}
