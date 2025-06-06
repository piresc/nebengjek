package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/constants"
	"github.com/piresc/nebengjek/internal/pkg/database"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/location"
)

const (
	// LocationTTL is how long we keep location data in Redis
	// We keep it for 24 hours to allow for trip history analysis
	LocationTTL = 24 * time.Hour
)

type locationRepo struct {
	redisClient     *database.RedisClient
	availabilityTTL time.Duration
}

// NewLocationRepository creates a new location repository
func NewLocationRepository(redisClient *database.RedisClient, config *models.Config) location.LocationRepo {
	// Default TTL to 30 minutes if not configured
	ttlMinutes := 30
	if config != nil && config.Location.AvailabilityTTLMinutes > 0 {
		ttlMinutes = config.Location.AvailabilityTTLMinutes
	}

	return &locationRepo{
		redisClient:     redisClient,
		availabilityTTL: time.Duration(ttlMinutes) * time.Minute,
	}
}

// StoreLocation stores a location update in Redis for a ride
func (r *locationRepo) StoreLocation(ctx context.Context, rideID string, location models.Location) error {
	// Store in ride location hash
	locationKey := fmt.Sprintf(constants.KeyRideLocation, rideID)
	locationData := map[string]interface{}{
		constants.FieldLatitude:  strconv.FormatFloat(location.Latitude, 'f', -1, 64),
		constants.FieldLongitude: strconv.FormatFloat(location.Longitude, 'f', -1, 64),
		constants.FieldTimestamp: strconv.FormatInt(location.Timestamp.Unix(), 10),
	}

	err := r.redisClient.HMSet(ctx, locationKey, locationData)
	if err != nil {
		return fmt.Errorf("failed to store location update: %w", err)
	}

	// Set TTL using Expire
	err = r.redisClient.Expire(ctx, locationKey, LocationTTL)
	if err != nil {
		return fmt.Errorf("failed to set location TTL: %w", err)
	}

	return nil
}

