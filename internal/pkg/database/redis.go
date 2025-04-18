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

// Exists checks if a key exists
func (r *RedisClient) Keys(ctx context.Context, key string) error {
	return r.client.Keys(ctx, key).Err()
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

// SAdd adds members to a set
func (r *RedisClient) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return r.client.SAdd(ctx, key, members...).Err()
}

// SRem removes members from a set
func (r *RedisClient) SRem(ctx context.Context, key string, members ...interface{}) error {
	return r.client.SRem(ctx, key, members...).Err()
}

// ZRem removes members from a sorted set
func (r *RedisClient) ZRem(ctx context.Context, key string, members ...interface{}) error {
	return r.client.ZRem(ctx, key, members...).Err()
}

// HMSet sets multiple hash fields
func (r *RedisClient) HMSet(ctx context.Context, key string, values map[string]interface{}) error {
	return r.client.HMSet(ctx, key, values).Err()
}

// HGetAll gets all fields in a hash
func (r *RedisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.client.HGetAll(ctx, key).Result()
}

// Expire sets an expiration on a key
func (r *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, key, expiration).Err()
}

// Close closes the Redis client
func (r *RedisClient) Close() error {
	return r.client.Close()
}
