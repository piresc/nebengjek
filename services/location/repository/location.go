package repository

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/location"
)

// LocationRepo implements the location.LocationRepo interface
type LocationRepo struct {
	db *sqlx.DB
}

// NewLocationRepository creates a new location repository
func NewLocationRepository(db *sqlx.DB) location.LocationRepo {
	return &LocationRepo{
		db: db,
	}
}

// UpdateDriverAvailability updates a driver's availability status
func (r *LocationRepo) UpdateDriverAvailability(ctx context.Context, driverID string, isAvailable bool) error {
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
func (r *LocationRepo) GetNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.User, error) {
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

// getDriverInfo retrieves driver information for a user
func (r *LocationRepo) getDriverInfo(ctx context.Context, userID string) (*models.Driver, error) {
	query := `SELECT * FROM drivers WHERE user_id = $1`

	var driver models.Driver
	err := r.db.GetContext(ctx, &driver, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get driver info: %w", err)
	}

	// Get driver location
	var location models.Location
	err = r.db.GetContext(ctx, &location, `
		SELECT * FROM driver_locations
		WHERE user_id = $1
		ORDER BY timestamp DESC
		LIMIT 1
	`, userID)
	if err != nil {
		// It's okay if there's no location yet
		driver.CurrentLocation = nil
	} else {
		driver.CurrentLocation = &location
	}

	return &driver, nil
}

// UpdateCustomerLocation updates a customer's current location
func (r *LocationRepo) UpdateCustomerLocation(ctx context.Context, customerID string, location *models.Location) error {
	// Ensure timestamp is set
	if location.Timestamp.IsZero() {
		location.Timestamp = time.Now()
	}

	// Create a map for location data
	locationData := map[string]interface{}{
		"user_id":   customerID,
		"latitude":  location.Latitude,
		"longitude": location.Longitude,
		"address":   location.Address,
		"timestamp": location.Timestamp,
	}

	// Insert new location record
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO customer_locations (user_id, latitude, longitude, address, timestamp)
		VALUES (:user_id, :latitude, :longitude, :address, :timestamp)
	`, locationData)
	if err != nil {
		return fmt.Errorf("failed to update customer location: %w", err)
	}

	return nil
}

// StoreLocationHistory stores a location update in the location history table
func (r *LocationRepo) StoreLocationHistory(ctx context.Context, userID string, role string, location *models.Location) error {
	// Ensure timestamp is set
	if location.Timestamp.IsZero() {
		location.Timestamp = time.Now()
	}

	// Create a map for location history data
	locationData := map[string]interface{}{
		"user_id":   userID,
		"role":      role,
		"latitude":  location.Latitude,
		"longitude": location.Longitude,
		"address":   location.Address,
		"timestamp": location.Timestamp,
	}

	// Insert new location history record
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO location_history (user_id, role, latitude, longitude, address, timestamp)
		VALUES (:user_id, :role, :latitude, :longitude, :address, :timestamp)
	`, locationData)
	if err != nil {
		return fmt.Errorf("failed to store location history: %w", err)
	}

	return nil
}

// GetLocationHistory retrieves location history for a user within a time range
func (r *LocationRepo) GetLocationHistory(ctx context.Context, userID string, startTime, endTime time.Time) ([]*models.Location, error) {
	// Query for location history within the time range
	query := `
		SELECT latitude, longitude, address, timestamp
		FROM location_history
		WHERE user_id = $1
		  AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp ASC
	`

	var locations []*models.Location
	err := r.db.SelectContext(ctx, &locations, query, userID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get location history: %w", err)
	}

	return locations, nil
}
