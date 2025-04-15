package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	"github.com/piresc/nebengjek/internal/utils"
)

// RateLimiterConfig contains configuration for the rate limiter
type RateLimiterConfig struct {
	RedisClient *redis.Client
	Key         string        // Key prefix for Redis
	Limit       int           // Maximum number of requests
	Period      time.Duration // Time period for the limit
}

// RateLimiterMiddleware creates a middleware for rate limiting using Redis
func RateLimiterMiddleware(config RateLimiterConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get client IP or user ID for rate limiting
			identifier := c.RealIP()
			if userID := c.Get("user_id"); userID != nil {
				identifier = userID.(string)
			}

			// Create a key for this route and identifier
			key := fmt.Sprintf("%s:%s:%s", config.Key, c.Path(), identifier)

			ctx := context.Background()

			// Get the current count
			val, err := config.RedisClient.Get(ctx, key).Result()
			if err != nil && err != redis.Nil {
				return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Rate limiter error")
			}

			var count int
			if err == redis.Nil {
				// Key doesn't exist, set it with expiration
				count = 1
				err = config.RedisClient.Set(ctx, key, count, config.Period).Err()
				if err != nil {
					return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Rate limiter error")
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
						return utils.ErrorResponseHandler(c, http.StatusTooManyRequests, "Rate limit exceeded")
					}

					// Set rate limit headers
					c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(config.Limit))
					c.Response().Header().Set("X-RateLimit-Remaining", "0")
					c.Response().Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(ttl).Unix(), 10))
					c.Response().Header().Set("Retry-After", strconv.FormatInt(int64(ttl.Seconds()), 10))

					return utils.ErrorResponseHandler(c, http.StatusTooManyRequests, "Rate limit exceeded")
				}

				// Update the count
				err = config.RedisClient.Set(ctx, key, count, config.RedisClient.TTL(ctx, key).Val()).Err()
				if err != nil {
					return utils.ErrorResponseHandler(c, http.StatusInternalServerError, "Rate limiter error")
				}
			}

			// Set rate limit headers
			c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(config.Limit))
			c.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(config.Limit-count))

			return next(c)
		}
	}
}

// IPRateLimiter creates a simple IP-based rate limiter
func IPRateLimiter(limit int, period time.Duration, redisClient *redis.Client) echo.MiddlewareFunc {
	return RateLimiterMiddleware(RateLimiterConfig{
		RedisClient: redisClient,
		Key:         "rate:ip",
		Limit:       limit,
		Period:      period,
	})
}

// UserRateLimiter creates a user-based rate limiter
func UserRateLimiter(limit int, period time.Duration, redisClient *redis.Client) echo.MiddlewareFunc {
	return RateLimiterMiddleware(RateLimiterConfig{
		RedisClient: redisClient,
		Key:         "rate:user",
		Limit:       limit,
		Period:      period,
	})
}
