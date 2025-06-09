package middleware

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/observability"
)

// Config holds configuration for the middleware
type Config struct {
	Logger      *slog.Logger
	Tracer      observability.Tracer
	APIKeys     map[string]string
	ServiceName string
}

// Middleware combines multiple middleware into a single, efficient handler
type Middleware struct {
	config Config
}

// NewMiddleware creates a new middleware instance
func NewMiddleware(config Config) *Middleware {
	return &Middleware{config: config}
}

// RegisterHealthEndpoints is a helper method to register health endpoints before applying middleware
func (m *Middleware) RegisterHealthEndpoints(e *echo.Echo, serviceName, version string, healthChecker func(ctx context.Context) (map[string]interface{}, error)) {
	healthGroup := e.Group("/health")

	// Basic health check (for load balancers)
	healthGroup.GET("", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":    "ok",
			"service":   serviceName,
			"timestamp": time.Now(),
		})
	})

	// Detailed health check with dependencies
	healthGroup.GET("/detailed", func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
		defer cancel()

		response, err := healthChecker(ctx)
		if err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
				"status":    "unhealthy",
				"service":   serviceName,
				"version":   version,
				"timestamp": time.Now(),
				"error":     err.Error(),
			})
		}

		response["service"] = serviceName
		response["version"] = version

		statusCode := http.StatusOK
		if status, ok := response["status"].(string); ok && status == "unhealthy" {
			statusCode = http.StatusServiceUnavailable
		}

		return c.JSON(statusCode, response)
	})

	// Readiness probe (for Kubernetes)
	healthGroup.GET("/ready", func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 3*time.Second)
		defer cancel()

		response, err := healthChecker(ctx)
		if err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
				"status":  "not ready",
				"service": serviceName,
				"error":   err.Error(),
			})
		}

		if status, ok := response["status"].(string); ok && status == "unhealthy" {
			return c.JSON(http.StatusServiceUnavailable, response)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":  "ready",
			"service": serviceName,
		})
	})

	// Liveness probe (for Kubernetes)
	healthGroup.GET("/live", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":  "alive",
			"service": serviceName,
		})
	})
}

// Handler returns the main middleware handler that combines all functionality
func (m *Middleware) Handler() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// 1. Generate Request ID
			requestID := c.Request().Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.New().String()
			}
			c.Response().Header().Set("X-Request-ID", requestID)

			// 2. Add to context for easy access
			ctx := context.WithValue(c.Request().Context(), "request_id", requestID)
			ctx = context.WithValue(ctx, "service_name", m.config.ServiceName)
			c.SetRequest(c.Request().WithContext(ctx))

			// 3. Setup APM transaction (if tracer is enabled)
			var txn observability.Transaction
			if m.config.Tracer != nil {
				txn = m.config.Tracer.StartTransaction(c.Request().URL.Path)
				defer txn.End()
				txn.SetWebRequest(c.Request())

				// Add transaction context - this ensures New Relic context is available for logging
				ctx = txn.GetContext()
				c.SetRequest(c.Request().WithContext(ctx))

				// Store transaction in Echo context for easy access
				c.Set("nr_txn", txn)
			}

			// 4. Wrap response writer to capture response body for error logging
			responseCapture := &responseBodyCapture{
				ResponseWriter: c.Response().Writer,
				body:           make([]byte, 0),
			}
			c.Response().Writer = responseCapture

			// 5. Panic recovery
			defer func() {
				if r := recover(); r != nil {
					m.handlePanic(c, r, requestID, txn)
				}
			}()

			// 6. Execute the actual handler
			err := next(c)

			// 7. Log the request
			duration := time.Since(start)
			m.logRequest(c, requestID, duration, err, responseCapture.body)

			// 8. Set APM response (if enabled)
			if txn != nil {
				txn.SetWebResponse(c.Response().Writer)
			}

			return err
		}
	}
}

// APIKeyHandler returns middleware for API key validation
func (m *Middleware) APIKeyHandler(allowedServices ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			apiKey := c.Request().Header.Get("X-API-Key")
			if apiKey == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "API key required")
			}

			// Validate API key against allowed services
			valid := false
			for _, service := range allowedServices {
				if expectedKey, exists := m.config.APIKeys[service]; exists && expectedKey == apiKey {
					valid = true
					c.Set("api_service", service)
					break
				}
			}

			if !valid {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid API key")
			}

			return next(c)
		}
	}
}

