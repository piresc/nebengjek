package match

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// MatchGW defines the match gateaways interface
type MatchGW interface {
	PublishBeaconEvent(ctx context.Context, userID string, beaconReq *models.BeaconRequest) error
}
