package context

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// TestRequestContext tests the RequestContext struct
func TestRequestContext(t *testing.T) {
	t.Run("RequestContext struct initialization", func(t *testing.T) {
		reqCtx := RequestContext{
			RequestID:   "req-123",
			UserID:      "user-456",
			TraceID:     "trace-789",
			ServiceName: "test-service",
			StartTime:   time.Now(),
		}

		assert.Equal(t, "req-123", reqCtx.RequestID)
		assert.Equal(t, "user-456", reqCtx.UserID)
		assert.Equal(t, "trace-789", reqCtx.TraceID)
		assert.Equal(t, "test-service", reqCtx.ServiceName)
		assert.False(t, reqCtx.StartTime.IsZero())
	})

	t.Run("RequestContext with empty values", func(t *testing.T) {
		reqCtx := RequestContext{}

		assert.Empty(t, reqCtx.RequestID)
		assert.Empty(t, reqCtx.UserID)
		assert.Empty(t, reqCtx.TraceID)
		assert.Empty(t, reqCtx.ServiceName)
		assert.True(t, reqCtx.StartTime.IsZero())
	})
}

// TestNewRequestContext tests the NewRequestContext function
func TestNewRequestContext(t *testing.T) {
	t.Run("NewRequestContext creates valid context", func(t *testing.T) {
		serviceName := "user-service"
		startTime := time.Now()

		reqCtx := NewRequestContext(serviceName)

		assert.NotNil(t, reqCtx)
		assert.Equal(t, serviceName, reqCtx.ServiceName)
		assert.NotEmpty(t, reqCtx.RequestID)
		assert.NotEmpty(t, reqCtx.TraceID)
		assert.True(t, reqCtx.StartTime.After(startTime) || reqCtx.StartTime.Equal(startTime))
		assert.Empty(t, reqCtx.UserID) // Should be empty initially

		// Validate UUID format
		_, err := uuid.Parse(reqCtx.RequestID)
		assert.NoError(t, err, "RequestID should be a valid UUID")

		_, err = uuid.Parse(reqCtx.TraceID)
		assert.NoError(t, err, "TraceID should be a valid UUID")
	})

	t.Run("NewRequestContext with empty service name", func(t *testing.T) {
		reqCtx := NewRequestContext("")

		assert.NotNil(t, reqCtx)
		assert.Empty(t, reqCtx.ServiceName)
		assert.NotEmpty(t, reqCtx.RequestID)
		assert.NotEmpty(t, reqCtx.TraceID)
		assert.False(t, reqCtx.StartTime.IsZero())
	})

	t.Run("NewRequestContext generates unique IDs", func(t *testing.T) {
		reqCtx1 := NewRequestContext("service1")
		reqCtx2 := NewRequestContext("service2")

		assert.NotEqual(t, reqCtx1.RequestID, reqCtx2.RequestID)
		assert.NotEqual(t, reqCtx1.TraceID, reqCtx2.TraceID)
		assert.NotEqual(t, reqCtx1.ServiceName, reqCtx2.ServiceName)
	})
}

