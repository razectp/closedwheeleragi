// Package tools provides intelligent retry mechanisms for tool execution
package tools

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ClosedWheeler/pkg/recovery"
)

// ToolExecutor interface for compatibility with both Executor and IntelligentRetryWrapper
type ToolExecutor interface {
	Execute(call ToolCall) (ToolResult, error)
	SetDebugLevel(level DebugLevel)
	GetDebugReport() string
	GetRecentFailures() []ExecutionTrace
	ExecuteFromJSON(jsonStr string) (ToolResult, error)
}

// RetryContext stores context about retry attempts
type RetryContext struct {
	ToolName     string
	Attempts     []RetryAttempt
	MaxAttempts  int
	CurrentTry   int
	LastError    error
	Suggestions  []string
	mu           sync.RWMutex
}

// RetryAttempt represents a single attempt to execute a tool
type RetryAttempt struct {
	AttemptNumber int
	Timestamp     time.Time
	Arguments     map[string]any
	Error         string
	ErrorType     string // permission, path_invalid, file_not_found, etc
	Recovered     bool
	Suggestion    string
}

// IntelligentRetryWrapper wraps tool execution with intelligent retry logic
// It implements the same interface as Executor so it can be used as a drop-in replacement
type IntelligentRetryWrapper struct {
	executor      *Executor
	contexts      map[string]*RetryContext
	mu            sync.RWMutex
	feedbackMode  bool // When true, returns detailed feedback to LLM
}

// NewIntelligentRetryWrapper creates a new intelligent retry wrapper
func NewIntelligentRetryWrapper(executor *Executor) *IntelligentRetryWrapper {
	return &IntelligentRetryWrapper{
		executor:     executor,
		contexts:     make(map[string]*RetryContext),
		feedbackMode: true,
	}
}

// Execute executes a tool call (compatible with Executor interface)
// This is the main entry point that provides intelligent retry
func (w *IntelligentRetryWrapper) Execute(call ToolCall) (ToolResult, error) {
	return w.ExecuteWithRetry(call)
}

// EnableFeedbackMode enables detailed feedback to LLM
func (w *IntelligentRetryWrapper) EnableFeedbackMode(enabled bool) {
	w.feedbackMode = enabled
}

// ExecuteWithRetry executes a tool with intelligent retry logic
func (w *IntelligentRetryWrapper) ExecuteWithRetry(call ToolCall) (ToolResult, error) {
	// Get or create retry context
	ctx := w.getOrCreateContext(call.Name)

	ctx.mu.Lock()
	ctx.CurrentTry++
	attemptNum := ctx.CurrentTry
	ctx.mu.Unlock()

	// Execute the tool
	result, err := w.executor.Execute(call)

	// Record attempt
	attempt := RetryAttempt{
		AttemptNumber: attemptNum,
		Timestamp:     time.Now(),
		Arguments:     call.Arguments,
		Recovered:     result.Success,
	}

	if err != nil || !result.Success {
		// Analyze error and provide feedback
		errorType, suggestions := w.analyzeError(call, result, err)

		attempt.Error = result.Error
		if err != nil {
			attempt.Error = err.Error()
		}
		attempt.ErrorType = errorType
		attempt.Suggestion = w.formatSuggestion(errorType, call, suggestions)

		ctx.mu.Lock()
		ctx.Attempts = append(ctx.Attempts, attempt)
		ctx.LastError = err
		ctx.Suggestions = suggestions
		ctx.mu.Unlock()

		// Log the error with recovery system
		recovery.HandleError(fmt.Errorf("%s", attempt.Error), "tool_execution", call.Name)

		// If feedback mode is enabled, enhance the error message for LLM
		if w.feedbackMode {
			result = w.enhanceErrorForLLM(result, attempt, ctx, call)
		}

		log.Printf("[TOOL RETRY] %s failed (attempt %d/%d): %s",
			call.Name, attemptNum, ctx.MaxAttempts, attempt.ErrorType)
	} else {
		attempt.Recovered = true
		ctx.mu.Lock()
		ctx.Attempts = append(ctx.Attempts, attempt)
		ctx.mu.Unlock()

		if attemptNum > 1 {
			log.Printf("[TOOL RETRY SUCCESS] %s succeeded after %d attempts", call.Name, attemptNum)
		}
	}

	return result, err
}

