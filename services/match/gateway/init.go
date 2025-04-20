package gateway

import (
	"fmt"

	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/match"
)

// matchGW handles match gateway operations
type matchGW struct {
	natsClient *natspkg.Client
}

// NewMatchGW creates a new NATS gateway instance
func NewMatchGW(natsURL string) (match.MatchGW, error) {
	client, err := natspkg.NewClient(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS client: %w", err)
	}

	return &matchGW{
		natsClient: client,
	}, nil
}

// Close closes the NATS connection
func (g *matchGW) Close() {
	if g.natsClient != nil {
		g.natsClient.Close()
	}
}
