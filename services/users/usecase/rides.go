package usecase

import (
	"context"
	"fmt"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// RideArrived publishes a ride arrival event to NATS
func (u *UserUC) RideArrived(ctx context.Context, event *models.RideArrivalReq) (*models.PaymentRequest, error) {
	// First notify the ride service about the arrival via HTTP
	paymentReq, err := u.UserGW.RideArrived(ctx, event)
	if err != nil {
		return nil, fmt.Errorf("failed to notify ride service of arrival via HTTP: %w", err)
	}

	return paymentReq, nil
}

// ProcessPayment processes the payment for a completed ride
func (u *UserUC) ProcessPayment(ctx context.Context, paymentReq *models.PaymentRequest) (*models.Payment, error) {
	// Call the ride service to process the payment
	payment, err := u.UserGW.ProcessPayment(paymentReq)
	if err != nil {
		return nil, fmt.Errorf("failed to process payment: %w", err)
	}

	return payment, nil
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
