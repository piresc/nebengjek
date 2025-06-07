package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// SlogConfig holds configuration for the slog logger
type SlogConfig struct {
	Level       slog.Level
	ServiceName string
	NewRelic    *newrelic.Application
	Format      string // "json" or "text"
}

// NewRelicHandler wraps slog.Handler to integrate with New Relic
type NewRelicHandler struct {
	handler slog.Handler
	app     *newrelic.Application
}

// NewSlogLogger creates a new slog logger with New Relic integration
func NewSlogLogger(config SlogConfig) *slog.Logger {
	var handler slog.Handler

	// Create base handler based on format
	opts := &slog.HandlerOptions{
		Level:     config.Level,
		AddSource: true,
	}

	switch config.Format {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	// Wrap with New Relic handler if available
	if config.NewRelic != nil {
		handler = &NewRelicHandler{
			handler: handler,
			app:     config.NewRelic,
		}
	}

	// Add service name to all logs
	if config.ServiceName != "" {
		handler = handler.WithAttrs([]slog.Attr{
			slog.String("service", config.ServiceName),
		})
	}

	return slog.New(handler)
}

// Handle implements slog.Handler interface
func (h *NewRelicHandler) Handle(ctx context.Context, record slog.Record) error {
	// Send to New Relic if it's an error
	if record.Level >= slog.LevelError && h.app != nil {
		// Try to get transaction from context
		if txn := newrelic.FromContext(ctx); txn != nil {
			// Create error from log message
			err := &LogError{
				Message: record.Message,
				Level:   record.Level.String(),
			}
			txn.NoticeError(err)

			// Add log attributes to transaction
			record.Attrs(func(attr slog.Attr) bool {
				txn.AddAttribute("log."+attr.Key, attr.Value.Any())
				return true
			})
		}
	}

	// Pass to underlying handler
	return h.handler.Handle(ctx, record)
}

// WithAttrs implements slog.Handler interface
func (h *NewRelicHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &NewRelicHandler{
		handler: h.handler.WithAttrs(attrs),
		app:     h.app,
	}
}

// WithGroup implements slog.Handler interface
func (h *NewRelicHandler) WithGroup(name string) slog.Handler {
	return &NewRelicHandler{
		handler: h.handler.WithGroup(name),
		app:     h.app,
	}
}

// Enabled implements slog.Handler interface
func (h *NewRelicHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// LogError implements error interface for New Relic
type LogError struct {
	Message string
	Level   string
}

func (e *LogError) Error() string {
	return e.Message
}

// ContextLogger provides context-aware logging helpers
type ContextLogger struct {
	logger *slog.Logger
}

// NewContextLogger creates a new context-aware logger
func NewContextLogger(logger *slog.Logger) *ContextLogger {
	return &ContextLogger{logger: logger}
}

// WithContext adds context values to log attributes
func (cl *ContextLogger) WithContext(ctx context.Context) *slog.Logger {
	attrs := []slog.Attr{}

	// Add request ID if available
	if requestID := ctx.Value("request_id"); requestID != nil {
		attrs = append(attrs, slog.String("request_id", requestID.(string)))
	}

	// Add user ID if available
	if userID := ctx.Value("user_id"); userID != nil {
		attrs = append(attrs, slog.String("user_id", userID.(string)))
	}

	// Add service name if available
	if serviceName := ctx.Value("service_name"); serviceName != nil {
		attrs = append(attrs, slog.String("service_name", serviceName.(string)))
	}

	// Add trace ID if available
	if traceID := ctx.Value("trace_id"); traceID != nil {
		attrs = append(attrs, slog.String("trace_id", traceID.(string)))
	}

	if len(attrs) > 0 {
		// Convert []slog.Attr to []any
		args := make([]any, len(attrs))
		for i, attr := range attrs {
			args[i] = attr
		}
		return cl.logger.With(args...)
	}

	return cl.logger
}

// Info logs an info message with context
func (cl *ContextLogger) Info(ctx context.Context, msg string, args ...any) {
	cl.WithContext(ctx).Info(msg, args...)
}

// Error logs an error message with context
func (cl *ContextLogger) Error(ctx context.Context, msg string, args ...any) {
	cl.WithContext(ctx).Error(msg, args...)
}

// Warn logs a warning message with context
func (cl *ContextLogger) Warn(ctx context.Context, msg string, args ...any) {
	cl.WithContext(ctx).Warn(msg, args...)
}

// Debug logs a debug message with context
func (cl *ContextLogger) Debug(ctx context.Context, msg string, args ...any) {
	cl.WithContext(ctx).Debug(msg, args...)
}
