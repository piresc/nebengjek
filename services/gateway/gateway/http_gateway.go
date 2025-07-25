package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/models"
	"github.com/piresc/nebengjek/services/gateway"
)

// HTTPGateway implements the GatewayGW interface for HTTP communication with microservices
type HTTPGateway struct {
	client    *http.Client
	config    *models.Config
	endpoints *ServiceEndpoints
}

// ServiceEndpoints maps service names to their base URLs
type ServiceEndpoints struct {
	UsersService    string
	MatchService    string
	RidesService    string
	LocationService string
}

// NewHTTPGateway creates a new HTTP gateway instance
func NewHTTPGateway(config *models.Config) gateway.GatewayGW {
	endpoints := &ServiceEndpoints{
		UsersService:    "http://localhost:9990",
		MatchService:    "http://localhost:9993",
		RidesService:    "http://localhost:9992",
		LocationService: "http://localhost:9994",
	}

	return &HTTPGateway{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		config:    config,
		endpoints: endpoints,
	}
}

// CallUsersService makes an HTTP call to the Users Service
func (g *HTTPGateway) CallUsersService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*models.ProxyResponse, error) {
	return g.makeHTTPCall(ctx, g.endpoints.UsersService, method, path, body, headers, queryParams, g.config.APIKey.UserService)
}

// CallMatchService makes an HTTP call to the Match Service
func (g *HTTPGateway) CallMatchService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*models.ProxyResponse, error) {
	return g.makeHTTPCall(ctx, g.endpoints.MatchService, method, path, body, headers, queryParams, g.config.APIKey.MatchService)
}

// CallRidesService makes an HTTP call to the Rides Service
func (g *HTTPGateway) CallRidesService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*models.ProxyResponse, error) {
	return g.makeHTTPCall(ctx, g.endpoints.RidesService, method, path, body, headers, queryParams, g.config.APIKey.RidesService)
}

// CallLocationService makes an HTTP call to the Location Service
func (g *HTTPGateway) CallLocationService(ctx context.Context, method, path string, body interface{}, headers map[string]string, queryParams map[string]string) (*models.ProxyResponse, error) {
	return g.makeHTTPCall(ctx, g.endpoints.LocationService, method, path, body, headers, queryParams, g.config.APIKey.LocationService)
}

// PublishNotification publishes a notification via NATS (placeholder implementation)
func (g *HTTPGateway) PublishNotification(ctx context.Context, notification *models.Notification) error {
	// TODO: Implement NATS notification publishing
	return nil
}

// PublishEvent publishes an event via NATS (placeholder implementation)
func (g *HTTPGateway) PublishEvent(ctx context.Context, subject string, data interface{}) error {
	// TODO: Implement NATS event publishing
	return nil
}

// makeHTTPCall is a helper method to make HTTP calls to microservices
func (g *HTTPGateway) makeHTTPCall(ctx context.Context, baseURL, method, path string, body interface{}, headers map[string]string, queryParams map[string]string, apiKey string) (*models.ProxyResponse, error) {
	// Build full URL
	fullURL := baseURL + "/internal" + path
	if len(queryParams) > 0 {
		fullURL += "?" + g.buildQueryString(queryParams)
	}

	// Prepare request body
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add API key authentication
	req.Header.Set("X-API-Key", apiKey)

	// Add content type for JSON requests
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &models.ProxyResponse{
		StatusCode: resp.StatusCode,
		Body:       respBody,
		Headers:    resp.Header,
	}, nil
}

// buildQueryString builds a query string from parameters
func (g *HTTPGateway) buildQueryString(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}

	var parts []string
	for key, value := range params {
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}

	result := ""
	for i, part := range parts {
		if i > 0 {
			result += "&"
		}
		result += part
	}
	return result
}
