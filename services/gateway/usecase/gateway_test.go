package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
	gatewaymocks "github.com/piresc/nebengjek/services/gateway/mocks"
	usersmocks "github.com/piresc/nebengjek/services/users/mocks"
	"github.com/stretchr/testify/assert"
)

func TestNewGatewayUC(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := usersmocks.NewMockUserUC(ctrl)
	mockGatewayGW := gatewaymocks.NewMockGatewayGW(ctrl)

	gatewayUC := NewGatewayUC(mockUserUC, mockGatewayGW)

	assert.NotNil(t, gatewayUC)
	
	// Check that it implements the interface
	_, ok := gatewayUC.(*GatewayUseCase)
	assert.True(t, ok)
}

func TestGatewayUseCase_UpdateBeaconStatus(t *testing.T) {
	tests := []struct {
		name          string
		request       *models.BeaconRequest
		mockResponse  *models.ProxyResponse
		mockError     error
		expectedError string
	}{
		{
			name: "successful beacon update",
			request: &models.BeaconRequest{
				MSISDN:    "1234567890",
				IsActive:  true,
				Latitude:  -6.175392,
				Longitude: 106.827153,
			},
			mockResponse: &models.ProxyResponse{
				StatusCode: 200,
				Body:       []byte(`{"success": true}`),
			},
			expectedError: "",
		},
		{
			name: "users service returns error",
			request: &models.BeaconRequest{
				MSISDN:    "1234567890",
				IsActive:  true,
				Latitude:  -6.175392,
				Longitude: 106.827153,
			},
			mockError:     errors.New("service connection failed"),
			expectedError: "failed to update beacon status: service connection failed",
		},
		{
			name: "users service returns error status code",
			request: &models.BeaconRequest{
				MSISDN:    "1234567890",
				IsActive:  true,
				Latitude:  -6.175392,
				Longitude: 106.827153,
			},
			mockResponse: &models.ProxyResponse{
				StatusCode: 400,
				Body:       []byte(`{"error": "invalid request"}`),
			},
			expectedError: "users service returned error: {\"error\": \"invalid request\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserUC := usersmocks.NewMockUserUC(ctrl)
			mockGatewayGW := gatewaymocks.NewMockGatewayGW(ctrl)

			if tt.mockError != nil {
				mockGatewayGW.EXPECT().
					CallUsersService(gomock.Any(), "POST", "/beacon/update", tt.request, nil, nil).
					Return(nil, tt.mockError).
					Times(1)
			} else {
				mockGatewayGW.EXPECT().
					CallUsersService(gomock.Any(), "POST", "/beacon/update", tt.request, nil, nil).
					Return(tt.mockResponse, nil).
					Times(1)
			}

			gatewayUC := NewGatewayUC(mockUserUC, mockGatewayGW)

			err := gatewayUC.UpdateBeaconStatus(context.Background(), tt.request)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGatewayUseCase_UpdateFinderStatus(t *testing.T) {
	tests := []struct {
		name          string
		request       *models.FinderRequest
		mockResponse  *models.ProxyResponse
		mockError     error
		expectedError string
	}{
		{
			name: "successful finder update",
			request: &models.FinderRequest{
				MSISDN:   "1234567890",
				IsActive: true,
				Location: models.Location{
					Latitude:  -6.175392,
					Longitude: 106.827153,
				},
				TargetLocation: models.Location{
					Latitude:  -6.200000,
					Longitude: 106.850000,
				},
			},
			mockResponse: &models.ProxyResponse{
				StatusCode: 200,
				Body:       []byte(`{"success": true}`),
			},
			expectedError: "",
		},
		{
			name: "service error",
			request: &models.FinderRequest{
				MSISDN:   "1234567890",
				IsActive: true,
				Location: models.Location{
					Latitude:  -6.175392,
					Longitude: 106.827153,
				},
				TargetLocation: models.Location{
					Latitude:  -6.200000,
					Longitude: 106.850000,
				},
			},
			mockError:     errors.New("connection timeout"),
			expectedError: "failed to update finder status: connection timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserUC := usersmocks.NewMockUserUC(ctrl)
			mockGatewayGW := gatewaymocks.NewMockGatewayGW(ctrl)

			if tt.mockError != nil {
				mockGatewayGW.EXPECT().
					CallUsersService(gomock.Any(), "POST", "/finder/update", tt.request, nil, nil).
					Return(nil, tt.mockError).
					Times(1)
			} else {
				mockGatewayGW.EXPECT().
					CallUsersService(gomock.Any(), "POST", "/finder/update", tt.request, nil, nil).
					Return(tt.mockResponse, nil).
					Times(1)
			}

			gatewayUC := NewGatewayUC(mockUserUC, mockGatewayGW)

			err := gatewayUC.UpdateFinderStatus(context.Background(), tt.request)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGatewayUseCase_ConfirmMatch(t *testing.T) {
	matchID := uuid.New().String()
	userID := uuid.New().String()

	tests := []struct {
		name           string
		request        *models.MatchConfirmRequest
		mockResponse   *models.ProxyResponse
		mockError      error
		expectedResult *models.MatchProposal
		expectedError  string
	}{
		{
			name: "successful match confirmation",
			request: &models.MatchConfirmRequest{
				ID:     matchID,
				UserID: userID,
			},
			mockResponse: &models.ProxyResponse{
				StatusCode: 200,
				Body:       []byte(`{"match_id": "` + matchID + `", "driver_id": "driver-123", "passenger_id": "passenger-456"}`),
			},
			expectedResult: &models.MatchProposal{
				ID:          matchID,
				DriverID:    "driver-123",
				PassengerID: "passenger-456",
			},
			expectedError: "",
		},
		{
			name: "match service error",
			request: &models.MatchConfirmRequest{
				ID:     matchID,
				UserID: userID,
			},
			mockError:     errors.New("match not found"),
			expectedError: "failed to confirm match: match not found",
		},
		{
			name: "match service returns error status",
			request: &models.MatchConfirmRequest{
				ID:     matchID,
				UserID: userID,
			},
			mockResponse: &models.ProxyResponse{
				StatusCode: 404,
				Body:       []byte(`{"error": "match not found"}`),
			},
			expectedError: "match service returned error: {\"error\": \"match not found\"}",
		},
		{
			name: "invalid response JSON",
			request: &models.MatchConfirmRequest{
				ID:     matchID,
				UserID: userID,
			},
			mockResponse: &models.ProxyResponse{
				StatusCode: 200,
				Body:       []byte(`invalid json`),
			},
			expectedError: "failed to parse match service response:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserUC := usersmocks.NewMockUserUC(ctrl)
			mockGatewayGW := gatewaymocks.NewMockGatewayGW(ctrl)

			expectedHeaders := map[string]string{"X-User-ID": userID}

			if tt.mockError != nil {
				mockGatewayGW.EXPECT().
					CallMatchService(gomock.Any(), "POST", "/matches/"+matchID+"/confirm", tt.request, expectedHeaders, nil).
					Return(nil, tt.mockError).
					Times(1)
			} else {
				mockGatewayGW.EXPECT().
					CallMatchService(gomock.Any(), "POST", "/matches/"+matchID+"/confirm", tt.request, expectedHeaders, nil).
					Return(tt.mockResponse, nil).
					Times(1)
			}

			gatewayUC := NewGatewayUC(mockUserUC, mockGatewayGW)

			result, err := gatewayUC.ConfirmMatch(context.Background(), tt.request)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedResult.ID, result.ID)
				assert.Equal(t, tt.expectedResult.DriverID, result.DriverID)
				assert.Equal(t, tt.expectedResult.PassengerID, result.PassengerID)
			}
		})
	}
}

func TestGatewayUseCase_UpdateUserLocation(t *testing.T) {
	tests := []struct {
		name          string
		request       *models.LocationUpdate
		mockResponse  *models.ProxyResponse
		mockError     error
		expectedError string
	}{
		{
			name: "successful location update",
			request: &models.LocationUpdate{
				DriverID: uuid.New().String(),
				Location: models.Location{
					Latitude:  -6.175392,
					Longitude: 106.827153,
				},
			},
			mockResponse: &models.ProxyResponse{
				StatusCode: 200,
				Body:       []byte(`{"success": true}`),
			},
			expectedError: "",
		},
		{
			name: "location service error",
			request: &models.LocationUpdate{
				DriverID: uuid.New().String(),
				Location: models.Location{
					Latitude:  -6.175392,
					Longitude: 106.827153,
				},
			},
			mockError:     errors.New("location service unavailable"),
			expectedError: "failed to update location: location service unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserUC := usersmocks.NewMockUserUC(ctrl)
			mockGatewayGW := gatewaymocks.NewMockGatewayGW(ctrl)

			if tt.mockError != nil {
				mockGatewayGW.EXPECT().
					CallLocationService(gomock.Any(), "POST", "/locations/update", tt.request, nil, nil).
					Return(nil, tt.mockError).
					Times(1)
			} else {
				mockGatewayGW.EXPECT().
					CallLocationService(gomock.Any(), "POST", "/locations/update", tt.request, nil, nil).
					Return(tt.mockResponse, nil).
					Times(1)
			}

			gatewayUC := NewGatewayUC(mockUserUC, mockGatewayGW)

			err := gatewayUC.UpdateUserLocation(context.Background(), tt.request)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGatewayUseCase_ProxyOperations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserUC := usersmocks.NewMockUserUC(ctrl)
	mockGatewayGW := gatewaymocks.NewMockGatewayGW(ctrl)
	gatewayUC := NewGatewayUC(mockUserUC, mockGatewayGW)

	ctx := context.Background()
	method := "GET"
	path := "/test"
	body := map[string]string{"test": "data"}
	headers := map[string]string{"Content-Type": "application/json"}
	queryParams := map[string]string{"param": "value"}

	expectedResponse := &models.ProxyResponse{
		StatusCode: 200,
		Body:       []byte(`{"success": true}`),
	}

	t.Run("ProxyToUsersService", func(t *testing.T) {
		mockGatewayGW.EXPECT().
			CallUsersService(ctx, method, path, body, headers, queryParams).
			Return(expectedResponse, nil).
			Times(1)

		result, err := gatewayUC.ProxyToUsersService(ctx, method, path, body, headers, queryParams)

		assert.NoError(t, err)
		assert.Equal(t, expectedResponse, result)
	})

	t.Run("ProxyToMatchService", func(t *testing.T) {
		mockGatewayGW.EXPECT().
			CallMatchService(ctx, method, path, body, headers, queryParams).
			Return(expectedResponse, nil).
			Times(1)

		result, err := gatewayUC.ProxyToMatchService(ctx, method, path, body, headers, queryParams)

		assert.NoError(t, err)
		assert.Equal(t, expectedResponse, result)
	})

	t.Run("ProxyToRidesService", func(t *testing.T) {
		mockGatewayGW.EXPECT().
			CallRidesService(ctx, method, path, body, headers, queryParams).
			Return(expectedResponse, nil).
			Times(1)

		result, err := gatewayUC.ProxyToRidesService(ctx, method, path, body, headers, queryParams)

		assert.NoError(t, err)
		assert.Equal(t, expectedResponse, result)
	})

	t.Run("ProxyToLocationService", func(t *testing.T) {
		mockGatewayGW.EXPECT().
			CallLocationService(ctx, method, path, body, headers, queryParams).
			Return(expectedResponse, nil).
			Times(1)

		result, err := gatewayUC.ProxyToLocationService(ctx, method, path, body, headers, queryParams)

		assert.NoError(t, err)
		assert.Equal(t, expectedResponse, result)
	})
}