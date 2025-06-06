package gateaway_http

import (
	"context"
	"fmt"
	"time"

	httpclient "github.com/piresc/nebengjek/internal/pkg/http"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

// RideClient is an enhanced HTTP client for communicating with the ride service
type RideClient struct {
	client    *httpclient.Client
	apiClient *httpclient.APIKeyClient
}

// NewRideClient creates a new enhanced ride HTTP client with API key authentication
func NewRideClient(rideServiceURL string, config *models.APIKeyConfig) *RideClient {
	return &RideClient{
		client:    httpclient.NewClient(rideServiceURL, 10*time.Second),
		apiClient: httpclient.NewAPIKeyClient(config, "rides-service", rideServiceURL),
	}
}

// StartRide sends a start trip request to the ride service with retry and circuit breaker
func (g *HTTPGateway) StartRide(req *models.RideStartRequest) (*models.Ride, error) {
	ctx := context.Background()
	endpoint := fmt.Sprintf("/internal/rides/%s/confirm", req.RideID)

	// Ensure API key client is available
	if g.rideClient.apiClient == nil {
		return nil, fmt.Errorf("API key client not configured for ride service")
	}

	var ride models.Ride
	err := g.rideClient.apiClient.PostJSON(ctx, endpoint, req, &ride)
	if err != nil {
		return nil, fmt.Errorf("failed to start ride: %w", err)
	}
	return &ride, nil
}

// RideArrived sends a ride arrival notification to the ride service with retry and circuit breaker
func (g *HTTPGateway) RideArrived(req *models.RideArrivalReq) (*models.PaymentRequest, error) {
	ctx := context.Background()
	endpoint := fmt.Sprintf("/internal/rides/%s/arrive", req.RideID)

	// Ensure API key client is available
	if g.rideClient.apiClient == nil {
		return nil, fmt.Errorf("API key client not configured for ride service")
	}

	var paymentRequest models.PaymentRequest
	err := g.rideClient.apiClient.PostJSON(ctx, endpoint, req, &paymentRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to process ride arrival: %w", err)
	}
	return &paymentRequest, nil
}

// ProcessPayment sends a payment processing request to the ride service with retry and circuit breaker
func (g *HTTPGateway) ProcessPayment(paymentReq *models.PaymentProccessRequest) (*models.Payment, error) {
	ctx := context.Background()
	endpoint := fmt.Sprintf("/internal/rides/%s/payment", paymentReq.RideID)

	// Ensure API key client is available
	if g.rideClient.apiClient == nil {
		return nil, fmt.Errorf("API key client not configured for ride service")
	}

	var payment models.Payment
	err := g.rideClient.apiClient.PostJSON(ctx, endpoint, paymentReq, &payment)
	if err != nil {
		return nil, fmt.Errorf("failed to process payment: %w", err)
	}
	return &payment, nil
}
