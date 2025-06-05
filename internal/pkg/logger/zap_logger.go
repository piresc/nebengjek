package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapLogger is our custom Zap logger that supports multiple outputs and New Relic integration
type ZapLogger struct {
	*zap.Logger
	sugar    *zap.SugaredLogger
	nrApp    *newrelic.Application
	filePath string
	file     *os.File
}

// newRelicCore is a zapcore.Core that forwards logs to New Relic
type newRelicCore struct {
	encoder zapcore.Encoder
	level   zapcore.Level
	nrApp   *newrelic.Application
}

// Enabled returns true if the given level is enabled
func (c *newRelicCore) Enabled(level zapcore.Level) bool {
	return c.level.Enabled(level)
}

// With returns a new core with the given fields added
func (c *newRelicCore) With(fields []zapcore.Field) zapcore.Core {
	clone := *c
	return &clone
}

// Check determines whether the supplied entry should be written
func (c *newRelicCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}

// Write logs the entry to New Relic
func (c *newRelicCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	if c.nrApp == nil {
		return nil
	}

	// Convert Zap fields to map
	encoder := zapcore.NewMapObjectEncoder()
	for _, field := range fields {
		field.AddTo(encoder)
	}

	// Prepare log data for New Relic
	logData := newrelic.LogData{
		Timestamp:  entry.Time.UnixMilli(),
		Message:    entry.Message,
		Severity:   entry.Level.String(),
		Attributes: encoder.Fields,
	}

	// Add additional context
	if logData.Attributes == nil {
		logData.Attributes = make(map[string]any)
	}
	logData.Attributes["service"] = "nebengjek-users-app"
	logData.Attributes["caller"] = entry.Caller.TrimmedPath()

	// Add stack trace if available
	if entry.Stack != "" {
		logData.Attributes["stacktrace"] = entry.Stack
	}

	// Send to New Relic
	c.nrApp.RecordLog(logData)
	return nil
}

// Sync is a no-op for New Relic core
func (c *newRelicCore) Sync() error {
	return nil
}

// ZapConfig holds Zap logger configuration
type ZapConfig struct {
	Level      string `json:"level" mapstructure:"level"`
	FilePath   string `json:"file_path" mapstructure:"file_path"`
	MaxSize    int64  `json:"max_size" mapstructure:"max_size"`       // Max size in MB before rotation
	MaxAge     int    `json:"max_age" mapstructure:"max_age"`         // Max age in days
	MaxBackups int    `json:"max_backups" mapstructure:"max_backups"` // Max number of backup files
	Compress   bool   `json:"compress" mapstructure:"compress"`       // Compress rotated files
}

// NewZapLogger creates a new Zap application logger
func NewZapLogger(config ZapConfig, nrApp *newrelic.Application) (*ZapLogger, error) {
	// Parse log level
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(config.Level)); err != nil {
		level = zapcore.InfoLevel
	}

	// Create encoder config for structured JSON logging
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.RFC3339TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create JSON encoder
	encoder := zapcore.NewJSONEncoder(encoderConfig)

	// Prepare writers
	var cores []zapcore.Core

	// Console output (always enabled for development)
	cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level))

	zapLogger := &ZapLogger{
		nrApp:    nrApp,
		filePath: config.FilePath,
	}

	// File output if path is provided
	if config.FilePath != "" {
		if err := zapLogger.setupFileOutput(config.FilePath); err != nil {
			return nil, fmt.Errorf("failed to setup file output: %w", err)
		}
		cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(zapLogger.file), level))
	}

	// Add New Relic log forwarding if app is available
	if nrApp != nil {
		nrCore := &newRelicCore{
			encoder: encoder,
			level:   level,
			nrApp:   nrApp,
		}
		cores = append(cores, nrCore)
	}

	// Create the final logger with multiple cores
	core := zapcore.NewTee(cores...)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	zapLogger.Logger = logger
	zapLogger.sugar = logger.Sugar()

	return zapLogger, nil
}

// setupFileOutput configures file output for the logger
func (zl *ZapLogger) setupFileOutput(filePath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	zl.file = file
	return nil
}

// Close closes the log file and syncs the logger
func (zl *ZapLogger) Close() error {
	// Sync the logger to flush any buffered logs
	_ = zl.Logger.Sync()
	_ = zl.sugar.Sync()

	if zl.file != nil {
		return zl.file.Close()
	}
	return nil
}

