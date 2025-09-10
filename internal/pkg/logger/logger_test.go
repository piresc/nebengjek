package logger

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/stretchr/testify/assert"
)

// testWriter implements io.Writer for testing
type testWriter struct{}

func (w *testWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func TestString(t *testing.T) {
	field := String("test_key", "test_value")
	assert.Equal(t, "test_key", field.Key)
	assert.Equal(t, "test_value", field.Value.String())
}

func TestErr(t *testing.T) {
	t.Run("With error", func(t *testing.T) {
		err := assert.AnError
		field := Err(err)
		assert.Equal(t, "error", field.Key)
		assert.Equal(t, err.Error(), field.Value.String())
	})

	t.Run("With nil error", func(t *testing.T) {
		field := Err(nil)
		assert.Equal(t, "error", field.Key)
		assert.Equal(t, "<nil>", field.Value.String())
	})
}

func TestInt(t *testing.T) {
	field := Int("test_key", 42)
	assert.Equal(t, "test_key", field.Key)
	assert.Equal(t, int64(42), field.Value.Int64())
}

func TestInt64(t *testing.T) {
	field := Int64("test_key", int64(9223372036854775807))
	assert.Equal(t, "test_key", field.Key)
	assert.Equal(t, int64(9223372036854775807), field.Value.Int64())
}

func TestUint32(t *testing.T) {
	field := Uint32("test_key", uint32(4294967295))
	assert.Equal(t, "test_key", field.Key)
	assert.Equal(t, uint64(4294967295), field.Value.Uint64())
}

func TestFloat64(t *testing.T) {
	field := Float64("test_key", 3.14159)
	assert.Equal(t, "test_key", field.Key)
	assert.Equal(t, 3.14159, field.Value.Float64())
}

func TestBool(t *testing.T) {
	t.Run("True value", func(t *testing.T) {
		field := Bool("test_key", true)
		assert.Equal(t, "test_key", field.Key)
		assert.Equal(t, true, field.Value.Bool())
	})

	t.Run("False value", func(t *testing.T) {
		field := Bool("test_key", false)
		assert.Equal(t, "test_key", field.Key)
		assert.Equal(t, false, field.Value.Bool())
	})
}

func TestAny(t *testing.T) {
	field := Any("test_key", map[string]interface{}{"nested": "value"})
	assert.Equal(t, "test_key", field.Key)
	assert.Equal(t, map[string]interface{}{"nested": "value"}, field.Value.Any())
}

func TestDuration(t *testing.T) {
	duration := 5 * time.Second
	field := Duration("test_key", duration)
	assert.Equal(t, "test_key", field.Key)
	assert.Equal(t, duration, field.Value.Duration())
}

func TestErrorField(t *testing.T) {
	err := assert.AnError
	field := ErrorField(err)
	assert.Equal(t, "error", field.Key)
	assert.Equal(t, err.Error(), field.Value.String())
}

func TestStrings(t *testing.T) {
	value := []string{"item1", "item2", "item3"}
	field := Strings("test_key", value)
	assert.Equal(t, "test_key", field.Key)
	assert.Equal(t, value, field.Value.Any())
}

func TestGlobalLogger(t *testing.T) {
	// Test initial state
	logger := GetGlobalLogger()
	assert.NotNil(t, logger)
	assert.Equal(t, slog.Default(), logger)

	// Test setting global logger
	newLogger := slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{}))
	SetGlobalLogger(newLogger)

	// Test getting updated logger
	updatedLogger := GetGlobalLogger()
	assert.Equal(t, newLogger, updatedLogger)
}

func TestContextLoggingFunctions(t *testing.T) {
	// Setup a test logger with a proper writer
	testLogger := slog.New(slog.NewTextHandler(&testWriter{}, &slog.HandlerOptions{Level: slog.LevelDebug}))
	SetGlobalLogger(testLogger)

	ctx := context.Background()

	// These should not panic
	InfoCtx(ctx, "test info message", String("key", "value"))
	ErrorCtx(ctx, "test error message", Err(assert.AnError))
	WarnCtx(ctx, "test warning message", Int("number", 42))
	DebugCtx(ctx, "test debug message", Bool("flag", true))
}

