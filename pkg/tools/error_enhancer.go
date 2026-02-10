// Package tools provides error enhancement for LLM feedback
package tools

import (
	"fmt"
	"path/filepath"
	"strings"
)

// EnhanceToolError enhances a tool error with concise, clean feedback for the LLM.
// Uses plain text (no markdown) so it renders cleanly in the TUI if echoed.
func EnhanceToolError(toolName string, args map[string]any, result ToolResult) ToolResult {
	if result.Success || result.Error == "" {
		return result
	}

	errorType, suggestions := analyzeToolError(toolName, args, result.Error)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("TOOL FAILED: %s [%s]\n", toolName, errorType))
	b.WriteString(fmt.Sprintf("Error: %s\n", result.Error))
	b.WriteString(fmt.Sprintf("Cause: %s\n", explainToolError(errorType, toolName, args)))

	if len(suggestions) > 0 {
		b.WriteString("Fix:\n")
		for i, s := range suggestions {
			b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, s))
		}
	}

	// Extra context for browser tools
	if strings.HasPrefix(toolName, "browser_") && toolName != "browser_navigate" {
		b.WriteString("Note: browser tools require browser_navigate to be called first to open a tab.\n")
	}

	result.Error = b.String()
	return result
}

// analyzeToolError returns error type and fix suggestions.
func analyzeToolError(toolName string, args map[string]any, errorMsg string) (string, []string) {
	lower := strings.ToLower(errorMsg)
	var suggestions []string

	if strings.Contains(lower, "permission denied") || strings.Contains(lower, "access denied") {
		return "permission_denied", []string{
			"Write to workplace/ directory instead",
			"Ensure the directory is not read-only",
		}
	}

	if strings.Contains(lower, "no such file") || strings.Contains(lower, "cannot find") ||
		strings.Contains(lower, "path does not exist") || strings.Contains(lower, "system cannot find the path") {
		if path, ok := args["path"].(string); ok {
			suggestions = append(suggestions,
				fmt.Sprintf("Directory '%s' does not exist — create it first", filepath.Dir(path)),
				"Use workplace/ which always exists",
			)
		} else {
			suggestions = append(suggestions, "Path does not exist", "Use an existing directory")
		}
		return "path_not_found", suggestions
	}

	if strings.Contains(lower, "invalid argument") || strings.Contains(lower, "illegal character") {
		return "invalid_path", []string{
			"Remove special characters (* ? < > | : \") from the path",
			"Use forward slashes for directories",
		}
	}

	if strings.Contains(lower, "already exists") {
		return "file_exists", []string{
			"Read the existing file first",
			"Use a different filename",
		}
	}

	if strings.Contains(lower, "no space left") || strings.Contains(lower, "disk full") {
		return "no_space", []string{"Free up disk space or use a different location"}
	}

	if strings.Contains(lower, "security") || strings.Contains(lower, "escapes project root") {
		return "security_violation", []string{
			"Use relative paths within the project",
			"Do not use '..' to go outside the project root",
		}
	}

	// Browser-specific — check timeout/deadline BEFORE "context" to avoid misclassification.
	// "context deadline exceeded" contains "context" but is a timeout, not a missing tab.
	if strings.HasPrefix(toolName, "browser_") {
		if strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline exceeded") {
			return "browser_timeout", []string{
				"The browser operation timed out",
				"Call browser_navigate again to reopen the tab, then retry",
				"Try web_fetch instead for static pages (no Chrome needed)",
			}
		}
		if strings.Contains(lower, "call browser_navigate") || strings.Contains(lower, "no browser tab") {
			return "browser_no_tab", []string{
				"Call browser_navigate with the same task_id first to open a page",
				"Use the same task_id for all browser operations in a session",
			}
		}
		if strings.Contains(lower, "context expired") || strings.Contains(lower, "context canceled") {
			return "browser_context_expired", []string{
				"The browser tab context was cancelled — call browser_navigate again to reopen",
				"Use the same task_id when re-navigating",
			}
		}
		return "browser_error", []string{
			"Call browser_navigate first to open a tab",
			"Check the URL is valid and the page loaded successfully",
			"Use web_fetch for static pages (faster, no Chrome needed)",
		}
	}

	return "unknown_error", []string{
		"Check the error message for details",
		"Verify all parameters are correct",
		"Try a simpler operation",
	}
}

// explainToolError gives a short plain-text explanation.
func explainToolError(errorType, toolName string, args map[string]any) string {
	switch errorType {
	case "permission_denied":
		return "No write permission at that location. Use workplace/ directory."
	case "path_not_found":
		if path, ok := args["path"].(string); ok {
			return fmt.Sprintf("Path '%s' does not exist. Parent directory must be created first.", filepath.Dir(path))
		}
		return "Specified path does not exist."
	case "invalid_path":
		return "Path contains invalid characters for this file system."
	case "file_exists":
		return "A file already exists at this path."
	case "no_space":
		return "Insufficient disk space."
	case "security_violation":
		return "Path escapes the allowed project directory."
	case "browser_no_tab":
		return "No browser tab open for this task_id. Call browser_navigate first."
	case "browser_timeout":
		return "Browser operation timed out waiting for the page."
	case "browser_context_expired":
		return "Browser tab context expired (timeout or prior error). Call browser_navigate again with the same task_id."
	case "browser_error":
		return "Browser operation failed. Ensure browser_navigate was called first."
	default:
		return "Unexpected error — check the original error message."
	}
}
