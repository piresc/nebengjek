package gateway

import (
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/users"
	gateaway_http "github.com/piresc/nebengjek/services/users/gateway/http"
	gateway_nats "github.com/piresc/nebengjek/services/users/gateway/nats"
)

// UserGW handles user gateway operations
type UserGW struct {
	natsGateway *gateway_nats.NATSGateway
	httpGateway *gateaway_http.HTTPGateway
}

// NewUserGW creates a new gateway instance with NATS and HTTP clients
func NewUserGW(natsClient *natspkg.Client, matchServiceURL string, rideServiceURL string) users.UserGW {
	return &UserGW{
		natsGateway: gateway_nats.NewNATSGateway(natsClient),
		httpGateway: gateaway_http.NewHTTPGateway(matchServiceURL, rideServiceURL),
	}
}
