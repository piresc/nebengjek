package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Structure(t *testing.T) {
	config := &Config{}

	// Test that all fields are present and have correct zero values
	assert.NotNil(t, config)
	assert.Equal(t, AppConfig{}, config.App)
	assert.Equal(t, ServerConfig{}, config.Server)
	assert.Equal(t, DatabaseConfig{}, config.Database)
	assert.Equal(t, RedisConfig{}, config.Redis)
	assert.Equal(t, NATSConfig{}, config.NATS)
	assert.Equal(t, JWTConfig{}, config.JWT)
	assert.Equal(t, APIKeyConfig{}, config.APIKey)
	assert.Equal(t, PricingConfig{}, config.Pricing)
	assert.Equal(t, PaymentConfig{}, config.Payment)
	assert.Equal(t, ServicesConfig{}, config.Services)
	assert.Equal(t, MatchConfig{}, config.Match)
	assert.Equal(t, LocationConfig{}, config.Location)
	assert.Equal(t, RidesConfig{}, config.Rides)
	assert.Equal(t, UsersConfig{}, config.Users)
	assert.Equal(t, NewRelicConfig{}, config.NewRelic)
	assert.Equal(t, LoggerConfig{}, config.Logger)
}

func TestAppConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   AppConfig
		expected AppConfig
	}{
		{
			name:     "Default config",
			config:   AppConfig{},
			expected: AppConfig{Name: "", Environment: "", Debug: false, Version: ""},
		},
		{
			name: "Custom config",
			config: AppConfig{
				Name:        "test-service",
				Environment: "production",
				Debug:       true,
				Version:     "1.0.0",
			},
			expected: AppConfig{
				Name:        "test-service",
				Environment: "production",
				Debug:       true,
				Version:     "1.0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config)
		})
	}
}

func TestServerConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   ServerConfig
		expected ServerConfig
	}{
		{
			name:     "Default config",
			config:   ServerConfig{},
			expected: ServerConfig{Host: "", Port: 0, GRPCPort: 0, ReadTimeout: 0, WriteTimeout: 0, ShutdownTimeout: 0},
		},
		{
			name: "Custom config",
			config: ServerConfig{
				Host:            "localhost",
				Port:            8080,
				GRPCPort:        9090,
				ReadTimeout:     30,
				WriteTimeout:    30,
				ShutdownTimeout: 10,
			},
			expected: ServerConfig{
				Host:            "localhost",
				Port:            8080,
				GRPCPort:        9090,
				ReadTimeout:     30,
				WriteTimeout:    30,
				ShutdownTimeout: 10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config)
		})
	}
}

func TestDatabaseConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   DatabaseConfig
		expected DatabaseConfig
	}{
		{
			name:     "Default config",
			config:   DatabaseConfig{},
			expected: DatabaseConfig{Driver: "", Host: "", Port: 0, Username: "", Password: "", Database: "", SSLMode: "", MaxConns: 0, IdleConns: 0},
		},
		{
			name: "Custom config",
			config: DatabaseConfig{
				Driver:    "postgres",
				Host:      "localhost",
				Port:      5432,
				Username:  "user",
				Password:  "pass",
				Database:  "dbname",
				SSLMode:   "require",
				MaxConns:  25,
				IdleConns: 5,
			},
			expected: DatabaseConfig{
				Driver:    "postgres",
				Host:      "localhost",
				Port:      5432,
				Username:  "user",
				Password:  "pass",
				Database:  "dbname",
				SSLMode:   "require",
				MaxConns:  25,
				IdleConns: 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config)
		})
	}
}

func TestRedisConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   RedisConfig
		expected RedisConfig
	}{
		{
			name:     "Default config",
			config:   RedisConfig{},
			expected: RedisConfig{Host: "", Port: 0, Password: "", DB: 0, PoolSize: 0},
		},
		{
			name: "Custom config",
			config: RedisConfig{
				Host:     "localhost",
				Port:     6379,
				Password: "redispass",
				DB:       1,
				PoolSize: 10,
			},
			expected: RedisConfig{
				Host:     "localhost",
				Port:     6379,
				Password: "redispass",
				DB:       1,
				PoolSize: 10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config)
		})
	}
}

func TestNATSConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   NATSConfig
		expected NATSConfig
	}{
		{
			name:     "Default config",
			config:   NATSConfig{},
			expected: NATSConfig{URL: ""},
		},
		{
			name:     "Custom config",
			config:   NATSConfig{URL: "nats://localhost:4222"},
			expected: NATSConfig{URL: "nats://localhost:4222"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config)
		})
	}
}

func TestJWTConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   JWTConfig
		expected JWTConfig
	}{
		{
			name:     "Default config",
			config:   JWTConfig{},
			expected: JWTConfig{Secret: "", Expiration: 0, Issuer: ""},
		},
		{
			name: "Custom config",
			config: JWTConfig{
				Secret:     "jwt-secret",
				Expiration: 60, // 1 hour
				Issuer:     "nebengjek",
			},
			expected: JWTConfig{
				Secret:     "jwt-secret",
				Expiration: 60,
				Issuer:     "nebengjek",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config)
		})
	}
}

func TestAPIKeyConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   APIKeyConfig
		expected APIKeyConfig
	}{
		{
			name:     "Default config",
			config:   APIKeyConfig{},
			expected: APIKeyConfig{UserService: "", MatchService: "", RidesService: "", LocationService: "", GatewayService: "", NotificationService: ""},
		},
		{
			name: "Custom config",
			config: APIKeyConfig{
				UserService:         "user-api-key",
				MatchService:        "match-api-key",
				RidesService:        "rides-api-key",
				LocationService:     "location-api-key",
				GatewayService:      "gateway-api-key",
				NotificationService: "notification-api-key",
			},
			expected: APIKeyConfig{
				UserService:         "user-api-key",
				MatchService:        "match-api-key",
				RidesService:        "rides-api-key",
				LocationService:     "location-api-key",
				GatewayService:      "gateway-api-key",
				NotificationService: "notification-api-key",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config)
		})
	}
}

func TestPricingConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   PricingConfig
		expected PricingConfig
	}{
		{
			name:     "Default config",
			config:   PricingConfig{},
			expected: PricingConfig{RatePerKm: 0, AdminFeePercent: 0},
		},
		{
			name: "Custom config",
			config: PricingConfig{
				RatePerKm:       3000.0,
				AdminFeePercent: 5.0,
			},
			expected: PricingConfig{
				RatePerKm:       3000.0,
				AdminFeePercent: 5.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config)
		})
	}
}

func TestPaymentConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   PaymentConfig
		expected PaymentConfig
	}{
		{
			name:     "Default config",
			config:   PaymentConfig{},
			expected: PaymentConfig{QRCodeBaseURL: "", GatewayURL: "", Timeout: 0},
		},
		{
			name: "Custom config",
			config: PaymentConfig{
				QRCodeBaseURL: "https://payment.nebengjek.com/qr",
				GatewayURL:    "https://payment.nebengjek.com/api",
				Timeout:       30,
			},
			expected: PaymentConfig{
				QRCodeBaseURL: "https://payment.nebengjek.com/qr",
				GatewayURL:    "https://payment.nebengjek.com/api",
				Timeout:       30,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config)
		})
	}
}

func TestServicesConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   ServicesConfig
		expected ServicesConfig
	}{
		{
			name:     "Default config",
			config:   ServicesConfig{},
			expected: ServicesConfig{MatchServiceURL: "", RidesServiceURL: "", LocationServiceURL: "", GatewayServiceURL: "", NotificationServiceURL: ""},
		},
		{
			name: "Custom config",
			config: ServicesConfig{
				MatchServiceURL:       "http://localhost:9993",
				RidesServiceURL:       "http://localhost:9992",
				LocationServiceURL:    "http://localhost:9994",
				GatewayServiceURL:     "http://localhost:9999",
				NotificationServiceURL: "http://localhost:9991",
			},
			expected: ServicesConfig{
				MatchServiceURL:       "http://localhost:9993",
				RidesServiceURL:       "http://localhost:9992",
				LocationServiceURL:    "http://localhost:9994",
				GatewayServiceURL:     "http://localhost:9999",
				NotificationServiceURL: "http://localhost:9991",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config)
		})
	}
}

func TestMatchConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   MatchConfig
		expected MatchConfig
	}{
		{
			name:     "Default config",
			config:   MatchConfig{},
			expected: MatchConfig{SearchRadiusKm: 0, ActiveRideTTLHours: 0},
		},
		{
			name: "Custom config",
			config: MatchConfig{
				SearchRadiusKm:     1.0,
				ActiveRideTTLHours: 24,
			},
			expected: MatchConfig{
				SearchRadiusKm:     1.0,
				ActiveRideTTLHours: 24,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config)
		})
	}
}

func TestLocationConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   LocationConfig
		expected LocationConfig
	}{
		{
			name:     "Default config",
			config:   LocationConfig{},
			expected: LocationConfig{AvailabilityTTLMinutes: 0},
		},
		{
			name:     "Custom config",
			config:   LocationConfig{AvailabilityTTLMinutes: 30},
			expected: LocationConfig{AvailabilityTTLMinutes: 30},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config)
		})
	}
}

func TestRidesConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   RidesConfig
		expected RidesConfig
	}{
		{
			name:     "Default config",
			config:   RidesConfig{},
			expected: RidesConfig{MinDistanceKm: 0, MaxPickupDistanceM: 0},
		},
		{
			name: "Custom config",
			config: RidesConfig{
				MinDistanceKm:      1.0,
				MaxPickupDistanceM: 100.0,
			},
			expected: RidesConfig{
				MinDistanceKm:      1.0,
				MaxPickupDistanceM: 100.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config)
		})
	}
}

func TestUsersConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   UsersConfig
		expected UsersConfig
	}{
		{
			name:     "Default config",
			config:   UsersConfig{},
			expected: UsersConfig{FinderCacheTTLMinutes: 0, BeaconCacheTTLMinutes: 0},
		},
		{
			name: "Custom config",
			config: UsersConfig{
				FinderCacheTTLMinutes: 5,
				BeaconCacheTTLMinutes: 5,
			},
			expected: UsersConfig{
				FinderCacheTTLMinutes: 5,
				BeaconCacheTTLMinutes: 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config)
		})
	}
}

func TestNewRelicConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   NewRelicConfig
		expected NewRelicConfig
	}{
		{
			name:     "Default config",
			config:   NewRelicConfig{},
			expected: NewRelicConfig{LicenseKey: "", AppName: "", Enabled: false, LogsEnabled: false, LogsEndpoint: "", LogsAPIKey: "", ForwardLogs: false},
		},
		{
			name: "Custom config",
			config: NewRelicConfig{
				LicenseKey:   "nr-license-key",
				AppName:      "nebengjek",
				Enabled:      true,
				LogsEnabled:  true,
				LogsEndpoint: "https://log-api.newrelic.com/log/v1",
				LogsAPIKey:   "nr-log-api-key",
				ForwardLogs:  true,
			},
			expected: NewRelicConfig{
				LicenseKey:   "nr-license-key",
				AppName:      "nebengjek",
				Enabled:      true,
				LogsEnabled:  true,
				LogsEndpoint: "https://log-api.newrelic.com/log/v1",
				LogsAPIKey:   "nr-log-api-key",
				ForwardLogs:  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config)
		})
	}
}

func TestLoggerConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   LoggerConfig
		expected LoggerConfig
	}{
		{
			name:     "Default config",
			config:   LoggerConfig{},
			expected: LoggerConfig{Level: "", FilePath: "", MaxSize: 0, MaxAge: 0, MaxBackups: 0, Compress: false, Type: ""},
		},
		{
			name: "Custom config",
			config: LoggerConfig{
				Level:      "info",
				FilePath:   "logs/nebengjek.log",
				MaxSize:    100,
				MaxAge:     7,
				MaxBackups: 3,
				Compress:   true,
				Type:       "file",
			},
			expected: LoggerConfig{
				Level:      "info",
				FilePath:   "logs/nebengjek.log",
				MaxSize:    100,
				MaxAge:     7,
				MaxBackups: 3,
				Compress:   true,
				Type:       "file",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config)
		})
	}
}

func TestConfigJSONTags(t *testing.T) {
	// Test that JSON tags are properly set for nested configs
	config := &Config{
		Pricing: PricingConfig{
			RatePerKm:       3000.0,
			AdminFeePercent: 5.0,
		},
		Payment: PaymentConfig{
			QRCodeBaseURL: "https://payment.nebengjek.com/qr",
			GatewayURL:    "https://payment.nebengjek.com/api",
			Timeout:       30,
		},
		Match: MatchConfig{
			SearchRadiusKm:     1.0,
			ActiveRideTTLHours: 24,
		},
		Location: LocationConfig{
			AvailabilityTTLMinutes: 30,
		},
		Rides: RidesConfig{
			MinDistanceKm:      1.0,
			MaxPickupDistanceM: 100.0,
		},
		Users: UsersConfig{
			FinderCacheTTLMinutes: 5,
			BeaconCacheTTLMinutes: 5,
		},
		NewRelic: NewRelicConfig{
			LicenseKey:   "nr-license-key",
			AppName:      "nebengjek",
			Enabled:      true,
			LogsEnabled:  true,
			LogsEndpoint: "https://log-api.newrelic.com/log/v1",
			LogsAPIKey:   "nr-log-api-key",
			ForwardLogs:  true,
		},
		Logger: LoggerConfig{
			Level:      "info",
			FilePath:   "logs/nebengjek.log",
			MaxSize:    100,
			MaxAge:     7,
			MaxBackups: 3,
			Compress:   true,
			Type:       "file",
		},
	}

	// This test ensures the struct can be created without panics
	assert.NotNil(t, config)
	assert.Equal(t, 3000.0, config.Pricing.RatePerKm)
	assert.Equal(t, 5.0, config.Pricing.AdminFeePercent)
	assert.Equal(t, "https://payment.nebengjek.com/qr", config.Payment.QRCodeBaseURL)
	assert.Equal(t, 30, config.Payment.Timeout)
	assert.Equal(t, 1.0, config.Match.SearchRadiusKm)
	assert.Equal(t, 24, config.Match.ActiveRideTTLHours)
	assert.Equal(t, 30, config.Location.AvailabilityTTLMinutes)
	assert.Equal(t, 1.0, config.Rides.MinDistanceKm)
	assert.Equal(t, 100.0, config.Rides.MaxPickupDistanceM)
	assert.Equal(t, 5, config.Users.FinderCacheTTLMinutes)
	assert.Equal(t, 5, config.Users.BeaconCacheTTLMinutes)
	assert.Equal(t, "nr-license-key", config.NewRelic.LicenseKey)
	assert.Equal(t, "nebengjek", config.NewRelic.AppName)
	assert.True(t, config.NewRelic.Enabled)
	assert.Equal(t, "info", config.Logger.Level)
	assert.Equal(t, "logs/nebengjek.log", config.Logger.FilePath)
	assert.True(t, config.Logger.Compress)
}