package newrelic

import (
	"testing"

	"github.com/piresc/nebengjek/internal/pkg/models/core"
	"github.com/stretchr/testify/assert"
)

func TestInitNewRelic_Disabled(t *testing.T) {
	config := &core.Config{
		NewRelic: core.NewRelicConfig{
			Enabled:   false,
			LicenseKey: "test-license",
			AppName:   "test-app",
		},
	}

	app := InitNewRelic(config)
	assert.Nil(t, app)
}

func TestInitNewRelic_NoLicenseKey(t *testing.T) {
	config := &core.Config{
		NewRelic: core.NewRelicConfig{
			Enabled:   true,
			LicenseKey: "",
			AppName:   "test-app",
		},
	}

	app := InitNewRelic(config)
	assert.Nil(t, app)
}

func TestInitNewRelic_EnabledWithLicense(t *testing.T) {
	config := &core.Config{
		NewRelic: core.NewRelicConfig{
			Enabled:   true,
			LicenseKey: "test-license",
			AppName:   "test-app",
		},
	}

	app := InitNewRelic(config)
	// Note: This will return nil because the license key is invalid,
	// but the function should not panic
	assert.Nil(t, app)
}

func TestInitNewRelic_NilConfig(t *testing.T) {
	app := InitNewRelic(nil)
	assert.Nil(t, app)
}

func TestInitNewRelic_EmptyAppName(t *testing.T) {
	config := &core.Config{
		NewRelic: core.NewRelicConfig{
			Enabled:   true,
			LicenseKey: "test-license",
			AppName:   "",
		},
	}

	app := InitNewRelic(config)
	assert.Nil(t, app)
}

func TestInitNewRelic_ValidConfig(t *testing.T) {
	config := &core.Config{
		NewRelic: core.NewRelicConfig{
			Enabled:   true,
			LicenseKey: "test-license-key-here",
			AppName:   "nebengjek-test",
		},
	}

	app := InitNewRelic(config)
	// With a fake license key, New Relic initialization will fail
	// but the function should handle it gracefully
	assert.Nil(t, app)
}

func TestInitNewRelic_ConfigEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		config   *core.Config
		expected bool
	}{
		{
			name: "Nil config",
			config: nil,
			expected: false,
		},
		{
			name: "Disabled with valid license",
			config: &core.Config{
				NewRelic: core.NewRelicConfig{
					Enabled:   false,
					LicenseKey: "valid-license-key",
					AppName:   "test-app",
				},
			},
			expected: false,
		},
		{
			name: "Enabled with empty license",
			config: &core.Config{
				NewRelic: core.NewRelicConfig{
					Enabled:   true,
					LicenseKey: "",
					AppName:   "test-app",
				},
			},
			expected: false,
		},
		{
			name: "Enabled with whitespace license",
			config: &core.Config{
				NewRelic: core.NewRelicConfig{
					Enabled:   true,
					LicenseKey: "   ",
					AppName:   "test-app",
				},
			},
			expected: false,
		},
		{
			name: "Enabled with empty app name",
			config: &core.Config{
				NewRelic: core.NewRelicConfig{
					Enabled:   true,
					LicenseKey: "test-license",
					AppName:   "",
				},
			},
			expected: false,
		},
		{
			name: "Enabled with valid config but invalid license",
			config: &core.Config{
				NewRelic: core.NewRelicConfig{
					Enabled:   true,
					LicenseKey: "invalid-license-key",
					AppName:   "test-app",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := InitNewRelic(tt.config)
			if tt.expected {
				assert.NotNil(t, app)
			} else {
				assert.Nil(t, app)
			}
		})
	}
}