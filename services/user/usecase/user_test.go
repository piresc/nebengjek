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

// Mock User Repository
type MockUserRepo struct {
	mock.Mock
}

func (m *MockUserRepo) CreateUser(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepo) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepo) GetUserByMSISDN(ctx context.Context, msisdn string) (*models.User, error) {
	args := m.Called(ctx, msisdn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepo) UpdateToDriver(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepo) CreateOTP(ctx context.Context, otp *models.OTP) error {
	args := m.Called(ctx, otp)
	return args.Error(0)
}

func (m *MockUserRepo) GetOTP(ctx context.Context, msisdn, code string) (*models.OTP, error) {
	args := m.Called(ctx, msisdn, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OTP), args.Error(1)
}

func (m *MockUserRepo) MarkOTPVerified(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Added missing methods to match interface
func (m *MockUserRepo) UpdateUser(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepo) ListUsers(ctx context.Context, offset, limit int) ([]*models.User, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

// Mock User Gateway
type MockUserGW struct {
	mock.Mock
}

func (m *MockUserGW) PublishBeaconEvent(ctx context.Context, beaconReq *models.BeaconRequest) error {
	args := m.Called(ctx, beaconReq)
	return args.Error(0)
}

func (m *MockUserGW) MatchAccept(mp *models.MatchProposal) error {
	args := m.Called(mp)
	return args.Error(0)
}

func (m *MockUserGW) PublishLocationUpdate(ctx context.Context, location *models.LocationUpdate) error {
	args := m.Called(ctx, location)
	return args.Error(0)
}

func (m *MockUserGW) PublishRideArrived(ctx context.Context, event *models.RideCompleteEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func TestRegisterUser_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockRepo := new(MockUserRepo)
	mockGW := new(MockUserGW)

	jwtConfig := models.JWTConfig{
		Secret:     "test-secret",
		Expiration: 60,
		Issuer:     "test-issuer",
	}

	uc := NewUserUC(mockRepo, mockGW, jwtConfig)

	user := &models.User{
		ID:       uuid.New(),
		MSISDN:   "+628123456789",
		FullName: "Test User",
		Role:     "passenger",
		IsActive: true,
	}

	mockRepo.On("CreateUser", ctx, user).Return(nil)

	// Act
	err := uc.RegisterUser(ctx, user)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestRegisterUser_ValidationError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockRepo := new(MockUserRepo)
	mockGW := new(MockUserGW)

	jwtConfig := models.JWTConfig{
		Secret:     "test-secret",
		Expiration: 60,
		Issuer:     "test-issuer",
	}

	uc := NewUserUC(mockRepo, mockGW, jwtConfig)

	testCases := []struct {
		name     string
		user     *models.User
		expected string
	}{
		{
			name:     "Nil User",
			user:     nil,
			expected: "user cannot be nil",
		},
		{
			name: "Empty MSISDN",
			user: &models.User{
				FullName: "Test User",
			},
			expected: "MSISDN is required",
		},
		{
			name: "Empty FullName",
			user: &models.User{
				MSISDN: "+628123456789",
			},
			expected: "full name is required",
		},
		{
			name: "Invalid MSISDN Format",
			user: &models.User{
				MSISDN:   "12345", // Invalid format
				FullName: "Test User",
			},
			expected: "invalid MSISDN format or not a Telkomsel number",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			err := uc.RegisterUser(ctx, tc.user)

			// Assert
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expected)
		})
	}
}

func TestRegisterUser_RepositoryError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockRepo := new(MockUserRepo)
	mockGW := new(MockUserGW)

	jwtConfig := models.JWTConfig{
		Secret:     "test-secret",
		Expiration: 60,
		Issuer:     "test-issuer",
	}

	uc := NewUserUC(mockRepo, mockGW, jwtConfig)

	user := &models.User{
		ID:       uuid.New(),
		MSISDN:   "+628123456789",
		FullName: "Test User",
		Role:     "passenger",
		IsActive: true,
	}

	expectedError := errors.New("database error")
	mockRepo.On("CreateUser", ctx, user).Return(expectedError)

	// Act
	err := uc.RegisterUser(ctx, user)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	mockRepo.AssertExpectations(t)
}

func TestGetUserByID_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockRepo := new(MockUserRepo)
	mockGW := new(MockUserGW)

	jwtConfig := models.JWTConfig{
		Secret:     "test-secret",
		Expiration: 60,
		Issuer:     "test-issuer",
	}

	uc := NewUserUC(mockRepo, mockGW, jwtConfig)

	userId := uuid.New().String()
	expected := &models.User{
		ID:       uuid.MustParse(userId),
		MSISDN:   "+628123456789",
		FullName: "Test User",
		Role:     "passenger",
		IsActive: true,
	}

	mockRepo.On("GetUserByID", ctx, userId).Return(expected, nil)

	// Act
	result, err := uc.GetUserByID(ctx, userId)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	mockRepo.AssertExpectations(t)
}

