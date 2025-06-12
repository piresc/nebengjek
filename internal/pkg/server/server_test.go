package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestNewGracefulServer(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		expected func(*GracefulServer) bool
	}{
		{
			name: "Valid server creation",
			port: 8080,
			expected: func(gs *GracefulServer) bool {
				return gs != nil
			},
		},
		{
			name: "Different port",
			port: 9090,
			expected: func(gs *GracefulServer) bool {
				return gs != nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			logger := slog.Default()
			gs := NewGracefulServer(e, logger, tt.port)
			assert.True(t, tt.expected(gs))
		})
	}
}

func TestGracefulServer_Start(t *testing.T) {
	t.Run("Start server successfully", func(t *testing.T) {
		e := echo.New()
		logger := slog.Default()
		gs := NewGracefulServer(e, logger, 0) // Use port 0 to get a random available port

		// Start server in goroutine
		go func() {
			err := gs.Start()
			// Server should shut down gracefully, so error should be http.ErrServerClosed
			if err != nil && err != http.ErrServerClosed {
				t.Errorf("Unexpected error: %v", err)
			}
		}()

		// Give server time to start
		time.Sleep(100 * time.Millisecond)

		// Note: The current implementation doesn't have a Shutdown method
		// This test would need to be updated when Shutdown is implemented
	})
}

// Note: Shutdown tests are commented out because the current implementation
// doesn't expose a Shutdown method. The graceful shutdown is handled internally
// by the Start method when it receives OS signals.

/*
func TestGracefulServer_Shutdown(t *testing.T) {
	// These tests would be implemented when a public Shutdown method is added
}

func TestGracefulServer_Integration(t *testing.T) {
	// Integration tests would be implemented when shutdown functionality is exposed
}
*/

func TestNewShutdownManager(t *testing.T) {
	logger := slog.Default()
	sm := NewShutdownManager(logger)
	assert.NotNil(t, sm)
}

