package newrelic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockHTTPClient is a mock implementation of HTTP client
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

func TestStartExternalSegment_NilContext(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com", nil)
	segment := StartExternalSegment(nil, req)
	assert.Nil(t, segment)
}

func TestStartExternalSegment_NilRequest(t *testing.T) {
	ctx := context.Background()
	segment := StartExternalSegment(ctx, nil)
	assert.Nil(t, segment)
}

func TestStartExternalSegment_NoTransaction(t *testing.T) {
	ctx := context.Background()
	req := httptest.NewRequest("GET", "http://example.com", nil)
	segment := StartExternalSegment(ctx, req)
	assert.Nil(t, segment)
}

func TestStartExternalSegment_ValidInputs(t *testing.T) {
	// Create a context with a New Relic transaction
	txn := &newrelic.Transaction{}
	ctx := newrelic.NewContext(context.Background(), txn)
	
	req := httptest.NewRequest("GET", "http://example.com", nil)
	segment := StartExternalSegment(ctx, req)
	// Note: This will still return nil because the transaction is a mock
	// but the function should not panic
	assert.NotNil(t, segment)
}

func TestInstrumentHTTPRequest_NilContext(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com", nil)
	
	resp, err := InstrumentHTTPRequest(nil, req, func() (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK}, nil
	})
	
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInstrumentHTTPRequest_NilRequest(t *testing.T) {
	ctx := context.Background()
	
	resp, err := InstrumentHTTPRequest(ctx, nil, func() (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK}, nil
	})
	
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInstrumentHTTPRequest_NilDoFunc(t *testing.T) {
	ctx := context.Background()
	req := httptest.NewRequest("GET", "http://example.com", nil)
	
	// This should panic, but we're testing it doesn't cause nil pointer exception
	assert.Panics(t, func() {
		InstrumentHTTPRequest(ctx, req, nil)
	})
}

