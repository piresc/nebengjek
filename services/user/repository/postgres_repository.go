package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// PostgresUserRepository implements UserRepository interface using PostgreSQL
type PostgresUserRepository struct {
	db *pgxpool.Pool
}

// NewPostgresUserRepository creates a new PostgreSQL user repository
func NewPostgresUserRepository(db *pgxpool.Pool) UserRepository {
	return &PostgresUserRepository{db: db}
}

// CreateUser creates a new user in the database
func (r *PostgresUserRepository) CreateUser(ctx context.Context, user *models.User) error {
	// Generate UUID if not provided
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Begin transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert user
	query := `
		INSERT INTO users (
			id, email, phone_number, full_name, password, role, 
			created_at, updated_at, is_active, rating
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err = tx.Exec(ctx, query,
		user.ID, user.Email, user.PhoneNumber, user.FullName, user.Password,
		user.Role, user.CreatedAt, user.UpdatedAt, user.IsActive, user.Rating,
	)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	// If user is a driver, insert driver info
	if user.Role == "driver" && user.DriverInfo != nil {
		query = `
			INSERT INTO drivers (
				user_id, vehicle_type, vehicle_plate, vehicle_model, vehicle_color,
				license_number, verified, verified_at, is_available
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`
		_, err = tx.Exec(ctx, query,
			user.ID, user.DriverInfo.VehicleType, user.DriverInfo.VehiclePlate,
			user.DriverInfo.VehicleModel, user.DriverInfo.VehicleColor,
			user.DriverInfo.LicenseNumber, user.DriverInfo.Verified,
			user.DriverInfo.VerifiedAt, user.DriverInfo.IsAvailable,
		)
		if err != nil {
			return fmt.Errorf("failed to insert driver info: %w", err)
		}

		// Insert driver documents if any
		if len(user.DriverInfo.Documents) > 0 {
			for _, doc := range user.DriverInfo.Documents {
				_, err = tx.Exec(ctx, `
					INSERT INTO driver_documents (user_id, document_url)
					VALUES ($1, $2)
				`, user.ID, doc)
				if err != nil {
					return fmt.Errorf("failed to insert driver document: %w", err)
				}
			}
		}

		// Insert driver location if available
		if user.DriverInfo.CurrentLocation != nil {
			_, err = tx.Exec(ctx, `
				INSERT INTO driver_locations (user_id, latitude, longitude, address, timestamp)
				VALUES ($1, $2, $3, $4, $5)
			`, user.ID,
				user.DriverInfo.CurrentLocation.Latitude,
				user.DriverInfo.CurrentLocation.Longitude,
				user.DriverInfo.CurrentLocation.Address,
				user.DriverInfo.CurrentLocation.Timestamp,
			)
			if err != nil {
				return fmt.Errorf("failed to insert driver location: %w", err)
			}
		}
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetUserByID retrieves a user by ID
func (r *PostgresUserRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	user, err := r.getUserByField(ctx, "id", id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByEmail retrieves a user by email
func (r *PostgresUserRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user, err := r.getUserByField(ctx, "email", email)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByPhoneNumber retrieves a user by phone number
func (r *PostgresUserRepository) GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*models.User, error) {
	user, err := r.getUserByField(ctx, "phone_number", phoneNumber)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// getUserByField is a helper function to get a user by a specific field
func (r *PostgresUserRepository) getUserByField(ctx context.Context, field, value string) (*models.User, error) {
	query := fmt.Sprintf(`
		SELECT 
			u.id, u.email, u.phone_number, u.full_name, u.password, u.role,
			u.created_at, u.updated_at, u.is_active, u.rating
		FROM users u
		WHERE u.%s = $1
	`, field)

	var user models.User
	err := r.db.QueryRow(ctx, query, value).Scan(
		&user.ID, &user.Email, &user.PhoneNumber, &user.FullName, &user.Password,
		&user.Role, &user.CreatedAt, &user.UpdatedAt, &user.IsActive, &user.Rating,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
func (r *PostgresUserRepository) getDriverInfo(ctx context.Context, userID string) (*models.Driver, error) {
	query := `
		SELECT 
			vehicle_type, vehicle_plate, vehicle_model, vehicle_color,
			license_number, verified, verified_at, is_available
		FROM drivers
		WHERE user_id = $1
	`

	var driver models.Driver
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&driver.VehicleType, &driver.VehiclePlate, &driver.VehicleModel, &driver.VehicleColor,
		&driver.LicenseNumber, &driver.Verified, &driver.VerifiedAt, &driver.IsAvailable,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get driver info: %w", err)
	}

	// Get driver documents
	rows, err := r.db.Query(ctx, `
		SELECT document_url FROM driver_documents
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get driver documents: %w", err)
	}
	defer rows.Close()

	var documents []string
	for rows.Next() {
		var doc string
		if err := rows.Scan(&doc); err != nil {
			return nil, fmt.Errorf("failed to scan driver document: %w", err)
		}
		documents = append(documents, doc)
	}
	driver.Documents = documents

	// Get driver location
	var location models.Location
	err = r.db.QueryRow(ctx, `
		SELECT latitude, longitude, address, timestamp
		FROM driver_locations
		WHERE user_id = $1
		ORDER BY timestamp DESC
		LIMIT 1
	`, userID).Scan(
		&location.Latitude, &location.Longitude, &location.Address, &location.Timestamp,
	)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("failed to get driver location: %w", err)
		}
	} else {
		driver.CurrentLocation = &location
	}

	return &driver, nil
}

