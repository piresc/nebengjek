package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/utils"
)

// GenerateOTP generates a new OTP for the given MSISDN
func (s *UserUC) GenerateOTP(ctx context.Context, msisdn string) error {
	// Validate MSISDN format and check if it's a Telkomsel number
	isValid, formattedMSISDN, err := utils.ValidateMSISDN(msisdn)
	if err != nil || !isValid {
		return fmt.Errorf("invalid MSISDN format or not a Telkomsel number")
	}

	// Generate dummy OTP using the last 4 digits of the MSISDN
	code := utils.GenerateDummyOTP(formattedMSISDN)

	// Create OTP record
	otp := &models.OTP{
		ID:         uuid.New().String(),
		MSISDN:     formattedMSISDN,
		Code:       code,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(5 * time.Minute), // OTP expires in 5 minutes
		IsVerified: false,
	}

	// Save OTP to database
	if err := s.repo.CreateOTP(ctx, otp); err != nil {
		return fmt.Errorf("failed to create OTP: %w", err)
	}

	// In a real implementation, we would integrate with Telkomsel's SMS API
	// For now, we'll just log it
	fmt.Printf("OTP for %s: %s\n", formattedMSISDN, code)

	return nil
}

// VerifyOTP verifies the OTP for the given MSISDN
func (s *UserUC) VerifyOTP(ctx context.Context, msisdn, code string) (*models.AuthResponse, error) {
	// Validate MSISDN format
	isValid, formattedMSISDN, err := utils.ValidateMSISDN(msisdn)
	if err != nil || !isValid {
		return nil, fmt.Errorf("invalid MSISDN format or not a Telkomsel number")
	}

	// Get OTP from database
	otp, err := s.repo.GetOTP(ctx, formattedMSISDN, code)
	if err != nil {
		return nil, fmt.Errorf("invalid OTP: %w", err)
	}

	// Check if OTP is expired
	if time.Now().After(otp.ExpiresAt) {
		return nil, fmt.Errorf("OTP expired")
	}

	// Check if OTP is already verified
	if otp.IsVerified {
		return nil, fmt.Errorf("OTP already used")
	}

	// Mark OTP as verified
	if err := s.repo.MarkOTPVerified(ctx, otp.ID); err != nil {
		return nil, fmt.Errorf("failed to mark OTP as verified: %w", err)
	}

	// Get or create user
	user, err := s.repo.GetUserByMSISDN(ctx, formattedMSISDN)
	if err != nil {
		// User doesn't exist, create a new one
		user = &models.User{
			ID:        uuid.New().String(),
			MSISDN:    formattedMSISDN,
			Role:      "passenger", // Default role is passenger
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			IsActive:  true,
			Rating:    0,
		}

		if err := s.repo.CreateUser(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	// Generate JWT token
	token, expiresAt, err := generateJWTToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Return auth response
	return &models.AuthResponse{
		Token:     token,
		UserID:    user.ID,
		Role:      user.Role,
		ExpiresAt: expiresAt,
	}, nil
}

// Helper functions

// generateJWTToken generates a JWT token for the given user
func generateJWTToken(user *models.User) (string, int64, error) {
	// Set token expiration time (24 hours)
	expirationTime := time.Now().Add(24 * time.Hour)
	expiresAt := expirationTime.Unix()

	// Create claims
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"msisdn":  user.MSISDN,
		"role":    user.Role,
		"exp":     expiresAt,
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// TODO: Use a proper secret key from configuration
	tokenString, err := token.SignedString([]byte("your-secret-key"))
	if err != nil {
		return "", 0, err
	}

	return tokenString, expiresAt, nil
}
