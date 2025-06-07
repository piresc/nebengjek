package logger

import (
	"context"
	"log/slog"
	"time"
)

// Field type alias for better abstraction - now using slog.Attr
type Field = slog.Attr

// Field construction functions - abstracts slog implementation
// This allows using logger field functions instead of importing slog directly
// Making it easier to switch logging frameworks without changing client code

// String constructs a field that carries a string value
func String(key, val string) Field {
	return slog.String(key, val)
}

// Err constructs a field that carries an error
func Err(err error) Field {
	if err == nil {
		return slog.String("error", "<nil>")
	}
	return slog.String("error", err.Error())
}

// Int constructs a field that carries an int value
func Int(key string, val int) Field {
	return slog.Int(key, val)
}

// Int64 constructs a field that carries an int64 value
func Int64(key string, val int64) Field {
	return slog.Int64(key, val)
}

// Uint32 constructs a field that carries a uint32 value
func Uint32(key string, val uint32) Field {
	return slog.Uint64(key, uint64(val))
}

// Float64 constructs a field that carries a float64 value
func Float64(key string, val float64) Field {
	return slog.Float64(key, val)
}

// Bool constructs a field that carries a boolean value
func Bool(key string, val bool) Field {
	return slog.Bool(key, val)
}

// Any constructs a field that carries an arbitrary value
func Any(key string, val interface{}) Field {
	return slog.Any(key, val)
}

// Duration constructs a field that carries a time.Duration value
func Duration(key string, val time.Duration) Field {
	return slog.Duration(key, val)
}

// ErrorField constructs a field that carries an error (alias for Err for backward compatibility)
func ErrorField(err error) Field {
	return Err(err)
}

// Strings constructs a field that carries a slice of strings
func Strings(key string, val []string) Field {
	return slog.Any(key, val)
}

// Global logger instance for compatibility
var globalLogger *slog.Logger

// SetGlobalLogger sets the global logger instance
func SetGlobalLogger(logger *slog.Logger) {
	globalLogger = logger
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *slog.Logger {
	if globalLogger == nil {
		// Return default logger if none is set
		globalLogger = slog.Default()
	}
	return globalLogger
}

// Context-aware logging functions for backward compatibility
func InfoCtx(ctx context.Context, msg string, fields ...Field) {
	logger := GetGlobalLogger()
	args := make([]any, len(fields))
	for i, field := range fields {
		args[i] = field
	}
	logger.InfoContext(ctx, msg, args...)
}

func ErrorCtx(ctx context.Context, msg string, fields ...Field) {
	logger := GetGlobalLogger()
	args := make([]any, len(fields))
	for i, field := range fields {
		args[i] = field
	}
	logger.ErrorContext(ctx, msg, args...)
}

func WarnCtx(ctx context.Context, msg string, fields ...Field) {
	logger := GetGlobalLogger()
	args := make([]any, len(fields))
	for i, field := range fields {
		args[i] = field
	}
	logger.WarnContext(ctx, msg, args...)
}

func DebugCtx(ctx context.Context, msg string, fields ...Field) {
	logger := GetGlobalLogger()
	args := make([]any, len(fields))
	for i, field := range fields {
		args[i] = field
	}
	logger.DebugContext(ctx, msg, args...)
}

// Non-context logging functions for backward compatibility
func Info(msg string, fields ...Field) {
	InfoCtx(context.Background(), msg, fields...)
}

func Error(msg string, fields ...Field) {
	ErrorCtx(context.Background(), msg, fields...)
}

func Warn(msg string, fields ...Field) {
	WarnCtx(context.Background(), msg, fields...)
}

func Debug(msg string, fields ...Field) {
	DebugCtx(context.Background(), msg, fields...)
}
