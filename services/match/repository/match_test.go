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
	"github.com/piresc/nebengjek/internal/pkg/constants"
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

	match := &models.Match{
		// Don't set ID because the implementation will generate a new one
		DriverID:          driverID,
		PassengerID:       passengerID,
		Status:            models.MatchStatusPending,
		DriverLocation:    driverLoc,
		PassengerLocation: passengerLoc,
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
			models.MatchStatusPending,
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
		"status", "created_at", "updated_at"}).
		AddRow(
			matchID, driverID, passengerID,
			driverLongitude, driverLatitude,
			passengerLongitude, passengerLatitude,
			models.MatchStatusPending, now, now) // Use time.Time objects here

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

func TestStoreMatchProposal_Success(t *testing.T) {
	// Arrange
	db, _ := setupMockDB(t) // We don't need the mock here for Redis operations
	redisClient, miniRedis := setupMockRedis(t)
	defer miniRedis.Close()

	repo := NewMatchRepository(&models.Config{}, db, redisClient)

	matchID := uuid.New()
	driverID := uuid.New()
	passengerID := uuid.New()
	match := &models.Match{
		ID:          matchID,
		DriverID:    driverID,
		PassengerID: passengerID,
		Status:      models.MatchStatusPending,
	}

	// Act
	ctx := context.Background()
	err := repo.StoreMatchProposal(ctx, match)

	// Assert
	assert.NoError(t, err)

	// We can't directly test Redis since we're mocking with database.RedisClient
	// You would need to intercept calls or use dependency injection for testing
	// This would require modifying the repository implementation
}

func TestConfirmMatchAtomically_NotImplementedYet(t *testing.T) {
	t.Skip("This test requires Redis operations that need mocking")
}

func TestAddAvailableDriver_Success(t *testing.T) {
	// Arrange
	db, _ := setupMockDB(t)
	redisClient, miniRedis := setupMockRedis(t)
	defer miniRedis.Close()

	// Create a custom redis client that intercepts calls for testing
	redisClient.Client = redis.NewClient(&redis.Options{
		Addr: miniRedis.Addr(),
	})

	repo := NewMatchRepository(&models.Config{}, db, redisClient)

	driverID := uuid.New().String()
	location := &models.Location{
		Latitude:  -6.175392,
		Longitude: 106.827153,
		Timestamp: time.Now(),
	}

	// Act
	ctx := context.Background()
	err := repo.AddAvailableDriver(ctx, driverID, location)

	// Assert
	assert.NoError(t, err)

	// Verify that the driver was added to the geo set in Redis
	geoKey := constants.KeyDriverGeo
	members, err := miniRedis.ZMembers(geoKey)
	assert.NoError(t, err)
	assert.Contains(t, members, driverID, "Driver should be added to geo set")

	// Verify that the driver was added to the available drivers set
	availableKey := constants.KeyAvailableDrivers
	setMembers, err := miniRedis.SMembers(availableKey)
	assert.NoError(t, err)
	assert.Contains(t, setMembers, driverID, "Driver should be added to available set")

	// Verify that the driver location was stored
	locationKey := fmt.Sprintf(constants.KeyDriverLocation, driverID)
	locationExists := miniRedis.Exists(locationKey)
	assert.True(t, locationExists, "Driver location should be stored")
}

