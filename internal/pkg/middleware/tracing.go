package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/tracing"
)

// TracingMiddleware provides automatic HTTP request tracing
type TracingMiddleware struct {
	tracer tracing.Tracer
	config *tracing.Config
}

// NewTracingMiddleware creates a new tracing middleware instance
func NewTracingMiddleware(config *tracing.Config, tracer tracing.Tracer) *TracingMiddleware {
	return &TracingMiddleware{
		tracer: tracer,
		config: config,
	}
}

// Handler returns an Echo middleware function with automatic tracing
func (m *TracingMiddleware) Handler() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip tracing for excluded paths
			if m.shouldExclude(c.Path()) {
				return next(c)
			}

			// Auto-create HTTP transaction
			ctx, txn := m.tracer.StartHTTPRequest(c.Request())
			defer txn.End()

			// Set transaction name based on route
			route := c.Path()
			if route == "" {
				route = c.Request().URL.Path
			}
			txn.SetName(formatTransactionName(c.Request().Method, route))

			// Add standard attributes
			txn.AddAttribute("http.method", c.Request().Method)
			txn.AddAttribute("http.url", c.Request().URL.String())
			txn.AddAttribute("http.route", route)
			txn.AddAttribute("user_agent", c.Request().UserAgent())
			txn.AddAttribute("remote_addr", c.RealIP())

			// Add user attributes if available
			if userID := c.Request().Header.Get("X-User-ID"); userID != "" {
				txn.AddAttribute("user.id", userID)
			}
			if userRole := c.Request().Header.Get("X-User-Role"); userRole != "" {
				txn.AddAttribute("user.role", userRole)
			}

			// Add request ID if available
			if requestID := c.Request().Header.Get("X-Request-ID"); requestID != "" {
				txn.AddAttribute("request.id", requestID)
			}

			// Inject tracing context for downstream calls
			c.SetRequest(c.Request().WithContext(ctx))

			// Auto-inject response headers for distributed tracing
			m.tracer.InjectContext(ctx, c.Response().Header())

			// Handle errors automatically
			err := next(c)
			if err != nil {
				txn.AddError(err)
				txn.AddAttribute("http.status_code", "500")
				txn.AddAttribute("error.message", err.Error())
			} else {
				status := c.Response().Status
				if status == 0 {
					status = 200 // Default to 200 if not set
				}
				txn.AddAttribute("http.status_code", http.StatusText(status))
			}

			return err
		}
	}
}

// shouldExclude checks if a path should be excluded from tracing
func (m *TracingMiddleware) shouldExclude(path string) bool {
	if !m.config.Enabled {
		return true
	}

	// Apply sampling
	if m.config.SampleRate < 1.0 && !shouldSample(m.config.SampleRate) {
		return true
	}

	// Check exclusions
	for _, excluded := range m.config.ExcludePaths {
		if strings.Contains(path, excluded) {
			return true
		}
	}

	return false
}

// formatTransactionName formats the transaction name for tracing
func formatTransactionName(method, route string) string {
	// Clean up the route name
	if route == "" || route == "/" {
		return strings.ToUpper(method) + " /"
	}

	// Remove trailing slash for consistency
	route = strings.TrimSuffix(route, "/")
	if route == "" {
		route = "/"
	}

	return strings.ToUpper(method) + " " + route
}

// shouldSample implements simple random sampling
func shouldSample(rate float64) bool {
	if rate >= 1.0 {
		return true
	}
	// Use a simple hash-based sampling for consistency
	// In production, you might want to use a more sophisticated sampling strategy
	return true // Simplified for now
}