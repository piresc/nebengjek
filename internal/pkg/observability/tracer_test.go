package observability

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test data structures
type TestSpanData struct {
	mu         sync.RWMutex
	Name       string
	Operation  string
	Tags       map[string]interface{}
	StartTime  time.Time
	EndTime    time.Time
	Duration   time.Duration
	Error      error
	Finished   bool
}

// MockTracer implements the Tracer interface for testing
type MockTracer struct {
	mu          sync.RWMutex
	spans       []*TestSpanData
	currentSpan *TestSpanData
	errorToSet error
}

func NewMockTracer() *MockTracer {
	return &MockTracer{
		spans: make([]*TestSpanData, 0),
	}
}

func (m *MockTracer) StartTransaction(name string) Transaction {
	span := &TestSpanData{
		Name:      name,
		Operation: "transaction",
		Tags:      make(map[string]interface{}),
		StartTime: time.Now(),
	}
	m.mu.Lock()
	m.spans = append(m.spans, span)
	m.currentSpan = span
	m.mu.Unlock()
	return &MockTransaction{span: span, tracer: m}
}

func (m *MockTracer) StartSegment(ctx context.Context, name string) (context.Context, func()) {
	return ctx, func() {}
}

func (m *MockTracer) GetSpans() []*TestSpanData {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Return a copy to avoid race conditions
	spansCopy := make([]*TestSpanData, len(m.spans))
	copy(spansCopy, m.spans)
	return spansCopy
}

func (m *MockTracer) GetCurrentSpan() *TestSpanData {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentSpan
}

func (m *MockTracer) SetErrorToSet(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorToSet = err
}

func (m *MockTracer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.spans = make([]*TestSpanData, 0)
	m.currentSpan = nil
	m.errorToSet = nil
}

// MockTransaction implements the Transaction interface for testing
type MockTransaction struct {
	span   *TestSpanData
	tracer *MockTracer
}

func (m *MockTransaction) SetTag(key string, value interface{}) {
	if m.span != nil {
		m.span.mu.Lock()
		m.span.Tags[key] = value
		m.span.mu.Unlock()
	}
}

func (m *MockTransaction) SetError(err error) {
	if m.span != nil {
		m.span.mu.Lock()
		m.span.Error = err
		m.span.mu.Unlock()
	}
}

func (m *MockTransaction) End() {
	if m.span != nil {
		m.span.mu.Lock()
		m.span.EndTime = time.Now()
		m.span.Duration = m.span.EndTime.Sub(m.span.StartTime)
		m.span.Finished = true
		m.span.mu.Unlock()
	}
}

func (m *MockTransaction) SetWebRequest(req *http.Request) {
	// Mock implementation - store request info if needed
}

func (m *MockTransaction) SetWebResponse(w http.ResponseWriter) {
	// Mock implementation - store response info if needed
}

func (m *MockTransaction) NoticeError(err error) {
	if m.span != nil {
		m.span.Error = err
	}
}

func (m *MockTransaction) AddAttribute(key string, value interface{}) {
	if m.span != nil {
		m.span.Tags[key] = value
	}
}

func (m *MockTransaction) GetContext() context.Context {
	return context.Background()
}

func TestNoOpTracer(t *testing.T) {
	t.Run("NoOpTracer implements Tracer interface", func(t *testing.T) {
		var tracer Tracer = &NoOpTracer{}
		assert.NotNil(t, tracer)
	})

	t.Run("StartTransaction returns NoOpTransaction", func(t *testing.T) {
		tracer := &NoOpTracer{}

		transaction := tracer.StartTransaction("test-transaction")

		assert.NotNil(t, transaction)
		assert.IsType(t, &NoOpTransaction{}, transaction)
	})

	t.Run("StartTransaction with different parameters", func(t *testing.T) {
		tracer := &NoOpTracer{}

		testCases := []string{
			"user-login",
			"database-query",
			"api-call",
			"message-processing",
			"",
			"very-long-transaction-name-with-special-chars-123!@#",
		}

		for _, tc := range testCases {
			t.Run(tc, func(t *testing.T) {
				transaction := tracer.StartTransaction(tc)
				assert.NotNil(t, transaction)
				assert.IsType(t, &NoOpTransaction{}, transaction)
			})
		}
	})

	t.Run("StartTransaction with empty name", func(t *testing.T) {
		tracer := &NoOpTracer{}

		// This should not panic
		assert.NotPanics(t, func() {
			transaction := tracer.StartTransaction("")
			assert.NotNil(t, transaction)
		})
	})
}

