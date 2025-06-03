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

func TestConfirmMatch_Success(t *testing.T) {
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

	userID := uuid.New()

	expectedUser := &models.User{
		ID:       userID,
		MSISDN:   "+628123456789",
		Role:     "driver",
		IsActive: true,
		FullName: "Test Driver",
	}

	confirmation := &models.MatchConfirmRequest{
		ID:     "match-123",
		UserID: userID.String(),
		Role:   "driver",
		Status: "ACCEPTED",
	}

	expectedMatch := &models.MatchProposal{
		ID:           "match-123",
		PassengerID:  "passenger-789",
		DriverID:     userID.String(),
		MatchStatus:  models.MatchStatusAccepted,
		UserLocation: models.Location{Latitude: -6.2088, Longitude: 106.8456},
	}

	mockRepo.EXPECT().GetUserByID(gomock.Any(), userID.String()).Return(expectedUser, nil)
	mockGW.EXPECT().MatchConfirm(confirmation).Return(expectedMatch, nil)

	// Act
	match, err := uc.ConfirmMatch(context.Background(), confirmation)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, match)
	assert.Equal(t, "match-123", match.ID)
}

func TestConfirmMatch_GatewayError(t *testing.T) {
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

	userID := uuid.New()

	expectedUser := &models.User{
		ID:       userID,
		MSISDN:   "+628123456789",
		Role:     "driver",
		IsActive: true,
		FullName: "Test Driver",
	}

	confirmation := &models.MatchConfirmRequest{
		ID:     "match-123",
		UserID: userID.String(),
		Role:   "driver",
		Status: "ACCEPTED",
	}

	expectedError := errors.New("gateway error")
	mockRepo.EXPECT().GetUserByID(gomock.Any(), userID.String()).Return(expectedUser, nil)
	mockGW.EXPECT().MatchConfirm(confirmation).Return(nil, expectedError)

	// Act
	match, err := uc.ConfirmMatch(context.Background(), confirmation)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, match)
	assert.Contains(t, err.Error(), "gateway error")
}

func TestConfirmMatch_InvalidStatus(t *testing.T) {
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

	confirmation := &models.MatchConfirmRequest{
		ID:     "match-123",
		UserID: uuid.New().String(),
		Role:   "driver",
		Status: "invalid_status", // Invalid status
	}

	// Act
	match, err := uc.ConfirmMatch(context.Background(), confirmation)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, match)
	assert.Contains(t, err.Error(), "invalid match status")
}

func TestConfirmMatch_UserNotFound(t *testing.T) {
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

	userID := uuid.New()
	confirmation := &models.MatchConfirmRequest{
		ID:     "match-123",
		UserID: userID.String(),
		Role:   "driver",
		Status: "ACCEPTED",
	}

	expectedError := errors.New("user not found")
	mockRepo.EXPECT().GetUserByID(gomock.Any(), userID.String()).Return(nil, expectedError)

	// Act
	match, err := uc.ConfirmMatch(context.Background(), confirmation)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, match)
	assert.Contains(t, err.Error(), "failed to get user")
}

func TestConfirmMatch_RejectMatch(t *testing.T) {
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

	userID := uuid.New()

	expectedUser := &models.User{
		ID:       userID,
		MSISDN:   "+628123456789",
		Role:     "driver",
		IsActive: true,
		FullName: "Test Driver",
	}

	confirmation := &models.MatchConfirmRequest{
		ID:     "match-123",
		UserID: userID.String(),
		Role:   "driver",
		Status: "REJECTED",
	}

	expectedMatch := &models.MatchProposal{
		ID:           "match-123",
		PassengerID:  "passenger-789",
		DriverID:     userID.String(),
		MatchStatus:  models.MatchStatusRejected,
		UserLocation: models.Location{Latitude: -6.2088, Longitude: 106.8456},
	}

	mockRepo.EXPECT().GetUserByID(gomock.Any(), userID.String()).Return(expectedUser, nil)
	mockGW.EXPECT().MatchConfirm(confirmation).Return(expectedMatch, nil)

	// Act
	match, err := uc.ConfirmMatch(context.Background(), confirmation)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, match)
	assert.Equal(t, models.MatchStatusRejected, match.MatchStatus)
}
