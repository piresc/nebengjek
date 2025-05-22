package users

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// UserGW defines the user gateaways interface
type UserGW interface {
	PublishBeaconEvent(ctx context.Context, beaconReq *models.BeaconEvent) error
	MatchAccept(mp *models.MatchProposal) error
	PublishLocationUpdate(ctx context.Context, locationEvent *models.LocationUpdate) error
	PublishRideArrived(ctx context.Context, event *models.RideCompleteEvent) error

	// PublishCustomerConfirmedEvent publishes an event when a customer confirms a match.
	PublishCustomerConfirmedEvent(ctx context.Context, mp models.MatchProposal) error
	// PublishCustomerRejectedEvent publishes an event when a customer rejects a match.
	PublishCustomerRejectedEvent(ctx context.Context, mp models.MatchProposal) error
}
