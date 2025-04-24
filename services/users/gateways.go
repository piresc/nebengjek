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
}
