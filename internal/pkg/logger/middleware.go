package logger

import (
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/sirupsen/logrus"
)

// EchoMiddleware creates middleware for Echo framework using our custom logger
func EchoMiddleware(logger *AppLogger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var txn *newrelic.Transaction

			// Start New Relic transaction if app is available
			if logger.nrApp != nil {
				txn = logger.nrApp.StartTransaction(c.Request().Method + " " + c.Path())
				defer txn.End()

				// Add transaction to context
				c.Set("nr_txn", txn)

				// Set web request/response for better tracking
				txn.SetWebRequestHTTP(c.Request())
				txn.SetWebResponse(c.Response().Writer)
			}

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

			// Add custom attributes to New Relic transaction
			if txn != nil {
				txn.AddAttribute("http.status_code", statusCode)
				txn.AddAttribute("http.method", method)
				txn.AddAttribute("http.url", c.Request().URL.String())
				txn.AddAttribute("response_time_ms", latency.Milliseconds())
				txn.AddAttribute("client_ip", clientIP)
				txn.AddAttribute("user_id", userIDStr)
				txn.AddAttribute("request_id", requestID)

				// Record error if present
				if err != nil {
					txn.NoticeError(err)
				}
			}

			// Log the HTTP request using our logger
			logger.LogHTTPRequest(txn, method, path, clientIP, userIDStr, requestID, statusCode, latency, err)

			return err
		}
	}
}

// GetTransactionFromContext retrieves New Relic transaction from Echo context
func GetTransactionFromContext(c echo.Context) *newrelic.Transaction {
	if txn := c.Get("nr_txn"); txn != nil {
		if nrTxn, ok := txn.(*newrelic.Transaction); ok {
			return nrTxn
		}
	}
	return nil
}

// LogWithContext logs a message with Echo context and New Relic transaction
func LogWithContext(logger *AppLogger, c echo.Context, level logrus.Level, message string, fields logrus.Fields) {
	txn := GetTransactionFromContext(c)

	entry := logger.WithFields(fields)
	if txn != nil {
		entry = logger.WithNewRelicContext(txn)
		if fields != nil {
			entry = entry.WithFields(fields)
		}
	}

	entry.Log(level, message)
}