// GetLastLocation gets the last stored location for a ride
func (r *locationRepo) GetLastLocation(ctx context.Context, rideID string) (*models.Location, error) {
	locationKey := fmt.Sprintf(constants.KeyRideLocation, rideID)

	// Get specific location fields from Redis hash using HMGET
	fields := []string{
		constants.FieldLatitude,
		constants.FieldLongitude,
		constants.FieldTimestamp,
	}

	values, err := r.redisClient.HMGet(ctx, locationKey, fields...)
	if err != nil {
		return nil, fmt.Errorf("failed to get location data: %w", err)
	}

	// Check if any values were returned
	hasValue := false
	for _, v := range values {
		if v != "" {
			hasValue = true
			break
		}
	}

	if !hasValue || len(values) != 3 {
		return nil, fmt.Errorf("no location data found for ride %s", rideID)
	}

	// Parse latitude
	lat, err := strconv.ParseFloat(values[0], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid latitude: %w", err)
	}

	// Parse longitude
	lng, err := strconv.ParseFloat(values[1], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid longitude: %w", err)
	}

	// Parse timestamp
	ts, err := strconv.ParseInt(values[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp: %w", err)
	}

	return &models.Location{
		Latitude:  lat,
		Longitude: lng,
		Timestamp: time.Unix(ts, 0),
	}, nil
}

// addToRedisGeo adds a user to Redis geospatial index with TTL
func (r *locationRepo) addToRedisGeo(ctx context.Context, geoKey, availableKey, locationKeyTemplate, userID string, location *models.Location) error {
	// Add to geo set
	if err := r.redisClient.GeoAdd(ctx, geoKey, location.Longitude, location.Latitude, userID); err != nil {
		return fmt.Errorf("failed to add to geo index: %w", err)
	}

	// Set TTL on geo set
	if err := r.redisClient.Expire(ctx, geoKey, r.availabilityTTL); err != nil {
		return fmt.Errorf("failed to set geo index TTL: %w", err)
	}

	// Add to available set
	if err := r.redisClient.SAdd(ctx, availableKey, userID); err != nil {
		return fmt.Errorf("failed to add to available set: %w", err)
	}

	// Set TTL on available set
	if err := r.redisClient.Expire(ctx, availableKey, r.availabilityTTL); err != nil {
		return fmt.Errorf("failed to set available set TTL: %w", err)
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

	// Set TTL on individual location
	if err := r.redisClient.Expire(ctx, locationKey, r.availabilityTTL); err != nil {
		return fmt.Errorf("failed to set location TTL: %w", err)
	}

	return nil
}

// removeFromRedisGeo removes a user from Redis geospatial index
func (r *locationRepo) removeFromRedisGeo(ctx context.Context, geoKey, availableKey, locationKeyTemplate, userID string) error {
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
func (r *locationRepo) AddAvailableDriver(ctx context.Context, driverID string, location *models.Location) error {
	return r.addToRedisGeo(ctx,
		constants.KeyDriverGeo,
		constants.KeyAvailableDrivers,
		constants.KeyDriverLocation,
		driverID,
		location)
}

// RemoveAvailableDriver removes a driver from the available drivers sets
func (r *locationRepo) RemoveAvailableDriver(ctx context.Context, driverID string) error {
	return r.removeFromRedisGeo(ctx,
		constants.KeyDriverGeo,
		constants.KeyAvailableDrivers,
		constants.KeyDriverLocation,
		driverID)
}

// AddAvailablePassenger adds a passenger to the Redis geospatial index
func (r *locationRepo) AddAvailablePassenger(ctx context.Context, passengerID string, location *models.Location) error {
	return r.addToRedisGeo(ctx,
		constants.KeyPassengerGeo,
		constants.KeyAvailablePassengers,
		constants.KeyPassengerLocation,
		passengerID,
		location)
}

// RemoveAvailablePassenger removes a passenger from the Redis geospatial index
func (r *locationRepo) RemoveAvailablePassenger(ctx context.Context, passengerID string) error {
	return r.removeFromRedisGeo(ctx,
		constants.KeyPassengerGeo,
		constants.KeyAvailablePassengers,
		constants.KeyPassengerLocation,
		passengerID)
}

// findNearbyUsers finds available users within the specified radius
func (r *locationRepo) findNearbyUsers(ctx context.Context, geoKey, availableKey string, location *models.Location, radiusKm float64) ([]*models.NearbyUser, error) {
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
func (r *locationRepo) FindNearbyDrivers(ctx context.Context, location *models.Location, radiusKm float64) ([]*models.NearbyUser, error) {
	return r.findNearbyUsers(ctx, constants.KeyDriverGeo, constants.KeyAvailableDrivers, location, radiusKm)
}

// GetDriverLocation retrieves a driver's last known location
func (r *locationRepo) GetDriverLocation(ctx context.Context, driverID string) (models.Location, error) {
	// Try to get from the Redis location key
	locationKey := fmt.Sprintf(constants.KeyDriverLocation, driverID)
	fields, err := r.redisClient.HGetAll(ctx, locationKey)
	if err != nil {
		return models.Location{}, fmt.Errorf("failed to get location from Redis: %w", err)
	}

	// If we got data from Redis, parse it
	if len(fields) > 0 {
		var lat, lng float64
		var timestamp int64

		if latStr, ok := fields[constants.FieldLatitude]; ok {
			lat, _ = strconv.ParseFloat(latStr, 64)
		}

		if lngStr, ok := fields[constants.FieldLongitude]; ok {
			lng, _ = strconv.ParseFloat(lngStr, 64)
		}

		if tsStr, ok := fields[constants.FieldTimestamp]; ok {
			timestamp, _ = strconv.ParseInt(tsStr, 10, 64)
		}

		return models.Location{
			Latitude:  lat,
			Longitude: lng,
			Timestamp: time.Unix(timestamp, 0),
		}, nil
	}

	// If not in Redis, return error since location service doesn't have database access
	return models.Location{}, fmt.Errorf("no location data found for driver %s", driverID)
}

// GetPassengerLocation retrieves a passenger's last known location
func (r *locationRepo) GetPassengerLocation(ctx context.Context, passengerID string) (models.Location, error) {
	// Try to get from the Redis location key
	locationKey := fmt.Sprintf(constants.KeyPassengerLocation, passengerID)
	fields, err := r.redisClient.HGetAll(ctx, locationKey)
	if err != nil {
		return models.Location{}, fmt.Errorf("failed to get location from Redis: %w", err)
	}

	// If we got data from Redis, parse it
	if len(fields) > 0 {
		var lat, lng float64
		var timestamp int64

		if latStr, ok := fields[constants.FieldLatitude]; ok {
			lat, _ = strconv.ParseFloat(latStr, 64)
		}

		if lngStr, ok := fields[constants.FieldLongitude]; ok {
			lng, _ = strconv.ParseFloat(lngStr, 64)
		}

		if tsStr, ok := fields[constants.FieldTimestamp]; ok {
			timestamp, _ = strconv.ParseInt(tsStr, 10, 64)
		}

		return models.Location{
			Latitude:  lat,
			Longitude: lng,
			Timestamp: time.Unix(timestamp, 0),
		}, nil
	}

	// If not in Redis, return error since location service doesn't have database access
	return models.Location{}, fmt.Errorf("no location data found for passenger %s", passengerID)
}
