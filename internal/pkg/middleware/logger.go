package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// LoggerMiddleware creates a middleware for request logging
func LoggerMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Stop timer
		end := time.Now()
		latency := end.Sub(start)

		// Get client IP
		clientIP := c.ClientIP()

		// Get status code
		statusCode := c.Writer.Status()

		// Get request method
		method := c.Request.Method

		// Format URL
		if raw != "" {
			path = path + "?" + raw
		}

		// Get user ID if available
		userID, exists := c.Get("user_id")
		userIDStr := "anonymous"
		if exists {
			userIDStr = fmt.Sprintf("%v", userID)
		}

		// Determine log level based on status code
		entry := logger.WithFields(logrus.Fields{
			"status":     statusCode,
			"latency":    latency.String(),
			"client_ip":  clientIP,
			"method":     method,
			"path":       path,
			"user_id":    userIDStr,
			"request_id": c.Writer.Header().Get("X-Request-ID"),
		})

		if statusCode >= 500 {
			entry.Error("Server error")
		} else if statusCode >= 400 {
			entry.Warn("Client error")
		} else {
			entry.Info("Request processed")
		}
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get request ID from header or generate a new one
		requestID := c.Request.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("%d%d", time.Now().UnixNano(), time.Now().Unix())
		}

		// Set request ID in response header
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Set("request_id", requestID)

		c.Next()
	}
}

// NewLogger creates a new logrus logger with default configuration
func NewLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	return logger
}
