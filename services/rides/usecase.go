package rides

import (
	"context"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// RideUseCase defines the interface for ride business logic
type RideUseCase interface {
	// Ride request operations
	CreateRideRequest(ctx context.Context, passengerID string, pickup, dropoff *models.Location) (*models.Trip, error)
	CancelRideRequest(ctx context.Context, tripID string, userID string) error

	// Driver operations
	AcceptRide(ctx context.Context, tripID string, driverID string) error
	RejectRide(ctx context.Context, tripID string, driverID string) error
	StartRide(ctx context.Context, tripID string, driverID string) error
	CompleteRide(ctx context.Context, tripID string, driverID string) error

	// Ride status operations
	GetRideStatus(ctx context.Context, tripID string) (*models.Trip, error)
	GetActiveRideForPassenger(ctx context.Context, passengerID string) (*models.Trip, error)
	GetActiveRideForDriver(ctx context.Context, driverID string) (*models.Trip, error)
	GetRideHistory(ctx context.Context, userID string, role string, startTime, endTime time.Time) ([]*models.Trip, error)

	// Fare calculation
	CalculateFare(ctx context.Context, tripID string) (*models.Fare, error)
	UpdateFare(ctx context.Context, tripID string, fare *models.Fare) error

	// Rating operations
	RateRide(ctx context.Context, tripID string, userID string, role string, rating float64) error
}
