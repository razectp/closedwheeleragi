package utils

import (
	"regexp"
	"strings"
)

// SensitivePatterns contains regex patterns for sensitive data
var SensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(api[_-]?key|secret|token|password|auth)\s*[:=]\s*['"]?([a-zA-Z0-9_\-+/=]{8,})['"]?`),
	regexp.MustCompile(`(?i)(bearer\s+)([a-zA-Z0-9_\-+/=]{20,})`),
	regexp.MustCompile(`(?i)(sk-[a-zA-Z0-9]{20,})`), // OpenAI keys
	regexp.MustCompile(`(?i)(x-api-key:\s*)([a-zA-Z0-9_\-+/=]{8,})`),
	regexp.MustCompile(`(?i)(authorization:\s*bearer\s+)([a-zA-Z0-9_\-+/=]{20,})`),
}

// SanitizeLog removes sensitive information from log messages
func SanitizeLog(message string) string {
	result := message
	
	for _, pattern := range SensitivePatterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			// Replace sensitive values with asterisks while preserving the key name
			parts := strings.SplitN(match, ":", 2)
			if len(parts) == 2 {
				return parts[0] + ": ***REDACTED***"
			}
			
			// Handle other patterns
			if strings.Contains(strings.ToLower(match), "sk-") {
				return "sk-***REDACTED***"
			}
			
			return "***REDACTED***"
		})
	}
	
	return result
}

// SanitizeForDebug removes sensitive information but preserves structure for debugging
func SanitizeForDebug(message string) string {
	result := message
	
	for _, pattern := range SensitivePatterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			// Replace with length indicator for debugging
			parts := strings.SplitN(match, ":", 2)
			if len(parts) == 2 {
				sensitiveValue := parts[1]
				cleanValue := strings.Trim(sensitiveValue, `"' `)
				return parts[0] + ": " + strings.Repeat("*", len(cleanValue))
			}
			
			if strings.Contains(strings.ToLower(match), "sk-") {
				return "sk-" + strings.Repeat("*", 20)
			}
			
			return strings.Repeat("*", len(match))
		})
	}
	
	return result
}
