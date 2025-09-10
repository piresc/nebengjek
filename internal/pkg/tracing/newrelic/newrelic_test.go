package newrelic

import (
	"context"
	"net/http"
	"testing"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/piresc/nebengjek/internal/pkg/tracing"
	"github.com/stretchr/testify/assert"
)

func TestNewTracer_WithNilApp(t *testing.T) {
	config := &tracing.Config{
		Enabled:     true,
		ServiceName: "test-service",
	}
	
	tracer := NewTracer(config, nil)
	
	assert.NotNil(t, tracer)
	assert.IsType(t, &NoOpTracer{}, tracer)
}

func TestNewTracer_WithDisabledConfig(t *testing.T) {
	config := &tracing.Config{
		Enabled:     false,
		ServiceName: "test-service",
	}
	
	// Create a mock app (it should be ignored)
	mockApp := &newrelic.Application{}
	
	tracer := NewTracer(config, mockApp)
	
	assert.NotNil(t, tracer)
	assert.IsType(t, &NoOpTracer{}, tracer)
}

func TestNewTracer_WithNoOpForTesting(t *testing.T) {
	config := &tracing.Config{
		NoOpForTesting: true,
		Enabled:        true,
		ServiceName:    "test-service",
	}
	
	// Create a mock app (it should be ignored)
	mockApp := &newrelic.Application{}
	
	tracer := NewTracer(config, mockApp)
	
	assert.NotNil(t, tracer)
	assert.IsType(t, &NoOpTracer{}, tracer)
}

func TestNewRelicTracer_IsEnabled(t *testing.T) {
	config := &tracing.Config{
		Enabled:     true,
		ServiceName: "test-service",
	}
	
	// Create a mock app
	mockApp := &newrelic.Application{}
	
	tracer := NewTracer(config, mockApp)
	
	// The NoOpTracer should return false since the app is nil
	assert.True(t, tracer.IsEnabled())
}

func TestNewRelicTracer_StartHTTP_WithNoOp(t *testing.T) {
	config := &tracing.Config{
		Enabled:     true,
		ServiceName: "test-service",
	}
	
	tracer := NewTracer(config, nil)
	
	req, _ := http.NewRequest("GET", "/test", nil)
	ctx, tx := tracer.StartHTTPRequest(req)
	defer tx.End()
	
	assert.NotNil(t, ctx)
	assert.NotNil(t, tx)
}

func TestNewRelicTracer_StartDatabase_WithNoOp(t *testing.T) {
	config := &tracing.Config{
		Enabled:     true,
		ServiceName: "test-service",
	}
	
	tracer := NewTracer(config, nil)
	
	ctx, done := tracer.StartDatabase(context.Background(), "SELECT", "postgres")
	defer done()
	
	assert.NotNil(t, ctx)
	assert.Equal(t, context.Background(), ctx)
}

func TestNewRelicTracer_StartMessage_WithNoOp(t *testing.T) {
	config := &tracing.Config{
		Enabled:     true,
		ServiceName: "test-service",
	}
	
	tracer := NewTracer(config, nil)
	
	ctx, done := tracer.StartMessage(context.Background(), "test.subject", "publish")
	defer done()
	
	assert.NotNil(t, ctx)
	assert.Equal(t, context.Background(), ctx)
}

func TestNewRelicTracer_AddAttribute_WithNoOp(t *testing.T) {
	config := &tracing.Config{
		Enabled:     true,
		ServiceName: "test-service",
	}
	
	tracer := NewTracer(config, nil)
	
	// Test with HTTP transaction
	req, _ := http.NewRequest("GET", "/test", nil)
	_, tx := tracer.StartHTTPRequest(req)
	defer tx.End()
	
	// Should not panic
	tx.AddAttribute("key", "value")
	tx.AddAttribute("number", "42")
	tx.AddAttribute("bool", "true")
	tx.AddAttribute("duration", "1s")
}

func TestNewRelicTracer_AddError_WithNoOp(t *testing.T) {
	config := &tracing.Config{
		Enabled:     true,
		ServiceName: "test-service",
	}
	
	tracer := NewTracer(config, nil)
	
	// Test with HTTP transaction
	req, _ := http.NewRequest("GET", "/test", nil)
	_, tx := tracer.StartHTTPRequest(req)
	defer tx.End()
	
	// Should not panic
	tx.AddError(assert.AnError)
}

func TestNewRelicTracer_StartSegment_WithNoOp(t *testing.T) {
	config := &tracing.Config{
		Enabled:     true,
		ServiceName: "test-service",
	}
	
	tracer := NewTracer(config, nil)
	
	ctx, done := tracer.StartSegment(context.Background(), "test-segment")
	defer done()
	
	assert.NotNil(t, ctx)
	assert.Equal(t, context.Background(), ctx)
}

func TestTracerConsistency(t *testing.T) {
	config := &tracing.Config{
		Enabled:     true,
		ServiceName: "test-service",
	}
	
	tracer := NewTracer(config, nil)
	
	// Test that all operations are consistent
	req, _ := http.NewRequest("GET", "/test", nil)
	ctx1, tx1 := tracer.StartHTTPRequest(req)
	defer tx1.End()
	
	ctx2, done2 := tracer.StartDatabase(ctx1, "SELECT", "postgres")
	defer done2()
	
	ctx3, done3 := tracer.StartMessage(ctx2, "test.subject", "publish")
	defer done3()
	
	assert.NotNil(t, ctx1)
	assert.NotNil(t, ctx2)
	assert.NotNil(t, ctx3)
}

func TestConcurrentOperations(t *testing.T) {
	config := &tracing.Config{
		Enabled:     true,
		ServiceName: "test-service",
	}
	
	tracer := NewTracer(config, nil)
	
	// Test concurrent operations
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			req, _ := http.NewRequest("GET", "/test", nil)
			ctx, tx := tracer.StartHTTPRequest(req)
			tx.AddAttribute("goroutine", string(rune('0' + id)))
			tx.End()
			
			assert.NotNil(t, ctx)
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestTracerWithDifferentConfigurations(t *testing.T) {
	tests := []struct {
		name     string
		config   *tracing.Config
		app      *newrelic.Application
		expectedType string
	}{
		{
			name: "Enabled with nil app",
			config: &tracing.Config{
				Enabled:     true,
				ServiceName: "test-service",
			},
			app:      nil,
			expectedType: "*newrelic.NoOpTracer",
		},
		{
			name: "Disabled with nil app",
			config: &tracing.Config{
				Enabled:     false,
				ServiceName: "test-service",
			},
			app:      nil,
			expectedType: "*newrelic.NoOpTracer",
		},
		{
			name: "NoOpForTesting",
			config: &tracing.Config{
				NoOpForTesting: true,
				Enabled:        true,
				ServiceName:    "test-service",
			},
			app:      &newrelic.Application{},
			expectedType: "*newrelic.NoOpTracer",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracer := NewTracer(tt.config, tt.app)
			assert.NotNil(t, tracer)
			assert.IsType(t, &NoOpTracer{}, tracer)
		})
	}
}