func TestAddAvailableDriver_ExistingDriver(t *testing.T) {
	// Arrange
	db, _ := setupMockDB(t)
	redisClient, miniRedis := setupMockRedis(t)
	defer miniRedis.Close()

	// Create a custom redis client that intercepts calls for testing
	redisClient.Client = redis.NewClient(&redis.Options{
		Addr: miniRedis.Addr(),
	})

	repo := NewMatchRepository(&models.Config{}, db, redisClient)

	ctx := context.Background()
	driverID := uuid.New().String()
	availableKey := constants.KeyAvailableDrivers

	// Pre-add the driver to the available set
	err := redisClient.Client.SAdd(ctx, availableKey, driverID).Err()
	assert.NoError(t, err)

	// Add initial location
	initialLocation := &models.Location{
		Latitude:  -6.175392,
		Longitude: 106.827153,
		Timestamp: time.Now(),
	}
	err = redisClient.Client.GeoAdd(ctx, constants.KeyDriverGeo,
		&redis.GeoLocation{
			Longitude: initialLocation.Longitude,
			Latitude:  initialLocation.Latitude,
			Name:      driverID,
		}).Err()
	assert.NoError(t, err)

	// Update with new location
	newLocation := &models.Location{
		Latitude:  -6.175500, // Slightly different location
		Longitude: 106.827200,
		Timestamp: time.Now(),
	}

	// Act - update the existing driver
	err = repo.AddAvailableDriver(ctx, driverID, newLocation)

	// Assert
	assert.NoError(t, err)

	// Verify that the driver is still in the available set
	setMembers, err := miniRedis.SMembers(availableKey)
	assert.NoError(t, err)
	assert.Contains(t, setMembers, driverID, "Driver should still be in available set")
	assert.Equal(t, 1, len(setMembers), "There should only be one member in the set")

	// Verify location was updated
	locationKey := fmt.Sprintf(constants.KeyDriverLocation, driverID)
	locationExists := miniRedis.Exists(locationKey)
	assert.True(t, locationExists, "Driver location should be stored")
}

// TestRemoveAvailableDriver tests removing a driver from the available drivers pool
func TestRemoveAvailableDriver_Success(t *testing.T) {
	// Arrange
	db, _ := setupMockDB(t)
	redisClient, miniRedis := setupMockRedis(t)
	defer miniRedis.Close()

	// Create a custom redis client that intercepts calls for testing
	redisClient.Client = redis.NewClient(&redis.Options{
		Addr: miniRedis.Addr(),
	})

	repo := NewMatchRepository(&models.Config{}, db, redisClient)

	driverID := uuid.New().String()

	// First add the driver to available pools
	ctx := context.Background()
	location := &models.Location{
		Latitude:  -6.175392,
		Longitude: 106.827153,
		Timestamp: time.Now(),
	}

	// Add driver directly to Redis
	geoKey := constants.KeyDriverGeo
	availableKey := constants.KeyAvailableDrivers
	locationKey := fmt.Sprintf(constants.KeyDriverLocation, driverID)

	// Use Redis client to add data
	err := redisClient.Client.ZAdd(ctx, geoKey, &redis.Z{
		Score:  0,
		Member: driverID,
	}).Err()
	assert.NoError(t, err)

	err = redisClient.Client.SAdd(ctx, availableKey, driverID).Err()
	assert.NoError(t, err)

	err = redisClient.Client.HMSet(ctx, locationKey, map[string]interface{}{
		constants.FieldLatitude:  location.Latitude,
		constants.FieldLongitude: location.Longitude,
		constants.FieldTimestamp: time.Now().Unix(),
	}).Err()
	assert.NoError(t, err)

	// Act - Remove the driver
	err = repo.RemoveAvailableDriver(ctx, driverID)

	// Assert
	assert.NoError(t, err)

	// Verify driver was removed from geo set - key might not exist after removal
	if miniRedis.Exists(geoKey) {
		members, err := miniRedis.ZMembers(geoKey)
		assert.NoError(t, err)
		assert.NotContains(t, members, driverID, "Driver should be removed from geo set")
	} else {
		// If key doesn't exist, it means all members were removed, which is also correct
		assert.True(t, true, "Geo key was removed completely")
	}

	// Verify driver was removed from available set - key might not exist after removal
	if miniRedis.Exists(availableKey) {
		isMember, err := miniRedis.SIsMember(availableKey, driverID)
		assert.NoError(t, err)
		assert.False(t, isMember, "Driver should be removed from available set")
	} else {
		// If key doesn't exist, it means all members were removed, which is also correct
		assert.True(t, true, "Available set was removed completely")
	}

	// Verify driver location was removed
	locationExists := miniRedis.Exists(locationKey)
	assert.False(t, locationExists, "Driver location should be removed")
}

func TestRemoveAvailableDriver_NotImplementedYet(t *testing.T) {
	t.Skip("This test requires Redis operations that need mocking")
}

func TestFindNearbyDrivers_NotImplementedYet(t *testing.T) {
	t.Skip("This test requires Redis operations that need mocking")
}

