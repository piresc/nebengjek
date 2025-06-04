package middleware

import (
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// LoggerMiddleware creates a middleware for request logging
func LoggerMiddleware(logger *logrus.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Start timer
			start := time.Now()
			path := c.Request().URL.Path
			raw := c.Request().URL.RawQuery

			// Process request
			err := next(c)

			// Stop timer
			end := time.Now()
			latency := end.Sub(start)

			// Get client IP
			clientIP := c.RealIP()

			// Get status code
			statusCode := c.Response().Status

			// Get request method
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

			// Get request ID from header
			requestID := c.Response().Header().Get("X-Request-ID")

			// Log with appropriate level based on status code
			entry := logger.WithFields(logrus.Fields{
				"status":     statusCode,
				"latency":    latency.String(),
				"client_ip":  clientIP,
				"method":     method,
				"path":       path,
				"user_id":    userIDStr,
				"request_id": requestID,
			})

			if statusCode >= 500 {
				entry.Error("Server error")
			} else if statusCode >= 400 {
				entry.Warn("Client error")
			} else {
				entry.Info("Request processed")
			}

			return err
		}
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get request ID from header or generate a new one
			requestID := c.Request().Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = fmt.Sprintf("%d%d", time.Now().UnixNano(), time.Now().Unix())
			}

			// Set request ID in response header
			c.Response().Header().Set("X-Request-ID", requestID)
			c.Set("request_id", requestID)

			return next(c)
		}
	}
}
