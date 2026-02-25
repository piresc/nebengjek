package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	coremodels "github.com/piresc/nebengjek/internal/pkg/models/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockHTTPClient is a mock implementation of HTTP client
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

func TestNewHTTPGateway(t *testing.T) {
	config := &coremodels.Config{
		APIKey: coremodels.APIKeyConfig{
			UserService:    "test-users-key",
			MatchService:    "test-match-key",
			RidesService:    "test-rides-key",
			LocationService: "test-location-key",
		},
	}

	gateway := NewHTTPGateway(config)
	
	assert.NotNil(t, gateway)
	httpGateway, ok := gateway.(*HTTPGateway)
	assert.True(t, ok)
	assert.NotNil(t, httpGateway.client)
	assert.NotNil(t, httpGateway.config)
	assert.NotNil(t, httpGateway.endpoints)
	
	// Verify default endpoints
	assert.Equal(t, "http://localhost:9990", httpGateway.endpoints.UsersService)
	assert.Equal(t, "http://localhost:9993", httpGateway.endpoints.MatchService)
	assert.Equal(t, "http://localhost:9992", httpGateway.endpoints.RidesService)
	assert.Equal(t, "http://localhost:9994", httpGateway.endpoints.LocationService)
}

func TestNewHTTPGateway_NilConfig(t *testing.T) {
	gateway := NewHTTPGateway(nil)
	assert.NotNil(t, gateway)
	httpGateway, ok := gateway.(*HTTPGateway)
	assert.True(t, ok)
	assert.NotNil(t, httpGateway.client)
	assert.Nil(t, httpGateway.config)
	assert.NotNil(t, httpGateway.endpoints)
}

func TestCallUsersService(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/internal/test/path", r.URL.Path)
		assert.Equal(t, "test-users-key", r.Header.Get("X-API-Key"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "custom-value", r.Header.Get("Custom-Header"))
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer server.Close()

	config := &coremodels.Config{
		APIKey: coremodels.APIKeyConfig{
			UserService: "test-users-key",
		},
	}

	gateway := &HTTPGateway{
		client: &http.Client{Timeout: 30 * time.Second},
		config: config,
		endpoints: &ServiceEndpoints{
			UsersService: server.URL,
		},
	}

	headers := map[string]string{"Custom-Header": "custom-value"}
	queryParams := map[string]string{"param": "value"}
	body := map[string]string{"test": "data"}

	resp, err := gateway.CallUsersService(context.Background(), "POST", "/test/path", body, headers, queryParams)
	
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, `{"status": "success"}`, string(resp.Body))
}

func TestCallMatchService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/internal/matches", r.URL.Path)
		assert.Equal(t, "test-match-key", r.Header.Get("X-API-Key"))
		
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"match_id": "123"}`))
	}))
	defer server.Close()

	config := &coremodels.Config{
		APIKey: coremodels.APIKeyConfig{
			MatchService: "test-match-key",
		},
	}

	gateway := &HTTPGateway{
		client: &http.Client{Timeout: 30 * time.Second},
		config: config,
		endpoints: &ServiceEndpoints{
			MatchService: server.URL,
		},
	}

	resp, err := gateway.CallMatchService(context.Background(), "GET", "/matches", nil, nil, nil)
	
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, `{"match_id": "123"}`, string(resp.Body))
}

func TestCallRidesService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/internal/rides/123", r.URL.Path)
		assert.Equal(t, "test-rides-key", r.Header.Get("X-API-Key"))
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ride_status": "updated"}`))
	}))
	defer server.Close()

	config := &coremodels.Config{
		APIKey: coremodels.APIKeyConfig{
			RidesService: "test-rides-key",
		},
	}

	gateway := &HTTPGateway{
		client: &http.Client{Timeout: 30 * time.Second},
		config: config,
		endpoints: &ServiceEndpoints{
			RidesService: server.URL,
		},
	}

	body := map[string]string{"status": "completed"}
	resp, err := gateway.CallRidesService(context.Background(), "PUT", "/rides/123", body, nil, nil)
	
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCallLocationService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/internal/locations/456", r.URL.Path)
		assert.Equal(t, "test-location-key", r.Header.Get("X-API-Key"))
		
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	config := &coremodels.Config{
		APIKey: coremodels.APIKeyConfig{
			LocationService: "test-location-key",
		},
	}

	gateway := &HTTPGateway{
		client: &http.Client{Timeout: 30 * time.Second},
		config: config,
		endpoints: &ServiceEndpoints{
			LocationService: server.URL,
		},
	}

	resp, err := gateway.CallLocationService(context.Background(), "DELETE", "/locations/456", nil, nil, nil)
	
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestMakeHTTPCall_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate delay
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &coremodels.Config{
		APIKey: coremodels.APIKeyConfig{
			UserService: "test-key",
		},
	}

	gateway := &HTTPGateway{
		client: &http.Client{Timeout: 30 * time.Second},
		config: config,
		endpoints: &ServiceEndpoints{
			UsersService: server.URL,
		},
	}

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	resp, err := gateway.makeHTTPCall(ctx, server.URL, "GET", "/test", nil, nil, nil, "test-key")
	
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestMakeHTTPCall_RequestCreationError(t *testing.T) {
	config := &coremodels.Config{
		APIKey: coremodels.APIKeyConfig{
			UserService: "test-key",
		},
	}

	gateway := &HTTPGateway{
		client: &http.Client{Timeout: 30 * time.Second},
		config: config,
		endpoints: &ServiceEndpoints{
			UsersService: "http://localhost:9990",
		},
	}

	// Test with invalid URL (this should cause request creation to fail)
	resp, err := gateway.makeHTTPCall(context.Background(), "://invalid-url", "GET", "/test", nil, nil, nil, "test-key")
	
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to create HTTP request")
}