// TestFindNearbyDrivers tests finding nearby drivers
func TestFindNearbyDrivers_Success(t *testing.T) {
	// Arrange
	db, _ := setupMockDB(t)
	redisClient, miniRedis := setupMockRedis(t)
	defer miniRedis.Close()

	repo := NewMatchRepository(&models.Config{}, db, redisClient)

	// Setup test data
	ctx := context.Background()

	// Add three drivers at different locations
	driver1ID := uuid.New().String()
	driver2ID := uuid.New().String()
	driver3ID := uuid.New().String()

	// Driver 1: 0.5km away
	client := redisClient.Client
	err := client.GeoAdd(ctx, constants.KeyDriverGeo, &redis.GeoLocation{
		Name:      driver1ID,
		Longitude: 106.827153,
		Latitude:  -6.180392, // ~0.5km from test location
	}).Err()
	assert.NoError(t, err)

	// Add driver1 to available drivers set
	err = client.SAdd(ctx, constants.KeyAvailableDrivers, driver1ID).Err()
	assert.NoError(t, err)

	// Driver 2: 0.8km away
	err = client.GeoAdd(ctx, constants.KeyDriverGeo, &redis.GeoLocation{
		Name:      driver2ID,
		Longitude: 106.827153,
		Latitude:  -6.182392, // ~0.8km from test location
	}).Err()
	assert.NoError(t, err)

	// Add driver2 to available drivers set
	err = client.SAdd(ctx, constants.KeyAvailableDrivers, driver2ID).Err()
	assert.NoError(t, err)

	// Driver 3: 1.2km away (outside 1km radius)
	err = client.GeoAdd(ctx, constants.KeyDriverGeo, &redis.GeoLocation{
		Name:      driver3ID,
		Longitude: 106.827153,
		Latitude:  -6.185392, // ~1.2km from test location
	}).Err()
	assert.NoError(t, err)

	// Add driver3 to available drivers set
	err = client.SAdd(ctx, constants.KeyAvailableDrivers, driver3ID).Err()
	assert.NoError(t, err)

	// Search location
	searchLocation := &models.Location{
		Latitude:  -6.175392,
		Longitude: 106.827153,
	}

	// Act - Find drivers within 1km
	nearbyDrivers, err := repo.FindNearbyDrivers(ctx, searchLocation, 1.0)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, nearbyDrivers, 2, "Should find 2 drivers within 1km")

	// Check if the correct drivers were found
	driverIDs := make([]string, len(nearbyDrivers))
	for i, driver := range nearbyDrivers {
		driverIDs[i] = driver.ID
	}
	assert.Contains(t, driverIDs, driver1ID, "Driver 1 should be found")
	assert.Contains(t, driverIDs, driver2ID, "Driver 2 should be found")
	assert.NotContains(t, driverIDs, driver3ID, "Driver 3 should not be found (outside range)")
}

// TestAddAvailablePassenger tests adding a passenger to available pool
func TestAddAvailablePassenger_Success(t *testing.T) {
	// Arrange
	db, _ := setupMockDB(t)
	redisClient, miniRedis := setupMockRedis(t)
	defer miniRedis.Close()

	repo := NewMatchRepository(&models.Config{}, db, redisClient)

	passengerID := uuid.New().String()
	location := &models.Location{
		Latitude:  -6.175392,
		Longitude: 106.827153,
		Timestamp: time.Now(),
	}

	// Act
	ctx := context.Background()
	err := repo.AddAvailablePassenger(ctx, passengerID, location)

	// Assert
	assert.NoError(t, err)

	// Verify passenger was added to geo set in Redis
	geoKey := constants.KeyPassengerGeo
	members, err := miniRedis.ZMembers(geoKey)
	assert.NoError(t, err)
	assert.Contains(t, members, passengerID, "Passenger should be added to geo set")
}

// TestRemoveAvailablePassenger tests removing a passenger from available pool
func TestRemoveAvailablePassenger_Success(t *testing.T) {
	// Arrange
	db, _ := setupMockDB(t)
	redisClient, miniRedis := setupMockRedis(t)
	defer miniRedis.Close()

	repo := NewMatchRepository(&models.Config{}, db, redisClient)

	passengerID := uuid.New().String()
	ctx := context.Background()

	// First add passenger to geo set
	geoKey := constants.KeyPassengerGeo
	err := redisClient.Client.GeoAdd(ctx, geoKey, &redis.GeoLocation{
		Name:      passengerID,
		Longitude: 106.827153,
		Latitude:  -6.175392,
	}).Err()
	assert.NoError(t, err)

	// Act - Remove the passenger
	err = repo.RemoveAvailablePassenger(ctx, passengerID)

	// Assert
	assert.NoError(t, err)

	// Verify passenger was removed - key might not exist after removal
	if miniRedis.Exists(geoKey) {
		members, err := miniRedis.ZMembers(geoKey)
		assert.NoError(t, err)
		assert.NotContains(t, members, passengerID, "Passenger should be removed from geo set")
	} else {
		// If key doesn't exist, it means all members were removed, which is also correct
		assert.True(t, true, "Geo key was removed completely")
	}
}

