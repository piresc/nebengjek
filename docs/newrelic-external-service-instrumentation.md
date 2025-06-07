# New Relic External Service Instrumentation

This document describes the external service instrumentation functions and their implementation across the nebengjek microservices architecture.

## Overview

The external service instrumentation provides comprehensive distributed tracing capabilities for HTTP-based service-to-service communication in the New Relic APM system. This enables monitoring of external dependencies, performance analysis, and distributed tracing across microservices.

## External Service Functions

### Core Functions (internal/pkg/newrelic/external.go)

#### `StartExternalSegment(ctx context.Context, request *http.Request) *newrelic.ExternalSegment`
- Creates and starts a new external segment for HTTP requests
- Automatically extracts service information from the request URL
- Returns an external segment that should be ended when the request completes

#### `InstrumentHTTPRequest(ctx context.Context, req *http.Request, doFunc func() (*http.Response, error)) (*http.Response, error)`
- Wraps HTTP requests with New Relic external segments
- Automatically handles segment lifecycle (start/end)
- Captures request/response information for tracing
- Used primarily in HTTP client implementations

#### `WithExternalSegment(ctx context.Context, serviceName, operation, url string, fn func() error) error`
- Generic wrapper for external operations with New Relic segments
- Allows custom service name, operation, and URL specification
- Suitable for any external operation that needs instrumentation

#### `InstrumentServiceCall(ctx context.Context, serviceName, endpoint string, fn func() error) error`
- Specialized wrapper for service-to-service calls
- Automatically constructs meaningful operation names
- Optimized for microservice communication patterns

## Implementation Coverage

### HTTP Clients

#### APIKeyClient (internal/pkg/http/client_with_apikey.go)
- **Method**: `doRequest()`
- **Instrumentation**: `nrpkg.InstrumentHTTPRequest()`
- **Coverage**: All HTTP methods (GET, POST, PUT, DELETE)

#### EnhancedClient (internal/pkg/http/enhanced_client.go)  
- **Method**: `Do()`
- **Instrumentation**: `nrpkg.InstrumentHTTPRequest()`
- **Coverage**: All HTTP requests through the enhanced client

### Gateway Services

#### Match Service Gateway (services/match/gateway/http.go)
All LocationClient methods are instrumented with `nrpkg.InstrumentServiceCall()`:
- `AddAvailableDriver()` - Adds driver to location service
- `RemoveAvailableDriver()` - Removes driver from location service  
- `AddAvailablePassenger()` - Adds passenger to location service
- `RemoveAvailablePassenger()` - Removes passenger from location service
- `FindNearbyDrivers()` - Queries nearby drivers from location service
- `GetDriverLocation()` - Retrieves driver location from location service
- `GetPassengerLocation()` - Retrieves passenger location from location service

#### Users Service - Rides Gateway (services/users/gateway/http/rides.go)
All RideClient methods are instrumented with `nrpkg.InstrumentServiceCall()`:
- `StartRide()` - Initiates ride via rides service
- `RideArrived()` - Updates ride status via rides service
- `ProcessPayment()` - Processes payment via rides service

#### Users Service - Match Gateway (services/users/gateway/http/match.go)
All MatchClient methods are instrumented with `nrpkg.InstrumentServiceCall()`:
- `MatchConfirm()` - Confirms match via match service

## Service Mapping

The instrumentation captures the following service interactions:

```
Users Service → Match Service    (match-service)
Users Service → Rides Service    (rides-service)  
Match Service → Location Service (location-service)
```

## Benefits

1. **Distributed Tracing**: Complete visibility into request flows across microservices
2. **Performance Monitoring**: Automatic capture of external service response times
3. **Error Tracking**: External service errors are captured and associated with transactions
4. **Service Dependencies**: Clear mapping of service interdependencies
5. **Throughput Analysis**: External service call frequency and patterns

## Usage Examples

### Manual External Segment
```go
func makeExternalCall(ctx context.Context) error {
    req, _ := http.NewRequest("GET", "http://external-service/api", nil)
    segment := nrpkg.StartExternalSegment(ctx, req)
    defer segment.End()
    
    // Make HTTP call
    resp, err := client.Do(req)
    return err
}
```

### Service Call Wrapper
```go
func callExternalService(ctx context.Context) error {
    return nrpkg.InstrumentServiceCall(ctx, "external-service", "http://external-service/api", func() error {
        // External service logic
        return makeServiceCall()
    })
}
```

## Configuration

External service instrumentation is automatically enabled when:
1. New Relic is properly configured via environment variables
2. Context contains a valid New Relic transaction
3. HTTP clients use the instrumented client implementations

No additional configuration is required for the instrumentation to function.
