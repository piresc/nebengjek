package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/config"
	"github.com/piresc/nebengjek/internal/utils"
)

const (
	APIKeyHeader = "X-API-Key"
)

// ServiceAPIKeys stores the mapping of service names to their API keys
var ServiceAPIKeys = map[string]string{
	"user-service":     config.GetEnv("USER_SERVICE_API_KEY", ""),
	"match-service":    config.GetEnv("MATCH_SERVICE_API_KEY", ""),
	"billing-service":  config.GetEnv("BILLING_SERVICE_API_KEY", ""),
	"location-service": config.GetEnv("LOCATION_SERVICE_API_KEY", ""),
	"trip-service":     config.GetEnv("TRIP_SERVICE_API_KEY", ""),
}

// ValidateAPIKey middleware validates the API key for service-to-service communication
func ValidateAPIKey(allowedServices ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			apiKey := c.Request().Header.Get(APIKeyHeader)
			if apiKey == "" {
				return utils.ErrorResponseHandler(c, http.StatusUnauthorized, "API key is required")
			}

			// Check if the API key belongs to any of the allowed services
			validKey := false
			for _, service := range allowedServices {
				if ServiceAPIKeys[service] != "" && strings.EqualFold(apiKey, ServiceAPIKeys[service]) {
					validKey = true
					break
				}
			}

			if !validKey {
				return utils.ErrorResponseHandler(c, http.StatusUnauthorized, "Invalid API key")
			}

			return next(c)
		}
	}
}
