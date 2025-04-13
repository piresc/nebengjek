package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

func InitConfig(envPath string) *models.Config {
	// Load .env file if it exists
	err := godotenv.Load(envPath)
	if err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// Create config from environment variables
	return loadConfigFromEnv()
}

func loadConfigFromEnv() *models.Config {
	configs := &models.Config{}

	// App config
	configs.App.Name = GetEnv("APP_NAME", "")
	configs.App.Environment = GetEnv("APP_ENV", "development")
	configs.App.Debug = GetEnvAsBool("APP_DEBUG", true)
	configs.App.Version = GetEnv("APP_VERSION", "1.0.0")

	// Server config
	configs.Server.Host = GetEnv("SERVER_HOST", "0.0.0.0")
	configs.Server.Port = GetEnvAsInt("SERVER_PORT", 8080)
	configs.Server.GRPCPort = GetEnvAsInt("SERVER_GRPC_PORT", 9090)
	configs.Server.ReadTimeout = GetEnvAsInt("SERVER_READ_TIMEOUT", 60)
	configs.Server.WriteTimeout = GetEnvAsInt("SERVER_WRITE_TIMEOUT", 60)
	configs.Server.ShutdownTimeout = GetEnvAsInt("SERVER_SHUTDOWN_TIMEOUT", 30)

	// Database config
	configs.Database.Driver = GetEnv("DB_DRIVER", "postgres")
	configs.Database.Host = GetEnv("DB_HOST", "localhost")
	configs.Database.Port = GetEnvAsInt("DB_PORT", 5432)
	configs.Database.Username = GetEnv("DB_USERNAME", "postgres")
	configs.Database.Password = GetEnv("DB_PASSWORD", "postgres")
	configs.Database.Database = GetEnv("DB_DATABASE", "nebengjek")
	configs.Database.SSLMode = GetEnv("DB_SSL_MODE", "disable")
	configs.Database.MaxConns = GetEnvAsInt("DB_MAX_CONNS", 100)
	configs.Database.IdleConns = GetEnvAsInt("DB_IDLE_CONNS", 10)

	// Redis config
	configs.Redis.Host = GetEnv("REDIS_HOST", "localhost")
	configs.Redis.Port = GetEnvAsInt("REDIS_PORT", 6379)
	configs.Redis.Password = GetEnv("REDIS_PASSWORD", "")
	configs.Redis.DB = GetEnvAsInt("REDIS_DB", 0)
	configs.Redis.PoolSize = GetEnvAsInt("REDIS_POOL_SIZE", 10)

	// NATS config
	configs.NATS.URL = GetEnv("NATS_URL", "nats://localhost:4222")

	// JWT config
	configs.JWT.Secret = GetEnv("JWT_SECRET", "your-secret-key-here")
	configs.JWT.Expiration = GetEnvAsInt("JWT_EXPIRATION", 60)
	configs.JWT.Issuer = GetEnv("JWT_ISSUER", "nebengjek")

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
