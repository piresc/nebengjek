package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

func InitConfig(configPath string) *models.Config {
	local := GetEnv("APP_ENV", "local")
	if local == "local" {
		// Load config from file
		err := godotenv.Load(configPath)
		if err != nil {
			log.Println("error loading config from file", err)
		}
	}
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

	// Services config
	configs.Services.MatchServiceURL = GetEnv("MATCH_SERVICE_URL", "http://localhost:9993")
	configs.Services.RidesServiceURL = GetEnv("RIDES_SERVICE_URL", "http://localhost:9992")
	configs.Services.LocationServiceURL = GetEnv("LOCATION_SERVICE_URL", "http://localhost:9991")

	// Match config
	configs.Match.SearchRadiusKm = GetEnvAsFloat("MATCH_SEARCH_RADIUS_KM", 1.0)

	// Rides config
	configs.Rides.MinDistanceKm = GetEnvAsFloat("RIDES_MIN_DISTANCE_KM", 1.0)

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

func GetEnvAsFloat(key string, defaultValue float64) float64 {
	valueStr := GetEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		log.Printf("Warning: Invalid float value for %s, using default: %v", key, defaultValue)
		return defaultValue
	}

	return value
}
