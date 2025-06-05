package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestPanicRecoveryWithZapMiddleware(t *testing.T) {
	// Create a buffer to capture log output
	var logBuffer bytes.Buffer

	// Create a zap logger that writes to our buffer
	config := zap.NewDevelopmentConfig()
	config.OutputPaths = []string{"stdout"}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(config.EncoderConfig),
		zapcore.AddSync(&logBuffer),
		zapcore.DebugLevel,
	)
	zapLogger := zap.New(core)

	// Create ZapLogger wrapper
	zapLoggerWrapper := &logger.ZapLogger{Logger: zapLogger}

	tests := []struct {
		name         string
		panicValue   interface{}
		expectStatus int
		expectInLogs []string
		setupContext func(c echo.Context)
	}{
		{
			name:         "string panic",
			panicValue:   "test panic message",
			expectStatus: http.StatusInternalServerError,
			expectInLogs: []string{
				"test panic message",
				"stack_trace",
				"panic_type",
				"Panic recovered during request processing",
			},
		},
		{
			name:         "error panic",
			panicValue:   fmt.Errorf("test error panic"),
			expectStatus: http.StatusInternalServerError,
			expectInLogs: []string{
				"test error panic",
				"stack_trace",
				"*errors.errorString",
			},
		},
		{
			name:         "nil panic",
			panicValue:   nil,
			expectStatus: http.StatusInternalServerError,
			expectInLogs: []string{
				"panic_value",
				"stack_trace",
			},
		},
		{
			name:         "panic with user context",
			panicValue:   "user context panic",
			expectStatus: http.StatusInternalServerError,
			expectInLogs: []string{
				"user context panic",
				"user123",
			},
			setupContext: func(c echo.Context) {
				c.Set("user_id", "user123")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset log buffer
			logBuffer.Reset()

			// Create Echo instance
			e := echo.New()

			// Create test handler that panics
			panicHandler := func(c echo.Context) error {
				if tt.setupContext != nil {
					tt.setupContext(c)
				}
				panic(tt.panicValue)
			}

			// Apply middleware
			middleware := PanicRecoveryWithZapMiddleware(zapLoggerWrapper)
			handler := middleware(panicHandler)

			// Create test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("User-Agent", "test-agent")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Execute handler (should not panic)
			err := handler(c)
			assert.NoError(t, err)

			// Check response status
			assert.Equal(t, tt.expectStatus, rec.Code)

			// Check response body
			var response map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, "Internal Server Error", response["error"])
			assert.Equal(t, "An unexpected error occurred while processing your request", response["message"])

			// Check logs
			logOutput := logBuffer.String()
			for _, expectedLog := range tt.expectInLogs {
				assert.Contains(t, logOutput, expectedLog, "Expected log content not found")
			}

			// Verify essential log fields are present
			assert.Contains(t, logOutput, "GET")        // method
			assert.Contains(t, logOutput, "/test")      // path
			assert.Contains(t, logOutput, "test-agent") // user agent
		})
	}
}

func TestPanicRecoveryWithNewRelicIntegration(t *testing.T) {
	// Create a buffer to capture log output
	var logBuffer bytes.Buffer

	// Create a zap logger that writes to our buffer
	config := zap.NewDevelopmentConfig()
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(config.EncoderConfig),
		zapcore.AddSync(&logBuffer),
		zapcore.DebugLevel,
	)
	zapLogger := zap.New(core)
	zapLoggerWrapper := &logger.ZapLogger{Logger: zapLogger}

	// Create a mock New Relic application (for testing, we'll use a real one with a fake key)
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("test-app"),
		newrelic.ConfigLicense("0000000000000000000000000000000000000000"), // fake license
		newrelic.ConfigEnabled(false),                                      // disable actual reporting
	)
	require.NoError(t, err)

	// Create Echo instance
	e := echo.New()

	// Create test handler that panics
	panicHandler := func(c echo.Context) error {
		panic("new relic panic test")
	}

	// Apply middleware
	middleware := PanicRecoveryWithZapMiddleware(zapLoggerWrapper)
	handler := middleware(panicHandler)

	// Create test request
	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	req.Header.Set("User-Agent", "new-relic-test-agent")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Add New Relic transaction to context
	txn := app.StartTransaction("test-transaction")
	c.Set("nr_txn", txn)

	// Execute handler
	err = handler(c)
	assert.NoError(t, err)

	// End transaction
	txn.End()

	// Check response
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	// Check logs contain New Relic context
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "new relic panic test")
	assert.Contains(t, logOutput, "POST")
	assert.Contains(t, logOutput, "/api/test")
	assert.Contains(t, logOutput, "new-relic-test-agent")
}

