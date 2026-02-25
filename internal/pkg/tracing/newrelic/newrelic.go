package newrelic

import (
	"context"
	"math/rand"
	"net/http"
	"strings"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/piresc/nebengjek/internal/pkg/tracing"
)

// NewRelicTracer implements the tracing.Tracer interface using New Relic
type NewRelicTracer struct {
	app    *newrelic.Application
	config *tracing.Config
}

// NewTracer creates a new New Relic tracer instance
func NewTracer(config *tracing.Config, nrApp *newrelic.Application) tracing.Tracer {
	if config.NoOpForTesting || !config.Enabled || nrApp == nil {
		return &NoOpTracer{}
	}

	return &NewRelicTracer{
		app:    nrApp,
		config: config,
	}
}

// StartHTTPRequest starts a HTTP request transaction
func (t *NewRelicTracer) StartHTTPRequest(r *http.Request) (context.Context, tracing.HTTPTransaction) {
	txn := t.app.StartTransaction("")
	txn.SetWebRequestHTTP(r)

	// Add distributed tracing headers
	txn.InsertDistributedTraceHeaders(r.Header)

	// Preserve the incoming request context so deadlines, cancellation,
	// and values from upstream middleware are not lost.
	return newrelic.NewContext(r.Context(), txn), &newRelicTransaction{txn: txn}
}

// StartExternalCall starts an external call segment
func (t *NewRelicTracer) StartExternalCall(ctx context.Context, host, method string) (context.Context, func()) {
	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return ctx, func() {}
	}

	segment := &newrelic.ExternalSegment{
		StartTime: newrelic.StartSegmentNow(txn),
		Host:      host,
		Procedure: method,
	}

	return ctx, func() {
		segment.End()
	}
}

// StartMessage starts a message operation segment
func (t *NewRelicTracer) StartMessage(ctx context.Context, topic, operation string) (context.Context, func()) {
	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return ctx, func() {}
	}

	segment := txn.StartSegment("Message")
	segment.AddAttribute("message.topic", topic)
	segment.AddAttribute("message.operation", operation)

	return ctx, func() {
		segment.End()
	}
}

// StartDatabase starts a database operation segment
func (t *NewRelicTracer) StartDatabase(ctx context.Context, operation, table string) (context.Context, func()) {
	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return ctx, func() {}
	}

	segment := &newrelic.DatastoreSegment{
		StartTime:  newrelic.StartSegmentNow(txn),
		Product:    newrelic.DatastorePostgres,
		Collection: table,
		Operation:  operation,
	}

	return ctx, func() {
		segment.End()
	}
}

// StartSegment starts a custom business segment
func (t *NewRelicTracer) StartSegment(ctx context.Context, name string) (context.Context, func()) {
	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return ctx, func() {}
	}

	segment := txn.StartSegment(name)
	return ctx, func() {
		segment.End()
	}
}

// InjectContext injects tracing context into a carrier
func (t *NewRelicTracer) InjectContext(ctx context.Context, carrier interface{}) {
	switch c := carrier.(type) {
	case http.Header:
		txn := newrelic.FromContext(ctx)
		if txn != nil {
			txn.InsertDistributedTraceHeaders(c)
		}
	}
}

// ExtractContext extracts tracing context from a carrier
func (t *NewRelicTracer) ExtractContext(ctx context.Context, carrier interface{}) context.Context {
	return ctx
}

// IsEnabled returns whether tracing is enabled
func (t *NewRelicTracer) IsEnabled() bool {
	return t.config.Enabled
}

// ShouldTrace determines if a path should be traced
func (t *NewRelicTracer) ShouldTrace(path string) bool {
	if !t.config.Enabled {
		return false
	}

	// Apply sampling
	if t.config.SampleRate < 1.0 && !shouldSample(t.config.SampleRate) {
		return false
	}

	// Check exclusions
	for _, excluded := range t.config.ExcludePaths {
		if strings.Contains(path, excluded) {
			return false
		}
	}

	return true
}

// newRelicTransaction implements tracing.HTTPTransaction for New Relic
type newRelicTransaction struct {
	txn *newrelic.Transaction
}

func (t *newRelicTransaction) Context() context.Context {
	return newrelic.NewContext(context.Background(), t.txn)
}

func (t *newRelicTransaction) SetName(name string) {
	t.txn.SetName(name)
}

func (t *newRelicTransaction) AddAttribute(key, value string) {
	t.txn.AddAttribute(key, value)
}

func (t *newRelicTransaction) AddError(err error) {
	t.txn.NoticeError(err)
}

func (t *newRelicTransaction) End() {
	t.txn.End()
}


// shouldSample implements simple random sampling
func shouldSample(rate float64) bool {
	if rate >= 1.0 {
		return true
	}
	return rand.Float64() < rate
}

// NoOpTracer provides a no-operation implementation for testing
type NoOpTracer struct{}

func (t *NoOpTracer) StartHTTPRequest(r *http.Request) (context.Context, tracing.HTTPTransaction) {
	return context.Background(), &NoOpTransaction{}
}

func (t *NoOpTracer) StartExternalCall(ctx context.Context, host, method string) (context.Context, func()) {
	return ctx, func() {}
}

func (t *NoOpTracer) StartMessage(ctx context.Context, topic, operation string) (context.Context, func()) {
	return ctx, func() {}
}

func (t *NoOpTracer) StartDatabase(ctx context.Context, operation, table string) (context.Context, func()) {
	return ctx, func() {}
}

func (t *NoOpTracer) StartSegment(ctx context.Context, name string) (context.Context, func()) {
	return ctx, func() {}
}

func (t *NoOpTracer) InjectContext(ctx context.Context, carrier interface{}) {}

func (t *NoOpTracer) ExtractContext(ctx context.Context, carrier interface{}) context.Context {
	return ctx
}

func (t *NoOpTracer) IsEnabled() bool {
	return false
}

func (t *NoOpTracer) ShouldTrace(path string) bool {
	return false
}

// NoOpTransaction provides a no-operation transaction implementation
type NoOpTransaction struct{}

func (t *NoOpTransaction) Context() context.Context { return context.Background() }
func (t *NoOpTransaction) SetName(name string)      {}
func (t *NoOpTransaction) AddAttribute(key, value string) {}
func (t *NoOpTransaction) AddError(err error)       {}
func (t *NoOpTransaction) End()                     {}