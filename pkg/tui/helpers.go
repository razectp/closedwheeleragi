package tui

import (
	"fmt"
)

// toInt converts an interface{} to an int with proper error handling
func toInt(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case string:
		if i, err := fmt.Sscanf(val, "%d", new(int)); err == nil && i == 1 {
			var result int
			fmt.Sscanf(val, "%d", &result)
			return result
		}
		return 0
	default:
		return 0
	}
}

// formatK formats numbers with K/M suffixes for better readability
func formatK(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// truncateText truncates text to fit within max length
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		return text[:maxLen]
	}
	return text[:maxLen-3] + "..."
}
