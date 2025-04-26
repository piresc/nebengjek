package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// getDriverInfo retrieves driver information for a user
func (r *UserRepo) getDriverInfo(ctx context.Context, userID uuid.UUID) (*models.Driver, error) {
	query := `SELECT * FROM drivers WHERE user_id = $1`

	var driver models.Driver
	err := r.db.GetContext(ctx, &driver, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get driver info: %w", err)
	}
	return &driver, nil
}

func (r *UserRepo) UpdateToDriver(ctx context.Context, user *models.User) error {
	// Begin transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	user.UpdatedAt = time.Now()
	defer tx.Rollback()
	// Update user role to driver
	query := `
		UPDATE users
		SET role = :role, updated_at = :updated_at
		WHERE id = :id
	`
	_, err = tx.NamedExecContext(ctx, query, user)
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}

	// Create a map for driver info with user_id
	driverData := map[string]interface{}{
		"user_id":       user.ID,
		"vehicle_type":  user.DriverInfo.VehicleType,
		"vehicle_plate": user.DriverInfo.VehiclePlate,
	}

	query = `
			INSERT INTO drivers (
				user_id, vehicle_type, vehicle_plate
			) VALUES (:user_id, :vehicle_type, :vehicle_plate)
		`
	_, err = tx.NamedExecContext(ctx, query, driverData)
	if err != nil {
		return fmt.Errorf("failed to insert driver info: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// GetUserByID retrieves a user by ID
func (r *UserRepo) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	user, err := r.getUserByField(ctx, "id", id)
	if err != nil {
		return nil, err
	}
	return user, nil
}