func TestNoOpTransaction(t *testing.T) {
	t.Run("NoOpTransaction implements Transaction interface", func(t *testing.T) {
		var transaction Transaction = &NoOpTransaction{}
		assert.NotNil(t, transaction)
	})

	t.Run("SetTag does not panic", func(t *testing.T) {
		transaction := &NoOpTransaction{}

		assert.NotPanics(t, func() {
			transaction.SetTag("key", "value")
			transaction.SetTag("number", 42)
			transaction.SetTag("boolean", true)
			transaction.SetTag("nil", nil)
			transaction.SetTag("", "")
		})
	})

	t.Run("SetError does not panic", func(t *testing.T) {
		transaction := &NoOpTransaction{}

		assert.NotPanics(t, func() {
			transaction.SetError(nil)
			transaction.SetError(assert.AnError)
			transaction.SetError(context.DeadlineExceeded)
		})
	})

	t.Run("End does not panic", func(t *testing.T) {
		transaction := &NoOpTransaction{}

		assert.NotPanics(t, func() {
			transaction.End()
			transaction.End() // Multiple calls should be safe
			transaction.End()
		})
	})

	t.Run("Full transaction lifecycle", func(t *testing.T) {
		transaction := &NoOpTransaction{}

		assert.NotPanics(t, func() {
			transaction.SetTag("user_id", "123")
			transaction.SetTag("operation", "create_user")
			transaction.SetTag("duration_ms", 150)
			transaction.SetError(assert.AnError)
			transaction.End()
		})
	})
}

func TestMockTracer(t *testing.T) {
	t.Run("MockTracer StartTransaction", func(t *testing.T) {
		tracer := NewMockTracer()

		transaction := tracer.StartTransaction("test-transaction")

		assert.NotNil(t, transaction)
		assert.IsType(t, &MockTransaction{}, transaction)

		// Check that span was created
		spans := tracer.GetSpans()
		require.Len(t, spans, 1)
		assert.Equal(t, "test-transaction", spans[0].Name)
		assert.False(t, spans[0].StartTime.IsZero())
		assert.False(t, spans[0].Finished)
	})

	t.Run("MockTracer multiple transactions", func(t *testing.T) {
		tracer := NewMockTracer()

		transactions := []struct {
			name string
		}{
			{"user-auth"},
			{"db-query"},
			{"api-call"},
		}

		for _, tx := range transactions {
			transaction := tracer.StartTransaction(tx.name)
			assert.NotNil(t, transaction)
		}

		spans := tracer.GetSpans()
		require.Len(t, spans, len(transactions))

		for i, tx := range transactions {
			assert.Equal(t, tx.name, spans[i].Name)
			assert.Equal(t, "transaction", spans[i].Operation)
		}
	})
}

