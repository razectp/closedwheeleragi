// Package health provides project health monitoring and reflection capabilities.
package health

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Status represents the health status of the project
type Status struct {
	Timestamp       time.Time
	BuildStatus     string // "passing", "failing", "unknown"
	BuildError      string
	TestStatus      string // "passing", "failing", "skipped", "unknown"
	TestError       string
	TestCoverage    string
	PendingTasks    int
	GitStatus       string
	GitBranch       string
	GitUncommitted  int
	MemoryUsage     MemoryStats
	Warnings        []string
	Recommendations []string
}

// MemoryStats holds memory system statistics
type MemoryStats struct {
	ShortTerm int
	Working   int
	LongTerm  int
}

// Checker performs health checks on the project
type Checker struct {
	projectPath string
	testCommand string
}

// NewChecker creates a new health checker
func NewChecker(projectPath, testCommand string) *Checker {
	if testCommand == "" {
		testCommand = "go test ./..."
	}

	return &Checker{
		projectPath: projectPath,
		testCommand: testCommand,
	}
}

// Check performs a comprehensive health check
func (c *Checker) Check() *Status {
	status := &Status{
		Timestamp:       time.Now(),
		BuildStatus:     "unknown",
		TestStatus:      "unknown",
		Warnings:        []string{},
		Recommendations: []string{},
	}

	// Check build status
	c.checkBuild(status)

	// Check test status
	c.checkTests(status)

	// Check git status
	c.checkGit(status)

	// Check pending tasks
	c.checkTasks(status)

	// Generate recommendations
	c.generateRecommendations(status)

	return status
}

// checkBuild checks if the project builds successfully
func (c *Checker) checkBuild(status *Status) {
	// Determine build command based on project type
	buildCmd := c.detectBuildCommand()
	if buildCmd == "" {
		status.BuildStatus = "skipped"
		return
	}

	parts := strings.Fields(buildCmd)
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = c.projectPath

	output, err := cmd.CombinedOutput()

	if err != nil {
		status.BuildStatus = "failing"
		status.BuildError = strings.TrimSpace(string(output))
		status.Warnings = append(status.Warnings, "Build is failing")
	} else {
		status.BuildStatus = "passing"
	}
}

// checkTests runs tests and checks status
func (c *Checker) checkTests(status *Status) {
	// Check if tests should be run
	if c.testCommand == "skip" || c.testCommand == "" {
		status.TestStatus = "skipped"
		return
	}

	parts := strings.Fields(c.testCommand)
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = c.projectPath

	// Set timeout for tests (30 seconds)
	timer := time.AfterFunc(30*time.Second, func() {
		cmd.Process.Kill()
	})
	defer timer.Stop()

	output, err := cmd.CombinedOutput()

	if err != nil {
		status.TestStatus = "failing"
		status.TestError = strings.TrimSpace(string(output))
		status.Warnings = append(status.Warnings, "Tests are failing")
	} else {
		status.TestStatus = "passing"

		// Try to extract coverage if available
		outputStr := string(output)
		if strings.Contains(outputStr, "coverage:") {
			lines := strings.Split(outputStr, "\n")
			for _, line := range lines {
				if strings.Contains(line, "coverage:") {
					status.TestCoverage = strings.TrimSpace(line)
					break
				}
			}
		}
	}
}

// checkGit checks git repository status
func (c *Checker) checkGit(status *Status) {
	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		status.GitStatus = "not a git repository"
		return
	}

	// Get current branch
	branchCmd := exec.Command("git", "branch", "--show-current")
	branchCmd.Dir = c.projectPath
	branchOutput, err := branchCmd.Output()
	if err == nil {
		status.GitBranch = strings.TrimSpace(string(branchOutput))
	}

	// Get uncommitted changes count
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = c.projectPath
	statusOutput, err := statusCmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(statusOutput)), "\n")
		if len(lines) > 0 && lines[0] != "" {
			status.GitUncommitted = len(lines)
		}
		status.GitStatus = "clean"
		if status.GitUncommitted > 0 {
			status.GitStatus = "uncommitted changes"
		}
	}
}

// checkTasks counts pending tasks
func (c *Checker) checkTasks(status *Status) {
	taskPath := filepath.Join(c.projectPath, "workplace", "task.md")

	content, err := os.ReadFile(taskPath)
	if err != nil {
		// Also check root task.md
		taskPath = filepath.Join(c.projectPath, "task.md")
		content, err = os.ReadFile(taskPath)
		if err != nil {
			return
		}
	}

	taskStr := string(content)
	pending := strings.Count(taskStr, "- [ ]") + strings.Count(taskStr, "- [/]")
	status.PendingTasks = pending
}

