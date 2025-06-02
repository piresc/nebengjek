package rides

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// RideGW defines the interface for ride gateway operations
// go:generate mockgen -destination=../mocks/mock_gateway.go -package=mocks github.com/piresc/nebengjek/services/rides RideGW
type RideGW interface {
	PublishRidePickup(ctx context.Context, ride *models.Ride) error
	PublishRideStarted(ctx context.Context, ride *models.Ride) error
	PublishRideCompleted(ctx context.Context, ride models.RideComplete) error
}
