package gateaway_http

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

// RideClient is an HTTP client for communicating with the ride service
type RideClient struct {
	client *httpclient.Client
}

// NewRideClient creates a new ride HTTP client
func NewRideClient(rideServiceURL string) *RideClient {
	return &RideClient{
		client: httpclient.NewClient(rideServiceURL, 10*time.Second),
	}
}

// StartTrip sends a start trip request to the ride service
func (g *HTTPGateway) StartRide(req *models.RideStartRequest) (*models.Ride, error) {
	url := fmt.Sprintf("%s/rides/%s/confirm", g.rideClient.client.BaseURL, req.RideID)

	// Marshal request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal start trip request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := g.rideClient.client.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send start trip request: %w", err)
	}
	defer resp.Body.Close()

	// Read the full response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("start trip request failed: (status: %d, body: %s)", resp.StatusCode, string(respBody))
	}

	// Parse the response using our utility function
	var response models.Ride
	if err := utils.ParseJSONResponse(respBody, &response); err != nil {
		// Log the raw response for debugging
		log.Printf("Error parsing response. Raw response body: %s", string(respBody))
		return nil, fmt.Errorf("failed to parse start trip response: %w", err)
	}

	return &response, nil
}
