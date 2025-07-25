package users

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

//go:generate mockgen -destination=mocks/mock_gateway.go -package=mocks github.com/piresc/nebengjek/services/users UserGW

// UserGW defines the user gateways interface
type UserGW interface {
	// NATS Gateway - publishes events for Match service consumption
	PublishBeaconEvent(ctx context.Context, event *models.BeaconEvent) error
	PublishFinderEvent(ctx context.Context, event *models.FinderEvent) error
}
