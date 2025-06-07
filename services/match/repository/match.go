package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	_ "github.com/newrelic/go-agent/v3/integrations/nrpq"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// MatchRepo implements the match repository interface
type MatchRepo struct {
	cfg         *models.Config
	db          *sqlx.DB
	redisClient *database.RedisClient
}

// NewMatchRepository creates a new match repository
func NewMatchRepository(
	cfg *models.Config,
	db *sqlx.DB,
	redisClient *database.RedisClient,
) *MatchRepo {
	return &MatchRepo{
		cfg:         cfg,
		db:          db,
		redisClient: redisClient,
	}
}

// checkExistingPendingMatch checks if a pending match already exists between driver and passenger
func (r *MatchRepo) checkExistingPendingMatch(ctx context.Context, driverID, passengerID uuid.UUID) (*models.Match, error) {
	query := `
		SELECT 
			id, driver_id, passenger_id,
			(driver_location[0])::float8 as driver_longitude,
			(driver_location[1])::float8 as driver_latitude,
			(passenger_location[0])::float8 as passenger_longitude,
			(passenger_location[1])::float8 as passenger_latitude,
			status, driver_confirmed, passenger_confirmed,
			created_at, updated_at
		FROM matches
		WHERE driver_id = $1 AND passenger_id = $2 AND status = $3
	`

	var dto models.MatchDTO
	err := r.db.QueryRowContext(ctx, query, driverID, passengerID, models.MatchStatusPending).Scan(
		&dto.ID, &dto.DriverID, &dto.PassengerID,
		&dto.DriverLongitude, &dto.DriverLatitude,
		&dto.PassengerLongitude, &dto.PassengerLatitude,
		&dto.Status, &dto.DriverConfirmed, &dto.PassengerConfirmed,
		&dto.CreatedAt, &dto.UpdatedAt,
	)

	if err != nil {
		return nil, err // Return error to caller to check if it's sql.ErrNoRows
	}

	return dto.ToMatch(), nil
}

// insertMatch inserts a new match into the database
func (r *MatchRepo) insertMatch(ctx context.Context, match *models.Match) error {
	dto := match.ToDTO()

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	insertQuery := `
		INSERT INTO matches (
			id, driver_id, passenger_id, 
			driver_location, passenger_location, target_location,
			status, driver_confirmed, passenger_confirmed,
			created_at, updated_at
		) VALUES (
			:id, :driver_id, :passenger_id,
			point(:driver_longitude, :driver_latitude), 
			point(:passenger_longitude, :passenger_latitude),
			point(:target_longitude, :target_latitude),
			:status, :driver_confirmed, :passenger_confirmed,
			:created_at, :updated_at
		)
	`
	_, err = tx.NamedExecContext(ctx, insertQuery, dto)
	if err != nil {
		return fmt.Errorf("failed to insert match: %w", err)
	}

	return tx.Commit()
}

// CreateMatch creates a new match in the database
func (r *MatchRepo) CreateMatch(ctx context.Context, match *models.Match) (*models.Match, error) {
	// Check for existing pending match
	existingMatch, err := r.checkExistingPendingMatch(ctx, match.DriverID, match.PassengerID)
	if err == nil {
		return existingMatch, nil // Return existing match
	}

	// Set up new match
	match.ID = uuid.New()
	now := time.Now()
	if match.CreatedAt.IsZero() {
		match.CreatedAt = now
	}
	match.UpdatedAt = now
	if match.Status == "" {
		match.Status = models.MatchStatusPending
	}

	if err := r.insertMatch(ctx, match); err != nil {
		return nil, err
	}

	return match, nil
}

// GetMatch retrieves a match by ID
func (r *MatchRepo) GetMatch(ctx context.Context, matchID string) (*models.Match, error) {
	// Get New Relic transaction from context for database instrumentation
	txn := newrelic.FromContext(ctx)
	dbCtx := newrelic.NewContext(ctx, txn)

	query := `
		SELECT
			id, driver_id, passenger_id,
			(driver_location[0])::float8 as driver_longitude,
			(driver_location[1])::float8 as driver_latitude,
			(passenger_location[0])::float8 as passenger_longitude,
			(passenger_location[1])::float8 as passenger_latitude,
			(target_location[0])::float8 as target_longitude,
			(target_location[1])::float8 as target_latitude,
			status, driver_confirmed, passenger_confirmed,
			created_at, updated_at
		FROM matches
		WHERE id = $1
	`

	var dto models.MatchDTO
	err := r.db.QueryRowContext(dbCtx, query, matchID).Scan(
		&dto.ID, &dto.DriverID, &dto.PassengerID,
		&dto.DriverLongitude, &dto.DriverLatitude,
		&dto.PassengerLongitude, &dto.PassengerLatitude,
		&dto.TargetLongitude, &dto.TargetLatitude,
		&dto.Status, &dto.DriverConfirmed, &dto.PassengerConfirmed,
		&dto.CreatedAt, &dto.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get match: %w", err)
	}

	return dto.ToMatch(), nil
}

