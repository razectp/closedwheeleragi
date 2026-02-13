package utils

import (
	"context"
	"time"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	tokens       chan struct{}
	refillRate   time.Duration
	maxTokens    int
	lastRefill   time.Time
	refillTicker *time.Ticker
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerSecond float64, maxBurst int) *RateLimiter {
	if requestsPerSecond <= 0 {
		requestsPerSecond = 1.0
	}
	if maxBurst <= 0 {
		maxBurst = 10
	}

	refillInterval := time.Duration(float64(time.Second) / requestsPerSecond)
	rl := &RateLimiter{
		tokens:     make(chan struct{}, maxBurst),
		refillRate: refillInterval,
		maxTokens:  maxBurst,
	}

	// Fill initial tokens
	for i := 0; i < maxBurst; i++ {
		rl.tokens <- struct{}{}
	}

	// Start refill ticker
	rl.refillTicker = time.NewTicker(refillInterval)
	go func() {
		for range rl.refillTicker.C {
			select {
			case rl.tokens <- struct{}{}:
			default:
				// Channel is full, skip
			}
		}
	}()

	return rl
}

// Wait blocks until a token is available or context is cancelled
func (rl *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TryWait attempts to acquire a token without blocking
func (rl *RateLimiter) TryWait() bool {
	select {
	case <-rl.tokens:
		return true
	default:
		return false
	}
}

// Stop stops the rate limiter
func (rl *RateLimiter) Stop() {
	if rl.refillTicker != nil {
		rl.refillTicker.Stop()
	}
}

// Global rate limiters for different API providers
var (
	OpenAIRateLimiter   = NewRateLimiter(60.0, 100)    // 60 requests/second, burst 100
	AnthropicRateLimiter = NewRateLimiter(50.0, 50)    // 50 requests/second, burst 50
	NVIDIARateLimiter   = NewRateLimiter(30.0, 30)     // 30 requests/second, burst 30
	DefaultRateLimiter  = NewRateLimiter(10.0, 20)     // 10 requests/second, burst 20
)

// GetRateLimiter returns the appropriate rate limiter for a provider
func GetRateLimiter(provider string) *RateLimiter {
	switch provider {
	case "openai":
		return OpenAIRateLimiter
	case "anthropic":
		return AnthropicRateLimiter
	case "nvidia":
		return NVIDIARateLimiter
	default:
		return DefaultRateLimiter
	}
}
