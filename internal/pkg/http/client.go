package http

import (
	"net/http"
	"time"
)

// Client is a generic HTTP client for communicating with services
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new HTTP client
func NewClient(serviceURL string, timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &Client{
		BaseURL: serviceURL,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}
