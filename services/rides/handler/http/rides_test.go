package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	ridemodels "github.com/piresc/nebengjek/internal/pkg/models/ride"
	locationmodels "github.com/piresc/nebengjek/internal/pkg/models/location"
	"github.com/piresc/nebengjek/services/rides/mocks"
	"github.com/stretchr/testify/assert"
)

func TestNewRidesHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	handler := NewRidesHandler(mockRideUC)

	assert.NotNil(t, handler)
	assert.Equal(t, mockRideUC, handler.rideUC)
}

func TestRidesHandler_StartRide_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	handler := NewRidesHandler(mockRideUC)

	rideID := uuid.New().String()
	req := ridemodels.RideStartRequest{
		RideID: rideID,
		DriverLocation: &locationmodels.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		PassengerLocation: &locationmodels.Location{
			Latitude:  -6.175400,
			Longitude: 106.827160,
		},
	}

	expectedResp := &ridemodels.Ride{
		RideID: uuid.MustParse(rideID),
		Status: ridemodels.RideStatusOngoing,
	}

	mockRideUC.EXPECT().
		StartRide(gomock.Any(), req).
		Return(expectedResp, nil).
		Times(1)

	e := echo.New()
	reqBody, _ := json.Marshal(map[string]interface{}{
		"driver_location": req.DriverLocation,
		"passenger_location": req.PassengerLocation,
	})
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(reqBody))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)
	c.SetParamNames("rideID")
	c.SetParamValues(rideID)

	err := handler.StartRide(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestRidesHandler_StartRide_MissingRideID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	handler := NewRidesHandler(mockRideUC)

	e := echo.New()
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)

	err := handler.StartRide(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestRidesHandler_StartRide_InvalidRequestBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	handler := NewRidesHandler(mockRideUC)

	rideID := uuid.New().String()

	e := echo.New()
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte("invalid json")))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)
	c.SetParamNames("rideID")
	c.SetParamValues(rideID)

	err := handler.StartRide(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestRidesHandler_StartRide_MissingLocations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	handler := NewRidesHandler(mockRideUC)

	rideID := uuid.New().String()

	testCases := []struct {
		name string
		body map[string]interface{}
	}{
		{
			name: "Missing driver location",
			body: map[string]interface{}{
				"passenger_location": map[string]float64{
					"latitude":  -6.175400,
					"longitude": 106.827160,
				},
			},
		},
		{
			name: "Missing passenger location",
			body: map[string]interface{}{
				"driver_location": map[string]float64{
					"latitude":  -6.175392,
					"longitude": 106.827153,
				},
			},
		},
		{
			name: "Missing both locations",
			body: map[string]interface{}{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			reqBody, _ := json.Marshal(tc.body)
			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(reqBody))
			request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			recorder := httptest.NewRecorder()
			c := e.NewContext(request, recorder)
			c.SetParamNames("rideID")
			c.SetParamValues(rideID)

			err := handler.StartRide(c)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, recorder.Code)
		})
	}
}

