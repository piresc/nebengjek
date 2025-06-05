package gateaway_http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	httpclient "github.com/piresc/nebengjek/internal/pkg/http"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/internal/utils"
)

// RideClient is an enhanced HTTP client for communicating with the ride service
type RideClient struct {
	client         *httpclient.Client
	enhancedClient *httpclient.EnhancedClient
	apiClient      *httpclient.APIKeyClient
	baseURL        string
}

// NewRideClient creates a new enhanced ride HTTP client
func NewRideClient(rideServiceURL string) *RideClient {
	return &RideClient{
		client:         httpclient.NewClient(rideServiceURL, 10*time.Second),
		enhancedClient: httpclient.NewEnhancedClient(logger.GetGlobalLogger(), 10*time.Second),
		baseURL:        rideServiceURL,
	}
}

// NewRideClientWithAPIKey creates a new enhanced ride HTTP client with API key authentication
func NewRideClientWithAPIKey(rideServiceURL string, config *models.APIKeyConfig) *RideClient {
	return &RideClient{
		client:         httpclient.NewClient(rideServiceURL, 10*time.Second),
		enhancedClient: httpclient.NewEnhancedClient(logger.GetGlobalLogger(), 10*time.Second),
		apiClient:      httpclient.NewAPIKeyClient(config, "rides-service", rideServiceURL),
		baseURL:        rideServiceURL,
	}
}

// StartRide sends a start trip request to the ride service with retry and circuit breaker
func (g *HTTPGateway) StartRide(req *models.RideStartRequest) (*models.Ride, error) {
	ctx := context.Background()
	endpoint := fmt.Sprintf("/rides/%s/confirm", req.RideID)

	// Use API key client if available, otherwise fallback to regular client
	if g.rideClient.apiClient != nil {
		var ride models.Ride
		err := g.rideClient.apiClient.PostJSON(ctx, endpoint, req, &ride)
		if err != nil {
			return nil, fmt.Errorf("failed to start ride with API key: %w", err)
		}
		return &ride, nil
	}

	// Fallback to original implementation
	url := fmt.Sprintf("%s/rides/%s/confirm", g.rideClient.baseURL, req.RideID)

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

	// Log the request attempt
	// logger.Info("Making start ride request",
	//	logger.String("url", url),
	//	logger.String("ride_id", req.RideID),
	//)

	// Send request with enhanced client (includes retry and circuit breaker)
	resp, err := g.rideClient.enhancedClient.Do(ctx, httpReq)
	if err != nil {
		logger.Error("Start ride request failed",
			logger.String("ride_id", req.RideID),
			logger.ErrorField(err),
		)
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
		logger.Error("Start ride request returned error",
			logger.String("ride_id", req.RideID),
			logger.Int("status_code", resp.StatusCode),
			logger.String("response_body", string(respBody)),
		)
		return nil, fmt.Errorf("start trip request failed: (status: %d, body: %s)", resp.StatusCode, string(respBody))
	}

	// Parse the response using our utility function
	var response models.Ride
	if err := utils.ParseJSONResponse(respBody, &response); err != nil {
		// Log the raw response for debugging
		logger.Error("Failed to parse start ride response",
			logger.String("ride_id", req.RideID),
			logger.String("raw_response", string(respBody)),
			logger.ErrorField(err),
		)
		return nil, fmt.Errorf("failed to parse start trip response: %w", err)
	}

	// logger.Info("Start ride request successful",
	//	logger.String("ride_id", req.RideID),
	//	logger.String("status", string(response.Status)),
	//)

	return &response, nil
}

