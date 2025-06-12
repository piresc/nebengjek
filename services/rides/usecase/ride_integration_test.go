package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/rides/mocks"
	"github.com/stretchr/testify/assert"
)

// Integration tests for ride usecase
func TestRideUC_CreateRide_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 0.5,
		},
	}

	uc, _ := NewRideUC(cfg, mockRepo, mockGW)

	// Test data
	matchProposal := models.MatchProposal{
		ID:          uuid.New().String(),
		PassengerID: uuid.New().String(),
		DriverID:    uuid.New().String(),
		UserLocation: models.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
		DriverLocation: models.Location{
			Latitude:  -6.2000,
			Longitude: 106.8400,
		},
		TargetLocation: models.Location{
			Latitude:  -6.1751,
			Longitude: 106.8650,
		},
		MatchStatus: models.MatchStatusDriverConfirmed,
	}

	// Mock expectations
	mockRepo.EXPECT().
		CreateRide(gomock.Any()).
		Return(&models.Ride{}, nil)

	mockGW.EXPECT().
		PublishRidePickup(gomock.Any(), gomock.Any()).
		Return(nil)

	// Act
	err := uc.CreateRide(context.Background(), matchProposal)

	// Assert
	assert.NoError(t, err)
}

func TestRideUC_StartRide_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 0.5,
		},
	}

	uc, _ := NewRideUC(cfg, mockRepo, mockGW)

	// Test data
	rideID := uuid.New()
	startRequest := models.RideStartRequest{
		RideID: rideID.String(),
		DriverLocation: &models.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
		PassengerLocation: &models.Location{
			Latitude:  -6.2090,
			Longitude: 106.8458,
		},
	}

	// Mock expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID.String()).
		Return(&models.Ride{
			RideID:      rideID,
			Status:      models.RideStatusDriverPickup,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}, nil)

	mockRepo.EXPECT().
		UpdateRideStatus(gomock.Any(), rideID.String(), models.RideStatusOngoing).
		Return(nil)

	// Act
	ride, err := uc.StartRide(context.Background(), startRequest)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, ride)
	assert.Equal(t, models.RideStatusOngoing, ride.Status)
}

func TestRideUC_RideArrived_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 0.5,
		},
		Pricing: models.PricingConfig{
			RatePerKm:       2000,
			AdminFeePercent: 10.0,
		},
	}

	uc, _ := NewRideUC(cfg, mockRepo, mockGW)

	// Test data
	rideID := uuid.New()
	arrivalReq := models.RideArrivalReq{
		RideID:           rideID.String(),
		AdjustmentFactor: 0.8,
	}

	expectedPaymentRequest := &models.PaymentRequest{
		RideID:      rideID.String(),
		PassengerID: uuid.New().String(),
		TotalCost:   15000,
		QRCodeURL:   "https://example.com/qr/payment",
	}

	// Mock expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID.String()).
		Return(&models.Ride{
			RideID:      rideID,
			PassengerID: uuid.MustParse(expectedPaymentRequest.PassengerID),
			Status:      models.RideStatusOngoing,
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

func TestRideUC_ProcessPayment_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 0.5,
		},
	}

	uc, _ := NewRideUC(cfg, mockRepo, mockGW)

	// Test data
	rideID := uuid.New()
	paymentReq := models.PaymentProccessRequest{
		RideID:    rideID.String(),
		TotalCost: 25000,
		Status:    models.PaymentStatusAccepted,
	}

	expectedPayment := &models.Payment{
		PaymentID:    uuid.New(),
		RideID:       rideID,
		AdjustedCost: 25000,
		AdminFee:     2500,
		DriverPayout: 22500,
		Status:       models.PaymentStatusAccepted,
		CreatedAt:    time.Now(),
	}

	// Mock expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID.String()).
		Return(&models.Ride{
			RideID: rideID,
			Status: models.RideStatusOngoing,
		}, nil)

	mockRepo.EXPECT().
		GetPaymentByRideID(gomock.Any(), rideID.String()).
		Return(&models.Payment{
			PaymentID:    expectedPayment.PaymentID,
			RideID:       rideID,
			AdjustedCost: 25000,
			Status:       models.PaymentStatusPending,
		}, nil)

	mockRepo.EXPECT().
		UpdatePaymentStatus(gomock.Any(), expectedPayment.PaymentID.String(), models.PaymentStatusAccepted).
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
	assert.Equal(t, models.PaymentStatusAccepted, payment.Status)
	assert.Equal(t, 25000, payment.AdjustedCost)
}

func TestRideUC_ProcessBillingUpdate_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRideRepo(ctrl)
	mockGW := mocks.NewMockRideGW(ctrl)
	cfg := &models.Config{
		Rides: models.RidesConfig{
			MinDistanceKm: 0.5,
		},
	}

	uc, _ := NewRideUC(cfg, mockRepo, mockGW)

	// Test data
	rideID := uuid.New().String()
	billingEntry := &models.BillingLedger{
		EntryID:   uuid.New(),
		RideID:    uuid.MustParse(rideID),
		Distance:  5.2,
		Cost:      15000,
		CreatedAt: time.Now(),
	}

	// Mock expectations
	mockRepo.EXPECT().
		GetRide(gomock.Any(), rideID).
		Return(&models.Ride{
			RideID: uuid.MustParse(rideID),
			Status: models.RideStatusOngoing,
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