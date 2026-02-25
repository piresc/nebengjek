package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/labstack/echo/v4"
	coremodels "github.com/piresc/nebengjek/internal/pkg/models/core"
	"github.com/piresc/nebengjek/services/gateway/mocks"
	"github.com/stretchr/testify/assert"
)

func TestNewProxyHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGatewayUC := mocks.NewMockGatewayUC(ctrl)

	handler := NewProxyHandler(mockGatewayUC)

	assert.NotNil(t, handler)
	assert.Equal(t, mockGatewayUC, handler.gatewayUC)
}

func TestProxyHandler_ProxyToUsersService(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		body           map[string]interface{}
		queryParams    map[string][]string
		headers        map[string]string
		userID         interface{}
		role           interface{}
		mockResponse   *coremodels.ProxyResponse
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "successful proxy request",
			method: "GET",
			path:   "/api/v1/users/profile",
			mockResponse: &coremodels.ProxyResponse{
				StatusCode: 200,
				Body:       []byte(`{"success": true}`),
				Headers:    map[string][]string{"Content-Type": {"application/json"}},
			},
			expectedStatus: 200,
			expectedBody:   `{"success": true}`,
		},
		{
			name:   "proxy request with body",
			method: "POST",
			path:   "/api/v1/users/register",
			body:   map[string]interface{}{"msisdn": "1234567890"},
			mockResponse: &coremodels.ProxyResponse{
				StatusCode: 201,
				Body:       []byte(`{"id": "123"}`),
				Headers:    map[string][]string{"Content-Type": {"application/json"}},
			},
			expectedStatus: 201,
			expectedBody:   `{"id": "123"}`,
		},
		{
			name:   "proxy request with user context",
			method: "GET",
			path:   "/api/v1/users/profile",
			userID: "user-123",
			role:   "driver",
			mockResponse: &coremodels.ProxyResponse{
				StatusCode: 200,
				Body:       []byte(`{"user": "profile"}`),
				Headers:    map[string][]string{},
			},
			expectedStatus: 200,
			expectedBody:   `{"user": "profile"}`,
		},
		{
			name:           "gateway usecase error",
			method:         "GET",
			path:           "/api/v1/users/profile",
			mockError:      errors.New("service unavailable"),
			expectedStatus: 502,
		},
		{
			name:   "service returns error status",
			method: "GET",
			path:   "/api/v1/users/profile",
			mockResponse: &coremodels.ProxyResponse{
				StatusCode: 404,
				Body:       []byte(`{"error": "not found"}`),
				Headers:    map[string][]string{},
			},
			expectedStatus: 404,
			expectedBody:   `{"error": "not found"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockGatewayUC := mocks.NewMockGatewayUC(ctrl)

			// Setup mock expectations
			if tt.mockError != nil {
				mockGatewayUC.EXPECT().
					ProxyToUsersService(gomock.Any(), tt.method, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, tt.mockError).
					Times(1)
			} else {
				mockGatewayUC.EXPECT().
					ProxyToUsersService(gomock.Any(), tt.method, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(tt.mockResponse, nil).
					Times(1)
			}

			handler := NewProxyHandler(mockGatewayUC)

			// Create request
			var bodyReader *bytes.Reader
			if tt.body != nil {
				bodyBytes, _ := json.Marshal(tt.body)
				bodyReader = bytes.NewReader(bodyBytes)
			} else {
				bodyReader = bytes.NewReader([]byte{})
			}

			req := httptest.NewRequest(tt.method, tt.path, bodyReader)
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}

			// Set query parameters
			if tt.queryParams != nil {
				q := req.URL.Query()
				for key, values := range tt.queryParams {
					for _, value := range values {
						q.Add(key, value)
					}
				}
				req.URL.RawQuery = q.Encode()
			}

			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)

			// Set user context if provided
			if tt.userID != nil {
				c.Set("user_id", tt.userID)
			}
			if tt.role != nil {
				c.Set("role", tt.role)
			}

			// Execute
			err := handler.ProxyToUsersService(c)

			// Assert
			if tt.mockError != nil {
				// Only expect HTTP errors when there's a mock error
				assert.Error(t, err)
				if httpErr, ok := err.(*echo.HTTPError); ok {
					assert.Equal(t, tt.expectedStatus, httpErr.Code)
				}
			} else {
				// When service returns a response (even error status), it should not return an error
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
				if tt.expectedBody != "" {
					assert.Equal(t, tt.expectedBody, rec.Body.String())
				}
			}
		})
	}
}

func TestProxyHandler_ProxyToMatchService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGatewayUC := mocks.NewMockGatewayUC(ctrl)
	mockResponse := &coremodels.ProxyResponse{
		StatusCode: 200,
		Body:       []byte(`{"matches": []}`),
		Headers:    map[string][]string{},
	}

	mockGatewayUC.EXPECT().
		ProxyToMatchService(gomock.Any(), "GET", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(mockResponse, nil).
		Times(1)

	handler := NewProxyHandler(mockGatewayUC)

	req := httptest.NewRequest("GET", "/api/v1/match/proposals", nil)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	err := handler.ProxyToMatchService(c)

	assert.NoError(t, err)
	assert.Equal(t, 200, rec.Code)
	assert.Equal(t, `{"matches": []}`, rec.Body.String())
}

func TestProxyHandler_ProxyToRidesService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGatewayUC := mocks.NewMockGatewayUC(ctrl)
	mockResponse := &coremodels.ProxyResponse{
		StatusCode: 200,
		Body:       []byte(`{"rides": []}`),
		Headers:    map[string][]string{},
	}

	mockGatewayUC.EXPECT().
		ProxyToRidesService(gomock.Any(), "GET", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(mockResponse, nil).
		Times(1)

	handler := NewProxyHandler(mockGatewayUC)

	req := httptest.NewRequest("GET", "/api/v1/rides/history", nil)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	err := handler.ProxyToRidesService(c)

	assert.NoError(t, err)
	assert.Equal(t, 200, rec.Code)
	assert.Equal(t, `{"rides": []}`, rec.Body.String())
}

func TestProxyHandler_ProxyToLocationService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGatewayUC := mocks.NewMockGatewayUC(ctrl)
	mockResponse := &coremodels.ProxyResponse{
		StatusCode: 200,
		Body:       []byte(`{"location": "updated"}`),
		Headers:    map[string][]string{},
	}

	mockGatewayUC.EXPECT().
		ProxyToLocationService(gomock.Any(), "POST", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(mockResponse, nil).
		Times(1)

	handler := NewProxyHandler(mockGatewayUC)

	body := map[string]interface{}{"latitude": -6.175392, "longitude": 106.827153}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/location/update", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	err := handler.ProxyToLocationService(c)

	assert.NoError(t, err)
	assert.Equal(t, 200, rec.Code)
	assert.Equal(t, `{"location": "updated"}`, rec.Body.String())
}

func TestProxyHandler_proxyRequest_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGatewayUC := mocks.NewMockGatewayUC(ctrl)
	handler := NewProxyHandler(mockGatewayUC)

	req := httptest.NewRequest("POST", "/api/v1/users/register", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	err := handler.ProxyToUsersService(c)

	assert.Error(t, err)
	if httpErr, ok := err.(*echo.HTTPError); ok {
		assert.Equal(t, http.StatusBadRequest, httpErr.Code)
		assert.Contains(t, httpErr.Message, "Invalid JSON body")
	}
}

func TestProxyHandler_cleanPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGatewayUC := mocks.NewMockGatewayUC(ctrl)
	handler := NewProxyHandler(mockGatewayUC)

	tests := []struct {
		name     string
		path     string
		service  string
		expected string
	}{
		{
			name:     "users service with api prefix",
			path:     "/api/v1/users/profile",
			service:  "users-service",
			expected: "/profile",
		},
		{
			name:     "users service without api prefix",
			path:     "/users/profile",
			service:  "users-service",
			expected: "/profile",
		},
		{
			name:     "match service path",
			path:     "/api/v1/match/proposals",
			service:  "match-service",
			expected: "/proposals",
		},
		{
			name:     "path without service prefix",
			path:     "/custom/endpoint",
			service:  "users-service",
			expected: "/custom/endpoint",
		},
		{
			name:     "root path",
			path:     "/",
			service:  "users-service",
			expected: "/",
		},
		{
			name:     "empty path",
			path:     "",
			service:  "users-service",
			expected: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.cleanPath(tt.path, tt.service)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProxyHandler_isSensitiveHeader(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGatewayUC := mocks.NewMockGatewayUC(ctrl)
	handler := NewProxyHandler(mockGatewayUC)

	sensitiveHeaders := []string{
		"Authorization",
		"X-API-Key",
		"Cookie",
		"Set-Cookie",
		"Host",
		"Content-Length",
		"Transfer-Encoding",
		"Connection",
		"Upgrade",
	}

	for _, header := range sensitiveHeaders {
		t.Run("sensitive_"+header, func(t *testing.T) {
			assert.True(t, handler.isSensitiveHeader(header))
			// Test case-insensitive
			assert.True(t, handler.isSensitiveHeader(header))
		})
	}

	nonSensitiveHeaders := []string{
		"Content-Type",
		"Accept",
		"User-Agent",
		"X-Request-ID",
		"Cache-Control",
	}

	for _, header := range nonSensitiveHeaders {
		t.Run("non_sensitive_"+header, func(t *testing.T) {
			assert.False(t, handler.isSensitiveHeader(header))
		})
	}
}