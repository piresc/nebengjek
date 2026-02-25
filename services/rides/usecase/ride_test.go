package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models/core"
	ridemodels "github.com/piresc/nebengjek/internal/pkg/models/ride"
	matchmodels "github.com/piresc/nebengjek/internal/pkg/models/match"
	locationmodels "github.com/piresc/nebengjek/internal/pkg/models/location"
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

	cfg := &core.Config{
		Rides: core.RidesConfig{
			MaxPickupDistanceM: 100.0, // Set pickup distance for test
		},
	}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	driverID := uuid.New().String()
	passengerID := uuid.New().String()
	matchID := uuid.New().String()

	matchProposal := matchmodels.MatchProposal{
		ID:          matchID,
		DriverID:    driverID,
		PassengerID: passengerID,
		UserLocation: locationmodels.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		DriverLocation: locationmodels.Location{
			Latitude:  -6.185392,
			Longitude: 106.837153,
		},
		MatchStatus: matchmodels.MatchStatusAccepted,
	}

	// Set up expectations
	mockRepo.EXPECT().
		CreateRide(gomock.Any()).
		DoAndReturn(func(ride *ridemodels.Ride) (*ridemodels.Ride, error) {
			assert.Equal(t, uuid.MustParse(matchID), ride.MatchID)
			assert.Equal(t, uuid.MustParse(driverID), ride.DriverID)
			assert.Equal(t, uuid.MustParse(passengerID), ride.PassengerID)

			// Add ride ID to simulate DB creation
			ride.RideID = uuid.New()
			return ride, nil
		})

	mockGW.EXPECT().
		PublishRidePickup(gomock.Any(), gomock.Any()).
		Return(nil)

	// Act
	err = uc.CreateRide(context.Background(), matchProposal)

	// Assert
	assert.NoError(t, err)
}

func TestCreateRide_RepositoryError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)

	cfg := &core.Config{
		Rides: core.RidesConfig{
			MaxPickupDistanceM: 100.0, // Set pickup distance for test
		},
	}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	driverID := uuid.New().String()
	passengerID := uuid.New().String()
	matchID := uuid.New().String()

	matchProposal := matchmodels.MatchProposal{
		ID:          matchID,
		DriverID:    driverID,
		PassengerID: passengerID,
		MatchStatus: matchmodels.MatchStatusAccepted,
	}

	expectedError := errors.New("database error")

	// Set up expectations
	mockRepo.EXPECT().
		CreateRide(gomock.Any()).
		Return(nil, expectedError)

	// Act
	err = uc.CreateRide(context.Background(), matchProposal)

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

	cfg := &core.Config{
		Rides: core.RidesConfig{
			MaxPickupDistanceM: 100.0, // Set pickup distance for test
		},
	}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	driverID := uuid.New().String()
	passengerID := uuid.New().String()
	matchID := uuid.New().String()

	matchProposal := matchmodels.MatchProposal{
		ID:          matchID,
		DriverID:    driverID,
		PassengerID: passengerID,
		MatchStatus: matchmodels.MatchStatusAccepted,
	}

	expectedError := errors.New("publish error")

	// Set up expectations
	mockRepo.EXPECT().
		CreateRide(gomock.Any()).
		DoAndReturn(func(ride *ridemodels.Ride) (*ridemodels.Ride, error) {
			ride.RideID = uuid.New()
			return ride, nil
		})

	mockGW.EXPECT().
		PublishRidePickup(gomock.Any(), gomock.Any()).
		Return(expectedError)

	// Act
	err = uc.CreateRide(context.Background(), matchProposal)

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

	cfg := &core.Config{
		Rides: core.RidesConfig{
			MaxPickupDistanceM: 100.0, // Set pickup distance for test
		},
	}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)

	entry := &ridemodels.BillingLedger{
		RideID:   rideUUID,
		Distance: 2.5,
		Cost:     7500,
	}

	ride := &ridemodels.Ride{
		RideID:    rideUUID,
		Status:    ridemodels.RideStatusOngoing,
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
	err = uc.ProcessBillingUpdate(context.Background(), rideID, entry)

	// Assert
	assert.NoError(t, err)
}

