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

func TestUpdateUserLocation_Success(t *testing.T) {
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

	driverID := uuid.New()

	expectedUser := &models.User{
		ID:       driverID,
		MSISDN:   "+628123456789",
		Role:     "driver",
		IsActive: true,
		FullName: "Test Driver",
	}

	locationUpdate := &models.LocationUpdate{
		RideID:   "ride-123",
		DriverID: driverID.String(),
		Location: models.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
	}

	mockRepo.EXPECT().GetUserByID(gomock.Any(), driverID.String()).Return(expectedUser, nil)
	mockGW.EXPECT().PublishLocationUpdate(gomock.Any(), locationUpdate).Return(nil)

	// Act
	err := uc.UpdateUserLocation(context.Background(), locationUpdate)

	// Assert
	assert.NoError(t, err)
}

func TestUpdateUserLocation_GatewayError(t *testing.T) {
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

	driverID := uuid.New()

	expectedUser := &models.User{
		ID:       driverID,
		MSISDN:   "+628123456789",
		Role:     "driver",
		IsActive: true,
		FullName: "Test Driver",
	}

	locationUpdate := &models.LocationUpdate{
		RideID:   "ride-123",
		DriverID: driverID.String(),
		Location: models.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
	}

	expectedError := errors.New("gateway error")
	mockRepo.EXPECT().GetUserByID(gomock.Any(), driverID.String()).Return(expectedUser, nil)
	mockGW.EXPECT().PublishLocationUpdate(gomock.Any(), locationUpdate).Return(expectedError)

	// Act
	err := uc.UpdateUserLocation(context.Background(), locationUpdate)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gateway error")
}

func TestUpdateUserLocation_NilLocation(t *testing.T) {
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
	err := uc.UpdateUserLocation(context.Background(), nil)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "location cannot be nil")
}

func TestUpdateUserLocation_InvalidCoordinates(t *testing.T) {
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

	locationUpdate := &models.LocationUpdate{
		RideID:   "ride-123",
		DriverID: uuid.New().String(),
		Location: models.Location{
			Latitude:  0, // Invalid coordinates
			Longitude: 0,
		},
	}

	// Act
	err := uc.UpdateUserLocation(context.Background(), locationUpdate)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid location coordinates")
}

func TestUpdateUserLocation_UserNotFound(t *testing.T) {
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

	driverID := uuid.New()
	locationUpdate := &models.LocationUpdate{
		RideID:   "ride-123",
		DriverID: driverID.String(),
		Location: models.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
	}

	expectedError := errors.New("user not found")
	mockRepo.EXPECT().GetUserByID(gomock.Any(), driverID.String()).Return(nil, expectedError)

	// Act
	err := uc.UpdateUserLocation(context.Background(), locationUpdate)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get user")
}

func TestUpdateUserLocation_NonDriverUser(t *testing.T) {
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

	driverID := uuid.New()
	expectedUser := &models.User{
		ID:       driverID,
		MSISDN:   "+628123456789",
		Role:     "passenger", // Not a driver
		IsActive: true,
		FullName: "Test User",
	}

	locationUpdate := &models.LocationUpdate{
		RideID:   "ride-123",
		DriverID: driverID.String(),
		Location: models.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
	}

	mockRepo.EXPECT().GetUserByID(gomock.Any(), driverID.String()).Return(expectedUser, nil)

	// Act
	err := uc.UpdateUserLocation(context.Background(), locationUpdate)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user is not a driver")
}
