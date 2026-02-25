package auth

import (
	"net/http"

	"github.com/golang-jwt/jwt/v4"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/models/core"
)

// AuthMiddleware handles authentication and authorization
type AuthMiddleware struct {
	config *core.Config
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(config *core.Config) *AuthMiddleware {
	return &AuthMiddleware{config: config}
}

// APIKeyHandler handles API key authentication for service-to-service communication
func (m *AuthMiddleware) APIKeyHandler(allowedService string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			apiKey := c.Request().Header.Get("X-API-Key")
			if apiKey == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "API key required")
			}

			expectedKey := m.getExpectedKey(allowedService)
			if expectedKey == "" || expectedKey != apiKey {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid API key")
			}

			c.Set("api_service", allowedService)
			return next(c)
		}
	}
}

// getExpectedKey returns the expected API key for a given service
func (m *AuthMiddleware) getExpectedKey(service string) string {
	switch service {
	case "users-service":
		return m.config.APIKey.UserService
	case "match-service":
		return m.config.APIKey.MatchService
	case "rides-service":
		return m.config.APIKey.RidesService
	case "location-service":
		return m.config.APIKey.LocationService
	case "gateway-service":
		return m.config.APIKey.GatewayService
	case "notification-service":
		return m.config.APIKey.NotificationService
	default:
		return ""
	}
}

// JWTHandler handles JWT authentication
func (m *AuthMiddleware) JWTHandler() echo.MiddlewareFunc {
	return echojwt.WithConfig(echojwt.Config{
		SigningKey: []byte(m.config.JWT.Secret),
		SuccessHandler: func(c echo.Context) {
			token, ok := c.Get("user").(*jwt.Token)
			if !ok || !token.Valid {
				return
			}
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if userID, exists := claims["user_id"]; exists {
					c.Set("user_id", userID)
				}
				if role, exists := claims["role"]; exists {
					c.Set("role", role)
				}
			}
		},
	})
}