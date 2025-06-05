package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/requestcontext"
)

// RequestContextMiddleware creates a middleware that adds request context to Echo context
func RequestContextMiddleware(serviceName string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Create request context from Echo context
			reqCtx := requestcontext.FromEchoContext(c)
			reqCtx.ServiceName = serviceName

			// Add request context to Echo context for easy access in handlers
			c.Set("request_context", reqCtx)

			// Add context values to the request context
			ctx := requestcontext.WithRequestContext(c.Request().Context(), reqCtx)
			c.SetRequest(c.Request().WithContext(ctx))

			// Set headers for distributed tracing
			c.Response().Header().Set("X-Request-ID", reqCtx.RequestID)
			c.Response().Header().Set("X-Trace-ID", reqCtx.TraceID)

			return next(c)
		}
	}
}

// GetRequestContext extracts request context from Echo context
func GetRequestContext(c echo.Context) *requestcontext.RequestContext {
	if reqCtx, ok := c.Get("request_context").(*requestcontext.RequestContext); ok {
		return reqCtx
	}
	return nil
}
