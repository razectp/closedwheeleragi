package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Tool retry and diagnostics commands

func cmdToolRetries(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if m.toolRetryWrapper == nil {
		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   "⚠️ Tool retry system is not initialized.",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	stats := m.toolRetryWrapper.GetRetryStats()

	var content strings.Builder
	content.WriteString("🔄 **Intelligent Tool Retry System**\n\n")

	totalAttempts := 0
	totalSuccess := 0
	successRate := 0.0

	if ta, ok := stats["total_attempts"].(int); ok {
		totalAttempts = ta
	}
	if ts, ok := stats["total_success"].(int); ok {
		totalSuccess = ts
	}
	if sr, ok := stats["success_rate"].(float64); ok {
		successRate = sr
	}

	content.WriteString("**Overview:**\n")
	content.WriteString(fmt.Sprintf("- Total Attempts: %d\n", totalAttempts))
	content.WriteString(fmt.Sprintf("- Successful: %d\n", totalSuccess))
	content.WriteString(fmt.Sprintf("- Success Rate: %.1f%%\n\n", successRate))

	if totalAttempts == 0 {
		content.WriteString("No tool retry attempts recorded yet.\n\n")
		content.WriteString("**What this system does:**\n")
		content.WriteString("- 🔍 Analyzes tool execution errors in detail\n")
		content.WriteString("- 💡 Provides intelligent suggestions to the LLM\n")
		content.WriteString("- 🔄 Allows automatic retry with corrected parameters\n")
		content.WriteString("- 📊 Tracks success rates and common error patterns\n\n")

		content.WriteString("**Supported Error Types:**\n")
		content.WriteString("- `permission_denied` - No write access to location\n")
		content.WriteString("- `path_not_found` - Directory or file doesn't exist\n")
		content.WriteString("- `invalid_path` - Invalid characters in path\n")
		content.WriteString("- `file_exists` - File already exists at location\n")
		content.WriteString("- `no_space` - Not enough disk space\n")
		content.WriteString("- `security_violation` - Path escapes project root\n\n")

		content.WriteString("**Example Flow:**\n")
		content.WriteString("1. LLM tries: `write_file(path=\"/restricted/file.txt\")`\n")
		content.WriteString("2. Error: Permission denied\n")
		content.WriteString("3. System analyzes: Suggests alternative paths\n")
		content.WriteString("4. LLM retries: `write_file(path=\"workplace/file.txt\")`\n")
		content.WriteString("5. Success! ✅\n")
	} else {
		if byTool, ok := stats["by_tool"].(map[string]int); ok && len(byTool) > 0 {
			content.WriteString("**Attempts by Tool:**\n")
			for tool, count := range byTool {
				content.WriteString(fmt.Sprintf("- `%s`: %d attempts\n", tool, count))
			}
			content.WriteString("\n")
		}

		if byError, ok := stats["by_error_type"].(map[string]int); ok && len(byError) > 0 {
			content.WriteString("**Errors by Type:**\n")
			for errType, count := range byError {
				content.WriteString(fmt.Sprintf("- `%s`: %d occurrences\n", errType, count))
			}
			content.WriteString("\n")
		}
	}

	content.WriteString("**Commands:**\n")
	content.WriteString("- `/tool-retries` - Show this report\n")
	content.WriteString("- `/retry-mode [on|off]` - Toggle feedback mode\n")

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   content.String(),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}

func cmdRetryMode(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if m.toolRetryWrapper == nil {
		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   "⚠️ Tool retry system is not initialized.",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	var content strings.Builder

	if len(args) == 0 {
		// Show current status
		content.WriteString("🔄 **Retry Feedback Mode**\n\n")
		content.WriteString("When enabled, the system provides detailed feedback to the LLM\n")
		content.WriteString("about tool execution errors, allowing intelligent retry attempts.\n\n")
		content.WriteString("**Usage:**\n")
		content.WriteString("- `/retry-mode on` - Enable detailed feedback\n")
		content.WriteString("- `/retry-mode off` - Disable detailed feedback\n")
	} else {
		mode := strings.ToLower(args[0])
		switch mode {
		case "on", "enable", "enabled", "true":
			m.toolRetryWrapper.EnableFeedbackMode(true)
			content.WriteString("✅ **Retry feedback mode enabled**\n\n")
			content.WriteString("The LLM will now receive detailed error analysis and suggestions\n")
			content.WriteString("when tool executions fail, enabling intelligent retry attempts.\n\n")
			content.WriteString("**Benefits:**\n")
			content.WriteString("- 🎯 More accurate retries\n")
			content.WriteString("- 💡 Learn from previous failures\n")
			content.WriteString("- 🚀 Higher success rate\n")
			content.WriteString("- 📚 Better error understanding\n")

		case "off", "disable", "disabled", "false":
			m.toolRetryWrapper.EnableFeedbackMode(false)
			content.WriteString("⚠️ **Retry feedback mode disabled**\n\n")
			content.WriteString("The LLM will receive basic error messages without detailed analysis.\n")
			content.WriteString("This may result in lower retry success rates.\n")

		default:
			content.WriteString("❌ Invalid option. Use:\n")
			content.WriteString("- `/retry-mode on` - Enable feedback\n")
			content.WriteString("- `/retry-mode off` - Disable feedback\n")
		}
	}

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   content.String(),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}
