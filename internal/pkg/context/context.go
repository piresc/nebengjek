package context

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ContextKey represents a key for context values
type ContextKey string

const (
	// RequestIDKey is the key for request ID in context
	RequestIDKey ContextKey = "request_id"
	// UserIDKey is the key for user ID in context
	UserIDKey ContextKey = "user_id"
	// TraceIDKey is the key for trace ID in context
	TraceIDKey ContextKey = "trace_id"
	// ServiceNameKey is the key for service name in context
	ServiceNameKey ContextKey = "service_name"
)

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	if requestID == "" {
		requestID = uuid.New().String()
	}
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// WithUserID adds a user ID to the context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// GetUserID retrieves the user ID from context
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

// WithTraceID adds a trace ID to the context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// GetTraceID retrieves the trace ID from context
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// WithTimeout creates a context with timeout for operations
func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// WithDeadline creates a context with deadline for operations
func WithDeadline(ctx context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(ctx, deadline)
}

// WithCancel creates a cancellable context
func WithCancel(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx)
}
