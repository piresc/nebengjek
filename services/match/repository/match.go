package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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

// CreateMatch creates a new match (trip) in the database
// If a pending match already exists with the same driver and passenger, returns the existing match
func (r *MatchRepo) CreateMatch(ctx context.Context, match *models.Match) (*models.Match, error) {
	// First check if a pending match exists with the same driver and passenger
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

	var existingDTO models.MatchDTO
	err := r.db.QueryRowContext(ctx, query, match.DriverID, match.PassengerID, models.MatchStatusPending).Scan(
		&existingDTO.ID, &existingDTO.DriverID, &existingDTO.PassengerID,
		&existingDTO.DriverLongitude, &existingDTO.DriverLatitude,
		&existingDTO.PassengerLongitude, &existingDTO.PassengerLatitude,
		&existingDTO.Status, &existingDTO.DriverConfirmed, &existingDTO.PassengerConfirmed,
		&existingDTO.CreatedAt, &existingDTO.UpdatedAt,
	)

	if err == nil {
		// A pending match already exists
		return existingDTO.ToMatch(), nil
	}

	// No existing match found, create a new one
	match.ID = uuid.New()

	// Set timestamps
	now := time.Now()
	if match.CreatedAt.IsZero() {
		match.CreatedAt = now
	}
	match.UpdatedAt = now
	if match.Status == "" {
		match.Status = models.MatchStatusPending
	}

	// Create DTO for database operation
	dto := match.ToDTO()

	// Begin transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert match
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
		return nil, fmt.Errorf("failed to insert match: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
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

	// Convert DTO to Match
	return dto.ToMatch(), nil
}

// UpdateMatchStatus updates the status of a match
func (r *MatchRepo) UpdateMatchStatus(ctx context.Context, matchID string, status models.MatchStatus) error {
	// Get current match to update the DTO
	match, err := r.GetMatch(ctx, matchID)
	if err != nil {
		return fmt.Errorf("failed to get match: %w", err)
	}

	// Update status and timestamp
	match.Status = status
	match.UpdatedAt = time.Now()
	dto := match.ToDTO()

	// Begin transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update using DTO
	query := `UPDATE matches SET status = :status, updated_at = :updated_at WHERE id = :id`
	result, err := tx.NamedExecContext(ctx, query, dto)
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
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// AddAvailableDriver adds a driver to the available drivers geo set
func (r *MatchRepo) AddAvailableDriver(ctx context.Context, driverID string, location *models.Location) error {
	// Store in geo set (always update location)
	err := r.redisClient.GeoAdd(ctx, constants.KeyDriverGeo, location.Longitude, location.Latitude, driverID)
	if err != nil {
		return fmt.Errorf("failed to add/update driver location: %w", err)
	}

	// Add to available drivers set (if not already there)
	// Redis SADD command only adds elements that don't already exist in the set
	err = r.redisClient.SAdd(ctx, constants.KeyAvailableDrivers, driverID)
	if err != nil {
		return fmt.Errorf("failed to add driver to available set: %w", err)
	}

	// Store individual driver location (always update this regardless of previous membership)
	locationKey := fmt.Sprintf(constants.KeyDriverLocation, driverID)
	locationData := map[string]interface{}{
		constants.FieldLatitude:  location.Latitude,
		constants.FieldLongitude: location.Longitude,
		constants.FieldTimestamp: time.Now().Unix(),
	}
	err = r.redisClient.HMSet(ctx, locationKey, locationData)
	if err != nil {
		return fmt.Errorf("failed to store driver location: %w", err)
	}

	return nil
}

// RemoveAvailableDriver removes a driver from the available drivers sets
func (r *MatchRepo) RemoveAvailableDriver(ctx context.Context, driverID string) error {
	// Remove from geo set
	err := r.redisClient.ZRem(ctx, constants.KeyDriverGeo, driverID)
	if err != nil {
		return fmt.Errorf("failed to remove driver location: %w", err)
	}

	// Remove from available set
	err = r.redisClient.SRem(ctx, constants.KeyAvailableDrivers, driverID)
	if err != nil {
		return fmt.Errorf("failed to remove driver from available set: %w", err)
	}

	// Remove individual location
	locationKey := fmt.Sprintf(constants.KeyDriverLocation, driverID)
	err = r.redisClient.Delete(ctx, locationKey)
	if err != nil {
		return fmt.Errorf("failed to remove driver location data: %w", err)
	}

	return nil
}

// StoreMatchProposal stores a match proposal in Redis
func (r *MatchRepo) StoreMatchProposal(ctx context.Context, match *models.Match) error {
	matchData, err := json.Marshal(match)
	if err != nil {
		return fmt.Errorf("failed to marshal match data: %w", err)
	}

	key := fmt.Sprintf(constants.KeyMatchProposal, match.ID)
	err = r.redisClient.Set(ctx, key, matchData, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to store match proposal: %w", err)
	}

	// Store references for both driver and passenger
	driverKey := fmt.Sprintf(constants.KeyDriverMatch, match.DriverID)
	passengerKey := fmt.Sprintf(constants.KeyPassengerMatch, match.PassengerID)

	err = r.redisClient.Set(ctx, driverKey, match.ID, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to store driver match reference: %w", err)
	}

	err = r.redisClient.Set(ctx, passengerKey, match.ID, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to store passenger match reference: %w", err)
	}

	return nil
}

// FindNearbyDrivers finds available drivers within the specified radius
func (r *MatchRepo) FindNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.NearbyUser, error) {
	// Get drivers within radius from geo index
	results, err := r.redisClient.GeoRadius(
		ctx,
		constants.KeyDriverGeo,
		location.Longitude,
		location.Latitude,
		radiusKm,
		"km",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby drivers: %w", err)
	}

	// Filter to keep only available drivers
	nearbyDrivers := make([]*models.NearbyUser, 0, len(results))
	for _, result := range results {
		// Check if this driver is in the available set
		isMember, err := r.redisClient.SIsMember(ctx, constants.KeyAvailableDrivers, result.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to check driver availability: %w", err)
		}

		// Only include available drivers
		if isMember {
			nearbyDrivers = append(nearbyDrivers, &models.NearbyUser{
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

	return nearbyDrivers, nil
}

// ProcessLocationUpdate processes a location update for a driver
func (r *MatchRepo) ProcessLocationUpdate(ctx context.Context, driverID string, location *models.Location) error {
	// Update driver's location in Redis
	err := r.redisClient.GeoAdd(
		ctx,
		constants.KeyDriverGeo,
		location.Longitude,
		location.Latitude,
		driverID,
	)
	if err != nil {
		return fmt.Errorf("failed to update driver location: %w", err)
	}
	return nil
}

// ProcessPassengerLocationUpdate processes a location update for a passenger
func (r *MatchRepo) ProcessPassengerLocationUpdate(ctx context.Context, passengerID string, location *models.Location) error {
	// Update passenger's location in Redis
	err := r.redisClient.GeoAdd(
		ctx,
		constants.KeyPassengerGeo,
		location.Longitude,
		location.Latitude,
		passengerID,
	)
	if err != nil {
		return fmt.Errorf("failed to update passenger location: %w", err)
	}
	return nil
}

// AddAvailablePassenger adds a passenger to the Redis geospatial index
func (r *MatchRepo) AddAvailablePassenger(ctx context.Context, passengerID string, location *models.Location) error {
	// Store in geo set (always update location)
	err := r.redisClient.GeoAdd(
		ctx,
		constants.KeyPassengerGeo,
		location.Longitude,
		location.Latitude,
		passengerID,
	)
	if err != nil {
		return fmt.Errorf("failed to add passenger to geo index: %w", err)
	}

	// Add to available passengers set (if not already there)
	// Redis SADD command only adds elements that don't already exist in the set
	err = r.redisClient.SAdd(ctx, constants.KeyAvailablePassengers, passengerID)
	if err != nil {
		return fmt.Errorf("failed to add passenger to available set: %w", err)
	}

	// Store individual passenger location
	locationKey := fmt.Sprintf(constants.KeyPassengerLocation, passengerID)
	locationData := map[string]interface{}{
		constants.FieldLatitude:  location.Latitude,
		constants.FieldLongitude: location.Longitude,
		constants.FieldTimestamp: time.Now().Unix(),
	}
	err = r.redisClient.HMSet(ctx, locationKey, locationData)
	if err != nil {
		return fmt.Errorf("failed to store passenger location: %w", err)
	}

	return nil
}

// RemoveAvailablePassenger removes a passenger from the Redis geospatial index
func (r *MatchRepo) RemoveAvailablePassenger(ctx context.Context, passengerID string) error {
	// Remove from passenger geo set
	err := r.redisClient.ZRem(ctx, constants.KeyPassengerGeo, passengerID)
	if err != nil {
		return fmt.Errorf("failed to remove passenger from geo index: %w", err)
	}

	// Remove from available set
	err = r.redisClient.SRem(ctx, constants.KeyAvailablePassengers, passengerID)
	if err != nil {
		return fmt.Errorf("failed to remove passenger from available set: %w", err)
	}

	// Remove individual location
	locationKey := fmt.Sprintf(constants.KeyPassengerLocation, passengerID)
	err = r.redisClient.Delete(ctx, locationKey)
	if err != nil {
		return fmt.Errorf("failed to remove passenger location data: %w", err)
	}

	return nil
}

// FindNearbyPassengers finds available passengers within the specified radius
func (r *MatchRepo) FindNearbyPassengers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.NearbyUser, error) {
	// Get passengers within radius from geo index
	results, err := r.redisClient.GeoRadius(
		ctx,
		constants.KeyPassengerGeo,
		location.Longitude,
		location.Latitude,
		radiusKm,
		"km",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby passengers: %w", err)
	}

	// Filter to keep only available passengers
	nearbyPassengers := make([]*models.NearbyUser, 0, len(results))
	for _, result := range results {
		// Check if this passenger is in the available set
		isMember, err := r.redisClient.SIsMember(ctx, constants.KeyAvailablePassengers, result.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to check passenger availability: %w", err)
		}

		// Only include available passengers
		if isMember {
			nearbyPassengers = append(nearbyPassengers, &models.NearbyUser{
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

	return nearbyPassengers, nil
}

// ListMatchesByDriver retrieves all matches for a driver
func (r *MatchRepo) ListMatchesByDriver(ctx context.Context, driverID string) ([]*models.Match, error) {
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
        WHERE driver_id = $1
        ORDER BY created_at DESC
    `

	rows, err := r.db.QueryContext(ctx, query, driverID)
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

// ConfirmMatchAtomically updates match status atomically with optimistic locking
func (r *MatchRepo) ConfirmMatchAtomically(ctx context.Context, matchID string, status models.MatchStatus) error {
	// Begin transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current match status with FOR UPDATE lock
	var currentStatus string
	var driverID string
	err = tx.QueryRowContext(ctx, `
        SELECT status::text, driver_id
        FROM matches 
        WHERE id = $1 
        FOR UPDATE
    `, matchID).Scan(&currentStatus, &driverID)
	if err != nil {
		return fmt.Errorf("failed to get current match status: %w", err)
	}

	// Check if match can be confirmed
	if currentStatus != string(models.MatchStatusPending) {
		return fmt.Errorf("match cannot be confirmed: current status is %s", currentStatus)
	}

	// Update match status
	result, err := tx.ExecContext(ctx, `
        UPDATE matches 
        SET status = $1, updated_at = NOW() 
        WHERE id = $2 AND status = $3
    `, status, matchID, models.MatchStatusPending)
	if err != nil {
		return fmt.Errorf("failed to update match status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("match status was changed by another transaction")
	}

	// Remove both users from available pools
	if err := r.RemoveAvailableDriver(ctx, driverID); err != nil {
		return fmt.Errorf("failed to remove driver from available pool: %w", err)
	}

	// Get passenger ID to remove from available pool
	var passengerID string
	err = tx.QueryRowContext(ctx, "SELECT passenger_id FROM matches WHERE id = $1", matchID).Scan(&passengerID)
	if err != nil {
		return fmt.Errorf("failed to get passenger ID: %w", err)
	}

	if err := r.RemoveAvailablePassenger(ctx, passengerID); err != nil {
		return fmt.Errorf("failed to remove passenger from available pool: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// CreatePendingMatch creates a match proposal in Redis with SetNX to ensure uniqueness
// Returns the match ID if successful, empty string if a match already exists between the driver and passenger
func (r *MatchRepo) CreatePendingMatch(ctx context.Context, match *models.Match) (string, error) {
	// Generate a match ID
	if match.ID == uuid.Nil {
		match.ID = uuid.New()
	}

	matchID := match.ID.String()
	driverID := match.DriverID.String()
	passengerID := match.PassengerID.String()

	// Create a key for this specific driver-passenger pair
	pairKey := fmt.Sprintf(constants.KeyPendingMatchPair, driverID, passengerID)

	// Try to set the key only if it doesn't exist (SetNX)
	matchData, err := json.Marshal(match)
	if err != nil {
		return "", fmt.Errorf("failed to marshal match data: %w", err)
	}

	// Set with NX flag and 1 minute expiration
	wasSet, err := r.redisClient.SetNX(ctx, pairKey, matchData, 1*time.Minute)
	if err != nil {
		return "", fmt.Errorf("failed to create pending match: %w", err)
	}

	// If the key already exists, return empty string to indicate no match was created
	if !wasSet {
		return "", nil
	}

	// Also store by match ID for direct lookup
	matchKey := fmt.Sprintf(constants.KeyMatchProposal, matchID)
	err = r.redisClient.Set(ctx, matchKey, matchData, 1*time.Minute)
	if err != nil {
		// Clean up the main key if we can't set the reference
		r.redisClient.Delete(ctx, pairKey)
		return "", fmt.Errorf("failed to store match by ID: %w", err)
	}

	// Store references for both driver and passenger
	driverKey := fmt.Sprintf(constants.KeyDriverMatch, driverID)
	passengerKey := fmt.Sprintf(constants.KeyPassengerMatch, passengerID)

	// Store for 1 minute to match the pending match expiration
	err = r.redisClient.Set(ctx, driverKey, matchID, 1*time.Minute)
	if err != nil {
		// Clean up the main key if we can't set the reference
		r.redisClient.Delete(ctx, pairKey)
		r.redisClient.Delete(ctx, matchKey)
		return "", fmt.Errorf("failed to store driver match reference: %w", err)
	}

	err = r.redisClient.Set(ctx, passengerKey, matchID, 1*time.Minute)
	if err != nil {
		// Clean up the previously set keys if we can't set all references
		r.redisClient.Delete(ctx, pairKey)
		r.redisClient.Delete(ctx, matchKey)
		r.redisClient.Delete(ctx, driverKey)
		return "", fmt.Errorf("failed to store passenger match reference: %w", err)
	}

	return matchID, nil
}

// ConfirmAndPersistMatch retrieves a pending match from Redis and persists it to the database
func (r *MatchRepo) ConfirmAndPersistMatch(ctx context.Context, driverID, passengerID string) (*models.Match, error) {
	// Get the pending match from Redis
	pairKey := fmt.Sprintf(constants.KeyPendingMatchPair, driverID, passengerID)
	matchData, err := r.redisClient.Get(ctx, pairKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending match: %w", err)
	}

	var match models.Match
	if err := json.Unmarshal([]byte(matchData), &match); err != nil {
		return nil, fmt.Errorf("failed to unmarshal match data: %w", err)
	}

	// Now save it to the database
	// Start transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Set timestamps and status
	now := time.Now()
	if match.CreatedAt.IsZero() {
		match.CreatedAt = now
	}
	match.UpdatedAt = now
	match.Status = models.MatchStatusAccepted

	// Create DTO for database operation
	dto := match.ToDTO()

	// Insert match
	query := `
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
	_, err = tx.NamedExecContext(ctx, query, dto)
	if err != nil {
		return nil, fmt.Errorf("failed to insert match: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Clean up Redis keys
	r.redisClient.Delete(ctx, pairKey)

	return &match, nil
}

// ConfirmMatchByUser handles confirmation by either driver or passenger
// Sets the appropriate confirmation flag and updates status if both parties have confirmed
func (r *MatchRepo) ConfirmMatchByUser(ctx context.Context, matchID string, userID string, isDriver bool) (*models.Match, error) {
	fmt.Printf("ConfirmMatchByUser called with matchID: %s, userID: %s, isDriver: %t\n", matchID, userID, isDriver)

	// Begin transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current match with FOR UPDATE lock to prevent race conditions
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

	fmt.Printf("Executing query for matchID: %s\n", matchID)

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

	// Convert to match object
	match := dto.ToMatch()

	// Verify the user is part of this match
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %w (userID: %s)", err, userID)
	}

	if isDriver && match.DriverID != userUUID {
		return nil, fmt.Errorf("user is not the driver for this match (userID: %s, driverID: %s)", userID, match.DriverID.String())
	}
	if !isDriver && match.PassengerID != userUUID {
		return nil, fmt.Errorf("user is not the passenger for this match (userID: %s, passengerID: %s)", userID, match.PassengerID.String())
	}

	// Check if match is in a confirmable state
	fmt.Printf("Current match status: %s, Driver confirmed: %t, Passenger confirmed: %t\n",
		match.Status, match.DriverConfirmed, match.PassengerConfirmed)

	if match.Status != models.MatchStatusPending &&
		match.Status != models.MatchStatusDriverConfirmed &&
		match.Status != models.MatchStatusPassengerConfirmed {
		return nil, fmt.Errorf("match cannot be confirmed: current status is %s", match.Status)
	}

	// Update confirmation flags
	if isDriver {
		if match.DriverConfirmed {
			return nil, fmt.Errorf("driver has already confirmed this match")
		}
		match.DriverConfirmed = true
	} else {
		if match.PassengerConfirmed {
			return nil, fmt.Errorf("passenger has already confirmed this match")
		}
		match.PassengerConfirmed = true
	}

	// Determine new status based on confirmation state
	var newStatus models.MatchStatus
	if match.DriverConfirmed && match.PassengerConfirmed {
		newStatus = models.MatchStatusAccepted
		fmt.Printf("Both driver and passenger have confirmed, setting status to ACCEPTED\n")
	} else if match.DriverConfirmed {
		newStatus = models.MatchStatusDriverConfirmed
		fmt.Printf("Only driver has confirmed, setting status to DRIVER_CONFIRMED\n")
	} else if match.PassengerConfirmed {
		newStatus = models.MatchStatusPassengerConfirmed
		fmt.Printf("Only passenger has confirmed, setting status to PASSENGER_CONFIRMED\n")
	} else {
		newStatus = models.MatchStatusPending
		fmt.Printf("Neither party has confirmed, keeping status as PENDING\n")
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

	// If both parties have confirmed, remove them from available pools
	if match.Status == models.MatchStatusAccepted {
		if err := r.RemoveAvailableDriver(ctx, match.DriverID.String()); err != nil {
			return nil, fmt.Errorf("failed to remove driver from available pool: %w", err)
		}
		if err := r.RemoveAvailablePassenger(ctx, match.PassengerID.String()); err != nil {
			return nil, fmt.Errorf("failed to remove passenger from available pool: %w", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return match, nil
}

// DeleteRedisKey deletes a key from Redis
func (r *MatchRepo) DeleteRedisKey(ctx context.Context, key string) error {
	return r.redisClient.Delete(ctx, key)
}

// StoreIDMapping stores a mapping between two IDs in Redis with an expiration
func (r *MatchRepo) StoreIDMapping(ctx context.Context, key string, value string, expiration time.Duration) error {
	return r.redisClient.Set(ctx, key, value, expiration)
}

// GetIDMapping retrieves a mapping for a match ID
func (r *MatchRepo) GetIDMapping(ctx context.Context, originalID string) (string, error) {
	key := fmt.Sprintf("match:id_mapping:%s", originalID)
	return r.redisClient.Get(ctx, key)
}