// analyzeError analyzes the error and returns error type and suggestions
func (w *IntelligentRetryWrapper) analyzeError(call ToolCall, result ToolResult, err error) (string, []string) {
	errorMsg := result.Error
	if err != nil {
		errorMsg = err.Error()
	}

	errorMsgLower := strings.ToLower(errorMsg)
	suggestions := []string{}

	// Permission errors
	if strings.Contains(errorMsgLower, "permission denied") ||
	   strings.Contains(errorMsgLower, "access denied") {
		suggestions = append(suggestions,
			"Try writing to a different directory (e.g., 'workplace/', 'temp/', or '.agi/temp/')",
			"Check if the file is read-only or locked by another process",
			"Try creating the directory first with proper permissions",
		)
		return "permission_denied", suggestions
	}

	// Path errors
	if strings.Contains(errorMsgLower, "no such file") ||
	   strings.Contains(errorMsgLower, "cannot find the path") ||
	   strings.Contains(errorMsgLower, "path does not exist") {

		// Extract path if possible
		if path, ok := call.Arguments["path"].(string); ok {
			suggestions = append(suggestions,
				fmt.Sprintf("The path '%s' does not exist", path),
				"Create the parent directory first",
				"Check for typos in the path",
				"Try using an absolute path or verify the relative path",
			)
		}
		return "path_not_found", suggestions
	}

	// Invalid path characters
	if strings.Contains(errorMsgLower, "invalid argument") ||
	   strings.Contains(errorMsgLower, "illegal character") {
		suggestions = append(suggestions,
			"Check for invalid characters in the path (e.g., *, ?, <, >, |, :, \")",
			"Avoid special characters in file names",
			"Use forward slashes (/) or proper backslashes (\\\\) in paths",
		)
		return "invalid_path", suggestions
	}

	// File already exists
	if strings.Contains(errorMsgLower, "file exists") ||
	   strings.Contains(errorMsgLower, "already exists") {
		suggestions = append(suggestions,
			"The file already exists - do you want to overwrite it?",
			"Try reading the file first to see its contents",
			"Use a different filename or path",
		)
		return "file_exists", suggestions
	}

	// Disk full / space errors
	if strings.Contains(errorMsgLower, "no space left") ||
	   strings.Contains(errorMsgLower, "disk full") {
		suggestions = append(suggestions,
			"Not enough disk space available",
			"Try cleaning up temporary files",
			"Choose a different location with more space",
		)
		return "no_space", suggestions
	}

	// Generic file system errors
	if strings.Contains(errorMsgLower, "file") ||
	   strings.Contains(errorMsgLower, "directory") {
		suggestions = append(suggestions,
			"Verify the path is correct",
			"Check file permissions",
			"Try using a different location",
		)
		return "filesystem_error", suggestions
	}

	// Security/audit errors
	if strings.Contains(errorMsgLower, "security") ||
	   strings.Contains(errorMsgLower, "escapes project root") {
		suggestions = append(suggestions,
			"Path escapes the allowed project root",
			"Use paths relative to the project directory",
			"Avoid using '..' to go outside the project",
		)
		return "security_violation", suggestions
	}

	// Generic error
	suggestions = append(suggestions,
		"Check the error message above for details",
		"Try simplifying the operation",
		"Verify all parameters are correct",
	)
	return "unknown_error", suggestions
}

// formatSuggestion formats suggestions for LLM
func (w *IntelligentRetryWrapper) formatSuggestion(errorType string, call ToolCall, suggestions []string) string {
	var result strings.Builder

	result.WriteString(fmt.Sprintf("Error Type: %s\n", errorType))
	result.WriteString("Suggestions to fix:\n")

	for i, sugg := range suggestions {
		result.WriteString(fmt.Sprintf("%d. %s\n", i+1, sugg))
	}

	// Add context-specific suggestions
	if call.Name == "write_file" {
		if path, ok := call.Arguments["path"].(string); ok {
			dir := filepath.Dir(path)
			result.WriteString(fmt.Sprintf("\nAlternative locations to try:\n"))
			result.WriteString(fmt.Sprintf("- workplace/%s\n", filepath.Base(path)))
			result.WriteString(fmt.Sprintf("- .agi/temp/%s\n", filepath.Base(path)))
			result.WriteString(fmt.Sprintf("- temp/%s\n", filepath.Base(path)))

			if dir != "." && dir != "" {
				result.WriteString(fmt.Sprintf("\nNote: Target directory is '%s' - does it exist?\n", dir))
			}
		}
	}

	return result.String()
}

