package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/sirupsen/logrus"
)

// AppLogger is our custom logger that supports multiple outputs
type AppLogger struct {
	*logrus.Logger
	nrApp    *newrelic.Application
	filePath string
	file     *os.File
}

// Config holds logger configuration
type Config struct {
	Level      string `json:"level" mapstructure:"level"`
	FilePath   string `json:"file_path" mapstructure:"file_path"`
	MaxSize    int64  `json:"max_size" mapstructure:"max_size"`       // Max size in MB before rotation
	MaxAge     int    `json:"max_age" mapstructure:"max_age"`         // Max age in days
	MaxBackups int    `json:"max_backups" mapstructure:"max_backups"` // Max number of backup files
	Compress   bool   `json:"compress" mapstructure:"compress"`       // Compress rotated files
}

// NewAppLogger creates a new application logger
func NewAppLogger(config Config, nrApp *newrelic.Application) (*AppLogger, error) {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set JSON formatter for structured logging
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	appLogger := &AppLogger{
		Logger: logger,
		nrApp:  nrApp,
	}

	// Setup file output if path is provided
	if config.FilePath != "" {
		if err := appLogger.setupFileOutput(config.FilePath); err != nil {
			return nil, fmt.Errorf("failed to setup file output: %w", err)
		}
	}

	return appLogger, nil
}

// setupFileOutput configures file output for the logger
func (al *AppLogger) setupFileOutput(filePath string) error {
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

	al.filePath = filePath
	al.file = file

	// Set output to both stdout and file
	al.Logger.SetOutput(io.MultiWriter(os.Stdout, file))

	return nil
}

// Close closes the log file
func (al *AppLogger) Close() error {
	if al.file != nil {
		return al.file.Close()
	}
	return nil
}

// WithNewRelicContext adds New Relic context to log entry
func (al *AppLogger) WithNewRelicContext(txn *newrelic.Transaction) *logrus.Entry {
	entry := al.Logger.WithFields(logrus.Fields{})

	if txn != nil {
		// Add New Relic context
		ctx := newrelic.NewContext(context.Background(), txn)
		entry = entry.WithContext(ctx)

		// Add trace correlation fields
		if mdw := txn.GetLinkingMetadata(); mdw.TraceID != "" {
			entry = entry.WithFields(logrus.Fields{
				"trace.id": mdw.TraceID,
				"span.id":  mdw.SpanID,
			})
		}
	}

	return entry
}

// WithRequestContext adds request context fields
func (al *AppLogger) WithRequestContext(requestID, userID, method, path string) *logrus.Entry {
	return al.Logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"user_id":    userID,
		"method":     method,
		"path":       path,
		"service":    "nebengjek-users-app",
	})
}

// WithError adds error field to log entry
func (al *AppLogger) WithError(err error) *logrus.Entry {
	return al.Logger.WithError(err)
}

// WithFields adds custom fields to log entry
func (al *AppLogger) WithFields(fields logrus.Fields) *logrus.Entry {
	// Always add service name
	if fields == nil {
		fields = logrus.Fields{}
	}
	fields["service"] = "nebengjek-users-app"

	return al.Logger.WithFields(fields)
}

// LogHTTPRequest logs HTTP request with all relevant context
func (al *AppLogger) LogHTTPRequest(txn *newrelic.Transaction, method, path, clientIP, userID, requestID string, statusCode int, latency time.Duration, err error) {
	entry := al.WithFields(logrus.Fields{
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
		entry = al.WithNewRelicContext(txn)
		entry = entry.WithFields(logrus.Fields{
			"status":     statusCode,
			"latency":    latency.String(),
			"latency_ms": latency.Milliseconds(),
			"client_ip":  clientIP,
			"method":     method,
			"path":       path,
			"user_id":    userID,
			"request_id": requestID,
		})
	}

	// Log with appropriate level based on status code
	if statusCode >= 500 {
		if err != nil {
			entry.WithError(err).Error("Server error")
		} else {
			entry.Error("Server error")
		}
	} else if statusCode >= 400 {
		if err != nil {
			entry.WithError(err).Warn("Client error")
		} else {
			entry.Warn("Client error")
		}
	} else {
		entry.Info("Request processed")
	}
}

// AddHook adds a logrus hook to the logger
func (al *AppLogger) AddHook(hook logrus.Hook) {
	al.Logger.AddHook(hook)
}

// GetFilePath returns the current log file path
func (al *AppLogger) GetFilePath() string {
	return al.filePath
}

// RotateFile manually rotates the log file (useful for external log rotation)
func (al *AppLogger) RotateFile() error {
	if al.file == nil {
		return nil
	}

	// Close current file
	if err := al.file.Close(); err != nil {
		return err
	}

	// Reopen file
	return al.setupFileOutput(al.filePath)
}

// InitAppLogger initializes and returns a configured application logger
// This is the main initialization function that should be called from main.go
func InitAppLogger(loggerConfig Config, newrelicConfig NewRelicLogConfig, nrApp *newrelic.Application) (*AppLogger, error) {

	// Create logger factory
	loggerFactory := NewLoggerFactory(loggerConfig, nrApp, &newrelicConfig)

	// Determine logger type based on configuration
	loggerType := LoggerType(GetLoggerTypeFromConfig(loggerConfig, newrelicConfig))

	// Create logger
	appLogger, err := loggerFactory.CreateLogger(loggerType)
	if err != nil {
		return nil, err
	}

	return appLogger, nil
}

// InitAppLoggerFromConfig initializes logger directly from config models - one-liner initialization
func InitAppLoggerFromConfig(configs *models.Config, nrApp *newrelic.Application) (*AppLogger, error) {

	// Convert config models to logger config
	loggerConfig := Config{
		Level:      configs.Logger.Level,
		FilePath:   configs.Logger.FilePath,
		MaxSize:    configs.Logger.MaxSize,
		MaxAge:     configs.Logger.MaxAge,
		MaxBackups: configs.Logger.MaxBackups,
		Compress:   configs.Logger.Compress,
	}

	// New Relic config for logger
	nrLogConfig := NewRelicLogConfig{
		Enabled:     configs.NewRelic.LogsEnabled,
		LicenseKey:  configs.NewRelic.LogsAPIKey,
		Endpoint:    configs.NewRelic.LogsEndpoint,
		Timeout:     5,
		BatchSize:   100,
		FlushPeriod: 5,
	}

	// Create logger factory
	loggerFactory := NewLoggerFactory(loggerConfig, nrApp, &nrLogConfig)

	// Determine logger type based on configuration
	loggerType := LoggerType(configs.Logger.Type)
	if configs.NewRelic.LogsEnabled {
		loggerType = NewRelicLogger
	}

	// Create logger
	appLogger, err := loggerFactory.CreateLogger(loggerType)
	if err != nil {
		return nil, err
	}

	return appLogger, nil
}

// GetLoggerTypeFromConfig determines the appropriate logger type based on configuration
func GetLoggerTypeFromConfig(loggerConfig Config, newrelicConfig NewRelicLogConfig) string {
	// Priority: NewRelic logs enabled > environment/config type
	if newrelicConfig.Enabled {
		return string(NewRelicLogger)
	}

	// Use the configured type or default to file
	if loggerConfig.Level != "" {
		// If we have a specific type in config, use it
		// For now, we'll default to file logger
		return string(FileLogger)
	}

	return string(FileLogger)
}