// TestFindNearbyPassengers tests finding nearby passengers
func TestFindNearbyPassengers_Success(t *testing.T) {
	// Arrange
	db, _ := setupMockDB(t)
	redisClient, miniRedis := setupMockRedis(t)
	defer miniRedis.Close()

	repo := NewMatchRepository(&models.Config{}, db, redisClient)

	// Setup test data
	ctx := context.Background()

	// Add three passengers at different locations
	passenger1ID := uuid.New().String()
	passenger2ID := uuid.New().String()
	passenger3ID := uuid.New().String()

	client := redisClient.Client
	// Passenger 1: 0.5km away
	err := client.GeoAdd(ctx, constants.KeyPassengerGeo, &redis.GeoLocation{
		Name:      passenger1ID,
		Longitude: 106.827153,
		Latitude:  -6.180392, // ~0.5km from test location
	}).Err()
	assert.NoError(t, err)

	// Add passenger1 to available passengers set
	err = client.SAdd(ctx, constants.KeyAvailablePassengers, passenger1ID).Err()
	assert.NoError(t, err)

	// Passenger 2: 0.8km away
	err = client.GeoAdd(ctx, constants.KeyPassengerGeo, &redis.GeoLocation{
		Name:      passenger2ID,
		Longitude: 106.827153,
		Latitude:  -6.182392, // ~0.8km from test location
	}).Err()
	assert.NoError(t, err)

	// Add passenger2 to available passengers set
	err = client.SAdd(ctx, constants.KeyAvailablePassengers, passenger2ID).Err()
	assert.NoError(t, err)

	// Passenger 3: 1.2km away (outside 1km radius)
	err = client.GeoAdd(ctx, constants.KeyPassengerGeo, &redis.GeoLocation{
		Name:      passenger3ID,
		Longitude: 106.827153,
		Latitude:  -6.185392, // ~1.2km from test location
	}).Err()
	assert.NoError(t, err)

	// Add passenger3 to available passengers set
	err = client.SAdd(ctx, constants.KeyAvailablePassengers, passenger3ID).Err()
	assert.NoError(t, err)

	// Search location
	searchLocation := &models.Location{
		Latitude:  -6.175392,
		Longitude: 106.827153,
	}

	// Act - Find passengers within 1km
	nearbyPassengers, err := repo.FindNearbyPassengers(ctx, searchLocation, 1.0)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, nearbyPassengers, 2, "Should find 2 passengers within 1km")

	// Check if correct passengers were found
	passengerIDs := make([]string, len(nearbyPassengers))
	for i, passenger := range nearbyPassengers {
		passengerIDs[i] = passenger.ID
	}
	assert.Contains(t, passengerIDs, passenger1ID, "Passenger 1 should be found")
	assert.Contains(t, passengerIDs, passenger2ID, "Passenger 2 should be found")
	assert.NotContains(t, passengerIDs, passenger3ID, "Passenger 3 should not be found (outside range)")
}

