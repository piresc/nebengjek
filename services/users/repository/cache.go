package repository

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// SetEventCacheWithTTL generates cache key and atomically sets it with TTL using SetNX
// Returns true if key was set (first time), false if key already exists
func (r *UserRepo) SetEventCacheWithTTL(ctx context.Context, eventType, userID string, req interface{}, ttl time.Duration) (bool, error) {
	cacheKey := r.generateEventCacheKey(eventType, userID, req)
	set, err := r.redisClient.SetNX(ctx, cacheKey, "1", ttl)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to set cache key in Redis (SetNX)",
			logger.String("cache_key", cacheKey),
			logger.String("event_type", eventType),
			logger.String("user_id", userID),
			logger.String("ttl", ttl.String()),
			logger.Err(err))
	}
	return set, err
}

// generateEventCacheKey creates a unique cache key for event deduplication
func (r *UserRepo) generateEventCacheKey(eventType, userID string, req interface{}) string {
	var content string
	
	switch r := req.(type) {
	case *models.FinderRequest:
		content = fmt.Sprintf("%s:%t:%.6f:%.6f:%.6f:%.6f", 
			userID,
			r.IsActive,
			r.Location.Latitude,
			r.Location.Longitude,
			r.TargetLocation.Latitude,
			r.TargetLocation.Longitude,
		)
	case *models.BeaconRequest:
		content = fmt.Sprintf("%s:%t:%.6f:%.6f", 
			userID,
			r.IsActive,
			r.Latitude,
			r.Longitude,
		)
	default:
		// Fallback for unknown types
		content = fmt.Sprintf("%s:%v", userID, req)
	}
	
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%s:%s:%x", eventType, userID, hash)
}