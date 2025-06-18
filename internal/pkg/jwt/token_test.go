package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestConfig() *models.Config {
	return &models.Config{
		JWT: models.JWTConfig{
			Secret:     "test-secret-key-for-jwt-signing",
			Expiration: 60, // 60 minutes
			Issuer:     "nebengjek-test",
		},
	}
}

func TestGenerateToken(t *testing.T) {
	tests := []struct {
		name        string
		userID      uuid.UUID
		msisdn      string
		role        string
		config      *models.Config
		expectError bool
	}{
		{
			name:        "Valid token generation",
			userID:      uuid.New(),
			msisdn:      "+6281234567890",
			role:        "driver",
			config:      getTestConfig(),
			expectError: false,
		},
		{
			name:        "Valid token generation for passenger",
			userID:      uuid.New(),
			msisdn:      "+6289876543210",
			role:        "passenger",
			config:      getTestConfig(),
			expectError: false,
		},
		{
			name:        "Empty MSISDN",
			userID:      uuid.New(),
			msisdn:      "",
			role:        "driver",
			config:      getTestConfig(),
			expectError: false, // Should still generate token
		},
		{
			name:        "Empty role",
			userID:      uuid.New(),
			msisdn:      "+6281234567890",
			role:        "",
			config:      getTestConfig(),
			expectError: false, // Should still generate token
		},
		{
			name:   "Zero UUID",
			userID: uuid.UUID{},
			msisdn: "+6281234567890",
			role:   "driver",
			config: getTestConfig(),
			expectError: false, // Should still generate token
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenString, expiresAt, err := GenerateToken(tt.userID, tt.msisdn, tt.role, tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, tokenString)
				assert.Zero(t, expiresAt)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, tokenString)
				assert.Greater(t, expiresAt, time.Now().Unix())

				// Verify token structure
				token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
					return []byte(tt.config.JWT.Secret), nil
				})
				require.NoError(t, err)
				require.True(t, token.Valid)

				// Verify claims
				claims, ok := token.Claims.(jwt.MapClaims)
				require.True(t, ok)

				// Check user_id claim
				userIDClaim, exists := claims["user_id"]
				assert.True(t, exists)
				assert.Equal(t, tt.userID.String(), userIDClaim)

				// Check msisdn claim
				msisdnClaim, exists := claims["msisdn"]
				assert.True(t, exists)
				assert.Equal(t, tt.msisdn, msisdnClaim)

				// Check role claim
				roleClaim, exists := claims["role"]
				assert.True(t, exists)
				assert.Equal(t, tt.role, roleClaim)

				// Check issuer claim
				issuerClaim, exists := claims["iss"]
				assert.True(t, exists)
				assert.Equal(t, tt.config.JWT.Issuer, issuerClaim)

				// Check expiration claim
				expClaim, exists := claims["exp"]
				assert.True(t, exists)
				assert.Equal(t, float64(expiresAt), expClaim)
			}
		})
	}
}

func TestGenerateToken_ExpirationTime(t *testing.T) {
	config := getTestConfig()
	config.JWT.Expiration = 30 // 30 minutes

	userID := uuid.New()
	msisdn := "+6281234567890"
	role := "driver"

	beforeGeneration := time.Now()
	tokenString, expiresAt, err := GenerateToken(userID, msisdn, role, config)
	afterGeneration := time.Now()

	assert.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	// Verify expiration time is approximately 30 minutes from now
	expectedExpiration := beforeGeneration.Add(30 * time.Minute).Unix()
	expectedExpirationMax := afterGeneration.Add(30 * time.Minute).Unix()

	assert.GreaterOrEqual(t, expiresAt, expectedExpiration)
	assert.LessOrEqual(t, expiresAt, expectedExpirationMax)
}

func TestValidateToken(t *testing.T) {
	config := getTestConfig()
	userID := uuid.New()
	msisdn := "+6281234567890"
	role := "driver"

	// Generate a valid token
	validToken, _, err := GenerateToken(userID, msisdn, role, config)
	require.NoError(t, err)

	tests := []struct {
		name        string
		tokenString string
		secret      string
		expectError bool
		setupToken  func() string
	}{
		{
			name:        "Valid token",
			tokenString: validToken,
			secret:      config.JWT.Secret,
			expectError: false,
		},
		{
			name:        "Invalid secret",
			tokenString: validToken,
			secret:      "wrong-secret",
			expectError: true,
		},
		{
			name:        "Malformed token",
			tokenString: "invalid.token.string",
			secret:      config.JWT.Secret,
			expectError: true,
		},
		{
			name:        "Empty token",
			tokenString: "",
			secret:      config.JWT.Secret,
			expectError: true,
		},
		{
			name:        "Token without signature",
			tokenString: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMTIzIiwibXNpc2RuIjoiKzYyODEyMzQ1Njc4OTAiLCJyb2xlIjoiZHJpdmVyIn0",
			secret:      config.JWT.Secret,
			expectError: true,
		},
		{
			name: "Expired token",
			setupToken: func() string {
				// Create an expired token
				expiredConfig := *config
				expiredConfig.JWT.Expiration = -1 // Expired 1 minute ago
				token, _, _ := GenerateToken(userID, msisdn, role, &expiredConfig)
				return token
			},
			secret:      config.JWT.Secret,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenToTest := tt.tokenString
			if tt.setupToken != nil {
				tokenToTest = tt.setupToken()
			}

			claims, err := ValidateToken(tokenToTest, tt.secret)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)

				// Verify claims content
				claimsMap := *claims
				assert.Equal(t, userID.String(), claimsMap["user_id"])
				assert.Equal(t, msisdn, claimsMap["msisdn"])
				assert.Equal(t, role, claimsMap["role"])
				assert.Equal(t, config.JWT.Issuer, claimsMap["iss"])
			}
		})
	}
}

