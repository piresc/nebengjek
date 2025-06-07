package newrelic

import (
	"context"

	"github.com/labstack/echo/v4"
	"github.com/newrelic/go-agent/v3/integrations/nrecho-v4"
	"github.com/newrelic/go-agent/v3/newrelic"
)

// FromEchoContext extracts New Relic transaction from Echo context
// This is the standard way to get transactions in HTTP handlers
func FromEchoContext(c echo.Context) *newrelic.Transaction {
	return nrecho.FromContext(c)
}

// FromContext extracts New Relic transaction from standard context
// This is used in use cases and other business logic layers
func FromContext(ctx context.Context) *newrelic.Transaction {
	return newrelic.FromContext(ctx)
}

// StartSegment creates a new segment for the given transaction
// Returns nil if transaction is not available
func StartSegment(txn *newrelic.Transaction, name string) *newrelic.Segment {
	if txn == nil {
		return nil
	}
	return txn.StartSegment(name)
}

// SetTransactionName sets the name of the transaction for better visibility
func SetTransactionName(txn *newrelic.Transaction, name string) {
	if txn != nil {
		txn.SetName(name)
	}
}

// AddTransactionAttribute adds a custom attribute to the transaction
func AddTransactionAttribute(txn *newrelic.Transaction, key string, value interface{}) {
	if txn != nil {
		txn.AddAttribute(key, value)
	}
}

// NoticeTransactionError reports an error to New Relic
func NoticeTransactionError(txn *newrelic.Transaction, err error) {
	if txn != nil && err != nil {
		txn.NoticeError(err)
	}
}

// WithSegment executes a function within a New Relic segment
// This is useful for wrapping business logic with automatic segment management
func WithSegment(ctx context.Context, segmentName string, fn func() error) error {
	txn := FromContext(ctx)
	segment := StartSegment(txn, segmentName)
	if segment != nil {
		defer segment.End()
	}

	return fn()
}

// WithSegmentAndReturn executes a function within a New Relic segment and returns a value
// This is useful for wrapping business logic that returns values
func WithSegmentAndReturn[T any](ctx context.Context, segmentName string, fn func() (T, error)) (T, error) {
	txn := FromContext(ctx)
	segment := StartSegment(txn, segmentName)
	if segment != nil {
		defer segment.End()
	}

	return fn()
}

// TraceHandler wraps an Echo handler with automatic transaction naming and error handling
func TraceHandler(handlerName string, handler echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		txn := FromEchoContext(c)
		SetTransactionName(txn, handlerName)

		err := handler(c)
		if err != nil {
			NoticeTransactionError(txn, err)
		}

		return err
	}
}

// TraceUseCase wraps a use case method with automatic segment creation
func TraceUseCase(ctx context.Context, useCaseName string, fn func(context.Context) error) error {
	return WithSegment(ctx, useCaseName, func() error {
		return fn(ctx)
	})
}

// TraceUseCaseWithReturn wraps a use case method that returns a value
func TraceUseCaseWithReturn[T any](ctx context.Context, useCaseName string, fn func(context.Context) (T, error)) (T, error) {
	return WithSegmentAndReturn(ctx, useCaseName, func() (T, error) {
		return fn(ctx)
	})
}
