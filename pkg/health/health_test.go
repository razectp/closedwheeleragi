package health

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewChecker(t *testing.T) {
	checker := NewChecker(".", "go test ./...")

	if checker == nil {
		t.Fatal("NewChecker returned nil")
	}

	if checker.projectPath != "." {
		t.Errorf("Expected projectPath '.', got '%s'", checker.projectPath)
	}

	if checker.testCommand != "go test ./..." {
		t.Errorf("Expected testCommand 'go test ./...', got '%s'", checker.testCommand)
	}
}

func TestNewChecker_DefaultTestCommand(t *testing.T) {
	checker := NewChecker(".", "")

	if checker.testCommand != "go test ./..." {
		t.Errorf("Expected default testCommand 'go test ./...', got '%s'", checker.testCommand)
	}
}

func TestChecker_DetectBuildCommand(t *testing.T) {
	tmpDir := t.TempDir()
	checker := NewChecker(tmpDir, "go test ./...")

	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"Go project", "go.mod", "go build ./..."},
		{"Node project", "package.json", "npm run build"},
		{"Python project", "setup.py", "python setup.py build"},
		{"Rust project", "Cargo.toml", "cargo build"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create marker file
			markerPath := filepath.Join(tmpDir, tt.filename)
			os.WriteFile(markerPath, []byte("test"), 0644)

			cmd := checker.detectBuildCommand()
			if cmd != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, cmd)
			}

			// Clean up
			os.Remove(markerPath)
		})
	}
}

func TestChecker_DetectBuildCommand_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	checker := NewChecker(tmpDir, "go test ./...")

	cmd := checker.detectBuildCommand()
	if cmd != "" {
		t.Errorf("Expected empty string, got '%s'", cmd)
	}
}

func TestChecker_Check(t *testing.T) {
	tmpDir := t.TempDir()
	checker := NewChecker(tmpDir, "skip") // Skip tests for this test

	status := checker.Check()

	if status == nil {
		t.Fatal("Check returned nil")
	}

	if status.Timestamp.IsZero() {
		t.Error("Timestamp not set")
	}

	if status.BuildStatus == "" {
		t.Error("BuildStatus not set")
	}

	if status.TestStatus != "skipped" {
		t.Errorf("Expected TestStatus 'skipped', got '%s'", status.TestStatus)
	}
}

func TestChecker_CheckTasks(t *testing.T) {
	tmpDir := t.TempDir()
	workplaceDir := filepath.Join(tmpDir, "workplace")
	os.MkdirAll(workplaceDir, 0755)

	// Create task.md with pending tasks
	taskPath := filepath.Join(workplaceDir, "task.md")
	taskContent := `# Tasks
- [ ] Task 1
- [x] Task 2
- [ ] Task 3
- [/] Task 4
`
	os.WriteFile(taskPath, []byte(taskContent), 0644)

	checker := NewChecker(tmpDir, "skip")
	status := checker.Check()

	// Should count: [ ] and [/]
	expectedTasks := 3 // Task 1, Task 3, Task 4
	if status.PendingTasks != expectedTasks {
		t.Errorf("Expected %d pending tasks, got %d", expectedTasks, status.PendingTasks)
	}
}

func TestChecker_FormatReport(t *testing.T) {
	checker := NewChecker(".", "skip")

	status := &Status{
		BuildStatus:     "passing",
		TestStatus:      "passing",
		GitStatus:       "clean",
		GitBranch:       "main",
		GitUncommitted:  0,
		PendingTasks:    5,
		Warnings:        []string{"Warning 1", "Warning 2"},
		Recommendations: []string{"Rec 1"},
	}

	report := checker.FormatReport(status)

	if !strings.Contains(report, "# 🏥 Project Health Report") {
		t.Error("Report missing title")
	}

	if !strings.Contains(report, "Build Status: ✅ passing") {
		t.Error("Report missing build status")
	}

	if !strings.Contains(report, "Test Status: ✅ passing") {
		t.Error("Report missing test status")
	}

	if !strings.Contains(report, "Branch:** main") {
		t.Error("Report missing git branch")
	}

	if !strings.Contains(report, "Pending tasks:** 5") {
		t.Error("Report missing tasks count")
	}

	if !strings.Contains(report, "Warning 1") {
		t.Error("Report missing warnings")
	}

	if !strings.Contains(report, "Rec 1") {
		t.Error("Report missing recommendations")
	}
}

func TestChecker_StatusEmoji(t *testing.T) {
	checker := NewChecker(".", "skip")

	tests := []struct {
		status   string
		expected string
	}{
		{"passing", "✅"},
		{"failing", "❌"},
		{"skipped", "⚪"},
		{"unknown", "⚪"},
		{"other", "❔"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			emoji := checker.statusEmoji(tt.status)
			if emoji != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, emoji)
			}
		})
	}
}

func TestChecker_GenerateRecommendations(t *testing.T) {
	checker := NewChecker(".", "skip")

	tests := []struct {
		name     string
		status   *Status
		expected string
	}{
		{
			name: "Build failing",
			status: &Status{
				BuildStatus:     "failing",
				Recommendations: []string{},
			},
			expected: "Fix build errors",
		},
		{
			name: "Tests failing",
			status: &Status{
				TestStatus:      "failing",
				Recommendations: []string{},
			},
			expected: "Address failing tests",
		},
		{
			name: "Many uncommitted files",
			status: &Status{
				GitUncommitted:  15,
				Recommendations: []string{},
			},
			expected: "Consider committing changes",
		},
		{
			name: "Many pending tasks",
			status: &Status{
				PendingTasks:    25,
				Recommendations: []string{},
			},
			expected: "Review and prioritize tasks",
		},
		{
			name: "All good",
			status: &Status{
				BuildStatus:     "passing",
				TestStatus:      "passing",
				Recommendations: []string{},
			},
			expected: "Project health looks good",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker.generateRecommendations(tt.status)

			found := false
			for _, rec := range tt.status.Recommendations {
				if strings.Contains(rec, tt.expected) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected recommendation containing '%s', got %v", tt.expected, tt.status.Recommendations)
			}
		})
	}
}
