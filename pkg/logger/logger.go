package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Level string

const (
	DEBUG Level = "DEBUG"
	INFO  Level = "INFO"
	WARN  Level = "WARN"
	ERROR Level = "ERROR"
)

type Logger struct {
	filePath string
}

func New(storagePath string) (*Logger, error) {
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, err
	}

	logPath := filepath.Join(storagePath, "debug.log")
	return &Logger{filePath: logPath}, nil
}

func (l *Logger) log(level Level, message string) {
	message = sanitize(message)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("[%s] %s: %s\n", timestamp, level, message)

	// Write to log file only — never to stderr (it corrupts the TUI)
	f, err := os.OpenFile(l.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return // silently fail — can't write to stderr during TUI
	}
	defer f.Close()

	f.WriteString(entry)
}

func (l *Logger) Debug(format string, v ...any) {
	l.log(DEBUG, fmt.Sprintf(format, v...))
}

func (l *Logger) Info(format string, v ...any) {
	l.log(INFO, fmt.Sprintf(format, v...))
}

func (l *Logger) Warn(format string, v ...any) {
	l.log(WARN, fmt.Sprintf(format, v...))
}

func (l *Logger) Error(format string, v ...any) {
	l.log(ERROR, fmt.Sprintf(format, v...))
}

func (l *Logger) GetLastLines(n int) string {
	content, err := os.ReadFile(l.filePath)
	if err != nil {
		return "Error reading log file"
	}

	lines := splitLines(string(content))
	if len(lines) <= n {
		return string(content)
	}

	return joinLines(lines[len(lines)-n:])
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func joinLines(lines []string) string {
	var sb strings.Builder
	for i, line := range lines {
		sb.WriteString(line)
		if i < len(lines)-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

var (
	openaiKeyRegex   = regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`)
	telegramBotRegex = regexp.MustCompile(`[0-9]{8,10}:[a-zA-Z0-9_-]{35}`)
)

func sanitize(s string) string {
	s = openaiKeyRegex.ReplaceAllStringFunc(s, func(m string) string {
		if len(m) <= 8 {
			return "****"
		}
		return m[:4] + "..." + m[len(m)-4:]
	})

	s = telegramBotRegex.ReplaceAllStringFunc(s, func(m string) string {
		parts := strings.Split(m, ":")
		if len(parts) != 2 {
			return "****"
		}
		return parts[0] + ":****"
	})

	return s
}
