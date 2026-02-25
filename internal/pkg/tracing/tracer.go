package tracing

import (
	"context"
	"net/http"
	"time"
)

// Tracer is the unified interface for all tracing operations
type Tracer interface {
	// HTTP Request Tracing
	StartHTTPRequest(r *http.Request) (context.Context, HTTPTransaction)
	
	// External Service Calls
	StartExternalCall(ctx context.Context, host, method string) (context.Context, func())
	
	// Message Queue Operations
	StartMessage(ctx context.Context, topic, operation string) (context.Context, func())
	
	// Database Operations
	StartDatabase(ctx context.Context, operation, table string) (context.Context, func())
	
	// Custom Business Segments
	StartSegment(ctx context.Context, name string) (context.Context, func())
	
	// Context Injection/Extraction
	InjectContext(ctx context.Context, carrier interface{})
	ExtractContext(ctx context.Context, carrier interface{}) context.Context
	
	// Configuration
	IsEnabled() bool
	ShouldTrace(path string) bool
}

// HTTPTransaction represents a HTTP request transaction
type HTTPTransaction interface {
	Context() context.Context
	SetName(name string)
	AddAttribute(key, value string)
	AddError(err error)
	End()
}

// Config defines the tracing configuration
type Config struct {
	Enabled           bool              `yaml:"enabled"`
	ServiceName       string            `yaml:"service_name"`
	SampleRate        float64           `yaml:"sample_rate"`        // 0.0 to 1.0
	ExcludePaths      []string          `yaml:"exclude_paths"`
	IncludeAttributes []string          `yaml:"include_attributes"`
	MaxSegments       int               `yaml:"max_segments"`
	Timeout           time.Duration     `yaml:"timeout"`
	NoOpForTesting    bool              `yaml:"noop_for_testing"`
}

// ShouldExclude checks if a path should be excluded from tracing
func (c *Config) ShouldExclude(path string) bool {
	for _, excluded := range c.ExcludePaths {
		if hasPrefix(path, excluded) {
			return true
		}
	}
	return false
}

// contains checks if a string starts with a substring
func hasPrefix(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}