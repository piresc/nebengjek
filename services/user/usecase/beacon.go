package usecase

import (
	"context"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// UpdateBeaconStatus updates a user's beacon status and location
func (uc *UserUC) UpdateBeaconStatus(ctx context.Context, beaconReq *models.BeaconRequest) error {
	// Update user's beacon status in database
	if _, err := uc.userRepo.GetUserByMSISDN(ctx, beaconReq.MSISDN); err != nil {
		return err
	}

	// Create and publish beacon event
	beaconEvent := &models.BeaconEvent{
		MSISDN:   beaconReq.MSISDN,
		IsActive: beaconReq.IsActive,
		Role:     beaconReq.Role,
		Location: models.GeoLocation{
			Latitude:  beaconReq.Latitude,
			Longitude: beaconReq.Longitude,
		},
		Timestamp: time.Now(),
	}

	return uc.UserGW.PublishBeaconEvent(beaconEvent)
}
