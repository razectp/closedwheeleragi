// Package utils provides common utility functions for the AGI agent.
package utils

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
)

// RetryConfig defines the configuration for retry logic using backoff/v4
type RetryConfig struct {
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       bool
}

// DefaultRetryConfig returns a standard retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
	}
}

// NewExponentialBackOff creates a backoff.ExponentialBackOff from RetryConfig
func (rc RetryConfig) NewExponentialBackOff() *backoff.ExponentialBackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = rc.InitialDelay
	b.MaxInterval = rc.MaxDelay
	b.Multiplier = rc.Multiplier
	if !rc.Jitter {
		b.RandomizationFactor = 0
	}
	return b
}

// ExecuteWithRetry executes a function with exponential backoff using backoff/v4
func ExecuteWithRetry(operation func() error, config RetryConfig) error {
	backoffConfig := config.NewExponentialBackOff()

	// Calculate max elapsed time based on max retries to prevent infinite retries
	// This is an approximation: sum of geometric series
	maxElapsedTime := time.Duration(0)
	currentDelay := config.InitialDelay
	for i := 0; i <= config.MaxRetries; i++ {
		maxElapsedTime += currentDelay
		currentDelay = time.Duration(float64(currentDelay) * config.Multiplier)
		if currentDelay > config.MaxDelay {
			currentDelay = config.MaxDelay
		}
	}
	backoffConfig.MaxElapsedTime = maxElapsedTime

	// Use backoff.Retry with custom notify function for logging
	err := backoff.RetryNotify(operation, backoffConfig, func(err error, next time.Duration) {
		// Optional: log retry attempts (commented out to reduce test noise)
		// fmt.Printf("Retry failed, waiting %v: %v\n", next, err)
	})

	if err != nil {
		return fmt.Errorf("operation failed after retries: %w", err)
	}

	return nil
}

// ExecuteWithRetryContext executes a function with exponential backoff using backoff/v4 with context
func ExecuteWithRetryContext(ctx context.Context, operation func() error, config RetryConfig) error {
	backoffConfig := config.NewExponentialBackOff()
	backoffConfig.MaxElapsedTime = 0 // No max elapsed time, use context instead

	// Use backoff.Retry with context cancellation
	operationWithContext := func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return operation()
		}
	}

	err := backoff.Retry(operationWithContext, backoffConfig)

	if err != nil {
		return fmt.Errorf("operation failed after retries: %w", err)
	}

	return nil
}

// IsRetryableError determines if an HTTP status code is retryable (429 or 5xx)
func IsRetryableError(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || (statusCode >= 500 && statusCode <= 599)
}
