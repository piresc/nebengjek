package logger

import (
	"github.com/newrelic/go-agent/v3/newrelic"
)

// LoggerType represents different logger configurations
type LoggerType string

const (
	// FileLogger writes logs to file only
	FileLogger LoggerType = "file"
	// ConsoleLogger writes logs to console only
	ConsoleLogger LoggerType = "console"
	// HybridLogger writes logs to both file and console
	HybridLogger LoggerType = "hybrid"
	// NewRelicLogger writes logs to file and forwards to New Relic
	NewRelicLogger LoggerType = "newrelic"
)

// LoggerFactory creates different types of loggers
type LoggerFactory struct {
	defaultConfig Config
	nrApp         *newrelic.Application
	nrConfig      *NewRelicLogConfig
}

// NewRelicLogConfig holds New Relic specific configuration
type NewRelicLogConfig struct {
	Enabled     bool   `json:"enabled" mapstructure:"enabled"`
	LicenseKey  string `json:"license_key" mapstructure:"license_key"`
	Endpoint    string `json:"endpoint" mapstructure:"endpoint"`
	Timeout     int    `json:"timeout" mapstructure:"timeout"` // seconds
	BatchSize   int    `json:"batch_size" mapstructure:"batch_size"`
	FlushPeriod int    `json:"flush_period" mapstructure:"flush_period"` // seconds
}

// NewLoggerFactory creates a new logger factory
func NewLoggerFactory(config Config, nrApp *newrelic.Application, nrConfig *NewRelicLogConfig) *LoggerFactory {
	return &LoggerFactory{
		defaultConfig: config,
		nrApp:         nrApp,
		nrConfig:      nrConfig,
	}
}

// CreateLogger creates a logger based on the specified type
func (f *LoggerFactory) CreateLogger(loggerType LoggerType) (*AppLogger, error) {
	switch loggerType {
	case FileLogger:
		return f.createFileLogger()
	case ConsoleLogger:
		return f.createConsoleLogger()
	case HybridLogger:
		return f.createHybridLogger()
	case NewRelicLogger:
		return f.createNewRelicLogger()
	default:
		return f.createHybridLogger() // Default to hybrid
	}
}

// createFileLogger creates a logger that writes only to file
func (f *LoggerFactory) createFileLogger() (*AppLogger, error) {
	config := f.defaultConfig
	// Ensure we have a file path
	if config.FilePath == "" {
		config.FilePath = "logs/nebengjek.log"
	}

	return NewAppLogger(config, f.nrApp)
}

// createConsoleLogger creates a logger that writes only to console
func (f *LoggerFactory) createConsoleLogger() (*AppLogger, error) {
	config := f.defaultConfig
	config.FilePath = "" // No file output

	return NewAppLogger(config, f.nrApp)
}

// createHybridLogger creates a logger that writes to both file and console
func (f *LoggerFactory) createHybridLogger() (*AppLogger, error) {
	config := f.defaultConfig
	// Ensure we have a file path
	if config.FilePath == "" {
		config.FilePath = "logs/nebengjek.log"
	}

	return NewAppLogger(config, f.nrApp)
}

// createNewRelicLogger creates a logger with New Relic integration
func (f *LoggerFactory) createNewRelicLogger() (*AppLogger, error) {
	// First create the base logger
	logger, err := f.createHybridLogger()
	if err != nil {
		return nil, err
	}

	// Add New Relic hook if configured
	if f.nrConfig != nil && f.nrConfig.Enabled && f.nrConfig.LicenseKey != "" {
		hook := NewNewRelicLogHook(*f.nrConfig)
		logger.AddHook(hook)
	}

	return logger, nil
}

// GetDefaultConfig returns a default logger configuration
func GetDefaultConfig() Config {
	return Config{
		Level:      "info",
		FilePath:   "logs/nebengjek.log",
		MaxSize:    100, // 100MB
		MaxAge:     7,   // 7 days
		MaxBackups: 3,   // 3 backup files
		Compress:   true,
	}
}

// GetDefaultNewRelicConfig returns a default New Relic configuration
func GetDefaultNewRelicConfig() NewRelicLogConfig {
	return NewRelicLogConfig{
		Enabled:     false,
		Endpoint:    "https://log-api.newrelic.com/log/v1",
		Timeout:     5, // 5 seconds
		BatchSize:   100,
		FlushPeriod: 5, // 5 seconds
	}
}
