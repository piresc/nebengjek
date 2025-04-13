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

// CreateUser creates a new user in the database
func (r *UserRepo) CreateUser(ctx context.Context, user *models.User) error {
	// Generate UUID if not provided
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Begin transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert user
	query := `
		INSERT INTO users (id, msisdn, fullname, role, 
			created_at, updated_at, is_active, rating
		) VALUES (:id, :msisdn, :fullname, :role, 
			:created_at, :updated_at, :is_active, :rating)
	`
	_, err = tx.NamedExecContext(ctx, query, user)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	// If user is a driver, insert driver info
	if user.Role == "driver" && user.DriverInfo != nil {
		// Create a map for driver info with user_id
		driverData := map[string]interface{}{
			"user_id":        user.ID,
			"vehicle_type":   user.DriverInfo.VehicleType,
			"vehicle_plate":  user.DriverInfo.VehiclePlate,
			"vehicle_model":  user.DriverInfo.VehicleModel,
			"vehicle_color":  user.DriverInfo.VehicleColor,
			"license_number": user.DriverInfo.LicenseNumber,
			"verified":       user.DriverInfo.Verified,
			"verified_at":    user.DriverInfo.VerifiedAt,
			"is_available":   user.DriverInfo.IsAvailable,
		}

		query = `
			INSERT INTO drivers (
				user_id, vehicle_type, vehicle_plate, vehicle_model, vehicle_color,
				license_number, verified, verified_at, is_available
			) VALUES (:user_id, :vehicle_type, :vehicle_plate, :vehicle_model, :vehicle_color,
				:license_number, :verified, :verified_at, :is_available)
		`
		_, err = tx.NamedExecContext(ctx, query, driverData)
		if err != nil {
			return fmt.Errorf("failed to insert driver info: %w", err)
		}

		// Driver documents functionality has been removed

		// Insert driver location if available
		if user.DriverInfo.CurrentLocation != nil {
			locationData := map[string]interface{}{
				"user_id":   user.ID,
				"latitude":  user.DriverInfo.CurrentLocation.Latitude,
				"longitude": user.DriverInfo.CurrentLocation.Longitude,
				"timestamp": user.DriverInfo.CurrentLocation.Timestamp,
			}

			_, err = tx.NamedExecContext(ctx, `
				INSERT INTO driver_locations (user_id, latitude, longitude, address, timestamp)
				VALUES (:user_id, :latitude, :longitude, :address, :timestamp)
			`, locationData)
			if err != nil {
				return fmt.Errorf("failed to insert driver location: %w", err)
			}
		}
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

// GetUserByPhoneNumber retrieves a user by phone number
func (r *UserRepo) GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*models.User, error) {
	user, err := r.getUserByField(ctx, "phone_number", phoneNumber)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// getUserByField is a helper function to get a user by a specific field
func (r *UserRepo) getUserByField(ctx context.Context, field, value string) (*models.User, error) {
	query := fmt.Sprintf(`
		SELECT * FROM users WHERE %s = $1
	`, field)

	var user models.User
	err := r.db.GetContext(ctx, &user, query, value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// If user is a driver, get driver info
	if user.Role == "driver" {
		driver, err := r.getDriverInfo(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		user.DriverInfo = driver
	}

	return &user, nil
}

// getDriverInfo retrieves driver information for a user
func (r *UserRepo) getDriverInfo(ctx context.Context, userID string) (*models.Driver, error) {
	query := `SELECT * FROM drivers WHERE user_id = $1`

	var driver models.Driver
	err := r.db.GetContext(ctx, &driver, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get driver info: %w", err)
	}

	// Driver documents functionality has been removed

	// Get driver location
	var location models.Location
	err = r.db.GetContext(ctx, &location, `
		SELECT * FROM driver_locations
		WHERE user_id = $1
		ORDER BY timestamp DESC
		LIMIT 1
	`, userID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to get driver location: %w", err)
		}
	} else {
		driver.CurrentLocation = &location
	}

	return &driver, nil
}

// UpdateUser updates an existing user
func (r *UserRepo) UpdateUser(ctx context.Context, user *models.User) error {
	// Begin transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update timestamps
	user.UpdatedAt = time.Now()

	// Update user
	updateQuery := `
		UPDATE users SET
			phone_number = :phone_number, full_name = :full_name, 
			role = :role, updated_at = :updated_at, is_active = :is_active, rating = :rating
		WHERE id = :id
	`
	result, err := tx.NamedExecContext(ctx, updateQuery, user)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	// If user is a driver, update driver info
	if user.Role == "driver" && user.DriverInfo != nil {
		// Check if driver info exists
		var exists bool
		err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM drivers WHERE user_id = $1)`, user.ID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check driver existence: %w", err)
		}

		// Create a map for driver data
		driverData := map[string]interface{}{
			"user_id":        user.ID,
			"vehicle_type":   user.DriverInfo.VehicleType,
			"vehicle_plate":  user.DriverInfo.VehiclePlate,
			"vehicle_model":  user.DriverInfo.VehicleModel,
			"vehicle_color":  user.DriverInfo.VehicleColor,
			"license_number": user.DriverInfo.LicenseNumber,
			"verified":       user.DriverInfo.Verified,
			"verified_at":    user.DriverInfo.VerifiedAt,
			"is_available":   user.DriverInfo.IsAvailable,
		}

		if exists {
			// Update driver info
			query := `
				UPDATE drivers SET
					vehicle_type = :vehicle_type, vehicle_plate = :vehicle_plate, 
					vehicle_model = :vehicle_model, vehicle_color = :vehicle_color,
					license_number = :license_number, verified = :verified, 
					verified_at = :verified_at, is_available = :is_available
				WHERE user_id = :user_id
			`
			_, err = tx.NamedExecContext(ctx, query, driverData)
			if err != nil {
				return fmt.Errorf("failed to update driver info: %w", err)
			}
		} else {
			// Insert driver info
			query := `
				INSERT INTO drivers (
					user_id, vehicle_type, vehicle_plate, vehicle_model, vehicle_color,
					license_number, verified, verified_at, is_available
				) VALUES (
					:user_id, :vehicle_type, :vehicle_plate, :vehicle_model, :vehicle_color,
					:license_number, :verified, :verified_at, :is_available
				)
			`
			_, err = tx.NamedExecContext(ctx, query, driverData)
			if err != nil {
				return fmt.Errorf("failed to insert driver info: %w", err)
			}
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteUser deletes a user by ID
func (r *UserRepo) DeleteUser(ctx context.Context, id string) error {
	// Begin transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if user exists and is a driver
	var isDriver bool
	err = tx.QueryRowContext(ctx, `SELECT role = 'driver' FROM users WHERE id = $1`, id).Scan(&isDriver)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("failed to check user role: %w", err)
	}

	// If user is a driver, delete driver-related data
	if isDriver {
		// Delete driver locations
		_, err = tx.ExecContext(ctx, `DELETE FROM driver_locations WHERE user_id = $1`, id)
		if err != nil {
			return fmt.Errorf("failed to delete driver locations: %w", err)
		}

		// Delete driver info
		_, err = tx.ExecContext(ctx, `DELETE FROM drivers WHERE user_id = $1`, id)
		if err != nil {
			return fmt.Errorf("failed to delete driver info: %w", err)
		}
	}

	// Delete user
	result, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ListUsers retrieves a list of users with pagination
func (r *UserRepo) ListUsers(ctx context.Context, offset, limit int) ([]*models.User, error) {
	query := `
		SELECT * FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	var users []*models.User
	err := r.db.SelectContext(ctx, &users, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Get driver info for drivers
	for _, user := range users {
		if user.Role == "driver" {
			driver, err := r.getDriverInfo(ctx, user.ID)
			if err != nil {
				return nil, err
			}
			user.DriverInfo = driver
		}
	}

	return users, nil
}