func TestMakeHTTPCall_RequestBodyMarshalError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &coremodels.Config{
		APIKey: coremodels.APIKeyConfig{
			UserService: "test-key",
		},
	}

	gateway := &HTTPGateway{
		client: &http.Client{Timeout: 30 * time.Second},
		config: config,
		endpoints: &ServiceEndpoints{
			UsersService: server.URL,
		},
	}

	// Test with unmarshalable body (channel cannot be marshaled to JSON)
	body := make(chan int)
	resp, err := gateway.makeHTTPCall(context.Background(), server.URL, "POST", "/test", body, nil, nil, "test-key")
	
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to marshal request body")
}

func TestMakeHTTPCall_NetworkError(t *testing.T) {
	config := &coremodels.Config{
		APIKey: coremodels.APIKeyConfig{
			UserService: "test-key",
		},
	}

	gateway := &HTTPGateway{
		client: &http.Client{Timeout: 1 * time.Millisecond}, // Very short timeout
		config: config,
		endpoints: &ServiceEndpoints{
			UsersService: "http://localhost:99999", // Non-existent port
		},
	}

	resp, err := gateway.makeHTTPCall(context.Background(), "http://localhost:99999", "GET", "/test", nil, nil, nil, "test-key")
	
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "HTTP request failed")
}

func TestMakeHTTPCall_SuccessWithAllFeatures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify all request components
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/internal/api/test", r.URL.Path)
		// Query parameters order is not guaranteed, so check both parameters exist
		assert.Contains(t, r.URL.RawQuery, "param1=value1")
		assert.Contains(t, r.URL.RawQuery, "param2=value2")
		assert.Equal(t, "test-api-key", r.Header.Get("X-API-Key"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "custom-header", r.Header.Get("Custom-Header"))
		assert.Equal(t, "another-header", r.Header.Get("Another-Header"))
		
		// Verify request body
		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, "test-value", body["test_key"])
		
		// Set response headers
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom-Response", "response-value")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"response": "success", "id": 123}`))
	}))
	defer server.Close()

	config := &coremodels.Config{
		APIKey: coremodels.APIKeyConfig{
			UserService: "test-api-key",
		},
	}

	gateway := &HTTPGateway{
		client: &http.Client{Timeout: 30 * time.Second},
		config: config,
		endpoints: &ServiceEndpoints{
			UsersService: server.URL,
		},
	}

	headers := map[string]string{
		"Custom-Header":  "custom-header",
		"Another-Header": "another-header",
	}
	queryParams := map[string]string{
		"param1": "value1",
		"param2": "value2",
	}
	body := map[string]interface{}{
		"test_key": "test-value",
	}

	resp, err := gateway.makeHTTPCall(context.Background(), server.URL, "POST", "/api/test", body, headers, queryParams, "test-api-key")
	
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, `{"response": "success", "id": 123}`, string(resp.Body))
	assert.Equal(t, "application/json", resp.Headers.Get("Content-Type"))
	assert.Equal(t, "response-value", resp.Headers.Get("X-Custom-Response"))
}

func TestBuildQueryString(t *testing.T) {
	gateway := &HTTPGateway{}

	tests := []struct {
		name     string
		params   map[string]string
		expected string
		contains []string // For order-independent checking
	}{
		{
			name:     "Empty parameters",
			params:   nil,
			expected: "",
		},
		{
			name:     "Single parameter",
			params:   map[string]string{"key": "value"},
			expected: "key=value",
		},
		{
			name:     "Multiple parameters",
			params:   map[string]string{"key1": "value1", "key2": "value2"},
			contains: []string{"key1=value1", "key2=value2"},
		},
		{
			name:     "Special characters",
			params:   map[string]string{"search": "hello world", "filter": "active"},
			contains: []string{"search=hello world", "filter=active"},
		},
		{
			name:     "Empty values",
			params:   map[string]string{"empty": "", "normal": "value"},
			contains: []string{"empty=", "normal=value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gateway.buildQueryString(tt.params)
			
			// For exact string matching (single parameter or empty cases)
			if tt.expected != "" {
				assert.Equal(t, tt.expected, result)
			}
			
			// For order-independent checking (multiple parameters)
			if len(tt.contains) > 0 {
				for _, contain := range tt.contains {
					assert.Contains(t, result, contain)
				}
				// Verify proper & separation
				assert.Equal(t, len(tt.contains)-1, strings.Count(result, "&"))
			}
		})
	}
}

