package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/utils"
)

const (
	APIKeyHeader = "X-API-Key"
)

// APIKeyMiddleware provides API key validation middleware
type APIKeyMiddleware struct {
	config *models.APIKeyConfig
}

// NewAPIKeyMiddleware creates a new API key middleware instance
func NewAPIKeyMiddleware(config *models.APIKeyConfig) *APIKeyMiddleware {
	return &APIKeyMiddleware{
		config: config,
	}
}

// ValidateAPIKey middleware validates the API key for service-to-service communication
func (m *APIKeyMiddleware) ValidateAPIKey(allowedServices ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			apiKey := c.Request().Header.Get(APIKeyHeader)
			if apiKey == "" {
				logger.Warn("API key missing in request",
					logger.String("path", c.Request().URL.Path),
					logger.String("method", c.Request().Method),
					logger.String("remote_addr", c.Request().RemoteAddr))
				return utils.ErrorResponseHandler(c, http.StatusUnauthorized, "API key is required")
			}

			// Get service API keys mapping
			serviceAPIKeys := m.getServiceAPIKeys()

			// Check if the API key belongs to any of the allowed services
			validKey := false
			validService := ""
			for _, service := range allowedServices {
				if serviceKey, exists := serviceAPIKeys[service]; exists && serviceKey != "" {
					if strings.EqualFold(apiKey, serviceKey) {
						validKey = true
						validService = service
						break
					}
				}
			}

			if !validKey {
				logger.Warn("Invalid API key provided",
					logger.String("path", c.Request().URL.Path),
					logger.String("method", c.Request().Method),
					logger.String("remote_addr", c.Request().RemoteAddr),
					logger.Strings("allowed_services", allowedServices))
				return utils.ErrorResponseHandler(c, http.StatusUnauthorized, "Invalid API key")
			}

			// Set the authenticated service in context
			c.Set("authenticated_service", validService)

			logger.Debug("API key validation successful",
				logger.String("service", validService),
				logger.String("path", c.Request().URL.Path),
				logger.String("method", c.Request().Method))

			return next(c)
		}
	}
}

// getServiceAPIKeys returns the mapping of service names to their API keys
func (m *APIKeyMiddleware) getServiceAPIKeys() map[string]string {
	return map[string]string{
		"user-service":     m.config.UserService,
		"match-service":    m.config.MatchService,
		"rides-service":    m.config.RidesService,
		"location-service": m.config.LocationService,
	}
}

// ValidateAPIKeyForServices is a convenience function that creates middleware for specific services
func ValidateAPIKeyForServices(config *models.APIKeyConfig, allowedServices ...string) echo.MiddlewareFunc {
	middleware := NewAPIKeyMiddleware(config)
	return middleware.ValidateAPIKey(allowedServices...)
}
