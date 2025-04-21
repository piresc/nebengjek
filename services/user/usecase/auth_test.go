package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/user/mocks"
	"github.com/stretchr/testify/assert"
)

func TestGenerateOTP_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	// Test data
	msisdn := "081234567890"
	formattedMSISDN := "6281234567890" // Corrected: Added trailing zero to match implementation

	// Expectations
	mockRepo.EXPECT().
		CreateOTP(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, otp *models.OTP) error {
			assert.Equal(t, formattedMSISDN, otp.MSISDN, "MSISDN should be formatted")
			// Just to make the test pass - the implementation will use the last 4 digits
			return nil
		})

	// Create usecase with mocked dependencies and test configuration
	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "nebengjek-test",
		},
	}
	uc := NewUserUC(mockRepo, mockGW, cfg)

	// Act
	err := uc.GenerateOTP(context.Background(), msisdn)

	// Assert
	assert.NoError(t, err)
}

func TestGenerateOTP_InvalidMSISDN(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	// Test data
	invalidMSISDN := "12345" // Invalid MSISDN

	// Create usecase with mocked dependencies
	cfg := &models.Config{}
	uc := NewUserUC(mockRepo, mockGW, cfg)

	// Act
	err := uc.GenerateOTP(context.Background(), invalidMSISDN)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid MSISDN format")
}

func TestGenerateOTP_CreateOTPError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	// Test data
	msisdn := "081234567890"
	formattedMSISDN := "6281234567890"
	expectedError := errors.New("database connection error")

	// Expectations
	mockRepo.EXPECT().
		CreateOTP(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, otp *models.OTP) error {
			assert.Equal(t, formattedMSISDN, otp.MSISDN)
			return expectedError
		})

	// Create usecase with mocked dependencies
	cfg := &models.Config{}
	uc := NewUserUC(mockRepo, mockGW, cfg)

	// Act
	err := uc.GenerateOTP(context.Background(), msisdn)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create OTP")
}

func TestVerifyOTP_Success_ExistingUser(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	// Test data
	msisdn := "081234567890"
	formattedMSISDN := "6281234567890" // Corrected: Added trailing zero to match implementation
	code := "1234"
	userID := uuid.New()
	user := &models.User{
		ID:        userID,
		MSISDN:    formattedMSISDN,
		Role:      "passenger",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		IsActive:  true,
	}
	otp := &models.OTP{
		ID:     uuid.New().String(),
		MSISDN: formattedMSISDN,
		Code:   code,
	}

	// Expectations
	mockRepo.EXPECT().
		GetOTP(gomock.Any(), formattedMSISDN, code).
		Return(otp, nil)

	mockRepo.EXPECT().
		GetUserByMSISDN(gomock.Any(), formattedMSISDN).
		Return(user, nil)

	mockRepo.EXPECT().
		MarkOTPVerified(gomock.Any(), formattedMSISDN, code).
		Return(nil)

	// Create usecase with mocked dependencies and test configuration
	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "nebengjek-test",
		},
	}
	uc := NewUserUC(mockRepo, mockGW, cfg)

	// Act
	response, err := uc.VerifyOTP(context.Background(), msisdn, code)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, userID.String(), response.UserID)
	assert.Equal(t, "passenger", response.Role)
	assert.Greater(t, response.ExpiresAt, int64(0))
}

func TestVerifyOTP_Success_NewUser(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	// Test data
	msisdn := "081234567890"
	formattedMSISDN := "6281234567890" // Corrected: Added trailing zero to match implementation
	code := "1234"
	otp := &models.OTP{
		ID:     uuid.New().String(),
		MSISDN: formattedMSISDN,
		Code:   code,
	}

	// Expectations
	mockRepo.EXPECT().
		GetOTP(gomock.Any(), formattedMSISDN, code).
		Return(otp, nil)

	mockRepo.EXPECT().
		GetUserByMSISDN(gomock.Any(), formattedMSISDN).
		Return(nil, errors.New("user not found"))

	mockRepo.EXPECT().
		CreateUser(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, user *models.User) error {
			assert.Equal(t, formattedMSISDN, user.MSISDN)
			assert.Equal(t, "passenger", user.Role)
			assert.True(t, user.IsActive)
			return nil
		})

	mockRepo.EXPECT().
		MarkOTPVerified(gomock.Any(), formattedMSISDN, code).
		Return(nil)

	// Create usecase with mocked dependencies and test configuration
	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "nebengjek-test",
		},
	}
	uc := NewUserUC(mockRepo, mockGW, cfg)

	// Act
	response, err := uc.VerifyOTP(context.Background(), msisdn, code)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, response.Token)
	assert.NotEmpty(t, response.UserID)
	assert.Equal(t, "passenger", response.Role)
	assert.Greater(t, response.ExpiresAt, int64(0))
}

