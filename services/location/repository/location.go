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

	// Get location data from Redis hash
	locationData, err := r.redisClient.HGetAll(ctx, locationKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get location data: %w", err)
	}

	// If no data found, return error
	if len(locationData) == 0 {
		return nil, fmt.Errorf("no location data found for ride %s", rideID)
	}

	// Parse latitude and longitude
	lat, err := strconv.ParseFloat(locationData[constants.FieldLatitude], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid latitude: %w", err)
	}

	lng, err := strconv.ParseFloat(locationData[constants.FieldLongitude], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid longitude: %w", err)
	}

	// Parse timestamp
	ts, err := strconv.ParseInt(locationData[constants.FieldTimestamp], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp: %w", err)
	}

	return &models.Location{
		Latitude:  lat,
		Longitude: lng,
		Timestamp: time.Unix(ts, 0),
	}, nil
}
