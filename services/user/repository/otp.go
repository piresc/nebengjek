package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// CreateOTP creates a new OTP record in the database
func (r *UserRepo) CreateOTP(ctx context.Context, otp *models.OTP) error {
	query := `
		INSERT INTO otps (id, msisdn, code, created_at, expires_at, is_verified)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		otp.ID,
		otp.MSISDN,
		otp.Code,
		otp.CreatedAt,
		otp.ExpiresAt,
		otp.IsVerified,
	)

	if err != nil {
		return fmt.Errorf("failed to create OTP: %w", err)
	}

	return nil
}

// GetOTP retrieves an OTP record from the database
func (r *UserRepo) GetOTP(ctx context.Context, msisdn, code string) (*models.OTP, error) {
	query := `
		SELECT id, msisdn, code, created_at, expires_at, is_verified
		FROM otps
		WHERE msisdn = $1 AND code = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	var otp models.OTP
	err := r.db.QueryRowContext(ctx, query, msisdn, code).Scan(
		&otp.ID,
		&otp.MSISDN,
		&otp.Code,
		&otp.CreatedAt,
		&otp.ExpiresAt,
		&otp.IsVerified,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("OTP not found")
		}
		return nil, fmt.Errorf("failed to get OTP: %w", err)
	}

	return &otp, nil
}

// MarkOTPVerified marks an OTP as verified
func (r *UserRepo) MarkOTPVerified(ctx context.Context, id string) error {
	query := `
		UPDATE otps
		SET is_verified = true
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to mark OTP as verified: %w", err)
	}

	return nil
}

// GetUserByMSISDN retrieves a user by MSISDN
func (r *UserRepo) GetUserByMSISDN(ctx context.Context, msisdn string) (*models.User, error) {
	query := `
		SELECT id, msisdn, email, full_name, role, created_at, updated_at, is_active, rating
		FROM users
		WHERE msisdn = $1
	`

	var user models.User
	err := r.db.QueryRowContext(ctx, query, msisdn).Scan(
		&user.ID,
		&user.MSISDN,
		&user.FullName,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsActive,
		&user.Rating,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is a driver
	if user.Role == "driver" {
		// Get driver info
		driver, err := r.getDriverInfo(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		user.DriverInfo = driver
	}

	return &user, nil
}
