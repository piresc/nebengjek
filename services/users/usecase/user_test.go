package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/users/mocks"
	"github.com/stretchr/testify/assert"
)

func TestRegisterUser_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "test-issuer",
		},
	}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	user := &models.User{
		ID:       uuid.New(),
		MSISDN:   "+628123456789",
		FullName: "Test User",
		Role:     "passenger",
		IsActive: true,
	}

	mockRepo.EXPECT().CreateUser(gomock.Any(), user).Return(nil)

	// Act
	err := uc.RegisterUser(context.Background(), user)

	// Assert
	assert.NoError(t, err)
}

func TestRegisterUser_ValidationError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "test-issuer",
		},
	}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	// User without required fields
	invalidUser := &models.User{
		ID: uuid.New(),
		// Missing MSISDN
		FullName: "Test User",
		Role:     "passenger",
		IsActive: true,
	}

	// Act
	err := uc.RegisterUser(context.Background(), invalidUser)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MSISDN is required") // Updated to match actual error message
}

func TestRegisterUser_MissingFullName(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "test-issuer",
		},
	}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	// User without FullName
	invalidUser := &models.User{
		ID:     uuid.New(),
		MSISDN: "+628123456789",
		// Missing FullName
		Role:     "passenger",
		IsActive: true,
	}

	// Act
	err := uc.RegisterUser(context.Background(), invalidUser)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "full name is required")
}

func TestRegisterUser_NilUser(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "test-issuer",
		},
	}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	// Act
	err := uc.RegisterUser(context.Background(), nil)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user cannot be nil")
}

func TestRegisterUser_InvalidMSISDN(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "test-issuer",
		},
	}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	// User with invalid MSISDN
	invalidUser := &models.User{
		ID:       uuid.New(),
		MSISDN:   "invalid-msisdn", // Invalid format
		FullName: "Test User",
		Role:     "passenger",
		IsActive: true,
	}

	// Act
	err := uc.RegisterUser(context.Background(), invalidUser)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid MSISDN format")
}

func TestRegisterUser_RepositoryError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "test-issuer",
		},
	}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	user := &models.User{
		ID:       uuid.New(),
		MSISDN:   "+628123456789",
		FullName: "Test User",
		Role:     "passenger",
		IsActive: true,
	}

	expectedError := errors.New("database error")
	mockRepo.EXPECT().CreateUser(gomock.Any(), user).Return(expectedError)

	// Act
	err := uc.RegisterUser(context.Background(), user)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestGetUserByID_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "test-issuer",
		},
	}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	userId := uuid.New().String()
	expected := &models.User{
		ID:       uuid.MustParse(userId),
		MSISDN:   "+628123456789",
		FullName: "Test User",
		Role:     "passenger",
		IsActive: true,
	}

	mockRepo.EXPECT().GetUserByID(gomock.Any(), userId).Return(expected, nil)

	// Act
	result, err := uc.GetUserByID(context.Background(), userId)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestGetUserByID_NotFound(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "test-issuer",
		},
	}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	userId := uuid.New().String()
	expectedError := errors.New("user not found")

	mockRepo.EXPECT().GetUserByID(gomock.Any(), userId).Return(nil, expectedError)

	// Act
	result, err := uc.GetUserByID(context.Background(), userId)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, expectedError, err)
}

func TestRegisterDriver_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "test-issuer",
		},
	}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	userId := uuid.New()
	existingUser := &models.User{
		ID:       userId,
		MSISDN:   "+628123456789",
		FullName: "Test User",
		Role:     "passenger", // Currently a passenger
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

	// The implementation strips the + from the MSISDN, so we need to expect "628123456789" instead of "+628123456789"
	mockRepo.EXPECT().GetUserByMSISDN(gomock.Any(), "628123456789").Return(existingUser, nil)

	mockRepo.EXPECT().UpdateToDriver(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *models.User) error {
			assert.Equal(t, userId, u.ID)
			assert.Equal(t, "driver", u.Role)
			assert.Equal(t, "car", u.DriverInfo.VehicleType)
			assert.Equal(t, "B 1234 ABC", u.DriverInfo.VehiclePlate)
			return nil
		})

	// Act
	err := uc.RegisterDriver(context.Background(), driverUser)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, userId, driverUser.ID)
}

func TestRegisterDriver_UserNotFound(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "test-issuer",
		},
	}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	driverUser := &models.User{
		MSISDN:   "+628123456789",
		FullName: "Test User",
		Role:     "driver",
		DriverInfo: &models.Driver{
			VehicleType:  "car",
			VehiclePlate: "B 1234 ABC",
		},
	}

	// The implementation strips the + from the MSISDN
	mockRepo.EXPECT().GetUserByMSISDN(gomock.Any(), "628123456789").Return(nil, errors.New("user not found"))

	// Act
	err := uc.RegisterDriver(context.Background(), driverUser)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestRegisterDriver_AlreadyDriver(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "test-issuer",
		},
	}

	uc := NewUserUC(mockRepo, mockGW, cfg)

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

	// The implementation strips the + from the MSISDN
	mockRepo.EXPECT().GetUserByMSISDN(gomock.Any(), "628123456789").Return(existingUser, nil)

	// Act
	err := uc.RegisterDriver(context.Background(), driverUser)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user is already registered as a driver")
}

