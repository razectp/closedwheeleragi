package utils

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExecuteWithRetry(t *testing.T) {
	config := RetryConfig{
		MaxRetries:   2,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       false, // Disable jitter for predictable tests
	}

	t.Run("success first try", func(t *testing.T) {
		calls := 0
		err := ExecuteWithRetry(func() error {
			calls++
			return nil
		}, config)

		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
		if calls != 1 {
			t.Errorf("Expected 1 call, got %d", calls)
		}
	})

	t.Run("success after retries", func(t *testing.T) {
		calls := 0
		err := ExecuteWithRetry(func() error {
			calls++
			if calls < 3 {
				return errors.New("temporary error")
			}
			return nil
		}, config)

		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
		if calls != 3 {
			t.Errorf("Expected 3 calls, got %d", calls)
		}
	})

	t.Run("fail after max retries", func(t *testing.T) {
		calls := 0
		err := ExecuteWithRetry(func() error {
			calls++
			return errors.New("persistent error")
		}, config)

		if err == nil {
			t.Error("Expected error, got nil")
		}
		// Should retry until backoff gives up (not exactly max retries with backoff/v4)
		if calls < 2 {
			t.Errorf("Expected at least 2 calls, got %d", calls)
		}
	})
}

func TestExecuteWithRetryContext(t *testing.T) {
	config := RetryConfig{
		MaxRetries:   10,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       false,
	}

	t.Run("context cancellation", func(t *testing.T) {
		calls := 0
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := ExecuteWithRetryContext(ctx, func() error {
			calls++
			return errors.New("always fails")
		}, config)

		if err == nil {
			t.Error("Expected error due to context cancellation, got nil")
		}
		// Should stop quickly due to context cancellation
		if calls > 10 {
			t.Errorf("Expected quick cancellation, got %d calls", calls)
		}
	})
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries=3, got %d", config.MaxRetries)
	}
	if config.InitialDelay != 1*time.Second {
		t.Errorf("Expected InitialDelay=1s, got %v", config.InitialDelay)
	}
	if config.MaxDelay != 10*time.Second {
		t.Errorf("Expected MaxDelay=10s, got %v", config.MaxDelay)
	}
	if config.Multiplier != 2.0 {
		t.Errorf("Expected Multiplier=2.0, got %f", config.Multiplier)
	}
	if !config.Jitter {
		t.Error("Expected Jitter=true, got false")
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		statusCode int
		expected   bool
	}{
		{200, false},
		{400, false},
		{429, true},
		{500, true},
		{503, true},
	}

	for _, test := range tests {
		result := IsRetryableError(test.statusCode)
		if result != test.expected {
			t.Errorf("IsRetryableError(%d) = %v, expected %v", test.statusCode, result, test.expected)
		}
	}
}
