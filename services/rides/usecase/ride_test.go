package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock Ride Repository
type MockRideRepo struct {
	mock.Mock
}

func (m *MockRideRepo) CreateRide(ctx context.Context, ride *models.Ride) error {
	args := m.Called(ctx, ride)
	return args.Error(0)
}

func (m *MockRideRepo) GetRide(ctx context.Context, rideID string) (*models.Ride, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ride), args.Error(1)
}

func (m *MockRideRepo) UpdateRideStatus(ctx context.Context, rideID string, status models.RideStatus) error {
	args := m.Called(ctx, rideID, status)
	return args.Error(0)
}

func (m *MockRideRepo) RecordBillingEntry(ctx context.Context, entry *models.BillingLedger) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockRideRepo) GetRideBillingTotal(ctx context.Context, rideID string) (*models.BillingTotal, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BillingTotal), args.Error(1)
}

func (m *MockRideRepo) RecordPayment(ctx context.Context, payment *models.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

// Mock Ride Gateway
type MockRideGW struct {
	mock.Mock
}

func (m *MockRideGW) PublishRideCreated(ctx context.Context, ride *models.Ride) error {
	args := m.Called(ctx, ride)
	return args.Error(0)
}

func (m *MockRideGW) PublishRideCompleted(ctx context.Context, payment *models.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func TestCreateRide_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockRideRepo)
	mockGW := new(MockRideGW)

	cfg := &models.Config{
		Billing: models.BillingConfig{
			FareRate:        3000, // 3000 IDR per km
			AdminFeePercent: 5,    // 5% admin fee
		},
	}

	uc := NewRideUC(cfg, mockRepo, mockGW)

	mp := models.MatchProposal{
		ID:          uuid.New().String(),
		DriverID:    uuid.New().String(),
		PassengerID: uuid.New().String(),
		MatchStatus: models.MatchStatusAccepted,
		PickupLocation: models.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		DestinationLocation: models.Location{
			Latitude:  -6.185392,
			Longitude: 106.837153,
		},
		EstimatedDistance: 5.2, // km
	}

	// Should create a ride with the correct data
	mockRepo.On("CreateRide", mock.Anything, mock.MatchedBy(func(r *models.Ride) bool {
		return r.DriverID.String() == mp.DriverID &&
			r.PassengerID.String() == mp.PassengerID &&
			r.Status == models.RideStatusCreated
	})).Return(nil)

	// Should publish ride created event
	mockGW.On("PublishRideCreated", mock.Anything, mock.AnythingOfType("*models.Ride")).Return(nil)

	// Act
	err := uc.CreateRide(context.Background(), mp)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockGW.AssertExpectations(t)
}

func TestCreateRide_RepositoryError(t *testing.T) {
	// Arrange
	mockRepo := new(MockRideRepo)
	mockGW := new(MockRideGW)

	cfg := &models.Config{
		Billing: models.BillingConfig{
			FareRate:        3000,
			AdminFeePercent: 5,
		},
	}

	uc := NewRideUC(cfg, mockRepo, mockGW)

	mp := models.MatchProposal{
		ID:          uuid.New().String(),
		DriverID:    uuid.New().String(),
		PassengerID: uuid.New().String(),
		MatchStatus: models.MatchStatusAccepted,
	}

	expectedError := errors.New("database error")
	mockRepo.On("CreateRide", mock.Anything, mock.AnythingOfType("*models.Ride")).Return(expectedError)

	// Act
	err := uc.CreateRide(context.Background(), mp)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create ride")
	mockRepo.AssertExpectations(t)
}

