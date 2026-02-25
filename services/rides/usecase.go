package rides

import (
	"context"

		ridemodels "github.com/piresc/nebengjek/internal/pkg/models/ride"
	matchmodels "github.com/piresc/nebengjek/internal/pkg/models/match"
)

// RideUC defines the interface for ride business logic
//
//go:generate mockgen -destination=mocks/mock_usecase.go -package=mocks github.com/piresc/nebengjek/services/rides RideUC
type RideUC interface {
	CreateRide(ctx context.Context, mp matchmodels.MatchProposal) error
	ProcessBillingUpdate(ctx context.Context, rideID string, entry *ridemodels.BillingLedger) error
	StartRide(ctx context.Context, req ridemodels.RideStartRequest) (*ridemodels.Ride, error)
	RideArrived(ctx context.Context, req ridemodels.RideArrivalReq) (*ridemodels.PaymentRequest, error)
	ProcessPayment(ctx context.Context, req ridemodels.PaymentProccessRequest) (*ridemodels.Payment, error)
}
