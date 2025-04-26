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

// GetUserByMSISDN retrieves a user by MSISDN
func (r *UserRepo) GetUserByMSISDN(ctx context.Context, msisdn string) (*models.User, error) {
	query := `
		SELECT id, msisdn, fullname, role, created_at, updated_at, is_active
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

// CreateUser creates a new user in the database
func (r *UserRepo) CreateUser(ctx context.Context, user *models.User) error {
	user.ID = uuid.New()
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
			created_at, updated_at, is_active
		) VALUES (:id, :msisdn, :fullname, :role, 
			:created_at, :updated_at, :is_active)
	`
	_, err = tx.NamedExecContext(ctx, query, user)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
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
