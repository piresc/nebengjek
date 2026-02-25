package gateway

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/database"
	locationmodels "github.com/piresc/nebengjek/internal/pkg/models/location"
	matchmodels "github.com/piresc/nebengjek/internal/pkg/models/match"
)

// RedisGateway handles Redis geospatial operations for location-based matching
type RedisGateway struct {
	redisClient *database.RedisClient
}

// NewRedisGateway creates a new Redis gateway
func NewRedisGateway(redisClient *database.RedisClient) *RedisGateway {
	return &RedisGateway{
		redisClient: redisClient,
	}
}

// Redis keys for geospatial data
const (
	KeyAvailableDrivers    = "match:available_drivers"
	KeyAvailablePassengers = "match:available_passengers" 
	KeyDriverLocations     = "match:driver_locations"
	KeyPassengerLocations  = "match:passenger_locations"
)

// AddAvailableDriver adds a driver to the available drivers geo set
func (gw *RedisGateway) AddAvailableDriver(ctx context.Context, driverID string, location *locationmodels.Location) error {
	// Add to geospatial set
	if err := gw.redisClient.GeoAdd(ctx, KeyAvailableDrivers, location.Longitude, location.Latitude, driverID); err != nil {
		return fmt.Errorf("failed to add driver to geo set: %w", err)
	}

	// Store detailed location info
	locationKey := fmt.Sprintf("%s:%s", KeyDriverLocations, driverID)
	locationData := map[string]interface{}{
		"latitude":  location.Latitude,
		"longitude": location.Longitude,
		"timestamp": time.Now().Unix(),
	}

	if err := gw.redisClient.HMSet(ctx, locationKey, locationData); err != nil {
		return fmt.Errorf("failed to store driver location: %w", err)
	}

	return nil
}

// RemoveAvailableDriver removes a driver from the available drivers sets
func (gw *RedisGateway) RemoveAvailableDriver(ctx context.Context, driverID string) error {
	// Remove from geospatial set
	if err := gw.redisClient.ZRem(ctx, KeyAvailableDrivers, driverID); err != nil {
		return fmt.Errorf("failed to remove driver from geo set: %w", err)
	}

	// Remove detailed location info
	locationKey := fmt.Sprintf("%s:%s", KeyDriverLocations, driverID)
	if err := gw.redisClient.Delete(ctx, locationKey); err != nil {
		return fmt.Errorf("failed to remove driver location: %w", err)
	}

	return nil
}

// AddAvailablePassenger adds a passenger to the Redis geospatial index
func (gw *RedisGateway) AddAvailablePassenger(ctx context.Context, passengerID string, location *locationmodels.Location) error {
	// Add to geospatial set
	if err := gw.redisClient.GeoAdd(ctx, KeyAvailablePassengers, location.Longitude, location.Latitude, passengerID); err != nil {
		return fmt.Errorf("failed to add passenger to geo set: %w", err)
	}

	// Store detailed location info
	locationKey := fmt.Sprintf("%s:%s", KeyPassengerLocations, passengerID)
	locationData := map[string]interface{}{
		"latitude":  location.Latitude,
		"longitude": location.Longitude,
		"timestamp": time.Now().Unix(),
	}

	if err := gw.redisClient.HMSet(ctx, locationKey, locationData); err != nil {
		return fmt.Errorf("failed to store passenger location: %w", err)
	}

	return nil
}

// RemoveAvailablePassenger removes a passenger from the Redis geospatial index
func (gw *RedisGateway) RemoveAvailablePassenger(ctx context.Context, passengerID string) error {
	// Remove from geospatial set
	if err := gw.redisClient.ZRem(ctx, KeyAvailablePassengers, passengerID); err != nil {
		return fmt.Errorf("failed to remove passenger from geo set: %w", err)
	}

	// Remove detailed location info
	locationKey := fmt.Sprintf("%s:%s", KeyPassengerLocations, passengerID)
	if err := gw.redisClient.Delete(ctx, locationKey); err != nil {
		return fmt.Errorf("failed to remove passenger location: %w", err)
	}

	return nil
}

// FindNearbyDrivers finds available drivers within the specified radius
func (gw *RedisGateway) FindNearbyDrivers(ctx context.Context, location *locationmodels.Location, radiusKm float64) ([]*matchmodels.NearbyUser, error) {
	// Use Redis GEORADIUS to find nearby drivers
	results, err := gw.redisClient.GeoRadius(ctx, KeyAvailableDrivers, location.Longitude, location.Latitude, radiusKm, "km")
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby drivers: %w", err)
	}

	nearbyDrivers := make([]*matchmodels.NearbyUser, 0, len(results))
	for _, result := range results {
		nearbyDriver := &matchmodels.NearbyUser{
			ID:       result.Name,
			Distance: result.Dist,
			Location: locationmodels.Location{
				Latitude:  result.Latitude,
				Longitude: result.Longitude,
			},
		}
		nearbyDrivers = append(nearbyDrivers, nearbyDriver)
	}

	return nearbyDrivers, nil
}

// GetDriverLocation retrieves a driver's last known location
func (gw *RedisGateway) GetDriverLocation(ctx context.Context, driverID string) (locationmodels.Location, error) {
	locationKey := fmt.Sprintf("%s:%s", KeyDriverLocations, driverID)
	
	locationData, err := gw.redisClient.HGetAll(ctx, locationKey)
	if err != nil {
		return locationmodels.Location{}, fmt.Errorf("failed to get driver location: %w", err)
	}

	if len(locationData) == 0 {
		return locationmodels.Location{}, fmt.Errorf("driver location not found")
	}

	location := locationmodels.Location{}
	
	if lat, ok := locationData["latitude"]; ok {
		if location.Latitude, err = strconv.ParseFloat(lat, 64); err != nil {
			return locationmodels.Location{}, fmt.Errorf("failed to parse latitude: %w", err)
		}
	}
	
	if lng, ok := locationData["longitude"]; ok {
		if location.Longitude, err = strconv.ParseFloat(lng, 64); err != nil {
			return locationmodels.Location{}, fmt.Errorf("failed to parse longitude: %w", err)
		}
	}


	return location, nil
}

// GetPassengerLocation retrieves a passenger's last known location
func (gw *RedisGateway) GetPassengerLocation(ctx context.Context, passengerID string) (locationmodels.Location, error) {
	locationKey := fmt.Sprintf("%s:%s", KeyPassengerLocations, passengerID)
	
	locationData, err := gw.redisClient.HGetAll(ctx, locationKey)
	if err != nil {
		return locationmodels.Location{}, fmt.Errorf("failed to get passenger location: %w", err)
	}

	if len(locationData) == 0 {
		return locationmodels.Location{}, fmt.Errorf("passenger location not found")
	}

	location := locationmodels.Location{}
	
	if lat, ok := locationData["latitude"]; ok {
		if location.Latitude, err = strconv.ParseFloat(lat, 64); err != nil {
			return locationmodels.Location{}, fmt.Errorf("failed to parse latitude: %w", err)
		}
	}
	
	if lng, ok := locationData["longitude"]; ok {
		if location.Longitude, err = strconv.ParseFloat(lng, 64); err != nil {
			return locationmodels.Location{}, fmt.Errorf("failed to parse longitude: %w", err)
		}
	}


	return location, nil
}