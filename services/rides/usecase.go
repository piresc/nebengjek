package rides

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// RideUC defines the interface for ride business logic
//
//go:generate mockgen -destination=mocks/mock_usecase.go -package=mocks github.com/piresc/nebengjek/services/rides RideUC
type RideUC interface {
	CreateRide(ctx context.Context, mp models.MatchProposal) error
	ProcessBillingUpdate(ctx context.Context, rideID string, entry *models.BillingLedger) error
	StartRide(ctx context.Context, req models.RideStartRequest) (*models.Ride, error)
	RideArrived(ctx context.Context, req models.RideArrivalReq) (*models.PaymentRequest, error)
	ProcessPayment(ctx context.Context, req models.PaymentProccessRequest) (*models.Payment, error)
}
