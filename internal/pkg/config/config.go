package config

import (
	"log"
	"os"
	"strconv"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

func InitConfig() *models.Config {
	// Create config from environment variables
	return loadConfigFromEnv()
}

func loadConfigFromEnv() *models.Config {
	configs := &models.Config{}

	// App config
	configs.App.Name = GetEnv("APP_NAME", "")
	configs.App.Environment = GetEnv("APP_ENV", "")
	configs.App.Debug = GetEnvAsBool("APP_DEBUG", true)
	configs.App.Version = GetEnv("APP_VERSION", "")

	// Server config
	configs.Server.Host = GetEnv("SERVER_HOST", "")
	configs.Server.Port = GetEnvAsInt("SERVER_PORT", 0)
	configs.Server.GRPCPort = GetEnvAsInt("SERVER_GRPC_PORT", 0)
	configs.Server.ReadTimeout = GetEnvAsInt("SERVER_READ_TIMEOUT", 0)
	configs.Server.WriteTimeout = GetEnvAsInt("SERVER_WRITE_TIMEOUT", 0)
	configs.Server.ShutdownTimeout = GetEnvAsInt("SERVER_SHUTDOWN_TIMEOUT", 0)

	// Database config
	configs.Database.Driver = GetEnv("DB_DRIVER", "")
	configs.Database.Host = GetEnv("DB_HOST", "")
	configs.Database.Port = GetEnvAsInt("DB_PORT", 0)
	configs.Database.Username = GetEnv("DB_USERNAME", "")
	configs.Database.Password = GetEnv("DB_PASSWORD", "")
	configs.Database.Database = GetEnv("DB_DATABASE", "")
	configs.Database.SSLMode = GetEnv("DB_SSL_MODE", "")
	configs.Database.MaxConns = GetEnvAsInt("DB_MAX_CONNS", 0)
	configs.Database.IdleConns = GetEnvAsInt("DB_IDLE_CONNS", 0)

	// Redis config
	configs.Redis.Host = GetEnv("REDIS_HOST", "")
	configs.Redis.Port = GetEnvAsInt("REDIS_PORT", 0)
	configs.Redis.Password = GetEnv("REDIS_PASSWORD", "")
	configs.Redis.DB = GetEnvAsInt("REDIS_DB", 0)
	configs.Redis.PoolSize = GetEnvAsInt("REDIS_POOL_SIZE", 0)

	// NATS config
	configs.NATS.URL = GetEnv("NATS_URL", "")

	// JWT config
	configs.JWT.Secret = GetEnv("JWT_SECRET", "")
	configs.JWT.Expiration = GetEnvAsInt("JWT_EXPIRATION", 0)
	configs.JWT.Issuer = GetEnv("JWT_ISSUER", "")

	return configs
}

// Helper functions to get environment variables with different types
func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func GetEnvAsInt(key string, defaultValue int) int {
	valueStr := GetEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Warning: Invalid integer value for %s, using default: %d", key, defaultValue)
		return defaultValue
	}

	return value
}

func GetEnvAsBool(key string, defaultValue bool) bool {
	valueStr := GetEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		log.Printf("Warning: Invalid boolean value for %s, using default: %v", key, defaultValue)
		return defaultValue
	}

	return value
}
