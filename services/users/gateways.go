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
	PublishRideArrived(ctx context.Context, RideCompleteEvent *models.RideCompleteEvent) error

	// HTTP Gateway
	MatchConfirm(req *models.MatchConfirmRequest) (*models.MatchProposal, error)
	StartRide(req *models.RideStartRequest) (*models.Ride, error)
}
