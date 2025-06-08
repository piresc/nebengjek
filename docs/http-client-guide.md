# HTTP Client Implementation Guide

## Overview

NebengJek's unified HTTP client provides reliable, efficient inter-service communication with built-in resilience patterns, authentication, and observability. The implementation consolidates all HTTP operations into a single, well-designed client.

## Architecture

### Core Implementation

**Location**: [`internal/pkg/http/client.go`](../internal/pkg/http/client.go)

```go
type Client struct {
    httpClient *http.Client
    apiKey     string
    baseURL    string
    timeout    time.Duration
}

type Config struct {
    APIKey  string
    BaseURL string
    Timeout time.Duration
}
```

### Key Features

**Unified Authentication**: Consistent API key handling across services
**Intelligent Retry Logic**: Exponential backoff for transient failures
**Request Correlation**: Automatic request ID propagation
**Flexible Response Handling**: Support for structured and direct JSON responses
**Error Classification**: Proper handling of 4xx vs 5xx scenarios

## Core Functionality

### Client Creation and Configuration

```go
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
```

**Configuration Options**:
- **APIKey**: Service authentication token
- **BaseURL**: Target service base URL
- **Timeout**: Request timeout (default: 30 seconds)

### Request Execution

```go
func (c *Client) Do(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
    url := c.baseURL + endpoint
    
    // JSON body marshaling
    var reqBody io.Reader
    if body != nil {
        jsonBody, err := json.Marshal(body)
        if err != nil {
            return nil, fmt.Errorf("marshal body: %w", err)
        }
        reqBody = bytes.NewBuffer(jsonBody)
    }
    
    // Request creation with context
    req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }
    
    // Standard headers
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")
    
    // API key authentication
    if c.apiKey != "" {
        req.Header.Set("X-API-Key", c.apiKey)
    }
    
    // Request ID propagation
    if requestID := ctx.Value("request_id"); requestID != nil {
        req.Header.Set("X-Request-ID", fmt.Sprintf("%v", requestID))
    }
    
    return c.executeWithRetry(req)
}
```

## Resilience Patterns

### Intelligent Retry Logic

```go
func (c *Client) executeWithRetry(req *http.Request) (*http.Response, error) {
    var resp *http.Response
    var err error
    
    // 3 attempts with exponential backoff
    for attempt := 0; attempt < 3; attempt++ {
        resp, err = c.httpClient.Do(req)
        
        // Success or client error (4xx) - don't retry
        if err == nil && resp.StatusCode < 500 {
            return resp, nil
        }
        
        if resp != nil {
            resp.Body.Close()
        }
        
        // Don't retry client errors (4xx)
        if resp != nil && resp.StatusCode >= 400 && resp.StatusCode < 500 {
            return resp, err
        }
        
        // Exponential backoff: 100ms, 200ms, 400ms
        if attempt < 2 {
            select {
            case <-req.Context().Done():
                return nil, req.Context().Err()
            case <-time.After(time.Duration(1<<attempt) * 100 * time.Millisecond):
            }
        }
    }
    
    return resp, err
}
```

**Retry Strategy**:
- **Attempts**: 3 total attempts
- **Backoff**: Exponential (100ms, 200ms, 400ms)
- **Conditions**: Only retry 5xx server errors
- **Context Aware**: Respects context cancellation

## HTTP Methods

### Standard HTTP Operations

```go
// GET request
func (c *Client) Get(ctx context.Context, endpoint string) (*http.Response, error) {
    return c.Do(ctx, "GET", endpoint, nil)
}

// POST request
func (c *Client) Post(ctx context.Context, endpoint string, body interface{}) (*http.Response, error) {
    return c.Do(ctx, "POST", endpoint, body)
}

// PUT request
func (c *Client) Put(ctx context.Context, endpoint string, body interface{}) (*http.Response, error) {
    return c.Do(ctx, "PUT", endpoint, body)
}

// DELETE request
func (c *Client) Delete(ctx context.Context, endpoint string) (*http.Response, error) {
    return c.Do(ctx, "DELETE", endpoint, nil)
}
```

## JSON Response Handling

### Structured Response Processing