func TestMockTransaction(t *testing.T) {
	t.Run("MockTransaction SetTag", func(t *testing.T) {
		tracer := NewMockTracer()

		transaction := tracer.StartTransaction("test")
		mockTx := transaction.(*MockTransaction)

		transaction.SetTag("user_id", "123")
		transaction.SetTag("method", "POST")
		transaction.SetTag("status_code", 200)
		transaction.SetTag("success", true)

		assert.Equal(t, "123", mockTx.span.Tags["user_id"])
		assert.Equal(t, "POST", mockTx.span.Tags["method"])
		assert.Equal(t, 200, mockTx.span.Tags["status_code"])
		assert.Equal(t, true, mockTx.span.Tags["success"])
	})

	t.Run("MockTransaction SetError", func(t *testing.T) {
		tracer := NewMockTracer()

		transaction := tracer.StartTransaction("test")
		mockTx := transaction.(*MockTransaction)

		testError := assert.AnError
		transaction.SetError(testError)

		assert.Equal(t, testError, mockTx.span.Error)
	})

	t.Run("MockTransaction End", func(t *testing.T) {
		tracer := NewMockTracer()

		transaction := tracer.StartTransaction("test")
		mockTx := transaction.(*MockTransaction)

		// Add some delay to test duration calculation
		time.Sleep(10 * time.Millisecond)

		assert.False(t, mockTx.span.Finished)
		assert.True(t, mockTx.span.EndTime.IsZero())
		assert.Equal(t, time.Duration(0), mockTx.span.Duration)

		transaction.End()

		assert.True(t, mockTx.span.Finished)
		assert.False(t, mockTx.span.EndTime.IsZero())
		assert.Greater(t, mockTx.span.Duration, time.Duration(0))
		assert.GreaterOrEqual(t, mockTx.span.Duration, 10*time.Millisecond)
	})

	t.Run("MockTransaction full lifecycle", func(t *testing.T) {
		tracer := NewMockTracer()

		transaction := tracer.StartTransaction("user-registration")

		// Simulate transaction operations
		transaction.SetTag("user_id", "user-123")
		transaction.SetTag("email", "test@example.com")
		transaction.SetTag("registration_type", "email")
		transaction.SetTag("ip_address", "192.168.1.1")

		// Simulate some processing time
		time.Sleep(5 * time.Millisecond)

		// Simulate an error
		testError := assert.AnError
		transaction.SetError(testError)

		// End transaction
		transaction.End()

		// Verify the span data
		spans := tracer.GetSpans()
		require.Len(t, spans, 1)

		span := spans[0]
		assert.Equal(t, "user-registration", span.Name)
		assert.Equal(t, "transaction", span.Operation)
		assert.Equal(t, "user-123", span.Tags["user_id"])
		assert.Equal(t, "test@example.com", span.Tags["email"])
		assert.Equal(t, "email", span.Tags["registration_type"])
		assert.Equal(t, "192.168.1.1", span.Tags["ip_address"])
		assert.Equal(t, testError, span.Error)
		assert.True(t, span.Finished)
		assert.Greater(t, span.Duration, time.Duration(0))
	})
}

func TestTracerIntegration(t *testing.T) {
	t.Run("Compare NoOpTracer and MockTracer behavior", func(t *testing.T) {
		tracers := []struct {
			name   string
			tracer Tracer
		}{
			{"NoOpTracer", &NoOpTracer{}},
			{"MockTracer", NewMockTracer()},
		}

		for _, tc := range tracers {
			t.Run(tc.name, func(t *testing.T) {
				// Both should not panic and return valid transactions
				assert.NotPanics(t, func() {
					transaction := tc.tracer.StartTransaction("test")
					assert.NotNil(t, transaction)

					transaction.SetTag("key", "value")
					transaction.SetError(assert.AnError)
					transaction.End()
				})
			})
		}
	})

	t.Run("Nested transactions simulation", func(t *testing.T) {
		tracer := NewMockTracer()

		// Parent transaction
		parentTx := tracer.StartTransaction("parent-operation")
		parentTx.SetTag("level", "parent")

		// Child transactions
		for i := 0; i < 3; i++ {
			childTx := tracer.StartTransaction(fmt.Sprintf("child-operation-%d", i))
			childTx.SetTag("level", "child")
			childTx.SetTag("index", i)
			childTx.End()
		}

		parentTx.End()

		// Verify all transactions were recorded
		spans := tracer.GetSpans()
		require.Len(t, spans, 4) // 1 parent + 3 children

		// Verify parent
		assert.Equal(t, "parent-operation", spans[0].Name)
		assert.Equal(t, "transaction", spans[0].Operation)
		assert.Equal(t, "parent", spans[0].Tags["level"])

		// Verify children
		for i := 1; i < 4; i++ {
			assert.Equal(t, fmt.Sprintf("child-operation-%d", i-1), spans[i].Name)
			assert.Equal(t, "transaction", spans[i].Operation)
			assert.Equal(t, "child", spans[i].Tags["level"])
			assert.Equal(t, i-1, spans[i].Tags["index"])
			assert.True(t, spans[i].Finished)
		}
	})
}

