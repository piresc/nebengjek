package users

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

//go:generate mockgen -destination=mocks/mock_gateway.go -package=mocks github.com/piresc/nebengjek/services/users UserGW

// UserGW defines the user gateaways interface
type UserGW interface {
	// NATS Gateway
	PublishBeaconEvent(ctx context.Context, beaconEvent *models.BeaconEvent) error
	PublishFinderEvent(ctx context.Context, finderevent *models.FinderEvent) error
	PublishLocationUpdate(ctx context.Context, locationEvent *models.LocationUpdate) error
	PublishRideStart(ctx context.Context, startTripEvent *models.RideStartTripEvent) error

	// HTTP Gateway
	MatchConfirm(ctx context.Context, req *models.MatchConfirmRequest) (*models.MatchProposal, error)
	StartRide(ctx context.Context, req *models.RideStartRequest) (*models.Ride, error)
	RideArrived(ctx context.Context, event *models.RideArrivalReq) (*models.PaymentRequest, error)
	ProcessPayment(ctx context.Context, paymentReq *models.PaymentProccessRequest) (*models.Payment, error)
}
