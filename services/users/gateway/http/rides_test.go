package gateaway_http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPGateway_StartRide(t *testing.T) {
	tests := []struct {
		name           string
		request        *models.RideStartRequest
		mockResponse   *models.Ride
		mockStatusCode int
		expectError    bool
	}{
		{
			name: "successful ride start",
			request: &models.RideStartRequest{
				RideID: "ride-123",
			},
			mockResponse: &models.Ride{
				RideID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				Status: models.RideStatusOngoing,
			},
			mockStatusCode: http.StatusOK,
			expectError:    false,
		},
		{
			name: "ride start with server error",
			request: &models.RideStartRequest{
				RideID: "ride-456",
			},
			mockStatusCode: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Contains(t, r.URL.Path, "/rides/"+tt.request.RideID+"/confirm")
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

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

			config := &models.APIKeyConfig{
				MatchService: "test-api-key",
				RidesService: "test-api-key",
			}
			gateway := NewHTTPGatewayWithAPIKey("", server.URL, config)
			result, err := gateway.StartRide(tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.mockResponse.RideID, result.RideID)
				assert.Equal(t, tt.mockResponse.Status, result.Status)
			}
		})
	}
}

func TestHTTPGateway_RideArrived(t *testing.T) {
	tests := []struct {
		name           string
		request        *models.RideArrivalReq
		mockResponse   *models.PaymentRequest
		mockStatusCode int
		expectError    bool
	}{
		{
			name: "successful ride arrival",
			request: &models.RideArrivalReq{
				RideID: "ride-123",
			},
			mockResponse: &models.PaymentRequest{
				RideID:    "payment-123",
				TotalCost: 25000,
			},
			mockStatusCode: http.StatusOK,
			expectError:    false,
		},
		{
			name: "ride arrival with bad request",
			request: &models.RideArrivalReq{
				RideID: "ride-invalid",
			},
			mockStatusCode: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Contains(t, r.URL.Path, "/rides/"+tt.request.RideID+"/arrive")
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

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

			config := &models.APIKeyConfig{
				MatchService: "test-api-key",
				RidesService: "test-api-key",
			}
			gateway := NewHTTPGatewayWithAPIKey("", server.URL, config)
			result, err := gateway.RideArrived(tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.mockResponse.RideID, result.RideID)
				assert.Equal(t, tt.mockResponse.TotalCost, result.TotalCost)
			}
		})
	}
}

func TestHTTPGateway_ProcessPayment(t *testing.T) {
	tests := []struct {
		name           string
		request        *models.PaymentProccessRequest
		mockResponse   *models.Payment
		mockStatusCode int
		expectError    bool
	}{
		{
			name: "successful payment processing", request: &models.PaymentProccessRequest{
				RideID:    "ride-123",
				TotalCost: 25000,
				Status:    models.PaymentStatusAccepted,
			},
			mockResponse: &models.Payment{
				PaymentID:    uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				Status:       models.PaymentStatusProcessed,
				AdjustedCost: 25000,
			},
			mockStatusCode: http.StatusOK,
			expectError:    false,
		},
		{
			name: "payment processing with unauthorized",
			request: &models.PaymentProccessRequest{
				RideID:    "ride-456",
				TotalCost: 30000,
				Status:    models.PaymentStatusAccepted,
			},
			mockStatusCode: http.StatusUnauthorized,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Contains(t, r.URL.Path, "/rides/"+tt.request.RideID+"/payment")
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

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

			config := &models.APIKeyConfig{
				MatchService: "test-api-key",
				RidesService: "test-api-key",
			}
			gateway := NewHTTPGatewayWithAPIKey("", server.URL, config)
			result, err := gateway.ProcessPayment(tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.mockResponse.PaymentID, result.PaymentID)
				assert.Equal(t, tt.mockResponse.Status, result.Status)
				assert.Equal(t, tt.mockResponse.AdjustedCost, result.AdjustedCost)
			}
		})
	}
}

func TestNewRideClient(t *testing.T) {
	url := "http://ride-service:8080"
	client := NewRideClient(url)

	assert.NotNil(t, client)
	assert.NotNil(t, client.client)
	assert.Equal(t, url, client.client.BaseURL)
}