func TestCreateRide_PublishError(t *testing.T) {
	// Arrange
	mockRepo := new(MockRideRepo)
	mockGW := new(MockRideGW)

	cfg := &models.Config{
		Billing: models.BillingConfig{
			FareRate:        3000,
			AdminFeePercent: 5,
		},
	}

	uc := NewRideUC(cfg, mockRepo, mockGW)

	mp := models.MatchProposal{
		ID:          uuid.New().String(),
		DriverID:    uuid.New().String(),
		PassengerID: uuid.New().String(),
		MatchStatus: models.MatchStatusAccepted,
	}

	// Repository succeeds but publishing fails
	mockRepo.On("CreateRide", mock.Anything, mock.AnythingOfType("*models.Ride")).Return(nil)

	expectedError := errors.New("publish error")
	mockGW.On("PublishRideCreated", mock.Anything, mock.AnythingOfType("*models.Ride")).Return(expectedError)

	// Act
	err := uc.CreateRide(context.Background(), mp)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish ride created event")
	mockRepo.AssertExpectations(t)
	mockGW.AssertExpectations(t)
}

func TestProcessBillingUpdate_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockRideRepo)
	mockGW := new(MockRideGW)

	cfg := &models.Config{
		Billing: models.BillingConfig{
			FareRate:        3000,
			AdminFeePercent: 5,
		},
	}

	uc := NewRideUC(cfg, mockRepo, mockGW)

	rideID := uuid.New().String()
	aggregate := models.LocationAggregate{
		RideID:   rideID,
		Distance: 2.5, // km
	}

	// Calculate expected cost based on fare rate
	expectedCost := int64(aggregate.Distance * float64(cfg.Billing.FareRate))

	// Mock BillingLedger entry
	mockRepo.On("RecordBillingEntry", mock.Anything, mock.MatchedBy(func(entry *models.BillingLedger) bool {
		return entry.RideID == rideID &&
			entry.Distance == aggregate.Distance &&
			entry.Cost == expectedCost
	})).Return(nil)

	// Act
	err := uc.ProcessBillingUpdate(context.Background(), aggregate)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestProcessBillingUpdate_RepositoryError(t *testing.T) {
	// Arrange
	mockRepo := new(MockRideRepo)
	mockGW := new(MockRideGW)

	cfg := &models.Config{
		Billing: models.BillingConfig{
			FareRate:        3000,
			AdminFeePercent: 5,
		},
	}

	uc := NewRideUC(cfg, mockRepo, mockGW)

	rideID := uuid.New().String()
	aggregate := models.LocationAggregate{
		RideID:   rideID,
		Distance: 2.5,
	}

	expectedError := errors.New("repository error")
	mockRepo.On("RecordBillingEntry", mock.Anything, mock.AnythingOfType("*models.BillingLedger")).Return(expectedError)

	// Act
	err := uc.ProcessBillingUpdate(context.Background(), aggregate)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	mockRepo.AssertExpectations(t)
}

func TestCompleteRide_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockRideRepo)
	mockGW := new(MockRideGW)

	cfg := &models.Config{
		Billing: models.BillingConfig{
			FareRate:        3000,
			AdminFeePercent: 5,
		},
	}

	uc := NewRideUC(cfg, mockRepo, mockGW)

	rideID := uuid.New().String()
	adjustmentFactor := 1.0 // No adjustment

	// Mock ride info
	ride := &models.Ride{
		ID:           uuid.MustParse(rideID),
		DriverID:     uuid.New(),
		PassengerID:  uuid.New(),
		Status:       models.RideStatusInProgress,
		EstimatedFee: 30000,
	}

	// Mock billing total
	billingTotal := &models.BillingTotal{
		TotalDistance: 10.0,
		TotalCost:     30000,
	}

	mockRepo.On("GetRide", mock.Anything, rideID).Return(ride, nil)
	mockRepo.On("GetRideBillingTotal", mock.Anything, rideID).Return(billingTotal, nil)
	mockRepo.On("UpdateRideStatus", mock.Anything, rideID, models.RideStatusCompleted).Return(nil)
	mockRepo.On("RecordPayment", mock.Anything, mock.MatchedBy(func(p *models.Payment) bool {
		// Check payment calculation
		expectedAmount := float64(billingTotal.TotalCost) * adjustmentFactor // 30000 * 1.0 = 30000
		expectedAdminFee := expectedAmount * 0.05                            // 5% of 30000 = 1500
		expectedDriverPayout := expectedAmount - expectedAdminFee            // 30000 - 1500 = 28500

		return p.RideID == rideID &&
			p.AdjustedCost == int64(expectedAmount) &&
			p.AdminFee == int64(expectedAdminFee) &&
			p.DriverPayout == int64(expectedDriverPayout)
	})).Return(nil)

	mockGW.On("PublishRideCompleted", mock.Anything, mock.AnythingOfType("*models.Payment")).Return(nil)

	// Act
	payment, err := uc.CompleteRide(context.Background(), rideID, adjustmentFactor)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, payment)
	assert.Equal(t, rideID, payment.RideID)
	mockRepo.AssertExpectations(t)
	mockGW.AssertExpectations(t)
}

