package logger

import (
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/newrelic/go-agent/v3/newrelic"
)

// ZapEchoMiddleware creates middleware for Echo framework using Zap logger
func ZapEchoMiddleware(logger *ZapLogger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get existing New Relic transaction from context (created by standard New Relic middleware)
			txn := newrelic.FromContext(c.Request().Context())

			// Start timer
			start := time.Now()
			path := c.Request().URL.Path
			raw := c.Request().URL.RawQuery

			// Process request
			err := next(c)

			// Calculate metrics
			latency := time.Since(start)
			statusCode := c.Response().Status
			clientIP := c.RealIP()
			method := c.Request().Method

			// Format URL
			if raw != "" {
				path = path + "?" + raw
			}

			// Get user ID if available
			userID := c.Get("user_id")
			userIDStr := "anonymous"
			if userID != nil {
				userIDStr = fmt.Sprintf("%v", userID)
			}

			// Get request ID
			requestID := c.Response().Header().Get("X-Request-ID")

			// Add custom attributes to New Relic transaction if available
			if txn != nil {
				txn.AddAttribute("user_id", userIDStr)
				txn.AddAttribute("request_id", requestID)
				txn.AddAttribute("response_time_ms", latency.Milliseconds())

				// Record error if present
				if err != nil {
					txn.NoticeError(err)
				}
			}

			// Log the HTTP request using our Zap logger
			logger.LogHTTPRequest(txn, method, path, clientIP, userIDStr, requestID, statusCode, latency, err)

			return err
		}
	}
}