// generateRecommendations generates actionable recommendations
func (c *Checker) generateRecommendations(status *Status) {
	if status.BuildStatus == "failing" {
		status.Recommendations = append(status.Recommendations,
			"🔧 Fix build errors before proceeding with new features")
	}

	if status.TestStatus == "failing" {
		status.Recommendations = append(status.Recommendations,
			"🧪 Address failing tests to maintain code quality")
	}

	if status.GitUncommitted > 10 {
		status.Recommendations = append(status.Recommendations,
			"📝 Consider committing changes - you have many uncommitted files")
	}

	if status.PendingTasks > 20 {
		status.Recommendations = append(status.Recommendations,
			"📋 Review and prioritize tasks - backlog is getting large")
	}

	if len(status.Recommendations) == 0 {
		status.Recommendations = append(status.Recommendations,
			"✅ Project health looks good!")
	}
}

// detectBuildCommand detects the appropriate build command for the project
func (c *Checker) detectBuildCommand() string {
	// Check for Go project
	if _, err := os.Stat(filepath.Join(c.projectPath, "go.mod")); err == nil {
		return "go build ./..."
	}

	// Check for Node.js project
	if _, err := os.Stat(filepath.Join(c.projectPath, "package.json")); err == nil {
		return "npm run build"
	}

	// Check for Python project
	if _, err := os.Stat(filepath.Join(c.projectPath, "setup.py")); err == nil {
		return "python setup.py build"
	}

	// Check for Rust project
	if _, err := os.Stat(filepath.Join(c.projectPath, "Cargo.toml")); err == nil {
		return "cargo build"
	}

	return ""
}

// FormatReport generates a formatted health report
func (c *Checker) FormatReport(status *Status) string {
	var sb strings.Builder

	sb.WriteString("# 🏥 Project Health Report\n\n")
	sb.WriteString(fmt.Sprintf("**Timestamp:** %s\n\n", status.Timestamp.Format("2006-01-02 15:04:05")))

	// Build Status
	buildEmoji := c.statusEmoji(status.BuildStatus)
	sb.WriteString(fmt.Sprintf("## 🔨 Build Status: %s %s\n", buildEmoji, status.BuildStatus))
	if status.BuildError != "" {
		sb.WriteString(fmt.Sprintf("```\n%s\n```\n", truncate(status.BuildError, 500)))
	}
	sb.WriteString("\n")

	// Test Status
	testEmoji := c.statusEmoji(status.TestStatus)
	sb.WriteString(fmt.Sprintf("## 🧪 Test Status: %s %s\n", testEmoji, status.TestStatus))
	if status.TestCoverage != "" {
		sb.WriteString(fmt.Sprintf("**Coverage:** %s\n", status.TestCoverage))
	}
	if status.TestError != "" {
		sb.WriteString(fmt.Sprintf("```\n%s\n```\n", truncate(status.TestError, 500)))
	}
	sb.WriteString("\n")

	// Git Status
	sb.WriteString(fmt.Sprintf("## 📦 Git Status\n"))
	sb.WriteString(fmt.Sprintf("- **Branch:** %s\n", status.GitBranch))
	sb.WriteString(fmt.Sprintf("- **Status:** %s\n", status.GitStatus))
	if status.GitUncommitted > 0 {
		sb.WriteString(fmt.Sprintf("- **Uncommitted files:** %d\n", status.GitUncommitted))
	}
	sb.WriteString("\n")

	// Tasks
	sb.WriteString(fmt.Sprintf("## 📋 Tasks\n"))
	sb.WriteString(fmt.Sprintf("- **Pending tasks:** %d\n\n", status.PendingTasks))

	// Warnings
	if len(status.Warnings) > 0 {
		sb.WriteString("## ⚠️ Warnings\n")
		for _, warning := range status.Warnings {
			sb.WriteString(fmt.Sprintf("- %s\n", warning))
		}
		sb.WriteString("\n")
	}

	// Recommendations
	sb.WriteString("## 💡 Recommendations\n")
	for _, rec := range status.Recommendations {
		sb.WriteString(fmt.Sprintf("- %s\n", rec))
	}

	return sb.String()
}

// statusEmoji returns an emoji for a given status
func (c *Checker) statusEmoji(status string) string {
	switch status {
	case "passing":
		return "✅"
	case "failing":
		return "❌"
	case "skipped", "unknown":
		return "⚪"
	default:
		return "❔"
	}
}

// truncate truncates a string to a maximum length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (truncated)"
}
