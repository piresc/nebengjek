package gateway

import (
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/users"
)

// UserGW handles user gateway operations
type UserGW struct {
	natsClient      *natspkg.Client
	matchHTTPClient *MatchHTTPClient
}

// NewUserGW creates a new gateway instance with NATS and HTTP clients
func NewUserGW(natsClient *natspkg.Client, matchServiceURL string) users.UserGW {
	return &UserGW{
		natsClient:      natsClient,
		matchHTTPClient: NewMatchHTTPClient(matchServiceURL),
	}
}