func TestMakeHTTPCall_QueryStringEncoding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "param1=value1&param2=value%20with%20spaces", r.URL.RawQuery)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &coremodels.Config{
		APIKey: coremodels.APIKeyConfig{
			UserService: "test-key",
		},
	}

	gateway := &HTTPGateway{
		client: &http.Client{Timeout: 30 * time.Second},
		config: config,
		endpoints: &ServiceEndpoints{
			UsersService: server.URL,
		},
	}

	queryParams := map[string]string{
		"param1": "value1",
		"param2": "value with spaces",
	}

	resp, err := gateway.makeHTTPCall(context.Background(), server.URL, "GET", "/test", nil, nil, queryParams, "test-key")
	
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestMakeHTTPCall_EmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &coremodels.Config{
		APIKey: coremodels.APIKeyConfig{
			UserService: "test-key",
		},
	}

	gateway := &HTTPGateway{
		client: &http.Client{Timeout: 30 * time.Second},
		config: config,
		endpoints: &ServiceEndpoints{
			UsersService: server.URL,
		},
	}

	resp, err := gateway.makeHTTPCall(context.Background(), server.URL, "GET", "/test", nil, nil, nil, "test-key")
	
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestMakeHTTPCall_NilHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-api-key", r.Header.Get("X-API-Key"))
		assert.NotEqual(t, "application/json", r.Header.Get("Content-Type")) // Should not be set for nil body
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &coremodels.Config{
		APIKey: coremodels.APIKeyConfig{
			UserService: "test-api-key",
		},
	}

	gateway := &HTTPGateway{
		client: &http.Client{Timeout: 30 * time.Second},
		config: config,
		endpoints: &ServiceEndpoints{
			UsersService: server.URL,
		},
	}

	resp, err := gateway.makeHTTPCall(context.Background(), server.URL, "GET", "/test", nil, nil, nil, "test-api-key")
	
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestMakeHTTPCall_HeaderOverwriting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The X-API-Key should be overwritten by the custom header (custom headers come after API key)
		assert.Equal(t, "this-should-be-overwritten", r.Header.Get("X-API-Key"))
		// Custom headers should be preserved
		assert.Equal(t, "custom-value", r.Header.Get("Custom-Header"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &coremodels.Config{
		APIKey: coremodels.APIKeyConfig{
			UserService: "test-api-key",
		},
	}

	gateway := &HTTPGateway{
		client: &http.Client{Timeout: 30 * time.Second},
		config: config,
		endpoints: &ServiceEndpoints{
			UsersService: server.URL,
		},
	}

	headers := map[string]string{
		"X-API-Key":    "this-should-be-overwritten",
		"Custom-Header": "custom-value",
	}

	resp, err := gateway.makeHTTPCall(context.Background(), server.URL, "GET", "/test", nil, headers, nil, "test-api-key")
	
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestServiceMethodsEdgeCases(t *testing.T) {
	config := &coremodels.Config{
		APIKey: coremodels.APIKeyConfig{
			UserService:    "test-users-key",
			MatchService:    "test-match-key",
			RidesService:    "test-rides-key",
			LocationService: "test-location-key",
		},
	}

	gateway := &HTTPGateway{
		client: &http.Client{Timeout: 30 * time.Second},
		config: config,
		endpoints: &ServiceEndpoints{
			UsersService:    "http://localhost:9990",
			MatchService:    "http://localhost:9993",
			RidesService:    "http://localhost:9992",
			LocationService: "http://localhost:9994",
		},
	}

	tests := []struct {
		name       string
		method     func(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*coremodels.ProxyResponse, error)
		serviceKey string
	}{
		{"UsersService", gateway.CallUsersService, "test-users-key"},
		{"MatchService", gateway.CallMatchService, "test-match-key"},
		{"RidesService", gateway.CallRidesService, "test-rides-key"},
		{"LocationService", gateway.CallLocationService, "test-location-key"},
	}

	for _, tt := range tests {
		t.Run(tt.name+" with empty parameters", func(t *testing.T) {
			// This should not panic but will likely fail due to network error
			resp, err := tt.method(context.Background(), "", "", nil, nil, nil)
			assert.Error(t, err)
			assert.Nil(t, resp)
		})
		
		t.Run(tt.name+" with nil context", func(t *testing.T) {
			resp, err := tt.method(nil, "GET", "/test", nil, nil, nil)
			assert.Error(t, err)
			assert.Nil(t, resp)
		})
	}
}