// UpdateUser updates an existing user
func (r *PostgresUserRepository) UpdateUser(ctx context.Context, user *models.User) error {
	// Begin transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Update timestamps
	user.UpdatedAt = time.Now()

	// Update user
	query := `
		UPDATE users SET
			email = $1, phone_number = $2, full_name = $3, 
			role = $4, updated_at = $5, is_active = $6, rating = $7
		WHERE id = $8
	`
	result, err := tx.Exec(ctx, query,
		user.Email, user.PhoneNumber, user.FullName, user.Role,
		user.UpdatedAt, user.IsActive, user.Rating, user.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	// If password is provided, update it
	if user.Password != "" {
		_, err = tx.Exec(ctx, `UPDATE users SET password = $1 WHERE id = $2`, user.Password, user.ID)
		if err != nil {
			return fmt.Errorf("failed to update password: %w", err)
		}
	}

	// If user is a driver, update driver info
	if user.Role == "driver" && user.DriverInfo != nil {
		// Check if driver info exists
		var exists bool
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM drivers WHERE user_id = $1)`, user.ID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check driver existence: %w", err)
		}

		if exists {
			// Update driver info
			query = `
				UPDATE drivers SET
					vehicle_type = $1, vehicle_plate = $2, vehicle_model = $3, vehicle_color = $4,
					license_number = $5, verified = $6, verified_at = $7, is_available = $8
				WHERE user_id = $9
			`
			_, err = tx.Exec(ctx, query,
				user.DriverInfo.VehicleType, user.DriverInfo.VehiclePlate,
				user.DriverInfo.VehicleModel, user.DriverInfo.VehicleColor,
				user.DriverInfo.LicenseNumber, user.DriverInfo.Verified,
				user.DriverInfo.VerifiedAt, user.DriverInfo.IsAvailable, user.ID,
			)
			if err != nil {
				return fmt.Errorf("failed to update driver info: %w", err)
			}
		} else {
			// Insert driver info
			query = `
				INSERT INTO drivers (
					user_id, vehicle_type, vehicle_plate, vehicle_model, vehicle_color,
					license_number, verified, verified_at, is_available
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			`
			_, err = tx.Exec(ctx, query,
				user.ID, user.DriverInfo.VehicleType, user.DriverInfo.VehiclePlate,
				user.DriverInfo.VehicleModel, user.DriverInfo.VehicleColor,
				user.DriverInfo.LicenseNumber, user.DriverInfo.Verified,
				user.DriverInfo.VerifiedAt, user.DriverInfo.IsAvailable,
			)
			if err != nil {
				return fmt.Errorf("failed to insert driver info: %w", err)
			}
		}

		// Update driver documents if any
		if len(user.DriverInfo.Documents) > 0 {
			// Delete existing documents
			_, err = tx.Exec(ctx, `DELETE FROM driver_documents WHERE user_id = $1`, user.ID)
			if err != nil {
				return fmt.Errorf("failed to delete driver documents: %w", err)
			}

			// Insert new documents
			for _, doc := range user.DriverInfo.Documents {
				_, err = tx.Exec(ctx, `
					INSERT INTO driver_documents (user_id, document_url)
					VALUES ($1, $2)
				`, user.ID, doc)
				if err != nil {
					return fmt.Errorf("failed to insert driver document: %w", err)
				}
			}
		}

		// Update driver location if available
		if user.DriverInfo.CurrentLocation != nil {
			_, err = tx.Exec(ctx, `
				INSERT INTO driver_locations (user_id, latitude, longitude, address, timestamp)
				VALUES ($1, $2, $3, $4, $5)
			`, user.ID,
				user.DriverInfo.CurrentLocation.Latitude,
				user.DriverInfo.CurrentLocation.Longitude,
				user.DriverInfo.CurrentLocation.Address,
				user.DriverInfo.CurrentLocation.Timestamp,
			)
			if err != nil {
				return fmt.Errorf("failed to insert driver location: %w", err)
			}
		}
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteUser deletes a user by ID
func (r *PostgresUserRepository) DeleteUser(ctx context.Context, id string) error {
	// Begin transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check if user exists and is a driver
	var isDriver bool
	err = tx.QueryRow(ctx, `SELECT role = 'driver' FROM users WHERE id = $1`, id).Scan(&isDriver)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("failed to check user role: %w", err)
	}

	// If user is a driver, delete driver-related data
	if isDriver {
		// Delete driver documents
		_, err = tx.Exec(ctx, `DELETE FROM driver_documents WHERE user_id = $1`, id)
		if err != nil {
			return fmt.Errorf("failed to delete driver documents: %w", err)
		}

		// Delete driver locations
		_, err = tx.Exec(ctx, `DELETE FROM driver_locations WHERE user_id = $1`, id)
		if err != nil {
			return fmt.Errorf("failed to delete driver locations: %w", err)
		}

		// Delete driver info
		_, err = tx.Exec(ctx, `DELETE FROM drivers WHERE user_id = $1`, id)
		if err != nil {
			return fmt.Errorf("failed to delete driver info: %w", err)
		}
	}

	// Delete user
	result, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ListUsers retrieves a list of users with pagination
func (r *PostgresUserRepository) ListUsers(ctx context.Context, offset, limit int) ([]*models.User, error) {
	query := `
		SELECT 
			id, email, phone_number, full_name, password, role,
			created_at, updated_at, is_active, rating
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID, &user.Email, &user.PhoneNumber, &user.FullName, &user.Password,
			&user.Role, &user.CreatedAt, &user.UpdatedAt, &user.IsActive, &user.Rating,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		// If user is a driver, get driver info
		if user.Role == "driver" {
			driver, err := r.getDriverInfo(ctx, user.ID)
			if err != nil {
				return nil, err
			}
			user.DriverInfo = driver
		}

		users = append(users, &user)
	}

	return users, nil
}

// UpdateDriverLocation updates a driver's current location
func (r *PostgresUserRepository) UpdateDriverLocation(ctx context.Context, driverID string, location *models.Location) error {
	// Ensure timestamp is set
	if location.Timestamp.IsZero() {
		location.Timestamp = time.Now()
	}

	// Insert new location
	_, err := r.db.Exec(ctx, `
		INSERT INTO driver_locations (user_id, latitude, longitude, address, timestamp)
		VALUES ($1, $2, $3, $4, $5)
	`, driverID, location.Latitude, location.Longitude, location.Address, location.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to update driver location: %w", err)
	}

	return nil
}

// UpdateDriverAvailability updates a driver's availability status
func (r *PostgresUserRepository) UpdateDriverAvailability(ctx context.Context, driverID string, isAvailable bool) error {
	result, err := r.db.Exec(ctx, `
		UPDATE drivers SET is_available = $1
		WHERE user_id = $2
	`, isAvailable, driverID)
	if err != nil {
		return fmt.Errorf("failed to update driver availability: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("driver not found")
	}

	return nil
}

// GetNearbyDrivers retrieves available drivers near a location within a radius
func (r *PostgresUserRepository) GetNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.User, error) {
	// This query uses the Haversine formula to calculate distance between two points on Earth
	query := `
		SELECT 
			u.id, u.email, u.phone_number, u.full_name, u.password, u.role,
			u.created_at, u.updated_at, u.is_active, u.rating,
			d.vehicle_type, d.vehicle_plate, d.vehicle_model, d.vehicle_color,
			d.license_number, d.verified, d.verified_at, d.is_available,
			dl.latitude, dl.longitude, dl.address, dl.timestamp
		FROM users u
		JOIN drivers d ON u.id = d.user_id
		JOIN (
			SELECT DISTINCT ON (user_id) user_id, latitude, longitude, address, timestamp
			FROM driver_locations
			ORDER BY user_id, timestamp DESC
		) dl ON u.id = dl.user_id
		WHERE d.is_available = true AND d.verified = true AND u.is_active = true
		AND (
			6371 * acos(
				cos(radians($1)) * 
				cos(radians(dl.latitude)) * 
				cos(radians(dl.longitude) - radians($2)) + 
				sin(radians($1)) * 
				sin(radians(dl.latitude))
			)
		) <= $3
		ORDER BY (
			6371 * acos(
				cos(radians($1)) * 
				cos(radians(dl.latitude)) * 
				cos(radians(dl.longitude) - radians($2)) + 
				sin(radians($1)) * 
				sin(radians(dl.latitude))
			)
		) ASC
	`

	rows, err := r.db.Query(ctx, query, location.Latitude, location.Longitude, radiusKm)
	if err != nil {
		return nil, fmt.Errorf("failed to get nearby drivers: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		var driver models.Driver
		var driverLocation models.Location

		err := rows.Scan(
			&user.ID, &user.Email, &user.PhoneNumber, &user.FullName, &user.Password,
			&user.Role, &user.CreatedAt, &user.UpdatedAt, &user.IsActive, &user.Rating,
			&driver.VehicleType, &driver.VehiclePlate, &driver.VehicleModel, &driver.VehicleColor,
			&driver.LicenseNumber, &driver.Verified, &driver.VerifiedAt, &driver.IsAvailable,
			&driverLocation.Latitude, &driverLocation.Longitude, &driverLocation.Address, &driverLocation.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan driver: %w", err)
		}

		driver.CurrentLocation = &driverLocation

		// Get driver documents
		docRows, err := r.db.Query(ctx, `
			SELECT document_url FROM driver_