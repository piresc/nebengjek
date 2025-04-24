package usecase

import (
	"context"
	"fmt"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// RideArrived publishes a ride arrival event to NATS
func (u *UserUC) RideArrived(ctx context.Context, event *models.RideCompleteEvent) error {
	err := u.UserGW.PublishRideArrived(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to publish ride arrived event: %w", err)
	}
	return nil
}
