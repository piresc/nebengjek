package nats

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/piresc/nebengjek/internal/pkg/tracing"
)

// TracedClient wraps a NATS client with automatic tracing
type TracedClient struct {
	client *Client
	tracer tracing.Tracer
}

// NewTracedClient creates a new traced NATS client
func NewTracedClient(client *Client, tracer tracing.Tracer) *TracedClient {
	return &TracedClient{
		client: client,
		tracer: tracer,
	}
}

// GetClient returns the underlying NATS client
func (tc *TracedClient) GetClient() *Client {
	return tc.client
}

// GetConn returns the underlying NATS connection
func (tc *TracedClient) GetConn() *nats.Conn {
	return tc.client.GetConn()
}

// Close closes the NATS client
func (tc *TracedClient) Close() {
	tc.client.Close()
}

// Publish publishes a message with automatic tracing
func (tc *TracedClient) Publish(subject string, data []byte) error {
	if !tc.tracer.IsEnabled() {
		return tc.client.Publish(subject, data)
	}

	_, done := tc.tracer.StartMessage(context.Background(), subject, "publish")
	defer done()

	return tc.client.Publish(subject, data)
}

// PublishWithOptions publishes a message with options and automatic tracing
func (tc *TracedClient) PublishWithOptions(opts PublishOptions) error {
	if !tc.tracer.IsEnabled() {
		return tc.client.PublishWithOptions(opts)
	}

	_, done := tc.tracer.StartMessage(context.Background(), opts.Subject, "publish")
	defer done()

	return tc.client.PublishWithOptions(opts)
}

// Subscribe subscribes to a subject with automatic tracing
func (tc *TracedClient) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	if !tc.tracer.IsEnabled() {
		return tc.client.Subscribe(subject, handler)
	}

	// Wrap handler with automatic tracing
	tracedHandler := func(msg *nats.Msg) {
		_, done := tc.tracer.StartMessage(context.Background(), msg.Subject, "consume")
		defer done()

		handler(msg)
	}

	return tc.client.Subscribe(subject, tracedHandler)
}

// Request sends a request and waits for a response with automatic tracing
func (tc *TracedClient) Request(subject string, data []byte) (*nats.Msg, error) {
	if !tc.tracer.IsEnabled() {
		return tc.client.Request(subject, data)
	}

	_, done := tc.tracer.StartMessage(context.Background(), subject, "request")
	defer done()

	return tc.client.Request(subject, data)
}