func TestTracerErrorScenarios(t *testing.T) {
	t.Run("Handle various error types", func(t *testing.T) {
		tracer := NewMockTracer()

		errorTypes := []error{
			nil,
			assert.AnError,
			context.DeadlineExceeded,
			context.Canceled,
			&customError{message: "custom error"},
		}

		for i, err := range errorTypes {
			transaction := tracer.StartTransaction(fmt.Sprintf("error-test-%d", i))
			transaction.SetError(err)
			transaction.End()
		}

		spans := tracer.GetSpans()
		require.Len(t, spans, len(errorTypes))

		for i, expectedErr := range errorTypes {
			assert.Equal(t, expectedErr, spans[i].Error)
		}
	})
}

// Custom error type for testing
type customError struct {
	message string
}

func (e *customError) Error() string {
	return e.message
}

func TestTracerConcurrency(t *testing.T) {
	t.Run("Concurrent transaction creation", func(t *testing.T) {
		tracer := NewMockTracer()
		numGoroutines := 10
		numTransactionsPerGoroutine := 5

		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				for j := 0; j < numTransactionsPerGoroutine; j++ {
					transaction := tracer.StartTransaction(fmt.Sprintf("concurrent-%d-%d", goroutineID, j))
					transaction.SetTag("goroutine_id", goroutineID)
					transaction.SetTag("transaction_id", j)
					transaction.End()
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Verify all transactions were recorded
		spans := tracer.GetSpans()
		expectedTotal := numGoroutines * numTransactionsPerGoroutine
		assert.Len(t, spans, expectedTotal)

		// Verify all spans are finished
		for _, span := range spans {
			assert.True(t, span.Finished)
			assert.Contains(t, span.Name, "concurrent-")
			assert.Equal(t, "transaction", span.Operation)
		}
	})
}

func BenchmarkNoOpTracer_StartTransaction(b *testing.B) {
	tracer := &NoOpTracer{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transaction := tracer.StartTransaction("benchmark-transaction")
		transaction.SetTag("iteration", i)
		transaction.End()
	}
}

func BenchmarkMockTracer_StartTransaction(b *testing.B) {
	tracer := NewMockTracer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transaction := tracer.StartTransaction("benchmark-transaction")
		transaction.SetTag("iteration", i)
		transaction.End()
	}
}

func BenchmarkTransaction_Operations(b *testing.B) {
	tracer := NewMockTracer()

	b.Run("SetTag", func(b *testing.B) {
		transaction := tracer.StartTransaction("benchmark")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			transaction.SetTag(fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
		}
		transaction.End()
	})

	b.Run("SetError", func(b *testing.B) {
		transaction := tracer.StartTransaction("benchmark")
		testError := &customError{message: "benchmark error"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			transaction.SetError(testError)
		}
		transaction.End()
	})

	b.Run("End", func(b *testing.B) {
		transactions := make([]Transaction, b.N)
		for i := 0; i < b.N; i++ {
			transactions[i] = tracer.StartTransaction("benchmark")
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			transactions[i].End()
		}
	})
}