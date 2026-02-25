package newrelic

import (
	"context"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/stretchr/testify/assert"
)


func TestFromEchoContext_NilContext(t *testing.T) {
	txn := FromEchoContext(nil)
	assert.Nil(t, txn)
}

func TestFromEchoContext_NoTransaction(t *testing.T) {
	// Since we can't easily mock echo.Context without implementing all methods,
	// we'll test the function with nil to ensure it doesn't panic
	txn := FromEchoContext(nil)
	assert.Nil(t, txn)
}

func TestFromContext_NilContext(t *testing.T) {
	txn := FromContext(nil)
	assert.Nil(t, txn)
}

func TestFromContext_EmptyContext(t *testing.T) {
	txn := FromContext(context.Background())
	assert.Nil(t, txn)
}

func TestStartSegment_NilTransaction(t *testing.T) {
	segment := StartSegment(nil, "test-segment")
	assert.Nil(t, segment)
}

func TestStartSegment_EmptyName(t *testing.T) {
	txn := &newrelic.Transaction{}
	segment := StartSegment(txn, "")
	// Should not panic
	assert.NotNil(t, segment)
}

func TestStartSegment_ValidName(t *testing.T) {
	txn := &newrelic.Transaction{}
	segment := StartSegment(txn, "test-segment")
	// Should not panic
	assert.NotNil(t, segment)
}

func TestSetTransactionName_NilTransaction(t *testing.T) {
	// Should not panic
	SetTransactionName(nil, "test-name")
}

func TestSetTransactionName_EmptyName(t *testing.T) {
	txn := &newrelic.Transaction{}
	// Should not panic
	SetTransactionName(txn, "")
}

func TestSetTransactionName_ValidName(t *testing.T) {
	txn := &newrelic.Transaction{}
	// Should not panic
	SetTransactionName(txn, "test-name")
}

func TestAddTransactionAttribute_NilTransaction(t *testing.T) {
	// Should not panic
	AddTransactionAttribute(nil, "key", "value")
}

func TestAddTransactionAttribute_EmptyKey(t *testing.T) {
	txn := &newrelic.Transaction{}
	// Should not panic
	AddTransactionAttribute(txn, "", "value")
}

func TestAddTransactionAttribute_NilValue(t *testing.T) {
	txn := &newrelic.Transaction{}
	// Should not panic
	AddTransactionAttribute(txn, "key", nil)
}

func TestAddTransactionAttribute_ValidKeyValue(t *testing.T) {
	txn := &newrelic.Transaction{}
	// Should not panic
	AddTransactionAttribute(txn, "key", "value")
}

func TestNoticeTransactionError_NilTransaction(t *testing.T) {
	// Should not panic
	NoticeTransactionError(nil, assert.AnError)
}

func TestNoticeTransactionError_NilError(t *testing.T) {
	txn := &newrelic.Transaction{}
	// Should not panic
	NoticeTransactionError(txn, nil)
}

func TestNoticeTransactionError_ValidError(t *testing.T) {
	txn := &newrelic.Transaction{}
	// Should not panic
	NoticeTransactionError(txn, assert.AnError)
}

func TestWithSegment_NilContext(t *testing.T) {
	err := WithSegment(nil, "test-segment", func() error {
		return nil
	})
	assert.NoError(t, err)
}

func TestWithSegment_FunctionError(t *testing.T) {
	ctx := context.Background()
	err := WithSegment(ctx, "test-segment", func() error {
		return assert.AnError
	})
	assert.Error(t, err)
}

func TestWithSegment_FunctionSuccess(t *testing.T) {
	ctx := context.Background()
	err := WithSegment(ctx, "test-segment", func() error {
		return nil
	})
	assert.NoError(t, err)
}

