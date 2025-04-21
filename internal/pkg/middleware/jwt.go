package middleware

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	jwtpkg "github.com/piresc/nebengjek/internal/pkg/jwt"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/utils"
)

// JWTAuthMiddleware creates a middleware for JWT authentication
func JWTAuthMiddleware(config models.JWTConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get the Authorization header
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return utils.UnauthorizedResponse(c, "Authorization header is required")
			}

			// Check if the Authorization header has the correct format
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				return utils.UnauthorizedResponse(c, "Invalid authorization format")
			}

			// Extract the token
			tokenString := parts[1]

			// Validate the token using our JWT package
			claims, err := jwtpkg.ValidateToken(tokenString, config.Secret)
			if err != nil {
				return utils.UnauthorizedResponse(c, "Invalid token")
			}

			// Extract user ID and role from claims
			userIDStr, ok := (*claims)["user_id"]
			if !ok {
				return utils.UnauthorizedResponse(c, "Invalid token: missing user_id claim")
			}

			role, ok := (*claims)["role"]
			if !ok {
				return utils.UnauthorizedResponse(c, "Invalid token: missing role claim")
			}

			// Parse the UUID
			userID, err := uuid.Parse(fmt.Sprintf("%v", userIDStr))
			if err != nil {
				return utils.UnauthorizedResponse(c, "Invalid token: user_id is not a valid UUID")
			}

			// Set the user ID and role in the context
			c.Set("user_id", userID)
			c.Set("user_role", role)

			return next(c)
		}
	}
}
