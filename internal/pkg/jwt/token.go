package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// Claims represents standard JWT claims plus custom fields
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	MSISDN string    `json:"msisdn"`
	Role   string    `json:"role"`
	jwt.StandardClaims
}

// GenerateToken generates a JWT token for the given user details
func GenerateToken(userID uuid.UUID, msisdn, role string, cfg *models.Config) (string, int64, error) {
	// Set token expiration time
	expirationTime := time.Now().Add(time.Duration(cfg.JWT.Expiration) * time.Minute)
	expiresAt := expirationTime.Unix()

	// Create claims
	claims := jwt.MapClaims{
		"user_id": userID,
		"msisdn":  msisdn,
		"role":    role,
		"exp":     expiresAt,
		"iss":     cfg.JWT.Issuer,
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with configured secret
	tokenString, err := token.SignedString([]byte(cfg.JWT.Secret))
	if err != nil {
		return "", 0, err
	}

	return tokenString, expiresAt, nil
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenString string, secret string) (*jwt.MapClaims, error) {
	// Parse token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return &claims, nil
	}

	return nil, err
}