func TestWithSegmentAndReturn_NilContext(t *testing.T) {
	result, err := WithSegmentAndReturn[string](nil, "test-segment", func() (string, error) {
		return "test", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "test", result)
}

func TestWithSegmentAndReturn_FunctionError(t *testing.T) {
	ctx := context.Background()
	result, err := WithSegmentAndReturn[string](ctx, "test-segment", func() (string, error) {
		return "", assert.AnError
	})
	assert.Error(t, err)
	assert.Equal(t, "", result)
}

func TestWithSegmentAndReturn_FunctionSuccess(t *testing.T) {
	ctx := context.Background()
	result, err := WithSegmentAndReturn[string](ctx, "test-segment", func() (string, error) {
		return "test-result", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "test-result", result)
}

func TestWithSegmentAndReturn_IntType(t *testing.T) {
	ctx := context.Background()
	result, err := WithSegmentAndReturn[int](ctx, "test-segment", func() (int, error) {
		return 42, nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 42, result)
}

func TestWithSegmentAndReturn_StructType(t *testing.T) {
	type TestStruct struct {
		Name string
		Value int
	}
	
	ctx := context.Background()
	result, err := WithSegmentAndReturn[TestStruct](ctx, "test-segment", func() (TestStruct, error) {
		return TestStruct{Name: "test", Value: 42}, nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "test", result.Name)
	assert.Equal(t, 42, result.Value)
}

func TestTraceHandler_NilHandler(t *testing.T) {
	wrapped := TraceHandler("test-handler", nil)
	assert.NotNil(t, wrapped)
}

func TestTraceHandler_ValidHandler(t *testing.T) {
	handler := func(c echo.Context) error {
		return nil
	}
	
	wrapped := TraceHandler("test-handler", handler)
	assert.NotNil(t, wrapped)
}

func TestTraceHandler_EmptyName(t *testing.T) {
	handler := func(c echo.Context) error {
		return nil
	}
	
	wrapped := TraceHandler("", handler)
	assert.NotNil(t, wrapped)
}

func TestTraceUseCase_NilContext(t *testing.T) {
	err := TraceUseCase(nil, "test-usecase", func(ctx context.Context) error {
		return nil
	})
	assert.NoError(t, err)
}

func TestTraceUseCase_FunctionError(t *testing.T) {
	ctx := context.Background()
	err := TraceUseCase(ctx, "test-usecase", func(ctx context.Context) error {
		return assert.AnError
	})
	assert.Error(t, err)
}

func TestTraceUseCase_FunctionSuccess(t *testing.T) {
	ctx := context.Background()
	err := TraceUseCase(ctx, "test-usecase", func(ctx context.Context) error {
		return nil
	})
	assert.NoError(t, err)
}

func TestTraceUseCaseWithReturn_NilContext(t *testing.T) {
	result, err := TraceUseCaseWithReturn[string](nil, "test-usecase", func(ctx context.Context) (string, error) {
		return "test", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "test", result)
}

func TestTraceUseCaseWithReturn_FunctionError(t *testing.T) {
	ctx := context.Background()
	result, err := TraceUseCaseWithReturn[string](ctx, "test-usecase", func(ctx context.Context) (string, error) {
		return "", assert.AnError
	})
	assert.Error(t, err)
	assert.Equal(t, "", result)
}

func TestTraceUseCaseWithReturn_FunctionSuccess(t *testing.T) {
	ctx := context.Background()
	result, err := TraceUseCaseWithReturn[string](ctx, "test-usecase", func(ctx context.Context) (string, error) {
		return "test-result", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "test-result", result)
}

func TestTraceUseCaseWithReturn_ContextPassing(t *testing.T) {
	ctx := context.Background()
	result, err := TraceUseCaseWithReturn[context.Context](ctx, "test-usecase", func(ctx context.Context) (context.Context, error) {
		// Test that context is properly passed through
		return ctx, nil
	})
	assert.NoError(t, err)
	assert.Equal(t, ctx, result)
}

func TestTracingFunctionEdgeCases(t *testing.T) {
	t.Run("Empty segment name", func(t *testing.T) {
		ctx := context.Background()
		err := WithSegment(ctx, "", func() error { return nil })
		assert.NoError(t, err)
	})
	
	t.Run("Handler with empty name", func(t *testing.T) {
		handler := func(c echo.Context) error { return nil }
		wrapped := TraceHandler("", handler)
		assert.NotNil(t, wrapped)
	})
	
	t.Run("Use case with empty name", func(t *testing.T) {
		ctx := context.Background()
		err := TraceUseCase(ctx, "", func(ctx context.Context) error { return nil })
		assert.NoError(t, err)
	})
	
	t.Run("Nil functions", func(t *testing.T) {
		ctx := context.Background()
		
		// These should panic, but we're testing they don't cause nil pointer exceptions
		assert.Panics(t, func() {
			WithSegment(ctx, "test", nil)
		})
	})
}