func TestGetUserByID_NotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockRepo := new(MockUserRepo)
	mockGW := new(MockUserGW)

	jwtConfig := models.JWTConfig{
		Secret:     "test-secret",
		Expiration: 60,
		Issuer:     "test-issuer",
	}

	uc := NewUserUC(mockRepo, mockGW, jwtConfig)

	userId := uuid.New().String()
	expectedError := errors.New("user not found")

	mockRepo.On("GetUserByID", ctx, userId).Return(nil, expectedError)

	// Act
	result, err := uc.GetUserByID(ctx, userId)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, expectedError, err)
	mockRepo.AssertExpectations(t)
}

func TestRegisterDriver_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockRepo := new(MockUserRepo)
	mockGW := new(MockUserGW)

	jwtConfig := models.JWTConfig{
		Secret:     "test-secret",
		Expiration: 60,
		Issuer:     "test-issuer",
	}

	uc := NewUserUC(mockRepo, mockGW, jwtConfig)

	userId := uuid.New()
	existingUser := &models.User{
		ID:       userId,
		MSISDN:   "+628123456789",
		FullName: "Test User",
		Role:     "passenger",
		IsActive: true,
	}

	driverUser := &models.User{
		MSISDN:   "+628123456789",
		FullName: "Test User",
		Role:     "driver",
		DriverInfo: &models.Driver{
			VehicleType:  "car",
			VehiclePlate: "B 1234 ABC",
		},
	}

	mockRepo.On("GetUserByMSISDN", ctx, "+628123456789").Return(existingUser, nil)
	mockRepo.On("UpdateToDriver", ctx, mock.MatchedBy(func(u *models.User) bool {
		return u.ID == userId && u.Role == "driver"
	})).Return(nil)

	// Act
	err := uc.RegisterDriver(ctx, driverUser)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	assert.Equal(t, userId, driverUser.ID)
}

func TestRegisterDriver_UserNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockRepo := new(MockUserRepo)
	mockGW := new(MockUserGW)

	jwtConfig := models.JWTConfig{
		Secret:     "test-secret",
		Expiration: 60,
		Issuer:     "test-issuer",
	}

	uc := NewUserUC(mockRepo, mockGW, jwtConfig)

	driverUser := &models.User{
		MSISDN:   "+628123456789",
		FullName: "Test User",
		Role:     "driver",
		DriverInfo: &models.Driver{
			VehicleType:  "car",
			VehiclePlate: "B 1234 ABC",
		},
	}

	expectedError := errors.New("user not found")
	mockRepo.On("GetUserByMSISDN", ctx, "+628123456789").Return(nil, expectedError)

	// Act
	err := uc.RegisterDriver(ctx, driverUser)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
	mockRepo.AssertExpectations(t)
}

func TestRegisterDriver_AlreadyDriver(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockRepo := new(MockUserRepo)
	mockGW := new(MockUserGW)

	jwtConfig := models.JWTConfig{
		Secret:     "test-secret",
		Expiration: 60,
		Issuer:     "test-issuer",
	}

	uc := NewUserUC(mockRepo, mockGW, jwtConfig)

	userId := uuid.New()
	existingUser := &models.User{
		ID:       userId,
		MSISDN:   "+628123456789",
		FullName: "Test User",
		Role:     "driver", // Already a driver
		IsActive: true,
	}

	driverUser := &models.User{
		MSISDN:   "+628123456789",
		FullName: "Test User",
		Role:     "driver",
		DriverInfo: &models.Driver{
			VehicleType:  "car",
			VehiclePlate: "B 1234 ABC",
		},
	}

	mockRepo.On("GetUserByMSISDN", ctx, "+628123456789").Return(existingUser, nil)

	// Act
	err := uc.RegisterDriver(ctx, driverUser)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user is already registered as a driver")
	mockRepo.AssertExpectations(t)
}

func TestRideArrived_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockRepo := new(MockUserRepo)
	mockGW := new(MockUserGW)

	jwtConfig := models.JWTConfig{
		Secret:     "test-secret",
		Expiration: 60,
		Issuer:     "test-issuer",
	}

	uc := NewUserUC(mockRepo, mockGW, jwtConfig)

	event := &models.RideCompleteEvent{
		RideID:           uuid.New().String(),
		AdjustmentFactor: 1.0,
	}

	mockGW.On("PublishRideArrived", ctx, event).Return(nil)

	// Act
	err := uc.RideArrived(ctx, event)

	// Assert
	assert.NoError(t, err)
	mockGW.AssertExpectations(t)
}

func TestRideArrived_GatewayError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockRepo := new(MockUserRepo)
	mockGW := new(MockUserGW)

	jwtConfig := models.JWTConfig{
		Secret:     "test-secret",
		Expiration: 60,
		Issuer:     "test-issuer",
	}

	uc := NewUserUC(mockRepo, mockGW, jwtConfig)

	event := &models.RideCompleteEvent{
		RideID:           uuid.New().String(),
		AdjustmentFactor: 1.0,
	}

	expectedError := errors.New("gateway error")
	mockGW.On("PublishRideArrived", ctx, event).Return(expectedError)

	// Act
	err := uc.RideArrived(ctx, event)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish ride arrived event")
	mockGW.AssertExpectations(t)
}
