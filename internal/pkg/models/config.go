package models

// Config represents application configuration
type Config struct {
	App      AppConfig
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	NATS     NATSConfig
	JWT      JWTConfig
	APIKey   APIKeyConfig
	Pricing  PricingConfig
	Payment  PaymentConfig
	Services ServicesConfig
	Match    MatchConfig
	Location LocationConfig
	Rides    RidesConfig
	NewRelic NewRelicConfig
	Logger   LoggerConfig
}

// ServicesConfig contains URLs for other microservices
type ServicesConfig struct {
	MatchServiceURL    string
	RidesServiceURL    string
	LocationServiceURL string
}

// AppConfig contains application-specific configuration
type AppConfig struct {
	Name        string
	Environment string
	Debug       bool
	Version     string
}

// ServerConfig contains HTTP/gRPC server configuration
type ServerConfig struct {
	Host            string
	Port            int
	GRPCPort        int
	ReadTimeout     int
	WriteTimeout    int
	ShutdownTimeout int
}

// DatabaseConfig contains database connection configuration
type DatabaseConfig struct {
	Driver    string
	Host      string
	Port      int
	Username  string
	Password  string
	Database  string
	SSLMode   string
	MaxConns  int
	IdleConns int
}

// RedisConfig contains Redis connection configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	PoolSize int
}

// NATSConfig contains NATS connection configuration
type NATSConfig struct {
	URL string
}

// JWTConfig contains JWT authentication configuration
type JWTConfig struct {
	Secret     string
	Expiration int // in minutes
	Issuer     string
}

// APIKeyConfig contains API key authentication configuration
type APIKeyConfig struct {
	UserService     string
	MatchService    string
	RidesService    string
	LocationService string
}

type PricingConfig struct {
	RatePerKm     float64 `json:"rate_per_km"`
	Currency      string  `json:"currency"`
	BaseFare      float64 `json:"base_fare"`
	PerKmRate     float64 `json:"per_km_rate"`
	PerMinuteRate float64 `json:"per_minute_rate"`
	SurgeFactor   float64 `json:"surge_factor"`
}

// PaymentConfig contains payment service configuration
type PaymentConfig struct {
	QRCodeBaseURL string `json:"qr_code_base_url"`
	GatewayURL    string `json:"gateway_url"`
	Timeout       int    `json:"timeout"` // timeout in seconds
}

// MatchConfig contains match service specific configuration
type MatchConfig struct {
	SearchRadiusKm     float64 `json:"search_radius_km"`      // Radius in kilometers for matching users
	ActiveRideTTLHours int     `json:"active_ride_ttl_hours"` // TTL in hours for active ride tracking
}

// LocationConfig contains location service specific configuration
type LocationConfig struct {
	AvailabilityTTLMinutes int `json:"availability_ttl_minutes"` // TTL in minutes for user availability in pools
}

// RidesConfig contains rides service specific configuration
type RidesConfig struct {
	MinDistanceKm float64 `json:"min_distance_km"` // Minimum distance in kilometers for billing
}

// NewRelicConfig contains New Relic monitoring configuration
type NewRelicConfig struct {
	LicenseKey   string `json:"license_key"`
	AppName      string `json:"app_name"`
	Enabled      bool   `json:"enabled"`
	LogsEnabled  bool   `json:"logs_enabled"`
	LogsEndpoint string `json:"logs_endpoint"`
	LogsAPIKey   string `json:"logs_api_key"`
	ForwardLogs  bool   `json:"forward_logs"`
}

// LoggerConfig contains logging configuration
type LoggerConfig struct {
	Level      string `json:"level" mapstructure:"level"`
	FilePath   string `json:"file_path" mapstructure:"file_path"`
	MaxSize    int64  `json:"max_size" mapstructure:"max_size"`       // Max size in MB before rotation
	MaxAge     int    `json:"max_age" mapstructure:"max_age"`         // Max age in days
	MaxBackups int    `json:"max_backups" mapstructure:"max_backups"` // Max number of backup files
	Compress   bool   `json:"compress" mapstructure:"compress"`       // Compress rotated files
	Type       string `json:"type" mapstructure:"type"`               // logger type: file, console, hybrid, newrelic
}
