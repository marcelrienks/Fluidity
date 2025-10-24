package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExecute_Success(t *testing.T) {
	config := DefaultConfig()
	config.MaxAttempts = 3

	attempts := 0
	err := Execute(context.Background(), config, AlwaysRetry(), func() error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestExecute_SuccessAfterRetry(t *testing.T) {
	config := DefaultConfig()
	config.MaxAttempts = 3
	config.InitialDelay = 10 * time.Millisecond

	attempts := 0
	testErr := errors.New("temporary error")

	err := Execute(context.Background(), config, AlwaysRetry(), func() error {
		attempts++
		if attempts < 3 {
			return testErr
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error after retry, got %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestExecute_MaxRetriesExceeded(t *testing.T) {
	config := DefaultConfig()
	config.MaxAttempts = 3
	config.InitialDelay = 10 * time.Millisecond

	attempts := 0
	testErr := errors.New("persistent error")

	err := Execute(context.Background(), config, AlwaysRetry(), func() error {
		attempts++
		return testErr
	})

	if err != testErr {
		t.Errorf("Expected test error, got %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestExecute_NonRetryableError(t *testing.T) {
	config := DefaultConfig()
	config.MaxAttempts = 3

	attempts := 0
	retryableErr := errors.New("retryable")
	nonRetryableErr := errors.New("non-retryable")

	shouldRetry := IsRetryable(retryableErr)

	err := Execute(context.Background(), config, shouldRetry, func() error {
		attempts++
		return nonRetryableErr
	})

	if err != nonRetryableErr {
		t.Errorf("Expected non-retryable error, got %v", err)
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt for non-retryable error, got %d", attempts)
	}
}

func TestExecute_ContextCancellation(t *testing.T) {
	config := DefaultConfig()
	config.MaxAttempts = 10
	config.InitialDelay = 100 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0
	testErr := errors.New("test error")

	// Cancel context after first attempt
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := Execute(ctx, config, AlwaysRetry(), func() error {
		attempts++
		return testErr
	})

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	if attempts > 2 {
		t.Errorf("Expected at most 2 attempts before cancellation, got %d", attempts)
	}
}

func TestExecuteWithResult_Success(t *testing.T) {
	config := DefaultConfig()
	config.MaxAttempts = 3

	attempts := 0
	expectedResult := 42

	result, err := ExecuteWithResult(context.Background(), config, AlwaysRetry(), func() (int, error) {
		attempts++
		return expectedResult, nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result != expectedResult {
		t.Errorf("Expected result %d, got %d", expectedResult, result)
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestExecuteWithResult_SuccessAfterRetry(t *testing.T) {
	config := DefaultConfig()
	config.MaxAttempts = 3
	config.InitialDelay = 10 * time.Millisecond

	attempts := 0
	expectedResult := "success"
	testErr := errors.New("temporary error")

	result, err := ExecuteWithResult(context.Background(), config, AlwaysRetry(), func() (string, error) {
		attempts++
		if attempts < 3 {
			return "", testErr
		}
		return expectedResult, nil
	})

	if err != nil {
		t.Errorf("Expected no error after retry, got %v", err)
	}

	if result != expectedResult {
		t.Errorf("Expected result %s, got %s", expectedResult, result)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestCalculateBackoff(t *testing.T) {
	initialDelay := 100 * time.Millisecond
	multiplier := 2.0
	maxDelay := 5 * time.Second

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{4, 800 * time.Millisecond},
		{5, 1600 * time.Millisecond},
		{6, 3200 * time.Millisecond},
		{7, 5000 * time.Millisecond}, // Capped at maxDelay
		{8, 5000 * time.Millisecond}, // Capped at maxDelay
	}

	for _, tt := range tests {
		result := CalculateBackoff(tt.attempt, initialDelay, multiplier, maxDelay)
		if result != tt.expected {
			t.Errorf("Attempt %d: expected %v, got %v", tt.attempt, tt.expected, result)
		}
	}
}

func TestExecute_ExponentialBackoff(t *testing.T) {
	config := Config{
		MaxAttempts:  3,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     500 * time.Millisecond,
		Multiplier:   2.0,
	}

	attempts := 0
	testErr := errors.New("test error")
	startTime := time.Now()

	_ = Execute(context.Background(), config, AlwaysRetry(), func() error {
		attempts++
		return testErr
	})

	duration := time.Since(startTime)

	// Expected delays: 50ms + 100ms = 150ms minimum
	expectedMin := 150 * time.Millisecond
	expectedMax := 300 * time.Millisecond // Allow some overhead

	if duration < expectedMin {
		t.Errorf("Expected duration >= %v, got %v", expectedMin, duration)
	}

	if duration > expectedMax {
		t.Errorf("Expected duration <= %v, got %v (allowing overhead)", expectedMax, duration)
	}
}

func TestIsRetryable(t *testing.T) {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	err3 := errors.New("error 3")

	shouldRetry := IsRetryable(err1, err2)

	if !shouldRetry(err1) {
		t.Error("Expected err1 to be retryable")
	}

	if !shouldRetry(err2) {
		t.Error("Expected err2 to be retryable")
	}

	if shouldRetry(err3) {
		t.Error("Expected err3 to not be retryable")
	}
}
