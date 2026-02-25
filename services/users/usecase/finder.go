package usecase

import (
	"context"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models/core"
)

// UpdateFinderStatus updates a user's finder status and location with Redis caching to prevent duplicate events
func (uc *UserUC) UpdateFinderStatus(ctx context.Context, finderReq *core.FinderRequest) error {
	// Validate the request
	user, err := uc.userRepo.GetUserByMSISDN(ctx, finderReq.MSISDN)
	if err != nil {
		return err
	}

	// Atomically check and set cache key - if SetNX returns false, key already exists
	cacheTTL := time.Duration(uc.cfg.Users.FinderCacheTTLMinutes) * time.Minute
	wasSet, err := uc.userRepo.SetEventCacheWithTTL(ctx, "finder_update", user.ID.String(), finderReq, cacheTTL)
	if err != nil {
		logger.WarnCtx(ctx, "Failed to check/set finder cache, continuing with event publish",
			logger.String("user_id", user.ID.String()),
			logger.Err(err))
	} else if !wasSet {
		logger.InfoCtx(ctx, "Finder event already processed recently, skipping duplicate",
			logger.String("user_id", user.ID.String()))
		// Duplicate event detected within TTL window; skipping processing
		// This prevents duplicate finder events from being processed and published to NATS
		return nil
	}

	// Debug logging to check target location data
	logger.InfoCtx(ctx, "Processing finder update with target location",
		logger.String("user_id", user.ID.String()),
		logger.Bool("is_active", finderReq.IsActive),
		logger.Float64("target_lat", finderReq.TargetLocation.Latitude),
		logger.Float64("target_lon", finderReq.TargetLocation.Longitude),
		logger.String("target_timestamp", time.Now().String()))

	// Create and publish finder event
	finderEvent := &core.FinderEvent{
		UserID:         user.ID.String(),
		IsActive:       finderReq.IsActive,
		Location:       finderReq.Location,
		TargetLocation: finderReq.TargetLocation,
		Timestamp:      time.Now(),
	}

	// Publish the event (cache key already set atomically above)
	return uc.UserGW.PublishFinderEvent(ctx, finderEvent)
}
