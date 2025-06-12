package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/location/mocks"
	"github.com/stretchr/testify/assert"
)

func TestNewLocationHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUC := mocks.NewMockLocationUC(ctrl)
	handler := NewLocationHandler(mockUC)

	assert.NotNil(t, handler)
	assert.Equal(t, mockUC, handler.locationUC)
}

func TestLocationHandler_AddAvailableDriver(t *testing.T) {
	tests := []struct {
		name           string
		driverID       string
		requestBody    interface{}
		mockSetup      func(*mocks.MockLocationUC)
		expectedStatus int
		expectedError  bool
	}{
		{
			name:     "Success",
			driverID: "driver-123",
			requestBody: map[string]interface{}{
				"location": map[string]interface{}{
					"latitude":  -6.175392,
					"longitude": 106.827153,
				},
			},
			mockSetup: func(mockUC *mocks.MockLocationUC) {
				mockUC.EXPECT().
					AddAvailableDriver(gomock.Any(), "driver-123", gomock.Any()).
					Return(nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:     "Missing driver ID",
			driverID: "",
			requestBody: map[string]interface{}{
				"location": map[string]interface{}{
					"latitude":  -6.175392,
					"longitude": 106.827153,
				},
			},
			mockSetup: func(mockUC *mocks.MockLocationUC) {
				// No expectations - should not call usecase
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  false,
		},
		{
			name:           "Invalid request body",
			driverID:       "driver-123",
			requestBody:    "invalid json",
			mockSetup: func(mockUC *mocks.MockLocationUC) {
				// No expectations - should not call usecase
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  false,
		},
		{
			name:     "Usecase error",
			driverID: "driver-123",
			requestBody: map[string]interface{}{
				"location": map[string]interface{}{
					"latitude":  -6.175392,
					"longitude": 106.827153,
				},
			},
			mockSetup: func(mockUC *mocks.MockLocationUC) {
				mockUC.EXPECT().
					AddAvailableDriver(gomock.Any(), "driver-123", gomock.Any()).
					Return(errors.New("database error")).
					Times(1)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUC := mocks.NewMockLocationUC(ctrl)
			tt.mockSetup(mockUC)

			handler := NewLocationHandler(mockUC)

			e := echo.New()
			
			var reqBody []byte
			if tt.requestBody != nil {
				reqBody, _ = json.Marshal(tt.requestBody)
			}
			
			req := httptest.NewRequest(http.MethodPost, "/drivers/:id/available", bytes.NewBuffer(reqBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.driverID)

			err := handler.AddAvailableDriver(c)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestLocationHandler_RemoveAvailableDriver(t *testing.T) {
	tests := []struct {
		name           string
		driverID       string
		mockSetup      func(*mocks.MockLocationUC)
		expectedStatus int
		expectedError  bool
	}{
		{
			name:     "Success",
			driverID: "driver-123",
			mockSetup: func(mockUC *mocks.MockLocationUC) {
				mockUC.EXPECT().
					RemoveAvailableDriver(gomock.Any(), "driver-123").
					Return(nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:     "Missing driver ID",
			driverID: "",
			mockSetup: func(mockUC *mocks.MockLocationUC) {
				// No expectations - should not call usecase
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  false,
		},
		{
			name:     "Usecase error",
			driverID: "driver-123",
			mockSetup: func(mockUC *mocks.MockLocationUC) {
				mockUC.EXPECT().
					RemoveAvailableDriver(gomock.Any(), "driver-123").
					Return(errors.New("redis error")).
					Times(1)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUC := mocks.NewMockLocationUC(ctrl)
			tt.mockSetup(mockUC)

			handler := NewLocationHandler(mockUC)

			e := echo.New()
			req := httptest.NewRequest(http.MethodDelete, "/drivers/:id/available", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.driverID)

			err := handler.RemoveAvailableDriver(c)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestLocationHandler_AddAvailablePassenger(t *testing.T) {
	tests := []struct {
		name           string
		passengerID    string
		requestBody    interface{}
		mockSetup      func(*mocks.MockLocationUC)
		expectedStatus int
		expectedError  bool
	}{
		{
			name:        "Success",
			passengerID: "passenger-123",
			requestBody: map[string]interface{}{
				"location": map[string]interface{}{
					"latitude":  -6.175392,
					"longitude": 106.827153,
				},
			},
			mockSetup: func(mockUC *mocks.MockLocationUC) {
				mockUC.EXPECT().
					AddAvailablePassenger(gomock.Any(), "passenger-123", gomock.Any()).
					Return(nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:        "Missing passenger ID",
			passengerID: "",
			requestBody: map[string]interface{}{
				"location": map[string]interface{}{
					"latitude":  -6.175392,
					"longitude": 106.827153,
				},
			},
			mockSetup: func(mockUC *mocks.MockLocationUC) {
				// No expectations - should not call usecase
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  false,
		},
		{
			name:        "Usecase error",
			passengerID: "passenger-123",
			requestBody: map[string]interface{}{
				"location": map[string]interface{}{
					"latitude":  -6.175392,
					"longitude": 106.827153,
				},
			},
			mockSetup: func(mockUC *mocks.MockLocationUC) {
				mockUC.EXPECT().
					AddAvailablePassenger(gomock.Any(), "passenger-123", gomock.Any()).
					Return(errors.New("redis error")).
					Times(1)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUC := mocks.NewMockLocationUC(ctrl)
			tt.mockSetup(mockUC)

			handler := NewLocationHandler(mockUC)

			e := echo.New()
			
			var reqBody []byte
			if tt.requestBody != nil {
				reqBody, _ = json.Marshal(tt.requestBody)
			}
			
			req := httptest.NewRequest(http.MethodPost, "/passengers/:id/available", bytes.NewBuffer(reqBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.passengerID)

			err := handler.AddAvailablePassenger(c)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestLocationHandler_RemoveAvailablePassenger(t *testing.T) {
	tests := []struct {
		name           string
		passengerID    string
		mockSetup      func(*mocks.MockLocationUC)
		expectedStatus int
		expectedError  bool
	}{
		{
			name:        "Success",
			passengerID: "passenger-123",
			mockSetup: func(mockUC *mocks.MockLocationUC) {
				mockUC.EXPECT().
					RemoveAvailablePassenger(gomock.Any(), "passenger-123").
					Return(nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:        "Missing passenger ID",
			passengerID: "",
			mockSetup: func(mockUC *mocks.MockLocationUC) {
				// No expectations - should not call usecase
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  false,
		},
		{
			name:        "Usecase error",
			passengerID: "passenger-123",
			mockSetup: func(mockUC *mocks.MockLocationUC) {
				mockUC.EXPECT().
					RemoveAvailablePassenger(gomock.Any(), "passenger-123").
					Return(errors.New("redis error")).
					Times(1)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUC := mocks.NewMockLocationUC(ctrl)
			tt.mockSetup(mockUC)

			handler := NewLocationHandler(mockUC)

			e := echo.New()
			req := httptest.NewRequest(http.MethodDelete, "/passengers/:id/available", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.passengerID)

			err := handler.RemoveAvailablePassenger(c)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestLocationHandler_FindNearbyDrivers(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    map[string]string
		mockSetup      func(*mocks.MockLocationUC)
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "Success",
			queryParams: map[string]string{
				"lat":    "-6.175392",
				"lng":    "106.827153",
				"radius": "5",
			},
			mockSetup: func(mockUC *mocks.MockLocationUC) {
				mockUC.EXPECT().
					FindNearbyDrivers(gomock.Any(), gomock.Any(), float64(5)).
					Return([]*models.NearbyUser{
						{ID: "driver-1", Distance: 1.5},
						{ID: "driver-2", Distance: 3.2},
					}, nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name: "Missing latitude",
			queryParams: map[string]string{
				"longitude": "106.827153",
				"radius":    "5",
			},
			mockSetup: func(mockUC *mocks.MockLocationUC) {
				// No expectations - should not call usecase
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  false,
		},
		{
			name: "Invalid latitude",
			queryParams: map[string]string{
				"latitude":  "invalid",
				"longitude": "106.827153",
				"radius":    "5",
			},
			mockSetup: func(mockUC *mocks.MockLocationUC) {
				// No expectations - should not call usecase
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  false,
		},
		{
			name: "Usecase error",
			queryParams: map[string]string{
				"lat":    "-6.175392",
				"lng":    "106.827153",
				"radius": "5",
			},
			mockSetup: func(mockUC *mocks.MockLocationUC) {
				mockUC.EXPECT().
					FindNearbyDrivers(gomock.Any(), gomock.Any(), float64(5)).
					Return(nil, errors.New("redis error")).
					Times(1)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUC := mocks.NewMockLocationUC(ctrl)
			tt.mockSetup(mockUC)

			handler := NewLocationHandler(mockUC)

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/drivers/nearby", nil)
			
			// Add query parameters
			q := req.URL.Query()
			for key, value := range tt.queryParams {
				q.Add(key, value)
			}
			req.URL.RawQuery = q.Encode()
			
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.FindNearbyDrivers(c)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
				
				// Verify response structure for success case
				if tt.expectedStatus == http.StatusOK {
					var response map[string]interface{}
					err = json.Unmarshal(rec.Body.Bytes(), &response)
					assert.NoError(t, err)
					assert.True(t, response["success"].(bool))
					assert.NotNil(t, response["data"])
				}
			}
		})
	}
}