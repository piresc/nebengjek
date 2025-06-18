package context

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
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
	if reqCtx == nil {
		return ctx
	}
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
