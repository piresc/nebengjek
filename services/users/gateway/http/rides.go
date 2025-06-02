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

// RideArrived sends a ride arrival notification to the ride service
func (g *HTTPGateway) RideArrived(req *models.RideArrivalReq) (*models.PaymentRequest, error) {
	url := fmt.Sprintf("%s/rides/%s/arrive", g.rideClient.client.BaseURL, req.RideID)

	// Marshal request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal start trip request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := g.rideClient.client.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send ride arrival request: %w", err)
	}
	defer resp.Body.Close()

	// Read the full response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ride arrival request failed: (status: %d, body: %s)", resp.StatusCode, string(respBody))
	}

	// Parse the response
	var paymentRequest models.PaymentRequest
	if err := utils.ParseJSONResponse(respBody, &paymentRequest); err != nil {
		// Log the raw response for debugging
		log.Printf("Error parsing response. Raw response body: %s", string(respBody))
		return nil, fmt.Errorf("failed to parse ride arrival response: %w", err)
	}

	// Apply the adjustment factor from the request
	paymentRequest.AdjustmentFactor = req.AdjustmentFactor

	return &paymentRequest, nil
}

// ProcessPayment sends a payment processing request to the ride service
func (g *HTTPGateway) ProcessPayment(paymentReq *models.PaymentProccessRequest) (*models.Payment, error) {
	url := fmt.Sprintf("%s/rides/%s/payment", g.rideClient.client.BaseURL, paymentReq.RideID)

	// Marshal request to JSON
	reqBody, err := json.Marshal(paymentReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payment request: %w", err)
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
		return nil, fmt.Errorf("failed to send payment request: %w", err)
	}
	defer resp.Body.Close()

	// Read the full response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("payment request failed: (status: %d, body: %s)", resp.StatusCode, string(respBody))
	}

	// Parse the response
	var payment models.Payment
	if err := utils.ParseJSONResponse(respBody, &payment); err != nil {
		// Log the raw response for debugging
		log.Printf("Error parsing response. Raw response body: %s", string(respBody))
		return nil, fmt.Errorf("failed to parse payment response: %w", err)
	}

	return &payment, nil
}
