package logger

import (
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/newrelic/go-agent/v3/newrelic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapEchoMiddleware creates middleware for Echo framework using Zap logger
func ZapEchoMiddleware(logger *ZapLogger) echo.MiddlewareFunc {
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

			// Log the HTTP request using our Zap logger
			logger.LogHTTPRequest(txn, method, path, clientIP, userIDStr, requestID, statusCode, latency, err)

			return err
		}
	}
}

// GetZapTransactionFromContext retrieves New Relic transaction from Echo context
func GetZapTransactionFromContext(c echo.Context) *newrelic.Transaction {
	if txn := c.Get("nr_txn"); txn != nil {
		if nrTxn, ok := txn.(*newrelic.Transaction); ok {
			return nrTxn
		}
	}
	return nil
}

// ZapLogWithContext logs a message with Echo context and New Relic transaction
func ZapLogWithContext(logger *ZapLogger, c echo.Context, level zapcore.Level, message string, fields map[string]interface{}) {
	txn := GetZapTransactionFromContext(c)

	zapLogger := logger.WithFields(fields)
	if txn != nil {
		zapLogger = logger.WithNewRelicContext(txn)

		// Re-add fields after adding New Relic context
		zapFields := make([]zap.Field, 0, len(fields))
		for key, value := range fields {
			zapFields = append(zapFields, zap.Any(key, value))
		}
		zapLogger = zapLogger.With(zapFields...)
	}

	switch level {
	case zapcore.DebugLevel:
		zapLogger.Debug(message)
	case zapcore.InfoLevel:
		zapLogger.Info(message)
	case zapcore.WarnLevel:
		zapLogger.Warn(message)
	case zapcore.ErrorLevel:
		zapLogger.Error(message)
	case zapcore.FatalLevel:
		zapLogger.Fatal(message)
	default:
		zapLogger.Info(message)
	}
}
