package gateaway_http

import (
	"context"
	"fmt"
	"time"

	httpclient "github.com/piresc/nebengjek/internal/pkg/http"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// MatchClient is an HTTP client for communicating with the match service
type MatchClient struct {
	client    *httpclient.Client
	apiClient *httpclient.APIKeyClient
}

// NewMatchClient creates a new match HTTP client
func NewMatchClient(matchServiceURL string) *MatchClient {
	return &MatchClient{
		client: httpclient.NewClient(matchServiceURL, 10*time.Second),
	}
}

// NewMatchClientWithAPIKey creates a new match HTTP client with API key authentication
func NewMatchClientWithAPIKey(matchServiceURL string, config *models.APIKeyConfig) *MatchClient {
	return &MatchClient{
		client:    httpclient.NewClient(matchServiceURL, 10*time.Second),
		apiClient: httpclient.NewAPIKeyClient(config, "match-service", matchServiceURL),
	}
}

// HTTPGateway implements the HTTP client operations for the users service
type HTTPGateway struct {
	matchClient *MatchClient
	rideClient  *RideClient
}

// NewHTTPGateway creates a new HTTP gateway for the users service
func NewHTTPGateway(matchServiceURL string, rideServiceURL string) *HTTPGateway {
	return &HTTPGateway{
		matchClient: NewMatchClient(matchServiceURL),
		rideClient:  NewRideClient(rideServiceURL),
	}
}

// NewHTTPGatewayWithAPIKey creates a new HTTP gateway with API key authentication
func NewHTTPGatewayWithAPIKey(matchServiceURL string, rideServiceURL string, config *models.APIKeyConfig) *HTTPGateway {
	return &HTTPGateway{
		matchClient: NewMatchClientWithAPIKey(matchServiceURL, config),
		rideClient:  NewRideClientWithAPIKey(rideServiceURL, config),
	}
}

// MatchAccept sends a match confirmation request to the match service
func (g *HTTPGateway) MatchConfirm(req *models.MatchConfirmRequest) (*models.MatchProposal, error) {
	ctx := context.Background()
	endpoint := fmt.Sprintf("/internal/matches/%s/confirm", req.ID)

	// Ensure API key client is available
	if g.matchClient.apiClient == nil {
		return nil, fmt.Errorf("API key client not configured for match service")
	}

	var matchProposal models.MatchProposal
	err := g.matchClient.apiClient.PostJSON(ctx, endpoint, req, &matchProposal)
	if err != nil {
		return nil, fmt.Errorf("failed to send match confirmation request: %w", err)
	}
	return &matchProposal, nil
}
