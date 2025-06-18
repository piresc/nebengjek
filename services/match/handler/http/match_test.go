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
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/match/mocks"
	"github.com/stretchr/testify/assert"
)

func TestNewMatchHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMatchUC := mocks.NewMockMatchUC(ctrl)
	handler := NewMatchHandler(mockMatchUC)

	assert.NotNil(t, handler)
	assert.Equal(t, mockMatchUC, handler.matchUC)
}

func TestMatchHandler_ConfirmMatch_Success_Accepted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMatchUC := mocks.NewMockMatchUC(ctrl)
	handler := NewMatchHandler(mockMatchUC)

	matchID := uuid.New().String()
	userID := uuid.New().String()
	req := models.MatchConfirmRequest{
		ID:     matchID,
		UserID: userID,
		Status: string(models.MatchStatusAccepted),
	}

	expectedResult := &models.MatchProposal{
		ID:          matchID,
		MatchStatus: models.MatchStatusAccepted,
	}

	mockMatchUC.EXPECT().
		ConfirmMatchStatus(gomock.Any(), &req).
		Return(*expectedResult, nil).
		Times(1)

	e := echo.New()
	reqBody, _ := json.Marshal(map[string]interface{}{
		"user_id": userID,
		"status":  string(models.MatchStatusAccepted),
	})
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(reqBody))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)
	c.SetParamNames("matchID")
	c.SetParamValues(matchID)

	err := handler.ConfirmMatch(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, recorder.Code)

	var response map[string]interface{}
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Match confirmation processed successfully", response["message"])
}

func TestMatchHandler_ConfirmMatch_Success_Rejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMatchUC := mocks.NewMockMatchUC(ctrl)
	handler := NewMatchHandler(mockMatchUC)

	matchID := uuid.New().String()
	userID := uuid.New().String()
	req := models.MatchConfirmRequest{
		ID:     matchID,
		UserID: userID,
		Status: string(models.MatchStatusRejected),
	}

	expectedResult := &models.MatchProposal{
		ID:          matchID,
		MatchStatus: models.MatchStatusRejected,
	}

	mockMatchUC.EXPECT().
		ConfirmMatchStatus(gomock.Any(), &req).
		Return(*expectedResult, nil).
		Times(1)

	e := echo.New()
	reqBody, _ := json.Marshal(map[string]interface{}{
		"user_id": userID,
		"status":  string(models.MatchStatusRejected),
	})
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(reqBody))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)
	c.SetParamNames("matchID")
	c.SetParamValues(matchID)

	err := handler.ConfirmMatch(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestMatchHandler_ConfirmMatch_MissingMatchID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMatchUC := mocks.NewMockMatchUC(ctrl)
	handler := NewMatchHandler(mockMatchUC)

	e := echo.New()
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)

	err := handler.ConfirmMatch(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	var response map[string]interface{}
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Match ID is required", response["error"])
}

func TestMatchHandler_ConfirmMatch_InvalidRequestBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMatchUC := mocks.NewMockMatchUC(ctrl)
	handler := NewMatchHandler(mockMatchUC)

	matchID := uuid.New().String()

	e := echo.New()
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte("invalid json")))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)
	c.SetParamNames("matchID")
	c.SetParamValues(matchID)

	err := handler.ConfirmMatch(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	var response map[string]interface{}
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Invalid request body")
}

func TestMatchHandler_ConfirmMatch_MissingUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMatchUC := mocks.NewMockMatchUC(ctrl)
	handler := NewMatchHandler(mockMatchUC)

	matchID := uuid.New().String()

	e := echo.New()
	reqBody, _ := json.Marshal(map[string]interface{}{
		"status": string(models.MatchStatusAccepted),
	})
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(reqBody))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)
	c.SetParamNames("matchID")
	c.SetParamValues(matchID)

	err := handler.ConfirmMatch(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	var response map[string]interface{}
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "User ID is required", response["error"])
}

func TestMatchHandler_ConfirmMatch_InvalidStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMatchUC := mocks.NewMockMatchUC(ctrl)
	handler := NewMatchHandler(mockMatchUC)

	matchID := uuid.New().String()
	userID := uuid.New().String()

	testCases := []struct {
		name   string
		status string
	}{
		{
			name:   "Invalid status - PENDING",
			status: "PENDING",
		},
		{
			name:   "Invalid status - COMPLETED",
			status: "COMPLETED",
		},
		{
			name:   "Invalid status - empty",
			status: "",
		},
		{
			name:   "Invalid status - random",
			status: "RANDOM_STATUS",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			reqBody, _ := json.Marshal(map[string]interface{}{
				"user_id": userID,
				"status":  tc.status,
			})
			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(reqBody))
			request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			recorder := httptest.NewRecorder()
			c := e.NewContext(request, recorder)
			c.SetParamNames("matchID")
			c.SetParamValues(matchID)

			err := handler.ConfirmMatch(c)

			assert.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, recorder.Code)

			var response map[string]interface{}
			err = json.Unmarshal(recorder.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, "Status must be either ACCEPTED or REJECTED", response["error"])
		})
	}
}

func TestMatchHandler_ConfirmMatch_UseCaseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMatchUC := mocks.NewMockMatchUC(ctrl)
	handler := NewMatchHandler(mockMatchUC)

	matchID := uuid.New().String()
	userID := uuid.New().String()
	req := models.MatchConfirmRequest{
		ID:     matchID,
		UserID: userID,
		Status: string(models.MatchStatusAccepted),
	}

	mockMatchUC.EXPECT().
		ConfirmMatchStatus(gomock.Any(), &req).
		Return(models.MatchProposal{}, errors.New("usecase error")).
		Times(1)

	e := echo.New()
	reqBody, _ := json.Marshal(map[string]interface{}{
		"user_id": userID,
		"status":  string(models.MatchStatusAccepted),
	})
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(reqBody))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)
	c.SetParamNames("matchID")
	c.SetParamValues(matchID)

	err := handler.ConfirmMatch(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)

	var response map[string]interface{}
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Failed to confirm match")
}