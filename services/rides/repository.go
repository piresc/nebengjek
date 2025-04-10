package rides

import (
	"context"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// RideRepo defines the interface for ride data access operations
type RideRepo interface {
	// Ride CRUD operations
	CreateRide(ctx context.Context, trip *models.Trip) error
	GetRideByID(ctx context.Context, id string) (*models.Trip, error)
	UpdateRideStatus(ctx context.Context, tripID string, status models.TripStatus) error
	UpdateRideTimestamp(ctx context.Context, tripID string, field string, timestamp time.Time) error

	// Ride query operations
	GetActiveRideByPassengerID(ctx context.Context, passengerID string) (*models.Trip, error)
	GetActiveRideByDriverID(ctx context.Context, driverID string) (*models.Trip, error)
	GetRideHistory(ctx context.Context, userID string, role string, startTime, endTime time.Time) ([]*models.Trip, error)

	// Fare operations
	UpdateRideFare(ctx context.Context, tripID string, fare *models.Fare) error

	// Rating operations
	UpdateRideRating(ctx context.Context, tripID string, role string, rating float64) error
}
