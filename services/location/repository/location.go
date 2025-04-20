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
	redisClient *database.RedisClient
}

// NewLocationRepository creates a new location repository
func NewLocationRepository(redisClient *database.RedisClient) location.LocationRepo {
	return &locationRepo{
		redisClient: redisClient,
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
