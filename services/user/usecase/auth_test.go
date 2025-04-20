package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock User Repository
type MockAuthUserRepo struct {
	mock.Mock
}

func (m *MockAuthUserRepo) CreateUser(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockAuthUserRepo) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthUserRepo) GetUserByMSISDN(ctx context.Context, msisdn string) (*models.User, error) {
	args := m.Called(ctx, msisdn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthUserRepo) UpdateToDriver(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockAuthUserRepo) CreateOTP(ctx context.Context, otp *models.OTP) error {
	args := m.Called(ctx, otp)
	return args.Error(0)
}

func (m *MockAuthUserRepo) GetOTP(ctx context.Context, msisdn, code string) (*models.OTP, error) {
	args := m.Called(ctx, msisdn, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OTP), args.Error(1)
}

func (m *MockAuthUserRepo) MarkOTPVerified(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Added missing methods to match interface
func (m *MockAuthUserRepo) UpdateUser(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockAuthUserRepo) ListUsers(ctx context.Context, offset, limit int) ([]*models.User, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

// Mock User Gateway
type MockAuthUserGW struct {
	mock.Mock
}

func (m *MockAuthUserGW) PublishBeaconEvent(ctx context.Context, beaconReq *models.BeaconRequest) error {
	args := m.Called(ctx, beaconReq)
	return args.Error(0)
}

func (m *MockAuthUserGW) MatchAccept(mp *models.MatchProposal) error {
	args := m.Called(mp)
	return args.Error(0)
}

func (m *MockAuthUserGW) PublishLocationUpdate(ctx context.Context, location *models.LocationUpdate) error {
	args := m.Called(ctx, location)
	return args.Error(0)
}

func (m *MockAuthUserGW) PublishRideArrived(ctx context.Context, event *models.RideCompleteEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func TestGenerateOTP_Success(t *testing.T) {
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

	msisdn := "+628123456789" // Valid Telkomsel number format

	// Match any OTP object with the correct MSISDN
	mockRepo.On("CreateOTP", ctx, mock.MatchedBy(func(otp *models.OTP) bool {
		return otp.MSISDN == msisdn && otp.Code != ""
	})).Return(nil)

	// Act
	err := uc.GenerateOTP(ctx, msisdn)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestGenerateOTP_InvalidMSISDN(t *testing.T) {
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
		name   string
		msisdn string
	}{
		{
			name:   "Empty MSISDN",
			msisdn: "",
		},
		{
			name:   "Invalid Format",
			msisdn: "12345",
		},
		{
			name:   "Non-Telkomsel Number",
			msisdn: "+6281987654321", // Assuming this format is rejected by validator
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			err := uc.GenerateOTP(ctx, tc.msisdn)

			// Assert
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid MSISDN format or not a Telkomsel number")
		})
	}
}

func TestGenerateOTP_DatabaseError(t *testing.T) {
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

	msisdn := "+628123456789"
	expectedError := errors.New("database error")

	mockRepo.On("CreateOTP", ctx, mock.MatchedBy(func(otp *models.OTP) bool {
		return otp.MSISDN == msisdn && otp.Code != ""
	})).Return(expectedError)

	// Act
	err := uc.GenerateOTP(ctx, msisdn)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create OTP")
	mockRepo.AssertExpectations(t)
}

func TestVerifyOTP_Success_ExistingUser(t *testing.T) {
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

	msisdn := "+628123456789"
	code := "123456"
	userID := uuid.New()

	otp := &models.OTP{
		ID:     uuid.New().String(),
		MSISDN: msisdn,
		Code:   code,
	}

	user := &models.User{
		ID:       userID,
		MSISDN:   msisdn,
		FullName: "Test User",
		Role:     "passenger",
		IsActive: true,
	}

	mockRepo.On("GetOTP", ctx, msisdn, code).Return(otp, nil)
	mockRepo.On("GetUserByMSISDN", ctx, msisdn).Return(user, nil)
	mockRepo.On("MarkOTPVerified", ctx, msisdn, code).Return(nil)

	// Act
	result, err := uc.VerifyOTP(ctx, msisdn, code)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, userID.String(), result.UserID)
	assert.Equal(t, "passenger", result.Role)
	assert.NotEmpty(t, result.Token)
	mockRepo.AssertExpectations(t)
}

func TestVerifyOTP_Success_NewUser(t *testing.T) {
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

	msisdn := "+628123456789"
	code := "123456"

	otp := &models.OTP{
		ID:     uuid.New().String(),
		MSISDN: msisdn,
		Code:   code,
	}

	// User not found in database
	userNotFoundError := errors.New("user not found")

	mockRepo.On("GetOTP", ctx, msisdn, code).Return(otp, nil)
	mockRepo.On("GetUserByMSISDN", ctx, msisdn).Return(nil, userNotFoundError)

	// Should create a new user
	mockRepo.On("CreateUser", ctx, mock.MatchedBy(func(u *models.User) bool {
		return u.MSISDN == msisdn && u.Role == "passenger" && u.IsActive == true
	})).Return(nil)

	mockRepo.On("MarkOTPVerified", ctx, msisdn, code).Return(nil)

	// Act
	result, err := uc.VerifyOTP(ctx, msisdn, code)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "passenger", result.Role)
	assert.NotEmpty(t, result.Token)
	mockRepo.AssertExpectations(t)
}

func TestVerifyOTP_InvalidMSISDN(t *testing.T) {
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

	// Act
	result, err := uc.VerifyOTP(ctx, "invalid", "123456")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid MSISDN format or not a Telkomsel number")
}

func TestVerifyOTP_InvalidOTP(t *testing.T) {
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

	msisdn := "+628123456789"
	code := "123456"

	mockRepo.On("GetOTP", ctx, msisdn, code).Return(nil, errors.New("OTP not found"))

	// Act
	result, err := uc.VerifyOTP(ctx, msisdn, code)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid OTP")
	mockRepo.AssertExpectations(t)
}

func TestVerifyOTP_NewUserCreationError(t *testing.T) {
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

	msisdn := "+628123456789"
	code := "123456"

	otp := &models.OTP{
		ID:     uuid.New().String(),
		MSISDN: msisdn,
		Code:   code,
	}

	// User not found in database
	userNotFoundError := errors.New("user not found")
	userCreationError := errors.New("failed to create user")

	mockRepo.On("GetOTP", ctx, msisdn, code).Return(otp, nil)
	mockRepo.On("GetUserByMSISDN", ctx, msisdn).Return(nil, userNotFoundError)

	// Creation fails
	mockRepo.On("CreateUser", ctx, mock.MatchedBy(func(u *models.User) bool {
		return u.MSISDN == msisdn && u.Role == "passenger"
	})).Return(userCreationError)

	// Act
	result, err := uc.VerifyOTP(ctx, msisdn, code)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create user")
	mockRepo.AssertExpectations(t)
}

func TestVerifyOTP_TokenGenerationError(t *testing.T) {
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
	// This test is a bit tricky since we can't easily mock the JWT token generation
	// In a real test, you might inject a token generator or use a test helper
	// For now, we'll skip this test
	t.Skip("Skipping token generation error test")
}

func TestVerifyOTP_MarkOTPVerifiedError(t *testing.T) {
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

	msisdn := "+628123456789"
	code := "123456"
	userID := uuid.New()

	otp := &models.OTP{
		ID:     uuid.New().String(),
		MSISDN: msisdn,
		Code:   code,
	}

	user := &models.User{
		ID:       userID,
		MSISDN:   msisdn,
		FullName: "Test User",
		Role:     "passenger",
		IsActive: true,
	}

	markOTPError := errors.New("failed to mark OTP as verified")

	mockRepo.On("GetOTP", ctx, msisdn, code).Return(otp, nil)
	mockRepo.On("GetUserByMSISDN", ctx, msisdn).Return(user, nil)
	mockRepo.On("MarkOTPVerified", ctx, msisdn, code).Return(markOTPError)

	// Act
	result, err := uc.VerifyOTP(ctx, msisdn, code)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to mark OTP as verified")
	mockRepo.AssertExpectations(t)
}

func TestGenerateJWTToken(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepo)
	mockGW := new(MockUserGW)

	jwtConfig := models.JWTConfig{
		Secret:     "test-secret",
		Expiration: 60, // 60 minutes
		Issuer:     "test-issuer",
	}

	uc := NewUserUC(mockRepo, mockGW, jwtConfig)

	userID := uuid.New()
	user := &models.User{
		ID:       userID,
		MSISDN:   "+628123456789",
		FullName: "Test User",
		Role:     "passenger",
		IsActive: true,
	}

	// Act
	token, expiresAt, err := uc.generateJWTToken(user)

	// Assert
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Validate token expiration time
	now := time.Now().Unix()
	assert.Greater(t, expiresAt, now)
	assert.LessOrEqual(t, expiresAt, now+(60*60)) // Should expire in 60 minutes or less

	// Validate token content
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtConfig.Secret), nil
	})
	assert.NoError(t, err)
	assert.True(t, parsedToken.Valid)

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	assert.True(t, ok)
	assert.Equal(t, userID.String(), claims["user_id"])
	assert.Equal(t, "+628123456789", claims["msisdn"])
	assert.Equal(t, "passenger", claims["role"])
	assert.Equal(t, "test-issuer", claims["iss"])
	assert.Equal(t, float64(expiresAt), claims["exp"])
}
