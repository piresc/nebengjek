package gateway

import (
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/users"
)

// UserGW handles user gateway operations
type UserGW struct {
	natsClient *natspkg.Client
}

// NewUserGW creates a new NATS gateway instance
func NewUserGW(client *natspkg.Client) users.UserGW {
	return &UserGW{
		natsClient: client,
	}
}
