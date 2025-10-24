package retry

import (
	"context"
	"errors"
	"math"
	"time"
)

var (
	ErrMaxRetriesExceeded = errors.New("maximum retries exceeded")
)

// Config holds retry configuration
type Config struct {
	MaxAttempts     int           // Maximum number of retry attempts (including initial)
	InitialDelay    time.Duration // Initial delay between retries
	MaxDelay        time.Duration // Maximum delay between retries
	Multiplier      float64       // Multiplier for exponential backoff
	RetryableErrors []error       // Specific errors that should trigger retry
}

// DefaultConfig returns default retry configuration
func DefaultConfig() Config {
	return Config{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
	}
}

// ShouldRetry determines if an error is retryable
type ShouldRetry func(error) bool

// Execute executes a function with retry logic
func Execute(ctx context.Context, config Config, shouldRetry ShouldRetry, fn func() error) error {
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 1
	}
	if config.InitialDelay <= 0 {
		config.InitialDelay = 100 * time.Millisecond
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = 10 * time.Second
	}
	if config.Multiplier <= 0 {
		config.Multiplier = 2.0
	}

	var lastErr error
	delay := config.InitialDelay

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Execute the function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if attempt >= config.MaxAttempts {
			break
		}

		// Check if error is retryable
		if shouldRetry != nil && !shouldRetry(err) {
			return err
		}

		// Wait before retrying
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Calculate next delay with exponential backoff
			delay = time.Duration(float64(delay) * config.Multiplier)
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
		}
	}

	return lastErr
}

// ExecuteWithResult executes a function with retry logic and returns a result
func ExecuteWithResult[T any](ctx context.Context, config Config, shouldRetry ShouldRetry, fn func() (T, error)) (T, error) {
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 1
	}
	if config.InitialDelay <= 0 {
		config.InitialDelay = 100 * time.Millisecond
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = 10 * time.Second
	}
	if config.Multiplier <= 0 {
		config.Multiplier = 2.0
	}

	var lastErr error
	var zeroValue T
	delay := config.InitialDelay

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Execute the function
		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if we should retry
		if attempt >= config.MaxAttempts {
			break
		}

		// Check if error is retryable
		if shouldRetry != nil && !shouldRetry(err) {
			return zeroValue, err
		}

		// Wait before retrying
		select {
		case <-ctx.Done():
			return zeroValue, ctx.Err()
		case <-time.After(delay):
			// Calculate next delay with exponential backoff
			delay = time.Duration(float64(delay) * config.Multiplier)
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
		}
	}

	return zeroValue, lastErr
}

// IsRetryable returns a ShouldRetry function that checks for specific error types
func IsRetryable(retryableErrors ...error) ShouldRetry {
	return func(err error) bool {
		for _, retryableErr := range retryableErrors {
			if errors.Is(err, retryableErr) {
				return true
			}
		}
		return false
	}
}

// IsTemporary returns a ShouldRetry function that checks for temporary errors
func IsTemporary() ShouldRetry {
	return func(err error) bool {
		type temporary interface {
			Temporary() bool
		}
		if temp, ok := err.(temporary); ok {
			return temp.Temporary()
		}
		return false
	}
}

// AlwaysRetry returns a ShouldRetry function that always retries
func AlwaysRetry() ShouldRetry {
	return func(err error) bool {
		return true
	}
}

// CalculateBackoff calculates the backoff duration for a given attempt
func CalculateBackoff(attempt int, initialDelay time.Duration, multiplier float64, maxDelay time.Duration) time.Duration {
	delay := float64(initialDelay) * math.Pow(multiplier, float64(attempt-1))
	if delay > float64(maxDelay) {
		return maxDelay
	}
	return time.Duration(delay)
}