func TestProcessBillingUpdate_GetRideError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)

	cfg := &core.Config{
		Rides: core.RidesConfig{
			MaxPickupDistanceM: 100.0, // Set pickup distance for test
		},
	}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)

	entry := &ridemodels.BillingLedger{
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
	err = uc.ProcessBillingUpdate(context.Background(), rideID, entry)

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

	cfg := &core.Config{
		Rides: core.RidesConfig{
			MaxPickupDistanceM: 100.0, // Set pickup distance for test
		},
	}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)

	entry := &ridemodels.BillingLedger{
		RideID:   rideUUID,
		Distance: 2.5,
		Cost:     7500,
	}

	ride := &ridemodels.Ride{
		RideID:    rideUUID,
		Status:    ridemodels.RideStatusCompleted, // Ride is already completed
		TotalCost: 10000,
	}

	// Set up expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID).
		Return(ride, nil)

	// Act
	err = uc.ProcessBillingUpdate(context.Background(), rideID, entry)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot update billing for non-active ride")
}

func TestStartRide_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)

	cfg := &core.Config{
		Rides: core.RidesConfig{
			MaxPickupDistanceM: 100.0, // Set pickup distance for test
		},
	}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)

	req := ridemodels.RideStartRequest{
		RideID: rideID,
		DriverLocation: &locationmodels.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		PassengerLocation: &locationmodels.Location{
			Latitude:  -6.175400, // Very close to driver
			Longitude: 106.827160,
		},
	}

	ride := &ridemodels.Ride{
		RideID: rideUUID,
		Status: ridemodels.RideStatusDriverPickup,
	}

	// Set up expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID).
		Return(ride, nil)

	mockRepo.EXPECT().
		UpdateRideStatus(gomock.Any(), rideID, ridemodels.RideStatusOngoing).
		Return(nil)

	// Act
	result, err := uc.StartRide(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, ridemodels.RideStatusOngoing, result.Status)
}

func TestStartRide_DriverTooFar(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)

	cfg := &core.Config{
		Rides: core.RidesConfig{
			MaxPickupDistanceM: 100.0, // Set pickup distance for test
		},
	}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)

	req := ridemodels.RideStartRequest{
		RideID: rideID,
		DriverLocation: &locationmodels.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		PassengerLocation: &locationmodels.Location{
			Latitude:  -6.185392, // Too far from driver
			Longitude: 106.837153,
		},
	}

	ride := &ridemodels.Ride{
		RideID: rideUUID,
		Status: ridemodels.RideStatusDriverPickup,
	}

	// Set up expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID).
		Return(ride, nil)

	// Act
	result, err := uc.StartRide(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "driver is too far from passenger")
	assert.Equal(t, ridemodels.Ride{}, *result)
}

func TestStartRide_InvalidStatus(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)

	cfg := &core.Config{
		Rides: core.RidesConfig{
			MaxPickupDistanceM: 100.0, // Set pickup distance for test
		},
	}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)

	req := ridemodels.RideStartRequest{
		RideID: rideID,
		DriverLocation: &locationmodels.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		PassengerLocation: &locationmodels.Location{
			Latitude:  -6.175400,
			Longitude: 106.827160,
		},
	}

	ride := &ridemodels.Ride{
		RideID: rideUUID,
		Status: ridemodels.RideStatusOngoing, // Wrong status
	}

	// Set up expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID).
		Return(ride, nil)

	// Act
	result, err := uc.StartRide(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot start trip for ride not in driver_pickup state")
	assert.Equal(t, ridemodels.Ride{}, *result)
}