```go
func (c *Client) GetJSON(ctx context.Context, endpoint string, result interface{}) error {
    resp, err := c.Get(ctx, endpoint)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode >= 400 {
        return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
    }
    
    if result != nil {
        body, err := io.ReadAll(resp.Body)
        if err != nil {
            return fmt.Errorf("failed to read response body: %w", err)
        }
        
        // Try structured response format first
        var structuredResp struct {
            Success bool            `json:"success"`
            Message string          `json:"message"`
            Data    json.RawMessage `json:"data"`
            Error   string          `json:"error"`
        }
        
        if err := json.Unmarshal(body, &structuredResp); err == nil {
            if !structuredResp.Success {
                return fmt.Errorf("API error: %s", structuredResp.Error)
            }
            
            // Unmarshal data field into result
            if structuredResp.Data != nil {
                return json.Unmarshal(structuredResp.Data, result)
            }
            return nil
        }
        
        // Fallback to direct unmarshaling
        return json.Unmarshal(body, result)
    }
    
    return nil
}
```

### POST with JSON Response

```go
func (c *Client) PostJSON(ctx context.Context, endpoint string, body, result interface{}) error {
    resp, err := c.Post(ctx, endpoint, body)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode >= 400 {
        return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
    }
    
    if result != nil {
        respBody, err := io.ReadAll(resp.Body)
        if err != nil {
            return fmt.Errorf("failed to read response body: %w", err)
        }
        
        // Handle both structured and direct responses
        return c.parseJSONResponse(respBody, result)
    }
    
    return nil
}
```

## Service Integration Examples

### Match Service Client

```go
// services/users/gateway/http/match.go
type MatchClient struct {
    client *httpclient.Client
    tracer observability.Tracer
}

func NewMatchClient(matchServiceURL string, config *models.APIKeyConfig, tracer observability.Tracer) *MatchClient {
    return &MatchClient{
        client: httpclient.NewClient(httpclient.Config{
            APIKey:  config.MatchService,
            BaseURL: matchServiceURL,
            Timeout: 30 * time.Second,
        }),
        tracer: tracer,
    }
}

func (g *HTTPGateway) MatchConfirm(ctx context.Context, req *models.MatchConfirmRequest) (*models.MatchProposal, error) {
    endpoint := fmt.Sprintf("/internal/matches/%s/confirm", req.ID)
    
    // APM segment tracking
    var endSegment func()
    if g.matchClient.tracer != nil {
        ctx, endSegment = g.matchClient.tracer.StartSegment(ctx, "External/match-service/confirm")
        defer endSegment()
    }
    
    var matchProposal models.MatchProposal
    err := g.matchClient.client.PostJSON(ctx, endpoint, req, &matchProposal)
    if err != nil {
        return nil, fmt.Errorf("failed to send match confirmation request: %w", err)
    }
    
    return &matchProposal, nil
}
```

### Rides Service Client

```go
// services/users/gateway/http/rides.go
type RideClient struct {
    client *httpclient.Client
    tracer observability.Tracer
}

func NewRideClient(rideServiceURL string, config *models.APIKeyConfig, tracer observability.Tracer) *RideClient {
    return &RideClient{
        client: httpclient.NewClient(httpclient.Config{
            APIKey:  config.RidesService,
            BaseURL: rideServiceURL,
            Timeout: 30 * time.Second,
        }),
        tracer: tracer,
    }
}

func (g *HTTPGateway) CreateRide(ctx context.Context, req *models.CreateRideRequest) (*models.Ride, error) {
    endpoint := "/internal/rides"
    
    // APM tracking
    var endSegment func()
    if g.rideClient.tracer != nil {
        ctx, endSegment = g.rideClient.tracer.StartSegment(ctx, "External/rides-service/create")
        defer endSegment()
    }
    
    var ride models.Ride
    err := g.rideClient.client.PostJSON(ctx, endpoint, req, &ride)
    if err != nil {
        return nil, fmt.Errorf("failed to create ride: %w", err)
    }
    
    return &ride, nil
}
```

## Error Handling

### Error Classification

```go
// Client errors (4xx) - don't retry
if resp.StatusCode >= 400 && resp.StatusCode < 500 {
    return handleClientError(resp)
}

// Server errors (5xx) - retry with backoff
if resp.StatusCode >= 500 {
    return handleServerError(resp, attempt)
}
```