func TestVerifyOTP_InvalidMSISDN(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	// Test data
	invalidMSISDN := "12345"
	code := "1234"

	// Create usecase with mocked dependencies
	cfg := &models.Config{}
	uc := NewUserUC(mockRepo, mockGW, cfg)

	// Act
	response, err := uc.VerifyOTP(context.Background(), invalidMSISDN, code)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "invalid MSISDN format")
}

func TestVerifyOTP_InvalidOTP(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	// Test data
	msisdn := "081234567890"
	formattedMSISDN := "6281234567890" // Corrected: Added trailing zero to match implementation
	code := "1234"

	// Expectations
	mockRepo.EXPECT().
		GetOTP(gomock.Any(), formattedMSISDN, code).
		Return(nil, errors.New("OTP not found"))

	// Create usecase with mocked dependencies
	cfg := &models.Config{}
	uc := NewUserUC(mockRepo, mockGW, cfg)

	// Act
	response, err := uc.VerifyOTP(context.Background(), msisdn, code)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "invalid OTP")
}

func TestVerifyOTP_NilOTP(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	// Test data
	msisdn := "081234567890"
	formattedMSISDN := "6281234567890"
	code := "1234"

	// Expectations
	mockRepo.EXPECT().
		GetOTP(gomock.Any(), formattedMSISDN, code).
		Return(nil, nil) // OTP not found, but no error

	// Create usecase with mocked dependencies
	cfg := &models.Config{}
	uc := NewUserUC(mockRepo, mockGW, cfg)

	// Act
	response, err := uc.VerifyOTP(context.Background(), msisdn, code)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "OTP not found or expired")
}

func TestVerifyOTP_OTPCodeMismatch(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	// Test data
	msisdn := "081234567890"
	formattedMSISDN := "6281234567890"
	code := "1234"
	otp := &models.OTP{
		ID:     uuid.New().String(),
		MSISDN: formattedMSISDN,
		Code:   "5678", // Different code
	}

	// Expectations
	mockRepo.EXPECT().
		GetOTP(gomock.Any(), formattedMSISDN, code).
		Return(otp, nil)

	// Create usecase with mocked dependencies
	cfg := &models.Config{}
	uc := NewUserUC(mockRepo, mockGW, cfg)

	// Act
	response, err := uc.VerifyOTP(context.Background(), msisdn, code)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "invalid OTP code")
}

func TestVerifyOTP_CreateUserError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	// Test data
	msisdn := "081234567890"
	formattedMSISDN := "6281234567890"
	code := "1234"
	otp := &models.OTP{
		ID:     uuid.New().String(),
		MSISDN: formattedMSISDN,
		Code:   code,
	}
	expectedError := errors.New("database error")

	// Expectations
	mockRepo.EXPECT().
		GetOTP(gomock.Any(), formattedMSISDN, code).
		Return(otp, nil)

	mockRepo.EXPECT().
		GetUserByMSISDN(gomock.Any(), formattedMSISDN).
		Return(nil, errors.New("user not found"))

	mockRepo.EXPECT().
		CreateUser(gomock.Any(), gomock.Any()).
		Return(expectedError)

	// Create usecase with mocked dependencies
	cfg := &models.Config{}
	uc := NewUserUC(mockRepo, mockGW, cfg)

	// Act
	response, err := uc.VerifyOTP(context.Background(), msisdn, code)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to create user")
}

func TestVerifyOTP_MarkOTPVerifiedError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	mockGW := mocks.NewMockUserGW(ctrl)

	// Test data
	msisdn := "081234567890"
	formattedMSISDN := "6281234567890" // Corrected: Added trailing zero to match implementation
	code := "1234"
	userID := uuid.New()
	user := &models.User{
		ID:        userID,
		MSISDN:    formattedMSISDN,
		Role:      "passenger",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		IsActive:  true,
	}
	otp := &models.OTP{
		ID:     uuid.New().String(),
		MSISDN: formattedMSISDN,
		Code:   code,
	}

	// Expectations
	mockRepo.EXPECT().
		GetOTP(gomock.Any(), formattedMSISDN, code).
		Return(otp, nil)

	mockRepo.EXPECT().
		GetUserByMSISDN(gomock.Any(), formattedMSISDN).
		Return(user, nil)

	mockRepo.EXPECT().
		MarkOTPVerified(gomock.Any(), formattedMSISDN, code).
		Return(errors.New("failed to mark OTP verified"))

	// Create usecase with mocked dependencies and test configuration
	cfg := &models.Config{
		JWT: models.JWTConfig{
			Secret:     "test-secret",
			Expiration: 60,
			Issuer:     "nebengjek-test",
		},
	}
	uc := NewUserUC(mockRepo, mockGW, cfg)

	// Act
	response, err := uc.VerifyOTP(context.Background(), msisdn, code)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to mark OTP as verified")
}
