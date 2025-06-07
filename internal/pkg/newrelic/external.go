package newrelic

import (
	"context"
	"net/http"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// StartExternalSegment creates an external service segment for HTTP requests
// This should be used to instrument outgoing HTTP calls to other services
func StartExternalSegment(ctx context.Context, request *http.Request) *newrelic.ExternalSegment {
	txn := FromContext(ctx)
	if txn == nil {
		return nil
	}

	return newrelic.StartExternalSegment(txn, request)
}

// InstrumentHTTPRequest wraps an HTTP request with New Relic external segment instrumentation
//
//	Usage: resp, err := InstrumentHTTPRequest(ctx, req, func() (*http.Response, error) {
//	  return client.Do(req)
//	})
func InstrumentHTTPRequest(ctx context.Context, req *http.Request, doFunc func() (*http.Response, error)) (*http.Response, error) {
	segment := StartExternalSegment(ctx, req)
	if segment != nil {
		defer segment.End()
	}

	resp, err := doFunc()

	// Add response details to segment if available
	if segment != nil && resp != nil {
		segment.Response = resp
	}

	return resp, err
}

// WithExternalSegment executes an HTTP operation within a New Relic external segment
// This is a more generic function for instrumenting any external service call
func WithExternalSegment(ctx context.Context, serviceName, operation, url string, fn func() error) error {
	txn := FromContext(ctx)
	if txn == nil {
		return fn()
	}

	segment := &newrelic.ExternalSegment{
		StartTime: txn.StartSegmentNow(),
		URL:       url,
		Procedure: operation,
		Library:   serviceName,
	}
	defer segment.End()

	err := fn()
	if err != nil {
		txn.NoticeError(err)
	}

	return err
}

// InstrumentServiceCall creates an external segment for service-to-service calls
// This is useful for gateway layer calls to other microservices
func InstrumentServiceCall(ctx context.Context, serviceName, endpoint string, fn func() error) error {
	return WithExternalSegment(ctx, serviceName, "HTTP", endpoint, fn)
}