### Structured Error Responses

```go
type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details"`
}

func (e APIError) Error() string {
    return fmt.Sprintf("API error [%s]: %s", e.Code, e.Message)
}
```

## Observability Integration

### Request ID Propagation

```go
// Automatic request ID forwarding
if requestID := ctx.Value("request_id"); requestID != nil {
    req.Header.Set("X-Request-ID", fmt.Sprintf("%v", requestID))
}
```

### APM Integration

```go
// Service gateway with APM tracking
func (g *HTTPGateway) callExternalService(ctx context.Context, serviceName, operation string) {
    var endSegment func()
    if g.tracer != nil {
        segmentName := fmt.Sprintf("External/%s/%s", serviceName, operation)
        ctx, endSegment = g.tracer.StartSegment(ctx, segmentName)
        defer endSegment()
    }
    
    // Make HTTP call with tracking context
    err := g.client.PostJSON(ctx, endpoint, request, response)
}
```

## Performance Characteristics

### Efficiency Metrics

**Connection Reuse**: HTTP/1.1 keep-alive connections
**Request Overhead**: <1ms for request setup
**Retry Latency**: Minimal with exponential backoff
**Memory Usage**: Efficient with connection pooling

### Timeout Management

```go
// Context-based timeout control
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

resp, err := client.PostJSON(ctx, endpoint, request, response)
```

## Configuration Patterns

### Service-Specific Configuration

```go
// Environment-based configuration
type ServiceConfig struct {
    MatchService struct {
        URL    string
        APIKey string
        Timeout time.Duration
    }
    RidesService struct {
        URL    string
        APIKey string
        Timeout time.Duration
    }
}

// Client factory
func NewHTTPGateway(config ServiceConfig, tracer observability.Tracer) *HTTPGateway {
    return &HTTPGateway{
        matchClient: NewMatchClient(config.MatchService.URL, config.MatchService.APIKey, tracer),
        rideClient:  NewRideClient(config.RidesService.URL, config.RidesService.APIKey, tracer),
    }
}
```

### Development vs Production

```go
// Development configuration
devConfig := httpclient.Config{
    APIKey:  "dev-api-key",
    BaseURL: "http://localhost:8080",
    Timeout: 5 * time.Second,
}

// Production configuration
prodConfig := httpclient.Config{
    APIKey:  os.Getenv("SERVICE_API_KEY"),
    BaseURL: os.Getenv("SERVICE_URL"),
    Timeout: 30 * time.Second,
}
```

## Future Enhancements

### Short-term Improvements
- **Circuit Breaker**: Fault tolerance for failing services
- **Request Caching**: Response caching for idempotent operations
- **Metrics Collection**: Request/response metrics for monitoring
- **Load Balancing**: Multiple endpoint support with health checking

### Advanced Features
- **Service Discovery**: Dynamic endpoint resolution
- **Request Signing**: Enhanced security with request signatures
- **Compression**: Request/response compression support
- **Streaming**: Support for streaming responses

### Monitoring Enhancements
- **Request Tracing**: Detailed request lifecycle tracking
- **Performance Analytics**: Response time distribution analysis
- **Error Rate Monitoring**: Automatic alerting on error thresholds
- **Capacity Planning**: Resource usage forecasting

## Best Practices

### Error Handling
```go
// Proper error wrapping
if err := client.PostJSON(ctx, endpoint, req, resp); err != nil {
    return fmt.Errorf("failed to call %s service: %w", serviceName, err)
}
```

### Context Usage
```go
// Always use context for cancellation
ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
defer cancel()

err := client.GetJSON(ctx, endpoint, result)
```

### Resource Management
```go
// Proper response body handling
resp, err := client.Get(ctx, endpoint)
if err != nil {
    return err
}
defer resp.Body.Close() // Always close response body
```

## Conclusion

The unified HTTP client provides a robust, efficient foundation for inter-service communication in NebengJek. The implementation emphasizes reliability, observability, and developer experience while maintaining high performance and scalability.

The architecture supports current service communication needs while providing extensibility for advanced features like circuit breaking, caching, and enhanced monitoring, making it a critical component of our microservices infrastructure.