func TestNonContextLoggingFunctions(t *testing.T) {
	// Setup a test logger
	testLogger := slog.New(slog.NewTextHandler(&testWriter{}, &slog.HandlerOptions{Level: slog.LevelDebug}))
	SetGlobalLogger(testLogger)

	// These should not panic
	Info("test info message", String("key", "value"))
	Error("test error message", Err(assert.AnError))
	Warn("test warning message", Int("number", 42))
	Debug("test debug message", Bool("flag", true))
}

func TestNewSlogLogger(t *testing.T) {
	tests := []struct {
		name        string
		config      SlogConfig
		expectJSON  bool
		expectNR    bool
	}{
		{
			name: "JSON format with New Relic",
			config: SlogConfig{
				Level:       slog.LevelInfo,
				ServiceName: "test-service",
				NewRelic:    &newrelic.Application{},
				Format:      "json",
			},
			expectJSON: true,
			expectNR:   true,
		},
		{
			name: "Text format without New Relic",
			config: SlogConfig{
				Level:       slog.LevelInfo,
				ServiceName: "test-service",
				NewRelic:    nil,
				Format:      "text",
			},
			expectJSON: false,
			expectNR:   false,
		},
		{
			name: "Default format with empty service name",
			config: SlogConfig{
				Level:       slog.LevelDebug,
				ServiceName: "",
				NewRelic:    nil,
				Format:      "unknown",
			},
			expectJSON: false,
			expectNR:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewSlogLogger(tt.config)
			assert.NotNil(t, logger)

			// Test that logger is created without actually logging
			assert.NotNil(t, logger)
		})
	}
}

func TestNewSlogLogger_WithServiceName(t *testing.T) {
	config := SlogConfig{
		Level:       slog.LevelInfo,
		ServiceName: "test-service",
		NewRelic:    nil,
		Format:      "json",
	}

	logger := NewSlogLogger(config)
	assert.NotNil(t, logger)

	// Test that logger is created with service name
	assert.NotNil(t, logger)
}

func TestNewRelicLogForwarder(t *testing.T) {
	t.Run("With New Relic app", func(t *testing.T) {
		baseHandler := slog.NewTextHandler(&testWriter{}, &slog.HandlerOptions{})
		forwarder := &NewRelicLogForwarder{
			handler: baseHandler,
			app:     &newrelic.Application{},
		}

		assert.True(t, forwarder.Enabled(context.Background(), slog.LevelInfo))
		assert.True(t, forwarder.Enabled(context.Background(), slog.LevelError))
	})

	t.Run("Without New Relic app", func(t *testing.T) {
		baseHandler := slog.NewTextHandler(&testWriter{}, &slog.HandlerOptions{})
		forwarder := &NewRelicLogForwarder{
			handler: baseHandler,
			app:     nil,
		}

		assert.True(t, forwarder.Enabled(context.Background(), slog.LevelInfo))
	})
}

func TestNewRelicLogForwarder_WithAttrs(t *testing.T) {
	baseHandler := slog.NewTextHandler(&testWriter{}, &slog.HandlerOptions{})
	forwarder := &NewRelicLogForwarder{
		handler: baseHandler,
		app:     &newrelic.Application{},
	}

	attrs := []slog.Attr{
		slog.String("key1", "value1"),
		slog.Int("key2", 42),
	}

	newForwarder := forwarder.WithAttrs(attrs)
	assert.NotNil(t, newForwarder)
	assert.IsType(t, &NewRelicLogForwarder{}, newForwarder)
}

func TestNewRelicLogForwarder_WithGroup(t *testing.T) {
	baseHandler := slog.NewTextHandler(&testWriter{}, &slog.HandlerOptions{})
	forwarder := &NewRelicLogForwarder{
		handler: baseHandler,
		app:     &newrelic.Application{},
	}

	newForwarder := forwarder.WithGroup("test-group")
	assert.NotNil(t, newForwarder)
	assert.IsType(t, &NewRelicLogForwarder{}, newForwarder)
}

