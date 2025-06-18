package observability

import (
	"context"
	"net/http"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// Tracer provides an abstraction for APM tracing
type Tracer interface {
	StartTransaction(name string) Transaction
	StartSegment(ctx context.Context, name string) (context.Context, func())
}

// Transaction represents a traced transaction
type Transaction interface {
	End()
	SetWebRequest(*http.Request)
	SetWebResponse(http.ResponseWriter)
	NoticeError(error)
	AddAttribute(key string, value interface{})
	GetContext() context.Context
	SetTag(key string, value interface{})
	SetError(error)
}

// NoOpTracer provides a no-operation implementation for testing
type NoOpTracer struct{}

// NoOpTransaction provides a no-operation transaction for testing
type NoOpTransaction struct {
	ctx context.Context
}

// NewNoOpTracer creates a new no-operation tracer
func NewNoOpTracer() *NoOpTracer {
	return &NoOpTracer{}
}

// StartTransaction creates a no-op transaction
func (t *NoOpTracer) StartTransaction(name string) Transaction {
	return &NoOpTransaction{ctx: context.Background()}
}

// StartSegment creates a no-op segment
func (t *NoOpTracer) StartSegment(ctx context.Context, name string) (context.Context, func()) {
	return ctx, func() {}
}

// No-op transaction methods
func (t *NoOpTransaction) End()                                       {}
func (t *NoOpTransaction) SetWebRequest(*http.Request)                {}
func (t *NoOpTransaction) SetWebResponse(http.ResponseWriter)         {}
func (t *NoOpTransaction) NoticeError(error)                          {}
func (t *NoOpTransaction) AddAttribute(key string, value interface{}) {}
func (t *NoOpTransaction) GetContext() context.Context                { return t.ctx }
func (t *NoOpTransaction) SetError(error)                             {}
func (t *NoOpTransaction) SetTag(key string, value interface{})       {}

// NewRelicTracer implements Tracer interface using New Relic
type NewRelicTracer struct {
	app *newrelic.Application
}

// NewNewRelicTracer creates a new New Relic tracer
func NewNewRelicTracer(app *newrelic.Application) *NewRelicTracer {
	if app == nil {
		return nil
	}
	return &NewRelicTracer{app: app}
}

// StartTransaction creates a new New Relic transaction
func (t *NewRelicTracer) StartTransaction(name string) Transaction {
	txn := t.app.StartTransaction(name)
	return &NewRelicTransaction{txn: txn}
}

// StartSegment creates a new segment within the current transaction
func (t *NewRelicTracer) StartSegment(ctx context.Context, name string) (context.Context, func()) {
	if txn := newrelic.FromContext(ctx); txn != nil {
		segment := txn.StartSegment(name)
		return ctx, segment.End
	}
	return ctx, func() {}
}

// NewRelicTransaction wraps a New Relic transaction
type NewRelicTransaction struct {
	txn *newrelic.Transaction
}

// End finishes the transaction
func (t *NewRelicTransaction) End() {
	if t.txn != nil {
		t.txn.End()
	}
}

// SetWebRequest sets the web request for the transaction
func (t *NewRelicTransaction) SetWebRequest(r *http.Request) {
	if t.txn != nil {
		t.txn.SetWebRequestHTTP(r)
	}
}

// SetWebResponse sets the web response for the transaction
func (t *NewRelicTransaction) SetWebResponse(w http.ResponseWriter) {
	if t.txn != nil {
		t.txn.SetWebResponse(w)
	}
}

// NoticeError reports an error to the transaction
func (t *NewRelicTransaction) NoticeError(err error) {
	if t.txn != nil && err != nil {
		t.txn.NoticeError(err)
	}
}

// AddAttribute adds a custom attribute to the transaction
func (t *NewRelicTransaction) AddAttribute(key string, value interface{}) {
	if t.txn != nil {
		t.txn.AddAttribute(key, value)
	}
}

// GetContext returns the context with the transaction
func (t *NewRelicTransaction) GetContext() context.Context {
	if t.txn != nil {
		return newrelic.NewContext(context.Background(), t.txn)
	}
	return context.Background()
}

// SetTag adds a custom tag to the transaction (alias for AddAttribute)
func (t *NewRelicTransaction) SetTag(key string, value interface{}) {
	if t.txn != nil {
		t.txn.AddAttribute(key, value)
	}
}

// SetError reports an error to the transaction (alias for NoticeError)
func (t *NewRelicTransaction) SetError(err error) {
	if t.txn != nil && err != nil {
		t.txn.NoticeError(err)
	}
}

// TracerFactory creates tracers based on configuration
type TracerFactory struct{}

// NewTracerFactory creates a new tracer factory
func NewTracerFactory() *TracerFactory {
	return &TracerFactory{}
}

// CreateTracer creates a tracer based on the provided New Relic app
func (f *TracerFactory) CreateTracer(nrApp *newrelic.Application) Tracer {
	if nrApp != nil {
		return NewNewRelicTracer(nrApp)
	}
	return NewNoOpTracer()
}

// SegmentHelper provides utilities for creating segments
type SegmentHelper struct {
	tracer Tracer
}

// NewSegmentHelper creates a new segment helper
func NewSegmentHelper(tracer Tracer) *SegmentHelper {
	return &SegmentHelper{tracer: tracer}
}

// StartDatabaseSegment starts a database segment
func (h *SegmentHelper) StartDatabaseSegment(ctx context.Context, operation, table string) (context.Context, func()) {
	segmentName := "Database/" + operation
	if table != "" {
		segmentName += "/" + table
	}
	return h.tracer.StartSegment(ctx, segmentName)
}

// StartExternalSegment starts an external service segment
func (h *SegmentHelper) StartExternalSegment(ctx context.Context, service, operation string) (context.Context, func()) {
	segmentName := "External/" + service
	if operation != "" {
		segmentName += "/" + operation
	}
	return h.tracer.StartSegment(ctx, segmentName)
}

// StartCustomSegment starts a custom segment
func (h *SegmentHelper) StartCustomSegment(ctx context.Context, name string) (context.Context, func()) {
	return h.tracer.StartSegment(ctx, "Custom/"+name)
}
