// Package tools provides error enhancement for LLM feedback
package tools

import (
	"fmt"
	"path/filepath"
	"strings"
)

func EnhanceToolError(toolName string, args map[string]any, result ToolResult) ToolResult {
	if result.Success || result.Error == "" {
		return result
	}

	errorType, suggestions := analyzeToolError(toolName, args, result.Error)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n   ⚠️  %s FAILED (%s)\n", strings.ToUpper(toolName), errorType))
	b.WriteString(fmt.Sprintf("      Error: %s\n", result.Error))
	b.WriteString(fmt.Sprintf("      Cause: %s\n", explainToolError(errorType, toolName, args)))

	if len(suggestions) > 0 {
		b.WriteString("\n      💡 Suggestions:\n")
		for _, s := range suggestions {
			b.WriteString(fmt.Sprintf("         • %s\n", s))
		}
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

	// Windows: cmd.exe exits with 9009 when a command is not found (locale-agnostic),
	// or the stderr contains "is not recognized" (EN locale).
	if toolName == "exec_command" &&
		(strings.Contains(lower, "exit status 9009") || strings.Contains(lower, "is not recognized")) {
		cmd, _ := args["command"].(string)
		return "unknown_command_windows", windowsCommandSuggestions(cmd)
	}

	// Unix: command not found
	if toolName == "exec_command" &&
		(strings.Contains(lower, "command not found") ||
			strings.Contains(lower, "no such file or directory")) {
		return "unknown_command_unix", []string{
			"Verify the program is installed and in $PATH",
			"Use 'which <program>' to check availability",
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

// windowsCommandSuggestions returns fix hints for a Windows unrecognized-command error.
func windowsCommandSuggestions(cmd string) []string {
	// Map common Unix commands to their Windows equivalents
	unixToWindows := map[string]string{
		"ls":    "dir",
		"cat":   "type",
		"mv":    "move",
		"cp":    "copy",
		"rm":    "del",
		"grep":  "findstr",
		"find":  "dir /s /b",
		"head":  "more +1",
		"touch": "type nul >",
		"pwd":   "cd",
		"clear": "cls",
		"which": "where",
	}
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return []string{
			"This is a Windows environment — Unix commands (ls, cat, grep, find, head) are not available",
			"Use Windows equivalents: dir (ls), type (cat), findstr (grep), del (rm), move (mv), where (which)",
		}
	}
	lower := strings.ToLower(fields[0])
	if win, ok := unixToWindows[lower]; ok {
		return []string{
			fmt.Sprintf("'%s' is a Unix command — use '%s' instead on Windows", lower, win),
			"This environment runs on Windows: use cmd.exe commands (dir, type, move, copy, del, mkdir, findstr, where)",
		}
	}
	return []string{
		"This is a Windows environment — Unix commands (ls, cat, grep, find, head) are not available",
		"Use Windows equivalents: dir (ls), type (cat), findstr (grep), del (rm), move (mv), where (which)",
		"Verify the program is installed and in the system PATH",
	}
}

// explainToolError gives a short plain-text explanation.
func explainToolError(errorType, _ string, args map[string]any) string {
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
	case "unknown_command_windows":
		return "Command not recognized by cmd.exe. This is a Windows environment — use Windows commands."
	case "unknown_command_unix":
		return "Command not found. Verify the program is installed and available in $PATH."
	default:
		return "Unexpected error — check the original error message."
	}
}
