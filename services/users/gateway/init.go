package gateway

import (
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/users"
	gateway_nats "github.com/piresc/nebengjek/services/users/gateway/nats"
)

// UserGW handles user gateway operations for external services
type UserGW struct {
	natsGateway *gateway_nats.NATSGateway
}

// NewUserGW creates a new gateway instance for external service calls
func NewUserGW(natsClient *natspkg.Client) users.UserGW {
	return &UserGW{
		natsGateway: gateway_nats.NewNATSGateway(natsClient),
	}
}
