package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	coremodels "github.com/piresc/nebengjek/internal/pkg/models/core"
	usermodels "github.com/piresc/nebengjek/internal/pkg/models/user"
	"github.com/piresc/nebengjek/services/users/mocks"
	"github.com/stretchr/testify/assert"
)

func TestUpdateBeaconStatus_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	cfg := &coremodels.Config{
		JWT: coremodels.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "test-issuer",
		},
	}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	request := &coremodels.BeaconRequest{
		MSISDN:    "+628123456789",
		IsActive:  true,
		Latitude:  -6.2088,
		Longitude: 106.8456,
	}

	expectedUser := &usermodels.User{
		ID:       uuid.New(),
		MSISDN:   "+628123456789",
		FullName: "Test User",
		Role:     "driver",
	}

	mockRepo.EXPECT().GetUserByMSISDN(gomock.Any(), "+628123456789").Return(expectedUser, nil)
	mockRepo.EXPECT().SetEventCacheWithTTL(gomock.Any(), "beacon_update", expectedUser.ID.String(), gomock.Any(), gomock.Any()).Return(true, nil)
	mockGW.EXPECT().PublishBeaconEvent(gomock.Any(), gomock.Any()).Return(nil)

	// Act
	err := uc.UpdateBeaconStatus(context.Background(), request)

	// Assert
	assert.NoError(t, err)
}

func TestUpdateBeaconStatus_GatewayError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	cfg := &coremodels.Config{
		JWT: coremodels.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "test-issuer",
		},
	}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	request := &coremodels.BeaconRequest{
		MSISDN:    "+628123456789",
		IsActive:  true,
		Latitude:  -6.2088,
		Longitude: 106.8456,
	}

	expectedUser := &usermodels.User{
		ID:       uuid.New(),
		MSISDN:   "+628123456789",
		FullName: "Test User",
		Role:     "driver",
	}

	expectedError := errors.New("gateway error")
	mockRepo.EXPECT().GetUserByMSISDN(gomock.Any(), "+628123456789").Return(expectedUser, nil)
	mockRepo.EXPECT().SetEventCacheWithTTL(gomock.Any(), "beacon_update", expectedUser.ID.String(), gomock.Any(), gomock.Any()).Return(true, nil)
	mockGW.EXPECT().PublishBeaconEvent(gomock.Any(), gomock.Any()).Return(expectedError)

	// Act
	err := uc.UpdateBeaconStatus(context.Background(), request)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestUpdateBeaconStatus_UserNotFound(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	cfg := &coremodels.Config{
		JWT: coremodels.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "test-issuer",
		},
	}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	request := &coremodels.BeaconRequest{
		MSISDN:    "+628123456789",
		IsActive:  true,
		Latitude:  -6.2088,
		Longitude: 106.8456,
	}

	expectedError := errors.New("user not found")
	mockRepo.EXPECT().GetUserByMSISDN(gomock.Any(), "+628123456789").Return(nil, expectedError)

	// Act
	err := uc.UpdateBeaconStatus(context.Background(), request)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestUpdateBeaconStatus_DeactivateBeacon(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	cfg := &coremodels.Config{
		JWT: coremodels.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "test-issuer",
		},
	}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	expectedUser := &usermodels.User{
		ID:       uuid.New(),
		MSISDN:   "+628123456789",
		Role:     "driver",
		IsActive: true,
		FullName: "Test Driver",
	}

	request := &coremodels.BeaconRequest{
		MSISDN:    "+628123456789",
		IsActive:  false, // Deactivating beacon
		Latitude:  -6.2088,
		Longitude: 106.8456,
	}

	mockRepo.EXPECT().GetUserByMSISDN(gomock.Any(), "+628123456789").Return(expectedUser, nil)
	mockRepo.EXPECT().SetEventCacheWithTTL(gomock.Any(), "beacon_update", expectedUser.ID.String(), gomock.Any(), gomock.Any()).Return(true, nil)
	mockGW.EXPECT().PublishBeaconEvent(gomock.Any(), gomock.Any()).Return(nil)

	// Act
	err := uc.UpdateBeaconStatus(context.Background(), request)

	// Assert
	assert.NoError(t, err)
}
