package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// MatchHTTPClient is an HTTP client for communicating with the match service
type MatchHTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewMatchHTTPClient creates a new match HTTP client
func NewMatchHTTPClient(matchServiceURL string) *MatchHTTPClient {
	return &MatchHTTPClient{
		baseURL: matchServiceURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// MatchConfirmRequest is the request structure for match confirmation
type MatchConfirmRequest struct {
	UserID string             `json:"userId"`
	Status models.MatchStatus `json:"status"`
}

// MatchConfirmResponse is the response structure for match confirmation
type MatchConfirmResponse struct {
	Success bool                 `json:"success"`
	Message string               `json:"message,omitempty"`
	MatchID string               `json:"matchId,omitempty"`
	Match   models.MatchProposal `json:"match,omitempty"`
}

// ConfirmMatch sends a match confirmation request to the match service
func (c *MatchHTTPClient) ConfirmMatch(matchID string, mp *models.MatchProposal) (*models.MatchProposal, error) {
	url := fmt.Sprintf("%s/matches/%s/confirm", c.baseURL, matchID)

	req := MatchConfirmRequest{
		UserID: mp.DriverID, // In our implementation, only drivers can confirm matches
		Status: mp.MatchStatus,
	}

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
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send match confirmation request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var response MatchConfirmResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode match confirmation response: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("match confirmation failed: %s", response.Message)
	}

	return &response.Match, nil
}