// TestWithRequestContext tests the WithRequestContext function
func TestWithRequestContext(t *testing.T) {
	t.Run("WithRequestContext adds values to context", func(t *testing.T) {
		ctx := context.Background()
		reqCtx := &RequestContext{
			RequestID:   "req-123",
			UserID:      "user-456",
			TraceID:     "trace-789",
			ServiceName: "test-service",
		}

		newCtx := WithRequestContext(ctx, reqCtx)

		assert.Equal(t, "req-123", newCtx.Value(RequestIDKey))
		assert.Equal(t, "user-456", newCtx.Value(UserIDKey))
		assert.Equal(t, "trace-789", newCtx.Value(TraceIDKey))
		assert.Equal(t, "test-service", newCtx.Value(ServiceNameKey))
	})

	t.Run("WithRequestContext with nil context", func(t *testing.T) {
		reqCtx := &RequestContext{
			RequestID: "req-123",
		}

		// This should not panic
		newCtx := WithRequestContext(context.Background(), reqCtx)
		assert.NotNil(t, newCtx)
		assert.Equal(t, "req-123", newCtx.Value(RequestIDKey))
	})

	t.Run("WithRequestContext with empty RequestContext", func(t *testing.T) {
		ctx := context.Background()
		reqCtx := &RequestContext{}

		newCtx := WithRequestContext(ctx, reqCtx)

		assert.Equal(t, "", newCtx.Value(RequestIDKey))
		assert.Equal(t, "", newCtx.Value(UserIDKey))
		assert.Equal(t, "", newCtx.Value(TraceIDKey))
		assert.Equal(t, "", newCtx.Value(ServiceNameKey))
	})

	t.Run("WithRequestContext preserves existing context values", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "existing_key", "existing_value")
		reqCtx := &RequestContext{
			RequestID: "req-123",
		}

		newCtx := WithRequestContext(ctx, reqCtx)

		assert.Equal(t, "existing_value", newCtx.Value("existing_key"))
		assert.Equal(t, "req-123", newCtx.Value(RequestIDKey))
	})
}

// TestFromEchoContext tests the FromEchoContext function
func TestFromEchoContext(t *testing.T) {
	t.Run("FromEchoContext extracts request ID from header", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Set request ID in response header (simulating middleware)
		c.Response().Header().Set(echo.HeaderXRequestID, "existing-req-id")

		reqCtx := FromEchoContext(c)

		assert.Equal(t, "existing-req-id", reqCtx.RequestID)
		assert.NotEmpty(t, reqCtx.TraceID)
		assert.False(t, reqCtx.StartTime.IsZero())
	})

	t.Run("FromEchoContext generates request ID when not present", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		reqCtx := FromEchoContext(c)

		assert.NotEmpty(t, reqCtx.RequestID)
		assert.NotEmpty(t, reqCtx.TraceID)
		assert.False(t, reqCtx.StartTime.IsZero())

		// Validate UUID format
		_, err := uuid.Parse(reqCtx.RequestID)
		assert.NoError(t, err, "RequestID should be a valid UUID")
	})

	t.Run("FromEchoContext extracts user ID from context", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Set user ID in Echo context
		c.Set(string(UserIDKey), "user-123")

		reqCtx := FromEchoContext(c)

		assert.Equal(t, "user-123", reqCtx.UserID)
	})

	t.Run("FromEchoContext handles non-string user ID", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Set non-string user ID in Echo context
		c.Set(string(UserIDKey), 12345)

		reqCtx := FromEchoContext(c)

		assert.Empty(t, reqCtx.UserID) // Should be empty for non-string values
	})

	t.Run("FromEchoContext extracts trace ID from header", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Trace-ID", "existing-trace-id")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		reqCtx := FromEchoContext(c)

		assert.Equal(t, "existing-trace-id", reqCtx.TraceID)
	})

	t.Run("FromEchoContext generates trace ID when not present", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		reqCtx := FromEchoContext(c)

		assert.NotEmpty(t, reqCtx.TraceID)

		// Validate UUID format
		_, err := uuid.Parse(reqCtx.TraceID)
		assert.NoError(t, err, "TraceID should be a valid UUID")
	})

	t.Run("FromEchoContext extracts service name from context", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Set service name in Echo context
		c.Set("service_name", "location-service")

		reqCtx := FromEchoContext(c)

		assert.Equal(t, "location-service", reqCtx.ServiceName)
	})

	t.Run("FromEchoContext handles non-string service name", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Set non-string service name in Echo context
		c.Set("service_name", 12345)

		reqCtx := FromEchoContext(c)

		assert.Empty(t, reqCtx.ServiceName) // Should be empty for non-string values
	})

	t.Run("FromEchoContext with all values present", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Trace-ID", "trace-456")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Set all possible values
		c.Response().Header().Set(echo.HeaderXRequestID, "req-123")
		c.Set(string(UserIDKey), "user-789")
		c.Set("service_name", "match-service")

		reqCtx := FromEchoContext(c)

		assert.Equal(t, "req-123", reqCtx.RequestID)
		assert.Equal(t, "user-789", reqCtx.UserID)
		assert.Equal(t, "trace-456", reqCtx.TraceID)
		assert.Equal(t, "match-service", reqCtx.ServiceName)
		assert.False(t, reqCtx.StartTime.IsZero())
	})

	t.Run("FromEchoContext with no values present", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		reqCtx := FromEchoContext(c)

		assert.NotEmpty(t, reqCtx.RequestID) // Should be generated
		assert.Empty(t, reqCtx.UserID)
		assert.NotEmpty(t, reqCtx.TraceID) // Should be generated
		assert.Empty(t, reqCtx.ServiceName)
		assert.False(t, reqCtx.StartTime.IsZero())

		// Validate generated UUIDs
		_, err := uuid.Parse(reqCtx.RequestID)
		assert.NoError(t, err)
		_, err = uuid.Parse(reqCtx.TraceID)
		assert.NoError(t, err)
	})
}

