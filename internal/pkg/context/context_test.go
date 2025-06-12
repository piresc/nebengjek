package context

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestContextKey_String(t *testing.T) {
	// Test that ContextKey can be used as string
	key := ContextKey("test_key")
	assert.Equal(t, "test_key", string(key))
}

func TestConstants(t *testing.T) {
	// Test that all constants are properly defined
	assert.Equal(t, "request_id", string(RequestIDKey))
	assert.Equal(t, "user_id", string(UserIDKey))
	assert.Equal(t, "trace_id", string(TraceIDKey))
	assert.Equal(t, "service_name", string(ServiceNameKey))
}

func TestWithRequestID(t *testing.T) {
	tests := []struct {
		name      string
		requestID string
		expected  func(string) bool
	}{
		{
			name:      "Valid request ID",
			requestID: "req-123-456",
			expected: func(result string) bool {
				return result == "req-123-456"
			},
		},
		{
			name:      "Empty request ID - should generate UUID",
			requestID: "",
			expected: func(result string) bool {
				// Should be a valid UUID
				_, err := uuid.Parse(result)
				return err == nil && result != ""
			},
		},
		{
			name:      "UUID format request ID",
			requestID: uuid.New().String(),
			expected: func(result string) bool {
				_, err := uuid.Parse(result)
				return err == nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			newCtx := WithRequestID(ctx, tt.requestID)

			// Verify context is different
			assert.NotEqual(t, ctx, newCtx)

			// Get the request ID from context
			result := GetRequestID(newCtx)

			// Verify the result meets expectations
			assert.True(t, tt.expected(result), "Request ID validation failed: %s", result)
		})
	}
}

