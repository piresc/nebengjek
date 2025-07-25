package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/models"
)

// HTTPClient handles HTTP requests to microservices with API key authentication
type HTTPClient struct {
	client *http.Client
	config *models.Config
}

// NewHTTPClient creates a new HTTP client with API key support
func NewHTTPClient(config *models.Config) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: config,
	}
}

// ServiceEndpoints maps service names to their base URLs
type ServiceEndpoints struct {
	UsersService    string
	MatchService    string
	RidesService    string
	LocationService string
}

// NewServiceEndpoints creates service endpoints configuration
func NewServiceEndpoints() *ServiceEndpoints {
	return &ServiceEndpoints{
		UsersService:    "http://users:9990",
		MatchService:    "http://match:9993",
		RidesService:    "http://rides:9992",
		LocationService: "http://location:9994",
	}
}

// Request represents an HTTP request to a microservice
type Request struct {
	Method      string
	Service     string
	Path        string
	Body        interface{}
	Headers     map[string]string
	QueryParams map[string]string
}

// Response represents an HTTP response from a microservice
type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// Do executes an HTTP request to a microservice with API key authentication
func (c *HTTPClient) Do(ctx context.Context, req *Request) (*Response, error) {
	// Get service endpoint
	endpoints := NewServiceEndpoints()
	baseURL, err := c.getServiceURL(req.Service, endpoints)
	if err != nil {
		return nil, fmt.Errorf("failed to get service URL: %w", err)
	}

	// Build full URL
	fullURL := baseURL + req.Path
	if len(req.QueryParams) > 0 {
		fullURL += "?" + c.buildQueryString(req.QueryParams)
	}

	// Prepare request body
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add API key authentication
	apiKey, err := c.getAPIKeyForService(req.Service)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	httpReq.Header.Set("X-API-Key", apiKey)

	// Add content type for JSON requests
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Add custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Execute request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Body:       respBody,
		Headers:    resp.Header,
	}, nil
}

// getServiceURL returns the base URL for a service
func (c *HTTPClient) getServiceURL(service string, endpoints *ServiceEndpoints) (string, error) {
	switch service {
	case "users-service":
		return endpoints.UsersService, nil
	case "match-service":
		return endpoints.MatchService, nil
	case "rides-service":
		return endpoints.RidesService, nil
	case "location-service":
		return endpoints.LocationService, nil
	default:
		return "", fmt.Errorf("unknown service: %s", service)
	}
}

// getAPIKeyForService returns the API key for a specific service
func (c *HTTPClient) getAPIKeyForService(service string) (string, error) {
	switch service {
	case "users-service":
		if c.config.APIKey.UserService == "" {
			return "", fmt.Errorf("API key for users service not configured")
		}
		return c.config.APIKey.UserService, nil
	case "match-service":
		if c.config.APIKey.MatchService == "" {
			return "", fmt.Errorf("API key for match service not configured")
		}
		return c.config.APIKey.MatchService, nil
	case "rides-service":
		if c.config.APIKey.RidesService == "" {
			return "", fmt.Errorf("API key for rides service not configured")
		}
		return c.config.APIKey.RidesService, nil
	case "location-service":
		if c.config.APIKey.LocationService == "" {
			return "", fmt.Errorf("API key for location service not configured")
		}
		return c.config.APIKey.LocationService, nil
	default:
		return "", fmt.Errorf("unknown service: %s", service)
	}
}

// buildQueryString builds a query string from parameters
func (c *HTTPClient) buildQueryString(params map[string]string) string {
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
