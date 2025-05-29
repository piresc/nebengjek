package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	httpclient "github.com/piresc/nebengjek/internal/pkg/http"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/utils"
)

// MatchClient is an HTTP client for communicating with the match service
type MatchClient struct {
	client *httpclient.Client
}

// NewMatchClient creates a new match HTTP client
func NewMatchClient(matchServiceURL string) *MatchClient {
	return &MatchClient{
		client: httpclient.NewClient(matchServiceURL, 10*time.Second),
	}
}

// HTTPGateway implements the HTTP client operations for the users service
type HTTPGateway struct {
	matchClient *MatchClient
}

// NewHTTPGateway creates a new HTTP gateway for the users service
func NewHTTPGateway(matchServiceURL string) *HTTPGateway {
	return &HTTPGateway{
		matchClient: NewMatchClient(matchServiceURL),
	}
}

// MatchAccept sends a match confirmation request to the match service
func (g *HTTPGateway) MatchConfirm(req *models.MatchConfirmRequest) (*models.MatchProposal, error) {
	url := fmt.Sprintf("%s/matches/%s/confirm", g.matchClient.client.BaseURL, req.ID)

	// Marshal request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal match confirmation request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := g.matchClient.client.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send match confirmation request: %w", err)
	}
	defer resp.Body.Close()

	// Read the full response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("match confirmation failed: (status: %d, body: %s)", resp.StatusCode, string(respBody))
	}

	// Parse the response using our utility function
	var matchProposal models.MatchProposal
	if err := utils.ParseJSONResponse(respBody, &matchProposal); err != nil {
		// Log the raw response for debugging
		log.Printf("Error parsing response. Raw response body: %s", string(respBody))
		return nil, fmt.Errorf("failed to parse match confirmation response: %w", err)
	}

	return &matchProposal, nil
}