// WithNewRelicContext adds New Relic context to log fields
func (zl *ZapLogger) WithNewRelicContext(txn *newrelic.Transaction) *zap.Logger {
	fields := []zap.Field{}

	if txn != nil {
		// Add trace correlation fields
		if mdw := txn.GetLinkingMetadata(); mdw.TraceID != "" {
			fields = append(fields,
				zap.String("trace.id", mdw.TraceID),
				zap.String("span.id", mdw.SpanID),
			)
		}
	}

	return zl.Logger.With(fields...)
}

// WithRequestContext adds request context fields
func (zl *ZapLogger) WithRequestContext(requestID, userID, method, path string) *zap.Logger {
	return zl.Logger.With(
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("method", method),
		zap.String("path", path),
		zap.String("service", "nebengjek-users-app"),
	)
}

// WithFields adds custom fields to log entry
func (zl *ZapLogger) WithFields(fields map[string]interface{}) *zap.Logger {
	zapFields := make([]zap.Field, 0, len(fields)+1)

	// Always add service name
	zapFields = append(zapFields, zap.String("service", "nebengjek-users-app"))

	// Convert map to zap fields
	for key, value := range fields {
		zapFields = append(zapFields, zap.Any(key, value))
	}

	return zl.Logger.With(zapFields...)
}

// LogHTTPRequest logs HTTP request with all relevant context
func (zl *ZapLogger) LogHTTPRequest(txn *newrelic.Transaction, method, path, clientIP, userID, requestID string, statusCode int, latency time.Duration, err error) {
	logger := zl.WithFields(map[string]interface{}{
		"status":     statusCode,
		"latency":    latency.String(),
		"latency_ms": latency.Milliseconds(),
		"client_ip":  clientIP,
		"method":     method,
		"path":       path,
		"user_id":    userID,
		"request_id": requestID,
	})

	// Add New Relic context if available
	if txn != nil {
		logger = zl.WithNewRelicContext(txn)
		logger = logger.With(
			zap.Int("status", statusCode),
			zap.String("latency", latency.String()),
			zap.Int64("latency_ms", latency.Milliseconds()),
			zap.String("client_ip", clientIP),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("user_id", userID),
			zap.String("request_id", requestID),
		)
	}

	// Log with appropriate level
	if statusCode >= 500 {
		if err != nil {
			logger.Error("Server error", zap.Error(err))
		} else {
			logger.Error("Server error")
		}
	} else if statusCode >= 400 {
		logger.Warn("Client error")
	} else {
		logger.Info("Request processed")
	}
}

// Sugar returns the sugared logger for easier use
func (zl *ZapLogger) Sugar() *zap.SugaredLogger {
	return zl.sugar
}

// GetFilePath returns the current log file path
func (zl *ZapLogger) GetFilePath() string {
	return zl.filePath
}

// InitZapLoggerFromConfig initializes Zap logger directly from config models
func InitZapLoggerFromConfig(configs *models.Config, nrApp *newrelic.Application) (*ZapLogger, error) {
	// Convert config models to Zap logger config
	zapConfig := ZapConfig{
		Level:      configs.Logger.Level,
		FilePath:   configs.Logger.FilePath,
		MaxSize:    configs.Logger.MaxSize,
		MaxAge:     configs.Logger.MaxAge,
		MaxBackups: configs.Logger.MaxBackups,
		Compress:   configs.Logger.Compress,
	}

	// Create logger with New Relic integration
	// New Relic log forwarding is handled by the agent itself when ConfigAppLogForwardingEnabled is set
	return NewZapLogger(zapConfig, nrApp)
}

// Helper methods for common logging patterns

// Info logs an info message with optional fields
func (zl *ZapLogger) Info(msg string, fields ...zap.Field) {
	zl.Logger.Info(msg, fields...)
}

// Error logs an error message with optional fields
func (zl *ZapLogger) Error(msg string, fields ...zap.Field) {
	zl.Logger.Error(msg, fields...)
}

// Warn logs a warning message with optional fields
func (zl *ZapLogger) Warn(msg string, fields ...zap.Field) {
	zl.Logger.Warn(msg, fields...)
}

// Debug logs a debug message with optional fields
func (zl *ZapLogger) Debug(msg string, fields ...zap.Field) {
	zl.Logger.Debug(msg, fields...)
}

// Fatal logs a fatal message and exits
func (zl *ZapLogger) Fatal(msg string, fields ...zap.Field) {
	zl.Logger.Fatal(msg, fields...)
}

// WithError creates a logger with an error field
func (zl *ZapLogger) WithError(err error) *zap.Logger {
	return zl.Logger.With(zap.Error(err))
}
