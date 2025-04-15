package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	log.Println("Initializing match repository")
	return &MatchRepo{
		cfg:         cfg,
		db:          db,
		redisClient: redisClient,
	}
}

// CreateMatch creates a new match (trip) in the database
func (r *MatchRepo) CreateMatch(ctx context.Context, match *models.Match) (*models.Match, error) {
	// Generate UUID if not provided
	if match.ID == "" {
		match.ID = uuid.New().String()
	}

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
	query := `
		INSERT INTO matches (
			id, driver_id, passenger_id, 
			driver_location, passenger_location,
			status, created_at, updated_at
		) VALUES (
			:id, :driver_id, :passenger_id,
			point(:driver_longitude, :driver_latitude), 
			point(:passenger_longitude, :passenger_latitude),
			:status, :created_at, :updated_at
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
			status, created_at, updated_at
		FROM matches
		WHERE id = $1
	`

	var dto models.MatchDTO
	err := r.db.QueryRowContext(ctx, query, matchID).Scan(
		&dto.ID, &dto.DriverID, &dto.PassengerID,
		&dto.DriverLongitude, &dto.DriverLatitude,
		&dto.PassengerLongitude, &dto.PassengerLatitude,
		&dto.Status, &dto.CreatedAt, &dto.UpdatedAt,
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
	// Store in geo set
	err := r.redisClient.GeoAdd(ctx, constants.KeyDriverGeo, location.Longitude, location.Latitude, driverID)
	if err != nil {
		return fmt.Errorf("failed to add driver location: %w", err)
	}

	// Add to available drivers set
	err = r.redisClient.SAdd(ctx, constants.KeyAvailableDrivers, driverID)
	if err != nil {
		return fmt.Errorf("failed to add driver to available set: %w", err)
	}

	// Store individual driver location
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

	nearbyDrivers := make([]*models.NearbyUser, 0, len(results))
	for _, result := range results {
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

// AddAvailablePassenger adds a passenger to the Redis geospatial index
func (r *MatchRepo) AddAvailablePassenger(ctx context.Context, passengerID string, location *models.Location) error {
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
	return nil
}

// RemoveAvailablePassenger removes a passenger from the Redis geospatial index
func (r *MatchRepo) RemoveAvailablePassenger(ctx context.Context, passengerID string) error {
	err := r.redisClient.Delete(ctx, fmt.Sprintf("%s:%s", constants.KeyPassengerGeo, passengerID))
	if err != nil {
		return fmt.Errorf("failed to remove passenger from geo index: %w", err)
	}
	return nil
}

// FindNearbyPassengers finds available passengers within the specified radius
func (r *MatchRepo) FindNearbyPassengers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.NearbyUser, error) {
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

	nearbyPassengers := make([]*models.NearbyUser, 0, len(results))
	for _, result := range results {
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
            status, created_at, updated_at
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
			&dto.Status, &dto.CreatedAt, &dto.UpdatedAt,
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
func (r *MatchRepo) ListMatchesByPassenger(ctx context.Context, passengerID string) ([]*models.Match, error) {
	query := `
        SELECT 
            id, driver_id, passenger_id,
            (driver_location[0])::float8 as driver_longitude,
            (driver_location[1])::float8 as driver_latitude,
            (passenger_location[0])::float8 as passenger_longitude,
            (passenger_location[1])::float8 as passenger_latitude,
            status, created_at, updated_at
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
			&dto.Status, &dto.CreatedAt, &dto.UpdatedAt,
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