func TestGetRequestID(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() context.Context
		expected string
	}{
		{
			name: "Context with request ID",
			setup: func() context.Context {
				return WithRequestID(context.Background(), "test-request-id")
			},
			expected: "test-request-id",
		},
		{
			name: "Context without request ID",
			setup: func() context.Context {
				return context.Background()
			},
			expected: "",
		},
		{
			name: "Context with wrong type value",
			setup: func() context.Context {
				return context.WithValue(context.Background(), RequestIDKey, 123)
			},
			expected: "",
		},
		{
			name: "Context with nil value",
			setup: func() context.Context {
				return context.WithValue(context.Background(), RequestIDKey, nil)
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup()
			result := GetRequestID(ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWithUserID(t *testing.T) {
	userIDs := []string{
		"user-123",
		uuid.New().String(),
		"", // Empty user ID should be allowed
		"admin@example.com",
	}

	for _, userID := range userIDs {
		t.Run("UserID: "+userID, func(t *testing.T) {
			ctx := context.Background()
			newCtx := WithUserID(ctx, userID)

			assert.NotEqual(t, ctx, newCtx)
			result := GetUserID(newCtx)
			assert.Equal(t, userID, result)
		})
	}
}

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() context.Context
		expected string
	}{
		{
			name: "Context with user ID",
			setup: func() context.Context {
				return WithUserID(context.Background(), "user-456")
			},
			expected: "user-456",
		},
		{
			name: "Context without user ID",
			setup: func() context.Context {
				return context.Background()
			},
			expected: "",
		},
		{
			name: "Context with wrong type value",
			setup: func() context.Context {
				return context.WithValue(context.Background(), UserIDKey, 456)
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup()
			result := GetUserID(ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWithTraceID(t *testing.T) {
	traceIDs := []string{
		"trace-789",
		uuid.New().String(),
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", // OpenTelemetry format
		"",
	}

	for _, traceID := range traceIDs {
		t.Run("TraceID: "+traceID, func(t *testing.T) {
			ctx := context.Background()
			newCtx := WithTraceID(ctx, traceID)

			assert.NotEqual(t, ctx, newCtx)
			result := GetTraceID(newCtx)
			assert.Equal(t, traceID, result)
		})
	}
}

func TestGetTraceID(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() context.Context
		expected string
	}{
		{
			name: "Context with trace ID",
			setup: func() context.Context {
				return WithTraceID(context.Background(), "trace-abc123")
			},
			expected: "trace-abc123",
		},
		{
			name: "Context without trace ID",
			setup: func() context.Context {
				return context.Background()
			},
			expected: "",
		},
		{
			name: "Context with wrong type value",
			setup: func() context.Context {
				return context.WithValue(context.Background(), TraceIDKey, []byte("trace"))
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup()
			result := GetTraceID(ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWithServiceName(t *testing.T) {
	serviceNames := []string{
		"user-service",
		"location-service",
		"match-service",
		"rides-service",
		"",
	}

	for _, serviceName := range serviceNames {
		t.Run("ServiceName: "+serviceName, func(t *testing.T) {
			ctx := context.Background()
			newCtx := WithServiceName(ctx, serviceName)

			assert.NotEqual(t, ctx, newCtx)
			result := GetServiceName(newCtx)
			assert.Equal(t, serviceName, result)
		})
	}
}

func TestGetServiceName(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() context.Context
		expected string
	}{
		{
			name: "Context with service name",
			setup: func() context.Context {
				return WithServiceName(context.Background(), "test-service")
			},
			expected: "test-service",
		},
		{
			name: "Context without service name",
			setup: func() context.Context {
				return context.Background()
			},
			expected: "",
		},
		{
			name: "Context with wrong type value",
			setup: func() context.Context {
				return context.WithValue(context.Background(), ServiceNameKey, 123)
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup()
			result := GetServiceName(ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWithTimeout(t *testing.T) {
	timeouts := []time.Duration{
		1 * time.Second,
		5 * time.Second,
		30 * time.Second,
		1 * time.Minute,
	}

	for _, timeout := range timeouts {
		t.Run("Timeout: "+timeout.String(), func(t *testing.T) {
			ctx := context.Background()
			newCtx, cancel := WithTimeout(ctx, timeout)
			defer cancel()

			assert.NotEqual(t, ctx, newCtx)
			assert.NotNil(t, cancel)

			// Check that context has deadline
			deadline, ok := newCtx.Deadline()
			assert.True(t, ok)
			assert.True(t, deadline.After(time.Now()))
			assert.True(t, deadline.Before(time.Now().Add(timeout+time.Second)))
		})
	}
}

func TestWithDeadline(t *testing.T) {
	deadlines := []time.Time{
		time.Now().Add(1 * time.Second),
		time.Now().Add(5 * time.Minute),
		time.Now().Add(1 * time.Hour),
	}

	for i, deadline := range deadlines {
		t.Run(fmt.Sprintf("Deadline_%d", i), func(t *testing.T) {
			ctx := context.Background()
			newCtx, cancel := WithDeadline(ctx, deadline)
			defer cancel()

			assert.NotEqual(t, ctx, newCtx)
			assert.NotNil(t, cancel)

			// Check that context has the correct deadline
			ctxDeadline, ok := newCtx.Deadline()
			assert.True(t, ok)
			assert.Equal(t, deadline.Unix(), ctxDeadline.Unix())
		})
	}
}

func TestWithCancel(t *testing.T) {
	ctx := context.Background()
	newCtx, cancel := WithCancel(ctx)

	assert.NotEqual(t, ctx, newCtx)
	assert.NotNil(t, cancel)

	// Context should not be done initially
	select {
	case <-newCtx.Done():
		t.Fatal("Context should not be done initially")
	default:
		// Expected
	}

	// Cancel the context
	cancel()

	// Context should be done after cancellation
	select {
	case <-newCtx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should be done after cancellation")
	}

	// Check error
	assert.Equal(t, context.Canceled, newCtx.Err())
}

func TestChainedContextOperations(t *testing.T) {
	// Test chaining multiple context operations
	ctx := context.Background()

	// Add request ID
	ctx = WithRequestID(ctx, "req-123")
	assert.Equal(t, "req-123", GetRequestID(ctx))

	// Add user ID
	ctx = WithUserID(ctx, "user-456")
	assert.Equal(t, "req-123", GetRequestID(ctx)) // Should still be there
	assert.Equal(t, "user-456", GetUserID(ctx))

	// Add trace ID
	ctx = WithTraceID(ctx, "trace-789")
	assert.Equal(t, "req-123", GetRequestID(ctx))
	assert.Equal(t, "user-456", GetUserID(ctx))
	assert.Equal(t, "trace-789", GetTraceID(ctx))

	// Add service name
	ctx = WithServiceName(ctx, "test-service")
	assert.Equal(t, "req-123", GetRequestID(ctx))
	assert.Equal(t, "user-456", GetUserID(ctx))
	assert.Equal(t, "trace-789", GetTraceID(ctx))
	assert.Equal(t, "test-service", GetServiceName(ctx))

	// Add timeout
	ctxWithTimeout, cancel := WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// All values should still be accessible
	assert.Equal(t, "req-123", GetRequestID(ctxWithTimeout))
	assert.Equal(t, "user-456", GetUserID(ctxWithTimeout))
	assert.Equal(t, "trace-789", GetTraceID(ctxWithTimeout))
	assert.Equal(t, "test-service", GetServiceName(ctxWithTimeout))

	// Should have deadline
	deadline, ok := ctxWithTimeout.Deadline()
	assert.True(t, ok)
	assert.True(t, deadline.After(time.Now()))
}

func TestContextOverwrite(t *testing.T) {
	// Test that values can be overwritten
	ctx := context.Background()

	// Set initial values
	ctx = WithRequestID(ctx, "req-1")
	ctx = WithUserID(ctx, "user-1")
	assert.Equal(t, "req-1", GetRequestID(ctx))
	assert.Equal(t, "user-1", GetUserID(ctx))

	// Overwrite values
	ctx = WithRequestID(ctx, "req-2")
	ctx = WithUserID(ctx, "user-2")
	assert.Equal(t, "req-2", GetRequestID(ctx))
	assert.Equal(t, "user-2", GetUserID(ctx))
}

func TestContextTimeout(t *testing.T) {
	// Test that timeout context actually times out
	ctx := context.Background()
	ctxWithTimeout, cancel := WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	// Wait for timeout
	select {
	case <-ctxWithTimeout.Done():
		// Expected - context should timeout
		assert.Equal(t, context.DeadlineExceeded, ctxWithTimeout.Err())
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should have timed out")
	}
}

func BenchmarkWithRequestID(b *testing.B) {
	ctx := context.Background()
	requestID := "benchmark-request-id"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WithRequestID(ctx, requestID)
	}
}

func BenchmarkGetRequestID(b *testing.B) {
	ctx := WithRequestID(context.Background(), "benchmark-request-id")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetRequestID(ctx)
	}
}

func BenchmarkChainedOperations(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		ctx = WithRequestID(ctx, "req-123")
		ctx = WithUserID(ctx, "user-456")
		ctx = WithTraceID(ctx, "trace-789")
		ctx = WithServiceName(ctx, "test-service")
		
		_ = GetRequestID(ctx)
		_ = GetUserID(ctx)
		_ = GetTraceID(ctx)
		_ = GetServiceName(ctx)
	}
}