func TestContextLogger(t *testing.T) {
	t.Run("NewContextLogger", func(t *testing.T) {
		baseLogger := slog.New(slog.NewTextHandler(&testWriter{}, &slog.HandlerOptions{}))
		contextLogger := NewContextLogger(baseLogger)

		assert.NotNil(t, contextLogger)
		assert.Equal(t, baseLogger, contextLogger.logger)
	})

	t.Run("WithContext with no context values", func(t *testing.T) {
		baseLogger := slog.New(slog.NewTextHandler(&testWriter{}, &slog.HandlerOptions{}))
		contextLogger := NewContextLogger(baseLogger)

		ctx := context.Background()
		logger := contextLogger.WithContext(ctx)

		assert.NotNil(t, logger)
		assert.Equal(t, baseLogger, logger)
	})

	t.Run("WithContext with context values", func(t *testing.T) {
		baseLogger := slog.New(slog.NewTextHandler(&testWriter{}, &slog.HandlerOptions{}))
		contextLogger := NewContextLogger(baseLogger)

		ctx := context.WithValue(context.Background(), "request_id", "test-request-id")
		ctx = context.WithValue(ctx, "user_id", "test-user-id")
		ctx = context.WithValue(ctx, "service_name", "test-service")
		ctx = context.WithValue(ctx, "trace_id", "test-trace-id")

		logger := contextLogger.WithContext(ctx)

		assert.NotNil(t, logger)
		// Note: The actual attributes are added to the logger, but we can't easily test them
		// without capturing the log output
	})

	t.Run("WithContext with some context values", func(t *testing.T) {
		baseLogger := slog.New(slog.NewTextHandler(&testWriter{}, &slog.HandlerOptions{}))
		contextLogger := NewContextLogger(baseLogger)

		ctx := context.WithValue(context.Background(), "request_id", "test-request-id")
		ctx = context.WithValue(ctx, "user_id", "test-user-id")

		logger := contextLogger.WithContext(ctx)

		assert.NotNil(t, logger)
	})

	t.Run("Logging methods", func(t *testing.T) {
		baseLogger := slog.New(slog.NewTextHandler(&testWriter{}, &slog.HandlerOptions{}))
		contextLogger := NewContextLogger(baseLogger)

		ctx := context.Background()

		// These should not panic
		contextLogger.Info(ctx, "test info message", String("key", "value"))
		contextLogger.Error(ctx, "test error message", Err(assert.AnError))
		contextLogger.Warn(ctx, "test warning message", Int("number", 42))
		contextLogger.Debug(ctx, "test debug message", Bool("flag", true))
	})
}

func TestContextLogger_InvalidContextValues(t *testing.T) {
	baseLogger := slog.New(slog.NewTextHandler(&testWriter{}, &slog.HandlerOptions{}))
	contextLogger := NewContextLogger(baseLogger)

	// Test with invalid context values (wrong types)
	ctx := context.WithValue(context.Background(), "request_id", 123) // Should be string
	ctx = context.WithValue(ctx, "user_id", true)                   // Should be string

	logger := contextLogger.WithContext(ctx)
	assert.NotNil(t, logger)

	// Should not panic when logging
	contextLogger.Info(ctx, "test message")
}

func TestFieldTypes(t *testing.T) {
	// Test that all field functions return valid slog.Attr
	assert.IsType(t, slog.Attr{}, String("key", "value"))
	assert.IsType(t, slog.Attr{}, Err(assert.AnError))
	assert.IsType(t, slog.Attr{}, Int("key", 42))
	assert.IsType(t, slog.Attr{}, Int64("key", int64(42)))
	assert.IsType(t, slog.Attr{}, Uint32("key", uint32(42)))
	assert.IsType(t, slog.Attr{}, Float64("key", 3.14))
	assert.IsType(t, slog.Attr{}, Bool("key", true))
	assert.IsType(t, slog.Attr{}, Any("key", "any value"))
	assert.IsType(t, slog.Attr{}, Duration("key", time.Second))
	assert.IsType(t, slog.Attr{}, ErrorField(assert.AnError))
	assert.IsType(t, slog.Attr{}, Strings("key", []string{"a", "b", "c"}))
}

