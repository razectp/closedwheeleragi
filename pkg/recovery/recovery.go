// Package recovery provides error recovery and resilience mechanisms
package recovery

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
)

// ErrorHandler handles errors gracefully without stopping execution
type ErrorHandler struct {
	mu              sync.RWMutex
	errorLog        []ErrorEntry
	maxLogSize      int
	retryStrategies map[string]RetryStrategy
	fallbacks       map[string]FallbackFunc
}

// ErrorEntry represents a logged error
type ErrorEntry struct {
	Timestamp  time.Time
	Error      error
	Context    string
	Operation  string
	Recovered  bool
	RetryCount int
	StackTrace string
}

// RetryStrategy defines how to retry an operation
type RetryStrategy struct {
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// FallbackFunc is a function that provides fallback behavior
type FallbackFunc func(error) error

// NewErrorHandler creates a new error handler
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		errorLog:        make([]ErrorEntry, 0),
		maxLogSize:      1000,
		retryStrategies: defaultRetryStrategies(),
		fallbacks:       make(map[string]FallbackFunc),
	}
}

// defaultRetryStrategies returns default retry strategies
func defaultRetryStrategies() map[string]RetryStrategy {
	return map[string]RetryStrategy{
		"file_write": {
			MaxRetries:   3,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     2 * time.Second,
			Multiplier:   2.0,
		},
		"file_read": {
			MaxRetries:   3,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     2 * time.Second,
			Multiplier:   2.0,
		},
		"api_call": {
			MaxRetries:   5,
			InitialDelay: 1 * time.Second,
			MaxDelay:     30 * time.Second,
			Multiplier:   2.0,
		},
		"network": {
			MaxRetries:   3,
			InitialDelay: 500 * time.Millisecond,
			MaxDelay:     5 * time.Second,
			Multiplier:   2.0,
		},
	}
}

// HandleError handles an error gracefully and logs it
func (eh *ErrorHandler) HandleError(err error, context, operation string) {
	if err == nil {
		return
	}

	entry := ErrorEntry{
		Timestamp:  time.Now(),
		Error:      err,
		Context:    context,
		Operation:  operation,
		Recovered:  false,
		StackTrace: string(debug.Stack()),
	}

	eh.mu.Lock()
	eh.errorLog = append(eh.errorLog, entry)
	if len(eh.errorLog) > eh.maxLogSize {
		eh.errorLog = eh.errorLog[len(eh.errorLog)-eh.maxLogSize:]
	}
	eh.mu.Unlock()

	// Log to stderr but don't panic
	log.Printf("[ERROR] %s in %s: %v", operation, context, err)
}

// RecoverFromPanic recovers from a panic and logs it
func (eh *ErrorHandler) RecoverFromPanic(context string) {
	if r := recover(); r != nil {
		err := fmt.Errorf("panic recovered: %v", r)
		entry := ErrorEntry{
			Timestamp:  time.Now(),
			Error:      err,
			Context:    context,
			Operation:  "panic_recovery",
			Recovered:  true,
			StackTrace: string(debug.Stack()),
		}

		eh.mu.Lock()
		eh.errorLog = append(eh.errorLog, entry)
		eh.mu.Unlock()

		log.Printf("[PANIC RECOVERED] %s: %v\n%s", context, r, entry.StackTrace)
	}
}

// RetryWithBackoff retries an operation with exponential backoff
func (eh *ErrorHandler) RetryWithBackoff(operation string, fn func() error) error {
	strategy, exists := eh.retryStrategies[operation]
	if !exists {
		strategy = eh.retryStrategies["file_write"] // default
	}

	config := backoff.NewExponentialBackOff()
	config.MaxInterval = strategy.MaxDelay
	config.Multiplier = strategy.Multiplier
	config.InitialInterval = strategy.InitialDelay
	config.MaxElapsedTime = 0
	config.RandomizationFactor = 0.1

	var lastErr error
	err := backoff.Retry(func() error {
		err := fn()
		if err != nil {
			lastErr = err
			eh.HandleError(err, "retry", operation)
		}
		return err
	}, config)

	if err != nil {
		return fmt.Errorf("operation %s failed after %d retries: %w", operation, strategy.MaxRetries, lastErr)
	}

	return nil
}

// SafeFileWrite writes to a file with retry and fallback
func (eh *ErrorHandler) SafeFileWrite(path string, data []byte) error {
	return eh.RetryWithBackoff("file_write", func() error {
		// Try to create directory if it doesn't exist
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			// If can't create dir, try temp directory
			tempDir := os.TempDir()
			tempPath := filepath.Join(tempDir, filepath.Base(path))
			log.Printf("[FALLBACK] Can't write to %s, trying %s", path, tempPath)

			if err := os.WriteFile(tempPath, data, 0644); err != nil {
				return fmt.Errorf("failed to write to both primary and temp locations: %w", err)
			}

			log.Printf("[FALLBACK SUCCESS] Wrote to temp location: %s", tempPath)
			return nil
		}

		// Try to write file
		return os.WriteFile(path, data, 0644)
	})
}

// SafeFileRead reads from a file with retry
func (eh *ErrorHandler) SafeFileRead(path string) ([]byte, error) {
	var data []byte
	err := eh.RetryWithBackoff("file_read", func() error {
		var readErr error
		data, readErr = os.ReadFile(path)
		return readErr
	})
	return data, err
}

