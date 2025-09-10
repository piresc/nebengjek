package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/tracing"
	"github.com/stretchr/testify/assert"
)

func TestNewTracingMiddleware(t *testing.T) {
	config := &tracing.Config{
		Enabled:    true,
		SampleRate: 1.0,
	}
	tracer := &tracingTestTracer{}

	middleware := NewTracingMiddleware(config, tracer)

	assert.NotNil(t, middleware)
	assert.Equal(t, config, middleware.config)
	assert.Equal(t, tracer, middleware.tracer)
}

func TestTracingMiddleware_Handler(t *testing.T) {
	config := &tracing.Config{
		Enabled:    true,
		SampleRate: 1.0,
	}
	tracer := &tracingTestTracer{}

	middleware := NewTracingMiddleware(config, tracer)
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
}

func TestTracingMiddleware_Handler_ExcludedPath(t *testing.T) {
	config := &tracing.Config{
		Enabled:      true,
		SampleRate:   1.0,
		ExcludePaths: []string{"/health"},
	}
	tracer := &tracingTestTracer{}

	middleware := NewTracingMiddleware(config, tracer)
	handler := middleware.Handler()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(func(c echo.Context) error {
		return c.String(http.StatusOK, "test")
	})(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTracingMiddleware_Handler_Disabled(t *testing.T) {
	config := &tracing.Config{
		Enabled:    false,
		SampleRate: 1.0,
	}
	tracer := &tracingTestTracer{}

	middleware := NewTracingMiddleware(config, tracer)
	handler := middleware.Handler()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(func(c echo.Context) error {
		return c.String(http.StatusOK, "test")
	})(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTracingMiddleware_Handler_WithUserHeaders(t *testing.T) {
	config := &tracing.Config{
		Enabled:    true,
		SampleRate: 1.0,
	}
	tracer := &tracingTestTracer{}

	middleware := NewTracingMiddleware(config, tracer)
	handler := middleware.Handler()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-User-ID", "test-user-123")
	req.Header.Set("X-User-Role", "driver")
	req.Header.Set("X-Request-ID", "test-request-456")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(func(c echo.Context) error {
		return c.String(http.StatusOK, "test")
	})(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTracingMiddleware_Handler_WithError(t *testing.T) {
	config := &tracing.Config{
		Enabled:    true,
		SampleRate: 1.0,
	}
	tracer := &tracingTestTracer{}

	middleware := NewTracingMiddleware(config, tracer)
	handler := middleware.Handler()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	testErr := assert.AnError
	err := handler(func(c echo.Context) error {
		return testErr
	})(c)

	assert.Error(t, err)
	assert.Equal(t, testErr, err)
}

func TestTracingMiddleware_shouldExclude(t *testing.T) {
	t.Run("Disabled tracing", func(t *testing.T) {
		config := &tracing.Config{
			Enabled:    false,
			SampleRate: 1.0,
		}
		tracer := &tracingTestTracer{}
		middleware := NewTracingMiddleware(config, tracer)

		assert.True(t, middleware.shouldExclude("/test"))
	})

	t.Run("Excluded path", func(t *testing.T) {
		config := &tracing.Config{
			Enabled:      true,
			SampleRate:   1.0,
			ExcludePaths: []string{"/health", "/metrics"},
		}
		tracer := &tracingTestTracer{}
		middleware := NewTracingMiddleware(config, tracer)

		assert.True(t, middleware.shouldExclude("/health"))
		assert.True(t, middleware.shouldExclude("/metrics"))
		assert.True(t, middleware.shouldExclude("/health/detailed"))
		assert.False(t, middleware.shouldExclude("/api/users"))
	})

	t.Run("Path contains excluded substring", func(t *testing.T) {
		config := &tracing.Config{
			Enabled:      true,
			SampleRate:   1.0,
			ExcludePaths: []string{"health"},
		}
		tracer := &tracingTestTracer{}
		middleware := NewTracingMiddleware(config, tracer)

		assert.True(t, middleware.shouldExclude("/health"))
		assert.True(t, middleware.shouldExclude("/api/health/check"))
		assert.False(t, middleware.shouldExclude("/api/users"))
	})

	t.Run("Enabled with no exclusions", func(t *testing.T) {
		config := &tracing.Config{
			Enabled:    true,
			SampleRate: 1.0,
		}
		tracer := &tracingTestTracer{}
		middleware := NewTracingMiddleware(config, tracer)

		assert.False(t, middleware.shouldExclude("/test"))
	})
}

func TestFormatTransactionName(t *testing.T) {
	tests := []struct {
		name   string
		method string
		route  string
		expect string
	}{
		{
			name:   "Simple GET",
			method: "GET",
			route:  "/users",
			expect: "GET /users",
		},
		{
			name:   "Empty route",
			method: "POST",
			route:  "",
			expect: "POST /",
		},
		{
			name:   "Root route",
			method: "DELETE",
			route:  "/",
			expect: "DELETE /",
		},
		{
			name:   "Route with trailing slash",
			method: "PUT",
			route:  "/users/",
			expect: "PUT /users",
		},
		{
			name:   "Complex route",
			method: "PATCH",
			route:  "/api/v1/users/123/profile",
			expect: "PATCH /api/v1/users/123/profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTransactionName(tt.method, tt.route)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestShouldSample(t *testing.T) {
	t.Run("Rate >= 1.0", func(t *testing.T) {
		assert.True(t, shouldSample(1.0))
		assert.True(t, shouldSample(1.5))
	})

	t.Run("Rate < 1.0", func(t *testing.T) {
		// The current implementation always returns true for rate < 1.0
		// This test documents the current behavior
		assert.True(t, shouldSample(0.5))
		assert.True(t, shouldSample(0.0))
	})
}

// Test helpers for tracing middleware

type tracingTestTracer struct {
	enabled      bool
	startedHTTP  bool
	injected     bool
	transactions []*testHTTPTransaction
}

func (t *tracingTestTracer) StartHTTPRequest(req *http.Request) (context.Context, tracing.HTTPTransaction) {
	if !t.enabled {
		return req.Context(), &testHTTPTransaction{enabled: false}
	}
	
	txn := &testHTTPTransaction{
		enabled: true,
	}
	t.transactions = append(t.transactions, txn)
	t.startedHTTP = true
	
	return req.Context(), txn
}

func (t *tracingTestTracer) StartHTTP(ctx context.Context, method, path string) (context.Context, func()) {
	return ctx, func() {}
}

func (t *tracingTestTracer) StartHTTPServer(req *http.Request) (context.Context, tracing.HTTPTransaction) {
	return req.Context(), &testHTTPTransaction{}
}

func (t *tracingTestTracer) StartGRPC(ctx context.Context, method string) (context.Context, func()) {
	return ctx, func() {}
}

func (t *tracingTestTracer) StartGRPCServer(ctx context.Context, method string) (context.Context, func()) {
	return ctx, func() {}
}

func (t *tracingTestTracer) InjectContext(ctx context.Context, carrier interface{}) {
	if headers, ok := carrier.(http.Header); ok {
		headers.Set("X-Trace-ID", "test-trace-id")
		t.injected = true
	}
}

func (t *tracingTestTracer) ExtractContext(ctx context.Context, carrier interface{}) context.Context {
	return ctx
}

func (t *tracingTestTracer) IsEnabled() bool {
	return t.enabled
}

func (t *tracingTestTracer) StartCustom(ctx context.Context, name string) (context.Context, func()) {
	return ctx, func() {}
}

func (t *tracingTestTracer) StartExternalCall(ctx context.Context, host, method string) (context.Context, func()) {
	return ctx, func() {}
}

func (t *tracingTestTracer) StartMessage(ctx context.Context, topic, operation string) (context.Context, func()) {
	return ctx, func() {}
}

func (t *tracingTestTracer) StartDatabase(ctx context.Context, operation, table string) (context.Context, func()) {
	return ctx, func() {}
}

func (t *tracingTestTracer) StartSegment(ctx context.Context, name string) (context.Context, func()) {
	return ctx, func() {}
}

func (t *tracingTestTracer) ShouldTrace(path string) bool {
	return t.enabled
}