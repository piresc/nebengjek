package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	nethttp "net/http"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/logger"
	"github.com/piresc/nebengjek/internal/pkg/models"
)

const (
	// DefaultTimeout for HTTP requests
	DefaultTimeout = 30 * time.Second
	// APIKeyHeader is the header name for API key
	APIKeyHeader = "X-API-Key"
)

// APIKeyClient is an HTTP client with API key authentication
type APIKeyClient struct {
	client      *nethttp.Client
	apiKey      string
	baseURL     string
	serviceName string
}

// NewAPIKeyClient creates a new HTTP client with API key authentication
func NewAPIKeyClient(config *models.APIKeyConfig, serviceName, baseURL string) *APIKeyClient {
	var apiKey string

	// Get the appropriate API key based on service name
	switch serviceName {
	case "user-service":
		apiKey = config.UserService
	case "match-service":
		apiKey = config.MatchService
	case "rides-service":
		apiKey = config.RidesService
	case "location-service":
		apiKey = config.LocationService
	default:
		logger.Warn("Unknown service name for API key", logger.String("service", serviceName))
	}

	return &APIKeyClient{
		client: &nethttp.Client{
			Timeout: DefaultTimeout,
		},
		apiKey:      apiKey,
		baseURL:     baseURL,
		serviceName: serviceName,
	}
}

// SetTimeout sets the HTTP client timeout
func (c *APIKeyClient) SetTimeout(timeout time.Duration) {
	c.client.Timeout = timeout
}

// Get performs a GET request with API key authentication
func (c *APIKeyClient) Get(ctx context.Context, endpoint string) (*nethttp.Response, error) {
	return c.doRequest(ctx, nethttp.MethodGet, endpoint, nil)
}

// Post performs a POST request with API key authentication
func (c *APIKeyClient) Post(ctx context.Context, endpoint string, body interface{}) (*nethttp.Response, error) {
	return c.doRequest(ctx, nethttp.MethodPost, endpoint, body)
}

// Put performs a PUT request with API key authentication
func (c *APIKeyClient) Put(ctx context.Context, endpoint string, body interface{}) (*nethttp.Response, error) {
	return c.doRequest(ctx, nethttp.MethodPut, endpoint, body)
}

// Delete performs a DELETE request with API key authentication
func (c *APIKeyClient) Delete(ctx context.Context, endpoint string) (*nethttp.Response, error) {
	return c.doRequest(ctx, nethttp.MethodDelete, endpoint, nil)
}

// PostJSON performs a POST request with JSON body and API key authentication
func (c *APIKeyClient) PostJSON(ctx context.Context, endpoint string, body interface{}, result interface{}) error {
	resp, err := c.Post(ctx, endpoint, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}

	return nil
}

// GetJSON performs a GET request and decodes JSON response
func (c *APIKeyClient) GetJSON(ctx context.Context, endpoint string, result interface{}) error {
	resp, err := c.Get(ctx, endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}

	return nil
}

// doRequest performs the actual HTTP request with API key authentication
func (c *APIKeyClient) doRequest(ctx context.Context, method, endpoint string, body interface{}) (*nethttp.Response, error) {
	url := c.baseURL + endpoint

	var reqBody io.Reader
	contentType := "application/json"

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			logger.Error("Failed to marshal request body",
				logger.String("method", method),
				logger.String("url", url),
				logger.Err(err))
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := nethttp.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		logger.Error("Failed to create HTTP request",
			logger.String("method", method),
			logger.String("url", url),
			logger.Err(err))
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")

	// Add API key header if available
	if c.apiKey != "" {
		req.Header.Set(APIKeyHeader, c.apiKey)
	}

	// Add request ID if available in context
	if requestID := ctx.Value("request_id"); requestID != nil {
		req.Header.Set("X-Request-ID", fmt.Sprintf("%v", requestID))
	}

	logger.Debug("Making HTTP request",
		logger.String("method", method),
		logger.String("url", url),
		logger.String("service", c.serviceName),
		logger.Bool("has_api_key", c.apiKey != ""))

	resp, err := c.client.Do(req)
	if err != nil {
		logger.Error("HTTP request failed",
			logger.String("method", method),
			logger.String("url", url),
			logger.String("service", c.serviceName),
			logger.Err(err))
		return nil, fmt.Errorf("request failed: %w", err)
	}

	logger.Debug("HTTP request completed",
		logger.String("method", method),
		logger.String("url", url),
		logger.String("service", c.serviceName),
		logger.Int("status_code", resp.StatusCode))

	return resp, nil
}

// Close closes the HTTP client (for interface compatibility)
func (c *APIKeyClient) Close() error {
	// HTTP client doesn't need explicit closing
	return nil
}