func TestRegisterDriver_InvalidMSISDN(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "test-issuer",
		},
	}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	driverUser := &models.User{
		MSISDN:   "invalid-msisdn", // Invalid format
		FullName: "Test User",
		Role:     "driver",
		DriverInfo: &models.Driver{
			VehicleType:  "car",
			VehiclePlate: "B 1234 ABC",
		},
	}

	// Act
	err := uc.RegisterDriver(context.Background(), driverUser)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid MSISDN format")
}

func TestRegisterDriver_MissingDriverInfo(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "test-issuer",
		},
	}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	driverUser := &models.User{
		MSISDN:     "+628123456789",
		FullName:   "Test User",
		Role:       "driver",
		DriverInfo: nil, // Missing driver info
	}

	// Mock GetUserByMSISDN to set up the test situation
	mockRepo.EXPECT().GetUserByMSISDN(gomock.Any(), "628123456789").Return(&models.User{
		ID:       uuid.New(),
		MSISDN:   "+628123456789",
		FullName: "Test User",
		Role:     "passenger",
	}, nil)

	// Act
	err := uc.RegisterDriver(context.Background(), driverUser)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "driver info cannot be nil")
}

func TestRegisterDriver_MissingVehicleInfo(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "test-issuer",
		},
	}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	// Test case 1: Missing vehicle type
	driverUser := &models.User{
		MSISDN:   "+628123456789",
		FullName: "Test User",
		Role:     "driver",
		DriverInfo: &models.Driver{
			// Missing VehicleType
			VehiclePlate: "B 1234 ABC",
		},
	}

	// Mock GetUserByMSISDN to set up the test situation
	mockRepo.EXPECT().GetUserByMSISDN(gomock.Any(), "628123456789").Return(&models.User{
		ID:       uuid.New(),
		MSISDN:   "+628123456789",
		FullName: "Test User",
		Role:     "passenger",
	}, nil)

	// Act
	err := uc.RegisterDriver(context.Background(), driverUser)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "vehicle type is required")

	// Test case 2: Missing vehicle plate
	ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo = mocks.NewMockUserRepo(ctrl)

	driverUser2 := &models.User{
		MSISDN:   "+628123456789",
		FullName: "Test User",
		Role:     "driver",
		DriverInfo: &models.Driver{
			VehicleType: "car",
			// Missing VehiclePlate
		},
	}

	mockRepo.EXPECT().GetUserByMSISDN(gomock.Any(), "628123456789").Return(&models.User{
		ID:       uuid.New(),
		MSISDN:   "+628123456789",
		FullName: "Test User",
		Role:     "passenger",
	}, nil)

	uc = NewUserUC(mockRepo, mockGW, cfg)

	// Act
	err = uc.RegisterDriver(context.Background(), driverUser2)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "vehicle plate is required")
}

func TestRegisterDriver_UpdateToDriverError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "test-issuer",
		},
	}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	userId := uuid.New()
	existingUser := &models.User{
		ID:       userId,
		MSISDN:   "+628123456789",
		FullName: "Test User",
		Role:     "passenger", // Currently a passenger
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

	expectedError := errors.New("database error")

	// The implementation strips the + from the MSISDN
	mockRepo.EXPECT().GetUserByMSISDN(gomock.Any(), "628123456789").Return(existingUser, nil)

	mockRepo.EXPECT().UpdateToDriver(gomock.Any(), gomock.Any()).Return(expectedError)

	// Act
	err := uc.RegisterDriver(context.Background(), driverUser)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestRideArrived_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "test-issuer",
		},
	}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	event := &models.RideCompleteEvent{
		RideID:           uuid.New().String(),
		AdjustmentFactor: 1.0,
	}

	mockGW.EXPECT().PublishRideArrived(gomock.Any(), event).Return(nil)

	// Act
	err := uc.RideArrived(context.Background(), event)

	// Assert
	assert.NoError(t, err)
}

func TestRideArrived_GatewayError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "test-issuer",
		},
	}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	event := &models.RideCompleteEvent{
		RideID:           uuid.New().String(),
		AdjustmentFactor: 1.0,
	}

	expectedError := errors.New("gateway error")
	mockGW.EXPECT().PublishRideArrived(gomock.Any(), event).Return(expectedError)

	// Act
	err := uc.RideArrived(context.Background(), event)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish ride arrived event")
}