func TestRideArrived_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)

	cfg := &core.Config{
		Pricing: core.PricingConfig{
			AdminFeePercent: 5.0, // 5% admin fee
		},
	}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)
	passengerID := uuid.New()
	adjustmentFactor := 0.8
	totalCost := 10000

	req := ridemodels.RideArrivalReq{
		RideID:           rideID,
		AdjustmentFactor: adjustmentFactor,
	}

	ride := &ridemodels.Ride{
		RideID:      rideUUID,
		PassengerID: passengerID,
		Status:      ridemodels.RideStatusOngoing,
	}

	// Expected values
	adjustedCost := int(float64(totalCost) * adjustmentFactor)
	adminFeePercent := 5.0 / 100.0                           // Use same default as config
	adminFee := int(float64(adjustedCost) * adminFeePercent) // Admin fee on adjusted cost
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
		DoAndReturn(func(_ context.Context, payment *ridemodels.Payment) error {
			assert.Equal(t, rideUUID, payment.RideID)
			assert.Equal(t, adjustedCost, payment.AdjustedCost)
			assert.Equal(t, adminFee, payment.AdminFee)
			assert.Equal(t, driverPayout, payment.DriverPayout)
			assert.Equal(t, ridemodels.PaymentStatusPending, payment.Status)
			return nil
		})

	// Act
	paymentRequest, err := uc.RideArrived(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, paymentRequest)
	assert.Equal(t, rideID, paymentRequest.RideID)
	assert.Equal(t, passengerID.String(), paymentRequest.PassengerID)
	assert.Equal(t, adjustedCost, paymentRequest.TotalCost)
}

func TestRideArrived_InvalidStatus(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)

	cfg := &core.Config{
		Rides: core.RidesConfig{
			MaxPickupDistanceM: 100.0, // Set pickup distance for test
		},
	}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)

	req := ridemodels.RideArrivalReq{
		RideID:           rideID,
		AdjustmentFactor: 0.8,
	}

	ride := &ridemodels.Ride{
		RideID: rideUUID,
		Status: ridemodels.RideStatusCompleted, // Wrong status
	}

	// Set up expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID).
		Return(ride, nil)

	// Act
	paymentRequest, err := uc.RideArrived(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot process arrival for ride that is not ongoing")
	assert.Nil(t, paymentRequest)
}

func TestProcessPayment_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)

	cfg := &core.Config{
		Rides: core.RidesConfig{
			MaxPickupDistanceM: 100.0, // Set pickup distance for test
		},
	}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)
	paymentID := uuid.New()
	totalCost := 8000

	req := ridemodels.PaymentProccessRequest{
		RideID:    rideID,
		TotalCost: totalCost,
		Status:    ridemodels.PaymentStatusAccepted,
	}

	ride := &ridemodels.Ride{
		RideID: rideUUID,
		Status: ridemodels.RideStatusOngoing,
	}

	payment := &ridemodels.Payment{
		PaymentID:    paymentID,
		RideID:       rideUUID,
		AdjustedCost: totalCost,
		Status:       ridemodels.PaymentStatusPending,
	}

	// Set up expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID).
		Return(ride, nil)

	mockRepo.EXPECT().
		GetPaymentByRideID(gomock.Any(), rideID).
		Return(payment, nil)

	mockRepo.EXPECT().
		UpdatePaymentStatus(gomock.Any(), paymentID.String(), ridemodels.PaymentStatusAccepted).
		Return(nil)

	mockRepo.EXPECT().
		CompleteRide(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, updatedRide *ridemodels.Ride) error {
			assert.Equal(t, ridemodels.RideStatusCompleted, updatedRide.Status)
			return nil
		})

	mockGW.EXPECT().
		PublishRideCompleted(gomock.Any(), gomock.Any()).
		Return(nil)

	// Act
	result, err := uc.ProcessPayment(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, ridemodels.PaymentStatusAccepted, result.Status)
}

func TestProcessPayment_TotalCostMismatch(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)

	cfg := &core.Config{
		Rides: core.RidesConfig{
			MaxPickupDistanceM: 100.0, // Set pickup distance for test
		},
	}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)
	paymentID := uuid.New()

	req := ridemodels.PaymentProccessRequest{
		RideID:    rideID,
		TotalCost: 5000, // Different from payment record
		Status:    ridemodels.PaymentStatusAccepted,
	}

	ride := &ridemodels.Ride{
		RideID: rideUUID,
		Status: ridemodels.RideStatusOngoing,
	}

	payment := &ridemodels.Payment{
		PaymentID:    paymentID,
		RideID:       rideUUID,
		AdjustedCost: 8000, // Different from request
		Status:       ridemodels.PaymentStatusPending,
	}

	// Set up expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID).
		Return(ride, nil)

	mockRepo.EXPECT().
		GetPaymentByRideID(gomock.Any(), rideID).
		Return(payment, nil)

	// Act
	result, err := uc.ProcessPayment(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "total cost mismatch")
	assert.Nil(t, result)
}

