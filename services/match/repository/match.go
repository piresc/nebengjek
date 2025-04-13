package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// MatchRepo implements the match repository interface
type MatchRepo struct {
	cfg         *models.Config
	db          *sqlx.DB
	redisClient *database.RedisClient
}

// Constants for Redis keys
const (
	AvailableDriversKey    = "available_drivers"    // Geo set for available drivers
	AvailablePassengersKey = "available_passengers" // Geo set for available passengers
)

// NewMatchRepository creates a new match repository
func NewMatchRepository(
	cfg *models.Config,
	db *sqlx.DB,
	redisClient *database.RedisClient,
) *MatchRepo {
	log.Println("Initializing match repository")
	return &MatchRepo{
		cfg:         cfg,
		db:          db,
		redisClient: redisClient,
	}
}

// CreateMatch creates a new match (trip) in the database
func (r *MatchRepo) CreateMatch(ctx context.Context, trip *models.Trip) error {
	// Generate UUID if not provided
	if trip.ID == "" {
		trip.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	if trip.RequestedAt.IsZero() {
		trip.RequestedAt = now
	}
	if trip.Status == "" {
		trip.Status = models.TripStatusRequested
	}
	if trip.Status == models.TripStatusMatched && trip.MatchedAt == nil {
		trip.MatchedAt = &now
	}

	// Begin transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert trip
	query := `
		INSERT INTO trips (
			id, passenger_id, driver_id, status, requested_at, matched_at,
			distance, duration, notes
		) VALUES (
			:id, :passenger_id, :driver_id, :status, :requested_at, :matched_at,
			:distance, :duration, :notes
		)
	`
	_, err = tx.NamedExecContext(ctx, query, trip)
	if err != nil {
		return fmt.Errorf("failed to insert trip: %w", err)
	}

	// Insert pickup location
	pickupQuery := `
		INSERT INTO trips_locations (
			trip_id, type, latitude, longitude, address
		) VALUES (
			$1, 'pickup', $2, $3, $4
		)
	`
	_, err = tx.ExecContext(
		ctx,
		pickupQuery,
		trip.ID,
		trip.PickupLocation.Latitude,
		trip.PickupLocation.Longitude,
	)
	if err != nil {
		return fmt.Errorf("failed to insert pickup location: %w", err)
	}

	// Insert dropoff location
	dropoffQuery := `
		INSERT INTO trips_locations (
			trip_id, type, latitude, longitude, address
		) VALUES (
			$1, 'dropoff', $2, $3, $4
		)
	`
	_, err = tx.ExecContext(
		ctx,
		dropoffQuery,
		trip.ID,
		trip.DropoffLocation.Latitude,
		trip.DropoffLocation.Longitude,
	)
	if err != nil {
		return fmt.Errorf("failed to insert dropoff location: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdateMatchStatus updates the status of a match
func (r *MatchRepo) UpdateMatchStatus(ctx context.Context, tripID string, status models.TripStatus) error {
	// Begin transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update status and corresponding timestamp
	var query string
	var args []interface{}

	switch status {
	case models.TripStatusMatched:
		query = `UPDATE trip SET status = $1, matched_at = $2 WHERE id = $3`
		args = []interface{}{status, time.Now(), tripID}
	case models.TripStatusAccepted:
		query = `UPDATE trip SET status = $1, accepted_at = $2 WHERE id = $3`
		args = []interface{}{status, time.Now(), tripID}
	case models.TripStatusRejected, models.TripStatusCancelled:
		query = `UPDATE trip SET status = $1, cancelled_at = $2 WHERE id = $3`
		args = []interface{}{status, time.Now(), tripID}
	case models.TripStatusInProgress:
		query = `UPDATE trip SET status = $1, started_at = $2 WHERE id = $3`
		args = []interface{}{status, time.Now(), tripID}
	case models.TripStatusCompleted:
		query = `UPDATE trip SET status = $1, completed_at = $2 WHERE id = $3`
		args = []interface{}{status, time.Now(), tripID}
	default:
		query = `UPDATE trip SET status = $1 WHERE id = $2`
		args = []interface{}{status, tripID}
	}

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update trip status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("trip not found")
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetMatchByID retrieves a match by ID
func (r *MatchRepo) GetMatchByID(ctx context.Context, id string) (*models.Trip, error) {
	// Query trip
	query := `SELECT * FROM trip WHERE id = $1`
	var trip models.Trip
	err := r.db.GetContext(ctx, &trip, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("trip not found")
		}
		return nil, fmt.Errorf("failed to get trip: %w", err)
	}

	// Query pickup location
	pickupQuery := `
		SELECT latitude, longitude, address, timestamp
		FROM trip_locations
		WHERE trip_id = $1 AND type = 'pickup'
	`
	err = r.db.GetContext(ctx, &trip.PickupLocation, pickupQuery, id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get pickup location: %w", err)
	}

	// Query dropoff location
	dropoffQuery := `
		SELECT latitude, longitude, address, timestamp
		FROM trip_locations
		WHERE trip_id = $1 AND type = 'dropoff'
	`
	err = r.db.GetContext(ctx, &trip.DropoffLocation, dropoffQuery, id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get dropoff location: %w", err)
	}

	return &trip, nil
}

// GetPendingMatchesByDriverID retrieves pending matches for a driver
func (r *MatchRepo) GetPendingMatchesByDriverID(ctx context.Context, driverID string) ([]*models.Trip, error) {
	query := `
		SELECT * FROM trip 
		WHERE driver_id = $1 AND status IN ($2, $3)
		ORDER BY requested_at DESC
	`

	var trip []*models.Trip
	err := r.db.SelectContext(
		ctx,
		&trip,
		query,
		driverID,
		models.TripStatusMatched,
		models.TripStatusRequested,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending trip: %w", err)
	}

	// Get locations for each trip
	for _, trip := range trip {
		// Query pickup location
		pickupQuery := `
			SELECT latitude, longitude, address, timestamp
			FROM trip_locations
			WHERE trip_id = $1 AND type = 'pickup'
		`
		err = r.db.GetContext(ctx, &trip.PickupLocation, pickupQuery, trip.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to get pickup location: %w", err)
		}

		// Query dropoff location
		dropoffQuery := `
			SELECT latitude, longitude, address, timestamp
			FROM trip_locations
			WHERE trip_id = $1 AND type = 'dropoff'
		`
		err = r.db.GetContext(ctx, &trip.DropoffLocation, dropoffQuery, trip.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to get dropoff location: %w", err)
		}
	}

	return trip, nil
}

// GetPendingMatchesByPassengerID retrieves pending matches for a passenger
func (r *MatchRepo) GetPendingMatchesByPassengerID(ctx context.Context, passengerID string) ([]*models.Trip, error) {
	query := `
		SELECT * FROM trip 
		WHERE passenger_id = $1 AND status IN ($2, $3, $4)
		ORDER BY requested_at DESC
	`

	var trip []*models.Trip
	err := r.db.SelectContext(
		ctx,
		&trip,
		query,
		passengerID,
		models.TripStatusRequested,
		models.TripStatusMatched,
		models.TripStatusAccepted,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending trip: %w", err)
	}

	// Get locations for each trip
	for _, trip := range trip {
		// Query pickup location
		pickupQuery := `
			SELECT latitude, longitude, address, timestamp
			FROM trip_locations
			WHERE trip_id = $1 AND type = 'pickup'
		`
		err = r.db.GetContext(ctx, &trip.PickupLocation, pickupQuery, trip.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to get pickup location: %w", err)
		}

		// Query dropoff location
		dropoffQuery := `
			SELECT latitude, longitude, address, timestamp
			FROM trip_locations
			WHERE trip_id = $1 AND type = 'dropoff'
		`
		err = r.db.GetContext(ctx, &trip.DropoffLocation, dropoffQuery, trip.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to get dropoff location: %w", err)
		}
	}

	return trip, nil
}

// AddAvailableDriver adds a driver to the Redis geospatial index
func (r *MatchRepo) AddAvailableDriver(ctx context.Context, driverMSISDN string, location *models.Location) error {
	err := r.redisClient.GeoAdd(
		ctx,
		AvailableDriversKey,
		location.Longitude,
		location.Latitude,
		driverMSISDN,
	)
	if err != nil {
		return fmt.Errorf("failed to add driver to geo index: %w", err)
	}
	return nil
}

// RemoveAvailableDriver removes a driver from the Redis geospatial index
func (r *MatchRepo) RemoveAvailableDriver(ctx context.Context, driverMSISDN string) error {
	err := r.redisClient.Delete(ctx, fmt.Sprintf("%s:%s", AvailableDriversKey, driverMSISDN))
	if err != nil {
		return fmt.Errorf("failed to remove driver from geo index: %w", err)
	}
	return nil
}

// FindNearbyDrivers finds available drivers within the specified radius
func (r *MatchRepo) FindNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]string, error) {
	results, err := r.redisClient.GeoRadius(
		ctx,
		AvailableDriversKey,
		location.Longitude,
		location.Latitude,
		radiusKm,
		"km",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby drivers: %w", err)
	}

	driverIDs := make([]string, len(results))
	for i, result := range results {
		driverIDs[i] = result.Name
	}

	return driverIDs, nil
}

// ProcessLocationUpdate processes a location update for a driver
func (r *MatchRepo) ProcessLocationUpdate(ctx context.Context, driverID string, location *models.Location) error {
	// Update driver's location in Redis
	err := r.redisClient.GeoAdd(
		ctx,
		AvailableDriversKey,
		location.Longitude,
		location.Latitude,
		driverID,
	)
	if err != nil {
		return fmt.Errorf("failed to update driver location: %w", err)
	}
	return nil
}

// FindMatchForPassenger finds a match for a passenger
func (r *MatchRepo) FindMatchForPassenger(ctx context.Context, passengerID string, location *models.Location, radiusKm float64) (*models.Trip, error) {
	// Find nearby drivers
	driverIDs, err := r.FindNearbyDrivers(ctx, location, radiusKm)
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby drivers: %w", err)
	}

	// Create a new trip
	trip := &models.Trip{
		PassengerMSISDN: passengerID,
		PickupLocation:  *location,
		Status:          models.TripStatusRequested,
		RequestedAt:     time.Now(),
	}

	// Assign the first available driver
	if len(driverIDs) > 0 {
		trip.DriverMSISDN = driverIDs[0]
		trip.Status = models.TripStatusMatched
		now := time.Now()
		trip.MatchedAt = &now
	}

	// Save the trip to the database
	err = r.CreateMatch(ctx, trip)
	if err != nil {
		return nil, fmt.Errorf("failed to create trip: %w", err)
	}

	return trip, nil
}

// AddAvailablePassenger adds a passenger to the Redis geospatial index
func (r *MatchRepo) AddAvailablePassenger(ctx context.Context, passengerID string, location *models.Location) error {
	err := r.redisClient.GeoAdd(
		ctx,
		AvailablePassengersKey,
		location.Longitude,
		location.Latitude,
		passengerID,
	)
	if err != nil {
		return fmt.Errorf("failed to add passenger to geo index: %w", err)
	}
	return nil
}

// RemoveAvailablePassenger removes a passenger from the Redis geospatial index
func (r *MatchRepo) RemoveAvailablePassenger(ctx context.Context, passengerID string) error {
	err := r.redisClient.Delete(ctx, fmt.Sprintf("%s:%s", AvailablePassengersKey, passengerID))
	if err != nil {
		return fmt.Errorf("failed to remove passenger from geo index: %w", err)
	}
	return nil
}

// FindNearbyPassengers finds available passengers within the specified radius
func (r *MatchRepo) FindNearbyPassengers(ctx context.Context, location *models.Location, radiusKm float64) ([]string, error) {
	results, err := r.redisClient.GeoRadius(
		ctx,
		AvailablePassengersKey,
		location.Longitude,
		location.Latitude,
		radiusKm,
		"km",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby passengers: %w", err)
	}

	passengerIDs := make([]string, len(results))
	for i, result := range results {
		passengerIDs[i] = result.Name
	}

	return passengerIDs, nil
}