func TestEnhancedPanicRecoveryMiddleware(t *testing.T) {
	// Create a buffer to capture log output
	var logBuffer bytes.Buffer

	config := zap.NewDevelopmentConfig()
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(config.EncoderConfig),
		zapcore.AddSync(&logBuffer),
		zapcore.DebugLevel,
	)
	zapLogger := zap.New(core)
	zapLoggerWrapper := &logger.ZapLogger{Logger: zapLogger}

	// Create Echo instance
	e := echo.New()

	// Create test handler that panics
	panicHandler := func(c echo.Context) error {
		c.Set("user_id", "enhanced-user-123")
		panic("enhanced panic test")
	}

	// Apply enhanced middleware
	middleware := EnhancedPanicRecoveryMiddleware(zapLoggerWrapper)
	handler := middleware(panicHandler)

	// Create test request with headers
	req := httptest.NewRequest("PUT", "/api/enhanced?param=value", strings.NewReader(`{"test": "data"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "enhanced-test-agent")
	req.Header.Set("X-Request-ID", "test-request-123")
	req.Header.Set("Authorization", "Bearer secret-token") // Should be filtered out

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute handler
	err := handler(c)
	assert.NoError(t, err)

	// Check response
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "test-request-123", response["request_id"])

	// Check enhanced logs
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "enhanced panic test")
	assert.Contains(t, logOutput, "enhanced-user-123")
	assert.Contains(t, logOutput, "PUT")
	assert.Contains(t, logOutput, "/api/enhanced")
	assert.Contains(t, logOutput, "param=value")
	assert.Contains(t, logOutput, "application/json")
	assert.Contains(t, logOutput, "enhanced-test-agent")
	assert.Contains(t, logOutput, "test-request-123")
	assert.Contains(t, logOutput, "runtime")
	assert.Contains(t, logOutput, "goroutines")
	assert.Contains(t, logOutput, "memory")

	// Verify authorization header is NOT logged (security)
	assert.NotContains(t, logOutput, "Bearer secret-token")
	assert.NotContains(t, logOutput, "secret-token")
}

func TestPanicRecoveryConfig(t *testing.T) {
	config := DefaultPanicRecoveryConfig()

	assert.Equal(t, 4<<10, config.StackSize) // 4 KB
	assert.False(t, config.DisableStackAll)
	assert.Nil(t, config.Logger)
}

func TestPanicRecoveryMiddleware_RequiresLogger(t *testing.T) {
	config := PanicRecoveryConfig{
		StackSize:       1024,
		DisableStackAll: false,
		Logger:          nil, // No logger provided
	}

	assert.Panics(t, func() {
		PanicRecoveryMiddleware(config)
	}, "Should panic when no logger is provided")
}

func TestExtractSafeHeaders(t *testing.T) {
	headers := http.Header{
		"Content-Type":  []string{"application/json"},
		"User-Agent":    []string{"test-agent"},
		"Authorization": []string{"Bearer secret"},
		"Cookie":        []string{"session=secret"},
		"X-Api-Key":     []string{"secret-key"},
		"X-Request-ID":  []string{"request-123"},
	}

	safe := extractSafeHeaders(headers)

	// Should include safe headers
	assert.Equal(t, "application/json", safe["Content-Type"])
	assert.Equal(t, "test-agent", safe["User-Agent"])
	assert.Equal(t, "request-123", safe["X-Request-ID"])

	// Should exclude sensitive headers
	assert.NotContains(t, safe, "Authorization")
	assert.NotContains(t, safe, "Cookie")
	assert.NotContains(t, safe, "X-Api-Key")
}

func TestGetMemoryStats(t *testing.T) {
	stats := getMemoryStats()

	// Should contain expected fields
	assert.Contains(t, stats, "alloc_mb")
	assert.Contains(t, stats, "total_alloc_mb")
	assert.Contains(t, stats, "sys_mb")
	assert.Contains(t, stats, "gc_cycles")

	// Values should be reasonable
	assert.IsType(t, uint64(0), stats["alloc_mb"])
	assert.GreaterOrEqual(t, stats["alloc_mb"].(uint64), uint64(0))
}

func TestGetCaller(t *testing.T) {
	caller := getCaller(1)

	assert.Contains(t, caller, "panic_recovery_test.go")
	assert.Contains(t, caller, "TestGetCaller")
}

func TestSendPanicResponse(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	sendPanicResponse(c, "test-request-456")

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Internal Server Error", response["error"])
	assert.Equal(t, "An unexpected error occurred while processing your request", response["message"])
	assert.Equal(t, "test-request-456", response["request_id"])
}

func TestSendPanicResponse_NoRequestID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	sendPanicResponse(c, "")

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Internal Server Error", response["error"])
	assert.Equal(t, "An unexpected error occurred while processing your request", response["message"])
	assert.NotContains(t, response, "request_id")
}

// Benchmark tests for performance
func BenchmarkPanicRecoveryMiddleware(b *testing.B) {
	// Create logger
	config := zap.NewDevelopmentConfig()
	config.OutputPaths = []string{"/dev/null"} // Discard output for benchmark
	zapLogger, _ := config.Build()
	zapLoggerWrapper := &logger.ZapLogger{Logger: zapLogger}

	// Create Echo and middleware
	e := echo.New()
	middleware := PanicRecoveryWithZapMiddleware(zapLoggerWrapper)

	// Normal handler (no panic)
	normalHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	}

	handler := middleware(normalHandler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler(c)
	}
}

func BenchmarkPanicRecoveryMiddleware_WithPanic(b *testing.B) {
	// Create logger that discards output
	config := zap.NewDevelopmentConfig()
	config.OutputPaths = []string{"/dev/null"}
	zapLogger, _ := config.Build()
	zapLoggerWrapper := &logger.ZapLogger{Logger: zapLogger}

	// Create Echo and middleware
	e := echo.New()
	middleware := PanicRecoveryWithZapMiddleware(zapLoggerWrapper)

	// Panic handler
	panicHandler := func(c echo.Context) error {
		panic("benchmark panic")
	}

	handler := middleware(panicHandler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler(c)
	}
}