// handlePanic handles panic recovery with enhanced diagnostic logging
func (m *Middleware) handlePanic(c echo.Context, r interface{}, requestID string, txn observability.Transaction) {
	stack := debug.Stack()

	// Enhanced diagnostic logging for WebSocket hijack failures
	if m.config.Logger != nil {
		logFields := []slog.Attr{
			slog.Any("panic", r),
			slog.String("request_id", requestID),
			slog.String("method", c.Request().Method),
			slog.String("path", c.Request().URL.Path),
			slog.String("ip", c.RealIP()),
			slog.String("proto", c.Request().Proto),
			slog.String("user_agent", c.Request().UserAgent()),
		}

		// Add WebSocket-specific diagnostics if this is a WebSocket request
		if c.Request().URL.Path == "/ws" || c.Request().Header.Get("Upgrade") == "websocket" {
			logFields = append(logFields,
				slog.String("upgrade_header", c.Request().Header.Get("Upgrade")),
				slog.String("connection_header", c.Request().Header.Get("Connection")),
				slog.String("sec_websocket_key", c.Request().Header.Get("Sec-WebSocket-Key")),
				slog.String("sec_websocket_version", c.Request().Header.Get("Sec-WebSocket-Version")),
			)

			// Check if response writer supports hijacking
			if _, ok := c.Response().Writer.(http.Hijacker); ok {
				logFields = append(logFields, slog.Bool("response_writer_supports_hijack", true))
			} else {
				logFields = append(logFields,
					slog.Bool("response_writer_supports_hijack", false),
					slog.String("response_writer_type", fmt.Sprintf("%T", c.Response().Writer)),
				)
			}

			// Check if original response writer supports hijacking
			if responseCapture, ok := c.Response().Writer.(*responseBodyCapture); ok {
				if _, ok := responseCapture.ResponseWriter.(http.Hijacker); ok {
					logFields = append(logFields, slog.Bool("original_response_writer_supports_hijack", true))
				} else {
					logFields = append(logFields,
						slog.Bool("original_response_writer_supports_hijack", false),
						slog.String("original_response_writer_type", fmt.Sprintf("%T", responseCapture.ResponseWriter)),
					)
				}
			}
		}

		logFields = append(logFields, slog.String("stack", string(stack)))

		// Convert slog.Attr slice to individual arguments
		args := make([]any, len(logFields)*2)
		for i, attr := range logFields {
			args[i*2] = attr.Key
			args[i*2+1] = attr.Value
		}

		m.config.Logger.ErrorContext(c.Request().Context(), "Panic recovered", args...)
	}

	// Report to APM if enabled
	if txn != nil {
		txn.NoticeError(fmt.Errorf("panic: %v", r))
	}

	// Send simple error response
	if !c.Response().Committed {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":      "Internal Server Error",
			"message":    "An unexpected error occurred",
			"request_id": requestID,
		})
	}
}

