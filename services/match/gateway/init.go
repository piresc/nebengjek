package gateway

import (
	"github.com/nats-io/nats.go"
)

// userGW handles user gateway operations
type matchGW struct {
	nc *nats.Conn
}

// NewMatchGW creates a new NATS gateway instance
func NewMatchGW(nc *nats.Conn) *matchGW {
	return &matchGW{
		nc: nc,
	}
}
