package logger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewLogger(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "logger_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger, err := New(tempDir)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	if logger == nil {
		t.Fatal("Logger is nil")
	}

	if logger.sugar == nil {
		t.Fatal("Logger sugar is nil")
	}
}

func TestLoggerLevels(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "logger_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger, err := New(tempDir)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test all log levels
	logger.Debug("Debug message: %s", "test")
	logger.Info("Info message: %s", "test")
	logger.Warn("Warning message: %s", "test")
	logger.Error("Error message: %s", "test")

	// Sync to ensure logs are written
	err = logger.Sync()
	if err != nil {
		t.Errorf("Failed to sync logger: %v", err)
	}
}

func TestGetLastLines(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "logger_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger, err := New(tempDir)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Write some log messages
	logger.Info("Line 1")
	logger.Info("Line 2")
	logger.Info("Line 3")
	logger.Sync()

	// Test getting last lines
	last2 := logger.GetLastLines(2)
	if last2 == "" {
		t.Error("Expected some content when getting last lines")
	}

	// Test getting more lines than exist
	last10 := logger.GetLastLines(10)
	if last10 == "" {
		t.Error("Expected some content when getting more lines than exist")
	}
}

func TestLoggerFileCreation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "logger_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger, err := New(tempDir)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logPath := filepath.Join(tempDir, "debug.log")

	// Write a log message
	logger.Info("Test message")
	logger.Sync()

	// Check if file exists and has content
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Log file is empty")
	}

	contentStr := string(content)
	if !contains(contentStr, "Test message") {
		t.Errorf("Log file doesn't contain expected message. Content: %s", contentStr)
	}
}

func TestLoggerSync(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "logger_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger, err := New(tempDir)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	err = logger.Sync()
	if err != nil {
		t.Errorf("Sync failed: %v", err)
	}
}

func TestMultipleLoggers(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "logger_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger1, err := New(tempDir)
	if err != nil {
		t.Fatalf("Failed to create logger1: %v", err)
	}

	logger2, err := New(tempDir)
	if err != nil {
		t.Fatalf("Failed to create logger2: %v", err)
	}

	logger1.Info("Logger 1 message")
	logger2.Info("Logger 2 message")

	logger1.Sync()
	logger2.Sync()

	// Both loggers should write to the same file
	logPath := filepath.Join(tempDir, "debug.log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !contains(contentStr, "Logger 1 message") {
		t.Error("Logger 1 message not found")
	}
	if !contains(contentStr, "Logger 2 message") {
		t.Error("Logger 2 message not found")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}