// logRequest logs the HTTP request with essential information
func (m *Middleware) logRequest(c echo.Context, requestID string, duration time.Duration, err error, responseBody []byte) {
	if m.config.Logger == nil {
		return
	}

	status := c.Response().Status

	// Handle cases where we have a Go error
	if err != nil {
		m.config.Logger.ErrorContext(c.Request().Context(), "Request failed",
			slog.String("request_id", requestID),
			slog.String("method", c.Request().Method),
			slog.String("path", c.Request().URL.Path),
			slog.Int("status", status),
			slog.Duration("duration", duration),
			slog.String("ip", c.RealIP()),
			slog.Int64("bytes_out", c.Response().Size),
			slog.Any("error", err),
		)
		return
	}

	// Handle error status codes (4xx, 5xx) even when no Go error was returned
	if status >= 400 {
		// Try to extract error details from response body
		errorDetails := m.extractErrorFromResponse(responseBody, status)

		logMessage := "Request completed with error"

		// Use Error level for 5xx status codes
		if status >= 500 {
			logMessage = "Request failed with server error"
		}

		if errorDetails != "" {
			if status >= 500 {
				m.config.Logger.ErrorContext(c.Request().Context(), logMessage,
					slog.String("request_id", requestID),
					slog.String("method", c.Request().Method),
					slog.String("path", c.Request().URL.Path),
					slog.Int("status", status),
					slog.Duration("duration", duration),
					slog.String("ip", c.RealIP()),
					slog.Int64("bytes_out", c.Response().Size),
					slog.String("error_details", errorDetails),
				)
			} else {
				m.config.Logger.WarnContext(c.Request().Context(), logMessage,
					slog.String("request_id", requestID),
					slog.String("method", c.Request().Method),
					slog.String("path", c.Request().URL.Path),
					slog.Int("status", status),
					slog.Duration("duration", duration),
					slog.String("ip", c.RealIP()),
					slog.Int64("bytes_out", c.Response().Size),
					slog.String("error_details", errorDetails),
				)
			}
		} else {
			if status >= 500 {
				m.config.Logger.ErrorContext(c.Request().Context(), logMessage,
					slog.String("request_id", requestID),
					slog.String("method", c.Request().Method),
					slog.String("path", c.Request().URL.Path),
					slog.Int("status", status),
					slog.Duration("duration", duration),
					slog.String("ip", c.RealIP()),
					slog.Int64("bytes_out", c.Response().Size),
				)
			} else {
				m.config.Logger.WarnContext(c.Request().Context(), logMessage,
					slog.String("request_id", requestID),
					slog.String("method", c.Request().Method),
					slog.String("path", c.Request().URL.Path),
					slog.Int("status", status),
					slog.Duration("duration", duration),
					slog.String("ip", c.RealIP()),
					slog.Int64("bytes_out", c.Response().Size),
				)
			}
		}
	} else {
		// Log successful requests at debug level to reduce noise
		m.config.Logger.DebugContext(c.Request().Context(), "Request completed",
			slog.String("request_id", requestID),
			slog.String("method", c.Request().Method),
			slog.String("path", c.Request().URL.Path),
			slog.Int("status", status),
			slog.Duration("duration", duration),
			slog.String("ip", c.RealIP()),
			slog.Int64("bytes_out", c.Response().Size),
		)
	}
}

// responseBodyCapture wraps the response writer to capture the response body
type responseBodyCapture struct {
	http.ResponseWriter
	body []byte
}

func (r *responseBodyCapture) Write(b []byte) (int, error) {
	// Capture the response body (limit to first 1KB to avoid memory issues)
	if len(r.body) < 1024 {
		remaining := 1024 - len(r.body)
		if len(b) <= remaining {
			r.body = append(r.body, b...)
		} else {
			r.body = append(r.body, b[:remaining]...)
		}
	}
	return r.ResponseWriter.Write(b)
}

// Hijack implements http.Hijacker interface to support WebSocket connections
func (r *responseBodyCapture) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := r.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("response writer does not support hijacking")
}

// extractErrorFromResponse attempts to extract error details from the response body
func (m *Middleware) extractErrorFromResponse(responseBody []byte, status int) string {
	if len(responseBody) == 0 {
		switch {
		case status >= 500:
			return fmt.Sprintf("Internal server error (HTTP %d)", status)
		case status >= 400:
			return fmt.Sprintf("Client error (HTTP %d)", status)
		default:
			return ""
		}
	}

	// Try to parse as JSON to extract error message
	var errorResp map[string]interface{}
	if err := json.Unmarshal(responseBody, &errorResp); err != nil {
		// If not JSON, return the raw body (truncated if too long)
		bodyStr := string(responseBody)
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500] + "..."
		}
		return bodyStr
	}

	// Extract error message from common error response formats
	if errorMsg, exists := errorResp["error"]; exists {
		if errorStr, ok := errorMsg.(string); ok {
			return errorStr
		}
	}

	// Try alternative error field names
	if errorMsg, exists := errorResp["message"]; exists {
		if errorStr, ok := errorMsg.(string); ok {
			return errorStr
		}
	}

	// If we can't extract a specific error, return the JSON as string (truncated)
	jsonStr, _ := json.Marshal(errorResp)
	if len(jsonStr) > 500 {
		return string(jsonStr[:500]) + "..."
	}

	return string(jsonStr)
}