func TestRideArrived_InvalidAdjustmentFactor(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)

	cfg := &core.Config{
		Pricing: core.PricingConfig{
			AdminFeePercent: 5.0, // 5% admin fee
		},
	}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)
	passengerID := uuid.New()
	totalCost := 10000

	req := ridemodels.RideArrivalReq{
		RideID:           rideID,
		AdjustmentFactor: 1.5, // Invalid - should be reset to 1.0
	}

	ride := &ridemodels.Ride{
		RideID:      rideUUID,
		PassengerID: passengerID,
		Status:      ridemodels.RideStatusOngoing,
	}

	// Expected values with adjustment factor reset to 1.0
	adjustedCost := totalCost
	adminFeePercent := 5.0 / 100.0 // Use same default as config
	adminFee := int(float64(adjustedCost) * adminFeePercent)
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
		DoAndReturn(func(_ context.Context, payment *ridemodels.Payment) error {
			assert.Equal(t, rideUUID, payment.RideID)
			assert.Equal(t, adjustedCost, payment.AdjustedCost)
			assert.Equal(t, adminFee, payment.AdminFee)
			assert.Equal(t, driverPayout, payment.DriverPayout)
			return nil
		})

	// Act
	paymentRequest, err := uc.RideArrived(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, paymentRequest)
	assert.Equal(t, totalCost, paymentRequest.TotalCost) // Should be reset to full cost
}

func TestProcessPayment_InvalidStatus(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)

	cfg := &core.Config{
		Rides: core.RidesConfig{
			MaxPickupDistanceM: 100.0, // Set pickup distance for test
		},
	}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)

	req := ridemodels.PaymentProccessRequest{
		RideID:    rideID,
		TotalCost: 8000,
		Status:    ridemodels.PaymentStatusAccepted,
	}

	ride := &ridemodels.Ride{
		RideID: rideUUID,
		Status: ridemodels.RideStatusCompleted, // Wrong status
	}

	// Set up expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID).
		Return(ride, nil)

	// Act
	result, err := uc.ProcessPayment(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot process payment for ride that is not ongoing")
	assert.Nil(t, result)
}

func TestProcessPayment_PaymentAlreadyProcessed(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)

	cfg := &core.Config{
		Rides: core.RidesConfig{
			MaxPickupDistanceM: 100.0, // Set pickup distance for test
		},
	}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)
	paymentID := uuid.New()

	req := ridemodels.PaymentProccessRequest{
		RideID:    rideID,
		TotalCost: 8000,
		Status:    ridemodels.PaymentStatusAccepted,
	}

	ride := &ridemodels.Ride{
		RideID: rideUUID,
		Status: ridemodels.RideStatusOngoing,
	}

	payment := &ridemodels.Payment{
		PaymentID:    paymentID,
		RideID:       rideUUID,
		AdjustedCost: 8000,
		Status:       ridemodels.PaymentStatusAccepted, // Already processed
	}

	// Set up expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID).
		Return(ride, nil)

	mockRepo.EXPECT().
		GetPaymentByRideID(gomock.Any(), rideID).
		Return(payment, nil)

	// Act
	result, err := uc.ProcessPayment(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot process payment with status")
	assert.Nil(t, result)
}

