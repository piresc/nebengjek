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

	logger.Info("Initializing New Relic...",
		logger.String("app_name", configs.NewRelic.AppName))

	nrApp, err := newrelic.NewApplication(
		newrelic.ConfigAppName(configs.NewRelic.AppName),
		newrelic.ConfigLicense(configs.NewRelic.LicenseKey),
		newrelic.ConfigDistributedTracerEnabled(true),
	)
	if err != nil {
		logger.Warn("Failed to initialize New Relic, continuing without New Relic",
			logger.Err(err))
		return nil
	}

	return nrApp
}
