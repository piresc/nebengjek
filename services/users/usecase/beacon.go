package usecase

import (
	"context"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// UpdateBeaconStatus updates a user's beacon status and location with Redis caching to prevent duplicate events
func (uc *UserUC) UpdateBeaconStatus(ctx context.Context, beaconReq *models.BeaconRequest) error {
	// Validate the request
	user, err := uc.userRepo.GetUserByMSISDN(ctx, beaconReq.MSISDN)
	if err != nil {
		return err
	}

	// Atomically check and set cache key - if SetNX returns false, key already exists
	cacheTTL := time.Duration(uc.cfg.Users.BeaconCacheTTLMinutes) * time.Minute
	wasSet, err := uc.userRepo.SetEventCacheWithTTL(ctx, "beacon_update", user.ID.String(), beaconReq, cacheTTL)
	if err != nil {
		logger.WarnCtx(ctx, "Failed to check/set beacon cache, continuing with event publish",
			logger.String("user_id", user.ID.String()),
			logger.Err(err))
	} else if !wasSet {
		logger.InfoCtx(ctx, "Beacon event already processed recently, skipping duplicate",
			logger.String("user_id", user.ID.String()))
		// Duplicate event detected within TTL window; skipping processing
		// This prevents duplicate beacon events from being processed and published to NATS
		return nil
	}

	// Create and publish beacon event
	beaconEvent := &models.BeaconEvent{
		UserID:   user.ID.String(),
		IsActive: beaconReq.IsActive,
		Location: models.Location{
			Latitude:  beaconReq.Latitude,
			Longitude: beaconReq.Longitude,
		},
		Timestamp: time.Now(),
	}

	// Publish the event (cache key already set atomically above)
	return uc.UserGW.PublishBeaconEvent(ctx, beaconEvent)
}
