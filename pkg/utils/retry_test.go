package utils

import (
	"errors"
	"testing"
	"time"
)

func TestExecuteWithRetry(t *testing.T) {
	config := RetryConfig{
		MaxRetries:   2,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		JitterFactor: 0.1,
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
		// Initial try + 2 retries = 3 calls
		if calls != 3 {
			t.Errorf("Expected 3 calls, got %d", calls)
		}
	})
}
