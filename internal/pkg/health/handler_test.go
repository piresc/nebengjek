package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildInfo(t *testing.T) {
	t.Run("Default build info structure", func(t *testing.T) {
		assert.Equal(t, "development", DefaultBuildInfo.Version)
		assert.Equal(t, "unknown", DefaultBuildInfo.GitCommit)
		assert.Equal(t, "unknown", DefaultBuildInfo.BuildTime)
		assert.Equal(t, runtime.Version(), DefaultBuildInfo.GoVersion)
		assert.Empty(t, DefaultBuildInfo.ServiceName)
		assert.Empty(t, DefaultBuildInfo.Hostname)
		assert.True(t, DefaultBuildInfo.ServerTime.IsZero())
	})

	t.Run("BuildInfo JSON serialization", func(t *testing.T) {
		buildInfo := BuildInfo{
			Version:     "1.0.0",
			GitCommit:   "abc123",
			BuildTime:   "2023-01-01T00:00:00Z",
			ServiceName: "test-service",
			GoVersion:   "go1.19",
			Hostname:    "test-host",
			ServerTime:  time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		jsonData, err := json.Marshal(buildInfo)
		require.NoError(t, err)

		var unmarshaled BuildInfo
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, buildInfo.Version, unmarshaled.Version)
		assert.Equal(t, buildInfo.GitCommit, unmarshaled.GitCommit)
		assert.Equal(t, buildInfo.BuildTime, unmarshaled.BuildTime)
		assert.Equal(t, buildInfo.ServiceName, unmarshaled.ServiceName)
		assert.Equal(t, buildInfo.GoVersion, unmarshaled.GoVersion)
		assert.Equal(t, buildInfo.Hostname, unmarshaled.Hostname)
	})
}

func TestNewPingHandler(t *testing.T) {
	// Save original environment variables
	originalEnv := make(map[string]string)
	envVars := []string{"VERSION", "GIT_COMMIT", "BUILD_TIME"}

	for _, envVar := range envVars {
		if val, exists := os.LookupEnv(envVar); exists {
			originalEnv[envVar] = val
		}
		os.Unsetenv(envVar)
	}

	// Restore environment after test
	defer func() {
		for _, envVar := range envVars {
			os.Unsetenv(envVar)
			if val, exists := originalEnv[envVar]; exists {
				os.Setenv(envVar, val)
			}
		}
	}()

	t.Run("Default ping handler", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := NewPingHandler("test-service")
		err := handler(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var response BuildInfo
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "test-service", response.ServiceName)
		assert.Equal(t, "development", response.Version)
		assert.Equal(t, "unknown", response.GitCommit)
		assert.Equal(t, "unknown", response.BuildTime)
		assert.Equal(t, runtime.Version(), response.GoVersion)
		assert.NotEmpty(t, response.Hostname)
		assert.False(t, response.ServerTime.IsZero())
	})

	t.Run("Ping handler with environment variables", func(t *testing.T) {
		// Set environment variables
		os.Setenv("VERSION", "2.0.0")
		os.Setenv("GIT_COMMIT", "def456")
		os.Setenv("BUILD_TIME", "2023-06-01T12:00:00Z")

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := NewPingHandler("prod-service")
		err := handler(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var response BuildInfo
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "prod-service", response.ServiceName)
		assert.Equal(t, "2.0.0", response.Version)
		assert.Equal(t, "def456", response.GitCommit)
		assert.Equal(t, "2023-06-01T12:00:00Z", response.BuildTime)
		assert.Equal(t, runtime.Version(), response.GoVersion)
		assert.NotEmpty(t, response.Hostname)
		assert.False(t, response.ServerTime.IsZero())
	})

	t.Run("Ping handler with empty service name", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := NewPingHandler("")
		err := handler(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var response BuildInfo
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "", response.ServiceName)
	})

	t.Run("Ping handler with partial environment variables", func(t *testing.T) {
		// Set only some environment variables
		os.Setenv("VERSION", "1.5.0")
		// Explicitly unset GIT_COMMIT and BUILD_TIME
		os.Unsetenv("GIT_COMMIT")
		os.Unsetenv("BUILD_TIME")

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := NewPingHandler("partial-service")
		err := handler(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var response BuildInfo
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "partial-service", response.ServiceName)
		assert.Equal(t, "1.5.0", response.Version)
		assert.Equal(t, "unknown", response.GitCommit) // Should use default
		assert.Equal(t, "unknown", response.BuildTime) // Should use default
	})

	t.Run("Multiple calls return updated server time", func(t *testing.T) {
		e := echo.New()
		handler := NewPingHandler("time-test-service")

		// First call
		req1 := httptest.NewRequest(http.MethodGet, "/ping", nil)
		rec1 := httptest.NewRecorder()
		c1 := e.NewContext(req1, rec1)
		err := handler(c1)
		assert.NoError(t, err)

		var response1 BuildInfo
		err = json.Unmarshal(rec1.Body.Bytes(), &response1)
		require.NoError(t, err)

		// Wait a bit
		time.Sleep(10 * time.Millisecond)

		// Second call
		req2 := httptest.NewRequest(http.MethodGet, "/ping", nil)
		rec2 := httptest.NewRecorder()
		c2 := e.NewContext(req2, rec2)
		err = handler(c2)
		assert.NoError(t, err)

		var response2 BuildInfo
		err = json.Unmarshal(rec2.Body.Bytes(), &response2)
		require.NoError(t, err)

		// Server time should be different (later)
		assert.True(t, response2.ServerTime.After(response1.ServerTime))
		// Other fields should be the same
		assert.Equal(t, response1.ServiceName, response2.ServiceName)
		assert.Equal(t, response1.Version, response2.Version)
	})
}

