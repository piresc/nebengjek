package gateway

import (
	"fmt"

	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/user"
)

// userGW handles user gateway operations
type UserGW struct {
	natsClient *natspkg.Client
}

// NewUserGW creates a new NATS gateway instance
func NewUserGW(natsURL string) (user.UserGW, error) {
	client, err := natspkg.NewClient(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS client: %w", err)
	}

	return &UserGW{
		natsClient: client,
	}, nil
}

// Close closes the NATS connection
func (g *UserGW) Close() {
	if g.natsClient != nil {
		g.natsClient.Close()
	}
}