func TestProcessPayment_RejectedPayment(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)

	cfg := &core.Config{
		Rides: core.RidesConfig{
			MaxPickupDistanceM: 100.0, // Set pickup distance for test
		},
	}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)
	paymentID := uuid.New()
	totalCost := 8000

	req := ridemodels.PaymentProccessRequest{
		RideID:    rideID,
		TotalCost: totalCost,
		Status:    ridemodels.PaymentStatusRejected, // Rejected payment
	}

	ride := &ridemodels.Ride{
		RideID: rideUUID,
		Status: ridemodels.RideStatusOngoing,
	}

	payment := &ridemodels.Payment{
		PaymentID:    paymentID,
		RideID:       rideUUID,
		AdjustedCost: totalCost,
		Status:       ridemodels.PaymentStatusPending,
	}

	// Set up expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID).
		Return(ride, nil)

	mockRepo.EXPECT().
		GetPaymentByRideID(gomock.Any(), rideID).
		Return(payment, nil)

	mockRepo.EXPECT().
		UpdatePaymentStatus(gomock.Any(), paymentID.String(), ridemodels.PaymentStatusRejected).
		Return(nil)

	// Note: No CompleteRide or PublishRideCompleted calls for rejected payment

	// Act
	result, err := uc.ProcessPayment(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, ridemodels.PaymentStatusRejected, result.Status)
}

// Additional unit tests (moved from integration test file)
func TestCreateRideWithFullConfig_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)
	cfg := &core.Config{
		Rides: core.RidesConfig{
			MinDistanceKm:      0.5,
			MaxPickupDistanceM: 100.0, // Set pickup distance for test
		},
	}

	uc, _ := NewRideUC(cfg, mockRepo, mockGW)

	// Test data
	matchProposal := matchmodels.MatchProposal{
		ID:          uuid.New().String(),
		PassengerID: uuid.New().String(),
		DriverID:    uuid.New().String(),
		UserLocation: locationmodels.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
		DriverLocation: locationmodels.Location{
			Latitude:  -6.2000,
			Longitude: 106.8400,
		},
		TargetLocation: locationmodels.Location{
			Latitude:  -6.1751,
			Longitude: 106.8650,
		},
		MatchStatus: matchmodels.MatchStatusDriverConfirmed,
	}

	// Mock expectations
	mockRepo.EXPECT().
		CreateRide(gomock.Any()).
		Return(&ridemodels.Ride{}, nil)

	mockGW.EXPECT().
		PublishRidePickup(gomock.Any(), gomock.Any()).
		Return(nil)

	// Act
	err := uc.CreateRide(context.Background(), matchProposal)

	// Assert
	assert.NoError(t, err)
}

func TestStartRideWithValidDistance_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)
	cfg := &core.Config{
		Rides: core.RidesConfig{
			MinDistanceKm:      0.5,
			MaxPickupDistanceM: 100.0, // Set pickup distance for test
		},
	}

	uc, _ := NewRideUC(cfg, mockRepo, mockGW)

	// Test data
	rideID := uuid.New()
	startRequest := ridemodels.RideStartRequest{
		RideID: rideID.String(),
		DriverLocation: &locationmodels.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
		PassengerLocation: &locationmodels.Location{
			Latitude:  -6.2090,
			Longitude: 106.8458,
		},
	}

	// Mock expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID.String()).
		Return(&ridemodels.Ride{
			RideID:      rideID,
			Status:      ridemodels.RideStatusDriverPickup,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}, nil)

	mockRepo.EXPECT().
		UpdateRideStatus(gomock.Any(), rideID.String(), ridemodels.RideStatusOngoing).
		Return(nil)

	// Act
	ride, err := uc.StartRide(context.Background(), startRequest)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, ride)
	assert.Equal(t, ridemodels.RideStatusOngoing, ride.Status)
}

func TestRideArrivalWithAdjustment_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)
	cfg := &core.Config{
		Rides: core.RidesConfig{
			MinDistanceKm:      0.5,
			MaxPickupDistanceM: 100.0, // Set pickup distance for test
		},
		Pricing: core.PricingConfig{
			RatePerKm:       2000,
			AdminFeePercent: 10.0,
		},
	}

	uc, _ := NewRideUC(cfg, mockRepo, mockGW)

	// Test data
	rideID := uuid.New()
	arrivalReq := ridemodels.RideArrivalReq{
		RideID:           rideID.String(),
		AdjustmentFactor: 0.8,
	}

	expectedPaymentRequest := &ridemodels.PaymentRequest{
		RideID:      rideID.String(),
		PassengerID: uuid.New().String(),
		TotalCost:   15000,
		QRCodeURL:   "https://example.com/qr/payment",
	}

	// Mock expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID.String()).
		Return(&ridemodels.Ride{
			RideID:      rideID,
			PassengerID: uuid.MustParse(expectedPaymentRequest.PassengerID),
			Status:      ridemodels.RideStatusOngoing,
		}, nil)

	mockRepo.EXPECT().
		GetBillingLedgerSum(gomock.Any(), rideID.String()).
		Return(15000, nil)

	mockRepo.EXPECT().
		CreatePayment(gomock.Any(), gomock.Any()).
		Return(nil)

	// Act
	paymentReq, err := uc.RideArrived(context.Background(), arrivalReq)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, paymentReq)
	assert.Equal(t, rideID.String(), paymentReq.RideID)
	// Expected: 15000 * 0.8 = 12000
	assert.Equal(t, 12000, paymentReq.TotalCost)
}

