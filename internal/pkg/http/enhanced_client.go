package http

import (
	"context"
	"net/http"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/circuitbreaker"
	"github.com/piresc/nebengjek/internal/pkg/logger"
	nrpkg "github.com/piresc/nebengjek/internal/pkg/newrelic"
	"github.com/piresc/nebengjek/internal/pkg/retry"
)

// EnhancedClient wraps http.Client with retry and circuit breaker functionality
type EnhancedClient struct {
	client         *http.Client
	retrier        *retry.Retrier
	circuitManager *circuitbreaker.Manager
	logger         *logger.ZapLogger
	defaultTimeout time.Duration
}

// NewEnhancedClient creates a new enhanced HTTP client
func NewEnhancedClient(log *logger.ZapLogger, timeout time.Duration) *EnhancedClient {
	return &EnhancedClient{
		client: &http.Client{
			Timeout: timeout,
		},
		retrier:        retry.NewWithDefaults(log),
		circuitManager: circuitbreaker.NewManager(log),
		logger:         log,
		defaultTimeout: timeout,
	}
}

// Do executes an HTTP request with retry and circuit breaker protection
func (c *EnhancedClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Use circuit breaker based on the host
	serviceName := req.URL.Host
	if serviceName == "" {
		serviceName = "unknown"
	}

	var resp *http.Response
	var err error

	// Execute with circuit breaker protection
	err = c.circuitManager.Execute(ctx, serviceName, func(ctx context.Context) error {
		// Execute with retry logic within New Relic external segment
		return c.retrier.Execute(ctx, func(ctx context.Context) error {
			// Instrument the actual HTTP call with New Relic
			resp, err = nrpkg.InstrumentHTTPRequest(ctx, req, func() (*http.Response, error) {
				return c.client.Do(req.WithContext(ctx))
			})
			if err != nil {
				return err
			}

			// Consider 5xx status codes as failures for retry
			if resp.StatusCode >= 500 {
				resp.Body.Close()
				return &HTTPError{
					StatusCode: resp.StatusCode,
					Message:    "Server error",
				}
			}

			return nil
		})
	})

	return resp, err
}

// Get performs a GET request with enhanced features
func (c *EnhancedClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req)
}

// Post performs a POST request with enhanced features
func (c *EnhancedClient) Post(ctx context.Context, url, contentType string, body interface{}) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	return c.Do(ctx, req)
}

// GetCircuitBreakerStats returns circuit breaker statistics
func (c *EnhancedClient) GetCircuitBreakerStats() map[string]circuitbreaker.CircuitBreakerStats {
	return c.circuitManager.GetStats()
}

// HTTPError represents an HTTP error
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return e.Message
}