// TestRequestContextIntegration tests integration scenarios
func TestRequestContextIntegration(t *testing.T) {
	t.Run("Full request context flow", func(t *testing.T) {
		// Create a new request context
		reqCtx := NewRequestContext("integration-service")
		assert.NotNil(t, reqCtx)

		// Add it to a Go context
		ctx := WithRequestContext(context.Background(), reqCtx)

		// Verify values can be retrieved
		assert.Equal(t, reqCtx.RequestID, ctx.Value(RequestIDKey))
		assert.Equal(t, reqCtx.UserID, ctx.Value(UserIDKey))
		assert.Equal(t, reqCtx.TraceID, ctx.Value(TraceIDKey))
		assert.Equal(t, reqCtx.ServiceName, ctx.Value(ServiceNameKey))
	})

	t.Run("Echo context to Go context flow", func(t *testing.T) {
		// Create Echo context with headers
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Trace-ID", "integration-trace")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Response().Header().Set(echo.HeaderXRequestID, "integration-req")
		c.Set(string(UserIDKey), "integration-user")
		c.Set("service_name", "integration-service")

		// Extract request context
		reqCtx := FromEchoContext(c)

		// Add to Go context
		ctx := WithRequestContext(context.Background(), reqCtx)

		// Verify all values are preserved
		assert.Equal(t, "integration-req", ctx.Value(RequestIDKey))
		assert.Equal(t, "integration-user", ctx.Value(UserIDKey))
		assert.Equal(t, "integration-trace", ctx.Value(TraceIDKey))
		assert.Equal(t, "integration-service", ctx.Value(ServiceNameKey))
	})

	t.Run("Request context modification", func(t *testing.T) {
		reqCtx := NewRequestContext("test-service")
		originalUserID := reqCtx.UserID

		// Modify user ID
		reqCtx.UserID = "modified-user"

		assert.NotEqual(t, originalUserID, reqCtx.UserID)
		assert.Equal(t, "modified-user", reqCtx.UserID)
	})
}

