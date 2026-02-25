package rides

import (
	"context"

		ridemodels "github.com/piresc/nebengjek/internal/pkg/models/ride"
)

// RideGW defines the interface for ride gateway operations
//
//go:generate mockgen -destination=mocks/mock_gateway.go -package=mocks github.com/piresc/nebengjek/services/rides RideGW
type RideGW interface {
	PublishRidePickup(ctx context.Context, ride *ridemodels.Ride) error
	PublishRideStarted(ctx context.Context, ride *ridemodels.Ride) error
	PublishRideCompleted(ctx context.Context, ride ridemodels.RideComplete) error
}