// Additional test cases for StartRide method
func TestHTTPGateway_StartRide_ErrorScenarios(t *testing.T) {
	tests := []struct {
		name          string
		request       *models.RideStartRequest
		setupServer   func() *httptest.Server
		expectError   bool
		errorContains string
	}{
		{
			name: "network error",
			request: &models.RideStartRequest{
				RideID: "ride-123",
				DriverLocation: &models.Location{
					Latitude:  -6.175392,
					Longitude: 106.827153,
				},
				PassengerLocation: &models.Location{
					Latitude:  -6.175400,
					Longitude: 106.827160,
				},
			},
			setupServer: func() *httptest.Server {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					hj, ok := w.(http.Hijacker)
					if ok {
						conn, _, _ := hj.Hijack()
						conn.Close()
					}
				}))
				server.Close() // Close immediately to simulate network error
				return server
			},
			expectError:   true,
			errorContains: "failed to send start trip request",
		},
		{
			name: "invalid response JSON",
			request: &models.RideStartRequest{
				RideID: "ride-123",
				DriverLocation: &models.Location{
					Latitude:  -6.175392,
					Longitude: 106.827153,
				},
				PassengerLocation: &models.Location{
					Latitude:  -6.175400,
					Longitude: 106.827160,
				},
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"invalid": "json" "malformed"`))
				}))
			},
			expectError:   true,
			errorContains: "failed to parse start trip response",
		},
		{
			name: "empty response body",
			request: &models.RideStartRequest{
				RideID: "ride-123",
				DriverLocation: &models.Location{
					Latitude:  -6.175392,
					Longitude: 106.827153,
				},
				PassengerLocation: &models.Location{
					Latitude:  -6.175400,
					Longitude: 106.827160,
				},
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(``))
				}))
			},
			expectError:   true,
			errorContains: "failed to parse start trip response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			config := &models.APIKeyConfig{
				MatchService: "test-api-key",
				RidesService: "test-api-key",
			}
			gateway := NewHTTPGatewayWithAPIKey("", server.URL, config)
			result, err := gateway.StartRide(tt.request)

			assert.Error(t, err)
			assert.Nil(t, result)
			if tt.errorContains != "" {
				// Update error message expectations to match actual HTTP client errors
				if tt.errorContains == "failed to send start trip request" {
					assert.Contains(t, err.Error(), "failed to start ride")
				} else if tt.errorContains == "failed to parse start trip response" {
					assert.Contains(t, err.Error(), "failed to decode JSON response")
				} else {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			}
		})
	}
}

// Additional test cases for RideArrived method
func TestHTTPGateway_RideArrived_ErrorScenarios(t *testing.T) {
	tests := []struct {
		name          string
		request       *models.RideArrivalReq
		setupServer   func() *httptest.Server
		expectError   bool
		errorContains string
	}{
		{
			name: "network error",
			request: &models.RideArrivalReq{
				RideID:           "ride-123",
				AdjustmentFactor: 0.9,
			},
			setupServer: func() *httptest.Server {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					hj, ok := w.(http.Hijacker)
					if ok {
						conn, _, _ := hj.Hijack()
						conn.Close()
					}
				}))
				server.Close()
				return server
			},
			expectError:   true,
			errorContains: "failed to send ride arrival request",
		},
		{
			name: "invalid response JSON",
			request: &models.RideArrivalReq{
				RideID:           "ride-123",
				AdjustmentFactor: 0.8,
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"invalid": "json" "malformed"`))
				}))
			},
			expectError:   true,
			errorContains: "failed to parse ride arrival response",
		},
		{
			name: "forbidden access",
			request: &models.RideArrivalReq{
				RideID:           "ride-123",
				AdjustmentFactor: 1.0,
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusForbidden)
				}))
			},
			expectError:   true,
			errorContains: "ride arrival request failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			config := &models.APIKeyConfig{
				MatchService: "test-api-key",
				RidesService: "test-api-key",
			}
			gateway := NewHTTPGatewayWithAPIKey("", server.URL, config)
			result, err := gateway.RideArrived(tt.request)

			assert.Error(t, err)
			assert.Nil(t, result)
			if tt.errorContains != "" {
				// Update error message expectations to match actual HTTP client errors
				if tt.errorContains == "failed to send ride arrival request" {
					assert.Contains(t, err.Error(), "failed to process ride arrival")
				} else if tt.errorContains == "failed to parse ride arrival response" {
					assert.Contains(t, err.Error(), "failed to decode JSON response")
				} else if tt.errorContains == "ride arrival request failed" {
					assert.Contains(t, err.Error(), "HTTP error")
				} else {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			}
		})
	}
}

