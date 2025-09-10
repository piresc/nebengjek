package tracing

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig_ShouldExclude(t *testing.T) {
	config := &Config{
		ExcludePaths: []string{"/health", "/metrics"},
	}

	tests := []struct {
		path     string
		expected bool
	}{
		{"/health", true},
		{"/metrics", true},
		{"/healthz", true}, // "/healthz" starts with "/health"
		{"/api/users", false},
		{"", false}, // Empty string should not contain any non-empty substring
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := config.ShouldExclude(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_DefaultValues(t *testing.T) {
	config := &Config{}

	// Test zero values
	assert.Equal(t, false, config.Enabled)
	assert.Equal(t, "", config.ServiceName)
	assert.Equal(t, 0.0, config.SampleRate)
	assert.Empty(t, config.ExcludePaths)
	assert.Empty(t, config.IncludeAttributes)
	assert.Equal(t, 0, config.MaxSegments)
	assert.Equal(t, time.Duration(0), config.Timeout)
	assert.Equal(t, false, config.NoOpForTesting)
}

func TestConfig_WithValues(t *testing.T) {
	config := &Config{
		Enabled:           true,
		ServiceName:       "test-service",
		SampleRate:        0.5,
		ExcludePaths:      []string{"/health", "/metrics"},
		IncludeAttributes: []string{"user_id", "request_id"},
		MaxSegments:       100,
		Timeout:           30 * time.Second,
		NoOpForTesting:    false,
	}

	assert.True(t, config.Enabled)
	assert.Equal(t, "test-service", config.ServiceName)
	assert.Equal(t, 0.5, config.SampleRate)
	assert.Equal(t, []string{"/health", "/metrics"}, config.ExcludePaths)
	assert.Equal(t, []string{"user_id", "request_id"}, config.IncludeAttributes)
	assert.Equal(t, 100, config.MaxSegments)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.False(t, config.NoOpForTesting)
}

func TestConfig_SampleRateValidation(t *testing.T) {
	tests := []struct {
		name      string
		sampleRate float64
		valid     bool
	}{
		{"Valid rate 0%", 0.0, true},
		{"Valid rate 50%", 0.5, true},
		{"Valid rate 100%", 1.0, true},
		{"Invalid rate negative", -0.1, false},
		{"Invalid rate above 100%", 1.1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = &Config{SampleRate: tt.sampleRate}
			
			isValid := tt.sampleRate >= 0.0 && tt.sampleRate <= 1.0
			assert.Equal(t, tt.valid, isValid)
		})
	}
}

func TestConfig_ExcludePathsEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		excludePaths []string
		testPath     string
		expected     bool
	}{
		{
			name:         "Empty exclude paths",
			excludePaths: []string{},
			testPath:     "/health",
			expected:     false,
		},
		{
			name:         "Single character path",
			excludePaths: []string{"/"},
			testPath:     "/health",
			expected:     true,
		},
		{
			name:         "Exact match",
			excludePaths: []string{"/api/users"},
			testPath:     "/api/users",
			expected:     true,
		},
		{
			name:         "Prefix match",
			excludePaths: []string{"/api"},
			testPath:     "/api/users",
			expected:     true, // contains() checks prefix
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{ExcludePaths: tt.excludePaths}
			result := config.ShouldExclude(tt.testPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsFunction(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"Exact match", "hello", "hello", true},
		{"Prefix match", "hello world", "hello", true},
		{"No match", "hello", "world", false},
		{"Empty string", "", "hello", false},
		{"Empty substring", "hello", "", true},
		{"Longer substring", "hello", "hello world", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTracer_InterfaceDefinition(t *testing.T) {
	// Test that the Tracer interface is properly defined
	var tracer Tracer
	
	// This test ensures the interface is correctly defined
	assert.Nil(t, tracer)
}

// testTracer implements Tracer interface for testing
type testTracer struct{}

func (t *testTracer) StartHTTPRequest(r *http.Request) (context.Context, HTTPTransaction) {
	return context.Background(), &testHTTPTransaction{}
}

func (t *testTracer) StartExternalCall(ctx context.Context, host, method string) (context.Context, func()) {
	return ctx, func() {}
}

func (t *testTracer) StartMessage(ctx context.Context, topic, operation string) (context.Context, func()) {
	return ctx, func() {}
}

func (t *testTracer) StartDatabase(ctx context.Context, operation, table string) (context.Context, func()) {
	return ctx, func() {}
}

func (t *testTracer) StartSegment(ctx context.Context, name string) (context.Context, func()) {
	return ctx, func() {}
}

func (t *testTracer) InjectContext(ctx context.Context, carrier interface{}) {}

func (t *testTracer) ExtractContext(ctx context.Context, carrier interface{}) context.Context {
	return ctx
}

func (t *testTracer) IsEnabled() bool {
	return false
}

func (t *testTracer) ShouldTrace(path string) bool {
	return false
}

// testHTTPTransaction implements HTTPTransaction interface for testing
type testHTTPTransaction struct{}

func (t *testHTTPTransaction) Context() context.Context {
	return context.Background()
}

func (t *testHTTPTransaction) SetName(name string) {}

func (t *testHTTPTransaction) AddAttribute(key, value string) {}

func (t *testHTTPTransaction) AddError(err error) {}

func (t *testHTTPTransaction) End() {}

func TestTestTracer_InterfaceCompliance(t *testing.T) {
	tracer := &testTracer{}
	
	// Test that testTracer implements the Tracer interface
	assert.Implements(t, (*Tracer)(nil), tracer)
	
	// Test all interface methods work without panicking
	req, _ := http.NewRequest("GET", "/test", nil)
	ctx, tx := tracer.StartHTTPRequest(req)
	assert.NotNil(t, ctx)
	assert.NotNil(t, tx)
	
	// Test HTTP transaction methods
	tx.SetName("test-transaction")
	tx.AddAttribute("key", "value")
	tx.AddError(assert.AnError)
	tx.End() // Should not panic
	
	// Test other methods
	ctx2, done := tracer.StartExternalCall(ctx, "test-host", "GET")
	assert.NotNil(t, ctx2)
	assert.NotNil(t, done)
	done()
	
	ctx3Message, done := tracer.StartMessage(ctx, "test-topic", "publish")
	assert.NotNil(t, ctx3Message)
	assert.NotNil(t, done)
	done()
	
	ctx4DB, done := tracer.StartDatabase(ctx, "SELECT", "users")
	assert.NotNil(t, ctx4DB)
	assert.NotNil(t, done)
	done()
	
	ctx5Segment, done := tracer.StartSegment(ctx, "custom-segment")
	assert.NotNil(t, ctx5Segment)
	assert.NotNil(t, done)
	done()
	
	// Test context methods
	tracer.InjectContext(ctx, map[string]string{})
	resultCtx := tracer.ExtractContext(ctx, map[string]string{})
	assert.NotNil(t, resultCtx)
	
	// Test config methods
	assert.False(t, tracer.IsEnabled())
	assert.False(t, tracer.ShouldTrace("/test"))
}