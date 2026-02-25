package rides

import (
	"context"

		ridemodels "github.com/piresc/nebengjek/internal/pkg/models/ride"
)

// RideRepo defines the interface for ride data access operations
//
//go:generate mockgen -destination=mocks/mock_repository.go -package=mocks github.com/piresc/nebengjek/services/rides RideRepo
type RideRepo interface {
	CreateRide(ride *ridemodels.Ride) (*ridemodels.Ride, error)
	AddBillingEntry(ctx context.Context, entry *ridemodels.BillingLedger) error
	UpdateTotalCost(ctx context.Context, rideID string, additionalCost int) error
	GetRide(ctx context.Context, rideID string) (*ridemodels.Ride, error)
	CompleteRide(ctx context.Context, ride *ridemodels.Ride) error
	GetBillingLedgerSum(ctx context.Context, rideID string) (int, error)
	CreatePayment(ctx context.Context, payment *ridemodels.Payment) error
	UpdateRideStatus(ctx context.Context, rideID string, status ridemodels.RideStatus) error
	GetPaymentByRideID(ctx context.Context, rideID string) (*ridemodels.Payment, error)
	UpdatePaymentStatus(ctx context.Context, paymentID string, status ridemodels.PaymentStatus) error
}
