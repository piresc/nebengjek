package repository

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/stretchr/testify/assert"
)

func setupMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	db := sqlx.NewDb(mockDB, "sqlmock")
	return db, mock
}

func setupMockRedis(t *testing.T) (*database.RedisClient, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to create miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Create a database.RedisClient wrapper around the redis client
	redisClient := &database.RedisClient{Client: client}

	return redisClient, mr
}

func TestCreateMatch_Success(t *testing.T) {
	// Arrange
	db, mock := setupMockDB(t)
	redisClient, miniRedis := setupMockRedis(t)
	defer miniRedis.Close()

	repo := NewMatchRepository(&models.Config{}, db, redisClient)

	// Note: The implementation will generate a new UUID inside CreateMatch
	// so we can't predict the exact ID that will be used
	driverID := uuid.New()
	passengerID := uuid.New()

	// Location data
	driverLoc := models.Location{Latitude: -6.175392, Longitude: 106.827153}
	passengerLoc := models.Location{Latitude: -6.185392, Longitude: 106.837153}
	targetLoc := models.Location{Latitude: -6.195392, Longitude: 106.847153}

	match := &models.Match{
		// Don't set ID because the implementation will generate a new one
		DriverID:          driverID,
		PassengerID:       passengerID,
		Status:            models.MatchStatusPending,
		DriverLocation:    driverLoc,
		PassengerLocation: passengerLoc,
		TargetLocation:    targetLoc,
	}

	// Mock transaction behavior
	mock.ExpectBegin()

	// Use sqlmock.AnyArg() for the ID since we can't predict what will be generated
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO matches")).
		WithArgs(
			sqlmock.AnyArg(), // ID will be generated in the implementation
			driverID,
			passengerID,
			driverLoc.Longitude,
			driverLoc.Latitude,
			passengerLoc.Longitude,
			passengerLoc.Latitude,
			targetLoc.Longitude,
			targetLoc.Latitude,
			models.MatchStatusPending,
			false,            // driver_confirmed
			false,            // passenger_confirmed
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect transaction commit
	mock.ExpectCommit()

	// Act
	ctx := context.Background()
	createdMatch, err := repo.CreateMatch(ctx, match)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, createdMatch)
	// Can't check exact ID since it's generated, but it should be a valid UUID
	assert.NotEqual(t, uuid.Nil, createdMatch.ID)
	assert.Equal(t, driverID, createdMatch.DriverID)
	assert.Equal(t, passengerID, createdMatch.PassengerID)
	assert.Equal(t, models.MatchStatusPending, createdMatch.Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMatch_Success(t *testing.T) {
	// Arrange
	db, mock := setupMockDB(t)
	redisClient, miniRedis := setupMockRedis(t)
	defer miniRedis.Close()

	repo := NewMatchRepository(&models.Config{}, db, redisClient)

	matchID := uuid.New()
	driverID := uuid.New()
	passengerID := uuid.New()
	matchIDString := matchID.String()

	// Setup location data for the match
	driverLongitude := 106.827153
	driverLatitude := -6.175392
	passengerLongitude := 106.837153
	passengerLatitude := -6.185392

	// Use actual time.Time objects instead of strings for the timestamps
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "driver_id", "passenger_id",
		"driver_longitude", "driver_latitude",
		"passenger_longitude", "passenger_latitude",
		"target_longitude", "target_latitude",
		"status", "driver_confirmed", "passenger_confirmed",
		"created_at", "updated_at"}).
		AddRow(
			matchID, driverID, passengerID,
			driverLongitude, driverLatitude,
			passengerLongitude, passengerLatitude,
			106.837153, -6.185392, // target location
			models.MatchStatusPending, false, false, // confirmation flags
			now, now) // Use time.Time objects here

	mock.ExpectQuery(regexp.QuoteMeta(`
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
	`)).
		WithArgs(matchIDString).
		WillReturnRows(rows)

	// Act
	ctx := context.Background()
	match, err := repo.GetMatch(ctx, matchIDString)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, match)
	assert.Equal(t, matchID, match.ID)
	assert.Equal(t, driverID, match.DriverID)
	assert.Equal(t, passengerID, match.PassengerID)
	assert.Equal(t, models.MatchStatusPending, match.Status)
	assert.Equal(t, driverLongitude, match.DriverLocation.Longitude)
	assert.Equal(t, driverLatitude, match.DriverLocation.Latitude)
	assert.Equal(t, passengerLongitude, match.PassengerLocation.Longitude)
	assert.Equal(t, passengerLatitude, match.PassengerLocation.Latitude)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMatchStatus_Success(t *testing.T) {
	// Arrange
	db, mock := setupMockDB(t)
	redisClient, miniRedis := setupMockRedis(t)
	defer miniRedis.Close()

	repo := NewMatchRepository(&models.Config{}, db, redisClient)

	matchID := uuid.New().String()
	driverID := uuid.New()
	passengerID := uuid.New()
	newStatus := models.MatchStatusAccepted
	now := time.Now()

	// First, the implementation will call GetMatch to retrieve the current match
	// Set up the expected query for GetMatch
	mockRows := sqlmock.NewRows([]string{
		"id", "driver_id", "passenger_id",
		"driver_longitude", "driver_latitude",
		"passenger_longitude", "passenger_latitude",
		"status", "created_at", "updated_at"}).
		AddRow(
			matchID, driverID, passengerID,
			106.827153, -6.175392,
			106.837153, -6.185392,
			models.MatchStatusPending, now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT 
			id, driver_id, passenger_id,
			(driver_location[0])::float8 as driver_longitude,
			(driver_location[1])::float8 as driver_latitude,
			(passenger_location[0])::float8 as passenger_longitude,
			(passenger_location[1])::float8 as passenger_latitude,
			status, created_at, updated_at
		FROM matches
		WHERE id = $1
	`)).WithArgs(matchID).WillReturnRows(mockRows)

	// Then it begins a transaction
	mock.ExpectBegin()

	// Then it executes the update query
	mock.ExpectExec(regexp.QuoteMeta("UPDATE matches SET")).
		WithArgs(newStatus, sqlmock.AnyArg(), matchID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Finally it commits the transaction
	mock.ExpectCommit()

	// Act
	ctx := context.Background()
	err := repo.UpdateMatchStatus(ctx, matchID, newStatus)

	// Assert
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// NOTE: Geo-spatial methods (AddAvailableDriver, RemoveAvailableDriver, FindNearbyDrivers,
// AddAvailablePassenger, RemoveAvailablePassenger) have been moved to the location service.
// These tests are no longer needed in the match repository.

// TestListMatchesByPassenger tests listing all matches for a passenger
func TestListMatchesByPassenger_Success(t *testing.T) {
	// Arrange
	db, mock := setupMockDB(t)
	redisClient, miniRedis := setupMockRedis(t)
	defer miniRedis.Close()

	repo := NewMatchRepository(&models.Config{}, db, redisClient)

	passengerID := uuid.New()
	now := time.Now()

	// Setup mock data for SQL query
	matchRows := sqlmock.NewRows([]string{
		"id", "driver_id", "passenger_id",
		"driver_longitude", "driver_latitude",
		"passenger_longitude", "passenger_latitude",
		"target_longitude", "target_latitude",
		"status", "driver_confirmed", "passenger_confirmed",
		"created_at", "updated_at"})

	// Add 3 matches for the passenger
	matchID1 := uuid.New()
	matchID2 := uuid.New()
	matchID3 := uuid.New()
	driverID1 := uuid.New()
	driverID2 := uuid.New()
	driverID3 := uuid.New()

	matchRows.AddRow(
		matchID1, driverID1, passengerID,
		106.827153, -6.175392, 106.837153, -6.185392,
		106.847153, -6.195392, // target location
		models.MatchStatusAccepted, false, true, // confirmation flags
		now, now)

	matchRows.AddRow(
		matchID2, driverID2, passengerID,
		106.827153, -6.175392, 106.837153, -6.185392,
		106.847153, -6.195392, // target location
		models.MatchStatusPending, false, false, // confirmation flags
		now, now)

	matchRows.AddRow(
		matchID3, driverID3, passengerID,
		106.827153, -6.175392, 106.837153, -6.185392,
		106.847153, -6.195392, // target location
		models.MatchStatusRejected, false, false, // confirmation flags
		now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`
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
    `)).WithArgs(passengerID).WillReturnRows(matchRows)

	// Act
	ctx := context.Background()
	matches, err := repo.ListMatchesByPassenger(ctx, passengerID)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, matches, 3, "Should return 3 matches")
	assert.Equal(t, matchID1, matches[0].ID)
	assert.Equal(t, matchID2, matches[1].ID)
	assert.Equal(t, matchID3, matches[2].ID)
	assert.Equal(t, driverID1, matches[0].DriverID)
	assert.Equal(t, driverID2, matches[1].DriverID)
	assert.Equal(t, driverID3, matches[2].DriverID)
	assert.Equal(t, models.MatchStatusAccepted, matches[0].Status)
	assert.Equal(t, models.MatchStatusPending, matches[1].Status)
	assert.Equal(t, models.MatchStatusRejected, matches[2].Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// MockRedisClientForErrors is a mock implementation that satisfies the database.RedisClient interface
type MockRedisClientForErrors struct {
	*database.RedisClient // embedding but not using the embedded methods
}

// GeoAdd simulates a Redis error for the test
func (m *MockRedisClientForErrors) GeoAdd(ctx context.Context, key string, longitude, latitude float64, member string) error {
	return fmt.Errorf("simulated Redis error")
}

// TestGetMatch_NotFound tests getting a non-existent match
func TestGetMatch_NotFound(t *testing.T) {
	// Arrange
	db, mock := setupMockDB(t)
	redisClient, miniRedis := setupMockRedis(t)
	defer miniRedis.Close()

	repo := NewMatchRepository(&models.Config{}, db, redisClient)

	matchID := uuid.New().String()

	mock.ExpectQuery(regexp.QuoteMeta(`
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
	`)).WithArgs(matchID).WillReturnError(fmt.Errorf("no rows in result set"))

	// Act
	ctx := context.Background()
	match, err := repo.GetMatch(ctx, matchID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, match)
	assert.Contains(t, err.Error(), "failed to get match")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateMatchStatus_NotFound tests updating a non-existent match
func TestUpdateMatchStatus_NotFound(t *testing.T) {
	// Arrange
	db, mock := setupMockDB(t)
	redisClient, miniRedis := setupMockRedis(t)
	defer miniRedis.Close()

	repo := NewMatchRepository(&models.Config{}, db, redisClient)

	matchID := uuid.New().String()

	// Mock GetMatch to return error
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT 
			id, driver_id, passenger_id,
			(driver_location[0])::float8 as driver_longitude,
			(driver_location[1])::float8 as driver_latitude,
			(passenger_location[0])::float8 as passenger_longitude,
			(passenger_location[1])::float8 as passenger_latitude,
			status, created_at, updated_at
		FROM matches
		WHERE id = $1
	`)).WithArgs(matchID).WillReturnError(fmt.Errorf("no rows in result set"))

	// Act
	ctx := context.Background()
	err := repo.UpdateMatchStatus(ctx, matchID, models.MatchStatusAccepted)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get match")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateMatchStatus_TransactionError tests error handling when beginning a transaction fails
