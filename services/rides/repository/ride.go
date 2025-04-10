package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

type RideRepo struct {
	cfg *models.Config
	db  *sqlx.DB
}

func NewRideRepository(
	cfg *models.Config,
	db *sqlx.DB,
) *RideRepo {
	log.Println("Initializing user repository")
	return &RideRepo{
		cfg: cfg,
		db:  db,
	}
}

// CreateRide creates a new ride in the database
func (r *RideRepo) CreateRide(ctx context.Context, trip *models.Trip) error {
	// SQL query to insert a new ride
	query := `
		INSERT INTO trips (
			id, passenger_id, driver_id, 
			pickup_latitude, pickup_longitude, pickup_address, pickup_timestamp,
			dropoff_latitude, dropoff_longitude, dropoff_address, dropoff_timestamp,
			requested_at, status, distance, duration,
			base_fare, distance_fare, duration_fare, surge_factor, total_fare, currency
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
	`

	// Execute the query
	_, err := r.db.ExecContext(
		ctx,
		query,
		trip.ID,
		trip.PassengerID,
		trip.DriverID,
		trip.PickupLocation.Latitude,
		trip.PickupLocation.Longitude,
		trip.PickupLocation.Address,
		trip.PickupLocation.Timestamp,
		trip.DropoffLocation.Latitude,
		trip.DropoffLocation.Longitude,
		trip.DropoffLocation.Address,
		trip.DropoffLocation.Timestamp,
		trip.RequestedAt,
		trip.Status,
		trip.Distance,
		trip.Duration,
		trip.Fare.BaseFare,
		trip.Fare.DistanceFare,
		trip.Fare.DurationFare,
		trip.Fare.SurgeFactor,
		trip.Fare.TotalFare,
		trip.Fare.Currency,
	)

	return err
}

// GetRideByID retrieves a ride by ID
func (r *RideRepo) GetRideByID(ctx context.Context, id string) (*models.Trip, error) {
	// SQL query to get a ride by ID
	query := `
		SELECT 
			id, passenger_id, driver_id, 
			pickup_latitude, pickup_longitude, pickup_address, pickup_timestamp,
			dropoff_latitude, dropoff_longitude, dropoff_address, dropoff_timestamp,
			requested_at, matched_at, accepted_at, started_at, completed_at, cancelled_at,
			status, distance, duration,
			base_fare, distance_fare, duration_fare, surge_factor, total_fare, currency,
			passenger_rating, driver_rating
		FROM trips
		WHERE id = $1
	`

	// Execute the query
	row := r.db.QueryRowContext(ctx, query, id)

	// Parse the result
	trip := &models.Trip{}
	var pickupLat, pickupLng, dropoffLat, dropoffLng float64
	var pickupAddr, dropoffAddr string
	var pickupTime, dropoffTime time.Time
	var matchedAt, acceptedAt, startedAt, completedAt, cancelledAt sql.NullTime
	var driverID sql.NullString
	var baseFare, distanceFare, durationFare, surgeFactor, totalFare float64
	var currency string
	var passengerRating, driverRating sql.NullFloat64

	err := row.Scan(
		&trip.ID,
		&trip.PassengerID,
		&driverID,
		&pickupLat,
		&pickupLng,
		&pickupAddr,
		&pickupTime,
		&dropoffLat,
		&dropoffLng,
		&dropoffAddr,
		&dropoffTime,
		&trip.RequestedAt,
		&matchedAt,
		&acceptedAt,
		&startedAt,
		&completedAt,
		&cancelledAt,
		&trip.Status,
		&trip.Distance,
		&trip.Duration,
		&baseFare,
		&distanceFare,
		&durationFare,
		&surgeFactor,
		&totalFare,
		&currency,
		&passengerRating,
		&driverRating,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("ride not found: %w", err)
		}
		return nil, err
	}

	// Set driver ID if present
	if driverID.Valid {
		trip.DriverID = driverID.String
	}

	// Set pickup and dropoff locations
	trip.PickupLocation = models.Location{
		Latitude:  pickupLat,
		Longitude: pickupLng,
		Address:   pickupAddr,
		Timestamp: pickupTime,
	}

	trip.DropoffLocation = models.Location{
		Latitude:  dropoffLat,
		Longitude: dropoffLng,
		Address:   dropoffAddr,
		Timestamp: dropoffTime,
	}

	// Set timestamps if present
	if matchedAt.Valid {
		trip.MatchedAt = &matchedAt.Time
	}
	if acceptedAt.Valid {
		trip.AcceptedAt = &acceptedAt.Time
	}
	if startedAt.Valid {
		trip.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		trip.CompletedAt = &completedAt.Time
	}
	if cancelledAt.Valid {
		trip.CancelledAt = &cancelledAt.Time
	}

	// Set fare
	trip.Fare = &models.Fare{
		BaseFare:     baseFare,
		DistanceFare: distanceFare,
		DurationFare: durationFare,
		SurgeFactor:  surgeFactor,
		TotalFare:    totalFare,
		Currency:     currency,
	}

	// Set ratings if present
	if passengerRating.Valid {
		trip.PassengerRating = &passengerRating.Float64
	}
	if driverRating.Valid {
		trip.DriverRating = &driverRating.Float64
	}

	return trip, nil
}