// GetRecentErrors returns recent errors
func (eh *ErrorHandler) GetRecentErrors(count int) []ErrorEntry {
	eh.mu.RLock()
	defer eh.mu.RUnlock()

	if count > len(eh.errorLog) {
		count = len(eh.errorLog)
	}

	// Return last N errors
	start := len(eh.errorLog) - count
	if start < 0 {
		start = 0
	}

	result := make([]ErrorEntry, count)
	copy(result, eh.errorLog[start:])
	return result
}

// GetErrorStats returns error statistics
func (eh *ErrorHandler) GetErrorStats() map[string]interface{} {
	eh.mu.RLock()
	defer eh.mu.RUnlock()

	totalErrors := len(eh.errorLog)
	recovered := 0
	byOperation := make(map[string]int)

	for _, entry := range eh.errorLog {
		if entry.Recovered {
			recovered++
		}
		byOperation[entry.Operation]++
	}

	return map[string]interface{}{
		"total_errors":  totalErrors,
		"recovered":     recovered,
		"by_operation":  byOperation,
		"recovery_rate": float64(recovered) / float64(totalErrors) * 100,
	}
}

// ClearErrorLog clears the error log
func (eh *ErrorHandler) ClearErrorLog() {
	eh.mu.Lock()
	defer eh.mu.Unlock()
	eh.errorLog = make([]ErrorEntry, 0)
}

// IsTransientError checks if an error is likely transient
func IsTransientError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	transientPatterns := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"temporary failure",
		"too many requests",
		"rate limit",
		"503",
		"502",
		"504",
		"network is unreachable",
		"no route to host",
	}

	for _, pattern := range transientPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// ShouldRetry determines if an operation should be retried
func ShouldRetry(err error, attempt int, maxRetries int) bool {
	if err == nil {
		return false
	}

	if attempt >= maxRetries {
		return false
	}

	// Always retry transient errors
	if IsTransientError(err) {
		return true
	}

	// Don't retry permission errors
	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "permission denied") ||
		strings.Contains(errStr, "access denied") {
		return false
	}

	// Retry file-related errors that aren't permission issues
	if strings.Contains(errStr, "file") ||
		strings.Contains(errStr, "directory") ||
		strings.Contains(errStr, "path") {
		return true
	}

	return false
}

// WrapWithRecovery wraps a function with panic recovery
func (eh *ErrorHandler) WrapWithRecovery(context string, fn func() error) error {
	var err error
	func() {
		defer eh.RecoverFromPanic(context)
		err = fn()
	}()
	return err
}

// SafeOperation executes an operation safely with full error handling
func (eh *ErrorHandler) SafeOperation(context, operation string, fn func() error) error {
	return eh.WrapWithRecovery(context, func() error {
		err := fn()
		if err != nil {
			eh.HandleError(err, context, operation)

			// Try to recover if it's a transient error
			if IsTransientError(err) {
				return eh.RetryWithBackoff(operation, fn)
			}
		}
		return err
	})
}

// FormatErrorReport formats a human-readable error report
func (eh *ErrorHandler) FormatErrorReport() string {
	eh.mu.RLock()
	defer eh.mu.RUnlock()

	if len(eh.errorLog) == 0 {
		return "No errors recorded."
	}

	var report strings.Builder
	report.WriteString("🔴 Error Report\n")
	report.WriteString(strings.Repeat("═", 60) + "\n\n")

	stats := eh.GetErrorStats()
	report.WriteString(fmt.Sprintf("Total Errors: %v\n", stats["total_errors"]))
	report.WriteString(fmt.Sprintf("Recovered: %v\n", stats["recovered"]))
	report.WriteString(fmt.Sprintf("Recovery Rate: %.1f%%\n\n", stats["recovery_rate"]))

	report.WriteString("Recent Errors (last 10):\n")
	report.WriteString(strings.Repeat("─", 60) + "\n")

	recent := eh.GetRecentErrors(10)
	for i, entry := range recent {
		status := "❌"
		if entry.Recovered {
			status = "✅"
		}

		report.WriteString(fmt.Sprintf("\n%d. %s [%s] %s\n",
			i+1, status, entry.Timestamp.Format("15:04:05"), entry.Operation))
		report.WriteString(fmt.Sprintf("   Context: %s\n", entry.Context))
		report.WriteString(fmt.Sprintf("   Error: %v\n", entry.Error))

		if entry.RetryCount > 0 {
			report.WriteString(fmt.Sprintf("   Retries: %d\n", entry.RetryCount))
		}
	}

	return report.String()
}

// Global error handler instance
var globalHandler *ErrorHandler
var once sync.Once

// GetGlobalHandler returns the global error handler
func GetGlobalHandler() *ErrorHandler {
	once.Do(func() {
		globalHandler = NewErrorHandler()
	})
	return globalHandler
}

// HandleError is a convenience function for the global handler
func HandleError(err error, context, operation string) {
	GetGlobalHandler().HandleError(err, context, operation)
}

// RecoverFromPanic is a convenience function for the global handler
func RecoverFromPanic(context string) {
	GetGlobalHandler().RecoverFromPanic(context)
}

// SafeFileWrite is a convenience function for the global handler
func SafeFileWrite(path string, data []byte) error {
	return GetGlobalHandler().SafeFileWrite(path, data)
}

// SafeFileRead is a convenience function for the global handler
func SafeFileRead(path string) ([]byte, error) {
	return GetGlobalHandler().SafeFileRead(path)
}