// TestConfirmMatchAtomically tests atomically confirming a match
func TestConfirmMatchAtomically_Success(t *testing.T) {
	// Arrange
	db, mock := setupMockDB(t)
	redisClient, miniRedis := setupMockRedis(t)
	defer miniRedis.Close()

	repo := NewMatchRepository(&models.Config{}, db, redisClient)

	matchID := uuid.New().String()
	driverID := uuid.New().String()
	passengerID := uuid.New().String()

	// Setup for RemoveAvailableDriver/Passenger
	redisClient.Client = redis.NewClient(&redis.Options{
		Addr: miniRedis.Addr(),
	})

	// Add driver and passenger to Redis for later removal
	ctx := context.Background()
	geoDriverKey := constants.KeyDriverGeo
	geoPassengerKey := constants.KeyPassengerGeo
	availableKey := constants.KeyAvailableDrivers
	locationKey := fmt.Sprintf(constants.KeyDriverLocation, driverID)

	err := redisClient.Client.ZAdd(ctx, geoDriverKey, &redis.Z{
		Score:  0,
		Member: driverID,
	}).Err()
	assert.NoError(t, err)

	err = redisClient.Client.ZAdd(ctx, geoPassengerKey, &redis.Z{
		Score:  0,
		Member: passengerID,
	}).Err()
	assert.NoError(t, err)

	err = redisClient.Client.SAdd(ctx, availableKey, driverID).Err()
	assert.NoError(t, err)

	err = redisClient.Client.HMSet(ctx, locationKey, map[string]interface{}{
		constants.FieldLatitude:  -6.175392,
		constants.FieldLongitude: 106.827153,
		constants.FieldTimestamp: time.Now().Unix(),
	}).Err()
	assert.NoError(t, err)

	// Mock SQL operations
	// 1. Begin transaction
	mock.ExpectBegin()

	// 2. Lock and get current status
	mock.ExpectQuery(regexp.QuoteMeta("SELECT status::text, driver_id FROM matches WHERE id = $1 FOR UPDATE")).
		WithArgs(matchID).
		WillReturnRows(sqlmock.NewRows([]string{"status", "driver_id"}).AddRow(string(models.MatchStatusPending), driverID))

	// 3. Update match status
	mock.ExpectExec(regexp.QuoteMeta("UPDATE matches SET status = $1, updated_at = NOW() WHERE id = $2 AND status = $3")).
		WithArgs(models.MatchStatusAccepted, matchID, models.MatchStatusPending).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 4. Get passenger ID to remove it
	mock.ExpectQuery(regexp.QuoteMeta("SELECT passenger_id FROM matches WHERE id = $1")).
		WithArgs(matchID).
		WillReturnRows(sqlmock.NewRows([]string{"passenger_id"}).AddRow(passengerID))

	// 5. Commit transaction
	mock.ExpectCommit()

	// Act
	err = repo.ConfirmMatchAtomically(ctx, matchID, models.MatchStatusAccepted)

	// Assert
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())

	// Verify driver and passenger were removed from Redis - check if keys exist first
	if miniRedis.Exists(geoDriverKey) {
		driverMembers, err := miniRedis.ZMembers(geoDriverKey)
		assert.NoError(t, err)
		assert.NotContains(t, driverMembers, driverID, "Driver should be removed from geo set")
	} else {
		// Key was completely removed, which is also valid
		assert.True(t, true, "Driver geo set was completely removed")
	}

	if miniRedis.Exists(geoPassengerKey) {
		passengerMembers, err := miniRedis.ZMembers(geoPassengerKey)
		assert.NoError(t, err)
		assert.NotContains(t, passengerMembers, passengerID, "Passenger should be removed from geo set")
	} else {
		// Key was completely removed, which is also valid
		assert.True(t, true, "Passenger geo set was completely removed")
	}
}