// enhanceErrorForLLM enhances error result with detailed feedback for LLM
func (w *IntelligentRetryWrapper) enhanceErrorForLLM(
	result ToolResult,
	attempt RetryAttempt,
	ctx *RetryContext,
	call ToolCall,
) ToolResult {

	var enhanced strings.Builder

	enhanced.WriteString("❌ TOOL EXECUTION FAILED\n\n")
	enhanced.WriteString(fmt.Sprintf("Tool: %s\n", call.Name))
	enhanced.WriteString(fmt.Sprintf("Attempt: %d/%d\n", attempt.AttemptNumber, ctx.MaxAttempts))
	enhanced.WriteString(fmt.Sprintf("Error Type: %s\n\n", attempt.ErrorType))

	enhanced.WriteString("📋 ORIGINAL ERROR:\n")
	enhanced.WriteString(fmt.Sprintf("%s\n\n", attempt.Error))

	enhanced.WriteString("💡 WHAT WENT WRONG:\n")
	enhanced.WriteString(w.explainError(attempt.ErrorType, call))
	enhanced.WriteString("\n\n")

	enhanced.WriteString("🔧 HOW TO FIX IT:\n")
	enhanced.WriteString(attempt.Suggestion)
	enhanced.WriteString("\n")

	// Show previous attempts if any
	ctx.mu.RLock()
	if len(ctx.Attempts) > 1 {
		enhanced.WriteString("\n📊 PREVIOUS ATTEMPTS:\n")
		for i, prevAttempt := range ctx.Attempts[:len(ctx.Attempts)-1] {
			enhanced.WriteString(fmt.Sprintf("%d. %s - %s\n",
				i+1, prevAttempt.Timestamp.Format("15:04:05"), prevAttempt.ErrorType))
		}
	}
	ctx.mu.RUnlock()

	enhanced.WriteString("\n⚡ ACTION REQUIRED:\n")
	enhanced.WriteString("Please analyze the error and try again with:\n")
	enhanced.WriteString("1. A corrected path or parameter\n")
	enhanced.WriteString("2. An alternative location (see suggestions above)\n")
	enhanced.WriteString("3. A different approach to achieve the same goal\n")
	enhanced.WriteString("\nI will automatically retry when you call the tool again with corrected parameters.\n")

	result.Error = enhanced.String()
	return result
}

// explainError provides human-readable explanation of error types
func (w *IntelligentRetryWrapper) explainError(errorType string, call ToolCall) string {
	switch errorType {
	case "permission_denied":
		return "You don't have permission to write to this location. This usually happens when:\n" +
			"- The directory requires administrator/root privileges\n" +
			"- The file is marked as read-only\n" +
			"- Another process has locked the file"

	case "path_not_found":
		if path, ok := call.Arguments["path"].(string); ok {
			dir := filepath.Dir(path)
			return fmt.Sprintf("The path doesn't exist. Specifically:\n"+
				"- Target file: %s\n"+
				"- Parent directory: %s\n"+
				"- The parent directory needs to be created first",
				filepath.Base(path), dir)
		}
		return "The specified path doesn't exist. The parent directory needs to be created first."

	case "invalid_path":
		return "The path contains invalid characters. On Windows, avoid: * ? < > | : \"\n" +
			"On all systems, use forward slashes (/) or properly escaped backslashes (\\\\)"

	case "file_exists":
		return "A file already exists at this location. You can:\n" +
			"- Read the existing file first\n" +
			"- Choose a different filename\n" +
			"- Explicitly overwrite if that's the intent"

	case "no_space":
		return "Not enough disk space to complete the operation."

	case "security_violation":
		return "The operation violates security constraints. The path is trying to access\n" +
			"locations outside the allowed project directory."

	default:
		return "An unexpected error occurred. Check the original error message for details."
	}
}

