package auth

import (
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v4"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/pkg/models/core"
	"github.com/piresc/nebengjek/internal/pkg/tracing"
)

// AuthMiddleware handles authentication and authorization
type AuthMiddleware struct {
	config *core.Config
	tracer tracing.Tracer
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(config *core.Config, tracer tracing.Tracer) *AuthMiddleware {
	return &AuthMiddleware{
		config: config,
		tracer: tracer,
	}
}

// APIKeyHandler handles API key authentication for service-to-service communication
func (m *AuthMiddleware) APIKeyHandler(allowedService string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if m.config == nil || !m.config.Auth.Enabled {
				return next(c)
			}

			apiKey := c.Request().Header.Get("X-API-Key")
			if apiKey == "" {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": "API key required",
				})
			}

			// Validate API key
			validKey := false
			for _, key := range m.config.Auth.APIKeys {
				if key.Key == apiKey && key.Service == allowedService {
					validKey = true
					break
				}
			}

			if !validKey {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": "Invalid API key",
				})
			}

			// Add service info to context
			c.Set("service_name", allowedService)
			return next(c)
		}
	}
}

// JWTHandler handles JWT authentication
func (m *AuthMiddleware) JWTHandler() echo.MiddlewareFunc {
	config := echojwt.Config{
		SigningKey:  []byte(m.config.Auth.JWTSecret),
		TokenLookup: "header:Authorization:Bearer ",
		ErrorHandler: func(c echo.Context, err error) error {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"error": "Invalid or expired token",
			})
		},
		SuccessHandler: func(c echo.Context) {
			token := c.Get("user").(*jwt.Token)
			claims := token.Claims.(jwt.MapClaims)
			
			// Extract user information
			userID, ok := claims["sub"].(string)
			if !ok {
				return
			}
			
			// Add user info to context
			c.Set("user_id", userID)
			c.Set("user_claims", claims)
		},
	}

	return echojwt.WithConfig(config)
}

// ExtractUserID extracts user ID from context
func ExtractUserID(c echo.Context) string {
	if userID, ok := c.Get("user_id").(string); ok {
		return userID
	}
	return ""
}

// ExtractServiceName extracts service name from context
func ExtractServiceName(c echo.Context) string {
	if serviceName, ok := c.Get("service_name").(string); ok {
		return serviceName
	}
	return ""
}

// ValidateJWTToken validates a JWT token and returns claims
func ValidateJWTToken(tokenString string, secret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}