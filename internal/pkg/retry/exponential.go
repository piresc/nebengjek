package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/logger"
)

// RetryableFunc represents a function that can be retried
type RetryableFunc func(ctx context.Context) error

// Config holds retry configuration
type Config struct {
	MaxRetries    int              // Maximum number of retry attempts
	BaseDelay     time.Duration    // Base delay between retries
	MaxDelay      time.Duration    // Maximum delay between retries
	Multiplier    float64          // Exponential backoff multiplier
	Jitter        bool             // Add randomization to prevent thundering herd
	RetryableFunc func(error) bool // Function to determine if error is retryable
}

// DefaultConfig returns a default retry configuration
func DefaultConfig() Config {
	return Config{
		MaxRetries: 3,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
		Jitter:     true,
		RetryableFunc: func(err error) bool {
			// By default, retry all errors
			return true
		},
	}
}

// Retrier handles retry logic with exponential backoff
type Retrier struct {
	config Config
	logger *logger.ZapLogger
}

// New creates a new retrier with the given configuration
func New(config Config, l *logger.ZapLogger) *Retrier {
	return &Retrier{
		config: config,
		logger: l,
	}
}

// NewWithDefaults creates a new retrier with default configuration
func NewWithDefaults(l *logger.ZapLogger) *Retrier {
	return New(DefaultConfig(), l)
}

// Execute executes the function with retry logic
func (r *Retrier) Execute(ctx context.Context, fn RetryableFunc) error {
	var lastErr error

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute the function
		err := fn(ctx)
		if err == nil { // Success
			if attempt > 0 {
				r.logger.Info("Function succeeded after retries",
					logger.Int("attempt", attempt+1),
					logger.Int("total_attempts", attempt+1))
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !r.config.RetryableFunc(err) {
			r.logger.Debug("Error is not retryable, stopping",
				logger.Err(err),
				logger.Int("attempt", attempt+1))
			return err
		}

		// Don't sleep after the last attempt
		if attempt == r.config.MaxRetries {
			break
		}

		// Calculate delay with exponential backoff
		delay := r.calculateDelay(attempt)

		r.logger.Debug("Function failed, retrying",
			logger.Err(err),
			logger.Int("attempt", attempt+1),
			logger.Duration("delay", delay),
			logger.Int("max_retries", r.config.MaxRetries))

		// Wait for the calculated delay
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	r.logger.Error("Function failed after all retries",
		logger.Err(lastErr),
		logger.Int("total_attempts", r.config.MaxRetries+1))

	return fmt.Errorf("retry limit exceeded after %d attempts: %w", r.config.MaxRetries+1, lastErr)
}

// calculateDelay calculates the delay for the given attempt number
func (r *Retrier) calculateDelay(attempt int) time.Duration {
	// Calculate exponential backoff delay
	delay := float64(r.config.BaseDelay) * math.Pow(r.config.Multiplier, float64(attempt))

	// Apply maximum delay limit
	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}

	// Add jitter if enabled
	if r.config.Jitter {
		// Add random jitter up to 10% of the delay
		jitter := delay * 0.1 * rand.Float64()
		delay += jitter
	}

	return time.Duration(delay)
}

// ExecuteWithMetrics executes the function with retry logic and returns metrics
func (r *Retrier) ExecuteWithMetrics(ctx context.Context, fn RetryableFunc) (error, RetryMetrics) {
	metrics := RetryMetrics{
		StartTime: time.Now(),
	}

	var lastErr error

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		metrics.Attempts++

		// Check if context is cancelled
		select {
		case <-ctx.Done():
			metrics.EndTime = time.Now()
			metrics.Success = false
			return ctx.Err(), metrics
		default:
		}

		attemptStart := time.Now()
		err := fn(ctx)
		attemptDuration := time.Since(attemptStart)
		metrics.AttemptDurations = append(metrics.AttemptDurations, attemptDuration)

		if err == nil {
			// Success
			metrics.EndTime = time.Now()
			metrics.Success = true
			if attempt > 0 {
				r.logger.Info("Function succeeded after retries",
					logger.Int("attempt", attempt+1),
					logger.Int("total_attempts", attempt+1),
					logger.Duration("total_duration", metrics.EndTime.Sub(metrics.StartTime)))
			}
			return nil, metrics
		}

		lastErr = err
		metrics.Errors = append(metrics.Errors, err.Error())

		// Check if error is retryable
		if !r.config.RetryableFunc(err) {
			metrics.EndTime = time.Now()
			metrics.Success = false
			return err, metrics
		}

		// Don't sleep after the last attempt
		if attempt == r.config.MaxRetries {
			break
		}

		// Calculate delay with exponential backoff
		delay := r.calculateDelay(attempt)
		metrics.Delays = append(metrics.Delays, delay)

		// Wait for the calculated delay
		select {
		case <-ctx.Done():
			metrics.EndTime = time.Now()
			metrics.Success = false
			return ctx.Err(), metrics
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	metrics.EndTime = time.Now()
	metrics.Success = false

	finalErr := fmt.Errorf("retry limit exceeded after %d attempts: %w", r.config.MaxRetries+1, lastErr)
	return finalErr, metrics
}

// RetryMetrics holds metrics about retry execution
type RetryMetrics struct {
	StartTime        time.Time
	EndTime          time.Time
	Attempts         int
	Success          bool
	Errors           []string
	Delays           []time.Duration
	AttemptDurations []time.Duration
}

// TotalDuration returns the total duration of all retry attempts
func (m RetryMetrics) TotalDuration() time.Duration {
	return m.EndTime.Sub(m.StartTime)
}

// NetworkRetryableFunc returns a function that determines if network errors are retryable
func NetworkRetryableFunc() func(error) bool {
	return func(err error) bool {
		if err == nil {
			return false
		}

		errStr := err.Error()

		// Common retryable network errors
		retryableErrors := []string{
			"connection refused",
			"connection reset",
			"connection timeout",
			"timeout",
			"temporary failure",
			"service unavailable",
			"bad gateway",
			"gateway timeout",
			"internal server error",
		}

		for _, retryableErr := range retryableErrors {
			if fmt.Sprintf("%v", err) != "" &&
				(fmt.Sprintf("%v", err) == retryableErr ||
					fmt.Sprintf("%v", errStr) == retryableErr) {
				return true
			}
		}

		return false
	}
}
