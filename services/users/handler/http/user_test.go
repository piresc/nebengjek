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

func TestCreateUser_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	userHandler := NewUserHandler(mockUserUC)

	// Setup Echo context
	e := echo.New()
	userID := uuid.New()
	requestBody := `{
		"fullname": "John Doe",
		"msisdn": "+6281234567890",
		"role": "passenger"
	}`
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Setup user for the mock
	// The usecase will construct the user from the request body
	// and then register it via RegisterUser
	mockUserUC.EXPECT().
		RegisterUser(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ interface{}, user *models.User) error {
			// Verify the user details are correct
			assert.Equal(t, "John Doe", user.FullName)
			assert.Equal(t, "+6281234567890", user.MSISDN)
			assert.Equal(t, "passenger", user.Role)

			// Set an ID for the return value
			user.ID = userID
			return nil
		})

	// Act
	err := userHandler.CreateUser(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	// Verify response body
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, true, response["success"])
	assert.Equal(t, "User created successfully", response["message"])

	data, ok := response["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "John Doe", data["fullname"])
	assert.Equal(t, "+6281234567890", data["msisdn"])
	assert.Equal(t, "passenger", data["role"])
}

func TestCreateUser_InvalidPayload(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	userHandler := NewUserHandler(mockUserUC)

	// Setup Echo context with invalid JSON
	e := echo.New()
	requestBody := `{invalid_json}`
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Act
	err := userHandler.CreateUser(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Verify response body
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, false, response["success"])
	assert.Equal(t, "Invalid request payload", response["error"])
	assert.Equal(t, float64(http.StatusBadRequest), response["code"])
}

func TestCreateUser_UseCaseError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	userHandler := NewUserHandler(mockUserUC)

	// Setup Echo context
	e := echo.New()
	requestBody := `{
		"fullname": "John Doe",
		"msisdn": "+6281234567890",
		"role": "passenger"
	}`
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Mock usecase to return an error
	mockUserUC.EXPECT().
		RegisterUser(gomock.Any(), gomock.Any()).
		Return(errors.New("database error"))

	// Act
	err := userHandler.CreateUser(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	// Verify response body
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, false, response["success"])
	assert.Equal(t, "Failed to create user", response["error"])
	assert.Equal(t, float64(http.StatusInternalServerError), response["code"])
}

func TestGetUser_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	userHandler := NewUserHandler(mockUserUC)

	// Setup Echo context
	e := echo.New()
	userID := uuid.New()
	userIDStr := userID.String()
	req := httptest.NewRequest(http.MethodGet, "/users/"+userIDStr, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(userIDStr)

	// Setup mock user object to return
	mockUser := &models.User{
		ID:       userID,
		MSISDN:   "+6281234567890",
		FullName: "John Doe",
		Role:     "passenger",
		IsActive: true,
	}

	// Expect the mock to be called with the correct user ID
	mockUserUC.EXPECT().
		GetUserByID(gomock.Any(), userIDStr).
		Return(mockUser, nil)

	// Act
	err := userHandler.GetUser(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify response body
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, true, response["success"])
	assert.Equal(t, "User retrieved successfully", response["message"])

	data, ok := response["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, userIDStr, data["id"])
	assert.Equal(t, "John Doe", data["fullname"])
	assert.Equal(t, "+6281234567890", data["msisdn"])
	assert.Equal(t, "passenger", data["role"])
	assert.Equal(t, true, data["is_active"])
}

func TestGetUser_MissingID(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	userHandler := NewUserHandler(mockUserUC)

	// Setup Echo context without user ID parameter
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/users/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// Intentionally not setting the ID parameter

	// Act
	err := userHandler.GetUser(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Verify response body
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, false, response["success"])
	assert.Equal(t, "Invalid user ID", response["error"])
	assert.Equal(t, float64(http.StatusBadRequest), response["code"])
}

func TestGetUser_UseCaseError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	userHandler := NewUserHandler(mockUserUC)

	// Setup Echo context
	e := echo.New()
	userID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/users/"+userID, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(userID)

	// Expect the mock to be called and return an error
	mockUserUC.EXPECT().
		GetUserByID(gomock.Any(), userID).
		Return(nil, errors.New("user not found"))

	// Act
	err := userHandler.GetUser(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	// Verify response body
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, false, response["success"])
	assert.Equal(t, "Failed to retrieve user", response["error"])
	assert.Equal(t, float64(http.StatusInternalServerError), response["code"])
}

func TestRegisterDriver_Success(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	userHandler := NewUserHandler(mockUserUC)

	// Setup Echo context
	e := echo.New()
	userID := uuid.New()
	requestBody := `{
		"msisdn": "+6281234567890",
		"fullname": "John Driver",
		"role": "driver",
		"driver_info": {
			"vehicle_type": "car",
			"vehicle_plate": "B 1234 ABC"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/drivers/register", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Mock usecase to accept the registration
	mockUserUC.EXPECT().
		RegisterDriver(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ interface{}, user *models.User) error {
			// Verify the driver details are correct
			assert.Equal(t, "John Driver", user.FullName)
			assert.Equal(t, "+6281234567890", user.MSISDN)
			assert.Equal(t, "driver", user.Role)
			assert.NotNil(t, user.DriverInfo)
			assert.Equal(t, "car", user.DriverInfo.VehicleType)
			assert.Equal(t, "B 1234 ABC", user.DriverInfo.VehiclePlate)

			// Set an ID for the return value
			user.ID = userID
			return nil
		})

	// Act
	err := userHandler.RegisterDriver(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	// Verify response body
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, true, response["success"])
	assert.Equal(t, "Driver registered successfully", response["message"])

	data, ok := response["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "John Driver", data["fullname"])
	assert.Equal(t, "+6281234567890", data["msisdn"])
	assert.Equal(t, "driver", data["role"])

	driverInfo, ok := data["driver_info"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "car", driverInfo["vehicle_type"])
	assert.Equal(t, "B 1234 ABC", driverInfo["vehicle_plate"])
}

func TestRegisterDriver_InvalidPayload(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	userHandler := NewUserHandler(mockUserUC)

	// Setup Echo context with invalid JSON
	e := echo.New()
	requestBody := `{invalid_json}`
	req := httptest.NewRequest(http.MethodPost, "/drivers/register", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Act
	err := userHandler.RegisterDriver(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Verify response body
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, false, response["success"])
	assert.Equal(t, "Invalid request payload", response["error"])
	assert.Equal(t, float64(http.StatusBadRequest), response["code"])
}

func TestRegisterDriver_UseCaseError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := mocks.NewMockUserUC(ctrl)
	userHandler := NewUserHandler(mockUserUC)

	// Setup Echo context
	e := echo.New()
	requestBody := `{
		"msisdn": "+6281234567890",
		"fullname": "John Driver",
		"role": "driver",
		"driver_info": {
			"vehicle_type": "car",
			"vehicle_plate": "B 1234 ABC"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/drivers/register", strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Mock usecase to return an error
	mockUserUC.EXPECT().
		RegisterDriver(gomock.Any(), gomock.Any()).
		Return(errors.New("invalid vehicle information"))

	// Act
	err := userHandler.RegisterDriver(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	// Verify response body
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, false, response["success"])
	assert.Equal(t, "Failed to register driver", response["error"])
	assert.Equal(t, float64(http.StatusInternalServerError), response["code"])
}
