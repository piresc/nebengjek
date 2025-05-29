package usecase

import (
	"context"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// UpdateBeaconStatus updates a user's beacon status and location
func (uc *UserUC) UpdateBeaconStatus(ctx context.Context, beaconReq *models.BeaconRequest) error {
	// Validate the request
	user, err := uc.userRepo.GetUserByMSISDN(ctx, beaconReq.MSISDN)
	if err != nil {
		return err
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

	return uc.UserGW.PublishBeaconEvent(ctx, beaconEvent)
}
