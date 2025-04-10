package repository

import (
	"context"
	"fmt"
)

// UpdateDriverAvailability updates a driver's availability status
func (r *UserRepo) UpdateDriverAvailability(ctx context.Context, driverID string, isAvailable bool) error {
	// Update driver availability
	result, err := r.db.ExecContext(ctx, `
		UPDATE drivers
		SET is_available = $1
		WHERE user_id = $2
	`, isAvailable, driverID)
	if err != nil {
		return fmt.Errorf("failed to update driver availability: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("driver not found")
	}

	return nil
}

// VerifyDriver marks a driver as verified
func (r *UserRepo) VerifyDriver(ctx context.Context, driverID string) error {
	// Begin transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if driver exists
	var exists bool
	err = tx.QueryRowxContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM users u
			JOIN drivers d ON u.id = d.user_id
			WHERE u.id = $1 AND u.role = 'driver'
		)
	`, driverID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check driver existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("driver not found")
	}

	// Update driver verification status
	result, err := tx.ExecContext(ctx, `
		UPDATE drivers
		SET verified = true, verified_at = NOW()
		WHERE user_id = $1
	`, driverID)
	if err != nil {
		return fmt.Errorf("failed to verify driver: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("driver not found or already verified")
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
