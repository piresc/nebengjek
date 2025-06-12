package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "Valid configuration",
			config: Config{
				BaseURL: "https://api.example.com",
				Timeout: 30 * time.Second,
			},
		},
		{
			name: "With trailing slash",
			config: Config{
				BaseURL: "https://api.example.com/",
				Timeout: 10 * time.Second,
			},
		},
		{
			name: "Localhost URL",
			config: Config{
				BaseURL: "http://localhost:8080",
				Timeout: 5 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.config)

			assert.NotNil(t, client)
			assert.Equal(t, tt.config.BaseURL, client.baseURL)
			assert.Equal(t, tt.config.Timeout, client.httpClient.Timeout)
			assert.NotNil(t, client.httpClient)
		})
	}
}

// TestClient_SetHeader removed - SetHeader method not implemented in current client

// TestClient_SetHeaders removed - SetHeaders method not implemented in current client

func TestClient_Get(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/test-endpoint", r.URL.Path)
		
		// Check default headers set by client
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, Timeout: 30*time.Second})

	ctx := context.Background()
	resp, err := client.Get(ctx, "/test-endpoint")

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, `{"message": "success"}`, string(body))
	resp.Body.Close()
}

func TestClient_Post(t *testing.T) {
	testPayload := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/users", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Read and verify request body
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		var receivedPayload map[string]interface{}
		err = json.Unmarshal(body, &receivedPayload)
		assert.NoError(t, err)
		assert.Equal(t, testPayload, receivedPayload)

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 123, "status": "created"}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, Timeout: 30*time.Second})

	ctx := context.Background()
	resp, err := client.Post(ctx, "/users", testPayload)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, `{"id": 123, "status": "created"}`, string(body))
	resp.Body.Close()
}

func TestClient_Put(t *testing.T) {
	testPayload := map[string]interface{}{
		"name":  "Jane Doe",
		"email": "jane@example.com",
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/users/123", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": 123, "status": "updated"}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, Timeout: 30*time.Second})

	ctx := context.Background()
	resp, err := client.Put(ctx, "/users/123", testPayload)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestClient_Delete(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/users/123", r.URL.Path)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, Timeout: 30*time.Second})

	ctx := context.Background()
	resp, err := client.Delete(ctx, "/users/123")

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()
}

func TestClient_Do(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		endpoint       string
		body           interface{}
		expectedMethod string
		expectedPath   string
		hasBody        bool
	}{
		{
			name:           "GET request",
			method:         "GET",
			endpoint:       "/api/data",
			body:           nil,
			expectedMethod: "GET",
			expectedPath:   "/api/data",
			hasBody:        false,
		},
		{
			name:           "POST request with body",
			method:         "POST",
			endpoint:       "/api/create",
			body:           map[string]string{"key": "value"},
			expectedMethod: "POST",
			expectedPath:   "/api/create",
			hasBody:        true,
		},
		{
			name:           "PATCH request",
			method:         "PATCH",
			endpoint:       "/api/update/123",
			body:           map[string]interface{}{"status": "active"},
			expectedMethod: "PATCH",
			expectedPath:   "/api/update/123",
			hasBody:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.expectedMethod, r.Method)
				assert.Equal(t, tt.expectedPath, r.URL.Path)

				if tt.hasBody {
					assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
					body, err := io.ReadAll(r.Body)
					assert.NoError(t, err)
					assert.NotEmpty(t, body)
				}

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"success": true}`))
			}))
			defer server.Close()

			client := NewClient(Config{BaseURL: server.URL, Timeout: 30*time.Second})

			ctx := context.Background()
			resp, err := client.Do(ctx, tt.method, tt.endpoint, tt.body)

			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
		})
	}
}

func TestClient_Do_WithContext(t *testing.T) {
	// Create test server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "delayed response"}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, Timeout: 30*time.Second})

	// Test with context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	resp, err := client.Do(ctx, "GET", "/delayed", nil)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestClient_Do_InvalidJSON(t *testing.T) {
	client := NewClient(Config{BaseURL: "https://api.example.com", Timeout: 30*time.Second})

	// Test with invalid JSON body (channel cannot be marshaled)
	invalidBody := make(chan int)

	ctx := context.Background()
	resp, err := client.Do(ctx, "POST", "/test", invalidBody)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "marshal body")
}

func TestClient_Do_InvalidURL(t *testing.T) {
	client := NewClient(Config{BaseURL: "invalid-url", Timeout: 30*time.Second})

	ctx := context.Background()
	resp, err := client.Do(ctx, "GET", "/test", nil)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestClient_Do_ServerError(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, Timeout: 30*time.Second})

	ctx := context.Background()
	resp, err := client.Do(ctx, "GET", "/error", nil)

	// Should not return error for HTTP error status codes
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	resp.Body.Close()
}

// TestClient_buildURL removed - buildURL method not implemented in current client

// TestClient_addHeaders removed - addHeaders method not implemented in current client

func TestClient_Timeout(t *testing.T) {
	// Create test server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "delayed response"}`))
	}))
	defer server.Close()

	// Create client with short timeout
	client := NewClient(Config{BaseURL: server.URL, Timeout: 100*time.Millisecond})

	ctx := context.Background()
	resp, err := client.Get(ctx, "/delayed")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestClient_RealWorldScenario(t *testing.T) {
	// Simulate a real-world API interaction
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth":
			assert.Equal(t, "POST", r.Method)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"token": "abc123", "expires_in": 3600}`))
		case "/users":
			assert.Equal(t, "GET", r.Method)
			// Check for API key header instead
			assert.Equal(t, "test-api-key", r.Header.Get("X-API-Key"))
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"users": [{"id": 1, "name": "John"}]}`))
		case "/users/1":
			assert.Equal(t, "PUT", r.Method)
			// Check for API key header instead
			assert.Equal(t, "test-api-key", r.Header.Get("X-API-Key"))
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": 1, "name": "John Updated"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "test-api-key", Timeout: 30*time.Second})
	ctx := context.Background()

	// Step 1: Authenticate
	authResp, err := client.Post(ctx, "/auth", map[string]string{
		"username": "testuser",
		"password": "testpass",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, authResp.StatusCode)
	authResp.Body.Close()

	// Step 2: Get users (API key automatically added by client)
	usersResp, err := client.Get(ctx, "/users")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, usersResp.StatusCode)
	usersResp.Body.Close()

	// Step 3: Update user
	updateResp, err := client.Put(ctx, "/users/1", map[string]string{
		"name": "John Updated",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)
	updateResp.Body.Close()
}

func BenchmarkClient_Get(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "benchmark"}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, Timeout: 30*time.Second})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(ctx, "/benchmark")
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkClient_Post(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 123}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, Timeout: 30*time.Second})
	ctx := context.Background()
	payload := map[string]string{"name": "benchmark"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Post(ctx, "/benchmark", payload)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}