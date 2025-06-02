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

// RideStartTrip publishes a ride start trip event to NATS
func (u *UserUC) RideStart(ctx context.Context, event *models.RideStartRequest) (*models.Ride, error) {

	req := &models.RideStartRequest{
		RideID:            event.RideID,
		DriverLocation:    event.DriverLocation,
		PassengerLocation: event.PassengerLocation,
	}

	// Make HTTP call to rides service
	resp, err := u.UserGW.StartRide(req)
	if err != nil {
		return nil, fmt.Errorf("failed to start ride via HTTP: %w", err)
	}

	return resp, nil
}
