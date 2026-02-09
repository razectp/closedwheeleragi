// Package tools provides error enhancement for LLM feedback
package tools

import (
	"fmt"
	"path/filepath"
	"strings"
)

// EnhanceToolError enhances a tool error with detailed feedback for the LLM
// This is called AFTER tool execution to provide rich error context
func EnhanceToolError(toolName string, args map[string]any, result ToolResult) ToolResult {
	// If successful or no error, return as-is
	if result.Success || result.Error == "" {
		return result
	}

	// Analyze the error
	errorType, suggestions := analyzeToolError(toolName, args, result.Error)

	// Enhance the error message
	var enhanced strings.Builder
	enhanced.WriteString("❌ **TOOL EXECUTION FAILED**\n\n")
	enhanced.WriteString(fmt.Sprintf("**Tool:** `%s`\n", toolName))
	enhanced.WriteString(fmt.Sprintf("**Error Type:** `%s`\n\n", errorType))

	enhanced.WriteString("**Original Error:**\n")
	enhanced.WriteString(fmt.Sprintf("```\n%s\n```\n\n", result.Error))

	enhanced.WriteString("**What went wrong:**\n")
	enhanced.WriteString(explainToolError(errorType, toolName, args))
	enhanced.WriteString("\n\n")

	enhanced.WriteString("**How to fix it:**\n")
	for i, sugg := range suggestions {
		enhanced.WriteString(fmt.Sprintf("%d. %s\n", i+1, sugg))
	}
	enhanced.WriteString("\n")

	// Add context-specific suggestions
	if toolName == "write_file" {
		if path, ok := args["path"].(string); ok {
			enhanced.WriteString("**Alternative locations to try:**\n")
			enhanced.WriteString(fmt.Sprintf("- `workplace/%s`\n", filepath.Base(path)))
			enhanced.WriteString(fmt.Sprintf("- `.agi/temp/%s`\n", filepath.Base(path)))
			enhanced.WriteString(fmt.Sprintf("- `temp/%s`\n\n", filepath.Base(path)))
		}
	}

	enhanced.WriteString("**Action required:**\n")
	enhanced.WriteString("Please analyze the error above and try again with:\n")
	enhanced.WriteString("1. Corrected parameters based on suggestions\n")
	enhanced.WriteString("2. An alternative location or approach\n")
	enhanced.WriteString("3. Create necessary directories first if needed\n\n")

	enhanced.WriteString("💡 **Tip:** You can retry with different parameters immediately.\n")

	result.Error = enhanced.String()
	return result
}

