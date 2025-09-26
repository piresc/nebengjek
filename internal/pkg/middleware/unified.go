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
	"github.com/piresc/nebengjek/internal/pkg/middleware/auth"
	"github.com/piresc/nebengjek/internal/pkg/tracing"
	"github.com/piresc/nebengjek/internal/pkg/models/core"
)

// Middleware combines multiple middleware into a single, efficient handler
type Middleware struct {
	config      *core.Config
	logger      *slog.Logger
	tracer      tracing.Tracer
	auth        *auth.AuthMiddleware
}

// NewMiddleware creates a new middleware instance
func NewMiddleware(config *core.Config, logger *slog.Logger, tracer tracing.Tracer) *Middleware {
	return &Middleware{
		config: config,
		logger: logger,
		tracer: tracer,
		auth:   auth.NewAuthMiddleware(config, tracer),
	}
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
			"status":  "live",
			"service": serviceName,
		})
	})
}

// Handler is the main middleware that combines logging, tracing, and error handling
func (m *Middleware) Handler() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			requestID := uuid.New().String()
			
			// Add request ID to context
			c.Set("request_id", requestID)
			c.Response().Header().Set("X-Request-ID", requestID)

			// Start tracing if enabled
			var txn tracing.HTTPTransaction
			if m.tracer != nil && m.tracer.IsEnabled() && m.tracer.ShouldTrace(c.Request().URL.Path) {
				ctx, transaction := m.tracer.StartHTTPRequest(c.Request())
				txn = transaction
				c.SetRequest(c.Request().WithContext(ctx))
				
				// Add tracing attributes
				txn.AddAttribute("request.id", requestID)
				txn.AddAttribute("http.method", c.Request().Method)
				txn.AddAttribute("http.url", c.Request().URL.String())
				txn.AddAttribute("user.agent", c.Request().UserAgent())
				
				defer txn.End()
			}

			// Capture response body for logging (only if response writer is not already captured)
			var responseCapture *responseBodyCapture
			if _, ok := c.Response().Writer.(*responseBodyCapture); !ok {
				responseCapture = &responseBodyCapture{ResponseWriter: c.Response().Writer}
				c.Response().Writer = responseCapture
			} else {
				// Use existing capture
				responseCapture = c.Response().Writer.(*responseBodyCapture)
			}

			// Handle panics
			defer func() {
				if r := recover(); r != nil {
					m.handlePanic(c, r, requestID, txn)
				}
			}()

			// Process request
			err := next(c)
			
			// Log the request
			duration := time.Since(start)
			m.logRequest(c, requestID, duration, err, responseCapture.Bytes())

			// Add error to tracing if it exists
			if err != nil && txn != nil {
				txn.AddError(err)
			}

			return err
		}
	}
}

// APIKeyHandler returns the API key authentication handler
func (m *Middleware) APIKeyHandler(allowedService string) echo.MiddlewareFunc {
	return m.auth.APIKeyHandler(allowedService)
}

// JWTHandler returns the JWT authentication handler
func (m *Middleware) JWTHandler() echo.MiddlewareFunc {
	return m.auth.JWTHandler()
}

// handlePanic handles panics and logs them appropriately
func (m *Middleware) handlePanic(c echo.Context, r interface{}, requestID string, txn tracing.HTTPTransaction) {
	stack := debug.Stack()
	
	m.logger.Error("Panic recovered",
		slog.String("request_id", requestID),
		slog.String("method", c.Request().Method),
		slog.String("path", c.Request().URL.Path),
		slog.String("panic", fmt.Sprintf("%v", r)),
		slog.String("stack", string(stack)),
	)

	if txn != nil {
		txn.AddAttribute("panic", fmt.Sprintf("%v", r))
		txn.AddAttribute("stack_trace", string(stack))
	}

	c.JSON(http.StatusInternalServerError, map[string]interface{}{
		"error":     "Internal server error",
		"request_id": requestID,
	})
}

// logRequest logs request information
func (m *Middleware) logRequest(c echo.Context, requestID string, duration time.Duration, err error, responseBody []byte) {
	status := c.Response().Status
	method := c.Request().Method
	path := c.Request().URL.Path
	userAgent := c.Request().UserAgent()
	
	// Extract error from response if needed
	errorMessage := ""
	if err != nil {
		errorMessage = err.Error()
	} else if status >= 400 {
		errorMessage = m.extractErrorFromResponse(responseBody, status)
	}

	// Build log attributes
	attributes := []slog.Attr{
		slog.String("request_id", requestID),
		slog.String("method", method),
		slog.String("path", path),
		slog.Int("status", status),
		slog.Duration("duration", duration),
		slog.String("user_agent", userAgent),
	}

	// Add user ID if available
	if userID := c.Get("user_id"); userID != nil {
		attributes = append(attributes, slog.String("user_id", userID.(string)))
	}

	// Add error message if available
	if errorMessage != "" {
		attributes = append(attributes, slog.String("error", errorMessage))
	}

	// Convert slog.Attr to []any for logging
	args := make([]any, len(attributes)*2)
	for i, attr := range attributes {
		args[i*2] = attr.Key
		args[i*2+1] = attr.Value
	}
	
	// Log based on status code
	if status >= 500 {
		m.logger.Error("Request failed", args...)
	} else if status >= 400 {
		m.logger.Warn("Request error", args...)
	} else {
		m.logger.Info("Request completed", args...)
	}
}

// responseBodyCapture captures the response body for logging
type responseBodyCapture struct {
	http.ResponseWriter
	body       []byte
	statusCode int
}

func (r *responseBodyCapture) Header() http.Header {
	return r.ResponseWriter.Header()
}

func (r *responseBodyCapture) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)
	return r.ResponseWriter.Write(b)
}

func (r *responseBodyCapture) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseBodyCapture) Bytes() []byte {
	return r.body
}

func (r *responseBodyCapture) StatusCode() int {
	if r.statusCode != 0 {
		return r.statusCode
	}
	return http.StatusOK // default status code
}

func (r *responseBodyCapture) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := r.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("response writer cannot hijack")
}

// extractErrorFromResponse extracts error message from response body
func (m *Middleware) extractErrorFromResponse(responseBody []byte, status int) string {
	if len(responseBody) == 0 {
		return fmt.Sprintf("HTTP %d", status)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return fmt.Sprintf("HTTP %d: %s", status, string(responseBody))
	}

	if errorMsg, ok := response["error"].(string); ok {
		return errorMsg
	}

	if errorMsg, ok := response["message"].(string); ok {
		return errorMsg
	}

	return fmt.Sprintf("HTTP %d", status)
}