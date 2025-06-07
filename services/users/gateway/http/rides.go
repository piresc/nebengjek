package gateaway_http

import (
	"context"
	"fmt"
	"time"

	httpclient "github.com/piresc/nebengjek/internal/pkg/http"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/pkg/observability"
)

// RideClient is a simplified HTTP client for communicating with the ride service
type RideClient struct {
	client *httpclient.UnifiedClient
	tracer observability.Tracer
}

// NewRideClient creates a new simplified ride HTTP client with API key authentication
func NewRideClient(rideServiceURL string, config *models.APIKeyConfig, tracer observability.Tracer) *RideClient {
	return &RideClient{
		client: httpclient.NewUnifiedClient(httpclient.UnifiedConfig{
			APIKey:  config.RidesService,
			BaseURL: rideServiceURL,
			Timeout: 30 * time.Second,
		}),
		tracer: tracer,
	}
}

// StartRide sends a start trip request to the ride service with simplified retry
func (g *HTTPGateway) StartRide(ctx context.Context, req *models.RideStartRequest) (*models.Ride, error) {
	endpoint := fmt.Sprintf("/internal/rides/%s/start", req.RideID)

	// Start APM segment if tracer is available
	var endSegment func()
	if g.rideClient.tracer != nil {
		ctx, endSegment = g.rideClient.tracer.StartSegment(ctx, "External/rides-service/start")
		defer endSegment()
	}

	var ride models.Ride
	err := g.rideClient.client.PostJSON(ctx, endpoint, req, &ride)
	if err != nil {
		return nil, fmt.Errorf("failed to start ride: %w", err)
	}
	return &ride, nil
}

// RideArrived sends a ride arrival notification to the ride service with simplified retry
func (g *HTTPGateway) RideArrived(ctx context.Context, req *models.RideArrivalReq) (*models.PaymentRequest, error) {
	endpoint := fmt.Sprintf("/internal/rides/%s/arrive", req.RideID)

	// Start APM segment if tracer is available
	var endSegment func()
	if g.rideClient.tracer != nil {
		ctx, endSegment = g.rideClient.tracer.StartSegment(ctx, "External/rides-service/arrive")
		defer endSegment()
	}

	var paymentRequest models.PaymentRequest
	err := g.rideClient.client.PostJSON(ctx, endpoint, req, &paymentRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to process ride arrival: %w", err)
	}
	return &paymentRequest, nil
}

// ProcessPayment sends a payment processing request to the ride service with simplified retry
func (g *HTTPGateway) ProcessPayment(ctx context.Context, paymentReq *models.PaymentProccessRequest) (*models.Payment, error) {
	endpoint := fmt.Sprintf("/internal/rides/%s/payment", paymentReq.RideID)

	// Start APM segment if tracer is available
	var endSegment func()
	if g.rideClient.tracer != nil {
		ctx, endSegment = g.rideClient.tracer.StartSegment(ctx, "External/rides-service/payment")
		defer endSegment()
	}

	var payment models.Payment
	err := g.rideClient.client.PostJSON(ctx, endpoint, paymentReq, &payment)
	if err != nil {
		return nil, fmt.Errorf("failed to process payment: %w", err)
	}
	return &payment, nil
}
