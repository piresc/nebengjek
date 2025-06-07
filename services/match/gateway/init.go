package gateway

import (
	"github.com/piresc/nebengjek/internal/pkg/models"
	natspkg "github.com/piresc/nebengjek/internal/pkg/nats"
	"github.com/piresc/nebengjek/services/match"
	gateway_nats "github.com/piresc/nebengjek/services/match/gateway/nats"
)

// MatchGW handles match gateway operations
type MatchGW struct {
	natsGateway *gateway_nats.NATSGateway
	httpGateway *HTTPGateway
}

// NewMatchGW creates a new unified gateway instance with NATS and HTTP clients with API key authentication
func NewMatchGW(natsClient *natspkg.Client, locationServiceURL string, config *models.APIKeyConfig) match.MatchGW {
	return &MatchGW{
		natsGateway: gateway_nats.NewNATSGateway(natsClient),
		httpGateway: NewHTTPGateway(locationServiceURL, config),
	}
}