func TestProcessPaymentWithFullFlow_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)
	cfg := &core.Config{
		Rides: core.RidesConfig{
			MinDistanceKm:      0.5,
			MaxPickupDistanceM: 100.0, // Set pickup distance for test
		},
	}

	uc, _ := NewRideUC(cfg, mockRepo, mockGW)

	// Test data
	rideID := uuid.New()
	paymentReq := ridemodels.PaymentProccessRequest{
		RideID:    rideID.String(),
		TotalCost: 25000,
		Status:    ridemodels.PaymentStatusAccepted,
	}

	expectedPayment := &ridemodels.Payment{
		PaymentID:    uuid.New(),
		RideID:       rideID,
		AdjustedCost: 25000,
		AdminFee:     2500,
		DriverPayout: 22500,
		Status:       ridemodels.PaymentStatusAccepted,
		CreatedAt:    time.Now(),
	}

	// Mock expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID.String()).
		Return(&ridemodels.Ride{
			RideID: rideID,
			Status: ridemodels.RideStatusOngoing,
		}, nil)

	mockRepo.EXPECT().
		GetPaymentByRideID(gomock.Any(), rideID.String()).
		Return(&ridemodels.Payment{
			PaymentID:    expectedPayment.PaymentID,
			RideID:       rideID,
			AdjustedCost: 25000,
			Status:       ridemodels.PaymentStatusPending,
		}, nil)

	mockRepo.EXPECT().
		UpdatePaymentStatus(gomock.Any(), expectedPayment.PaymentID.String(), ridemodels.PaymentStatusAccepted).
		Return(nil)

	mockRepo.EXPECT().
		CompleteRide(gomock.Any(), gomock.Any()).
		Return(nil)

	mockGW.EXPECT().
		PublishRideCompleted(gomock.Any(), gomock.Any()).
		Return(nil)

	// Act
	payment, err := uc.ProcessPayment(context.Background(), paymentReq)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, payment)
	assert.Equal(t, ridemodels.PaymentStatusAccepted, payment.Status)
	assert.Equal(t, 25000, payment.AdjustedCost)
}

func TestProcessBillingWithLedger_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)
	cfg := &core.Config{
		Rides: core.RidesConfig{
			MinDistanceKm:      0.5,
			MaxPickupDistanceM: 100.0, // Set pickup distance for test
		},
	}

	uc, _ := NewRideUC(cfg, mockRepo, mockGW)

	// Test data
	rideID := uuid.New().String()
	billingEntry := &ridemodels.BillingLedger{
		EntryID:   uuid.New(),
		RideID:    uuid.MustParse(rideID),
		Distance:  5.2,
		Cost:      15000,
		CreatedAt: time.Now(),
	}

	// Mock expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID).
		Return(&ridemodels.Ride{
			RideID: uuid.MustParse(rideID),
			Status: ridemodels.RideStatusOngoing,
		}, nil)

	mockRepo.EXPECT().
		AddBillingEntry(gomock.Any(), billingEntry).
		Return(nil)

	mockRepo.EXPECT().
		UpdateTotalCost(gomock.Any(), rideID, billingEntry.Cost).
		Return(nil)

	// Act
	err := uc.ProcessBillingUpdate(context.Background(), rideID, billingEntry)

	// Assert
	assert.NoError(t, err)
}
