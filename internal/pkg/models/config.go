package models

// Config represents application configuration
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	NSQ      NSQConfig      `mapstructure:"nsq"`
	JWT      JWTConfig      `mapstructure:"jwt"`
}

// AppConfig contains application-specific configuration
type AppConfig struct {
	Name        string `mapstructure:"name"`
	Environment string `mapstructure:"environment"`
	Debug       bool   `mapstructure:"debug"`
	Version     string `mapstructure:"version"`
}

// ServerConfig contains HTTP/gRPC server configuration
type ServerConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	GRPCPort        int    `mapstructure:"grpc_port"`
	ReadTimeout     int    `mapstructure:"read_timeout"`
	WriteTimeout    int    `mapstructure:"write_timeout"`
	ShutdownTimeout int    `mapstructure:"shutdown_timeout"`
}

// DatabaseConfig contains database connection configuration
type DatabaseConfig struct {
	Driver    string `mapstructure:"driver"`
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	Username  string `mapstructure:"username"`
	Password  string `mapstructure:"password"`
	Database  string `mapstructure:"database"`
	SSLMode   string `mapstructure:"ssl_mode"`
	MaxConns  int    `mapstructure:"max_conns"`
	IdleConns int    `mapstructure:"idle_conns"`
}

// RedisConfig contains Redis connection configuration
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

// NSQConfig contains NSQ connection configuration
type NSQConfig struct {
	LookupAddresses []string `mapstructure:"lookup_addresses"`
	ProducerAddress string   `mapstructure:"producer_address"`
	ConsumerAddress string   `mapstructure:"consumer_address"`
}

// JWTConfig contains JWT authentication configuration
type JWTConfig struct {
	Secret     string `mapstructure:"secret"`
	Expiration int    `mapstructure:"expiration"` // in minutes
	Issuer     string `mapstructure:"issuer"`
}
