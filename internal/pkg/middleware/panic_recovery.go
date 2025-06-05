package middleware

import (
	"fmt"
	"net/http"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/piresc/nebengjek/internal/pkg/logger"
)

// PanicRecoveryConfig holds configuration for panic recovery middleware
type PanicRecoveryConfig struct {
	StackSize       int
	DisableStackAll bool
	Logger          *logger.ZapLogger
}

// DefaultPanicRecoveryConfig returns default configuration for panic recovery
func DefaultPanicRecoveryConfig() PanicRecoveryConfig {
	return PanicRecoveryConfig{
		StackSize:       4 << 10, // 4 KB
		DisableStackAll: false,
		Logger:          nil,
	}
}

// PanicRecoveryMiddleware creates a middleware that recovers from panics
// and logs them with New Relic integration and stack traces
func PanicRecoveryMiddleware(config PanicRecoveryConfig) echo.MiddlewareFunc {
	if config.Logger == nil {
		panic("PanicRecoveryMiddleware requires a logger")
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					handlePanic(c, r, config)
				}
			}()

			return next(c)
		}
	}
}

// PanicRecoveryWithZapMiddleware creates panic recovery middleware with Zap logger
func PanicRecoveryWithZapMiddleware(zapLogger *logger.ZapLogger) echo.MiddlewareFunc {
	config := DefaultPanicRecoveryConfig()
	config.Logger = zapLogger
	return PanicRecoveryMiddleware(config)
}

// handlePanic handles the panic recovery, logging, and response
func handlePanic(c echo.Context, r interface{}, config PanicRecoveryConfig) {
	// Get stack trace
	stack := debug.Stack()
	stackTrace := string(stack)

	// Get request details
	method := c.Request().Method
	path := c.Request().URL.Path
	clientIP := c.RealIP()
	userAgent := c.Request().UserAgent()

	// Get user ID if available
	userID := "anonymous"
	if uid := c.Get("user_id"); uid != nil {
		userID = fmt.Sprintf("%v", uid)
	}

	// Get request ID
	requestID := c.Response().Header().Get("X-Request-ID")
	if requestID == "" {
		requestID = c.Request().Header.Get("X-Request-ID")
	}

	// Get New Relic transaction if available
	var txn *newrelic.Transaction
	if t := c.Get("nr_txn"); t != nil {
		if nrTxn, ok := t.(*newrelic.Transaction); ok {
			txn = nrTxn
		}
	}

	// Create error message
	panicMsg := fmt.Sprintf("Panic recovered: %v", r)

	// Get caller information
	var callerInfo string
	if pc, file, line, ok := runtime.Caller(4); ok {
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			callerInfo = fmt.Sprintf("%s:%d %s", file, line, fn.Name())
		} else {
			callerInfo = fmt.Sprintf("%s:%d", file, line)
		}
	}

	// Prepare structured log fields
	logFields := map[string]interface{}{
		"panic_value": r,
		"stack_trace": stackTrace,
		"caller":      callerInfo,
		"method":      method,
		"path":        path,
		"client_ip":   clientIP,
		"user_agent":  userAgent,
		"user_id":     userID,
		"request_id":  requestID,
		"panic_type":  fmt.Sprintf("%T", r),
		"service":     "nebengjek",
		"component":   "panic_recovery",
	}

	// Log with Zap logger
	zapLogger := config.Logger.WithFields(logFields)

	// Add New Relic context if available
	if txn != nil {
		nrLogger := config.Logger.WithNewRelicContext(txn)

		// Add all fields to the New Relic-aware logger
		zapFields := make([]logger.Field, 0, len(logFields))
		for key, value := range logFields {
			zapFields = append(zapFields, logger.Any(key, value))
		}
		nrLogger = nrLogger.With(zapFields...)

		// Record error in New Relic transaction
		txn.NoticeError(newrelic.Error{
			Message: panicMsg,
			Class:   "PanicError",
			Attributes: map[string]interface{}{
				"panic.value":     r,
				"panic.type":      fmt.Sprintf("%T", r),
				"panic.caller":    callerInfo,
				"stack_trace":     stackTrace,
				"http.method":     method,
				"http.path":       path,
				"http.client_ip":  clientIP,
				"http.user_agent": userAgent,
				"user_id":         userID,
				"request_id":      requestID,
			},
		})

		// Add custom attributes for better debugging
		txn.AddAttribute("panic.recovered", true)
		txn.AddAttribute("panic.value", fmt.Sprintf("%v", r))
		txn.AddAttribute("panic.type", fmt.Sprintf("%T", r))
		txn.AddAttribute("panic.caller", callerInfo)

		// Log with New Relic context
		nrLogger.Error("Panic recovered during request processing",
			logger.Any("panic_value", r),
			logger.String("panic_type", fmt.Sprintf("%T", r)),
			logger.String("stack_trace", stackTrace),
			logger.String("caller", callerInfo),
			logger.String("method", method),
			logger.String("path", path),
			logger.String("client_ip", clientIP),
			logger.String("user_agent", userAgent),
			logger.String("user_id", userID),
			logger.String("request_id", requestID),
		)
	} else {
		// Log without New Relic context
		zapLogger.Error("Panic recovered during request processing",
			logger.Any("panic_value", r),
			logger.String("panic_type", fmt.Sprintf("%T", r)),
			logger.String("stack_trace", stackTrace),
			logger.String("caller", callerInfo),
			logger.String("method", method),
			logger.String("path", path),
			logger.String("client_ip", clientIP),
			logger.String("user_agent", userAgent),
			logger.String("user_id", userID),
			logger.String("request_id", requestID),
		)
	}

	// Send internal server error response
	if !c.Response().Committed {
		err := c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":      "Internal Server Error",
			"message":    "An unexpected error occurred while processing your request",
			"request_id": requestID,
			"timestamp":  fmt.Sprintf("%d", c.Request().Context().Value("start_time")),
		})
		if err != nil {
			// If we can't send JSON, try plain text
			c.String(http.StatusInternalServerError, "Internal Server Error")
		}
	}
}

