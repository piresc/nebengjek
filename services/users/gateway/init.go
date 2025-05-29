package gateway

import (
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/users"
)

// UserGW handles user gateway operations
type UserGW struct {
	natsGateway *NATSGateway
	httpGateway *HTTPGateway
}

// NewUserGW creates a new gateway instance with NATS and HTTP clients
func NewUserGW(natsClient *natspkg.Client, matchServiceURL string) users.UserGW {
	return &UserGW{
		natsGateway: NewNATSGateway(natsClient),
		httpGateway: NewHTTPGateway(matchServiceURL),
	}
}
