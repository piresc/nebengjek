package requestcontext

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// ContextKey type for context keys to avoid collisions
type ContextKey string

const (
	// RequestIDKey is the context key for request ID
	RequestIDKey ContextKey = "request_id"
	// UserIDKey is the context key for user ID
	UserIDKey ContextKey = "user_id"
	// TraceIDKey is the context key for trace ID
	TraceIDKey ContextKey = "trace_id"
	// ServiceNameKey is the context key for service name
	ServiceNameKey ContextKey = "service_name"
)

// RequestContext holds request-specific information
type RequestContext struct {
	RequestID   string
	UserID      string
	TraceID     string
	ServiceName string
	StartTime   time.Time
}

// NewRequestContext creates a new request context
func NewRequestContext(serviceName string) *RequestContext {
	return &RequestContext{
		RequestID:   uuid.New().String(),
		TraceID:     uuid.New().String(),
		ServiceName: serviceName,
		StartTime:   time.Now(),
	}
}

// WithRequestContext adds request context to the given context
func WithRequestContext(ctx context.Context, reqCtx *RequestContext) context.Context {
	ctx = context.WithValue(ctx, RequestIDKey, reqCtx.RequestID)
	ctx = context.WithValue(ctx, UserIDKey, reqCtx.UserID)
	ctx = context.WithValue(ctx, TraceIDKey, reqCtx.TraceID)
	ctx = context.WithValue(ctx, ServiceNameKey, reqCtx.ServiceName)
	return ctx
}

// FromEchoContext extracts request context from Echo context
func FromEchoContext(c echo.Context) *RequestContext {
	reqCtx := &RequestContext{
		StartTime: time.Now(),
	}

	if requestID := c.Response().Header().Get(echo.HeaderXRequestID); requestID != "" {
		reqCtx.RequestID = requestID
	} else {
		reqCtx.RequestID = uuid.New().String()
	}

	if userID := c.Get("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			reqCtx.UserID = uid
		}
	}

	if traceID := c.Request().Header.Get("X-Trace-ID"); traceID != "" {
		reqCtx.TraceID = traceID
	} else {
		reqCtx.TraceID = uuid.New().String()
	}

	if serviceName := c.Get("service_name"); serviceName != nil {
		if sn, ok := serviceName.(string); ok {
			reqCtx.ServiceName = sn
		}
	}

	return reqCtx
}

// GetRequestID extracts request ID from context
func GetRequestID(ctx context.Context) string {
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok {
		return reqID
	}
	return ""
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

// GetTraceID extracts trace ID from context
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// GetServiceName extracts service name from context
func GetServiceName(ctx context.Context) string {
	if serviceName, ok := ctx.Value(ServiceNameKey).(string); ok {
		return serviceName
	}
	return ""
}

// WithTimeout creates a context with timeout and preserves request context
func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// WithCancel creates a cancelable context and preserves request context
func WithCancel(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx)
}
