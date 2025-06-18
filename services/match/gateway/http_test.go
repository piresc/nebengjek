package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPGateway(t *testing.T) {
	locationServiceURL := "http://localhost:8080"
	config := &models.APIKeyConfig{
		MatchService: "test-api-key",
	}

	gateway := NewHTTPGateway(locationServiceURL, config, nil, nil)

	assert.NotNil(t, gateway)
	assert.NotNil(t, gateway.locationClient)
	assert.Equal(t, locationServiceURL, gateway.locationClient.baseURL)
}

func TestLocationClient_AddAvailableDriver_Success(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/internal/drivers/driver-123/available", r.URL.Path)
		assert.Equal(t, "test-api-key", r.Header.Get("X-API-Key"))

		// Verify request body
		var requestBody map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&requestBody)
		require.NoError(t, err)
		location := requestBody["location"].(map[string]interface{})
		assert.Equal(t, -6.175392, location["latitude"])
		assert.Equal(t, 106.827153, location["longitude"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"success": true,
			"message": "Driver added successfully",
			"data":    map[string]string{"status": "success"},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &models.APIKeyConfig{
		MatchService: "test-api-key",
	}
	gateway := NewHTTPGateway(server.URL, config, nil, nil)

	location := &models.Location{
		Latitude:  -6.175392,
		Longitude: 106.827153,
	}

	err := gateway.locationClient.AddAvailableDriver(context.Background(), "driver-123", location)
	assert.NoError(t, err)
}

func TestLocationClient_AddAvailableDriver_ServerError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	config := &models.APIKeyConfig{
		MatchService: "test-api-key",
	}
	gateway := NewHTTPGateway(server.URL, config, nil, nil)

	location := &models.Location{
		Latitude:  -6.175392,
		Longitude: 106.827153,
	}

	err := gateway.locationClient.AddAvailableDriver(context.Background(), "driver-123", location)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add available driver")
}

func TestLocationClient_RemoveAvailableDriver_Success(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/internal/drivers/driver-123/available", r.URL.Path)
		assert.Equal(t, "test-api-key", r.Header.Get("X-API-Key"))

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &models.APIKeyConfig{
		MatchService: "test-api-key",
	}
	gateway := NewHTTPGateway(server.URL, config, nil, nil)

	err := gateway.locationClient.RemoveAvailableDriver(context.Background(), "driver-123")
	assert.NoError(t, err)
}

func TestLocationClient_RemoveAvailableDriver_ServerError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	config := &models.APIKeyConfig{
		MatchService: "test-api-key",
	}
	gateway := NewHTTPGateway(server.URL, config, nil, nil)

	err := gateway.locationClient.RemoveAvailableDriver(context.Background(), "driver-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove available driver")
}

func TestLocationClient_FindNearbyDrivers_Success(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/internal/drivers/nearby", r.URL.Path)
		assert.Equal(t, "test-api-key", r.Header.Get("X-API-Key"))

		// Check query parameters
		query := r.URL.Query()
		assert.Equal(t, "-6.175392", query.Get("lat"))
		assert.Equal(t, "106.827153", query.Get("lng"))
		assert.Equal(t, "5.000000", query.Get("radius"))

		// Return mock nearby drivers
		nearbyDrivers := []*models.NearbyUser{
			{
				ID:       "driver-1",
				Distance: 1.5,
			},
			{
				ID:       "driver-2",
				Distance: 3.2,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(nearbyDrivers)
	}))
	defer server.Close()

	config := &models.APIKeyConfig{
		MatchService: "test-api-key",
	}
	gateway := NewHTTPGateway(server.URL, config, nil, nil)

	location := &models.Location{
		Latitude:  -6.175392,
		Longitude: 106.827153,
	}

	drivers, err := gateway.locationClient.FindNearbyDrivers(context.Background(), location, 5.0)
	assert.NoError(t, err)
	assert.Len(t, drivers, 2)
	assert.Equal(t, "driver-1", drivers[0].ID)
	assert.Equal(t, 1.5, drivers[0].Distance)
	assert.Equal(t, "driver-2", drivers[1].ID)
	assert.Equal(t, 3.2, drivers[1].Distance)
}

func TestLocationClient_FindNearbyDrivers_ServerError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	config := &models.APIKeyConfig{
		MatchService: "test-api-key",
	}
	gateway := NewHTTPGateway(server.URL, config, nil, nil)

	location := &models.Location{
		Latitude:  -6.175392,
		Longitude: 106.827153,
	}

	drivers, err := gateway.locationClient.FindNearbyDrivers(context.Background(), location, 5.0)
	assert.Error(t, err)
	assert.Nil(t, drivers)
	assert.Contains(t, err.Error(), "failed to find nearby drivers")
}

func TestLocationClient_FindNearbyDrivers_EmptyResponse(t *testing.T) {
	// Create a test server that returns empty array
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]*models.NearbyUser{})
	}))
	defer server.Close()

	config := &models.APIKeyConfig{
		MatchService: "test-api-key",
	}
	gateway := NewHTTPGateway(server.URL, config, nil, nil)

	location := &models.Location{
		Latitude:  -6.175392,
		Longitude: 106.827153,
	}

	drivers, err := gateway.locationClient.FindNearbyDrivers(context.Background(), location, 5.0)
	assert.NoError(t, err)
	assert.Len(t, drivers, 0)
}

func TestHTTPGateway_FindNearbyDrivers(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nearbyDrivers := []*models.NearbyUser{
			{
				ID:       "driver-1",
				Distance: 2.1,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(nearbyDrivers)
	}))
	defer server.Close()

	config := &models.APIKeyConfig{
		MatchService: "test-api-key",
	}
	gateway := NewHTTPGateway(server.URL, config, nil, nil)

	location := &models.Location{
		Latitude:  -6.175392,
		Longitude: 106.827153,
	}

	// Test the gateway wrapper method
	drivers, err := gateway.FindNearbyDrivers(context.Background(), location, 3.0)
	assert.NoError(t, err)
	assert.Len(t, drivers, 1)
	assert.Equal(t, "driver-1", drivers[0].ID)
	assert.Equal(t, 2.1, drivers[0].Distance)
}

func TestLocationClient_WithTimeout(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Delay longer than client timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &models.APIKeyConfig{
		MatchService: "test-api-key",
	}
	gateway := NewHTTPGateway(server.URL, config, nil, nil)

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	location := &models.Location{
		Latitude:  -6.175392,
		Longitude: 106.827153,
	}

	err := gateway.locationClient.AddAvailableDriver(ctx, "driver-123", location)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}