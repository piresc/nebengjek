package rides

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// RideRepo defines the interface for ride data access operations
type RideRepo interface {
	CreateRide(ride *models.Ride) (*models.Ride, error)
	AddBillingEntry(ctx context.Context, entry *models.BillingLedger) error
	UpdateTotalCost(ctx context.Context, rideID string, additionalCost int) error
	GetRide(ctx context.Context, rideID string) (*models.Ride, error)
}
