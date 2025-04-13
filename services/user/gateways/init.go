package gateways

import (
	"github.com/nats-io/nats.go"
)

// userGW handles user gateway operations
type UserGW struct {
	nc *nats.Conn
}

// NewNATSGateway creates a new NATS gateway instance
func NewUserGW(nc *nats.Conn) *UserGW {
	return &UserGW{
		nc: nc,
	}
}
