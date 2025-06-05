package newrelic

import (
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// InitNewRelic initializes New Relic application based on configuration
func InitNewRelic(configs *models.Config) *newrelic.Application {
	if !configs.NewRelic.Enabled || configs.NewRelic.LicenseKey == "" {
		logger.Info("New Relic is disabled or license key not provided")
		return nil
	}

	logger.Info("Initializing New Relic...")
	logger.Info("New Relic enabled",
		logger.String("app_name", configs.NewRelic.AppName))
	logger.Info("New Relic configuration",
		logger.Bool("logs_enabled", configs.NewRelic.LogsEnabled))

	// Configure New Relic application with proper log forwarding
	nrApp, err := newrelic.NewApplication(
		newrelic.ConfigAppName(configs.NewRelic.AppName),
		newrelic.ConfigLicense(configs.NewRelic.LicenseKey),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(configs.NewRelic.ForwardLogs),
		newrelic.ConfigAppLogDecoratingEnabled(true), // Enable log decoration for correlation
	)
	if err != nil {
		logger.Warn("Failed to initialize New Relic, continuing without New Relic",
			logger.Err(err))
		return nil // Continue without New Relic in development
	}

	return nrApp
}
