package gateaway_http

import (
	"context"
	"fmt"
	"time"

	httpclient "github.com/piresc/nebengjek/internal/pkg/http"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/pkg/observability"
)

// MatchClient is an HTTP client for communicating with the match service
type MatchClient struct {
	client *httpclient.UnifiedClient
	tracer observability.Tracer
}

// NewMatchClient creates a new match HTTP client with API key authentication
func NewMatchClient(matchServiceURL string, config *models.APIKeyConfig, tracer observability.Tracer) *MatchClient {
	return &MatchClient{
		client: httpclient.NewUnifiedClient(httpclient.UnifiedConfig{
			APIKey:  config.MatchService,
			BaseURL: matchServiceURL,
			Timeout: 30 * time.Second,
		}),
		tracer: tracer,
	}
}

// HTTPGateway implements the HTTP client operations for the users service
type HTTPGateway struct {
	matchClient *MatchClient
	rideClient  *RideClient
}

// NewHTTPGateway creates a new HTTP gateway for the users service with API key authentication
func NewHTTPGateway(matchServiceURL string, rideServiceURL string, config *models.APIKeyConfig, tracer observability.Tracer) *HTTPGateway {
	return &HTTPGateway{
		matchClient: NewMatchClient(matchServiceURL, config, tracer),
		rideClient:  NewRideClient(rideServiceURL, config, tracer),
	}
}

// MatchConfirm sends a match confirmation request to the match service
func (g *HTTPGateway) MatchConfirm(ctx context.Context, req *models.MatchConfirmRequest) (*models.MatchProposal, error) {
	endpoint := fmt.Sprintf("/internal/matches/%s/confirm", req.ID)

	// Start APM segment if tracer is available
	var endSegment func()
	if g.matchClient.tracer != nil {
		ctx, endSegment = g.matchClient.tracer.StartSegment(ctx, "External/match-service/confirm")
		defer endSegment()
	}

	var matchProposal models.MatchProposal
	err := g.matchClient.client.PostJSON(ctx, endpoint, req, &matchProposal)
	if err != nil {
		return nil, fmt.Errorf("failed to send match confirmation request: %w", err)
	}
	return &matchProposal, nil
}
