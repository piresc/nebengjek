package database

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redismock/v8"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestRedisConfig() models.RedisConfig {
	return models.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
		PoolSize: 10,
	}
}

func TestNewRedisClient(t *testing.T) {
	// Note: This test requires a running Redis instance
	// In a real test environment, you might want to use testcontainers
	t.Skip("Skipping integration test - requires running Redis instance")

	config := getTestRedisConfig()
	client, err := NewRedisClient(config)

	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, client.GetClient())
}

func TestNewRedisClient_ConnectionError(t *testing.T) {
	// Test with invalid configuration
	config := models.RedisConfig{
		Host:     "invalid-host",
		Port:     9999,
		Password: "",
		DB:       0,
		PoolSize: 10,
	}

	client, err := NewRedisClient(config)

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to connect to redis")
}

func TestRedisClient_Set(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &RedisClient{Client: db}

	ctx := context.Background()
	key := "test:key"
	value := "test-value"
	expiration := time.Hour

	mock.ExpectSet(key, value, expiration).SetVal("OK")

	err := client.Set(ctx, key, value, expiration)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRedisClient_Set_Error(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &RedisClient{Client: db}

	ctx := context.Background()
	key := "test:key"
	value := "test-value"
	expiration := time.Hour

	mock.ExpectSet(key, value, expiration).SetErr(redis.Nil)

	err := client.Set(ctx, key, value, expiration)

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRedisClient_SetNX(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		value          interface{}
		expiration     time.Duration
		mockResult     bool
		mockError      error
		expectedResult bool
		expectedError  bool
	}{
		{
			name:           "Key set successfully",
			key:            "test:new:key",
			value:          "new-value",
			expiration:     time.Hour,
			mockResult:     true,
			mockError:      nil,
			expectedResult: true,
			expectedError:  false,
		},
		{
			name:           "Key already exists",
			key:            "test:existing:key",
			value:          "existing-value",
			expiration:     time.Hour,
			mockResult:     false,
			mockError:      nil,
			expectedResult: false,
			expectedError:  false,
		},
		{
			name:           "Redis error",
			key:            "test:error:key",
			value:          "error-value",
			expiration:     time.Hour,
			mockResult:     false,
			mockError:      redis.Nil,
			expectedResult: false,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			client := &RedisClient{Client: db}

			ctx := context.Background()

			if tt.mockError != nil {
				mock.ExpectSetNX(tt.key, tt.value, tt.expiration).SetErr(tt.mockError)
			} else {
				mock.ExpectSetNX(tt.key, tt.value, tt.expiration).SetVal(tt.mockResult)
			}

			result, err := client.SetNX(ctx, tt.key, tt.value, tt.expiration)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRedisClient_Get(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		mockValue     string
		mockError     error
		expectedValue string
		expectedError bool
	}{
		{
			name:          "Key exists",
			key:           "test:existing:key",
			mockValue:     "existing-value",
			mockError:     nil,
			expectedValue: "existing-value",
			expectedError: false,
		},
		{
			name:          "Key does not exist",
			key:           "test:nonexistent:key",
			mockValue:     "",
			mockError:     redis.Nil,
			expectedValue: "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			client := &RedisClient{Client: db}

			ctx := context.Background()

			if tt.mockError != nil {
				mock.ExpectGet(tt.key).SetErr(tt.mockError)
			} else {
				mock.ExpectGet(tt.key).SetVal(tt.mockValue)
			}

			value, err := client.Get(ctx, tt.key)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedValue, value)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRedisClient_Delete(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &RedisClient{Client: db}

	ctx := context.Background()
	key := "test:delete:key"

	mock.ExpectDel(key).SetVal(1)

	err := client.Delete(ctx, key)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRedisClient_Delete_Error(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &RedisClient{Client: db}

	ctx := context.Background()
	key := "test:delete:key"

	mock.ExpectDel(key).SetErr(redis.Nil)

	err := client.Delete(ctx, key)

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRedisClient_GeoAdd(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &RedisClient{Client: db}

	ctx := context.Background()
	key := "drivers:location"
	longitude := 106.827153
	latitude := -6.175392
	member := "driver-123"

	mock.ExpectGeoAdd(key, &redis.GeoLocation{
		Longitude: longitude,
		Latitude:  latitude,
		Name:      member,
	}).SetVal(1)

	err := client.GeoAdd(ctx, key, longitude, latitude, member)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRedisClient_GeoAdd_Error(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &RedisClient{Client: db}

	ctx := context.Background()
	key := "drivers:location"
	longitude := 106.827153
	latitude := -6.175392
	member := "driver-123"

	mock.ExpectGeoAdd(key, &redis.GeoLocation{
		Longitude: longitude,
		Latitude:  latitude,
		Name:      member,
	}).SetErr(redis.Nil)

	err := client.GeoAdd(ctx, key, longitude, latitude, member)

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRedisClient_GeoRadius(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &RedisClient{Client: db}

	ctx := context.Background()
	key := "drivers:location"
	longitude := 106.827153
	latitude := -6.175392
	radius := 5.0
	unit := "km"

	expectedLocations := []redis.GeoLocation{
		{
			Name:      "driver-1",
			Longitude: 106.825,
			Latitude:  -6.173,
			Dist:      1.5,
		},
		{
			Name:      "driver-2",
			Longitude: 106.830,
			Latitude:  -6.178,
			Dist:      3.2,
		},
	}

	mock.ExpectGeoRadius(key, longitude, latitude, &redis.GeoRadiusQuery{
		Radius:    radius,
		Unit:      unit,
		WithCoord: true,
		WithDist:  true,
		Sort:      "ASC",
	}).SetVal(expectedLocations)

	locations, err := client.GeoRadius(ctx, key, longitude, latitude, radius, unit)

	assert.NoError(t, err)
	assert.Equal(t, expectedLocations, locations)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRedisClient_GeoRadius_Error(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &RedisClient{Client: db}

	ctx := context.Background()
	key := "drivers:location"
	longitude := 106.827153
	latitude := -6.175392
	radius := 5.0
	unit := "km"

	mock.ExpectGeoRadius(key, longitude, latitude, &redis.GeoRadiusQuery{
		Radius:    radius,
		Unit:      unit,
		WithCoord: true,
		WithDist:  true,
		Sort:      "ASC",
	}).SetErr(redis.Nil)

	locations, err := client.GeoRadius(ctx, key, longitude, latitude, radius, unit)

	assert.Error(t, err)
	assert.Nil(t, locations)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRedisClient_SAdd(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &RedisClient{Client: db}

	ctx := context.Background()
	key := "active:drivers"
	members := []interface{}{"driver-1", "driver-2", "driver-3"}

	mock.ExpectSAdd(key, members...).SetVal(3)

	err := client.SAdd(ctx, key, members...)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRedisClient_SAdd_Error(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &RedisClient{Client: db}

	ctx := context.Background()
	key := "active:drivers"
	members := []interface{}{"driver-1"}

	mock.ExpectSAdd(key, members...).SetErr(redis.Nil)

	err := client.SAdd(ctx, key, members...)

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRedisClient_SIsMember(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		member         interface{}
		mockResult     bool
		mockError      error
		expectedResult bool
		expectedError  bool
	}{
		{
			name:           "Member exists",
			key:            "active:drivers",
			member:         "driver-1",
			mockResult:     true,
			mockError:      nil,
			expectedResult: true,
			expectedError:  false,
		},
		{
			name:           "Member does not exist",
			key:            "active:drivers",
			member:         "driver-999",
			mockResult:     false,
			mockError:      nil,
			expectedResult: false,
			expectedError:  false,
		},
		{
			name:           "Redis error",
			key:            "active:drivers",
			member:         "driver-1",
			mockResult:     false,
			mockError:      redis.Nil,
			expectedResult: false,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			client := &RedisClient{Client: db}

			ctx := context.Background()

			if tt.mockError != nil {
				mock.ExpectSIsMember(tt.key, tt.member).SetErr(tt.mockError)
			} else {
				mock.ExpectSIsMember(tt.key, tt.member).SetVal(tt.mockResult)
			}

			result, err := client.SIsMember(ctx, tt.key, tt.member)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRedisClient_GetClient(t *testing.T) {
	db, _ := redismock.NewClientMock()
	client := &RedisClient{Client: db}

	result := client.GetClient()

	assert.Equal(t, db, result)
	assert.NotNil(t, result)
}

func TestRedisClient_Keys_Method(t *testing.T) {
	// Note: The Keys method in the original code has an error - it should return []string, not error
	// This test demonstrates the current implementation
	db, mock := redismock.NewClientMock()
	client := &RedisClient{Client: db}

	ctx := context.Background()
	pattern := "test:*"

	mock.ExpectKeys(pattern).SetVal([]string{"test:key1", "test:key2"})

	err := client.Keys(ctx, pattern)

	// The current implementation returns an error, but it should return []string
	// This is likely a bug in the original code
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRedisClient_IntegrationScenario(t *testing.T) {
	// Test a realistic scenario with multiple operations
	db, mock := redismock.NewClientMock()
	client := &RedisClient{Client: db}

	ctx := context.Background()

	// Scenario: Add driver location, check if active, find nearby drivers
	driverKey := "drivers:location"
	activeKey := "active:drivers"
	driverID := "driver-123"
	longitude := 106.827153
	latitude := -6.175392

	// Step 1: Add driver to geospatial index
	mock.ExpectGeoAdd(driverKey, &redis.GeoLocation{
		Longitude: longitude,
		Latitude:  latitude,
		Name:      driverID,
	}).SetVal(1)

	// Step 2: Add driver to active set
	mock.ExpectSAdd(activeKey, driverID).SetVal(1)

	// Step 3: Check if driver is active
	mock.ExpectSIsMember(activeKey, driverID).SetVal(true)

	// Step 4: Find nearby drivers
	mock.ExpectGeoRadius(driverKey, longitude, latitude, &redis.GeoRadiusQuery{
		Radius:    5.0,
		Unit:      "km",
		WithCoord: true,
		WithDist:  true,
		Sort:      "ASC",
	}).SetVal([]redis.GeoLocation{
		{Name: driverID, Longitude: longitude, Latitude: latitude, Dist: 0},
	})

	// Execute scenario
	err := client.GeoAdd(ctx, driverKey, longitude, latitude, driverID)
	require.NoError(t, err)

	err = client.SAdd(ctx, activeKey, driverID)
	require.NoError(t, err)

	isActive, err := client.SIsMember(ctx, activeKey, driverID)
	require.NoError(t, err)
	assert.True(t, isActive)

	nearbyDrivers, err := client.GeoRadius(ctx, driverKey, longitude, latitude, 5.0, "km")
	require.NoError(t, err)
	assert.Len(t, nearbyDrivers, 1)
	assert.Equal(t, driverID, nearbyDrivers[0].Name)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func BenchmarkRedisClient_Set(b *testing.B) {
	db, mock := redismock.NewClientMock()
	client := &RedisClient{Client: db}

	ctx := context.Background()
	key := "benchmark:key"
	value := "benchmark-value"
	expiration := time.Hour

	// Setup mock expectations for all iterations
	for i := 0; i < b.N; i++ {
		mock.ExpectSet(key, value, expiration).SetVal("OK")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.Set(ctx, key, value, expiration)
	}
}

func BenchmarkRedisClient_Get(b *testing.B) {
	db, mock := redismock.NewClientMock()
	client := &RedisClient{Client: db}

	ctx := context.Background()
	key := "benchmark:key"
	value := "benchmark-value"

	// Setup mock expectations for all iterations
	for i := 0; i < b.N; i++ {
		mock.ExpectGet(key).SetVal(value)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Get(ctx, key)
	}
}