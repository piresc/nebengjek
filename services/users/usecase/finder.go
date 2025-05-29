package usecase

import (
	"context"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// UpdateFinderStatus updates a user's finder status and location
func (uc *UserUC) UpdateFinderStatus(ctx context.Context, finderReq *models.FinderRequest) error {
	// Validate the request
	user, err := uc.userRepo.GetUserByMSISDN(ctx, finderReq.MSISDN)
	if err != nil {
		return err
	}

	// Create and publish finder event
	finderEvent := &models.FinderEvent{
		UserID:         user.ID.String(),
		IsActive:       finderReq.IsActive,
		Location:       finderReq.Location,
		TargetLocation: finderReq.TargetLocation,
		Timestamp:      time.Now(),
	}

	return uc.UserGW.PublishFinderEvent(ctx, finderEvent)
}