func TestLoggingWithVariousFieldTypes(t *testing.T) {
	// Setup a test logger
	testLogger := slog.New(slog.NewTextHandler(&testWriter{}, &slog.HandlerOptions{Level: slog.LevelDebug}))
	SetGlobalLogger(testLogger)

	ctx := context.Background()

	// Test logging with various field types
	InfoCtx(ctx, "complex message",
		String("string", "value"),
		Int("int", 42),
		Int64("int64", int64(9223372036854775807)),
		Uint32("uint32", uint32(4294967295)),
		Float64("float64", 3.14159),
		Bool("bool", true),
		Any("any", map[string]interface{}{"nested": "value"}),
		Duration("duration", 5*time.Second),
		ErrorField(assert.AnError),
		Strings("strings", []string{"a", "b", "c"}),
	)
}

func TestConcurrentLogging(t *testing.T) {
	// Setup a test logger
	testLogger := slog.New(slog.NewTextHandler(&testWriter{}, &slog.HandlerOptions{Level: slog.LevelDebug}))
	SetGlobalLogger(testLogger)

	ctx := context.Background()
	done := make(chan bool, 10)

	// Test concurrent logging
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			InfoCtx(ctx, "concurrent message", Int("id", id))
			ErrorCtx(ctx, "concurrent error", Err(assert.AnError), Int("id", id))
			WarnCtx(ctx, "concurrent warning", String("id", string(rune('A'+id))))
			DebugCtx(ctx, "concurrent debug", Float64("id", float64(id)))
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestLoggerWithNilGlobalLogger(t *testing.T) {
	// Reset global logger to nil
	globalLogger = nil

	// Test that GetGlobalLogger returns a default logger
	logger := GetGlobalLogger()
	assert.NotNil(t, logger)
	assert.Equal(t, slog.Default(), logger)

	// Test that logging functions work with nil global logger
	ctx := context.Background()
	InfoCtx(ctx, "test message", String("key", "value"))
}

func TestNewRelicLogForwarder_Handle(t *testing.T) {
	t.Run("Error level with New Relic app", func(t *testing.T) {
		baseHandler := slog.NewTextHandler(&testWriter{}, &slog.HandlerOptions{})
		forwarder := &NewRelicLogForwarder{
			handler: baseHandler,
			app:     &newrelic.Application{},
		}

		record := slog.Record{
			Level:   slog.LevelError,
			Message: "test error message",
			Time:    time.Now(),
		}

		// Should not panic
		err := forwarder.Handle(context.Background(), record)
		assert.NoError(t, err)
	})

	t.Run("Info level with New Relic app", func(t *testing.T) {
		baseHandler := slog.NewTextHandler(&testWriter{}, &slog.HandlerOptions{})
		forwarder := &NewRelicLogForwarder{
			handler: baseHandler,
			app:     &newrelic.Application{},
		}

		record := slog.Record{
			Level:   slog.LevelInfo,
			Message: "test info message",
			Time:    time.Now(),
		}

		// Should not panic, but should not forward to New Relic
		err := forwarder.Handle(context.Background(), record)
		assert.NoError(t, err)
	})

	t.Run("Error level without New Relic app", func(t *testing.T) {
		baseHandler := slog.NewTextHandler(&testWriter{}, &slog.HandlerOptions{})
		forwarder := &NewRelicLogForwarder{
			handler: baseHandler,
			app:     nil,
		}

		record := slog.Record{
			Level:   slog.LevelError,
			Message: "test error message",
			Time:    time.Now(),
		}

		// Should not panic, but should not forward to New Relic
		err := forwarder.Handle(context.Background(), record)
		assert.NoError(t, err)
	})
}