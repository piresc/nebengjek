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

func TestStartRide_Success(t *testing.T) {
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

	req := models.RideStartRequest{
		RideID: rideID,
		DriverLocation: &models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		PassengerLocation: &models.Location{
			Latitude:  -6.175400, // Very close to driver
			Longitude: 106.827160,
		},
	}

	ride := &models.Ride{
		RideID: rideUUID,
		Status: models.RideStatusDriverPickup,
	}

	// Set up expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID).
		Return(ride, nil)

	mockRepo.EXPECT().
		UpdateRideStatus(gomock.Any(), rideID, models.RideStatusOngoing).
		Return(nil)

	// Act
	result, err := uc.StartRide(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.RideStatusOngoing, result.Status)
}

func TestStartRide_DriverTooFar(t *testing.T) {
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

	req := models.RideStartRequest{
		RideID: rideID,
		DriverLocation: &models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		PassengerLocation: &models.Location{
			Latitude:  -6.185392, // Too far from driver
			Longitude: 106.837153,
		},
	}

	ride := &models.Ride{
		RideID: rideUUID,
		Status: models.RideStatusDriverPickup,
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
	assert.Equal(t, models.Ride{}, *result)
}

func TestStartRide_InvalidStatus(t *testing.T) {
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

	req := models.RideStartRequest{
		RideID: rideID,
		DriverLocation: &models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		PassengerLocation: &models.Location{
			Latitude:  -6.175400,
			Longitude: 106.827160,
		},
	}

	ride := &models.Ride{
		RideID: rideUUID,
		Status: models.RideStatusOngoing, // Wrong status
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
	assert.Equal(t, models.Ride{}, *result)
}

func TestRideArrived_Success(t *testing.T) {
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
	passengerID := uuid.New()
	adjustmentFactor := 0.8
	totalCost := 10000

	req := models.RideArrivalReq{
		RideID:           rideID,
		AdjustmentFactor: adjustmentFactor,
	}

	ride := &models.Ride{
		RideID:      rideUUID,
		PassengerID: passengerID,
		Status:      models.RideStatusOngoing,
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
			assert.Equal(t, models.PaymentStatusPending, payment.Status)
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

	cfg := &models.Config{}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)

	req := models.RideArrivalReq{
		RideID:           rideID,
		AdjustmentFactor: 0.8,
	}

	ride := &models.Ride{
		RideID: rideUUID,
		Status: models.RideStatusCompleted, // Wrong status
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

	cfg := &models.Config{}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)
	paymentID := uuid.New()
	totalCost := 8000

	req := models.PaymentProccessRequest{
		RideID:    rideID,
		TotalCost: totalCost,
		Status:    models.PaymentStatusAccepted,
	}

	ride := &models.Ride{
		RideID: rideUUID,
		Status: models.RideStatusOngoing,
	}

	payment := &models.Payment{
		PaymentID:    paymentID,
		RideID:       rideUUID,
		AdjustedCost: totalCost,
		Status:       models.PaymentStatusPending,
	}

	// Set up expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID).
		Return(ride, nil)

	mockRepo.EXPECT().
		GetPaymentByRideID(gomock.Any(), rideID).
		Return(payment, nil)

	mockRepo.EXPECT().
		UpdatePaymentStatus(gomock.Any(), paymentID.String(), models.PaymentStatusAccepted).
		Return(nil)

	mockRepo.EXPECT().
		CompleteRide(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, updatedRide *models.Ride) error {
			assert.Equal(t, models.RideStatusCompleted, updatedRide.Status)
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
	assert.Equal(t, models.PaymentStatusAccepted, result.Status)
}

func TestProcessPayment_TotalCostMismatch(t *testing.T) {
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
	paymentID := uuid.New()

	req := models.PaymentProccessRequest{
		RideID:    rideID,
		TotalCost: 5000, // Different from payment record
		Status:    models.PaymentStatusAccepted,
	}

	ride := &models.Ride{
		RideID: rideUUID,
		Status: models.RideStatusOngoing,
	}

	payment := &models.Payment{
		PaymentID:    paymentID,
		RideID:       rideUUID,
		AdjustedCost: 8000, // Different from request
		Status:       models.PaymentStatusPending,
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

	cfg := &models.Config{}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)
	passengerID := uuid.New()
	totalCost := 10000

	req := models.RideArrivalReq{
		RideID:           rideID,
		AdjustmentFactor: 1.5, // Invalid - should be reset to 1.0
	}

	ride := &models.Ride{
		RideID:      rideUUID,
		PassengerID: passengerID,
		Status:      models.RideStatusOngoing,
	}

	// Expected values with adjustment factor reset to 1.0
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

	cfg := &models.Config{}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)

	req := models.PaymentProccessRequest{
		RideID:    rideID,
		TotalCost: 8000,
		Status:    models.PaymentStatusAccepted,
	}

	ride := &models.Ride{
		RideID: rideUUID,
		Status: models.RideStatusCompleted, // Wrong status
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

	cfg := &models.Config{}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)
	paymentID := uuid.New()

	req := models.PaymentProccessRequest{
		RideID:    rideID,
		TotalCost: 8000,
		Status:    models.PaymentStatusAccepted,
	}

	ride := &models.Ride{
		RideID: rideUUID,
		Status: models.RideStatusOngoing,
	}

	payment := &models.Payment{
		PaymentID:    paymentID,
		RideID:       rideUUID,
		AdjustedCost: 8000,
		Status:       models.PaymentStatusAccepted, // Already processed
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

	cfg := &models.Config{}
	uc, err := NewRideUC(cfg, mockRepo, mockGW)
	require.NoError(t, err)

	rideID := uuid.New().String()
	rideUUID := uuid.MustParse(rideID)
	paymentID := uuid.New()
	totalCost := 8000

	req := models.PaymentProccessRequest{
		RideID:    rideID,
		TotalCost: totalCost,
		Status:    models.PaymentStatusRejected, // Rejected payment
	}

	ride := &models.Ride{
		RideID: rideUUID,
		Status: models.RideStatusOngoing,
	}

	payment := &models.Payment{
		PaymentID:    paymentID,
		RideID:       rideUUID,
		AdjustedCost: totalCost,
		Status:       models.PaymentStatusPending,
	}

	// Set up expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID).
		Return(ride, nil)

	mockRepo.EXPECT().
		GetPaymentByRideID(gomock.Any(), rideID).
		Return(payment, nil)

	mockRepo.EXPECT().
		UpdatePaymentStatus(gomock.Any(), paymentID.String(), models.PaymentStatusRejected).
		Return(nil)

	// Note: No CompleteRide or PublishRideCompleted calls for rejected payment

	// Act
	result, err := uc.ProcessPayment(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.PaymentStatusRejected, result.Status)
}
