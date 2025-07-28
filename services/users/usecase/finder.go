package usecase

import (
	"context"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// UpdateFinderStatus updates a user's finder status and location with Redis caching to prevent duplicate events
func (uc *UserUC) UpdateFinderStatus(ctx context.Context, finderReq *models.FinderRequest) error {
	// Validate the request
	user, err := uc.userRepo.GetUserByMSISDN(ctx, finderReq.MSISDN)
	if err != nil {
		return err
	}

	// Atomically check and set cache key - if SetNX returns false, key already exists
	wasSet, err := uc.userRepo.SetEventCacheWithTTL(ctx, "finder_update", user.ID.String(), finderReq, 5*time.Minute)
	if err != nil {
		logger.WarnCtx(ctx, "Failed to check/set finder cache, continuing with event publish",
			logger.String("user_id", user.ID.String()),
			logger.Err(err))
	} else if !wasSet {
		logger.InfoCtx(ctx, "Finder event already processed recently, skipping duplicate",
			logger.String("user_id", user.ID.String()))
		return nil // Skip duplicate event
	}

	// Create and publish finder event
	finderEvent := &models.FinderEvent{
		UserID:         user.ID.String(),
		IsActive:       finderReq.IsActive,
		Location:       finderReq.Location,
		TargetLocation: finderReq.TargetLocation,
		Timestamp:      time.Now(),
	}

	// Publish the event (cache key already set atomically above)
	return uc.UserGW.PublishFinderEvent(ctx, finderEvent)
}
