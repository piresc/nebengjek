package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/piresc/nebengjek/internal/pkg/logger"
)

// State represents the circuit breaker state
type State int

const (
	// StateClosed allows requests to pass through
	StateClosed State = iota
	// StateOpen blocks requests and returns immediately
	StateOpen
	// StateHalfOpen allows a limited number of requests to test the service
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// Config holds circuit breaker configuration
type Config struct {
	Name             string                                  // Name of the circuit breaker for logging
	MaxRequests      uint32                                  // Max requests allowed in half-open state
	Interval         time.Duration                           // Interval to clear counters in closed state
	Timeout          time.Duration                           // Timeout to switch from open to half-open
	FailureThreshold uint32                                  // Number of failures to trigger open state
	SuccessThreshold uint32                                  // Number of successes in half-open to close
	OnStateChange    func(name string, from State, to State) // State change callback
	IsFailure        func(err error) bool                    // Function to determine if error should count as failure
}

// DefaultConfig returns a default circuit breaker configuration
func DefaultConfig(name string) Config {
	return Config{
		Name:             name,
		MaxRequests:      1,
		Interval:         30 * time.Second,
		Timeout:          60 * time.Second,
		FailureThreshold: 5,
		SuccessThreshold: 1,
		IsFailure: func(err error) bool {
			return err != nil
		},
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config Config
	logger *logger.ZapLogger

	mutex  sync.RWMutex
	state  State
	counts Counts
	expiry time.Time
}

// Counts holds the counters for circuit breaker
type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

// New creates a new circuit breaker
func New(config Config, l *logger.ZapLogger) *CircuitBreaker {
	cb := &CircuitBreaker{
		config: config,
		logger: l,
		state:  StateClosed,
		expiry: time.Now().Add(config.Interval),
	}

	return cb
}

// Execute executes the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	// Check if we can execute
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	// Execute the function
	err := fn(ctx)

	// Handle the result
	cb.afterRequest(err)

	return err
}

// beforeRequest checks if the request should be allowed
func (cb *CircuitBreaker) beforeRequest() error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		if cb.expiry.Before(now) {
			cb.resetCounts()
			cb.expiry = now.Add(cb.config.Interval)
		}

	case StateOpen:
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen)
			cb.resetCounts()
		} else {
			return ErrCircuitBreakerOpen
		}

	case StateHalfOpen:
		if cb.counts.Requests >= cb.config.MaxRequests {
			return ErrTooManyRequests
		}
	}

	cb.counts.Requests++
	return nil
}

// afterRequest handles the result of the request
func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if cb.config.IsFailure(err) {
		cb.counts.TotalFailures++
		cb.counts.ConsecutiveFailures++
		cb.counts.ConsecutiveSuccesses = 0

		if cb.state == StateClosed && cb.counts.ConsecutiveFailures >= cb.config.FailureThreshold {
			cb.setState(StateOpen)
			cb.expiry = time.Now().Add(cb.config.Timeout)
		} else if cb.state == StateHalfOpen {
			cb.setState(StateOpen)
			cb.expiry = time.Now().Add(cb.config.Timeout)
		}
	} else {
		cb.counts.TotalSuccesses++
		cb.counts.ConsecutiveSuccesses++
		cb.counts.ConsecutiveFailures = 0

		if cb.state == StateHalfOpen && cb.counts.ConsecutiveSuccesses >= cb.config.SuccessThreshold {
			cb.setState(StateClosed)
			cb.expiry = time.Now().Add(cb.config.Interval)
		}
	}
}

// setState changes the state and triggers callbacks
func (cb *CircuitBreaker) setState(state State) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	cb.logger.Info("Circuit breaker state changed",
		logger.String("name", cb.config.Name),
		logger.String("from", prev.String()),
		logger.String("to", state.String()),
		logger.Uint32("total_requests", cb.counts.Requests),
		logger.Uint32("total_failures", cb.counts.TotalFailures),
		logger.Uint32("consecutive_failures", cb.counts.ConsecutiveFailures))

	if cb.config.OnStateChange != nil {
		cb.config.OnStateChange(cb.config.Name, prev, state)
	}
}

// resetCounts resets all counters
func (cb *CircuitBreaker) resetCounts() {
	cb.counts = Counts{}
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() State {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// Counts returns the current counts
func (cb *CircuitBreaker) Counts() Counts {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.counts
}

// Name returns the circuit breaker name
func (cb *CircuitBreaker) Name() string {
	return cb.config.Name
}

// Errors
var (
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
	ErrTooManyRequests    = errors.New("too many requests in half-open state")
)

// Manager manages multiple circuit breakers
type Manager struct {
	breakers map[string]*CircuitBreaker
	mutex    sync.RWMutex
	logger   *logger.ZapLogger
}

// NewManager creates a new circuit breaker manager
func NewManager(l *logger.ZapLogger) *Manager {
	return &Manager{
		breakers: make(map[string]*CircuitBreaker),
		logger:   l,
	}
}

// GetOrCreate gets an existing circuit breaker or creates a new one
func (m *Manager) GetOrCreate(name string, config Config) *CircuitBreaker {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if cb, exists := m.breakers[name]; exists {
		return cb
	}

	config.Name = name
	cb := New(config, m.logger)
	m.breakers[name] = cb

	m.logger.Info("Created new circuit breaker",
		logger.String("name", name),
		logger.Uint32("failure_threshold", config.FailureThreshold),
		logger.Duration("timeout", config.Timeout))

	return cb
}

// Get retrieves a circuit breaker by name
func (m *Manager) Get(name string) (*CircuitBreaker, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	cb, exists := m.breakers[name]
	return cb, exists
}

// Execute executes a function with the named circuit breaker
func (m *Manager) Execute(ctx context.Context, name string, fn func(context.Context) error) error {
	config := DefaultConfig(name)
	cb := m.GetOrCreate(name, config)
	return cb.Execute(ctx, fn)
}

// ExecuteWithConfig executes a function with a custom circuit breaker configuration
func (m *Manager) ExecuteWithConfig(ctx context.Context, name string, config Config, fn func(context.Context) error) error {
	cb := m.GetOrCreate(name, config)
	return cb.Execute(ctx, fn)
}

// GetStats returns statistics for all circuit breakers
func (m *Manager) GetStats() map[string]CircuitBreakerStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := make(map[string]CircuitBreakerStats)
	for name, cb := range m.breakers {
		counts := cb.Counts()
		stats[name] = CircuitBreakerStats{
			Name:                 name,
			State:                cb.State().String(),
			TotalRequests:        counts.Requests,
			TotalSuccesses:       counts.TotalSuccesses,
			TotalFailures:        counts.TotalFailures,
			ConsecutiveSuccesses: counts.ConsecutiveSuccesses,
			ConsecutiveFailures:  counts.ConsecutiveFailures,
		}
	}

	return stats
}

// CircuitBreakerStats holds statistics for a circuit breaker
type CircuitBreakerStats struct {
	Name                 string `json:"name"`
	State                string `json:"state"`
	TotalRequests        uint32 `json:"total_requests"`
	TotalSuccesses       uint32 `json:"total_successes"`
	TotalFailures        uint32 `json:"total_failures"`
	ConsecutiveSuccesses uint32 `json:"consecutive_successes"`
	ConsecutiveFailures  uint32 `json:"consecutive_failures"`
}
