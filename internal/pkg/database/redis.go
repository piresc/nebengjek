package database

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// RedisClient represents a Redis client
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient creates a new Redis client
func NewRedisClient(config models.RedisConfig) (*RedisClient, error) {
	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       config.DB,
		PoolSize: config.PoolSize,
	})

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisClient{client: client}, nil
}

// GetClient returns the underlying Redis client
func (r *RedisClient) GetClient() *redis.Client {
	return r.client
}

// Set stores a key-value pair with an optional expiration
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// Get retrieves a value by key
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// Delete removes a key
func (r *RedisClient) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// GeoAdd adds geospatial data to a sorted set
func (r *RedisClient) GeoAdd(ctx context.Context, key string, longitude, latitude float64, member string) error {
	return r.client.GeoAdd(ctx, key, &redis.GeoLocation{
		Longitude: longitude,
		Latitude:  latitude,
		Name:      member,
	}).Err()
}

// GeoRadius finds members within a radius from a point
func (r *RedisClient) GeoRadius(ctx context.Context, key string, longitude, latitude float64, radius float64, unit string) ([]redis.GeoLocation, error) {
	return r.client.GeoRadius(ctx, key, longitude, latitude, &redis.GeoRadiusQuery{
		Radius:    radius,
		Unit:      unit,
		WithCoord: true,
		WithDist:  true,
		Sort:      "ASC",
	}).Result()
}

// Close closes the Redis client
func (r *RedisClient) Close() error {
	return r.client.Close()
}
