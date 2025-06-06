package gateaway_http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPGateway_MatchConfirm(t *testing.T) {
	tests := []struct {
		name           string
		request        *models.MatchConfirmRequest
		mockResponse   *models.MatchProposal
		mockStatusCode int
		mockError      bool
		expectError    bool
	}{
		{
			name: "successful match confirmation",
			request: &models.MatchConfirmRequest{
				ID:     "match-123",
				UserID: "user-456",
				Role:   "driver",
				Status: "accepted",
			},
			mockResponse: &models.MatchProposal{
				ID:          "match-123",
				DriverID:    "user-456",
				PassengerID: "passenger-789",
				UserLocation: models.Location{
					Latitude:  -6.175392,
					Longitude: 106.827153,
				},
				DriverLocation: models.Location{
					Latitude:  -6.185392,
					Longitude: 106.837153,
				},
				TargetLocation: models.Location{
					Latitude:  -6.195392,
					Longitude: 106.847153,
				},
				MatchStatus: "accepted",
			},
			mockStatusCode: http.StatusOK,
			expectError:    false,
		},
		{
			name: "match confirmation with server error",
			request: &models.MatchConfirmRequest{
				ID:     "match-456",
				UserID: "user-789",
				Role:   "passenger",
				Status: "rejected",
			},
			mockStatusCode: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name: "match confirmation with not found",
			request: &models.MatchConfirmRequest{
				ID:     "match-nonexistent",
				UserID: "user-123",
				Role:   "driver",
				Status: "accepted",
			},
			mockStatusCode: http.StatusNotFound,
			expectError:    true,
		},
		{
			name: "match confirmation with bad request",
			request: &models.MatchConfirmRequest{
				ID:     "invalid-match",
				UserID: "user-123",
				Role:   "driver",
				Status: "accepted",
			},
			mockStatusCode: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and path
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Contains(t, r.URL.Path, "/matches/"+tt.request.ID+"/confirm")
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Verify request body
				var requestBody models.MatchConfirmRequest
				json.NewDecoder(r.Body).Decode(&requestBody)
				assert.Equal(t, tt.request.ID, requestBody.ID)
				if tt.request.UserID != "" {
					assert.Equal(t, tt.request.UserID, requestBody.UserID)
				}
				if tt.request.Role != "" {
					assert.Equal(t, tt.request.Role, requestBody.Role)
				}
				if tt.request.Status != "" {
					assert.Equal(t, tt.request.Status, requestBody.Status)
				}

				w.WriteHeader(tt.mockStatusCode)
				if tt.mockResponse != nil {
					response := map[string]interface{}{
						"success": true,
						"data":    tt.mockResponse,
					}
					json.NewEncoder(w).Encode(response)
				}
			}))
			defer server.Close()

			// Create gateway with mock server URL and API key config
			config := &models.APIKeyConfig{
				MatchService: "test-api-key",
				RidesService: "test-api-key",
			}
			gateway := NewHTTPGatewayWithAPIKey(server.URL, "", config)

			// Execute test
			result, err := gateway.MatchConfirm(tt.request)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.mockResponse.ID, result.ID)
				assert.Equal(t, tt.mockResponse.MatchStatus, result.MatchStatus)
				if tt.mockResponse.DriverID != "" {
					assert.Equal(t, tt.mockResponse.DriverID, result.DriverID)
				}
				if tt.mockResponse.PassengerID != "" {
					assert.Equal(t, tt.mockResponse.PassengerID, result.PassengerID)
				}
				assert.Equal(t, tt.mockResponse.UserLocation.Latitude, result.UserLocation.Latitude)
				assert.Equal(t, tt.mockResponse.UserLocation.Longitude, result.UserLocation.Longitude)
				assert.Equal(t, tt.mockResponse.DriverLocation.Latitude, result.DriverLocation.Latitude)
				assert.Equal(t, tt.mockResponse.DriverLocation.Longitude, result.DriverLocation.Longitude)
				assert.Equal(t, tt.mockResponse.TargetLocation.Latitude, result.TargetLocation.Latitude)
				assert.Equal(t, tt.mockResponse.TargetLocation.Longitude, result.TargetLocation.Longitude)
			}
		})
	}
}

func TestHTTPGateway_MatchConfirm_NetworkError(t *testing.T) {
	// Test with server that immediately closes connection
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Close connection immediately
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, _ := hj.Hijack()
			conn.Close()
		}
	}))
	server.Close() // Close server immediately to simulate network error

	config := &models.APIKeyConfig{
		MatchService: "test-api-key",
		RidesService: "test-api-key",
	}
	gateway := NewHTTPGatewayWithAPIKey(server.URL, "", config)
	request := &models.MatchConfirmRequest{
		ID:     "match-123",
		UserID: "user-456",
		Role:   "driver",
		Status: "accepted",
	}

	result, err := gateway.MatchConfirm(request)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to send match confirmation request")
}

func TestHTTPGateway_MatchConfirm_InvalidResponseJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"invalid": "json" "malformed"`)) // Malformed JSON
	}))
	defer server.Close()

	config := &models.APIKeyConfig{
		MatchService: "test-api-key",
		RidesService: "test-api-key",
	}
	gateway := NewHTTPGatewayWithAPIKey(server.URL, "", config)
	request := &models.MatchConfirmRequest{
		ID:     "match-123",
		UserID: "user-456",
		Role:   "driver",
		Status: "accepted",
	}

	result, err := gateway.MatchConfirm(request)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to decode JSON response")
}

func TestHTTPGateway_MatchConfirm_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(``)) // Empty response
	}))
	defer server.Close()

	config := &models.APIKeyConfig{
		MatchService: "test-api-key",
		RidesService: "test-api-key",
	}
	gateway := NewHTTPGatewayWithAPIKey(server.URL, "", config)
	request := &models.MatchConfirmRequest{
		ID:     "match-123",
		UserID: "user-456",
		Role:   "driver",
		Status: "accepted",
	}

	result, err := gateway.MatchConfirm(request)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to decode JSON response")
}

func TestNewMatchClient(t *testing.T) {
	url := "http://match-service:8080"
	client := NewMatchClient(url)

	assert.NotNil(t, client)
	assert.NotNil(t, client.client)
	assert.Equal(t, url, client.client.BaseURL)
}

func TestNewHTTPGateway(t *testing.T) {
	matchURL := "http://match-service:8080"
	rideURL := "http://ride-service:8080"

	gateway := NewHTTPGateway(matchURL, rideURL)

	assert.NotNil(t, gateway)
	assert.NotNil(t, gateway.matchClient)
	assert.NotNil(t, gateway.rideClient)
	assert.Equal(t, matchURL, gateway.matchClient.client.BaseURL)
	assert.Equal(t, rideURL, gateway.rideClient.client.BaseURL)
}
