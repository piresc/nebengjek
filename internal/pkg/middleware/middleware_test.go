package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/pkg/tracing"
	"github.com/stretchr/testify/assert"
)

func TestNewMiddleware(t *testing.T) {
	config := &models.Config{
		App: models.AppConfig{
			Name: "test-service",
		},
		APIKey: models.APIKeyConfig{
			GatewayService: "test-api-key",
		},
	}
	logger := slog.New(&testLogger{})
	tracer := &testTracer{}

	middleware := NewMiddleware(config, logger, tracer)

	assert.NotNil(t, middleware)
	assert.Equal(t, config, middleware.config)
	assert.Equal(t, logger, middleware.logger)
	assert.Equal(t, tracer, middleware.tracer)
}

func TestMiddleware_Handler(t *testing.T) {
	config := &models.Config{
		App: models.AppConfig{
			Name: "test-service",
		},
	}
	logger := slog.New(&testLogger{})
	tracer := &testTracer{}

	middleware := NewMiddleware(config, logger, tracer)
	handler := middleware.Handler()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test successful request
	err := handler(func(c echo.Context) error {
		return c.String(http.StatusOK, "test")
	})(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("X-Request-ID"))
}

func TestMiddleware_Handler_WithRequestID(t *testing.T) {
	config := &models.Config{
		App: models.AppConfig{
			Name: "test-service",
		},
	}
	logger := slog.New(&testLogger{})
	tracer := &testTracer{}

	middleware := NewMiddleware(config, logger, tracer)
	handler := middleware.Handler()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "test-request-id")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(func(c echo.Context) error {
		return c.String(http.StatusOK, "test")
	})(c)

	assert.NoError(t, err)
	assert.Equal(t, "test-request-id", rec.Header().Get("X-Request-ID"))
}

func TestMiddleware_Handler_WithError(t *testing.T) {
	config := &models.Config{
		App: models.AppConfig{
			Name: "test-service",
		},
	}
	logger := slog.New(&testLogger{})
	tracer := &testTracer{}

	middleware := NewMiddleware(config, logger, tracer)
	handler := middleware.Handler()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	testErr := fmt.Errorf("test error")
	err := handler(func(c echo.Context) error {
		return testErr
	})(c)

	assert.Error(t, err)
	assert.Equal(t, testErr, err)
}

func TestMiddleware_Handler_WithPanic(t *testing.T) {
	config := &models.Config{
		App: models.AppConfig{
			Name: "test-service",
		},
	}
	logger := slog.New(&testLogger{})
	tracer := &testTracer{}

	middleware := NewMiddleware(config, logger, tracer)
	handler := middleware.Handler()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(func(c echo.Context) error {
		panic("test panic")
	})(c)

	assert.NoError(t, err) // Panic is recovered and handled
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestMiddleware_APIKeyHandler(t *testing.T) {
	config := &models.Config{
		APIKey: models.APIKeyConfig{
			GatewayService: "correct-api-key",
		},
	}
	logger := slog.New(&testLogger{})
	tracer := &testTracer{}

	middleware := NewMiddleware(config, logger, tracer)
	handler := middleware.APIKeyHandler("gateway-service")

	e := echo.New()

	t.Run("Valid API key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-API-Key", "correct-api-key")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler(func(c echo.Context) error {
			return c.String(http.StatusOK, "test")
		})(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Missing API key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler(func(c echo.Context) error {
			return c.String(http.StatusOK, "test")
		})(c)

		assert.Error(t, err)
		assert.Equal(t, http.StatusUnauthorized, err.(*echo.HTTPError).Code)
	})

	t.Run("Invalid API key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-API-Key", "wrong-api-key")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler(func(c echo.Context) error {
			return c.String(http.StatusOK, "test")
		})(c)

		assert.Error(t, err)
		assert.Equal(t, http.StatusUnauthorized, err.(*echo.HTTPError).Code)
	})

	t.Run("Unknown service", func(t *testing.T) {
		handler := middleware.APIKeyHandler("unknown-service")
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-API-Key", "some-key")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler(func(c echo.Context) error {
			return c.String(http.StatusOK, "test")
		})(c)

		assert.Error(t, err)
		assert.Equal(t, http.StatusUnauthorized, err.(*echo.HTTPError).Code)
	})
}

