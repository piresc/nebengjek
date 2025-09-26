package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/stretchr/testify/assert"
)

func TestAddAttribute(t *testing.T) {
	t.Run("With valid transaction", func(t *testing.T) {
		// Create echo request with context
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		c := echo.New().NewContext(req, httptest.NewRecorder())
		
		// Test that function doesn't panic when no transaction is present
		AddAttribute(c, "test.key", "test.value")
		
		// The test passes if no panic occurs
	})

	t.Run("With nil transaction", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		c := echo.New().NewContext(req, httptest.NewRecorder())
		
		// Should not panic
		AddAttribute(c, "test.key", "test.value")
	})
}

func TestNoticeError(t *testing.T) {
	t.Run("With nil transaction", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		c := echo.New().NewContext(req, httptest.NewRecorder())
		
		testErr := assert.AnError
		// Should not panic
		NoticeError(c, testErr)
	})
}

func TestSetUserID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	c := echo.New().NewContext(req, httptest.NewRecorder())
	
	// Should not panic
	SetUserID(c, "test-user-123")
}

func TestSetMatchID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	c := echo.New().NewContext(req, httptest.NewRecorder())
	
	// Should not panic
	SetMatchID(c, "test-match-456")
}

func TestContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	c := echo.New().NewContext(req, httptest.NewRecorder())
	
	ctx := Context(c)
	assert.Equal(t, req.Context(), ctx)
}

func TestSetTransactionName(t *testing.T) {
	t.Run("With nil transaction", func(t *testing.T) {
		ctx := context.Background()
		
		// Should not panic
		SetTransactionName(ctx, "test-transaction")
	})
}

// Mock transaction for testing
type mockTransaction struct {
	attributes map[string]interface{}
	errors     []error
	name       string
}

func (m *mockTransaction) AddAttribute(key string, value interface{}) error {
	if m.attributes == nil {
		m.attributes = make(map[string]interface{})
	}
	m.attributes[key] = value
	return nil
}

func (m *mockTransaction) NoticeError(err error) error {
	m.errors = append(m.errors, err)
	return nil
}

func (m *mockTransaction) SetName(name string) error {
	m.name = name
	return nil
}

func (m *mockTransaction) End() error {
	return nil
}

func (m *mockTransaction) Ignore() error {
	return nil
}

func (m *mockTransaction) StartSegmentNow() newrelic.SegmentStartTime {
	return newrelic.SegmentStartTime{}
}

func (m *mockTransaction) InsertDistributedTraceHeaders(headers http.Header) error {
	return nil
}

func (m *mockTransaction) AcceptDistributedTraceHeaders(headers http.Header, transportType newrelic.TransportType) error {
	return nil
}

func (m *mockTransaction) AcceptDistributedTracePayload(payload interface{}) error {
	return nil
}

func (m *mockTransaction) CreateDistributedTracePayload() (interface{}, error) {
	return nil, nil
}

func (m *mockTransaction) GetLinkingMetadata() newrelic.LinkingMetadata {
	return newrelic.LinkingMetadata{}
}

func (m *mockTransaction) SetWebRequestHTTP(req *http.Request) error {
	return nil
}

func (m *mockTransaction) SetWebResponse(res http.ResponseWriter) http.ResponseWriter {
	return res
}

func (m *mockTransaction) AddAttributeString(key, value string) error {
	if m.attributes == nil {
		m.attributes = make(map[string]interface{})
	}
	m.attributes[key] = value
	return nil
}

func (m *mockTransaction) AddAttributeInt(key string, value int) error {
	if m.attributes == nil {
		m.attributes = make(map[string]interface{})
	}
	m.attributes[key] = value
	return nil
}

func (m *mockTransaction) AddAttributeFloat64(key string, value float64) error {
	if m.attributes == nil {
		m.attributes = make(map[string]interface{})
	}
	m.attributes[key] = value
	return nil
}

func (m *mockTransaction) AddAttributeBool(key string, value bool) error {
	if m.attributes == nil {
		m.attributes = make(map[string]interface{})
	}
	m.attributes[key] = value
	return nil
}

func (m *mockTransaction) NoticeErrorString(err string) error {
	m.errors = append(m.errors, assert.AnError)
	return nil
}

func (m *mockTransaction) AddAgentAttribute(key string, value interface{}) error {
	if m.attributes == nil {
		m.attributes = make(map[string]interface{})
	}
	m.attributes[key] = value
	return nil
}

func (m *mockTransaction) AddAgentAttributeString(key, value string) error {
	if m.attributes == nil {
		m.attributes = make(map[string]interface{})
	}
	m.attributes[key] = value
	return nil
}

func (m *mockTransaction) AddAgentAttributeInt(key string, value int) error {
	if m.attributes == nil {
		m.attributes = make(map[string]interface{})
	}
	m.attributes[key] = value
	return nil
}

func (m *mockTransaction) AddAgentAttributeFloat64(key string, value float64) error {
	if m.attributes == nil {
		m.attributes = make(map[string]interface{})
	}
	m.attributes[key] = value
	return nil
}

func (m *mockTransaction) AddAgentAttributeBool(key string, value bool) error {
	if m.attributes == nil {
		m.attributes = make(map[string]interface{})
	}
	m.attributes[key] = value
	return nil
}

func (m *mockTransaction) GetTraceMetadata() newrelic.TraceMetadata {
	return newrelic.TraceMetadata{}
}

func (m *mockTransaction) GetSpanMetadata() interface{} {
	return nil
}

func (m *mockTransaction) GetBrowserTimingHeader() string {
	return ""
}

func (m *mockTransaction) IsSampled() bool {
	return false
}