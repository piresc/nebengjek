package rides

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// RideRepo defines the interface for ride data access operations
// go:generate mockgen -destination=../mocks/mock_repository.go -package=mocks github.com/piresc/nebengjek/services/rides RideRepo
type RideRepo interface {
	CreateRide(ride *models.Ride) (*models.Ride, error)
	AddBillingEntry(ctx context.Context, entry *models.BillingLedger) error
	UpdateTotalCost(ctx context.Context, rideID string, additionalCost int) error
	GetRide(ctx context.Context, rideID string) (*models.Ride, error)
	CompleteRide(ctx context.Context, ride *models.Ride) error
	GetBillingLedgerSum(ctx context.Context, rideID string) (int, error)
	CreatePayment(ctx context.Context, payment *models.Payment) error
	UpdateRideStatus(ctx context.Context, rideID string, status models.RideStatus) error
	GetPaymentByRideID(ctx context.Context, rideID string) (*models.Payment, error)
}