func TestMiddleware_JWTHandler(t *testing.T) {
	config := &models.Config{
		JWT: models.JWTConfig{
			Secret: "test-secret-key",
		},
	}
	logger := slog.New(&testLogger{})
	tracer := &testTracer{}

	middleware := NewMiddleware(config, logger, tracer)
	handler := middleware.JWTHandler()

	e := echo.New()

	t.Run("Valid JWT token", func(t *testing.T) {
		// Create a valid JWT token
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": "test-user-id",
			"role":    "user",
			"exp":     time.Now().Add(time.Hour).Unix(),
		})
		tokenString, _ := token.SignedString([]byte(config.JWT.Secret))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler(func(c echo.Context) error {
			return c.String(http.StatusOK, "test")
		})(c)

		assert.NoError(t, err)
	})

	t.Run("Invalid JWT token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler(func(c echo.Context) error {
			return c.String(http.StatusOK, "test")
		})(c)

		assert.Error(t, err)
	})
}

func TestMiddleware_RegisterHealthEndpoints(t *testing.T) {
	config := &models.Config{
		App: models.AppConfig{
			Name: "test-service",
		},
	}
	logger := slog.New(&testLogger{})
	tracer := &testTracer{}

	middleware := NewMiddleware(config, logger, tracer)

	e := echo.New()
	healthChecker := func(ctx context.Context) (map[string]interface{}, error) {
		return map[string]interface{}{
			"status": "healthy",
			"database": "connected",
		}, nil
	}

	middleware.RegisterHealthEndpoints(e, "test-service", "1.0.0", healthChecker)

	t.Run("Basic health check", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "ok", response["status"])
		assert.Equal(t, "test-service", response["service"])
	})

	t.Run("Detailed health check", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health/detailed", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "healthy", response["status"])
		assert.Equal(t, "test-service", response["service"])
		assert.Equal(t, "1.0.0", response["version"])
		assert.Equal(t, "connected", response["database"])
	})

	t.Run("Readiness check", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "ready", response["status"])
		assert.Equal(t, "test-service", response["service"])
	})

	t.Run("Liveness check", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "alive", response["status"])
		assert.Equal(t, "test-service", response["service"])
	})
}

func TestMiddleware_RegisterHealthEndpoints_WithError(t *testing.T) {
	config := &models.Config{
		App: models.AppConfig{
			Name: "test-service",
		},
	}
	logger := slog.New(&testLogger{})
	tracer := &testTracer{}

	middleware := NewMiddleware(config, logger, tracer)

	e := echo.New()
	healthChecker := func(ctx context.Context) (map[string]interface{}, error) {
		return nil, fmt.Errorf("database connection failed")
	}

	middleware.RegisterHealthEndpoints(e, "test-service", "1.0.0", healthChecker)

	t.Run("Detailed health check with error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health/detailed", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "unhealthy", response["status"])
		assert.Equal(t, "test-service", response["service"])
		assert.Equal(t, "1.0.0", response["version"])
		assert.Equal(t, "database connection failed", response["error"])
	})

	t.Run("Readiness check with error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "not ready", response["status"])
		assert.Equal(t, "test-service", response["service"])
		assert.Equal(t, "database connection failed", response["error"])
	})
}

func TestResponseBodyCapture(t *testing.T) {
	t.Run("Write body capture", func(t *testing.T) {
		underlyingWriter := httptest.NewRecorder()
		capture := &responseBodyCapture{
			ResponseWriter: underlyingWriter,
			body:           make([]byte, 0),
		}

		data := []byte("test response data")
		n, err := capture.Write(data)

		assert.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, data, capture.body)
		assert.Equal(t, data, underlyingWriter.Body.Bytes())
	})

	t.Run("Write body limit", func(t *testing.T) {
		underlyingWriter := httptest.NewRecorder()
		capture := &responseBodyCapture{
			ResponseWriter: underlyingWriter,
			body:           make([]byte, 0),
		}

		// Write more than 1KB to test the limit
		largeData := make([]byte, 1500)
		for i := range largeData {
			largeData[i] = 'A'
		}

		n, err := capture.Write(largeData)

		assert.NoError(t, err)
		assert.Equal(t, len(largeData), n)
		assert.Equal(t, 1024, len(capture.body)) // Should be limited to 1KB
		assert.Equal(t, largeData, underlyingWriter.Body.Bytes())
	})
}