// UpdateRideStatus updates the status of a ride
func (r *RideRepo) UpdateRideStatus(ctx context.Context, tripID string, status models.TripStatus) error {
	// SQL query to update ride status
	query := `UPDATE trips SET status = $1 WHERE id = $2`

	// Execute the query
	_, err := r.db.ExecContext(ctx, query, status, tripID)
	return err
}

// UpdateRideTimestamp updates a timestamp field of a ride
func (r *RideRepo) UpdateRideTimestamp(ctx context.Context, tripID string, field string, timestamp time.Time) error {
	// Map field name to column name
	columnMap := map[string]string{
		"matched_at":   "matched_at",
		"accepted_at":  "accepted_at",
		"started_at":   "started_at",
		"completed_at": "completed_at",
		"cancelled_at": "cancelled_at",
	}

	column, ok := columnMap[field]
	if !ok {
		return fmt.Errorf("invalid timestamp field: %s", field)
	}

	// SQL query to update timestamp
	query := fmt.Sprintf("UPDATE trips SET %s = $1 WHERE id = $2", column)

	// Execute the query
	_, err := r.db.ExecContext(ctx, query, timestamp, tripID)
	return err
}

// GetActiveRideByPassengerID retrieves the active ride for a passenger
func (r *RideRepo) GetActiveRideByPassengerID(ctx context.Context, passengerID string) (*models.Trip, error) {
	// SQL query to get active ride for passenger
	query := `
		SELECT id FROM trips
		WHERE passenger_id = $1
		AND status IN ('REQUESTED', 'MATCHED', 'ACCEPTED', 'IN_PROGRESS')
		ORDER BY requested_at DESC
		LIMIT 1
	`

	// Execute the query
	var tripID string
	err := r.db.QueryRowContext(ctx, query, passengerID).Scan(&tripID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No active ride
		}
		return nil, err
	}

	// Get the full ride details
	return r.GetRideByID(ctx, tripID)
}

// GetActiveRideByDriverID retrieves the active ride for a driver
func (r *RideRepo) GetActiveRideByDriverID(ctx context.Context, driverID string) (*models.Trip, error) {
	// SQL query to get active ride for driver
	query := `
		SELECT id FROM trips
		WHERE driver_id = $1
		AND status IN ('ACCEPTED', 'IN_PROGRESS')
		ORDER BY accepted_at DESC
		LIMIT 1
	`

	// Execute the query
	var tripID string
	err := r.db.QueryRowContext(ctx, query, driverID).Scan(&tripID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No active ride
		}
		return nil, err
	}

	// Get the full ride details
	return r.GetRideByID(ctx, tripID)
}

// GetRideHistory retrieves the ride history for a user
func (r *RideRepo) GetRideHistory(ctx context.Context, userID string, role string, startTime, endTime time.Time) ([]*models.Trip, error) {
	// SQL query to get ride history
	var query string
	if role == "passenger" {
		query = `
			SELECT id FROM trips
			WHERE passenger_id = $1
			AND requested_at BETWEEN $2 AND $3
			ORDER BY requested_at DESC
		`
	} else if role == "driver" {
		query = `
			SELECT id FROM trips
			WHERE driver_id = $1
			AND requested_at BETWEEN $2 AND $3
			ORDER BY requested_at DESC
		`
	} else {
		return nil, fmt.Errorf("invalid role: %s", role)
	}

	// Execute the query
	rows, err := r.db.QueryContext(ctx, query, userID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Parse the results
	var tripIDs []string
	for rows.Next() {
		var tripID string
		if err := rows.Scan(&tripID); err != nil {
			return nil, err
		}
		tripIDs = append(tripIDs, tripID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Get the full ride details for each trip
	trips := make([]*models.Trip, 0, len(tripIDs))
	for _, tripID := range tripIDs {
		trip, err := r.GetRideByID(ctx, tripID)
		if err != nil {
			return nil, err
		}
		trips = append(trips, trip)
	}

	return trips, nil
}

// UpdateRideFare updates the fare for a ride
func (r *RideRepo) UpdateRideFare(ctx context.Context, tripID string, fare *models.Fare) error {
	// SQL query to update fare
	query := `
		UPDATE trips
		SET base_fare = $1, distance_fare = $2, duration_fare = $3, surge_factor = $4, total_fare = $5, currency = $6
		WHERE id = $7
	`

	// Execute the query
	_, err := r.db.ExecContext(
		ctx,
		query,
		fare.BaseFare,
		fare.DistanceFare,
		fare.DurationFare,
		fare.SurgeFactor,
		fare.TotalFare,
		fare.Currency,
		tripID,
	)

	return err
}

// UpdateRideRating updates the rating for a ride
func (r *RideRepo) UpdateRideRating(ctx context.Context, tripID string, role string, rating float64) error {
	// SQL query to update rating
	var query string
	if role == "passenger" {
		query = `UPDATE trips SET passenger_rating = $1 WHERE id = $2`
	} else if role == "driver" {
		query = `UPDATE trips SET driver_rating = $1 WHERE id = $2`
	} else {
		return fmt.Errorf("invalid role: %s", role)
	}

	// Execute the query
	_, err := r.db.ExecContext(ctx, query, rating, tripID)
	return err
}
