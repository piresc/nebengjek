package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client provides a simple HTTP client with API key support and basic retry
// This replaces both EnhancedClient and APIKeyClient with a simpler implementation
type Client struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
	timeout    time.Duration
}

// Config holds configuration for the HTTP client
type Config struct {
	APIKey  string
	BaseURL string
	Timeout time.Duration
}

// NewClient creates a new simplified HTTP client
func NewClient(config Config) *Client {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &Client{
		httpClient: &http.Client{Timeout: config.Timeout},
		apiKey:     config.APIKey,
		baseURL:    config.BaseURL,
		timeout:    config.Timeout,
	}
}

// Do executes an HTTP request with simple retry logic
func (c *Client) Do(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	url := c.baseURL + endpoint

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	// Add request ID if available in context
	if requestID := ctx.Value("request_id"); requestID != nil {
		req.Header.Set("X-Request-ID", fmt.Sprintf("%v", requestID))
	}

	// Simple retry logic (3 attempts with exponential backoff)
	var resp *http.Response
	for attempt := 0; attempt < 3; attempt++ {
		resp, err = c.httpClient.Do(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		if resp != nil {
			resp.Body.Close()
		}

		// Don't retry on client errors (4xx)
		if resp != nil && resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return resp, err
		}

		// Wait before retry (100ms, 200ms, 400ms)
		if attempt < 2 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(1<<attempt) * 100 * time.Millisecond):
			}
		}
	}

	return resp, err
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, endpoint string) (*http.Response, error) {
	return c.Do(ctx, "GET", endpoint, nil)
}

// Post performs a POST request
func (c *Client) Post(ctx context.Context, endpoint string, body interface{}) (*http.Response, error) {
	return c.Do(ctx, "POST", endpoint, body)
}

// Put performs a PUT request
func (c *Client) Put(ctx context.Context, endpoint string, body interface{}) (*http.Response, error) {
	return c.Do(ctx, "PUT", endpoint, body)
}

// Delete performs a DELETE request
func (c *Client) Delete(ctx context.Context, endpoint string) (*http.Response, error) {
	return c.Do(ctx, "DELETE", endpoint, nil)
}

// GetJSON performs a GET request and decodes JSON response
func (c *Client) GetJSON(ctx context.Context, endpoint string, result interface{}) error {
	resp, err := c.Get(ctx, endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	if result != nil {
		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		// Try to parse as structured response first
		var structuredResp struct {
			Success bool            `json:"success"`
			Message string          `json:"message"`
			Data    json.RawMessage `json:"data"`
			Error   string          `json:"error"`
		}

		if err := json.Unmarshal(body, &structuredResp); err == nil {
			// If it's a structured response, handle it
			if !structuredResp.Success {
				return fmt.Errorf("API error: %s", structuredResp.Error)
			}

			// Unmarshal the data field into the result
			if structuredResp.Data != nil {
				if err := json.Unmarshal(structuredResp.Data, result); err != nil {
					return fmt.Errorf("failed to decode JSON response: %w", err)
				}
				return nil
			}
			return nil
		}

		// If not structured response, try direct unmarshaling
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("failed to decode JSON response: %w", err)
		}
		return nil
	}

	return nil
}

// PostJSON performs a POST request with JSON body and decodes JSON response
func (c *Client) PostJSON(ctx context.Context, endpoint string, body, result interface{}) error {
	resp, err := c.Post(ctx, endpoint, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	if result != nil {
		// Read the response body
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		// Try to parse as structured response first
		var structuredResp struct {
			Success bool            `json:"success"`
			Message string          `json:"message"`
			Data    json.RawMessage `json:"data"`
			Error   string          `json:"error"`
		}

		if err := json.Unmarshal(respBody, &structuredResp); err == nil {
			// If it's a structured response, handle it
			if !structuredResp.Success {
				return fmt.Errorf("API error: %s", structuredResp.Error)
			}

			// Unmarshal the data field into the result
			if structuredResp.Data != nil {
				if err := json.Unmarshal(structuredResp.Data, result); err != nil {
					return fmt.Errorf("failed to decode JSON response: %w", err)
				}
				return nil
			}
			return nil
		}

		// If not structured response, try direct unmarshaling
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to decode JSON response: %w", err)
		}
		return nil
	}

	return nil
}

// PutJSON performs a PUT request with JSON body and decodes JSON response
func (c *Client) PutJSON(ctx context.Context, endpoint string, body, result interface{}) error {
	resp, err := c.Put(ctx, endpoint, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	if result != nil {
		// Read the response body
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		// Try to parse as structured response first
		var structuredResp struct {
			Success bool            `json:"success"`
			Message string          `json:"message"`
			Data    json.RawMessage `json:"data"`
			Error   string          `json:"error"`
		}

		if err := json.Unmarshal(respBody, &structuredResp); err == nil {
			// If it's a structured response, handle it
			if !structuredResp.Success {
				return fmt.Errorf("API error: %s", structuredResp.Error)
			}

			// Unmarshal the data field into the result
			if structuredResp.Data != nil {
				if err := json.Unmarshal(structuredResp.Data, result); err != nil {
					return fmt.Errorf("failed to decode JSON response: %w", err)
				}
				return nil
			}
			return nil
		}

		// If not structured response, try direct unmarshaling
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to decode JSON response: %w", err)
		}
		return nil
	}

	return nil
}

// Close closes the HTTP client (for interface compatibility)
func (c *Client) Close() error {
	// HTTP client doesn't need explicit closing
	return nil
}
