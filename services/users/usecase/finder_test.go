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

func TestUpdateFinderStatus_Success(t *testing.T) {
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

	expectedUser := &models.User{
		ID:       uuid.New(),
		MSISDN:   "+628123456789",
		Role:     "passenger",
		IsActive: true,
		FullName: "Test User",
	}

	request := &models.FinderRequest{
		MSISDN:         "+628123456789",
		IsActive:       true,
		Location:       models.Location{Latitude: -6.2088, Longitude: 106.8456},
		TargetLocation: models.Location{Latitude: -6.1751, Longitude: 106.8650},
	}

	mockRepo.EXPECT().GetUserByMSISDN(gomock.Any(), "+628123456789").Return(expectedUser, nil)
	mockGW.EXPECT().PublishFinderEvent(gomock.Any(), gomock.Any()).Return(nil)

	// Act
	err := uc.UpdateFinderStatus(context.Background(), request)

	// Assert
	assert.NoError(t, err)
}

func TestUpdateFinderStatus_GatewayError(t *testing.T) {
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

	expectedUser := &models.User{
		ID:       uuid.New(),
		MSISDN:   "+628123456789",
		Role:     "passenger",
		IsActive: true,
		FullName: "Test User",
	}

	request := &models.FinderRequest{
		MSISDN:         "+628123456789",
		IsActive:       true,
		Location:       models.Location{Latitude: -6.2088, Longitude: 106.8456},
		TargetLocation: models.Location{Latitude: -6.1751, Longitude: 106.8650},
	}

	expectedError := errors.New("gateway error")
	mockRepo.EXPECT().GetUserByMSISDN(gomock.Any(), "+628123456789").Return(expectedUser, nil)
	mockGW.EXPECT().PublishFinderEvent(gomock.Any(), gomock.Any()).Return(expectedError)

	// Act
	err := uc.UpdateFinderStatus(context.Background(), request)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gateway error")
}

func TestUpdateFinderStatus_UserNotFound(t *testing.T) {
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

	request := &models.FinderRequest{
		MSISDN:         "+628123456789",
		IsActive:       true,
		Location:       models.Location{Latitude: -6.2088, Longitude: 106.8456},
		TargetLocation: models.Location{Latitude: -6.1751, Longitude: 106.8650},
	}

	expectedError := errors.New("user not found")
	mockRepo.EXPECT().GetUserByMSISDN(gomock.Any(), "+628123456789").Return(nil, expectedError)

	// Act
	err := uc.UpdateFinderStatus(context.Background(), request)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestUpdateFinderStatus_DeactivateFinder(t *testing.T) {
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

	expectedUser := &models.User{
		ID:       uuid.New(),
		MSISDN:   "+628123456789",
		Role:     "passenger",
		IsActive: true,
		FullName: "Test User",
	}

	request := &models.FinderRequest{
		MSISDN:         "+628123456789",
		IsActive:       false, // Deactivating finder
		Location:       models.Location{Latitude: -6.2088, Longitude: 106.8456},
		TargetLocation: models.Location{Latitude: -6.1751, Longitude: 106.8650},
	}

	mockRepo.EXPECT().GetUserByMSISDN(gomock.Any(), "+628123456789").Return(expectedUser, nil)
	mockGW.EXPECT().PublishFinderEvent(gomock.Any(), gomock.Any()).Return(nil)

	// Act
	err := uc.UpdateFinderStatus(context.Background(), request)

	// Assert
	assert.NoError(t, err)
}
