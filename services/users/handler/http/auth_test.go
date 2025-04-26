package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/users/mocks"
	"github.com/stretchr/testify/assert"
)

func TestGenerateOTP_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	authHandler := NewAuthHandler(mockUserUC)

	// Setup Echo context
	e := echo.New()
	requestBody := `{"msisdn": "+6281234567890"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/otp/generate", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Expect the mock to be called with the correct MSISDN
	mockUserUC.EXPECT().
		GenerateOTP(gomock.Any(), "+6281234567890").
		Return(nil)

	// Act
	err := authHandler.GenerateOTP(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify response body contains success message
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, true, response["success"])
	assert.Equal(t, "OTP sent successfully", response["message"])
}

func TestGenerateOTP_EmptyMSISDN(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	authHandler := NewAuthHandler(mockUserUC)

	// Setup Echo context with empty MSISDN
	e := echo.New()
	requestBody := `{"msisdn": ""}`
	req := httptest.NewRequest(http.MethodPost, "/auth/otp/generate", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Act
	err := authHandler.GenerateOTP(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Verify response body contains error message
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, false, response["success"])
	assert.Equal(t, "MSISDN is required", response["error"])
	assert.Equal(t, float64(http.StatusBadRequest), response["code"])
}

func TestGenerateOTP_InvalidPayload(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	authHandler := NewAuthHandler(mockUserUC)

	// Setup Echo context with invalid JSON
	e := echo.New()
	requestBody := `{invalid_json}`
	req := httptest.NewRequest(http.MethodPost, "/auth/otp/generate", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Act
	err := authHandler.GenerateOTP(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Verify response body contains error message
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, false, response["success"])
	assert.Equal(t, "Invalid request payload", response["error"])
	assert.Equal(t, float64(http.StatusBadRequest), response["code"])
}

func TestGenerateOTP_UseCaseError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	authHandler := NewAuthHandler(mockUserUC)

	// Setup Echo context
	e := echo.New()
	requestBody := `{"msisdn": "+6281234567890"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/otp/generate", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Expect the mock to be called and return an error
	mockUserUC.EXPECT().
		GenerateOTP(gomock.Any(), "+6281234567890").
		Return(errors.New("failed to generate OTP"))

	// Act
	err := authHandler.GenerateOTP(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	// Verify response body contains error message
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, false, response["success"])
	assert.Equal(t, "failed to generate OTP", response["error"])
	assert.Equal(t, float64(http.StatusInternalServerError), response["code"])
}

func TestVerifyOTP_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	authHandler := NewAuthHandler(mockUserUC)

	// Setup Echo context
	e := echo.New()
	requestBody := `{"msisdn": "+6281234567890", "otp": "1234"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/otp/verify", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	userID := uuid.New().String()
	// Mock response from use case
	authResponse := &models.AuthResponse{
		Token:     "jwt-token",
		UserID:    userID,
		Role:      "passenger",
		ExpiresAt: 1677729600, // Example timestamp
	}

	// Expect the mock to be called with the correct parameters
	mockUserUC.EXPECT().
		VerifyOTP(gomock.Any(), "+6281234567890", "1234").
		Return(authResponse, nil)

	// Act
	err := authHandler.VerifyOTP(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify response body contains token
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, true, response["success"])
	assert.Equal(t, "OTP verified successfully", response["message"])

	data, ok := response["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "jwt-token", data["token"])
	assert.Equal(t, userID, data["user_id"])
	assert.Equal(t, "passenger", data["role"])
	assert.Equal(t, float64(1677729600), data["expires_at"])
}

func TestVerifyOTP_MissingParameters(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	authHandler := NewAuthHandler(mockUserUC)

	// Test cases
	testCases := []struct {
		name        string
		requestBody string
	}{
		{
			name:        "Empty MSISDN",
			requestBody: `{"msisdn": "", "otp": "1234"}`,
		},
		{
			name:        "Empty OTP",
			requestBody: `{"msisdn": "+6281234567890", "otp": ""}`,
		},
		{
			name:        "Both Empty",
			requestBody: `{"msisdn": "", "otp": ""}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup Echo context
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/auth/otp/verify", strings.NewReader(tc.requestBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Act
			err := authHandler.VerifyOTP(c)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, rec.Code)

			// Verify response body contains error message
			var response map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, false, response["success"])
			assert.Equal(t, "MSISDN and OTP are required", response["error"])
			assert.Equal(t, float64(http.StatusBadRequest), response["code"])
		})
	}
}

func TestVerifyOTP_InvalidPayload(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	authHandler := NewAuthHandler(mockUserUC)

	// Setup Echo context with invalid JSON
	e := echo.New()
	requestBody := `{invalid_json}`
	req := httptest.NewRequest(http.MethodPost, "/auth/otp/verify", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Act
	err := authHandler.VerifyOTP(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Verify response body contains error message
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, false, response["success"])
	assert.Equal(t, "Invalid request payload", response["error"])
	assert.Equal(t, float64(http.StatusBadRequest), response["code"])
}

func TestVerifyOTP_UseCaseError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	authHandler := NewAuthHandler(mockUserUC)

	// Setup Echo context
	e := echo.New()
	requestBody := `{"msisdn": "+6281234567890", "otp": "1234"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/otp/verify", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Expect the mock to be called and return an error
	mockUserUC.EXPECT().
		VerifyOTP(gomock.Any(), "+6281234567890", "1234").
		Return(nil, errors.New("invalid OTP code"))

	// Act
	err := authHandler.VerifyOTP(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// Verify response body contains error message
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, false, response["success"])
	assert.Equal(t, "Invalid OTP", response["error"])
	assert.Equal(t, float64(http.StatusUnauthorized), response["code"])
}