// getOrCreateContext gets or creates a retry context for a tool
func (w *IntelligentRetryWrapper) getOrCreateContext(toolName string) *RetryContext {
	w.mu.Lock()
	defer w.mu.Unlock()

	if ctx, exists := w.contexts[toolName]; exists {
		return ctx
	}

	ctx := &RetryContext{
		ToolName:    toolName,
		Attempts:    make([]RetryAttempt, 0),
		MaxAttempts: 5,
		CurrentTry:  0,
	}
	w.contexts[toolName] = ctx
	return ctx
}

// ResetContext resets retry context for a tool
func (w *IntelligentRetryWrapper) ResetContext(toolName string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.contexts, toolName)
}

// GetRetryStats returns statistics about retry attempts
func (w *IntelligentRetryWrapper) GetRetryStats() map[string]interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	totalAttempts := 0
	totalSuccess := 0
	byTool := make(map[string]int)
	byErrorType := make(map[string]int)

	for toolName, ctx := range w.contexts {
		ctx.mu.RLock()
		byTool[toolName] = len(ctx.Attempts)
		totalAttempts += len(ctx.Attempts)

		for _, attempt := range ctx.Attempts {
			if attempt.Recovered {
				totalSuccess++
			}
			if attempt.ErrorType != "" {
				byErrorType[attempt.ErrorType]++
			}
		}
		ctx.mu.RUnlock()
	}

	successRate := 0.0
	if totalAttempts > 0 {
		successRate = float64(totalSuccess) / float64(totalAttempts) * 100
	}

	return map[string]interface{}{
		"total_attempts": totalAttempts,
		"total_success":  totalSuccess,
		"success_rate":   successRate,
		"by_tool":        byTool,
		"by_error_type":  byErrorType,
	}
}

// FormatRetryReport formats a human-readable retry report
func (w *IntelligentRetryWrapper) FormatRetryReport() string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var report strings.Builder
	report.WriteString("🔄 Intelligent Retry System Report\n")
	report.WriteString(strings.Repeat("═", 60) + "\n\n")

	stats := w.GetRetryStats()
	report.WriteString(fmt.Sprintf("Total Attempts: %v\n", stats["total_attempts"]))
	report.WriteString(fmt.Sprintf("Successful: %v\n", stats["total_success"]))
	report.WriteString(fmt.Sprintf("Success Rate: %.1f%%\n\n", stats["success_rate"]))

	if byTool, ok := stats["by_tool"].(map[string]int); ok && len(byTool) > 0 {
		report.WriteString("Attempts by Tool:\n")
		for tool, count := range byTool {
			report.WriteString(fmt.Sprintf("- %s: %d\n", tool, count))
		}
		report.WriteString("\n")
	}

	if byError, ok := stats["by_error_type"].(map[string]int); ok && len(byError) > 0 {
		report.WriteString("Errors by Type:\n")
		for errType, count := range byError {
			report.WriteString(fmt.Sprintf("- %s: %d\n", errType, count))
		}
	}

	return report.String()
}

// Executor compatibility methods - delegate to wrapped executor

// SetDebugLevel sets the debug level (delegates to wrapped executor)
func (w *IntelligentRetryWrapper) SetDebugLevel(level DebugLevel) {
	if w.executor != nil {
		w.executor.SetDebugLevel(level)
	}
}

// GetDebugReport gets debug report (delegates to wrapped executor)
func (w *IntelligentRetryWrapper) GetDebugReport() string {
	if w.executor != nil {
		return w.executor.GetDebugReport()
	}
	return ""
}

// GetRecentFailures gets recent failures (delegates to wrapped executor)
func (w *IntelligentRetryWrapper) GetRecentFailures() []ExecutionTrace {
	if w.executor != nil {
		return w.executor.GetRecentFailures()
	}
	return []ExecutionTrace{}
}

// ExecuteFromJSON executes from JSON (delegates with retry)
func (w *IntelligentRetryWrapper) ExecuteFromJSON(jsonStr string) (ToolResult, error) {
	if w.executor != nil {
		return w.executor.ExecuteFromJSON(jsonStr)
	}
	return ToolResult{Success: false, Error: "no executor available"}, fmt.Errorf("no executor available")
}
