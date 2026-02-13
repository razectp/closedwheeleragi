package logger

import (
	"ClosedWheeler/pkg/utils"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Level string

const (
	DEBUG Level = "DEBUG"
	INFO  Level = "INFO"
	WARN  Level = "WARN"
	ERROR Level = "ERROR"
)

// Logger wraps zap.SugaredLogger for compatibility
type Logger struct {
	sugar    *zap.SugaredLogger
	filePath string
}

func New(storagePath string) (*Logger, error) {
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, err
	}

	logPath := filepath.Join(storagePath, "debug.log")

	// Open log file
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	// Create zap logger for file output
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		MessageKey:     "message",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	}

	fileEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	fileCore := zapcore.NewCore(fileEncoder, zapcore.AddSync(logFile), zapcore.DebugLevel)
	zapLogger := zap.New(fileCore, zap.AddCaller(), zap.AddCallerSkip(1))

	return &Logger{
		sugar:    zapLogger.Sugar(),
		filePath: logPath,
	}, nil
}

func (l *Logger) log(level Level, message string) {
	message = sanitize(message)

	switch level {
	case DEBUG:
		l.sugar.Debug(message)
	case INFO:
		l.sugar.Info(message)
	case WARN:
		l.sugar.Warn(message)
	case ERROR:
		l.sugar.Error(message)
	}
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

func (l *Logger) Sync() error {
	return l.sugar.Sync()
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

func sanitize(s string) string {
	return utils.SanitizeLog(s)
}
