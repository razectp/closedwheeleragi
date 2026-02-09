package llm

import (
	"fmt"
	"log"
	"strings"
	"time"

	"ClosedWheeler/pkg/recovery"
)

// HandleErrorGracefully handles LLM errors gracefully and returns a helpful message
func HandleErrorGracefully(err error, operation string) string {
	if err == nil {
		return ""
	}

	// Log the error
	recovery.HandleError(err, "llm_client", operation)

	// Create helpful error message
	errorMessage := fmt.Sprintf(`⚠️ I encountered an error while processing your request:

**Error:** %v

**What I'm doing about it:**
- I've logged this error for analysis
- I'm continuing to work despite this issue
- This error won't stop me from helping you

**Suggestions:**
- If this was a file operation, check file permissions
- If this was an API call, check your network connection
- Try simplifying your request
- Use /errors to see recent errors
- Use /resilience for system status

I'm designed to keep working even when errors occur. Please continue with your next request!`, err)

	return errorMessage
}

// RetryOperation retries an operation with exponential backoff
func RetryOperation(operation string, maxRetries int, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(attempt) * time.Second
			log.Printf("[RETRY] Attempt %d/%d for %s after %v", attempt, maxRetries, operation, delay)
			time.Sleep(delay)
		}

		// Try with panic recovery
		err := recovery.GetGlobalHandler().WrapWithRecovery(operation, fn)

		if err == nil {
			if attempt > 0 {
				log.Printf("[RETRY SUCCESS] %s succeeded after %d attempts", operation, attempt)
			}
			return nil
		}

		lastErr = err
		recovery.HandleError(err, "retry", operation)

		// Check if we should continue retrying
		if !recovery.ShouldRetry(err, attempt, maxRetries) {
			log.Printf("[RETRY ABORT] %s not retryable: %v", operation, err)
			break
		}

		// Handle specific error types
		if strings.Contains(err.Error(), "rate limit") {
			log.Println("[RETRY] Rate limit hit, waiting 10s...")
			time.Sleep(10 * time.Second)
		}
	}

	return fmt.Errorf("operation %s failed after %d retries: %w", operation, maxRetries, lastErr)
}