func TestMiddleware_extractErrorFromResponse(t *testing.T) {
	config := &models.Config{}
	logger := slog.New(&testLogger{})
	tracer := &testTracer{}

	middleware := NewMiddleware(config, logger, tracer)

	t.Run("Empty response body", func(t *testing.T) {
		errorDetails := middleware.extractErrorFromResponse([]byte{}, 500)
		assert.Equal(t, "Internal server error (HTTP 500)", errorDetails)

		errorDetails = middleware.extractErrorFromResponse([]byte{}, 400)
		assert.Equal(t, "Client error (HTTP 400)", errorDetails)
	})

	t.Run("JSON error response", func(t *testing.T) {
		errorResp := map[string]interface{}{
			"error": "Test error message",
		}
		jsonData, _ := json.Marshal(errorResp)

		errorDetails := middleware.extractErrorFromResponse(jsonData, 500)
		assert.Equal(t, "Test error message", errorDetails)
	})

	t.Run("JSON message response", func(t *testing.T) {
		errorResp := map[string]interface{}{
			"message": "Test message",
		}
		jsonData, _ := json.Marshal(errorResp)

		errorDetails := middleware.extractErrorFromResponse(jsonData, 400)
		assert.Equal(t, "Test message", errorDetails)
	})

	t.Run("Non-JSON response", func(t *testing.T) {
		errorDetails := middleware.extractErrorFromResponse([]byte("Plain text error"), 500)
		assert.Equal(t, "Plain text error", errorDetails)
	})

	t.Run("Long response truncation", func(t *testing.T) {
		longText := strings.Repeat("A", 600)
		errorDetails := middleware.extractErrorFromResponse([]byte(longText), 500)
		assert.Equal(t, longText[:500]+"...", errorDetails)
	})
}

// Test helpers

type testLogger struct {
	logs []map[string]interface{}
}

func (l *testLogger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.logs = append(l.logs, map[string]interface{}{
		"level":   "info",
		"message": msg,
		"args":    args,
	})
}

func (l *testLogger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.logs = append(l.logs, map[string]interface{}{
		"level":   "error",
		"message": msg,
		"args":    args,
	})
}

func (l *testLogger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.logs = append(l.logs, map[string]interface{}{
		"level":   "warn",
		"message": msg,
		"args":    args,
	})
}

func (l *testLogger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.logs = append(l.logs, map[string]interface{}{
		"level":   "debug",
		"message": msg,
		"args":    args,
	})
}

func (l *testLogger) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (l *testLogger) Handle(ctx context.Context, record slog.Record) error {
	return nil
}

func (l *testLogger) WithAttrs(attrs []slog.Attr) slog.Handler {
	return l
}

func (l *testLogger) WithGroup(name string) slog.Handler {
	return l
}

type testTracer struct {
	enabled bool
}

func (t *testTracer) StartHTTPRequest(req *http.Request) (context.Context, tracing.HTTPTransaction) {
	if !t.enabled {
		return req.Context(), &testHTTPTransaction{}
	}
	return req.Context(), &testHTTPTransaction{enabled: true}
}

func (t *testTracer) StartExternalCall(ctx context.Context, host, method string) (context.Context, func()) {
	return ctx, func() {}
}

func (t *testTracer) StartMessage(ctx context.Context, topic, operation string) (context.Context, func()) {
	return ctx, func() {}
}

func (t *testTracer) StartDatabase(ctx context.Context, operation, table string) (context.Context, func()) {
	return ctx, func() {}
}

func (t *testTracer) StartSegment(ctx context.Context, name string) (context.Context, func()) {
	return ctx, func() {}
}

func (t *testTracer) InjectContext(ctx context.Context, carrier interface{}) {
}

func (t *testTracer) ExtractContext(ctx context.Context, carrier interface{}) context.Context {
	return ctx
}

func (t *testTracer) IsEnabled() bool {
	return t.enabled
}

func (t *testTracer) ShouldTrace(path string) bool {
	return t.enabled
}

type testHTTPTransaction struct {
	enabled bool
	attrs   map[string]interface{}
	errors  []error
}

func (t *testHTTPTransaction) Context() context.Context {
	return context.Background()
}

func (t *testHTTPTransaction) SetName(name string) {}

func (t *testHTTPTransaction) AddAttribute(key, value string) {
	if t.attrs == nil {
		t.attrs = make(map[string]interface{})
	}
	t.attrs[key] = value
}

func (t *testHTTPTransaction) AddError(err error) {
	t.errors = append(t.errors, err)
}

func (t *testHTTPTransaction) End() {}