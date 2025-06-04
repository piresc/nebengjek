package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/sirupsen/logrus"
)

// NewRelicLogger wraps logrus with New Relic integration
type NewRelicLogger struct {
	*logrus.Logger
	nrApp *newrelic.Application
}

// NewRelicLogHook is a logrus hook that sends logs to New Relic Logs API
type NewRelicLogHook struct {
	licenseKey string
	endpoint   string
	client     *http.Client
}

// NewNewRelicLogHook creates a new New Relic log hook
func NewNewRelicLogHook(licenseKey, endpoint string) *NewRelicLogHook {
	return &NewRelicLogHook{
		licenseKey: licenseKey,
		endpoint:   endpoint,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Levels returns the levels this hook handles
func (hook *NewRelicLogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire sends the log entry to New Relic
func (hook *NewRelicLogHook) Fire(entry *logrus.Entry) error {
	if hook.licenseKey == "" || hook.endpoint == "" {
		return nil // Skip if not configured
	}

	// Prepare log data for New Relic
	logData := map[string]interface{}{
		"timestamp": entry.Time.UnixMilli(),
		"message":   entry.Message,
		"level":     entry.Level.String(),
		"service":   "nebengjek-users-app",
	}

	// Add all fields from the log entry
	for key, value := range entry.Data {
		logData[key] = value
	}

	// Wrap in New Relic logs format
	payload := []map[string]interface{}{
		{
			"logs": []map[string]interface{}{logData},
		},
	}

	// Send to New Relic asynchronously to avoid blocking
	go func() {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			fmt.Printf("[ERROR] Failed to marshal log data: %v\n", err)
			return
		}

		req, err := http.NewRequest("POST", hook.endpoint, bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("[ERROR] Failed to create request: %v\n", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Api-Key", hook.licenseKey)

		resp, err := hook.client.Do(req)
		if err != nil {
			fmt.Printf("[ERROR] Failed to send log to New Relic: %v\n", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 && resp.StatusCode != 202 {
			fmt.Printf("[WARNING] New Relic API returned status: %d\n", resp.StatusCode)
		} else {
			fmt.Printf("[DEBUG] Successfully sent log to New Relic, status: %d\n", resp.StatusCode)
		}
	}()

	return nil
}

// NewNewRelicLogger creates a logger with New Relic integration
func NewNewRelicLogger(nrApp *newrelic.Application) *NewRelicLogger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	return &NewRelicLogger{
		Logger: logger,
		nrApp:  nrApp,
	}
}

// NewNewRelicLoggerWithConfig creates a logger with New Relic integration and API forwarding
func NewNewRelicLoggerWithConfig(nrApp *newrelic.Application, licenseKey, endpoint string, enabled bool) *NewRelicLogger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	// Add New Relic log forwarding hook if enabled
	if enabled && licenseKey != "" && endpoint != "" {
		hook := NewNewRelicLogHook(licenseKey, endpoint)
		logger.AddHook(hook)
	}

	return &NewRelicLogger{
		Logger: logger,
		nrApp:  nrApp,
	}
}

// NewRelicLoggerMiddleware creates middleware with New Relic transaction tracking
func NewRelicLoggerMiddleware(nrLogger *NewRelicLogger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var txn *newrelic.Transaction

			// Start New Relic transaction if app is available
			if nrLogger.nrApp != nil {
				txn = nrLogger.nrApp.StartTransaction(c.Request().Method + " " + c.Path())
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

			// Log with structured fields
			entry := nrLogger.WithFields(logrus.Fields{
				"status":     statusCode,
				"latency":    latency.String(),
				"latency_ms": latency.Milliseconds(),
				"client_ip":  clientIP,
				"method":     method,
				"path":       path,
				"user_id":    userIDStr,
				"request_id": requestID,
			})

			// Add New Relic context to log entry if available
			if txn != nil {
				// Use New Relic's log-in-context functionality
				ctx := newrelic.NewContext(context.Background(), txn)
				entry = entry.WithContext(ctx)

				// Add New Relic linking metadata for log correlation
				if mdw := txn.GetLinkingMetadata(); mdw.TraceID != "" {
					entry = entry.WithField("trace.id", mdw.TraceID)
					entry = entry.WithField("span.id", mdw.SpanID)
				}
			}

			// Log with appropriate level
			if statusCode >= 500 {
				if err != nil {
					entry.WithError(err).Error("Server error")
				} else {
					entry.Error("Server error")
				}
			} else if statusCode >= 400 {
				entry.Warn("Client error")
			} else {
				entry.Info("Request processed")
			}

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

// LogWithNewRelic logs a message with New Relic context
func LogWithNewRelic(logger *NewRelicLogger, txn *newrelic.Transaction, level logrus.Level, message string, fields logrus.Fields) {
	entry := logger.WithFields(fields)
	if txn != nil {
		entry = entry.WithContext(newrelic.NewContext(context.Background(), txn))
	}
	entry.Log(level, message)
}