// RideArrived sends a ride arrival notification to the ride service with retry and circuit breaker
func (g *HTTPGateway) RideArrived(req *models.RideArrivalReq) (*models.PaymentRequest, error) {
	ctx := context.Background()
	endpoint := fmt.Sprintf("/rides/%s/arrive", req.RideID)

	// Use API key client if available, otherwise fallback to regular client
	if g.rideClient.apiClient != nil {
		var paymentRequest models.PaymentRequest
		err := g.rideClient.apiClient.PostJSON(ctx, endpoint, req, &paymentRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to process ride arrival with API key: %w", err)
		}
		return &paymentRequest, nil
	}

	// Fallback to original implementation
	url := fmt.Sprintf("%s/rides/%s/arrive", g.rideClient.baseURL, req.RideID)

	// Marshal request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ride arrival request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Log the request attempt
	// logger.Info("Making ride arrival request",
	//	logger.String("url", url),
	//	logger.String("ride_id", req.RideID),
	//	logger.Float64("adjustment_factor", req.AdjustmentFactor),
	//)

	// Send request with enhanced client (includes retry and circuit breaker)
	resp, err := g.rideClient.enhancedClient.Do(ctx, httpReq)
	if err != nil {
		logger.Error("Ride arrival request failed",
			logger.String("ride_id", req.RideID),
			logger.ErrorField(err),
		)
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
		logger.Error("Ride arrival request returned error",
			logger.String("ride_id", req.RideID),
			logger.Int("status_code", resp.StatusCode),
			logger.String("response_body", string(respBody)),
		)
		return nil, fmt.Errorf("ride arrival request failed: (status: %d, body: %s)", resp.StatusCode, string(respBody))
	}

	// Parse the response
	var paymentRequest models.PaymentRequest
	if err := utils.ParseJSONResponse(respBody, &paymentRequest); err != nil {
		// Log the raw response for debugging
		logger.Error("Failed to parse ride arrival response",
			logger.String("ride_id", req.RideID),
			logger.String("raw_response", string(respBody)),
			logger.ErrorField(err),
		)
		return nil, fmt.Errorf("failed to parse ride arrival response: %w", err)
	}

	// logger.Info("Ride arrival request successful",
	//	logger.String("ride_id", req.RideID),
	//	logger.String("payment_id", paymentRequest.RideID),
	//	logger.Int("total_cost", paymentRequest.TotalCost),
	//)

	return &paymentRequest, nil
}

// ProcessPayment sends a payment processing request to the ride service with retry and circuit breaker
func (g *HTTPGateway) ProcessPayment(paymentReq *models.PaymentProccessRequest) (*models.Payment, error) {
	ctx := context.Background()
	endpoint := fmt.Sprintf("/rides/%s/payment", paymentReq.RideID)

	// Use API key client if available, otherwise fallback to regular client
	if g.rideClient.apiClient != nil {
		var payment models.Payment
		err := g.rideClient.apiClient.PostJSON(ctx, endpoint, paymentReq, &payment)
		if err != nil {
			return nil, fmt.Errorf("failed to process payment with API key: %w", err)
		}
		return &payment, nil
	}

	// Fallback to original implementation
	url := fmt.Sprintf("%s/rides/%s/payment", g.rideClient.baseURL, paymentReq.RideID)

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

	// Log the request attempt
	// logger.Info("Making payment processing request",
	//	logger.String("url", url),
	//	logger.String("ride_id", paymentReq.RideID),
	//	logger.String("payment_status", string(paymentReq.Status)),
	//)

	// Send request with enhanced client (includes retry and circuit breaker)
	resp, err := g.rideClient.enhancedClient.Do(ctx, httpReq)
	if err != nil {
		logger.Error("Payment processing request failed",
			logger.String("ride_id", paymentReq.RideID),
			logger.ErrorField(err),
		)
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
		logger.Error("Payment processing request returned error",
			logger.String("ride_id", paymentReq.RideID),
			logger.Int("status_code", resp.StatusCode),
			logger.String("response_body", string(respBody)),
		)
		return nil, fmt.Errorf("payment request failed: (status: %d, body: %s)", resp.StatusCode, string(respBody))
	}

	// Parse the response
	var payment models.Payment
	if err := utils.ParseJSONResponse(respBody, &payment); err != nil {
		// Log the raw response for debugging
		logger.Error("Failed to parse payment response",
			logger.String("ride_id", paymentReq.RideID),
			logger.String("raw_response", string(respBody)),
			logger.ErrorField(err),
		)
		return nil, fmt.Errorf("failed to parse payment response: %w", err)
	}

	// logger.Info("Payment processing request successful",
	//	logger.String("ride_id", paymentReq.RideID),
	//	logger.String("payment_id", payment.PaymentID.String()),
	//	logger.Int("total_cost", payment.AdjustedCost),
	//)

	return &payment, nil
}
