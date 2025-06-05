package logger

import (
	"context"
	"sync"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// globalLogger holds the singleton logger instance
	globalLogger *ZapLogger
	// once ensures the logger is initialized only once
	once sync.Once
	// mu protects access to the global logger
	mu sync.RWMutex
)

// SetGlobalLogger sets the global logger instance
// This should be called once during application startup
func SetGlobalLogger(logger *ZapLogger) {
	mu.Lock()
	defer mu.Unlock()
	globalLogger = logger
}

// GetGlobalLogger returns the global logger instance
// If no logger is set, it returns a default logger
func GetGlobalLogger() *ZapLogger {
	mu.RLock()
	defer mu.RUnlock()

	if globalLogger == nil {
		// Return a default logger if none is set (for safety)
		once.Do(func() {
			defaultLogger, _ := zap.NewProduction()
			globalLogger = &ZapLogger{
				Logger: defaultLogger,
				sugar:  defaultLogger.Sugar(),
			}
		})
	}

	return globalLogger
}

// Global logger convenience functions

// Info logs an info message using the global logger
func Info(msg string, fields ...Field) {
	GetGlobalLogger().Info(msg, fields...)
}

// ErrorLog logs an error message using the global logger
func ErrorLog(msg string, fields ...Field) {
	GetGlobalLogger().Error(msg, fields...)
}

// Warn logs a warning message using the global logger
func Warn(msg string, fields ...Field) {
	GetGlobalLogger().Warn(msg, fields...)
}

// Debug logs a debug message using the global logger
func Debug(msg string, fields ...Field) {
	GetGlobalLogger().Debug(msg, fields...)
}

// Error logs an error message using the global logger (alias for ErrorLog for backward compatibility)
func Error(msg string, fields ...Field) {
	GetGlobalLogger().Error(msg, fields...)
}

// Fatal logs a fatal message and exits using the global logger
func Fatal(msg string, fields ...Field) {
	GetGlobalLogger().Fatal(msg, fields...)
}

// WithFields returns a logger with additional fields using the global logger
func WithFields(fields map[string]interface{}) *zap.Logger {
	return GetGlobalLogger().WithFields(fields)
}

// WithError returns a logger with an error field using the global logger
func WithError(err error) *zap.Logger {
	return GetGlobalLogger().WithError(err)
}

// WithNewRelicContext adds New Relic context to the global logger
func WithNewRelicContext(txn *newrelic.Transaction) *zap.Logger {
	return GetGlobalLogger().WithNewRelicContext(txn)
}

// WithRequestContext adds request context fields to the global logger
func WithRequestContext(requestID, userID, method, path string) *zap.Logger {
	return GetGlobalLogger().WithRequestContext(requestID, userID, method, path)
}

// Sugar returns the sugared logger from the global logger
func Sugar() *zap.SugaredLogger {
	return GetGlobalLogger().Sugar()
}

// LogHTTPRequest logs HTTP request using the global logger
func LogHTTPRequest(txn *newrelic.Transaction, method, path, clientIP, userID, requestID string, statusCode int, latency time.Duration, err error) {
	GetGlobalLogger().LogHTTPRequest(txn, method, path, clientIP, userID, requestID, statusCode, latency, err)
}

// Context-aware logging

// InfoCtx logs an info message with context using the global logger
func InfoCtx(ctx context.Context, msg string, fields ...Field) {
	logger := GetGlobalLogger()
	if txn := newrelic.FromContext(ctx); txn != nil {
		logger = &ZapLogger{
			Logger: logger.WithNewRelicContext(txn),
			sugar:  logger.sugar,
			nrApp:  logger.nrApp,
		}
	}
	logger.Info(msg, fields...)
}

// ErrorCtx logs an error message with context using the global logger
func ErrorCtx(ctx context.Context, msg string, fields ...Field) {
	logger := GetGlobalLogger()
	if txn := newrelic.FromContext(ctx); txn != nil {
		logger = &ZapLogger{
			Logger: logger.WithNewRelicContext(txn),
			sugar:  logger.sugar,
			nrApp:  logger.nrApp,
		}
	}
	logger.Error(msg, fields...)
}

// WarnCtx logs a warning message with context using the global logger
func WarnCtx(ctx context.Context, msg string, fields ...Field) {
	logger := GetGlobalLogger()
	if txn := newrelic.FromContext(ctx); txn != nil {
		logger = &ZapLogger{
			Logger: logger.WithNewRelicContext(txn),
			sugar:  logger.sugar,
			nrApp:  logger.nrApp,
		}
	}
	logger.Warn(msg, fields...)
}

// DebugCtx logs a debug message with context using the global logger
func DebugCtx(ctx context.Context, msg string, fields ...Field) {
	logger := GetGlobalLogger()
	if txn := newrelic.FromContext(ctx); txn != nil {
		logger = &ZapLogger{
			Logger: logger.WithNewRelicContext(txn),
			sugar:  logger.sugar,
			nrApp:  logger.nrApp,
		}
	}
	logger.Debug(msg, fields...)
}

// LogWithContext logs a message at the specified level with context
func LogWithContext(ctx context.Context, level zapcore.Level, msg string, fields ...zap.Field) {
	switch level {
	case zapcore.DebugLevel:
		DebugCtx(ctx, msg, fields...)
	case zapcore.InfoLevel:
		InfoCtx(ctx, msg, fields...)
	case zapcore.WarnLevel:
		WarnCtx(ctx, msg, fields...)
	case zapcore.ErrorLevel:
		ErrorCtx(ctx, msg, fields...)
	case zapcore.FatalLevel:
		Fatal(msg, fields...)
	default:
		InfoCtx(ctx, msg, fields...)
	}
}