// Enhanced panic recovery with more detailed error context
func EnhancedPanicRecoveryMiddleware(zapLogger *logger.ZapLogger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					handleEnhancedPanic(c, r, zapLogger)
				}
			}()

			return next(c)
		}
	}
}

// handleEnhancedPanic provides more detailed panic handling with enhanced context
func handleEnhancedPanic(c echo.Context, r interface{}, zapLogger *logger.ZapLogger) {
	// Capture full stack trace
	stack := debug.Stack()
	stackTrace := string(stack)

	// Get detailed request information
	req := c.Request()
	resp := c.Response()

	// Extract comprehensive context
	context := map[string]interface{}{
		"panic": map[string]interface{}{
			"value":   r,
			"type":    fmt.Sprintf("%T", r),
			"message": fmt.Sprintf("%v", r),
		},
		"request": map[string]interface{}{
			"method":       req.Method,
			"url":          req.URL.String(),
			"path":         req.URL.Path,
			"query":        req.URL.RawQuery,
			"client_ip":    c.RealIP(),
			"user_agent":   req.UserAgent(),
			"content_type": req.Header.Get("Content-Type"),
			"headers":      extractSafeHeaders(req.Header),
		},
		"response": map[string]interface{}{
			"status":    resp.Status,
			"size":      resp.Size,
			"committed": resp.Committed,
		},
		"runtime": map[string]interface{}{
			"goroutines": runtime.NumGoroutine(),
			"memory":     getMemoryStats(),
		},
		"trace": map[string]interface{}{
			"stack_trace": stackTrace,
			"caller":      getCaller(4),
		},
	}

	// Get user context
	if userID := c.Get("user_id"); userID != nil {
		context["user_id"] = userID
	}

	// Get request ID
	if requestID := getRequestID(c); requestID != "" {
		context["request_id"] = requestID
	}

	// Get New Relic transaction and add error
	if txn := getNewRelicTransaction(c); txn != nil {
		// Record detailed error in New Relic
		txn.NoticeError(newrelic.Error{
			Message: fmt.Sprintf("Panic: %v", r),
			Class:   "PanicError",
			Attributes: map[string]interface{}{
				"panic.value":       r,
				"panic.type":        fmt.Sprintf("%T", r),
				"stack_trace":       stackTrace,
				"request.method":    req.Method,
				"request.path":      req.URL.Path,
				"request.client_ip": c.RealIP(),
				"goroutines":        runtime.NumGoroutine(),
			},
		})

		// Add context to logger - get New Relic aware logger
		nrLogger := zapLogger.WithNewRelicContext(txn)
		nrLogger.Error("Panic recovered with enhanced context",
			logger.Any("context", context),
			logger.String("stack_trace", stackTrace),
		)
	} else {
		// Log with full context without New Relic
		zapLogger.Error("Panic recovered with enhanced context",
			logger.Any("context", context),
			logger.String("stack_trace", stackTrace),
		)
	}

	// Send error response
	sendPanicResponse(c, getRequestID(c))
}

// Helper functions

func extractSafeHeaders(headers http.Header) map[string]string {
	safe := make(map[string]string)
	sensitiveHeaders := map[string]bool{
		"authorization": true,
		"cookie":        true,
		"set-cookie":    true,
		"x-api-key":     true,
	}

	for name, values := range headers {
		lowerName := strings.ToLower(name)
		if !sensitiveHeaders[lowerName] && len(values) > 0 {
			safe[name] = values[0]
		}
	}
	return safe
}

func getMemoryStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"alloc_mb":       bToMb(m.Alloc),
		"total_alloc_mb": bToMb(m.TotalAlloc),
		"sys_mb":         bToMb(m.Sys),
		"gc_cycles":      m.NumGC,
	}
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func getCaller(skip int) string {
	if pc, file, line, ok := runtime.Caller(skip); ok {
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			return fmt.Sprintf("%s:%d in %s", file, line, fn.Name())
		}
		return fmt.Sprintf("%s:%d", file, line)
	}
	return "unknown"
}

func getRequestID(c echo.Context) string {
	if requestID := c.Response().Header().Get("X-Request-ID"); requestID != "" {
		return requestID
	}
	if requestID := c.Request().Header.Get("X-Request-ID"); requestID != "" {
		return requestID
	}
	if requestID := c.Get("request_id"); requestID != nil {
		return fmt.Sprintf("%v", requestID)
	}
	return ""
}

func getNewRelicTransaction(c echo.Context) *newrelic.Transaction {
	if txn := c.Get("nr_txn"); txn != nil {
		if nrTxn, ok := txn.(*newrelic.Transaction); ok {
			return nrTxn
		}
	}
	return nil
}

func sendPanicResponse(c echo.Context, requestID string) {
	if !c.Response().Committed {
		response := map[string]interface{}{
			"error":   "Internal Server Error",
			"message": "An unexpected error occurred while processing your request",
		}

		if requestID != "" {
			response["request_id"] = requestID
		}

		if err := c.JSON(http.StatusInternalServerError, response); err != nil {
			// Fallback to plain text if JSON fails
			c.String(http.StatusInternalServerError, "Internal Server Error")
		}
	}
}
