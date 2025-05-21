package database

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// RedisClient represents a Redis Client
type RedisClient struct {
	Client *redis.Client
}

// NewRedisClient creates a new Redis Client
func NewRedisClient(config models.RedisConfig) (*RedisClient, error) {
	// Create Redis Client
	Client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       config.DB,
		PoolSize: config.PoolSize,
	})

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := Client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisClient{Client: Client}, nil
}

// GetClient returns the underlying Redis Client
func (r *RedisClient) GetClient() *redis.Client {
	return r.Client
}

// Set stores a key-value pair with an optional expiration
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.Client.Set(ctx, key, value, expiration).Err()
}

// SetNX sets value if key doesn't exist (Set if Not eXists)
// Returns true if key was set, false if key already exists
func (r *RedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return r.Client.SetNX(ctx, key, value, expiration).Result()
}

// Get retrieves a value by key
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.Client.Get(ctx, key).Result()
}

// Delete removes a key
func (r *RedisClient) Delete(ctx context.Context, key string) error {
	return r.Client.Del(ctx, key).Err()
}

// Exists checks if a key exists
func (r *RedisClient) Keys(ctx context.Context, key string) error {
	return r.Client.Keys(ctx, key).Err()
}

// GeoAdd adds geospatial data to a sorted set
func (r *RedisClient) GeoAdd(ctx context.Context, key string, longitude, latitude float64, member string) error {
	return r.Client.GeoAdd(ctx, key, &redis.GeoLocation{
		Longitude: longitude,
		Latitude:  latitude,
		Name:      member,
	}).Err()
}

// GeoRadius finds members within a radius from a point
func (r *RedisClient) GeoRadius(ctx context.Context, key string, longitude, latitude float64, radius float64, unit string) ([]redis.GeoLocation, error) {
	return r.Client.GeoRadius(ctx, key, longitude, latitude, &redis.GeoRadiusQuery{
		Radius:    radius,
		Unit:      unit,
		WithCoord: true,
		WithDist:  true,
		Sort:      "ASC",
	}).Result()
}

// SAdd adds members to a set
// Only adds elements that don't already exist in the set
func (r *RedisClient) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return r.Client.SAdd(ctx, key, members...).Err()
}

// SIsMember checks if a value is a member of a set
func (r *RedisClient) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return r.Client.SIsMember(ctx, key, member).Result()
}

// SRem removes members from a set
func (r *RedisClient) SRem(ctx context.Context, key string, members ...interface{}) error {
	return r.Client.SRem(ctx, key, members...).Err()
}

// ZRem removes members from a sorted set
func (r *RedisClient) ZRem(ctx context.Context, key string, members ...interface{}) error {
	return r.Client.ZRem(ctx, key, members...).Err()
}

// HMSet sets multiple hash fields
func (r *RedisClient) HMSet(ctx context.Context, key string, values map[string]interface{}) error {
	return r.Client.HMSet(ctx, key, values).Err()
}

// HGetAll gets all fields in a hash
func (r *RedisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.Client.HGetAll(ctx, key).Result()
}

// HMGet gets specified fields of a hash
func (r *RedisClient) HMGet(ctx context.Context, key string, fields ...string) ([]string, error) {
	// Get values from Redis
	vals, err := r.Client.HMGet(ctx, key, fields...).Result()
	if err != nil {
		return nil, err
	}

	// Convert []interface{} to []string
	results := make([]string, len(vals))
	for i, val := range vals {
		if val == nil {
			results[i] = ""
		} else {
			results[i] = fmt.Sprint(val)
		}
	}

	return results, nil
}

// Expire sets an expiration on a key
func (r *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.Client.Expire(ctx, key, expiration).Err()
}

// Close closes the Redis Client
func (r *RedisClient) Close() error {
	return r.Client.Close()
}
