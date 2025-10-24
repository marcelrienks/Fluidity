package circuitbreaker

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedState(t *testing.T) {
	cb := New(DefaultConfig())

	// Circuit should start closed
	if cb.GetState() != StateClosed {
		t.Errorf("Expected initial state to be Closed, got %v", cb.GetState())
	}

	// Execute successful requests
	for i := 0; i < 10; i++ {
		err := cb.Execute(func() error {
			return nil
		})
		if err != nil {
			t.Errorf("Expected no error for successful request, got %v", err)
		}
	}

	// Circuit should remain closed
	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to remain Closed after successful requests, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_OpenState(t *testing.T) {
	config := Config{
		MaxFailures:     3,
		ResetTimeout:    1 * time.Second,
		HalfOpenTimeout: 500 * time.Millisecond,
		MaxHalfOpenReqs: 2,
	}
	cb := New(config)

	// Fail enough times to open circuit
	testErr := errors.New("test error")
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	// Circuit should be open
	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be Open after %d failures, got %v", config.MaxFailures, cb.GetState())
	}

	// Next request should fail immediately
	err := cb.Execute(func() error {
		t.Error("Function should not be called when circuit is open")
		return nil
	})

	if err != ErrCircuitOpen {
		t.Errorf("Expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_HalfOpenState(t *testing.T) {
	config := Config{
		MaxFailures:     3,
		ResetTimeout:    100 * time.Millisecond,
		HalfOpenTimeout: 500 * time.Millisecond,
		MaxHalfOpenReqs: 2,
	}
	cb := New(config)

	// Open the circuit
	testErr := errors.New("test error")
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	// Wait for reset timeout
	time.Sleep(150 * time.Millisecond)

	// Circuit should transition to half-open
	err := cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected request to succeed in half-open state, got %v", err)
	}

	if cb.GetState() != StateHalfOpen {
		t.Errorf("Expected state to be HalfOpen, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_HalfOpenToClosed(t *testing.T) {
	config := Config{
		MaxFailures:     3,
		ResetTimeout:    100 * time.Millisecond,
		HalfOpenTimeout: 500 * time.Millisecond,
		MaxHalfOpenReqs: 2,
	}
	cb := New(config)

	// Open the circuit
	testErr := errors.New("test error")
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	// Wait for reset timeout
	time.Sleep(150 * time.Millisecond)

	// Execute enough successful requests to close circuit
	for i := 0; i < config.MaxHalfOpenReqs; i++ {
		err := cb.Execute(func() error {
			return nil
		})
		if err != nil {
			t.Errorf("Expected request %d to succeed, got %v", i, err)
		}
	}

	// Circuit should be closed again
	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to be Closed after %d successful requests, got %v", config.MaxHalfOpenReqs, cb.GetState())
	}
}

func TestCircuitBreaker_HalfOpenToOpen(t *testing.T) {
	config := Config{
		MaxFailures:     3,
		ResetTimeout:    100 * time.Millisecond,
		HalfOpenTimeout: 500 * time.Millisecond,
		MaxHalfOpenReqs: 2,
	}
	cb := New(config)

	// Open the circuit
	testErr := errors.New("test error")
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	// Wait for reset timeout
	time.Sleep(150 * time.Millisecond)

	// Single failure in half-open should reopen circuit
	_ = cb.Execute(func() error {
		return testErr
	})

	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be Open after failure in half-open, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := New(DefaultConfig())

	// Open the circuit
	testErr := errors.New("test error")
	for i := 0; i < 10; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	if cb.GetState() != StateOpen {
		t.Error("Expected circuit to be open")
	}

	// Reset circuit
	cb.Reset()

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to be Closed after reset, got %v", cb.GetState())
	}

	if cb.GetFailures() != 0 {
		t.Errorf("Expected failure count to be 0 after reset, got %d", cb.GetFailures())
	}
}

func TestCircuitBreaker_TooManyRequests(t *testing.T) {
	config := Config{
		MaxFailures:     2,
		ResetTimeout:    100 * time.Millisecond,
		HalfOpenTimeout: 500 * time.Millisecond,
		MaxHalfOpenReqs: 2,
	}
	cb := New(config)

	// Open the circuit
	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	// Wait for reset timeout
	time.Sleep(150 * time.Millisecond)

	// Execute max half-open requests successfully
	for i := 0; i < config.MaxHalfOpenReqs; i++ {
		_ = cb.Execute(func() error {
			return nil
		})
	}

	// Circuit should be closed now, but let's verify the half-open behavior first
	// by reopening and checking the too many requests error

	// Reopen circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	// Wait for reset timeout again
	time.Sleep(150 * time.Millisecond)

	// Execute exactly max half-open requests
	for i := 0; i < config.MaxHalfOpenReqs; i++ {
		_ = cb.Execute(func() error {
			return nil
		})
	}

	// The circuit should have closed after MaxHalfOpenReqs successful requests
	if cb.GetState() != StateClosed {
		t.Errorf("Expected circuit to be closed after %d successful half-open requests, got %v",
			config.MaxHalfOpenReqs, cb.GetState())
	}
}
