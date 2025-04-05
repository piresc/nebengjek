package redis

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/piresc/nebengjek/match-service/domain/entity"
)

type driverRepository struct {
	client *redis.Client
}

func NewDriverRepository(client *redis.Client) *driverRepository {
	return &driverRepository{
		client: client,
	}
}

func (r *driverRepository) UpdateLocation(ctx context.Context, driverID string, lat, lon float64, status string) error {
	// Update driver location using Redis GEOADD
	_, err := r.client.GeoAdd(ctx, "driver_locations", &redis.GeoLocation{
		Name:      driverID,
		Latitude:  lat,
		Longitude: lon,
	}).Result()

	if err != nil {
		return err
	}

	// Update driver status
	_, err = r.client.HSet(ctx, "driver_status", driverID, status).Result()
	return err
}

func (r *driverRepository) FindNearbyDrivers(ctx context.Context, lat, lon, radiusKm float64) ([]*entity.Driver, error) {
	// Use Redis GEORADIUS to find nearby drivers
	result, err := r.client.GeoRadius(ctx, "driver_locations", lon, lat, &redis.GeoRadiusQuery{
		Radius:    radiusKm,
		Unit:      "km",
		WithDist:  true,
		WithCoord: true,
		Count:     10,
		Sort:      "ASC",
	}).Result()

	if err != nil {
		return nil, err
	}

	drivers := make([]*entity.Driver, 0, len(result))
	for _, loc := range result {
		status, _ := r.client.HGet(ctx, "driver_status", loc.Name).Result()
		drivers = append(drivers, &entity.Driver{
			ID:        loc.Name,
			Latitude:  loc.Latitude,
			Longitude: loc.Longitude,
			Distance:  loc.Dist,
			Status:    status,
		})
	}

	return drivers, nil
}
