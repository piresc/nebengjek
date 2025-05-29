package users

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

//go:generate mockgen -destination=mocks/mock_gateway.go -package=mocks github.com/piresc/nebengjek/services/users UserGW

// UserGW defines the user gateaways interface
type UserGW interface {
	// NATS Gateway
	PublishBeaconEvent(ctx context.Context, beaconReq *models.BeaconEvent) error
	PublishLocationUpdate(ctx context.Context, locationEvent *models.LocationUpdate) error
	PublishRideArrived(ctx context.Context, event *models.RideCompleteEvent) error

	// HTTP Gateway
	MatchConfirm(req *models.MatchConfirmRequest) (*models.MatchProposal, error)
}
