package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/internal/pkg/models/core"
)

const otpExpirationTime = 5 * time.Minute

// CreateOTP creates a new OTP record in Redis
func (r *UserRepo) CreateOTP(ctx context.Context, otp *core.OTP) error {
	// Convert OTP to JSON
	otpJSON, err := json.Marshal(otp)
	if err != nil {
		return fmt.Errorf("failed to marshal OTP: %w", err)
	}

	// Store in Redis with expiration using standardized key format
	key := fmt.Sprintf(database.KeyUserOTP, otp.MSISDN)
	if err := r.redisClient.Set(ctx, key, string(otpJSON), otpExpirationTime); err != nil {
		return fmt.Errorf("failed to store OTP in Redis: %w", err)
	}

	return nil
}

// GetOTP retrieves an OTP record from Redis
func (r *UserRepo) GetOTP(ctx context.Context, msisdn, code string) (*core.OTP, error) {
	key := fmt.Sprintf(database.KeyUserOTP, msisdn)
	otpJSON, err := r.redisClient.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("OTP not found or expired")
	}

	var otp core.OTP
	if err := json.Unmarshal([]byte(otpJSON), &otp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OTP: %w", err)
	}

	return &otp, nil
}

// MarkOTPVerified marks an OTP as verified and deletes it from Redis
func (r *UserRepo) MarkOTPVerified(ctx context.Context, msisdn string, code string) error {
	key := fmt.Sprintf(database.KeyUserOTP, msisdn)
	err := r.redisClient.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete OTP: %w", err)
	}
	return nil
}
