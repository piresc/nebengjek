package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/users/mocks"
	"github.com/stretchr/testify/assert"
)

// Integration tests for user management
func TestUserUC_CompleteUserRegistration_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)
	cfg := &models.Config{}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	// Test data
	userID := uuid.New()
	msisdn := "+6281234567890"
	fullName := "John Doe"

	// Step 1: Register user
	user := &models.User{
		MSISDN:   msisdn,
		FullName: fullName,
		Role:     "passenger",
	}

	// Create user
	mockRepo.EXPECT().
		CreateUser(gomock.Any(), user).
		DoAndReturn(func(ctx context.Context, u *models.User) error {
			u.ID = userID
			u.CreatedAt = time.Now()
			u.IsActive = true
			return nil
		})

	// Act - Step 1: Register
	err := uc.RegisterUser(context.Background(), user)

	// Assert - Step 1
	assert.NoError(t, err)
	assert.Equal(t, userID, user.ID)
	assert.Equal(t, "6281234567890", user.MSISDN) // ValidateMSISDN returns formatted without +
	assert.True(t, user.IsActive)

}

func TestUserUC_GetUserByID_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)
	cfg := &models.Config{}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	userID := uuid.New()
	expectedUser := &models.User{
		ID:       userID,
		MSISDN:   "+6281234567890",
		FullName: "John Doe",
		Role:     "passenger",
		IsActive: true,
		CreatedAt: time.Now(),
	}

	mockRepo.EXPECT().
		GetUserByID(gomock.Any(), userID.String()).
		Return(expectedUser, nil)

	// Act
	user, err := uc.GetUserByID(context.Background(), userID.String())

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, userID, user.ID)
	assert.Equal(t, "+6281234567890", user.MSISDN)
	assert.Equal(t, "John Doe", user.FullName)
	assert.True(t, user.IsActive)
}

func TestUserUC_GetUserByID_NotFound(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)
	cfg := &models.Config{}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	userID := uuid.New().String()

	mockRepo.EXPECT().
		GetUserByID(gomock.Any(), userID).
		Return(nil, errors.New("user not found"))

	// Act
	user, err := uc.GetUserByID(context.Background(), userID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "user not found")
}

func TestUserUC_RegisterUser_InvalidMSISDN(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)
	cfg := &models.Config{}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	// Invalid MSISDN (not Telkomsel format)
	user := &models.User{
		MSISDN:   "+6287123456789", // 871 is not a Telkomsel prefix
		FullName: "John Doe",
		Role:     "passenger",
	}

	// Act
	err := uc.RegisterUser(context.Background(), user)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid MSISDN format")
}

func TestUserUC_RegisterUser_WithDriverInfo_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)
	cfg := &models.Config{}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	userID := uuid.New()
	user := &models.User{
		MSISDN:   "+6281234567890",
		FullName: "John Driver",
		Role:     "driver",
		DriverInfo: &models.Driver{
			VehicleType:  "motorcycle",
			VehiclePlate: "B1234XYZ",
		},
	}

	mockRepo.EXPECT().
		CreateUser(gomock.Any(), user).
		DoAndReturn(func(ctx context.Context, u *models.User) error {
			u.ID = userID
			u.CreatedAt = time.Now()
			u.IsActive = true
			if u.DriverInfo != nil {
				u.DriverInfo.UserID = userID
			}
			return nil
		})

	// Act
	err := uc.RegisterUser(context.Background(), user)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "driver", user.Role)
	assert.NotNil(t, user.DriverInfo)
	assert.Equal(t, "motorcycle", user.DriverInfo.VehicleType)
	assert.Equal(t, "B1234XYZ", user.DriverInfo.VehiclePlate)
	assert.Equal(t, userID, user.DriverInfo.UserID)
}

func TestUserUC_RegisterUser_EmptyRole_DefaultsToPassenger(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)
	cfg := &models.Config{}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	userID := uuid.New()
	user := &models.User{
		MSISDN:   "+6281234567890",
		FullName: "John Doe",
		// Role is empty, should default to "passenger"
	}

	mockRepo.EXPECT().
		CreateUser(gomock.Any(), user).
		DoAndReturn(func(ctx context.Context, u *models.User) error {
			u.ID = userID
			u.CreatedAt = time.Now()
			u.IsActive = true
			return nil
		})

	// Act
	err := uc.RegisterUser(context.Background(), user)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "passenger", user.Role) // Should be set to default
	assert.True(t, user.IsActive)
}

func TestUserUC_RegisterUser_RepositoryError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)
	cfg := &models.Config{}

	uc := NewUserUC(mockRepo, mockGW, cfg)

	user := &models.User{
		MSISDN:   "+6281234567890",
		FullName: "John Doe",
		Role:     "passenger",
	}

	mockRepo.EXPECT().
		CreateUser(gomock.Any(), user).
		Return(errors.New("database connection failed"))

	// Act
	err := uc.RegisterUser(context.Background(), user)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection failed")
}