func TestCompleteRide_GetRideError(t *testing.T) {
	// Arrange
	mockRepo := new(MockRideRepo)
	mockGW := new(MockRideGW)

	cfg := &models.Config{
		Billing: models.BillingConfig{
			FareRate:        3000,
			AdminFeePercent: 5,
		},
	}

	uc := NewRideUC(cfg, mockRepo, mockGW)

	rideID := uuid.New().String()
	adjustmentFactor := 1.0

	expectedError := errors.New("ride not found")
	mockRepo.On("GetRide", mock.Anything, rideID).Return(nil, expectedError)

	// Act
	payment, err := uc.CompleteRide(context.Background(), rideID, adjustmentFactor)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, payment)
	assert.Equal(t, expectedError, err)
	mockRepo.AssertExpectations(t)
}

func TestCompleteRide_InvalidRideStatus(t *testing.T) {
	// Arrange
	mockRepo := new(MockRideRepo)
	mockGW := new(MockRideGW)

	cfg := &models.Config{
		Billing: models.BillingConfig{
			FareRate:        3000,
			AdminFeePercent: 5,
		},
	}

	uc := NewRideUC(cfg, mockRepo, mockGW)

	rideID := uuid.New().String()
	adjustmentFactor := 1.0

	// Already completed ride
	ride := &models.Ride{
		ID:     uuid.MustParse(rideID),
		Status: models.RideStatusCompleted,
	}

	mockRepo.On("GetRide", mock.Anything, rideID).Return(ride, nil)

	// Act
	payment, err := uc.CompleteRide(context.Background(), rideID, adjustmentFactor)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, payment)
	assert.Contains(t, err.Error(), "ride is not in progress")
	mockRepo.AssertExpectations(t)
}

func TestCompleteRide_GetBillingTotalError(t *testing.T) {
	// Arrange
	mockRepo := new(MockRideRepo)
	mockGW := new(MockRideGW)

	cfg := &models.Config{
		Billing: models.BillingConfig{
			FareRate:        3000,
			AdminFeePercent: 5,
		},
	}

	uc := NewRideUC(cfg, mockRepo, mockGW)

	rideID := uuid.New().String()
	adjustmentFactor := 1.0

	ride := &models.Ride{
		ID:     uuid.MustParse(rideID),
		Status: models.RideStatusInProgress,
	}

	expectedError := errors.New("billing error")
	mockRepo.On("GetRide", mock.Anything, rideID).Return(ride, nil)
	mockRepo.On("GetRideBillingTotal", mock.Anything, rideID).Return(nil, expectedError)

	// Act
	payment, err := uc.CompleteRide(context.Background(), rideID, adjustmentFactor)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, payment)
	assert.Equal(t, expectedError, err)
	mockRepo.AssertExpectations(t)
}