func TestRegisterHealthEndpoints(t *testing.T) {
	t.Run("Register all health endpoints", func(t *testing.T) {
		e := echo.New()
		RegisterHealthEndpoints(e, "health-test-service")

		// Test /ping endpoint
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var buildInfo BuildInfo
		err := json.Unmarshal(rec.Body.Bytes(), &buildInfo)
		assert.NoError(t, err)
		assert.Equal(t, "health-test-service", buildInfo.ServiceName)

		// Test /health endpoint
		req = httptest.NewRequest(http.MethodGet, "/health", nil)
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "OK", rec.Body.String())

		// Test /healthz endpoint
		req = httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "OK", rec.Body.String())

		// Test /ready endpoint
		req = httptest.NewRequest(http.MethodGet, "/ready", nil)
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "OK", rec.Body.String())
	})

	t.Run("Health endpoints with different HTTP methods", func(t *testing.T) {
		e := echo.New()
		RegisterHealthEndpoints(e, "method-test-service")

		// Test POST request to health endpoint (should return 405 Method Not Allowed)
		req := httptest.NewRequest(http.MethodPost, "/health", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)

		// Test PUT request to ready endpoint (should return 405 Method Not Allowed)
		req = httptest.NewRequest(http.MethodPut, "/ready", nil)
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	})

	t.Run("Non-existent endpoint", func(t *testing.T) {
		e := echo.New()
		RegisterHealthEndpoints(e, "notfound-test-service")

		// Test non-existent endpoint
		req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("Multiple service registrations", func(t *testing.T) {
		e1 := echo.New()
		e2 := echo.New()

		RegisterHealthEndpoints(e1, "service-1")
		RegisterHealthEndpoints(e2, "service-2")

		// Test service 1
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		rec := httptest.NewRecorder()
		e1.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var buildInfo1 BuildInfo
		err := json.Unmarshal(rec.Body.Bytes(), &buildInfo1)
		assert.NoError(t, err)
		assert.Equal(t, "service-1", buildInfo1.ServiceName)

		// Test service 2
		req = httptest.NewRequest(http.MethodGet, "/ping", nil)
		rec = httptest.NewRecorder()
		e2.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var buildInfo2 BuildInfo
		err = json.Unmarshal(rec.Body.Bytes(), &buildInfo2)
		assert.NoError(t, err)
		assert.Equal(t, "service-2", buildInfo2.ServiceName)
	})
}

func TestHealthEndpointsIntegration(t *testing.T) {
	t.Run("Full health check flow", func(t *testing.T) {
		e := echo.New()
		RegisterHealthEndpoints(e, "integration-test-service")

		// Start server
		server := httptest.NewServer(e)
		defer server.Close()

		// Test all endpoints
		endpoints := []string{"/ping", "/health", "/healthz", "/ready"}

		for _, endpoint := range endpoints {
			resp, err := http.Get(server.URL + endpoint)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
		}
	})
}

func TestBuildInfoEnvironmentHandling(t *testing.T) {
	// Save original environment variables
	originalEnv := make(map[string]string)
	envVars := []string{"VERSION", "GIT_COMMIT", "BUILD_TIME"}

	for _, envVar := range envVars {
		if val, exists := os.LookupEnv(envVar); exists {
			originalEnv[envVar] = val
		}
		os.Unsetenv(envVar)
	}

	// Restore environment after test
	defer func() {
		for _, envVar := range envVars {
			os.Unsetenv(envVar)
			if val, exists := originalEnv[envVar]; exists {
				os.Setenv(envVar, val)
			}
		}
	}()

	t.Run("Empty environment variables", func(t *testing.T) {
		os.Setenv("VERSION", "")
		os.Setenv("GIT_COMMIT", "")
		os.Setenv("BUILD_TIME", "")

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := NewPingHandler("empty-env-service")
		err := handler(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var response BuildInfo
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		// Empty environment variables should not override defaults
		assert.Equal(t, "development", response.Version)
		assert.Equal(t, "unknown", response.GitCommit)
		assert.Equal(t, "unknown", response.BuildTime)
	})

	t.Run("Special characters in environment variables", func(t *testing.T) {
		os.Setenv("VERSION", "v1.0.0-beta+build.123")
		os.Setenv("GIT_COMMIT", "abc123def456!@#$%^&*()")
		os.Setenv("BUILD_TIME", "2023-12-25T23:59:59.999Z")

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := NewPingHandler("special-chars-service")
		err := handler(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var response BuildInfo
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "v1.0.0-beta+build.123", response.Version)
		assert.Equal(t, "abc123def456!@#$%^&*()", response.GitCommit)
		assert.Equal(t, "2023-12-25T23:59:59.999Z", response.BuildTime)
	})
}

func BenchmarkNewPingHandler(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewPingHandler("benchmark-service")
	}
}

func BenchmarkPingHandlerExecution(b *testing.B) {
	e := echo.New()
	handler := NewPingHandler("benchmark-service")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler(c)
	}
}

func BenchmarkRegisterHealthEndpoints(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e := echo.New()
		RegisterHealthEndpoints(e, "benchmark-service")
	}
}