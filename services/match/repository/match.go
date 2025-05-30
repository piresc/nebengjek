package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/database"
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
	err := r.db.QueryRowContext(ctx, query, matchID).Scan(
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
	query := `UPDATE matches SET status = $1, updated_at = $2 WHERE id = $3`
	result, err := r.db.ExecContext(ctx, query, status, time.Now(), matchID)
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

	return nil
}

// addToRedisGeo adds a user to Redis geospatial index
func (r *MatchRepo) addToRedisGeo(ctx context.Context, geoKey, availableKey, locationKeyTemplate, userID string, location *models.Location) error {
	// Add to geo set
	if err := r.redisClient.GeoAdd(ctx, geoKey, location.Longitude, location.Latitude, userID); err != nil {
		return fmt.Errorf("failed to add to geo index: %w", err)
	}

	// Add to available set
	if err := r.redisClient.SAdd(ctx, availableKey, userID); err != nil {
		return fmt.Errorf("failed to add to available set: %w", err)
	}

	// Store individual location
	locationKey := fmt.Sprintf(locationKeyTemplate, userID)
	locationData := map[string]interface{}{
		constants.FieldLatitude:  location.Latitude,
		constants.FieldLongitude: location.Longitude,
		constants.FieldTimestamp: time.Now().Unix(),
	}
	if err := r.redisClient.HMSet(ctx, locationKey, locationData); err != nil {
		return fmt.Errorf("failed to store location: %w", err)
	}

	return nil
}

// removeFromRedisGeo removes a user from Redis geospatial index
func (r *MatchRepo) removeFromRedisGeo(ctx context.Context, geoKey, availableKey, locationKeyTemplate, userID string) error {
	// Remove from geo set
	if err := r.redisClient.ZRem(ctx, geoKey, userID); err != nil {
		return fmt.Errorf("failed to remove from geo index: %w", err)
	}

	// Remove from available set
	if err := r.redisClient.SRem(ctx, availableKey, userID); err != nil {
		return fmt.Errorf("failed to remove from available set: %w", err)
	}

	// Remove individual location
	locationKey := fmt.Sprintf(locationKeyTemplate, userID)
	if err := r.redisClient.Delete(ctx, locationKey); err != nil {
		return fmt.Errorf("failed to remove location data: %w", err)
	}

	return nil
}

// AddAvailableDriver adds a driver to the available drivers geo set
func (r *MatchRepo) AddAvailableDriver(ctx context.Context, driverID string, location *models.Location) error {
	return r.addToRedisGeo(ctx,
		constants.KeyDriverGeo,
		constants.KeyAvailableDrivers,
		constants.KeyDriverLocation,
		driverID,
		location)
}

// RemoveAvailableDriver removes a driver from the available drivers sets
func (r *MatchRepo) RemoveAvailableDriver(ctx context.Context, driverID string) error {
	return r.removeFromRedisGeo(ctx,
		constants.KeyDriverGeo,
		constants.KeyAvailableDrivers,
		constants.KeyDriverLocation,
		driverID)
}

// AddAvailablePassenger adds a passenger to the Redis geospatial index
func (r *MatchRepo) AddAvailablePassenger(ctx context.Context, passengerID string, location *models.Location) error {
	return r.addToRedisGeo(ctx,
		constants.KeyPassengerGeo,
		constants.KeyAvailablePassengers,
		constants.KeyPassengerLocation,
		passengerID,
		location)
}

// RemoveAvailablePassenger removes a passenger from the Redis geospatial index
func (r *MatchRepo) RemoveAvailablePassenger(ctx context.Context, passengerID string) error {
	return r.removeFromRedisGeo(ctx,
		constants.KeyPassengerGeo,
		constants.KeyAvailablePassengers,
		constants.KeyPassengerLocation,
		passengerID)
}

// findNearbyUsers finds available users within the specified radius
func (r *MatchRepo) findNearbyUsers(ctx context.Context, geoKey, availableKey string, location *models.Location, radiusKm float64) ([]*models.NearbyUser, error) {
	results, err := r.redisClient.GeoRadius(
		ctx,
		geoKey,
		location.Longitude,
		location.Latitude,
		radiusKm,
		"km",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby users: %w", err)
	}

	nearbyUsers := make([]*models.NearbyUser, 0, len(results))
	for _, result := range results {
		isMember, err := r.redisClient.SIsMember(ctx, availableKey, result.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to check user availability: %w", err)
		}

		if isMember {
			nearbyUsers = append(nearbyUsers, &models.NearbyUser{
				ID: result.Name,
				Location: models.Location{
					Latitude:  result.Latitude,
					Longitude: result.Longitude,
					Timestamp: time.Now(),
				},
				Distance: result.Dist,
			})
		}
	}

	return nearbyUsers, nil
}

// FindNearbyDrivers finds available drivers within the specified radius
func (r *MatchRepo) FindNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.NearbyUser, error) {
	return r.findNearbyUsers(ctx, constants.KeyDriverGeo, constants.KeyAvailableDrivers, location, radiusKm)
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
	tx, err := r.db.BeginTxx(ctx, nil)
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
	err = tx.QueryRowContext(ctx, query, matchID).Scan(
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

	result, err := tx.NamedExecContext(ctx, updateQuery, updatedDTO)
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

	// Remove from available pools if fully confirmed
	if match.Status == models.MatchStatusAccepted {
		if err := r.RemoveAvailableDriver(ctx, match.DriverID.String()); err != nil {
			return nil, fmt.Errorf("failed to remove driver from available pool: %w", err)
		}
		if err := r.RemoveAvailablePassenger(ctx, match.PassengerID.String()); err != nil {
			return nil, fmt.Errorf("failed to remove passenger from available pool: %w", err)
		}
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
		WHERE id = ANY($3) AND status IN ('pending', 'driver_confirmed', 'passenger_confirmed')
	`

	result, err := r.db.ExecContext(ctx, query, status, time.Now(), pq.Array(uuidIDs))
	if err != nil {
		return fmt.Errorf("failed to batch update match statuses: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	log.Printf("Batch updated %d matches to status %s", rowsAffected, status)
	return nil
}
