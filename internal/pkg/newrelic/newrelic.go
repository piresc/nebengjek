package newrelic

import (
	"log"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// InitNewRelic initializes New Relic application based on configuration
func InitNewRelic(configs *models.Config) *newrelic.Application {
	if !configs.NewRelic.Enabled || configs.NewRelic.LicenseKey == "" {
		log.Println("New Relic is disabled or license key not provided")
		return nil
	}

	log.Println("Initializing New Relic...")
	log.Printf("New Relic enabled - App: %s", configs.NewRelic.AppName)
	log.Printf("New Relic logs enabled: %v", configs.NewRelic.LogsEnabled)

	// Configure New Relic application with proper log forwarding
	nrApp, err := newrelic.NewApplication(
		newrelic.ConfigAppName(configs.NewRelic.AppName),
		newrelic.ConfigLicense(configs.NewRelic.LicenseKey),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(configs.NewRelic.ForwardLogs),
		newrelic.ConfigAppLogDecoratingEnabled(true), // Enable log decoration for correlation
	)
	if err != nil {
		log.Printf("Failed to initialize New Relic: %v (continuing without New Relic)", err)
		return nil // Continue without New Relic in development
	}

	return nrApp
}
