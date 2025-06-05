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

	// NewRelic config
	configs.NewRelic.LicenseKey = GetEnv("NEW_RELIC_LICENSE_KEY", "")
	configs.NewRelic.AppName = GetEnv("NEW_RELIC_APP_NAME", "")
	configs.NewRelic.Enabled = GetEnvAsBool("NEW_RELIC_ENABLED", false)
	configs.NewRelic.LogsEnabled = GetEnvAsBool("NEW_RELIC_LOGS_ENABLED", false)
	configs.NewRelic.LogsEndpoint = GetEnv("NEW_RELIC_LOGS_ENDPOINT", "")
	configs.NewRelic.LogsAPIKey = GetEnv("NEW_RELIC_LOGS_API_KEY", "")
	configs.NewRelic.ForwardLogs = GetEnvAsBool("NEW_RELIC_FORWARD_LOGS", false)

	// Logger config
	configs.Logger.Level = GetEnv("LOG_LEVEL", "info")
	configs.Logger.FilePath = GetEnv("LOG_FILE_PATH", "logs/nebengjek.log")
	configs.Logger.MaxSize = GetEnvAsInt64("LOG_MAX_SIZE", 100)
	configs.Logger.MaxAge = GetEnvAsInt("LOG_MAX_AGE", 7)
	configs.Logger.MaxBackups = GetEnvAsInt("LOG_MAX_BACKUPS", 3)
	configs.Logger.Compress = GetEnvAsBool("LOG_COMPRESS", true)
	configs.Logger.Type = GetEnv("LOG_TYPE", "file")

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

func GetEnvAsInt64(key string, defaultValue int64) int64 {
	valueStr := GetEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		log.Printf("Warning: Invalid int64 value for %s, using default: %d", key, defaultValue)
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
