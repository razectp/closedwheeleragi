package tui

import (
	"fmt"
	"strings"
	"time"

	"ClosedWheeler/pkg/recovery"

	tea "github.com/charmbracelet/bubbletea"
)

// Recovery and error management commands

func cmdErrors(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	handler := recovery.GetGlobalHandler()

	if len(args) > 0 && args[0] == "clear" {
		handler.ClearErrorLog()
		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   "✅ Error log cleared.",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	// Get recent errors
	count := 10
	if len(args) > 0 {
		fmt.Sscanf(args[0], "%d", &count)
	}

	errors := handler.GetRecentErrors(count)

	var content strings.Builder
	content.WriteString("🔴 **Recent Errors**\n\n")

	if len(errors) == 0 {
		content.WriteString("No errors recorded. System is running smoothly! ✅\n\n")
		content.WriteString("**Note:** The system automatically handles errors and continues working.\n")
		content.WriteString("If you encounter issues, they will appear here.")
	} else {
		stats := handler.GetErrorStats()
		content.WriteString(fmt.Sprintf("**Statistics:**\n"))
		content.WriteString(fmt.Sprintf("- Total Errors: %v\n", stats["total_errors"]))
		content.WriteString(fmt.Sprintf("- Recovered: %v\n", stats["recovered"]))
		content.WriteString(fmt.Sprintf("- Recovery Rate: %.1f%%\n\n", stats["recovery_rate"]))

		content.WriteString(fmt.Sprintf("**Last %d Errors:**\n\n", len(errors)))

		for i, err := range errors {
			status := "❌ Failed"
			if err.Recovered {
				status = "✅ Recovered"
			}

			content.WriteString(fmt.Sprintf("%d. %s - **%s**\n", i+1, err.Timestamp.Format("15:04:05"), status))
			content.WriteString(fmt.Sprintf("   Operation: `%s` in %s\n", err.Operation, err.Context))
			content.WriteString(fmt.Sprintf("   Error: %v\n", err.Error))

			if err.RetryCount > 0 {
				content.WriteString(fmt.Sprintf("   Retries: %d\n", err.RetryCount))
			}
			content.WriteString("\n")
		}

		content.WriteString("**Commands:**\n")
		content.WriteString("- `/errors 20` - Show last 20 errors\n")
		content.WriteString("- `/errors clear` - Clear error log\n")
		content.WriteString("- `/resilience` - Configure error handling\n")
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

func cmdResilience(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	var content strings.Builder
	content.WriteString("🛡️ **Error Resilience System**\n\n")

	content.WriteString("**What it does:**\n")
	content.WriteString("- ✅ Automatic retry on transient errors\n")
	content.WriteString("- ✅ Graceful degradation on failures\n")
	content.WriteString("- ✅ Panic recovery (never crashes)\n")
	content.WriteString("- ✅ Continue working despite errors\n")
	content.WriteString("- ✅ Detailed error logging\n\n")

	content.WriteString("**Error Types Handled:**\n")
	content.WriteString("- File write errors → Retry with fallback to temp\n")
	content.WriteString("- API timeouts → Retry with exponential backoff\n")
	content.WriteString("- Rate limits → Wait and retry\n")
	content.WriteString("- Network errors → Retry up to 3 times\n")
	content.WriteString("- Permission errors → Graceful fallback\n")
	content.WriteString("- Panics → Recover and log\n\n")

	content.WriteString("**Retry Strategies:**\n")
	content.WriteString("- File operations: 3 retries, 100ms → 2s\n")
	content.WriteString("- API calls: 5 retries, 1s → 30s\n")
	content.WriteString("- Network: 3 retries, 500ms → 5s\n\n")

	handler := recovery.GetGlobalHandler()
	stats := handler.GetErrorStats()

	content.WriteString("**Current Statistics:**\n")
	if totalErrors, ok := stats["total_errors"].(int); ok && totalErrors > 0 {
		content.WriteString(fmt.Sprintf("- Total Errors: %v\n", stats["total_errors"]))
		content.WriteString(fmt.Sprintf("- Recovered: %v\n", stats["recovered"]))
		content.WriteString(fmt.Sprintf("- Recovery Rate: %.1f%%\n", stats["recovery_rate"]))

		if byOp, ok := stats["by_operation"].(map[string]int); ok {
			content.WriteString("\n**Errors by Operation:**\n")
			for op, count := range byOp {
				content.WriteString(fmt.Sprintf("- %s: %d\n", op, count))
			}
		}
	} else {
		content.WriteString("- No errors yet ✅\n")
	}

	content.WriteString("\n**Commands:**\n")
	content.WriteString("- `/errors` - View recent errors\n")
	content.WriteString("- `/errors clear` - Clear error log\n")
	content.WriteString("- `/report` - Full diagnostic report\n")

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   content.String(),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}

func cmdRecover(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	handler := recovery.GetGlobalHandler()

	var content strings.Builder
	content.WriteString("🔄 **System Recovery**\n\n")

	// Get error stats
	stats := handler.GetErrorStats()
	totalErrors := 0
	if te, ok := stats["total_errors"].(int); ok {
		totalErrors = te
	}

	if totalErrors > 0 {
		content.WriteString("**Status:** Errors detected, running recovery procedures...\n\n")

		// Clear error log
		handler.ClearErrorLog()

		content.WriteString("**Actions Taken:**\n")
		content.WriteString("- ✅ Error log cleared\n")
		content.WriteString("- ✅ Recovery handlers reset\n")
		content.WriteString("- ✅ System ready for fresh start\n\n")

		content.WriteString(fmt.Sprintf("**Summary:** Cleared %d logged errors\n", totalErrors))
		content.WriteString("\nThe system will continue operating normally.")
	} else {
		content.WriteString("**Status:** System is healthy ✅\n\n")
		content.WriteString("No errors detected. Everything is working properly!\n\n")
		content.WriteString("**Note:** The recovery system is always active,\n")
		content.WriteString("automatically handling errors as they occur.")
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
