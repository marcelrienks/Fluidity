package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrCircuitOpen     = errors.New("circuit breaker is open")
	ErrTooManyRequests = errors.New("too many requests")
)

// State represents the circuit breaker state
type State int

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

// String returns the string representation of the state
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half-open"
	case StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	mu              sync.RWMutex
	maxFailures     int
	resetTimeout    time.Duration
	halfOpenTimeout time.Duration
	state           State
	failures        int
	lastFailureTime time.Time
	lastStateChange time.Time
	successCount    int
	maxHalfOpenReqs int
}

// Config holds circuit breaker configuration
type Config struct {
	MaxFailures     int           // Number of failures before opening circuit
	ResetTimeout    time.Duration // Time to wait before attempting to close circuit
	HalfOpenTimeout time.Duration // Time to wait in half-open state before returning to closed
	MaxHalfOpenReqs int           // Max successful requests in half-open before closing
}

// DefaultConfig returns default circuit breaker configuration
func DefaultConfig() Config {
	return Config{
		MaxFailures:     5,
		ResetTimeout:    30 * time.Second,
		HalfOpenTimeout: 10 * time.Second,
		MaxHalfOpenReqs: 3,
	}
}

// New creates a new circuit breaker with the given configuration
func New(config Config) *CircuitBreaker {
	if config.MaxFailures <= 0 {
		config.MaxFailures = 5
	}
	if config.ResetTimeout <= 0 {
		config.ResetTimeout = 30 * time.Second
	}
	if config.HalfOpenTimeout <= 0 {
		config.HalfOpenTimeout = 10 * time.Second
	}
	if config.MaxHalfOpenReqs <= 0 {
		config.MaxHalfOpenReqs = 3
	}

	return &CircuitBreaker{
		maxFailures:     config.MaxFailures,
		resetTimeout:    config.ResetTimeout,
		halfOpenTimeout: config.HalfOpenTimeout,
		maxHalfOpenReqs: config.MaxHalfOpenReqs,
		state:           StateClosed,
		lastStateChange: time.Now(),
	}
}

// Execute runs the given function if the circuit is closed or half-open
func (cb *CircuitBreaker) Execute(fn func() error) error {
	// Check current state
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	// Execute the function
	err := fn()

	// Update state based on result
	cb.afterRequest(err)

	return err
}

// beforeRequest checks if the request can be executed
func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return nil

	case StateOpen:
		// Check if we should transition to half-open
		if now.Sub(cb.lastStateChange) >= cb.resetTimeout {
			cb.state = StateHalfOpen
			cb.successCount = 0
			cb.lastStateChange = now
			return nil
		}
		return ErrCircuitOpen

	case StateHalfOpen:
		// Allow limited requests in half-open state
		if cb.successCount >= cb.maxHalfOpenReqs {
			return ErrTooManyRequests
		}
		return nil

	default:
		return ErrCircuitOpen
	}
}

// afterRequest updates the circuit breaker state after a request
func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	if err != nil {
		// Request failed
		cb.failures++
		cb.lastFailureTime = now

		// Transition to open if failure threshold exceeded
		if cb.state == StateClosed && cb.failures >= cb.maxFailures {
			cb.state = StateOpen
			cb.lastStateChange = now
		} else if cb.state == StateHalfOpen {
			// Single failure in half-open state reopens the circuit
			cb.state = StateOpen
			cb.lastStateChange = now
			cb.successCount = 0
		}
	} else {
		// Request succeeded
		if cb.state == StateHalfOpen {
			cb.successCount++
			// Transition to closed if enough successes in half-open
			if cb.successCount >= cb.maxHalfOpenReqs {
				cb.state = StateClosed
				cb.failures = 0
				cb.successCount = 0
				cb.lastStateChange = now
			}
		} else if cb.state == StateClosed {
			// Reset failure count on success in closed state
			cb.failures = 0
		}
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetFailures returns the current failure count
func (cb *CircuitBreaker) GetFailures() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failures
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failures = 0
	cb.successCount = 0
	cb.lastStateChange = time.Now()
}