func TestRidesHandler_StartRide_UseCaseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	handler := NewRidesHandler(mockRideUC)

	rideID := uuid.New().String()
	req := ridemodels.RideStartRequest{
		RideID: rideID,
		DriverLocation: &locationmodels.Location{
			Latitude:  -6.175392,
			Longitude: 106.827153,
		},
		PassengerLocation: &locationmodels.Location{
			Latitude:  -6.175400,
			Longitude: 106.827160,
		},
	}

	mockRideUC.EXPECT().
		StartRide(gomock.Any(), req).
		Return(nil, errors.New("usecase error")).
		Times(1)

	e := echo.New()
	reqBody, _ := json.Marshal(map[string]interface{}{
		"driver_location": req.DriverLocation,
		"passenger_location": req.PassengerLocation,
	})
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(reqBody))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)
	c.SetParamNames("rideID")
	c.SetParamValues(rideID)

	err := handler.StartRide(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestRidesHandler_RideArrived_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	handler := NewRidesHandler(mockRideUC)

	rideID := uuid.New().String()
	req := ridemodels.RideArrivalReq{
		RideID:           rideID,
		AdjustmentFactor: 1.0,
	}

	expectedPayment := &ridemodels.PaymentRequest{
		RideID:    rideID,
		TotalCost: 100000,
	}

	mockRideUC.EXPECT().
		RideArrived(gomock.Any(), req).
		Return(expectedPayment, nil).
		Times(1)

	e := echo.New()
	reqBody, _ := json.Marshal(req)
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(reqBody))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)
	c.SetParamNames("rideID")
	c.SetParamValues(rideID)

	err := handler.RideArrived(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestRidesHandler_RideArrived_MissingRideID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	handler := NewRidesHandler(mockRideUC)

	e := echo.New()
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)

	err := handler.RideArrived(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestRidesHandler_RideArrived_InvalidRequestBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	handler := NewRidesHandler(mockRideUC)

	rideID := uuid.New().String()

	e := echo.New()
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte("invalid json")))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)
	c.SetParamNames("rideID")
	c.SetParamValues(rideID)

	err := handler.RideArrived(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestRidesHandler_RideArrived_UseCaseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	handler := NewRidesHandler(mockRideUC)

	rideID := uuid.New().String()
	req := ridemodels.RideArrivalReq{
		RideID:           rideID,
		AdjustmentFactor: 1.0,
	}

	mockRideUC.EXPECT().
		RideArrived(gomock.Any(), req).
		Return(nil, errors.New("usecase error")).
		Times(1)

	e := echo.New()
	reqBody, _ := json.Marshal(req)
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(reqBody))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)
	c.SetParamNames("rideID")
	c.SetParamValues(rideID)

	err := handler.RideArrived(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestRidesHandler_ProcessPayment_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	handler := NewRidesHandler(mockRideUC)

	rideID := uuid.New().String()
	req := ridemodels.PaymentProccessRequest{
		RideID:    rideID,
		TotalCost: 100000,
		Status:    ridemodels.PaymentStatusAccepted,
	}

	expectedPayment := &ridemodels.Payment{
		PaymentID:    uuid.New(),
		RideID:       uuid.MustParse(rideID),
		AdjustedCost: 100000,
		AdminFee:     7500,
		DriverPayout: 92500,
		Status:       ridemodels.PaymentStatusAccepted,
	}

	mockRideUC.EXPECT().
		ProcessPayment(gomock.Any(), req).
		Return(expectedPayment, nil).
		Times(1)

	e := echo.New()
	reqBody, _ := json.Marshal(map[string]interface{}{
		"total_cost": req.TotalCost,
		"status":     req.Status,
	})
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(reqBody))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)
	c.SetParamNames("rideID")
	c.SetParamValues(rideID)

	err := handler.ProcessPayment(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, recorder.Code)

	// Parse response body to verify payment details
	var response struct {
		Success bool            `json:"success"`
		Message string          `json:"message"`
		Data    ridemodels.Payment  `json:"data"`
	}
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, uuid.MustParse(rideID), response.Data.RideID)
	assert.Equal(t, 100000, response.Data.AdjustedCost)
	assert.Equal(t, ridemodels.PaymentStatusAccepted, response.Data.Status)
}

func TestRidesHandler_ProcessPayment_MissingRideID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	handler := NewRidesHandler(mockRideUC)

	e := echo.New()
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)

	err := handler.ProcessPayment(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestRidesHandler_ProcessPayment_InvalidRequestBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	handler := NewRidesHandler(mockRideUC)

	rideID := uuid.New().String()

	e := echo.New()
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte("invalid json")))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)
	c.SetParamNames("rideID")
	c.SetParamValues(rideID)

	err := handler.ProcessPayment(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestRidesHandler_ProcessPayment_UseCaseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRideUC := mocks.NewMockRideUC(ctrl)
	handler := NewRidesHandler(mockRideUC)

	rideID := uuid.New().String()
	req := ridemodels.PaymentProccessRequest{
		RideID:    rideID,
		TotalCost: 100000,
		Status:    ridemodels.PaymentStatusAccepted,
	}

	mockRideUC.EXPECT().
		ProcessPayment(gomock.Any(), req).
		Return(nil, errors.New("usecase error")).
		Times(1)

	e := echo.New()
	reqBody, _ := json.Marshal(map[string]interface{}{
		"total_cost": req.TotalCost,
		"status":     req.Status,
	})
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(reqBody))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)
	c.SetParamNames("rideID")
	c.SetParamValues(rideID)

	err := handler.ProcessPayment(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}