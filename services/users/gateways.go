package users

import (
	"context"

	"github.com/piresc/nebengjek/internal/pkg/models/core"
)

//go:generate mockgen -destination=mocks/mock_gateway.go -package=mocks github.com/piresc/nebengjek/services/users UserGW

// UserGW defines the user gateways interface for external service calls
type UserGW interface {
	PublishBeaconEvent(ctx context.Context, event *core.BeaconEvent) error
	PublishFinderEvent(ctx context.Context, event *core.FinderEvent) error
}