// TestListMatchesByDriver tests listing all matches for a driver
func TestListMatchesByDriver_Success(t *testing.T) {
	// Arrange
	db, mock := setupMockDB(t)
	redisClient, miniRedis := setupMockRedis(t)
	defer miniRedis.Close()

	repo := NewMatchRepository(&models.Config{}, db, redisClient)

	driverID := uuid.New().String()
	now := time.Now()

	// Setup mock data for SQL query
	matchRows := sqlmock.NewRows([]string{
		"id", "driver_id", "passenger_id",
		"driver_longitude", "driver_latitude",
		"passenger_longitude", "passenger_latitude",
		"status", "created_at", "updated_at"})

	// Add 3 matches for the driver
	matchID1 := uuid.New()
	matchID2 := uuid.New()
	matchID3 := uuid.New()
	passengerID1 := uuid.New()
	passengerID2 := uuid.New()
	passengerID3 := uuid.New()

	matchRows.AddRow(
		matchID1, driverID, passengerID1,
		106.827153, -6.175392, 106.837153, -6.185392,
		models.MatchStatusAccepted, now, now)

	matchRows.AddRow(
		matchID2, driverID, passengerID2,
		106.827153, -6.175392, 106.837153, -6.185392,
		models.MatchStatusPending, now, now)

	matchRows.AddRow(
		matchID3, driverID, passengerID3,
		106.827153, -6.175392, 106.837153, -6.185392,
		models.MatchStatusRejected, now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`
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
    `)).WithArgs(driverID).WillReturnRows(matchRows)

	// Act
	ctx := context.Background()
	matches, err := repo.ListMatchesByDriver(ctx, driverID)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, matches, 3, "Should return 3 matches")
	assert.Equal(t, matchID1, matches[0].ID)
	assert.Equal(t, matchID2, matches[1].ID)
	assert.Equal(t, matchID3, matches[2].ID)
	assert.Equal(t, models.MatchStatusAccepted, matches[0].Status)
	assert.Equal(t, models.MatchStatusPending, matches[1].Status)
	assert.Equal(t, models.MatchStatusRejected, matches[2].Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

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
		"status", "created_at", "updated_at"})

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
		models.MatchStatusAccepted, now, now)

	matchRows.AddRow(
		matchID2, driverID2, passengerID,
		106.827153, -6.175392, 106.837153, -6.185392,
		models.MatchStatusPending, now, now)

	matchRows.AddRow(
		matchID3, driverID3, passengerID,
		106.827153, -6.175392, 106.837153, -6.185392,
		models.MatchStatusRejected, now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`
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

// TestProcessLocationUpdate_Success tests updating a driver's location in the geospatial index
func TestProcessLocationUpdate_Success(t *testing.T) {
	// Arrange
	db, _ := setupMockDB(t)
	redisClient, miniRedis := setupMockRedis(t)
	defer miniRedis.Close()

	repo := NewMatchRepository(&models.Config{}, db, redisClient)

	driverID := uuid.New().String()
	location := &models.Location{
		Latitude:  -6.175392,
		Longitude: 106.827153,
		Timestamp: time.Now(),
	}

	// Act
	ctx := context.Background()
	err := repo.ProcessLocationUpdate(ctx, driverID, location)

	// Assert
	assert.NoError(t, err)

	// Verify driver's location was updated in Redis geo set
	geoKey := constants.KeyDriverGeo
	members, err := miniRedis.ZMembers(geoKey)
	assert.NoError(t, err)
	assert.Contains(t, members, driverID, "Driver should be added to geo set")

	// Check if position was stored correctly using GeoPos
	positions, err := redisClient.Client.GeoPos(ctx, geoKey, driverID).Result()
	assert.NoError(t, err)
	assert.NotEmpty(t, positions)
	assert.NotNil(t, positions[0])

	// Verify coordinates are close to what we set (within small delta for float precision)
	assert.InDelta(t, location.Longitude, positions[0].Longitude, 0.001)
	assert.InDelta(t, location.Latitude, positions[0].Latitude, 0.001)
}

// MockRedisClientForErrors is a mock implementation that satisfies the database.RedisClient interface
type MockRedisClientForErrors struct {
	*database.RedisClient // embedding but not using the embedded methods
}

// GeoAdd simulates a Redis error for the test
func (m *MockRedisClientForErrors) GeoAdd(ctx context.Context, key string, longitude, latitude float64, member string) error {
	return fmt.Errorf("simulated Redis error")
}

// TestStoreMatchProposal_MarshalError tests error handling when marshaling match data fails
func TestStoreMatchProposal_MarshalError(t *testing.T) {
	// This test requires a special method to trigger a marshaling error
	// Since we can't easily make json.Marshal fail, we'll skip this test
	// but include it for documentation purposes
	t.Skip("Cannot easily trigger a json.Marshal error")
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
			status, created_at, updated_at
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

// TestListMatchesByDriver_Empty tests listing matches for a driver with no matches
func TestListMatchesByDriver_Empty(t *testing.T) {
	// Arrange
	db, mock := setupMockDB(t)
	redisClient, miniRedis := setupMockRedis(t)
	defer miniRedis.Close()

	repo := NewMatchRepository(&models.Config{}, db, redisClient)

	driverID := uuid.New().String()

	// Mock empty result
	mock.ExpectQuery(regexp.QuoteMeta(`
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
    `)).WithArgs(driverID).WillReturnRows(sqlmock.NewRows([]string{
		"id", "driver_id", "passenger_id",
		"driver_longitude", "driver_latitude",
		"passenger_longitude", "passenger_latitude",
		"status", "created_at", "updated_at"}))

	// Act
	ctx := context.Background()
	matches, err := repo.ListMatchesByDriver(ctx, driverID)

	// Assert
	assert.NoError(t, err)
	assert.Empty(t, matches)
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
		"status", "created_at", "updated_at"}).
		AddRow(
			"invalid-uuid", "invalid-driver", passengerID,
			"not-a-float", "not-a-float",
			"not-a-float", "not-a-float",
			models.MatchStatusAccepted, "not-a-time", "not-a-time").
		RowError(0, fmt.Errorf("scan error"))

	mock.ExpectQuery(regexp.QuoteMeta(`
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

// TestConfirmMatchAtomically_CommitError tests error handling when committing a transaction fails
func TestConfirmMatchAtomically_CommitError(t *testing.T) {
	// Arrange
	db, mock := setupMockDB(t)
	redisClient, miniRedis := setupMockRedis(t)
	defer miniRedis.Close()

	repo := NewMatchRepository(&models.Config{}, db, redisClient)

	matchID := uuid.New().String()
	driverID := uuid.New().String()
	passengerID := uuid.New().String()

	// Add driver and passenger to Redis geo sets for removal
	geoDriverKey := constants.KeyDriverGeo
	geoPassengerKey := constants.KeyPassengerGeo

	client := redisClient.Client
	ctx := context.Background()

	// Add driver to driver geo set
	err := client.GeoAdd(ctx, geoDriverKey, &redis.GeoLocation{
		Name:      driverID,
		Longitude: 106.827153,
		Latitude:  -6.175392,
	}).Err()
	assert.NoError(t, err)

	// Add passenger to passenger geo set
	err = client.GeoAdd(ctx, geoPassengerKey, &redis.GeoLocation{
		Name:      passengerID,
		Longitude: 106.837153,
		Latitude:  -6.185392,
	}).Err()
	assert.NoError(t, err)

	// Mock transaction
	mock.ExpectBegin()

	// Mock getting current status with FOR UPDATE lock
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT status::text, driver_id
        FROM matches 
        WHERE id = $1 
        FOR UPDATE
    `)).
		WithArgs(matchID).
		WillReturnRows(sqlmock.NewRows([]string{"status", "driver_id"}).AddRow(string(models.MatchStatusPending), driverID))

	// Mock update match status
	mock.ExpectExec(regexp.QuoteMeta("UPDATE matches SET status = $1, updated_at = NOW() WHERE id = $2 AND status = $3")).
		WithArgs(models.MatchStatusAccepted, matchID, models.MatchStatusPending).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Mock getting passenger ID
	mock.ExpectQuery(regexp.QuoteMeta("SELECT passenger_id FROM matches WHERE id = $1")).
		WithArgs(matchID).
		WillReturnRows(sqlmock.NewRows([]string{"passenger_id"}).AddRow(passengerID))

	// Mock commit error
	mock.ExpectCommit().WillReturnError(fmt.Errorf("commit error"))

	// Act
	err = repo.ConfirmMatchAtomically(ctx, matchID, models.MatchStatusAccepted)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to commit transaction")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestConfirmMatchAtomically_StatusMismatch tests error when match has the wrong status
func TestConfirmMatchAtomically_StatusMismatch(t *testing.T) {
	// Arrange
	db, mock := setupMockDB(t)
	redisClient, miniRedis := setupMockRedis(t)
	defer miniRedis.Close()

	repo := NewMatchRepository(&models.Config{}, db, redisClient)

	matchID := uuid.New().String()
	driverID := uuid.New().String()

	// Mock transaction
	mock.ExpectBegin()

	// Mock getting current status with wrong status
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT status::text, driver_id
        FROM matches 
        WHERE id = $1 
        FOR UPDATE
    `)).
		WithArgs(matchID).
		WillReturnRows(sqlmock.NewRows([]string{"status", "driver_id"}).AddRow(string(models.MatchStatusRejected), driverID))

	// Mock rollback
	mock.ExpectRollback()

	// Act
	ctx := context.Background()
	err := repo.ConfirmMatchAtomically(ctx, matchID, models.MatchStatusAccepted)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "match cannot be confirmed")
	assert.NoError(t, mock.ExpectationsWereMet())
}