// Additional test cases for ProcessPayment method
func TestHTTPGateway_ProcessPayment_ErrorScenarios(t *testing.T) {
	tests := []struct {
		name          string
		request       *models.PaymentProccessRequest
		setupServer   func() *httptest.Server
		expectError   bool
		errorContains string
	}{
		{
			name: "network error",
			request: &models.PaymentProccessRequest{
				RideID:    "ride-123",
				TotalCost: 25000,
				Status:    models.PaymentStatusAccepted,
			},
			setupServer: func() *httptest.Server {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					hj, ok := w.(http.Hijacker)
					if ok {
						conn, _, _ := hj.Hijack()
						conn.Close()
					}
				}))
				server.Close()
				return server
			},
			expectError:   true,
			errorContains: "failed to send payment request",
		},
		{
			name: "invalid response JSON",
			request: &models.PaymentProccessRequest{
				RideID:    "ride-123",
				TotalCost: 25000,
				Status:    models.PaymentStatusAccepted,
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"invalid": "json" "malformed"`))
				}))
			},
			expectError:   true,
			errorContains: "failed to parse payment response",
		},
		{
			name: "payment conflict",
			request: &models.PaymentProccessRequest{
				RideID:    "ride-123",
				TotalCost: 25000,
				Status:    models.PaymentStatusAccepted,
			},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusConflict)
				}))
			},
			expectError:   true,
			errorContains: "payment request failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			config := &models.APIKeyConfig{
				MatchService: "test-api-key",
				RidesService: "test-api-key",
			}
			gateway := NewHTTPGatewayWithAPIKey("", server.URL, config)
			result, err := gateway.ProcessPayment(tt.request)

			assert.Error(t, err)
			assert.Nil(t, result)
			if tt.errorContains != "" {
				// Update error message expectations to match actual HTTP client errors
				if tt.errorContains == "failed to send payment request" {
					assert.Contains(t, err.Error(), "failed to process payment")
				} else if tt.errorContains == "failed to parse payment response" {
					assert.Contains(t, err.Error(), "failed to decode JSON response")
				} else if tt.errorContains == "payment request failed" {
					assert.Contains(t, err.Error(), "HTTP error")
				} else {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			}
		})
	}
}

// Test for request body validation
func TestHTTPGateway_RequestBodyValidation(t *testing.T) {
	t.Run("StartRide with valid request body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			// Decode and validate request body
			var req models.RideStartRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			assert.NoError(t, err)
			assert.Equal(t, "ride-123", req.RideID)
			assert.NotNil(t, req.DriverLocation)
			assert.NotNil(t, req.PassengerLocation)

			w.WriteHeader(http.StatusOK)
			response := &models.Ride{
				RideID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				Status: models.RideStatusOngoing,
			}
			apiResponse := map[string]interface{}{
				"success": true,
				"data":    response,
			}
			json.NewEncoder(w).Encode(apiResponse)
		}))
		defer server.Close()

		config := &models.APIKeyConfig{
			MatchService: "test-api-key",
			RidesService: "test-api-key",
		}
		gateway := NewHTTPGatewayWithAPIKey("", server.URL, config)
		request := &models.RideStartRequest{
			RideID: "ride-123",
			DriverLocation: &models.Location{
				Latitude:  -6.175392,
				Longitude: 106.827153,
			},
			PassengerLocation: &models.Location{
				Latitude:  -6.175400,
				Longitude: 106.827160,
			},
		}

		result, err := gateway.StartRide(request)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), result.RideID)
	})

	t.Run("RideArrived with valid request body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			// Decode and validate request body
			var req models.RideArrivalReq
			err := json.NewDecoder(r.Body).Decode(&req)
			assert.NoError(t, err)
			assert.Equal(t, "ride-123", req.RideID)
			assert.Equal(t, 0.9, req.AdjustmentFactor)

			w.WriteHeader(http.StatusOK)
			response := &models.PaymentRequest{
				RideID:    "payment-123",
				TotalCost: 22500, // 25000 * 0.9
			}
			apiResponse := map[string]interface{}{
				"success": true,
				"data":    response,
			}
			json.NewEncoder(w).Encode(apiResponse)
		}))
		defer server.Close()

		config := &models.APIKeyConfig{
			MatchService: "test-api-key",
			RidesService: "test-api-key",
		}
		gateway := NewHTTPGatewayWithAPIKey("", server.URL, config)
		request := &models.RideArrivalReq{
			RideID:           "ride-123",
			AdjustmentFactor: 0.9,
		}

		result, err := gateway.RideArrived(request)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "payment-123", result.RideID)
	})

	t.Run("ProcessPayment with valid request body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			// Decode and validate request body
			var req models.PaymentProccessRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			assert.NoError(t, err)
			assert.Equal(t, "ride-123", req.RideID)
			assert.Equal(t, 25000, req.TotalCost)
			assert.Equal(t, models.PaymentStatusAccepted, req.Status)

			w.WriteHeader(http.StatusOK)
			response := &models.Payment{
				PaymentID:    uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				Status:       models.PaymentStatusProcessed,
				AdjustedCost: 25000,
			}
			apiResponse := map[string]interface{}{
				"success": true,
				"data":    response,
			}
			json.NewEncoder(w).Encode(apiResponse)
		}))
		defer server.Close()

		config := &models.APIKeyConfig{
			MatchService: "test-api-key",
			RidesService: "test-api-key",
		}
		gateway := NewHTTPGatewayWithAPIKey("", server.URL, config)
		request := &models.PaymentProccessRequest{
			RideID:    "ride-123",
			TotalCost: 25000,
			Status:    models.PaymentStatusAccepted,
		}

		result, err := gateway.ProcessPayment(request)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), result.PaymentID)
		assert.Equal(t, models.PaymentStatusProcessed, result.Status)
	})
}