// analyzeToolError analyzes the tool error and returns error type and suggestions
func analyzeToolError(toolName string, args map[string]any, errorMsg string) (string, []string) {
	errorMsgLower := strings.ToLower(errorMsg)
	suggestions := []string{}

	// Permission errors
	if strings.Contains(errorMsgLower, "permission denied") ||
		strings.Contains(errorMsgLower, "access denied") {
		suggestions = append(suggestions,
			"Try writing to a different directory with write permissions",
			"Use workplace/ directory which is designed for agent outputs",
			"Check if the file is read-only or locked",
			"On Windows, ensure the directory exists and is not protected",
		)
		return "permission_denied", suggestions
	}

	// Path not found errors
	if strings.Contains(errorMsgLower, "no such file") ||
		strings.Contains(errorMsgLower, "cannot find the path") ||
		strings.Contains(errorMsgLower, "path does not exist") ||
		strings.Contains(errorMsgLower, "system cannot find the path") {

		if path, ok := args["path"].(string); ok {
			dir := filepath.Dir(path)
			suggestions = append(suggestions,
				fmt.Sprintf("The directory '%s' doesn't exist", dir),
				"Create the parent directory first using list_files or another write_file",
				"Verify the path is correct and doesn't have typos",
				"Try using a simpler path like 'workplace/file.txt'",
			)
		} else {
			suggestions = append(suggestions,
				"The specified path doesn't exist",
				"Create parent directories first",
				"Use an existing directory",
			)
		}
		return "path_not_found", suggestions
	}

	// Invalid path characters
	if strings.Contains(errorMsgLower, "invalid argument") ||
		strings.Contains(errorMsgLower, "illegal character") ||
		strings.Contains(errorMsgLower, "invalid path") {
		suggestions = append(suggestions,
			"Remove special characters from the path: * ? < > | : \"",
			"Use only letters, numbers, hyphens, and underscores in filenames",
			"Use forward slashes (/) for directory separators",
			"Avoid spaces in filenames or use underscores instead",
		)
		return "invalid_path", suggestions
	}

	// File exists errors
	if strings.Contains(errorMsgLower, "file exists") ||
		strings.Contains(errorMsgLower, "already exists") {
		suggestions = append(suggestions,
			"Read the existing file first to see its contents",
			"Choose a different filename",
			"If you want to update, read then modify the content",
		)
		return "file_exists", suggestions
	}

	// Disk space errors
	if strings.Contains(errorMsgLower, "no space left") ||
		strings.Contains(errorMsgLower, "disk full") {
		suggestions = append(suggestions,
			"Not enough disk space available",
			"Try writing to a different location",
			"Reduce the content size",
		)
		return "no_space", suggestions
	}

	// Security violations
	if strings.Contains(errorMsgLower, "security") ||
		strings.Contains(errorMsgLower, "escapes project root") {
		suggestions = append(suggestions,
			"Path attempts to access outside the project directory",
			"Use relative paths within the project",
			"Avoid using '..' to go up directories outside the project",
		)
		return "security_violation", suggestions
	}

	// Generic error
	suggestions = append(suggestions,
		"Check the error message above for details",
		"Verify all parameters are correct",
		"Try simplifying the operation",
		"Check tool documentation with /help",
	)
	return "unknown_error", suggestions
}

// explainToolError provides human-readable explanation of error types
func explainToolError(errorType, toolName string, args map[string]any) string {
	switch errorType {
	case "permission_denied":
		return "You don't have permission to write to this location. This commonly happens with:\n" +
			"- System directories that require administrator access\n" +
			"- Read-only files or directories\n" +
			"- Files locked by another process\n" +
			"**Solution:** Use the 'workplace/' directory which is designed for your outputs."

	case "path_not_found":
		if path, ok := args["path"].(string); ok {
			dir := filepath.Dir(path)
			base := filepath.Base(path)
			return fmt.Sprintf("The path doesn't exist:\n"+
				"- Target file: `%s`\n"+
				"- Parent directory: `%s`\n"+
				"- **The parent directory must exist before writing the file.**\n"+
				"**Solution:** Either create the directory structure first, or use an existing directory like 'workplace/'.",
				base, dir)
		}
		return "The specified path doesn't exist. Parent directories must be created first."

	case "invalid_path":
		return "The path contains characters that are not allowed by the file system.\n" +
			"**Windows:** Avoid * ? < > | : \"\n" +
			"**All systems:** Use forward slashes (/) for directories\n" +
			"**Solution:** Use only letters, numbers, hyphens, underscores, and forward slashes."

	case "file_exists":
		return "A file already exists at this location.\n" +
			"**Options:**\n" +
			"1. Read the existing file first to see what's there\n" +
			"2. Choose a different filename\n" +
			"3. If updating is intended, read the file, modify it, then write back"

	case "no_space":
		return "There isn't enough disk space to complete the write operation.\n" +
			"**Solution:** Choose a different location or reduce the content size."

	case "security_violation":
		return "The path violates security constraints by attempting to access locations\n" +
			"outside the allowed project directory.\n" +
			"**Solution:** Use relative paths within the project, don't use '..' to escape the project."

	default:
		return "An unexpected error occurred. Review the original error message for details.\n" +
			"Common causes: typos in paths, incorrect parameters, or temporary system issues."
	}
}
