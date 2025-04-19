package user

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// UserGW defines the user gateaways interface
type UserGW interface {
	PublishBeaconEvent(ctx context.Context, beaconReq *models.BeaconRequest) error
	MatchAccept(mp *models.MatchProposal) error
	PublishLocationUpdate(ctx context.Context, locationEvent *models.LocationUpdate) error
	PublishRideArrived(ctx context.Context, event *models.RideCompleteEvent) error
}
