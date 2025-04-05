package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// JWTAuthMiddleware creates a middleware for JWT authentication
func JWTAuthMiddleware(config models.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			return
		}

		// Check if the Authorization header has the correct format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			return
		}

		// Extract the token
		tokenString := parts[1]

		// Parse and validate the token
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(config.Secret), nil
		})

		// Handle token parsing errors
		if err != nil {
			var validationError *jwt.ValidationError
			if errors.As(err, &validationError) {
				if validationError.Errors&jwt.ValidationErrorExpired != 0 {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token has expired"})
					return
				}
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// Check if the token is valid
		if !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// Set the user ID and role in the context
		c.Set("user_id", claims.UserID)
		c.Set("user_role", claims.Role)

		c.Next()
	}
}

// Claims represents the JWT claims
type Claims struct {
	jwt.RegisteredClaims
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// GenerateToken generates a new JWT token
func GenerateToken(userID, role string, config models.JWTConfig) (string, error) {
	// Set expiration time
	expiration := time.Now().Add(time.Duration(config.Expiration) * time.Minute)

	// Create claims
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiration),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    config.Issuer,
		},
		UserID: userID,
		Role:   role,
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret
	tokenString, err := token.SignedString([]byte(config.Secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenString string, config models.JWTConfig) (*Claims, error) {
	// Parse and validate the token
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.Secret), nil
	})

	// Handle token parsing errors
	if err != nil {
		return nil, err
	}

	// Check if the token is valid
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
