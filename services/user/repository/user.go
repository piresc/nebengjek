package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

type UserRepo struct {
	cfg *models.Config
	db  *sqlx.DB
}

func NewUserRepository(
	cfg *models.Config,
	db *sqlx.DB,
) *UserRepo {
	log.Println("Initializing user repository")
	return &UserRepo{
		cfg: cfg,
		db:  db,
	}
}

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
		INSERT INTO users (
			id, email, phone_number, full_name, password, role, 
			created_at, updated_at, is_active, rating
		) VALUES (:id, :email, :phone_number, :full_name, :password, :role, 
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

		// Insert driver documents if any
		if len(user.DriverInfo.Documents) > 0 {
			for _, doc := range user.DriverInfo.Documents {
				_, err = tx.ExecContext(ctx, `
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
			locationData := map[string]interface{}{
				"user_id":   user.ID,
				"latitude":  user.DriverInfo.CurrentLocation.Latitude,
				"longitude": user.DriverInfo.CurrentLocation.Longitude,
				"address":   user.DriverInfo.CurrentLocation.Address,
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

// GetUserByEmail retrieves a user by email
func (r *UserRepo) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user, err := r.getUserByField(ctx, "email", email)
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

	// Get driver documents
	var documents []string
	err = r.db.SelectContext(ctx, &documents, `
		SELECT document_url FROM driver_documents
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get driver documents: %w", err)
	}
	driver.Documents = documents

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
			email = :email, phone_number = :phone_number, full_name = :full_name, 
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

	// If password is provided, update it
	if user.Password != "" {
		_, err = tx.ExecContext(ctx, `UPDATE users SET password = $1 WHERE id = $2`, user.Password, user.ID)
		if err != nil {
			return fmt.Errorf("failed to update password: %w", err)
		}
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

		// Update driver documents if any
		if len(user.DriverInfo.Documents) > 0 {
			// Delete existing documents
			_, err = tx.ExecContext(ctx, `DELETE FROM driver_documents WHERE user_id = $1`, user.ID)
			if err != nil {
				return fmt.Errorf("failed to delete driver documents: %w", err)
			}

			// Insert new documents
			for _, doc := range user.DriverInfo.Documents {
				_, err = tx.ExecContext(ctx, `
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
			locationData := map[string]interface{}{
				"user_id":   user.ID,
				"latitude":  user.DriverInfo.CurrentLocation.Latitude,
				"longitude": user.DriverInfo.CurrentLocation.Longitude,
				"address":   user.DriverInfo.CurrentLocation.Address,
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
		// Delete driver documents
		_, err = tx.ExecContext(ctx, `DELETE FROM driver_documents WHERE user_id = $1`, id)
		if err != nil {
			return fmt.Errorf("failed to delete driver documents: %w", err)
		}

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

// UpdateDriverLocation updates a driver's current location
func (r *UserRepo) UpdateDriverLocation(ctx context.Context, driverID string, location *models.Location) error {
	// Set timestamp if not provided
	if location.Timestamp.IsZero() {
		location.Timestamp = time.Now()
	}

	// Create a map for location data
	locationData := map[string]interface{}{
		"user_id":   driverID,
		"latitude":  location.Latitude,
		"longitude": location.Longitude,
		"address":   location.Address,
		"timestamp": location.Timestamp,
	}

	// Insert new location record
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO driver_locations (user_id, latitude, longitude, address, timestamp)
		VALUES (:user_id, :latitude, :longitude, :address, :timestamp)
	`, locationData)
	if err != nil {
		return fmt.Errorf("failed to update driver location: %w", err)
	}

	return nil
}

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

// GetNearbyDrivers retrieves drivers near a specific location within a radius
func (r *UserRepo) GetNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.User, error) {
	// This is a simplified implementation using a bounding box
	// For production, consider using PostGIS for proper geospatial queries

	// Calculate approximate latitude/longitude bounds for the given radius
	// 1 degree of latitude is approximately 111 km
	latDelta := radiusKm / 111.0
	// 1 degree of longitude varies with latitude, approximate at the given latitude
	lngDelta := radiusKm / (111.0 * math.Cos(location.Latitude*math.Pi/180.0))

	// Query for drivers within the bounding box who are available
	query := `
		SELECT DISTINCT u.* 
		FROM users u
		JOIN drivers d ON u.id = d.user_id
		JOIN driver_locations dl ON u.id = dl.user_id
		WHERE u.role = 'driver'
		  AND d.is_available = true
		  AND dl.latitude BETWEEN $1 AND $2
		  AND dl.longitude BETWEEN $3 AND $4
		  AND dl.timestamp > $5
		ORDER BY dl.timestamp DESC
	`

	// Only consider locations updated in the last hour
	oneHourAgo := time.Now().Add(-1 * time.Hour)

	var drivers []*models.User
	err := r.db.SelectContext(ctx, &drivers, query,
		location.Latitude-latDelta, location.Latitude+latDelta,
		location.Longitude-lngDelta, location.Longitude+lngDelta,
		oneHourAgo,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby drivers: %w", err)
	}

	// Get driver info for each driver
	for _, user := range drivers {
		driver, err := r.getDriverInfo(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		user.DriverInfo = driver
	}

	return drivers, nil
}

// VerifyDriver marks a driver as verified
func (r *UserRepo) VerifyDriver(ctx context.Context, driverID string) error {
	// Update driver verification status
	result, err := r.db.ExecContext(ctx, `
		UPDATE drivers
		SET verified = true, verified_at = $1
		WHERE user_id = $2
	`, time.Now(), driverID)
	if err != nil {
		return fmt.Errorf("failed to verify driver: %w", err)
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
