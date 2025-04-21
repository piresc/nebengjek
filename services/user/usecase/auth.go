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
func (u *UserUC) GenerateOTP(ctx context.Context, msisdn string) error {
	// Validate MSISDN format and check if it's a Telkomsel number
	isValid, formattedMSISDN, err := utils.ValidateMSISDN(msisdn)
	if err != nil || !isValid {
		return fmt.Errorf("invalid MSISDN format or not a Telkomsel number")
	}

	// Generate dummy OTP using the last 4 digits of the MSISDN
	code := utils.GenerateDummyOTP(formattedMSISDN)

	// Create OTP record
	otp := &models.OTP{
		ID:     uuid.New().String(),
		MSISDN: formattedMSISDN,
		Code:   code,
	}

	// Save OTP to database
	if err := u.userRepo.CreateOTP(ctx, otp); err != nil {
		return fmt.Errorf("failed to create OTP: %w", err)
	}

	// In a real implementation, we would integrate with Telkomsel's SMS API
	// For now, we'll just log it
	fmt.Printf("OTP for %s: %s\n", formattedMSISDN, code)

	return nil
}

// VerifyOTP verifies the OTP for the given MSISDN
func (u *UserUC) VerifyOTP(ctx context.Context, msisdn, code string) (*models.AuthResponse, error) {
	// Validate MSISDN format
	isValid, formattedMSISDN, err := utils.ValidateMSISDN(msisdn)
	if err != nil || !isValid {
		return nil, fmt.Errorf("invalid MSISDN format or not a Telkomsel number")
	}

	// Get OTP from database
	otp, err := u.userRepo.GetOTP(ctx, formattedMSISDN, code)
	if err != nil {
		return nil, fmt.Errorf("invalid OTP: %w", err)
	}
	if otp == nil {
		return nil, fmt.Errorf("OTP not found or expired")
	}
	if otp.Code != code {
		return nil, fmt.Errorf("invalid OTP code")
	}

	// Get or create user
	user, err := u.userRepo.GetUserByMSISDN(ctx, formattedMSISDN)
	if err != nil {
		// User doesn't exist, create a new one
		user = &models.User{
			MSISDN:    formattedMSISDN,
			Role:      "passenger", // Default role is passenger
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			IsActive:  true,
		}

		if err := u.userRepo.CreateUser(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	// Generate JWT token
	token, expiresAt, err := u.generateJWTToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Mark OTP as verified
	if err := u.userRepo.MarkOTPVerified(ctx, formattedMSISDN, code); err != nil {
		return nil, fmt.Errorf("failed to mark OTP as verified: %w", err)
	}

	// Return auth response
	return &models.AuthResponse{
		Token:     token,
		UserID:    user.ID.String(),
		Role:      user.Role,
		ExpiresAt: expiresAt,
	}, nil
}

// generateJWTToken generates a JWT token for the given user
func (u *UserUC) generateJWTToken(user *models.User) (string, int64, error) {
	// Set token expiration time
	expirationTime := time.Now().Add(time.Duration(u.cfg.JWT.Expiration) * time.Minute)
	expiresAt := expirationTime.Unix()

	// Create claims
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"msisdn":  user.MSISDN,
		"role":    user.Role,
		"exp":     expiresAt,
		"iss":     u.cfg.JWT.Issuer,
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with configured secret
	tokenString, err := token.SignedString([]byte(u.cfg.JWT.Secret))
	if err != nil {
		return "", 0, err
	}

	return tokenString, expiresAt, nil
}