func TestCompleteRide_UpdateStatusError(t *testing.T) {
	// Arrange
	mockRepo := new(MockRideRepo)
	mockGW := new(MockRideGW)

	cfg := &models.Config{
		Billing: models.BillingConfig{
			FareRate:        3000,
			AdminFeePercent: 5,
		},
	}

	uc := NewRideUC(cfg, mockRepo, mockGW)

	rideID := uuid.New().String()
	adjustmentFactor := 1.0

	ride := &models.Ride{
		ID:     uuid.MustParse(rideID),
		Status: models.RideStatusInProgress,
	}

	billingTotal := &models.BillingTotal{
		TotalDistance: 10.0,
		TotalCost:     30000,
	}

	expectedError := errors.New("status update error")
	mockRepo.On("GetRide", mock.Anything, rideID).Return(ride, nil)
	mockRepo.On("GetRideBillingTotal", mock.Anything, rideID).Return(billingTotal, nil)
	mockRepo.On("UpdateRideStatus", mock.Anything, rideID, models.RideStatusCompleted).Return(expectedError)

	// Act
	payment, err := uc.CompleteRide(context.Background(), rideID, adjustmentFactor)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, payment)
	assert.Equal(t, expectedError, err)
	mockRepo.AssertExpectations(t)
}

func TestCompleteRide_RecordPaymentError(t *testing.T) {
	// Arrange
	mockRepo := new(MockRideRepo)
	mockGW := new(MockRideGW)

	cfg := &models.Config{
		Billing: models.BillingConfig{
			FareRate:        3000,
			AdminFeePercent: 5,
		},
	}

	uc := NewRideUC(cfg, mockRepo, mockGW)

	rideID := uuid.New().String()
	adjustmentFactor := 1.0

	ride := &models.Ride{
		ID:     uuid.MustParse(rideID),
		Status: models.RideStatusInProgress,
	}

	billingTotal := &models.BillingTotal{
		TotalDistance: 10.0,
		TotalCost:     30000,
	}

	expectedError := errors.New("payment error")
	mockRepo.On("GetRide", mock.Anything, rideID).Return(ride, nil)
	mockRepo.On("GetRideBillingTotal", mock.Anything, rideID).Return(billingTotal, nil)
	mockRepo.On("UpdateRideStatus", mock.Anything, rideID, models.RideStatusCompleted).Return(nil)
	mockRepo.On("RecordPayment", mock.Anything, mock.AnythingOfType("*models.Payment")).Return(expectedError)

	// Act
	payment, err := uc.CompleteRide(context.Background(), rideID, adjustmentFactor)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, payment)
	assert.Equal(t, expectedError, err)
	mockRepo.AssertExpectations(t)
}

func TestCompleteRide_PublishError(t *testing.T) {
	// Arrange
	mockRepo := new(MockRideRepo)
	mockGW := new(MockRideGW)

	cfg := &models.Config{
		Billing: models.BillingConfig{
			FareRate:        3000,
			AdminFeePercent: 5,
		},
	}

	uc := NewRideUC(cfg, mockRepo, mockGW)

	rideID := uuid.New().String()
	adjustmentFactor := 1.0

	ride := &models.Ride{
		ID:     uuid.MustParse(rideID),
		Status: models.RideStatusInProgress,
	}

	billingTotal := &models.BillingTotal{
		TotalDistance: 10.0,
		TotalCost:     30000,
	}

	expectedError := errors.New("publish error")
	mockRepo.On("GetRide", mock.Anything, rideID).Return(ride, nil)
	mockRepo.On("GetRideBillingTotal", mock.Anything, rideID).Return(billingTotal, nil)
	mockRepo.On("UpdateRideStatus", mock.Anything, rideID, models.RideStatusCompleted).Return(nil)
	mockRepo.On("RecordPayment", mock.Anything, mock.AnythingOfType("*models.Payment")).Return(nil)
	mockGW.On("PublishRideCompleted", mock.Anything, mock.AnythingOfType("*models.Payment")).Return(expectedError)

	// Act
	payment, err := uc.CompleteRide(context.Background(), rideID, adjustmentFactor)

	// Assert
	assert.Error(t, err)
	assert.NotNil(t, payment) // Payment should still be created
	assert.Contains(t, err.Error(), "failed to publish ride completed event")
	mockRepo.AssertExpectations(t)
	mockGW.AssertExpectations(t)
}
