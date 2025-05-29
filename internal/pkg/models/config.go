package models

// Config represents application configuration
type Config struct {
	App      AppConfig
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	NATS     NATSConfig
	JWT      JWTConfig
	Pricing  PricingConfig
	Services ServicesConfig
	Match    MatchConfig
	Rides    RidesConfig
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

type PricingConfig struct {
	RatePerKm     float64 `json:"rate_per_km"`
	Currency      string  `json:"currency"`
	BaseFare      float64 `json:"base_fare"`
	PerKmRate     float64 `json:"per_km_rate"`
	PerMinuteRate float64 `json:"per_minute_rate"`
	SurgeFactor   float64 `json:"surge_factor"`
}

// MatchConfig contains match service specific configuration
type MatchConfig struct {
	SearchRadiusKm float64 `json:"search_radius_km"` // Radius in kilometers for matching users
}

// RidesConfig contains rides service specific configuration
type RidesConfig struct {
	MinDistanceKm float64 `json:"min_distance_km"` // Minimum distance in kilometers for billing
}
