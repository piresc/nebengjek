package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrslog"
	"github.com/newrelic/go-agent/v3/newrelic"
)

// SlogConfig holds configuration for the slog logger
type SlogConfig struct {
	Level       slog.Level
	ServiceName string
	NewRelic    *newrelic.Application
	Format      string // "json" or "text"
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
		// First wrap with nrslog for context enhancement
		handler = nrslog.WrapHandler(config.NewRelic, handler)

		// Then wrap with our custom forwarder for actual log forwarding
		handler = &NewRelicLogForwarder{
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

// NewRelicLogForwarder forwards logs to New Relic Application Logs
type NewRelicLogForwarder struct {
	handler slog.Handler
	app     *newrelic.Application
}

// Handle implements slog.Handler interface and forwards logs to New Relic
func (h *NewRelicLogForwarder) Handle(ctx context.Context, record slog.Record) error {
	// Forward to New Relic Application Logs for ERROR level and above
	if record.Level >= slog.LevelError && h.app != nil {
		// Create log data for New Relic
		logData := map[string]interface{}{
			"message":   record.Message,
			"level":     record.Level.String(),
			"timestamp": record.Time.UnixMilli(),
		}

		// Add all log attributes
		record.Attrs(func(attr slog.Attr) bool {
			logData[attr.Key] = attr.Value.Any()
			return true
		})

		// Send to New Relic Application Logs
		h.app.RecordLog(newrelic.LogData{
			Message:    record.Message,
			Severity:   record.Level.String(),
			Timestamp:  record.Time.UnixMilli(),
			Attributes: logData,
		})
	}

	// Pass to underlying handler (for console output)
	return h.handler.Handle(ctx, record)
}

// WithAttrs implements slog.Handler interface
func (h *NewRelicLogForwarder) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &NewRelicLogForwarder{
		handler: h.handler.WithAttrs(attrs),
		app:     h.app,
	}
}

// WithGroup implements slog.Handler interface
func (h *NewRelicLogForwarder) WithGroup(name string) slog.Handler {
	return &NewRelicLogForwarder{
		handler: h.handler.WithGroup(name),
		app:     h.app,
	}
}

// Enabled implements slog.Handler interface
func (h *NewRelicLogForwarder) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
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