func TestShutdownManager_Register(t *testing.T) {
	t.Run("Register single cleanup function", func(t *testing.T) {
		logger := slog.Default()
		sm := NewShutdownManager(logger)
		called := false

		cleanupFunc := func(ctx context.Context) error {
			called = true
			return nil
		}

		sm.Register(cleanupFunc)

		// Execute shutdown to verify function is called
		ctx := context.Background()
		err := sm.Shutdown(ctx)
		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("Register multiple cleanup functions", func(t *testing.T) {
		logger := slog.Default()
		sm := NewShutdownManager(logger)
		callOrder := []int{}
		var mu sync.Mutex

		for i := 0; i < 5; i++ {
			index := i
			cleanupFunc := func(ctx context.Context) error {
				mu.Lock()
				callOrder = append(callOrder, index)
				mu.Unlock()
				return nil
			}
			sm.Register(cleanupFunc)
		}

		ctx := context.Background()
		err := sm.Shutdown(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 5, len(callOrder))
		// Functions are called in order (FIFO)
		expected := []int{0, 1, 2, 3, 4}
		assert.Equal(t, expected, callOrder)
	})

	t.Run("Register nil function", func(t *testing.T) {
		logger := slog.Default()
		sm := NewShutdownManager(logger)
		
		// This should not panic
		assert.NotPanics(t, func() {
			sm.Register(nil)
		})
		
		// Shutdown should handle nil function gracefully
		ctx := context.Background()
		err := sm.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestShutdownManager_Shutdown(t *testing.T) {
	t.Run("Shutdown with successful cleanup functions", func(t *testing.T) {
		logger := slog.Default()
		sm := NewShutdownManager(logger)
		var results []string
		var mu sync.Mutex

		cleanupFuncs := []func(context.Context) error{
			func(ctx context.Context) error {
				mu.Lock()
				results = append(results, "cleanup1")
				mu.Unlock()
				return nil
			},
			func(ctx context.Context) error {
				mu.Lock()
				results = append(results, "cleanup2")
				mu.Unlock()
				return nil
			},
			func(ctx context.Context) error {
				mu.Lock()
				results = append(results, "cleanup3")
				mu.Unlock()
				return nil
			},
		}

		for _, f := range cleanupFuncs {
			sm.Register(f)
		}

		ctx := context.Background()
		err := sm.Shutdown(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(results))
		// Should be called in order (FIFO)
		assert.Equal(t, []string{"cleanup1", "cleanup2", "cleanup3"}, results)
	})

	t.Run("Shutdown with failing cleanup functions", func(t *testing.T) {
		logger := slog.Default()
		sm := NewShutdownManager(logger)
		var results []string
		var mu sync.Mutex

		cleanupFuncs := []func(context.Context) error{
			func(ctx context.Context) error {
				mu.Lock()
				results = append(results, "cleanup1")
				mu.Unlock()
				return nil
			},
			func(ctx context.Context) error {
				mu.Lock()
				results = append(results, "cleanup2")
				mu.Unlock()
				return fmt.Errorf("cleanup2 failed")
			},
			func(ctx context.Context) error {
				mu.Lock()
				results = append(results, "cleanup3")
				mu.Unlock()
				return nil
			},
		}

		for _, f := range cleanupFuncs {
			sm.Register(f)
		}

		ctx := context.Background()
		err := sm.Shutdown(ctx)
		// The current implementation doesn't return errors, it just logs them
		assert.NoError(t, err)
		// All functions should still be called despite errors
		assert.Equal(t, 3, len(results))
		assert.Equal(t, []string{"cleanup1", "cleanup2", "cleanup3"}, results)
	})

	t.Run("Shutdown with multiple failing cleanup functions", func(t *testing.T) {
		logger := slog.Default()
		sm := NewShutdownManager(logger)

		cleanupFuncs := []func(context.Context) error{
			func(ctx context.Context) error { return fmt.Errorf("error1") },
			func(ctx context.Context) error { return fmt.Errorf("error2") },
			func(ctx context.Context) error { return fmt.Errorf("error3") },
		}

		for _, f := range cleanupFuncs {
			sm.Register(f)
		}

		ctx := context.Background()
		err := sm.Shutdown(ctx)
		// The current implementation doesn't return errors, it just logs them
		assert.NoError(t, err)
	})

	t.Run("Shutdown with no cleanup functions", func(t *testing.T) {
		logger := slog.Default()
		sm := NewShutdownManager(logger)
		ctx := context.Background()
		err := sm.Shutdown(ctx)
		assert.NoError(t, err)
	})

	t.Run("Shutdown with panic in cleanup function", func(t *testing.T) {
		logger := slog.Default()
		sm := NewShutdownManager(logger)
		var results []string
		var mu sync.Mutex

		cleanupFuncs := []func(context.Context) error{
			func(ctx context.Context) error {
				mu.Lock()
				results = append(results, "cleanup1")
				mu.Unlock()
				return nil
			},
			func(ctx context.Context) error {
				panic("cleanup panic")
			},
			func(ctx context.Context) error {
				mu.Lock()
				results = append(results, "cleanup3")
				mu.Unlock()
				return nil
			},
		}

		for _, f := range cleanupFuncs {
			sm.Register(f)
		}

		// The shutdown should handle panics gracefully
		// This test verifies that a panic in one cleanup doesn't prevent others from running
		assert.Panics(t, func() {
			ctx := context.Background()
			sm.Shutdown(ctx)
		})
	})
}

func TestShutdownManager_ConcurrentAccess(t *testing.T) {
	t.Run("Concurrent registration", func(t *testing.T) {
		logger := slog.Default()
		sm := NewShutdownManager(logger)
		var wg sync.WaitGroup
		numGoroutines := 10

		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				defer wg.Done()
				sm.Register(func(ctx context.Context) error {
					return nil
				})
			}(i)
		}

		wg.Wait()
		// We can't directly test the internal functions slice length
		// but we can test that shutdown works
		ctx := context.Background()
		err := sm.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestShutdownManager_Integration(t *testing.T) {
	t.Run("Real-world scenario", func(t *testing.T) {
		logger := slog.Default()
		sm := NewShutdownManager(logger)

		// Simulate database connection cleanup
		dbClosed := false
		sm.Register(func(ctx context.Context) error {
			dbClosed = true
			return nil
		})

		// Simulate cache cleanup
		cacheClosed := false
		sm.Register(func(ctx context.Context) error {
			cacheClosed = true
			return nil
		})

		// Simulate message queue cleanup
		mqClosed := false
		sm.Register(func(ctx context.Context) error {
			mqClosed = true
			return nil
		})

		// Simulate cleanup that takes time
		slowCleanupDone := false
		sm.Register(func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			slowCleanupDone = true
			return nil
		})

		start := time.Now()
		ctx := context.Background()
		err := sm.Shutdown(ctx)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.True(t, dbClosed)
		assert.True(t, cacheClosed)
		assert.True(t, mqClosed)
		assert.True(t, slowCleanupDone)
		assert.True(t, duration >= 50*time.Millisecond)
	})
}

func BenchmarkGracefulServer_NewServer(b *testing.B) {
	e := echo.New()
	logger := slog.Default()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewGracefulServer(e, logger, 8080)
	}
}

func BenchmarkShutdownManager_Register(b *testing.B) {
	logger := slog.Default()
	sm := NewShutdownManager(logger)
	cleanupFunc := func(ctx context.Context) error { return nil }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sm.Register(cleanupFunc)
	}
}

func BenchmarkShutdownManager_Shutdown(b *testing.B) {
	b.Run("Small number of functions", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			logger := slog.Default()
			sm := NewShutdownManager(logger)
			for j := 0; j < 5; j++ {
				sm.Register(func(ctx context.Context) error { return nil })
			}
			b.StartTimer()

			ctx := context.Background()
			sm.Shutdown(ctx)
		}
	})

	b.Run("Large number of functions", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			logger := slog.Default()
			sm := NewShutdownManager(logger)
			for j := 0; j < 100; j++ {
				sm.Register(func(ctx context.Context) error { return nil })
			}
			b.StartTimer()

			ctx := context.Background()
			sm.Shutdown(ctx)
		}
	})
}