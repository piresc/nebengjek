package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/piresc/nebengjek/internal/pkg/models/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue string
		expected     string
		setup        func()
		cleanup      func()
	}{
		{
			name:         "Environment variable exists",
			key:          "TEST_ENV_VAR",
			value:        "test_value",
			defaultValue: "default_value",
			expected:     "test_value",
			setup: func() {
				os.Setenv("TEST_ENV_VAR", "test_value")
			},
			cleanup: func() {
				os.Unsetenv("TEST_ENV_VAR")
			},
		},
		{
			name:         "Environment variable does not exist",
			key:          "TEST_ENV_VAR_NONEXISTENT",
			value:        "",
			defaultValue: "default_value",
			expected:     "default_value",
			setup:        func() {},
			cleanup:      func() {},
		},
		{
			name:         "Empty environment variable",
			key:          "TEST_ENV_VAR_EMPTY",
			value:        "",
			defaultValue: "default_value",
			expected:     "default_value",
			setup: func() {
				os.Setenv("TEST_ENV_VAR_EMPTY", "")
			},
			cleanup: func() {
				os.Unsetenv("TEST_ENV_VAR_EMPTY")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			result := GetEnv(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvAsInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue int
		expected     int
		setup        func()
		cleanup      func()
	}{
		{
			name:         "Valid integer",
			key:          "TEST_INT_VAR",
			value:        "42",
			defaultValue: 0,
			expected:     42,
			setup: func() {
				os.Setenv("TEST_INT_VAR", "42")
			},
			cleanup: func() {
				os.Unsetenv("TEST_INT_VAR")
			},
		},
		{
			name:         "Invalid integer",
			key:          "TEST_INT_INVALID",
			value:        "not_a_number",
			defaultValue: 10,
			expected:     10,
			setup: func() {
				os.Setenv("TEST_INT_INVALID", "not_a_number")
			},
			cleanup: func() {
				os.Unsetenv("TEST_INT_INVALID")
			},
		},
		{
			name:         "Empty environment variable",
			key:          "TEST_INT_EMPTY",
			value:        "",
			defaultValue: 5,
			expected:     5,
			setup:        func() {},
			cleanup:      func() {},
		},
		{
			name:         "Negative integer",
			key:          "TEST_INT_NEGATIVE",
			value:        "-100",
			defaultValue: 0,
			expected:     -100,
			setup: func() {
				os.Setenv("TEST_INT_NEGATIVE", "-100")
			},
			cleanup: func() {
				os.Unsetenv("TEST_INT_NEGATIVE")
			},
		},
		{
			name:         "Zero value",
			key:          "TEST_INT_ZERO",
			value:        "0",
			defaultValue: 10,
			expected:     0,
			setup: func() {
				os.Setenv("TEST_INT_ZERO", "0")
			},
			cleanup: func() {
				os.Unsetenv("TEST_INT_ZERO")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			result := GetEnvAsInt(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvAsInt64(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue int64
		expected     int64
		setup        func()
		cleanup      func()
	}{
		{
			name:         "Valid int64",
			key:          "TEST_INT64_VAR",
			value:        "9223372036854775807",
			defaultValue: 0,
			expected:     9223372036854775807,
			setup: func() {
				os.Setenv("TEST_INT64_VAR", "9223372036854775807")
			},
			cleanup: func() {
				os.Unsetenv("TEST_INT64_VAR")
			},
		},
		{
			name:         "Invalid int64",
			key:          "TEST_INT64_INVALID",
			value:        "not_a_number",
			defaultValue: 100,
			expected:     100,
			setup: func() {
				os.Setenv("TEST_INT64_INVALID", "not_a_number")
			},
			cleanup: func() {
				os.Unsetenv("TEST_INT64_INVALID")
			},
		},
		{
			name:         "Minimum int64 value",
			key:          "TEST_INT64_MIN",
			value:        "-9223372036854775808",
			defaultValue: 0,
			expected:     -9223372036854775808,
			setup: func() {
				os.Setenv("TEST_INT64_MIN", "-9223372036854775808")
			},
			cleanup: func() {
				os.Unsetenv("TEST_INT64_MIN")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			result := GetEnvAsInt64(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvAsBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue bool
		expected     bool
		setup        func()
		cleanup      func()
	}{
		{
			name:         "True value",
			key:          "TEST_BOOL_TRUE",
			value:        "true",
			defaultValue: false,
			expected:     true,
			setup: func() {
				os.Setenv("TEST_BOOL_TRUE", "true")
			},
			cleanup: func() {
				os.Unsetenv("TEST_BOOL_TRUE")
			},
		},
		{
			name:         "False value",
			key:          "TEST_BOOL_FALSE",
			value:        "false",
			defaultValue: true,
			expected:     false,
			setup: func() {
				os.Setenv("TEST_BOOL_FALSE", "false")
			},
			cleanup: func() {
				os.Unsetenv("TEST_BOOL_FALSE")
			},
		},
		{
			name:         "1 value",
			key:          "TEST_BOOL_1",
			value:        "1",
			defaultValue: false,
			expected:     true,
			setup: func() {
				os.Setenv("TEST_BOOL_1", "1")
			},
			cleanup: func() {
				os.Unsetenv("TEST_BOOL_1")
			},
		},
		{
			name:         "0 value",
			key:          "TEST_BOOL_0",
			value:        "0",
			defaultValue: true,
			expected:     false,
			setup: func() {
				os.Setenv("TEST_BOOL_0", "0")
			},
			cleanup: func() {
				os.Unsetenv("TEST_BOOL_0")
			},
		},
		{
			name:         "Invalid boolean",
			key:          "TEST_BOOL_INVALID",
			value:        "not_boolean",
			defaultValue: true,
			expected:     true,
			setup: func() {
				os.Setenv("TEST_BOOL_INVALID", "not_boolean")
			},
			cleanup: func() {
				os.Unsetenv("TEST_BOOL_INVALID")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			result := GetEnvAsBool(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvAsFloat(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue float64
		expected     float64
		setup        func()
		cleanup      func()
	}{
		{
			name:         "Valid float",
			key:          "TEST_FLOAT_VAR",
			value:        "3.14159",
			defaultValue: 0.0,
			expected:     3.14159,
			setup: func() {
				os.Setenv("TEST_FLOAT_VAR", "3.14159")
			},
			cleanup: func() {
				os.Unsetenv("TEST_FLOAT_VAR")
			},
		},
		{
			name:         "Invalid float",
			key:          "TEST_FLOAT_INVALID",
			value:        "not_a_float",
			defaultValue: 2.5,
			expected:     2.5,
			setup: func() {
				os.Setenv("TEST_FLOAT_INVALID", "not_a_float")
			},
			cleanup: func() {
				os.Unsetenv("TEST_FLOAT_INVALID")
			},
		},
		{
			name:         "Negative float",
			key:          "TEST_FLOAT_NEGATIVE",
			value:        "-2.718",
			defaultValue: 0.0,
			expected:     -2.718,
			setup: func() {
				os.Setenv("TEST_FLOAT_NEGATIVE", "-2.718")
			},
			cleanup: func() {
				os.Unsetenv("TEST_FLOAT_NEGATIVE")
			},
		},
		{
			name:         "Zero float",
			key:          "TEST_FLOAT_ZERO",
			value:        "0.0",
			defaultValue: 1.0,
			expected:     0.0,
			setup: func() {
				os.Setenv("TEST_FLOAT_ZERO", "0.0")
			},
			cleanup: func() {
				os.Unsetenv("TEST_FLOAT_ZERO")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			result := GetEnvAsFloat(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	// Setup test environment variables
	testEnv := map[string]string{
		"APP_NAME":                       "test-app",
		"APP_ENV":                        "test",
		"APP_DEBUG":                      "false",
		"APP_VERSION":                    "1.0.0",
		"SERVER_HOST":                    "localhost",
		"SERVER_PORT":                    "8080",
		"SERVER_GRPC_PORT":               "9090",
		"SERVER_READ_TIMEOUT":            "30",
		"SERVER_WRITE_TIMEOUT":           "30",
		"SERVER_SHUTDOWN_TIMEOUT":        "30",
		"DB_DRIVER":                      "postgres",
		"DB_HOST":                        "localhost",
		"DB_PORT":                        "5432",
		"DB_USERNAME":                    "testuser",
		"DB_PASSWORD":                    "testpass",
		"DB_DATABASE":                    "testdb",
		"DB_SSL_MODE":                    "disable",
		"DB_MAX_CONNS":                   "10",
		"DB_IDLE_CONNS":                  "5",
		"REDIS_HOST":                     "localhost",
		"REDIS_PORT":                     "6379",
		"REDIS_PASSWORD":                 "redispass",
		"REDIS_DB":                       "0",
		"REDIS_POOL_SIZE":                "10",
		"NATS_URL":                       "nats://localhost:4222",
		"JWT_SECRET":                     "jwt-secret",
		"JWT_EXPIRATION":                 "3600",
		"JWT_ISSUER":                     "test-issuer",
		"MATCH_SERVICE_URL":              "http://localhost:9993",
		"RIDES_SERVICE_URL":             "http://localhost:9992",
		"LOCATION_SERVICE_URL":           "http://localhost:9994",
		"MATCH_SEARCH_RADIUS_KM":         "5.0",
		"PRICING_RATE_PER_KM":            "3000.0",
		"BILLING_ADMIN_FEE_PERCENT":      "5.0",
		"RIDES_MIN_DISTANCE_KM":          "1.0",
		"RIDES_MAX_PICKUP_DISTANCE_M":    "100.0",
		"USERS_FINDER_CACHE_TTL_MINUTES": "5",
		"USERS_BEACON_CACHE_TTL_MINUTES": "5",
		"PAYMENT_QR_CODE_BASE_URL":       "https://payment.test.com/qr",
		"PAYMENT_GATEWAY_URL":            "https://payment.test.com/api",
		"PAYMENT_TIMEOUT":                "30",
		"NEW_RELIC_LICENSE_KEY":          "test-license",
		"NEW_RELIC_APP_NAME":             "test-app",
		"NEW_RELIC_ENABLED":              "true",
		"NEW_RELIC_LOGS_ENABLED":         "true",
		"NEW_RELIC_LOGS_ENDPOINT":        "https://log-api.newrelic.com/log/v1",
		"NEW_RELIC_LOGS_API_KEY":         "test-log-api-key",
		"NEW_RELIC_FORWARD_LOGS":         "true",
		"API_KEY_USER_SERVICE":           "user-api-key",
		"API_KEY_MATCH_SERVICE":          "match-api-key",
		"API_KEY_RIDES_SERVICE":         "rides-api-key",
		"API_KEY_LOCATION_SERVICE":       "location-api-key",
		"API_KEY_GATEWAY_SERVICE":       "gateway-api-key",
		"API_KEY_NOTIFICATION_SERVICE":   "notification-api-key",
		"LOG_LEVEL":                      "info",
		"LOG_FILE_PATH":                  "logs/test.log",
		"LOG_MAX_SIZE":                   "100",
		"LOG_MAX_AGE":                    "7",
		"LOG_MAX_BACKUPS":                "3",
		"LOG_COMPRESS":                   "true",
		"LOG_TYPE":                       "file",
	}

	// Set environment variables
	for key, value := range testEnv {
		os.Setenv(key, value)
	}

	// Cleanup function
	cleanup := func() {
		for key := range testEnv {
			os.Unsetenv(key)
		}
	}
	defer cleanup()

	// Test loading config
	config := loadConfigFromEnv()

	// Verify all configurations are loaded correctly
	assert.Equal(t, "test-app", config.App.Name)
	assert.Equal(t, "test", config.App.Environment)
	assert.Equal(t, false, config.App.Debug)
	assert.Equal(t, "1.0.0", config.App.Version)

	assert.Equal(t, "localhost", config.Server.Host)
	assert.Equal(t, 8080, config.Server.Port)
	assert.Equal(t, 9090, config.Server.GRPCPort)
	assert.Equal(t, 30, config.Server.ReadTimeout)
	assert.Equal(t, 30, config.Server.WriteTimeout)
	assert.Equal(t, 30, config.Server.ShutdownTimeout)

	assert.Equal(t, "postgres", config.Database.Driver)
	assert.Equal(t, "localhost", config.Database.Host)
	assert.Equal(t, 5432, config.Database.Port)
	assert.Equal(t, "testuser", config.Database.Username)
	assert.Equal(t, "testpass", config.Database.Password)
	assert.Equal(t, "testdb", config.Database.Database)
	assert.Equal(t, "disable", config.Database.SSLMode)
	assert.Equal(t, 10, config.Database.MaxConns)
	assert.Equal(t, 5, config.Database.IdleConns)

	assert.Equal(t, "localhost", config.Redis.Host)
	assert.Equal(t, 6379, config.Redis.Port)
	assert.Equal(t, "redispass", config.Redis.Password)
	assert.Equal(t, 0, config.Redis.DB)
	assert.Equal(t, 10, config.Redis.PoolSize)

	assert.Equal(t, "nats://localhost:4222", config.NATS.URL)

	assert.Equal(t, "jwt-secret", config.JWT.Secret)
	assert.Equal(t, 3600, config.JWT.Expiration)
	assert.Equal(t, "test-issuer", config.JWT.Issuer)

	assert.Equal(t, "http://localhost:9993", config.Services.MatchServiceURL)
	assert.Equal(t, "http://localhost:9992", config.Services.RidesServiceURL)
	assert.Equal(t, "http://localhost:9994", config.Services.LocationServiceURL)

	assert.Equal(t, 5.0, config.Match.SearchRadiusKm)
	assert.Equal(t, 3000.0, config.Pricing.RatePerKm)
	assert.Equal(t, 5.0, config.Pricing.AdminFeePercent)
	assert.Equal(t, 1.0, config.Rides.MinDistanceKm)
	assert.Equal(t, 100.0, config.Rides.MaxPickupDistanceM)
	assert.Equal(t, 5, config.Users.FinderCacheTTLMinutes)
	assert.Equal(t, 5, config.Users.BeaconCacheTTLMinutes)

	assert.Equal(t, "https://payment.test.com/qr", config.Payment.QRCodeBaseURL)
	assert.Equal(t, "https://payment.test.com/api", config.Payment.GatewayURL)
	assert.Equal(t, 30, config.Payment.Timeout)

	assert.Equal(t, "test-license", config.NewRelic.LicenseKey)
	assert.Equal(t, "test-app", config.NewRelic.AppName)
	assert.Equal(t, true, config.NewRelic.Enabled)
	assert.Equal(t, true, config.NewRelic.LogsEnabled)
	assert.Equal(t, "https://log-api.newrelic.com/log/v1", config.NewRelic.LogsEndpoint)
	assert.Equal(t, "test-log-api-key", config.NewRelic.LogsAPIKey)
	assert.Equal(t, true, config.NewRelic.ForwardLogs)

	assert.Equal(t, "user-api-key", config.APIKey.UserService)
	assert.Equal(t, "match-api-key", config.APIKey.MatchService)
	assert.Equal(t, "rides-api-key", config.APIKey.RidesService)
	assert.Equal(t, "location-api-key", config.APIKey.LocationService)
	assert.Equal(t, "gateway-api-key", config.APIKey.GatewayService)
	assert.Equal(t, "notification-api-key", config.APIKey.NotificationService)

	assert.Equal(t, "info", config.Logger.Level)
	assert.Equal(t, "logs/test.log", config.Logger.FilePath)
	assert.Equal(t, int64(100), config.Logger.MaxSize)
	assert.Equal(t, 7, config.Logger.MaxAge)
	assert.Equal(t, 3, config.Logger.MaxBackups)
	assert.Equal(t, true, config.Logger.Compress)
	assert.Equal(t, "file", config.Logger.Type)
}

func TestLoadConfigFromEnv_DefaultValues(t *testing.T) {
	// Clear all relevant environment variables
	envVars := []string{
		"APP_NAME", "APP_ENV", "APP_DEBUG", "APP_VERSION",
		"SERVER_HOST", "SERVER_PORT", "SERVER_GRPC_PORT",
		"SERVER_READ_TIMEOUT", "SERVER_WRITE_TIMEOUT", "SERVER_SHUTDOWN_TIMEOUT",
		"DB_DRIVER", "DB_HOST", "DB_PORT", "DB_USERNAME", "DB_PASSWORD", "DB_DATABASE",
		"DB_SSL_MODE", "DB_MAX_CONNS", "DB_IDLE_CONNS",
		"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB", "REDIS_POOL_SIZE",
		"NATS_URL", "JWT_SECRET", "JWT_EXPIRATION", "JWT_ISSUER",
		"MATCH_SEARCH_RADIUS_KM", "PRICING_RATE_PER_KM", "BILLING_ADMIN_FEE_PERCENT",
		"RIDES_MIN_DISTANCE_KM", "RIDES_MAX_PICKUP_DISTANCE_M",
		"USERS_FINDER_CACHE_TTL_MINUTES", "USERS_BEACON_CACHE_TTL_MINUTES",
		"PAYMENT_QR_CODE_BASE_URL", "PAYMENT_GATEWAY_URL", "PAYMENT_TIMEOUT",
		"NEW_RELIC_LICENSE_KEY", "NEW_RELIC_APP_NAME", "NEW_RELIC_ENABLED",
		"NEW_RELIC_LOGS_ENABLED", "NEW_RELIC_LOGS_ENDPOINT", "NEW_RELIC_LOGS_API_KEY",
		"NEW_RELIC_FORWARD_LOGS",
		"API_KEY_USER_SERVICE", "API_KEY_MATCH_SERVICE", "API_KEY_RIDES_SERVICE",
		"API_KEY_LOCATION_SERVICE", "API_KEY_GATEWAY_SERVICE", "API_KEY_NOTIFICATION_SERVICE",
		"LOG_LEVEL", "LOG_FILE_PATH", "LOG_MAX_SIZE", "LOG_MAX_AGE", "LOG_MAX_BACKUPS",
		"LOG_COMPRESS", "LOG_TYPE",
	}

	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}

	config := loadConfigFromEnv()

	// Verify default values are used
	assert.Equal(t, "", config.App.Name)
	assert.Equal(t, "", config.App.Environment)
	assert.Equal(t, true, config.App.Debug)
	assert.Equal(t, "", config.App.Version)

	assert.Equal(t, "", config.Server.Host)
	assert.Equal(t, 0, config.Server.Port)
	assert.Equal(t, 0, config.Server.GRPCPort)
	assert.Equal(t, 0, config.Server.ReadTimeout)
	assert.Equal(t, 0, config.Server.WriteTimeout)
	assert.Equal(t, 0, config.Server.ShutdownTimeout)

	assert.Equal(t, "", config.Database.Driver)
	assert.Equal(t, "", config.Database.Host)
	assert.Equal(t, 0, config.Database.Port)
	assert.Equal(t, "", config.Database.Username)
	assert.Equal(t, "", config.Database.Password)
	assert.Equal(t, "", config.Database.Database)
	assert.Equal(t, "", config.Database.SSLMode)
	assert.Equal(t, 0, config.Database.MaxConns)
	assert.Equal(t, 0, config.Database.IdleConns)

	assert.Equal(t, "", config.Redis.Host)
	assert.Equal(t, 0, config.Redis.Port)
	assert.Equal(t, "", config.Redis.Password)
	assert.Equal(t, 0, config.Redis.DB)
	assert.Equal(t, 0, config.Redis.PoolSize)

	assert.Equal(t, "", config.NATS.URL)

	assert.Equal(t, "", config.JWT.Secret)
	assert.Equal(t, 0, config.JWT.Expiration)
	assert.Equal(t, "", config.JWT.Issuer)

	assert.Equal(t, "http://localhost:9993", config.Services.MatchServiceURL)
	assert.Equal(t, "http://localhost:9992", config.Services.RidesServiceURL)
	assert.Equal(t, "http://localhost:9994", config.Services.LocationServiceURL)

	assert.Equal(t, 1.0, config.Match.SearchRadiusKm)
	assert.Equal(t, 3000.0, config.Pricing.RatePerKm)
	assert.Equal(t, 5.0, config.Pricing.AdminFeePercent)
	assert.Equal(t, 1.0, config.Rides.MinDistanceKm)
	assert.Equal(t, 100.0, config.Rides.MaxPickupDistanceM)
	assert.Equal(t, 5, config.Users.FinderCacheTTLMinutes)
	assert.Equal(t, 5, config.Users.BeaconCacheTTLMinutes)

	assert.Equal(t, "https://payment.nebengjek.com/qr", config.Payment.QRCodeBaseURL)
	assert.Equal(t, "https://payment.nebengjek.com/api", config.Payment.GatewayURL)
	assert.Equal(t, 30, config.Payment.Timeout)

	assert.Equal(t, "", config.NewRelic.LicenseKey)
	assert.Equal(t, "", config.NewRelic.AppName)
	assert.Equal(t, false, config.NewRelic.Enabled)
	assert.Equal(t, false, config.NewRelic.LogsEnabled)
	assert.Equal(t, "", config.NewRelic.LogsEndpoint)
	assert.Equal(t, "", config.NewRelic.LogsAPIKey)
	assert.Equal(t, false, config.NewRelic.ForwardLogs)

	assert.Equal(t, "", config.APIKey.UserService)
	assert.Equal(t, "", config.APIKey.MatchService)
	assert.Equal(t, "", config.APIKey.RidesService)
	assert.Equal(t, "", config.APIKey.LocationService)
	assert.Equal(t, "", config.APIKey.GatewayService)
	assert.Equal(t, "", config.APIKey.NotificationService)

	assert.Equal(t, "info", config.Logger.Level)
	assert.Equal(t, "logs/nebengjek.log", config.Logger.FilePath)
	assert.Equal(t, int64(100), config.Logger.MaxSize)
	assert.Equal(t, 7, config.Logger.MaxAge)
	assert.Equal(t, 3, config.Logger.MaxBackups)
	assert.Equal(t, true, config.Logger.Compress)
	assert.Equal(t, "file", config.Logger.Type)
}

func TestInitConfig_LocalEnvironment(t *testing.T) {
	// Create a temporary .env file
	tempDir := t.TempDir()
	envFilePath := filepath.Join(tempDir, "test.env")
	envContent := `APP_NAME=test-app-from-file
APP_ENV=local
APP_DEBUG=true
APP_VERSION=1.0.0-test
SERVER_HOST=localhost-from-file
SERVER_PORT=8080`

	err := os.WriteFile(envFilePath, []byte(envContent), 0644)
	require.NoError(t, err)

	// Set APP_ENV to local to trigger file loading
	os.Setenv("APP_ENV", "local")
	defer os.Unsetenv("APP_ENV")

	// Clear other environment variables to ensure file loading is tested
	os.Unsetenv("APP_NAME")
	os.Unsetenv("APP_DEBUG")
	os.Unsetenv("APP_VERSION")
	os.Unsetenv("SERVER_HOST")
	os.Unsetenv("SERVER_PORT")

	config := InitConfig(envFilePath)

	// Verify config is loaded from file
	assert.Equal(t, "test-app-from-file", config.App.Name)
	assert.Equal(t, "local", config.App.Environment)
	assert.Equal(t, true, config.App.Debug)
	assert.Equal(t, "1.0.0-test", config.App.Version)
	assert.Equal(t, "localhost-from-file", config.Server.Host)
	assert.Equal(t, 8080, config.Server.Port)
}

func TestInitConfig_NonLocalEnvironment(t *testing.T) {
	// Set APP_ENV to production to skip file loading
	os.Setenv("APP_ENV", "production")
	defer os.Unsetenv("APP_ENV")

	// Set some environment variables
	os.Setenv("APP_NAME", "production-app")
	os.Setenv("APP_DEBUG", "false")
	defer os.Unsetenv("APP_NAME")
	defer os.Unsetenv("APP_DEBUG")

	config := InitConfig("nonexistent-file.env")

	// Verify config is loaded from environment (not file)
	assert.Equal(t, "production-app", config.App.Name)
	assert.Equal(t, "production", config.App.Environment)
	assert.Equal(t, false, config.App.Debug)
}

func TestInitConfig_MissingFile(t *testing.T) {
	// Set APP_ENV to local but provide a non-existent file
	os.Setenv("APP_ENV", "local")
	defer os.Unsetenv("APP_ENV")

	// Clear environment variables to test default behavior
	os.Unsetenv("APP_NAME")
	os.Unsetenv("APP_DEBUG")
	os.Unsetenv("APP_VERSION")

	config := InitConfig("nonexistent-file.env")

	// Verify default values are used when file doesn't exist
	assert.Equal(t, "", config.App.Name)
	assert.Equal(t, "local", config.App.Environment)
	assert.Equal(t, true, config.App.Debug)
	assert.Equal(t, "", config.App.Version)
}

func TestConcurrentConfigLoading(t *testing.T) {
	// Set up environment variables
	testEnv := map[string]string{
		"APP_NAME":    "concurrent-test",
		"APP_VERSION": "1.0.0",
		"SERVER_PORT": "8080",
	}

	for key, value := range testEnv {
		os.Setenv(key, value)
	}

	// Cleanup function
	cleanup := func() {
		for key := range testEnv {
			os.Unsetenv(key)
		}
	}
	defer cleanup()

	// Test concurrent config loading
	done := make(chan bool, 10)
	configs := make(chan *core.Config, 10)

	for i := 0; i < 10; i++ {
		go func() {
			config := loadConfigFromEnv()
			configs <- config
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all configs are identical
	close(configs)
	var firstConfig *core.Config
	for config := range configs {
		if firstConfig == nil {
			firstConfig = config
		} else {
			assert.Equal(t, firstConfig, config)
		}
	}

	// Verify config values
	assert.Equal(t, "concurrent-test", firstConfig.App.Name)
	assert.Equal(t, "1.0.0", firstConfig.App.Version)
	assert.Equal(t, 8080, firstConfig.Server.Port)
}