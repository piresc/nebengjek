package user

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// UserGW defines the user gateaways interface
type UserGW interface {
	PublishBeaconEvent(ctx context.Context, userID string, beaconReq *models.BeaconRequest) error
}
