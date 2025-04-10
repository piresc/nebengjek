package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

func InitConfig(appName string) *models.Config {
	// Load .env file if it exists
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// Create config from environment variables
	return loadConfigFromEnv()
}

func loadConfigFromEnv() *models.Config {
	configs := &models.Config{}

	// App config
	configs.App.Name = getEnv("APP_NAME", "")
	configs.App.Environment = getEnv("APP_ENV", "development")
	configs.App.Debug = getEnvAsBool("APP_DEBUG", true)
	configs.App.Version = getEnv("APP_VERSION", "1.0.0")

	// Server config
	configs.Server.Host = getEnv("SERVER_HOST", "0.0.0.0")
	configs.Server.Port = getEnvAsInt("SERVER_PORT", 8080)
	configs.Server.GRPCPort = getEnvAsInt("SERVER_GRPC_PORT", 9090)
	configs.Server.ReadTimeout = getEnvAsInt("SERVER_READ_TIMEOUT", 60)
	configs.Server.WriteTimeout = getEnvAsInt("SERVER_WRITE_TIMEOUT", 60)
	configs.Server.ShutdownTimeout = getEnvAsInt("SERVER_SHUTDOWN_TIMEOUT", 30)

	// Database config
	configs.Database.Driver = getEnv("DB_DRIVER", "postgres")
	configs.Database.Host = getEnv("DB_HOST", "localhost")
	configs.Database.Port = getEnvAsInt("DB_PORT", 5432)
	configs.Database.Username = getEnv("DB_USERNAME", "postgres")
	configs.Database.Password = getEnv("DB_PASSWORD", "postgres")
	configs.Database.Database = getEnv("DB_DATABASE", "nebengjek")
	configs.Database.SSLMode = getEnv("DB_SSL_MODE", "disable")
	configs.Database.MaxConns = getEnvAsInt("DB_MAX_CONNS", 100)
	configs.Database.IdleConns = getEnvAsInt("DB_IDLE_CONNS", 10)

	// Redis config
	configs.Redis.Host = getEnv("REDIS_HOST", "localhost")
	configs.Redis.Port = getEnvAsInt("REDIS_PORT", 6379)
	configs.Redis.Password = getEnv("REDIS_PASSWORD", "")
	configs.Redis.DB = getEnvAsInt("REDIS_DB", 0)
	configs.Redis.PoolSize = getEnvAsInt("REDIS_POOL_SIZE", 10)

	// NATS config
	configs.NATS.URL = getEnv("NATS_URL", "nats://localhost:4222")

	// JWT config
	configs.JWT.Secret = getEnv("JWT_SECRET", "your-secret-key-here")
	configs.JWT.Expiration = getEnvAsInt("JWT_EXPIRATION", 60)
	configs.JWT.Issuer = getEnv("JWT_ISSUER", "nebengjek")

	return configs
}

// Helper functions to get environment variables with different types
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
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

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
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