func TestUpdateMatchStatus_TransactionError(t *testing.T) {
	// Arrange
	db, mock := setupMockDB(t)
	redisClient, miniRedis := setupMockRedis(t)
	defer miniRedis.Close()

	repo := NewMatchRepository(&models.Config{}, db, redisClient)

	matchID := uuid.New().String()
	driverID := uuid.New()
	passengerID := uuid.New()

	now := time.Now()

	// Mock successful GetMatch
	rows := sqlmock.NewRows([]string{
		"id", "driver_id", "passenger_id",
		"driver_longitude", "driver_latitude",
		"passenger_longitude", "passenger_latitude",
		"status", "created_at", "updated_at"}).
		AddRow(
			matchID, driverID, passengerID,
			106.827153, -6.175392,
			106.837153, -6.185392,
			models.MatchStatusPending, now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT 
			id, driver_id, passenger_id,
			(driver_location[0])::float8 as driver_longitude,
			(driver_location[1])::float8 as driver_latitude,
			(passenger_location[0])::float8 as passenger_longitude,
			(passenger_location[1])::float8 as passenger_latitude,
			status, created_at, updated_at
		FROM matches
		WHERE id = $1
	`)).WithArgs(matchID).WillReturnRows(rows)

	// Mock transaction error
	mock.ExpectBegin().WillReturnError(fmt.Errorf("connection error"))

	// Act
	ctx := context.Background()
	err := repo.UpdateMatchStatus(ctx, matchID, models.MatchStatusAccepted)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to begin transaction")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateMatchStatus_NoRowsAffected tests when update doesn't affect any rows
func TestUpdateMatchStatus_NoRowsAffected(t *testing.T) {
	// Arrange
	db, mock := setupMockDB(t)
	redisClient, miniRedis := setupMockRedis(t)
	defer miniRedis.Close()

	repo := NewMatchRepository(&models.Config{}, db, redisClient)

	matchID := uuid.New().String()
	driverID := uuid.New()
	passengerID := uuid.New()

	now := time.Now()

	// Mock successful GetMatch
	rows := sqlmock.NewRows([]string{
		"id", "driver_id", "passenger_id",
		"driver_longitude", "driver_latitude",
		"passenger_longitude", "passenger_latitude",
		"status", "created_at", "updated_at"}).
		AddRow(
			matchID, driverID, passengerID,
			106.827153, -6.175392,
			106.837153, -6.185392,
			models.MatchStatusPending, now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT 
			id, driver_id, passenger_id,
			(driver_location[0])::float8 as driver_longitude,
			(driver_location[1])::float8 as driver_latitude,
			(passenger_location[0])::float8 as passenger_longitude,
			(passenger_location[1])::float8 as passenger_latitude,
			status, created_at, updated_at
		FROM matches
		WHERE id = $1
	`)).WithArgs(matchID).WillReturnRows(rows)

	// Mock transaction with no rows affected
	mock.ExpectBegin()

	// Use a regexp that matches both PostgreSQL ($1) and MySQL (?) style placeholders
	mock.ExpectExec(`UPDATE matches SET status = (.+), updated_at = (.+) WHERE id = (.+)`).
		WithArgs(models.MatchStatusAccepted, sqlmock.AnyArg(), matchID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectRollback()

	// Act
	ctx := context.Background()
	err := repo.UpdateMatchStatus(ctx, matchID, models.MatchStatusAccepted)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "match not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestListMatchesByPassenger_RowError tests error handling during row scanning
func TestListMatchesByPassenger_RowError(t *testing.T) {
	// Arrange
	db, mock := setupMockDB(t)
	redisClient, miniRedis := setupMockRedis(t)
	defer miniRedis.Close()

	repo := NewMatchRepository(&models.Config{}, db, redisClient)

	passengerID := uuid.New()

	// Create a rows mock that will return an error when Next() is called
	rows := sqlmock.NewRows([]string{
		"id", "driver_id", "passenger_id",
		"driver_longitude", "driver_latitude",
		"passenger_longitude", "passenger_latitude",
		"target_longitude", "target_latitude",
		"status", "driver_confirmed", "passenger_confirmed",
		"created_at", "updated_at"}).
		AddRow(
			"invalid-uuid", "invalid-driver", passengerID,
			"not-a-float", "not-a-float",
			"not-a-float", "not-a-float",
			"not-a-float", "not-a-float",
			models.MatchStatusAccepted, false, false,
			"not-a-time", "not-a-time").
		RowError(0, fmt.Errorf("scan error"))

	mock.ExpectQuery(regexp.QuoteMeta(`
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
    `)).WithArgs(passengerID).WillReturnRows(rows)

	// Act
	ctx := context.Background()
	matches, err := repo.ListMatchesByPassenger(ctx, passengerID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, matches)
	assert.Contains(t, err.Error(), "error iterating matches")
	assert.NoError(t, mock.ExpectationsWereMet())
}