func TestInstrumentHTTPRequest_DoFuncError(t *testing.T) {
	ctx := context.Background()
	req := httptest.NewRequest("GET", "http://example.com", nil)
	
	resp, err := InstrumentHTTPRequest(ctx, req, func() (*http.Response, error) {
		return nil, assert.AnError
	})
	
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestInstrumentHTTPRequest_DoFuncSuccess(t *testing.T) {
	ctx := context.Background()
	req := httptest.NewRequest("GET", "http://example.com", nil)
	
	resp, err := InstrumentHTTPRequest(ctx, req, func() (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK}, nil
	})
	
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInstrumentHTTPRequest_WithNewRelicTransaction(t *testing.T) {
	// Create a context with a New Relic transaction
	txn := &newrelic.Transaction{}
	ctx := newrelic.NewContext(context.Background(), txn)
	
	req := httptest.NewRequest("GET", "http://example.com", nil)
	
	resp, err := InstrumentHTTPRequest(ctx, req, func() (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusCreated}, nil
	})
	
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestWithExternalSegment_NilContext(t *testing.T) {
	err := WithExternalSegment(nil, "test-service", "test-operation", "http://example.com", func() error {
		return nil
	})
	assert.NoError(t, err)
}

func TestWithExternalSegment_EmptyServiceName(t *testing.T) {
	ctx := context.Background()
	
	err := WithExternalSegment(ctx, "", "test-operation", "http://example.com", func() error {
		return nil
	})
	assert.NoError(t, err)
}

func TestWithExternalSegment_EmptyOperation(t *testing.T) {
	ctx := context.Background()
	
	err := WithExternalSegment(ctx, "test-service", "", "http://example.com", func() error {
		return nil
	})
	assert.NoError(t, err)
}

func TestWithExternalSegment_EmptyURL(t *testing.T) {
	ctx := context.Background()
	
	err := WithExternalSegment(ctx, "test-service", "test-operation", "", func() error {
		return nil
	})
	assert.NoError(t, err)
}

func TestWithExternalSegment_FunctionError(t *testing.T) {
	ctx := context.Background()
	
	err := WithExternalSegment(ctx, "test-service", "test-operation", "http://example.com", func() error {
		return assert.AnError
	})
	assert.Error(t, err)
}

func TestWithExternalSegment_FunctionSuccess(t *testing.T) {
	ctx := context.Background()
	
	err := WithExternalSegment(ctx, "test-service", "test-operation", "http://example.com", func() error {
		return nil
	})
	assert.NoError(t, err)
}

func TestWithExternalSegment_WithNewRelicTransaction(t *testing.T) {
	// Create a context with a New Relic transaction
	txn := &newrelic.Transaction{}
	ctx := newrelic.NewContext(context.Background(), txn)
	
	err := WithExternalSegment(ctx, "test-service", "test-operation", "http://example.com", func() error {
		return nil
	})
	assert.NoError(t, err)
}

func TestInstrumentServiceCall_NilContext(t *testing.T) {
	err := InstrumentServiceCall(nil, "test-service", "/test-endpoint", func() error {
		return nil
	})
	assert.NoError(t, err)
}

func TestInstrumentServiceCall_EmptyServiceName(t *testing.T) {
	ctx := context.Background()
	
	err := InstrumentServiceCall(ctx, "", "/test-endpoint", func() error {
		return nil
	})
	assert.NoError(t, err)
}

func TestInstrumentServiceCall_EmptyEndpoint(t *testing.T) {
	ctx := context.Background()
	
	err := InstrumentServiceCall(ctx, "test-service", "", func() error {
		return nil
	})
	assert.NoError(t, err)
}

func TestInstrumentServiceCall_FunctionError(t *testing.T) {
	ctx := context.Background()
	
	err := InstrumentServiceCall(ctx, "test-service", "/test-endpoint", func() error {
		return assert.AnError
	})
	assert.Error(t, err)
}

func TestInstrumentServiceCall_FunctionSuccess(t *testing.T) {
	ctx := context.Background()
	
	err := InstrumentServiceCall(ctx, "test-service", "/test-endpoint", func() error {
		return nil
	})
	assert.NoError(t, err)
}

func TestInstrumentServiceCall_WithNewRelicTransaction(t *testing.T) {
	// Create a context with a New Relic transaction
	txn := &newrelic.Transaction{}
	ctx := newrelic.NewContext(context.Background(), txn)
	
	err := InstrumentServiceCall(ctx, "test-service", "/test-endpoint", func() error {
		return nil
	})
	assert.NoError(t, err)
}

func TestExternalFunctionEdgeCases(t *testing.T) {
	t.Run("Empty parameters", func(t *testing.T) {
		ctx := context.Background()
		err := WithExternalSegment(ctx, "", "", "", func() error { return nil })
		assert.NoError(t, err)
	})
	
	t.Run("Special characters in URL", func(t *testing.T) {
		ctx := context.Background()
		err := WithExternalSegment(ctx, "test-service", "GET", "http://example.com/path?param=value&other=test", func() error {
			return nil
		})
		assert.NoError(t, err)
	})
	
	t.Run("Very long service name", func(t *testing.T) {
		ctx := context.Background()
		longName := "very-long-service-name-that-exceeds-normal-lengths-for-testing-purposes"
		err := WithExternalSegment(ctx, longName, "test-operation", "http://example.com", func() error {
			return nil
		})
		assert.NoError(t, err)
	})
	
	t.Run("Nil function", func(t *testing.T) {
		ctx := context.Background()
		assert.Panics(t, func() {
			WithExternalSegment(ctx, "test-service", "test-operation", "http://example.com", nil)
		})
	})
}

func TestRealHTTPRequestInstrumentation(t *testing.T) {
	// Create a test server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer testServer.Close()

	ctx := context.Background()
	
	// Test with a real HTTP request
	req, err := http.NewRequest("GET", testServer.URL, nil)
	assert.NoError(t, err)
	
	resp, err := InstrumentHTTPRequest(ctx, req, func() (*http.Response, error) {
		client := &http.Client{}
		return client.Do(req)
	})
	
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// Clean up
	resp.Body.Close()
}

func TestExternalSegmentWithDifferentOperations(t *testing.T) {
	ctx := context.Background()
	
	operations := []string{
		"GET",
		"POST",
		"PUT",
		"DELETE",
		"PATCH",
		"HEAD",
		"OPTIONS",
		"CONNECT",
		"TRACE",
	}
	
	for _, op := range operations {
		t.Run(op, func(t *testing.T) {
			err := WithExternalSegment(ctx, "test-service", op, "http://example.com", func() error {
				return nil
			})
			assert.NoError(t, err)
		})
	}
}

func TestServiceCallWithDifferentEndpoints(t *testing.T) {
	ctx := context.Background()
	
	endpoints := []string{
		"/api/users",
		"/api/rides",
		"/api/matches",
		"/health",
		"/metrics",
		"/internal/debug",
		"/v1/websocket",
		"/auth/login",
		"/payment/process",
	}
	
	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			err := InstrumentServiceCall(ctx, "test-service", endpoint, func() error {
				return nil
			})
			assert.NoError(t, err)
		})
	}
}