func TestValidateToken_ClaimsExtraction(t *testing.T) {
	config := getTestConfig()
	userID := uuid.New()
	msisdn := "+6289876543210"
	role := "passenger"

	// Generate token
	tokenString, expiresAt, err := GenerateToken(userID, msisdn, role, config)
	require.NoError(t, err)

	// Validate token
	claims, err := ValidateToken(tokenString, config.JWT.Secret)
	require.NoError(t, err)
	require.NotNil(t, claims)

	claimsMap := *claims

	// Test all claim extractions
	assert.Equal(t, userID.String(), claimsMap["user_id"])
	assert.Equal(t, msisdn, claimsMap["msisdn"])
	assert.Equal(t, role, claimsMap["role"])
	assert.Equal(t, config.JWT.Issuer, claimsMap["iss"])
	assert.Equal(t, float64(expiresAt), claimsMap["exp"])

	// Test type assertions
	userIDStr, ok := claimsMap["user_id"].(string)
	assert.True(t, ok)
	assert.Equal(t, userID.String(), userIDStr)

	msisdnStr, ok := claimsMap["msisdn"].(string)
	assert.True(t, ok)
	assert.Equal(t, msisdn, msisdnStr)

	roleStr, ok := claimsMap["role"].(string)
	assert.True(t, ok)
	assert.Equal(t, role, roleStr)
}

func TestClaims_Struct(t *testing.T) {
	// Test the Claims struct (though it's not used in the current implementation)
	userID := uuid.New()
	msisdn := "+6281234567890"
	role := "driver"

	claims := Claims{
		UserID: userID,
		MSISDN: msisdn,
		Role:   role,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
			Issuer:    "test-issuer",
		},
	}

	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, msisdn, claims.MSISDN)
	assert.Equal(t, role, claims.Role)
	assert.Equal(t, "test-issuer", claims.Issuer)
}

func TestGenerateToken_DifferentConfigurations(t *testing.T) {
	tests := []struct {
		name       string
		config     *models.Config
		expiration int
	}{
		{
			name: "Short expiration",
			config: &models.Config{
				JWT: models.JWTConfig{
					Secret:     "short-secret",
					Expiration: 5, // 5 minutes
					Issuer:     "short-issuer",
				},
			},
			expiration: 5,
		},
		{
			name: "Long expiration",
			config: &models.Config{
				JWT: models.JWTConfig{
					Secret:     "long-secret-key-for-testing",
					Expiration: 1440, // 24 hours
					Issuer:     "long-issuer",
				},
			},
			expiration: 1440,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := uuid.New()
			msisdn := "+6281234567890"
			role := "driver"

			beforeGeneration := time.Now()
			tokenString, expiresAt, err := GenerateToken(userID, msisdn, role, tt.config)
			afterGeneration := time.Now()

			assert.NoError(t, err)
			assert.NotEmpty(t, tokenString)

			// Verify expiration time
			expectedMin := beforeGeneration.Add(time.Duration(tt.expiration) * time.Minute).Unix()
			expectedMax := afterGeneration.Add(time.Duration(tt.expiration) * time.Minute).Unix()

			assert.GreaterOrEqual(t, expiresAt, expectedMin)
			assert.LessOrEqual(t, expiresAt, expectedMax)

			// Validate token with correct secret
			claims, err := ValidateToken(tokenString, tt.config.JWT.Secret)
			assert.NoError(t, err)
			assert.NotNil(t, claims)

			// Verify issuer
			claimsMap := *claims
			assert.Equal(t, tt.config.JWT.Issuer, claimsMap["iss"])
		})
	}
}

func BenchmarkGenerateToken(b *testing.B) {
	config := getTestConfig()
	userID := uuid.New()
	msisdn := "+6281234567890"
	role := "driver"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = GenerateToken(userID, msisdn, role, config)
	}
}

func BenchmarkValidateToken(b *testing.B) {
	config := getTestConfig()
	userID := uuid.New()
	msisdn := "+6281234567890"
	role := "driver"

	// Generate token once
	tokenString, _, err := GenerateToken(userID, msisdn, role, config)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ValidateToken(tokenString, config.JWT.Secret)
	}
}