// UpdateMatchStatus updates the status of a match
func (r *MatchRepo) UpdateMatchStatus(ctx context.Context, matchID string, status models.MatchStatus) error {
	// First, verify the match exists
	selectQuery := `
		SELECT 
			id, driver_id, passenger_id,
			(driver_location[0])::float8 as driver_longitude,
			(driver_location[1])::float8 as driver_latitude,
			(passenger_location[0])::float8 as passenger_longitude,
			(passenger_location[1])::float8 as passenger_latitude,
			status, created_at, updated_at
		FROM matches
		WHERE id = $1
	`

	var dto models.MatchDTO
	err := r.db.QueryRowContext(ctx, selectQuery, matchID).Scan(
		&dto.ID, &dto.DriverID, &dto.PassengerID,
		&dto.DriverLongitude, &dto.DriverLatitude,
		&dto.PassengerLongitude, &dto.PassengerLatitude,
		&dto.Status, &dto.CreatedAt, &dto.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to get match: %w", err)
	}

	// Begin transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update the match status
	updateQuery := `UPDATE matches SET status = $1, updated_at = $2 WHERE id = $3`
	result, err := tx.ExecContext(ctx, updateQuery, status, time.Now(), matchID)
	if err != nil {
		return fmt.Errorf("failed to update match status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("match not found")
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// validateUserForMatch validates that the user is part of the match
func (r *MatchRepo) validateUserForMatch(match *models.Match, userID string, isDriver bool) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}

	if isDriver && match.DriverID != userUUID {
		return fmt.Errorf("user is not the driver for this match")
	}
	if !isDriver && match.PassengerID != userUUID {
		return fmt.Errorf("user is not the passenger for this match")
	}

	return nil
}

// updateMatchConfirmationFlags updates the confirmation flags and determines new status
func (r *MatchRepo) updateMatchConfirmationFlags(match *models.Match, isDriver bool) (models.MatchStatus, error) {
	// Check if already confirmed
	if isDriver && match.DriverConfirmed {
		return "", fmt.Errorf("driver has already confirmed this match")
	}
	if !isDriver && match.PassengerConfirmed {
		return "", fmt.Errorf("passenger has already confirmed this match")
	}

	// Update confirmation flags
	if isDriver {
		match.DriverConfirmed = true
	} else {
		match.PassengerConfirmed = true
	}

	// Determine new status
	if match.DriverConfirmed && match.PassengerConfirmed {
		return models.MatchStatusAccepted, nil
	} else if match.DriverConfirmed {
		return models.MatchStatusDriverConfirmed, nil
	} else if match.PassengerConfirmed {
		return models.MatchStatusPassengerConfirmed, nil
	}

	return models.MatchStatusPending, nil
}

// ConfirmMatchByUser handles confirmation by either driver or passenger
func (r *MatchRepo) ConfirmMatchByUser(ctx context.Context, matchID string, userID string, isDriver bool) (*models.Match, error) {
	// Get New Relic transaction from context for database instrumentation
	txn := newrelic.FromContext(ctx)
	dbCtx := newrelic.NewContext(ctx, txn)

	tx, err := r.db.BeginTxx(dbCtx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current match with lock
	query := `
		SELECT
			id, driver_id, passenger_id,
			(driver_location[0])::float8 as driver_longitude,
			(driver_location[1])::float8 as driver_latitude,
			(passenger_location[0])::float8 as passenger_longitude,
			(passenger_location[1])::float8 as passenger_latitude,
			(target_location[0])::float8 as target_longitude,
			(target_location[1])::float8 as target_latitude,
			status, driver_confirmed, passenger_confirmed,
			created_at, updated_at
		FROM matches
		WHERE id = $1
		FOR UPDATE
	`

	var dto models.MatchDTO
	err = tx.QueryRowContext(dbCtx, query, matchID).Scan(
		&dto.ID, &dto.DriverID, &dto.PassengerID,
		&dto.DriverLongitude, &dto.DriverLatitude,
		&dto.PassengerLongitude, &dto.PassengerLatitude,
		&dto.TargetLongitude, &dto.TargetLatitude,
		&dto.Status, &dto.DriverConfirmed, &dto.PassengerConfirmed,
		&dto.CreatedAt, &dto.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get match: %w", err)
	}

	match := dto.ToMatch()

	// Validate user and match state
	if err := r.validateUserForMatch(match, userID, isDriver); err != nil {
		return nil, err
	}

	if match.Status != models.MatchStatusPending &&
		match.Status != models.MatchStatusDriverConfirmed &&
		match.Status != models.MatchStatusPassengerConfirmed {
		return nil, fmt.Errorf("match cannot be confirmed: current status is %s", match.Status)
	}

	// Update confirmation flags and get new status
	newStatus, err := r.updateMatchConfirmationFlags(match, isDriver)
	if err != nil {
		return nil, err
	}

	// Update match in database
	match.Status = newStatus
	match.UpdatedAt = time.Now()
	updatedDTO := match.ToDTO()

	updateQuery := `
		UPDATE matches
		SET status = :status,
		    driver_confirmed = :driver_confirmed,
		    passenger_confirmed = :passenger_confirmed,
		    updated_at = :updated_at
		WHERE id = :id
	`

	result, err := tx.NamedExecContext(dbCtx, updateQuery, updatedDTO)
	if err != nil {
		return nil, fmt.Errorf("failed to update match: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return nil, fmt.Errorf("match not found or was modified by another transaction")
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return match, nil
}

// ListMatchesByPassenger retrieves all matches for a passenger
func (r *MatchRepo) ListMatchesByPassenger(ctx context.Context, passengerID uuid.UUID) ([]*models.Match, error) {
	query := `
        SELECT 
            id, driver_id, passenger_id,
            (driver_location[0])::float8 as driver_longitude,
            (driver_location[1])::float8 as driver_latitude,
            (passenger_location[0])::float8 as passenger_longitude,
            (passenger_location[1])::float8 as passenger_latitude,
            (target_location[0])::float8 as target_longitude,
            (target_location[1])::float8 as target_latitude,
            status, driver_confirmed, passenger_confirmed,
            created_at, updated_at
        FROM matches
        WHERE passenger_id = $1
        ORDER BY created_at DESC
    `

	rows, err := r.db.QueryContext(ctx, query, passengerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list matches: %w", err)
	}
	defer rows.Close()

	var matches []*models.Match
	for rows.Next() {
		var dto models.MatchDTO
		err := rows.Scan(
			&dto.ID, &dto.DriverID, &dto.PassengerID,
			&dto.DriverLongitude, &dto.DriverLatitude,
			&dto.PassengerLongitude, &dto.PassengerLatitude,
			&dto.TargetLongitude, &dto.TargetLatitude,
			&dto.Status, &dto.DriverConfirmed, &dto.PassengerConfirmed,
			&dto.CreatedAt, &dto.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan match: %w", err)
		}

		matches = append(matches, dto.ToMatch())
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating matches: %w", err)
	}

	return matches, nil
}

// BatchUpdateMatchStatus updates the status of multiple matches
func (r *MatchRepo) BatchUpdateMatchStatus(ctx context.Context, matchIDs []string, status models.MatchStatus) error {
	if len(matchIDs) == 0 {
		return nil
	}

	// Convert string IDs to UUIDs for the query
	uuidIDs := make([]uuid.UUID, len(matchIDs))
	for i, id := range matchIDs {
		parsedUUID, err := uuid.Parse(id)
		if err != nil {
			return fmt.Errorf("invalid match ID format: %s", id)
		}
		uuidIDs[i] = parsedUUID
	}

	// Use SQL IN clause for efficient batch update
	query := `
		UPDATE matches 
		SET status = $1, updated_at = $2 
		WHERE id = ANY($3) AND status IN ('PENDING', 'DRIVER_CONFIRMED', 'PASSENGER_CONFIRMED')
	`

	result, err := r.db.ExecContext(ctx, query, status, time.Now(), pq.Array(uuidIDs))
	if err != nil {
		return fmt.Errorf("failed to batch update match statuses: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	logger.Info("Batch updated matches",
		logger.Int64("rows_affected", rowsAffected),
		logger.String("status", string(status)))
	return nil
}

// SetActiveRide stores active ride information for both driver and passenger
func (r *MatchRepo) SetActiveRide(ctx context.Context, driverID, passengerID, rideID string) error {
	txn := newrelic.FromContext(ctx)
	redisCtx := newrelic.NewContext(ctx, txn)

	// Get TTL from config, default to 24 hours if not configured
	ttlHours := 24
	if r.cfg != nil && r.cfg.Match.ActiveRideTTLHours > 0 {
		ttlHours = r.cfg.Match.ActiveRideTTLHours
	}
	ttl := time.Duration(ttlHours) * time.Hour

	// Set active ride for driver with TTL
	driverKey := fmt.Sprintf(constants.KeyActiveRideDriver, driverID)
	if err := r.redisClient.Set(redisCtx, driverKey, rideID, ttl); err != nil {
		return fmt.Errorf("failed to set active ride for driver: %w", err)
	}

	// Set active ride for passenger with TTL
	passengerKey := fmt.Sprintf(constants.KeyActiveRidePassenger, passengerID)
	if err := r.redisClient.Set(redisCtx, passengerKey, rideID, ttl); err != nil {
		return fmt.Errorf("failed to set active ride for passenger: %w", err)
	}

	logger.Info("Set active ride",
		logger.String("ride_id", rideID),
		logger.String("driver_id", driverID),
		logger.String("passenger_id", passengerID),
		logger.String("ttl", ttl.String()))
	return nil
}

// RemoveActiveRide removes active ride information for both driver and passenger
func (r *MatchRepo) RemoveActiveRide(ctx context.Context, driverID, passengerID string) error {
	txn := newrelic.FromContext(ctx)
	redisCtx := newrelic.NewContext(ctx, txn)

	// Remove active ride for driver
	driverKey := fmt.Sprintf(constants.KeyActiveRideDriver, driverID)
	if err := r.redisClient.Delete(redisCtx, driverKey); err != nil {
		logger.Warn("Failed to remove active ride for driver",
			logger.String("driver_id", driverID),
			logger.ErrorField(err))
		// Continue with passenger cleanup even if driver cleanup fails
	}

	// Remove active ride for passenger
	passengerKey := fmt.Sprintf(constants.KeyActiveRidePassenger, passengerID)
	if err := r.redisClient.Delete(redisCtx, passengerKey); err != nil {
		logger.Warn("Failed to remove active ride for passenger",
			logger.String("passenger_id", passengerID),
			logger.ErrorField(err))
		// Don't return error for cleanup operations
	}

	logger.Info("Removed active ride",
		logger.String("driver_id", driverID),
		logger.String("passenger_id", passengerID))
	return nil
}

// GetActiveRideByDriver retrieves the active ride ID for a driver
func (r *MatchRepo) GetActiveRideByDriver(ctx context.Context, driverID string) (string, error) {
	// Get New Relic transaction from context for Redis instrumentation
	txn := newrelic.FromContext(ctx)
	redisCtx := newrelic.NewContext(ctx, txn)

	driverKey := fmt.Sprintf(constants.KeyActiveRideDriver, driverID)
	rideID, err := r.redisClient.Get(redisCtx, driverKey)
	if err != nil {
		// If key doesn't exist, it's not an error - just means no active ride
		if err == redis.Nil {
			return "", nil
		}
		return "", fmt.Errorf("failed to get active ride for driver: %w", err)
	}
	return rideID, nil
}

// GetActiveRideByPassenger retrieves the active ride ID for a passenger
func (r *MatchRepo) GetActiveRideByPassenger(ctx context.Context, passengerID string) (string, error) {
	// Get New Relic transaction from context for Redis instrumentation
	txn := newrelic.FromContext(ctx)
	redisCtx := newrelic.NewContext(ctx, txn)

	passengerKey := fmt.Sprintf(constants.KeyActiveRidePassenger, passengerID)
	rideID, err := r.redisClient.Get(redisCtx, passengerKey)
	if err != nil {
		// If key doesn't exist, it's not an error - just means no active ride
		if err == redis.Nil {
			return "", nil
		}
		return "", fmt.Errorf("failed to get active ride for passenger: %w", err)
	}
	return rideID, nil
}