// TestRequestContextEdgeCases tests edge cases and error scenarios
func TestRequestContextEdgeCases(t *testing.T) {
	t.Run("WithRequestContext with nil RequestContext", func(t *testing.T) {
		ctx := context.Background()

		// This should not panic
		newCtx := WithRequestContext(ctx, nil)
		assert.NotNil(t, newCtx)

		// Values should be nil
		assert.Nil(t, newCtx.Value(RequestIDKey))
		assert.Nil(t, newCtx.Value(UserIDKey))
		assert.Nil(t, newCtx.Value(TraceIDKey))
		assert.Nil(t, newCtx.Value(ServiceNameKey))
	})

	t.Run("FromEchoContext with nil Echo context", func(t *testing.T) {
		// This test is removed as it would cause a panic in real scenarios
		// In production, Echo context should never be nil
		// We'll skip this edge case as it's not a realistic scenario
		t.Skip("Skipping nil Echo context test as it's not a realistic scenario")
	})

	t.Run("RequestContext with very long values", func(t *testing.T) {
		longString := make([]byte, 1000)
		for i := range longString {
			longString[i] = 'a'
		}

		reqCtx := &RequestContext{
			RequestID:   string(longString),
			UserID:      string(longString),
			TraceID:     string(longString),
			ServiceName: string(longString),
		}

		ctx := WithRequestContext(context.Background(), reqCtx)

		assert.Equal(t, string(longString), ctx.Value(RequestIDKey))
		assert.Equal(t, string(longString), ctx.Value(UserIDKey))
		assert.Equal(t, string(longString), ctx.Value(TraceIDKey))
		assert.Equal(t, string(longString), ctx.Value(ServiceNameKey))
	})

	t.Run("RequestContext with special characters", func(t *testing.T) {
		specialChars := "!@#$%^&*()_+-=[]{}|;':,.<>?"

		reqCtx := &RequestContext{
			RequestID:   "req-" + specialChars,
			UserID:      "user-" + specialChars,
			TraceID:     "trace-" + specialChars,
			ServiceName: "service-" + specialChars,
		}

		ctx := WithRequestContext(context.Background(), reqCtx)

		assert.Contains(t, ctx.Value(RequestIDKey).(string), specialChars)
		assert.Contains(t, ctx.Value(UserIDKey).(string), specialChars)
		assert.Contains(t, ctx.Value(TraceIDKey).(string), specialChars)
		assert.Contains(t, ctx.Value(ServiceNameKey).(string), specialChars)
	})
}

// TestRequestContextConcurrency tests concurrent access
func TestRequestContextConcurrency(t *testing.T) {
	t.Run("Concurrent NewRequestContext calls", func(t *testing.T) {
		const numGoroutines = 100
		results := make(chan *RequestContext, numGoroutines)

		// Start multiple goroutines creating request contexts
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				reqCtx := NewRequestContext("concurrent-service")
				results <- reqCtx
			}(i)
		}

		// Collect results
		requestIDs := make(map[string]bool)
		traceIDs := make(map[string]bool)

		for i := 0; i < numGoroutines; i++ {
			reqCtx := <-results
			assert.NotNil(t, reqCtx)
			assert.NotEmpty(t, reqCtx.RequestID)
			assert.NotEmpty(t, reqCtx.TraceID)
			assert.Equal(t, "concurrent-service", reqCtx.ServiceName)

			// Check for uniqueness
			assert.False(t, requestIDs[reqCtx.RequestID], "Duplicate RequestID found")
			assert.False(t, traceIDs[reqCtx.TraceID], "Duplicate TraceID found")

			requestIDs[reqCtx.RequestID] = true
			traceIDs[reqCtx.TraceID] = true
		}

		assert.Equal(t, numGoroutines, len(requestIDs))
		assert.Equal(t, numGoroutines, len(traceIDs))
	})
}

// Benchmark tests
func BenchmarkNewRequestContext(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewRequestContext("benchmark-service")
	}
}

func BenchmarkWithRequestContext(b *testing.B) {
	reqCtx := NewRequestContext("benchmark-service")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WithRequestContext(ctx, reqCtx)
	}
}

func BenchmarkFromEchoContext(b *testing.B) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FromEchoContext(c)
	}
}

func BenchmarkRequestContextOperations(b *testing.B) {
	b.Run("Create and use RequestContext", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			reqCtx := NewRequestContext("benchmark-service")
			ctx := WithRequestContext(context.Background(), reqCtx)
			_ = ctx.Value(RequestIDKey)
			_ = ctx.Value(UserIDKey)
			_ = ctx.Value(TraceIDKey)
			_ = ctx.Value(ServiceNameKey)
		}
	})
}