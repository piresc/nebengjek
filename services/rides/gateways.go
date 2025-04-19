package rides

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// RideGW defines the interface for ride gateway operations
type RideGW interface {
	PublishRideStarted(ctx context.Context, ride *models.Ride) error
	PublishRideCompleted(ctx context.Context, ride models.RideComplete) error
}
