package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMiniredis creates a new miniredis server and returns a Redis client connected to it
func setupMiniredis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return mr, client
}

func TestStoreLocation(t *testing.T) {
	// Setup miniredis
	mr, client := setupMiniredis(t)
	defer mr.Close()

	// Create the repository with the Redis client
	repo := NewLocationRepository(&database.RedisClient{
		Client: client,
	})

	// Test data
	ctx := context.Background()
	rideID := "ride-123"
	timestamp := time.Now()
	location := models.Location{
		Latitude:  -6.175392,
		Longitude: 106.827153,
		Timestamp: timestamp,
	}

	// Act - Call the method being tested
	err := repo.StoreLocation(ctx, rideID, location)

	// Assert the results
	assert.NoError(t, err)

	// Verify the data was stored correctly in Redis using the client
	locationKey := fmt.Sprintf(constants.KeyRideLocation, rideID)

	// Check keys in Redis directly
	keys := mr.Keys()
	assert.Contains(t, keys, locationKey)

	// Check that hash fields exist in Redis
	assert.True(t, mr.Exists(locationKey))

	// Check specific fields in hash
	fields := []string{constants.FieldLatitude, constants.FieldLongitude, constants.FieldTimestamp}
	vals, err := client.HMGet(ctx, locationKey, fields...).Result()
	require.NoError(t, err)

	// All fields should have values
	for _, val := range vals {
		assert.NotNil(t, val)
	}
}

func TestStoreLocation_RedisError(t *testing.T) {
	// Setup miniredis
	mr, client := setupMiniredis(t)
	defer mr.Close()

	// Create the repository with the Redis client

	// Create the repository with the Redis client
	repo := NewLocationRepository(&database.RedisClient{
		Client: client,
	})
	// Test data
	ctx := context.Background()
	rideID := "ride-123"
	timestamp := time.Now()
	location := models.Location{
		Latitude:  -6.175392,
		Longitude: 106.827153,
		Timestamp: timestamp,
	}

	// Force Redis to fail by closing the connection
	mr.Close()

	// Act - Call the method being tested
	err := repo.StoreLocation(ctx, rideID, location)

	// Assert the results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store location")
}

func TestGetLastLocation(t *testing.T) {
	// Setup miniredis
	mr, client := setupMiniredis(t)
	defer mr.Close()

	// Create the repository with the Redis client

	// Create the repository with the Redis client
	repo := NewLocationRepository(&database.RedisClient{
		Client: client,
	})
	// Test data
	ctx := context.Background()
	rideID := "ride-123"
	locationKey := fmt.Sprintf(constants.KeyRideLocation, rideID)

	// Pre-populate Redis with test data using miniredis API
	mr.HSet(locationKey, constants.FieldLatitude, "-6.175392")
	mr.HSet(locationKey, constants.FieldLongitude, "106.827153")
	mr.HSet(locationKey, constants.FieldTimestamp, "1681977600")

	// Act - Call the method being tested
	location, err := repo.GetLastLocation(ctx, rideID)

	// Assert the results
	assert.NoError(t, err)
	assert.NotNil(t, location)
	assert.Equal(t, -6.175392, location.Latitude)
	assert.Equal(t, 106.827153, location.Longitude)
	assert.Equal(t, int64(1681977600), location.Timestamp.Unix())
}

func TestGetLastLocation_NoData(t *testing.T) {
	// Setup miniredis
	mr, client := setupMiniredis(t)
	defer mr.Close()

	// Create the repository with the Redis client

	// Create the repository with the Redis client
	repo := NewLocationRepository(&database.RedisClient{
		Client: client,
	})
	// Test data
	ctx := context.Background()
	rideID := "ride-123"

	// Redis has no data for this key

	// Act - Call the method being tested
	location, err := repo.GetLastLocation(ctx, rideID)

	// Assert the results
	assert.Error(t, err)
	assert.Nil(t, location)
	assert.Contains(t, err.Error(), "no location data found")
}

func TestGetLastLocation_InvalidLatitude(t *testing.T) {
	// Setup miniredis
	mr, client := setupMiniredis(t)
	defer mr.Close()

	// Create the repository with the Redis client

	// Create the repository with the Redis client
	repo := NewLocationRepository(&database.RedisClient{
		Client: client,
	})
	// Test data
	ctx := context.Background()
	rideID := "ride-123"
	locationKey := fmt.Sprintf(constants.KeyRideLocation, rideID)

	// Pre-populate Redis with invalid latitude
	mr.HSet(locationKey, constants.FieldLatitude, "not-a-number")
	mr.HSet(locationKey, constants.FieldLongitude, "106.827153")
	mr.HSet(locationKey, constants.FieldTimestamp, "1681977600")

	// Act - Call the method being tested
	location, err := repo.GetLastLocation(ctx, rideID)

	// Assert the results
	assert.Error(t, err)
	assert.Nil(t, location)
	assert.Contains(t, err.Error(), "invalid latitude")
}

func TestGetLastLocation_InvalidLongitude(t *testing.T) {
	// Setup miniredis
	mr, client := setupMiniredis(t)
	defer mr.Close()

	// Create the repository with the Redis client

	// Create the repository with the Redis client
	repo := NewLocationRepository(&database.RedisClient{
		Client: client,
	})
	// Test data
	ctx := context.Background()
	rideID := "ride-123"
	locationKey := fmt.Sprintf(constants.KeyRideLocation, rideID)

	// Pre-populate Redis with invalid longitude
	mr.HSet(locationKey, constants.FieldLatitude, "-6.175392")
	mr.HSet(locationKey, constants.FieldLongitude, "not-a-number")
	mr.HSet(locationKey, constants.FieldTimestamp, "1681977600")

	// Act - Call the method being tested
	location, err := repo.GetLastLocation(ctx, rideID)

	// Assert the results
	assert.Error(t, err)
	assert.Nil(t, location)
	assert.Contains(t, err.Error(), "invalid longitude")
}

func TestGetLastLocation_InvalidTimestamp(t *testing.T) {
	// Setup miniredis
	mr, client := setupMiniredis(t)
	defer mr.Close()

	// Create the repository with the Redis client

	// Create the repository with the Redis client
	repo := NewLocationRepository(&database.RedisClient{
		Client: client,
	})
	// Test data
	ctx := context.Background()
	rideID := "ride-123"
	locationKey := fmt.Sprintf(constants.KeyRideLocation, rideID)

	// Pre-populate Redis with invalid timestamp
	mr.HSet(locationKey, constants.FieldLatitude, "-6.175392")
	mr.HSet(locationKey, constants.FieldLongitude, "106.827153")
	mr.HSet(locationKey, constants.FieldTimestamp, "not-a-number")

	// Act - Call the method being tested
	location, err := repo.GetLastLocation(ctx, rideID)

	// Assert the results
	assert.Error(t, err)
	assert.Nil(t, location)
	assert.Contains(t, err.Error(), "invalid timestamp")
}

func TestGetLastLocation_RedisError(t *testing.T) {
	// Setup miniredis
	mr, client := setupMiniredis(t)
	defer mr.Close()

	// Create the repository with the Redis client

	// Create the repository with the Redis client
	repo := NewLocationRepository(&database.RedisClient{
		Client: client,
	})
	// Test data
	ctx := context.Background()
	rideID := "ride-123"

	// Force Redis to fail by closing the connection
	mr.Close()

	// Act - Call the method being tested
	location, err := repo.GetLastLocation(ctx, rideID)

	// Assert the results
	assert.Error(t, err)
	assert.Nil(t, location)
	assert.Contains(t, err.Error(), "failed to get location data")
}
