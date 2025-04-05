package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// RateLimiterConfig contains configuration for the rate limiter
type RateLimiterConfig struct {
	RedisClient *redis.Client
	Key         string        // Key prefix for Redis
	Limit       int           // Maximum number of requests
	Period      time.Duration // Time period for the limit
}

// RateLimiterMiddleware creates a middleware for rate limiting using Redis
func RateLimiterMiddleware(config RateLimiterConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client IP or user ID for rate limiting
		identifier := c.ClientIP()
		if userID, exists := c.Get("user_id"); exists {
			identifier = userID.(string)
		}

		// Create a key for this route and identifier
		key := fmt.Sprintf("%s:%s:%s", config.Key, c.FullPath(), identifier)

		ctx := context.Background()

		// Get the current count
		val, err := config.RedisClient.Get(ctx, key).Result()
		if err != nil && err != redis.Nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Rate limiter error"})
			return
		}

		var count int
		if err == redis.Nil {
			// Key doesn't exist, set it with expiration
			count = 1
			err = config.RedisClient.Set(ctx, key, count, config.Period).Err()
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Rate limiter error"})
				return
			}
		} else {
			// Key exists, increment it
			count, _ = strconv.Atoi(val)
			count++

			// Check if the limit is exceeded
			if count > config.Limit {
				// Get TTL for the key to determine reset time
				ttl, err := config.RedisClient.TTL(ctx, key).Result()
				if err != nil {
					c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
					return
				}

				// Set rate limit headers
				c.Header("X-RateLimit-Limit", strconv.Itoa(config.Limit))
				c.Header("X-RateLimit-Remaining", "0")
				c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(ttl).Unix(), 10))
				c.Header("Retry-After", strconv.FormatInt(int64(ttl.Seconds()), 10))

				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
				return
			}

			// Update the count
			err = config.RedisClient.Set(ctx, key, count, config.RedisClient.TTL(ctx, key).Val()).Err()
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Rate limiter error"})
				return
			}
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(config.Limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(config.Limit-count))

		c.Next()
	}
}

// IPRateLimiter creates a simple IP-based rate limiter
func IPRateLimiter(limit int, period time.Duration, redisClient *redis.Client) gin.HandlerFunc {
	return RateLimiterMiddleware(RateLimiterConfig{
		RedisClient: redisClient,
		Key:         "rate:ip",
		Limit:       limit,
		Period:      period,
	})
}

// UserRateLimiter creates a user-based rate limiter
func UserRateLimiter(limit int, period time.Duration, redisClient *redis.Client) gin.HandlerFunc {
	return RateLimiterMiddleware(RateLimiterConfig{
		RedisClient: redisClient,
		Key:         "rate:user",
		Limit:       limit,
		Period:      period,
	})
}
