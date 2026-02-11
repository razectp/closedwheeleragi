// Package roadmap provides strategic long-term planning capabilities.
// The roadmap is stored in workplace/roadmap.md as a visible, editable document.
package roadmap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Goal represents a strategic goal
type Goal struct {
	ID           string
	Title        string
	Description  string
	Status       string // "planned", "in-progress", "blocked", "completed"
	Priority     string // "high", "medium", "low"
	DueDate      *time.Time
	Dependencies []string // IDs of goals this depends on
	Tags         []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Milestone represents a major achievement
type Milestone struct {
	Title       string
	Description string
	Goals       []string // Goal IDs
	TargetDate  *time.Time
	CompletedAt *time.Time
}

// Roadmap manages strategic planning
type Roadmap struct {
	projectPath string
	roadmapPath string
	mu          sync.RWMutex
}

// NewRoadmap creates a new roadmap instance
func NewRoadmap(projectPath string) *Roadmap {
	return &Roadmap{
		projectPath: projectPath,
		roadmapPath: filepath.Join(projectPath, "roadmap.md"),
	}
}

// Initialize creates the roadmap.md file if it doesn't exist
func (r *Roadmap) Initialize() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Ensure parent directory exists
	parentDir := filepath.Dir(r.roadmapPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create roadmap directory: %w", err)
	}

	// Check if roadmap.md exists
	if _, err := os.Stat(r.roadmapPath); os.IsNotExist(err) {
		initialContent := `# 🗺️ Strategic Roadmap

Strategic long-term planning for the project.

## 🎯 Vision

<!-- Describe the long-term vision of the project -->

## 🏆 Milestones

<!-- Important milestones and their goals -->

## 📊 Strategic Objectives

### 🔴 High Priority

<!-- High priority objectives -->

### 🟡 Medium Priority

<!-- Medium priority objectives -->

### 🟢 Low Priority

<!-- Low priority objectives -->

## ✅ Completed

<!-- Already completed objectives -->

## 🚫 Blocked

<!-- Blocked objectives and reasons -->

---

*Last update: ` + time.Now().Format("2006-01-02 15:04:05") + `*
`
		return os.WriteFile(r.roadmapPath, []byte(initialContent), 0644)
	}

	return nil
}

// AddGoal adds a new strategic goal to the roadmap
func (r *Roadmap) AddGoal(goal Goal) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Read current content
	content, err := os.ReadFile(r.roadmapPath)
	if err != nil {
		return fmt.Errorf("failed to read roadmap.md: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	// Determine section based on priority
	sectionMarker := r.getPrioritySection(goal.Priority)
	sectionIdx := r.findSectionIndex(lines, sectionMarker)

	if sectionIdx == -1 {
		return fmt.Errorf("section not found for priority: %s", goal.Priority)
	}

	// Format the new goal
	formattedGoal := r.formatGoal(goal)

	// Insert after section header and comment
	insertIdx := sectionIdx + 2

	// Insert the goal
	newLines := append(lines[:insertIdx], append([]string{formattedGoal, ""}, lines[insertIdx:]...)...)

	// Update timestamp
	r.updateTimestamp(newLines)

	// Write back
	return os.WriteFile(r.roadmapPath, []byte(strings.Join(newLines, "\n")), 0644)
}

// UpdateGoalStatus updates the status of a goal (moves it between sections if needed)
func (r *Roadmap) UpdateGoalStatus(goalID, newStatus string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	content, err := os.ReadFile(r.roadmapPath)
	if err != nil {
		return fmt.Errorf("failed to read roadmap.md: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	// Find the goal and its current section
	goalStart, goalEnd := r.findGoal(lines, goalID)
	if goalStart == -1 {
		return fmt.Errorf("goal not found: %s", goalID)
	}

	// Extract goal lines
	goalLines := lines[goalStart:goalEnd]

	// Update status in goal text
	for i := range goalLines {
		if strings.HasPrefix(goalLines[i], "**Status:**") {
			goalLines[i] = fmt.Sprintf("**Status:** %s", newStatus)
		}
		if strings.HasPrefix(goalLines[i], "**Updated:**") {
			goalLines[i] = fmt.Sprintf("**Updated:** %s", time.Now().Format("2006-01-02 15:04"))
		}
	}

	// Remove goal from current position
	newLines := append(lines[:goalStart], lines[goalEnd:]...)

	// Find target section based on status
	var targetSection string
	if newStatus == "completed" {
		targetSection = "## ✅ Completed"
	} else if newStatus == "blocked" {
		targetSection = "## 🚫 Blocked"
	} else {
		// Stay in current priority section
		return r.updateGoalInPlace(goalLines, goalStart)
	}

	insertIdx := r.findSectionIndex(newLines, targetSection)
	if insertIdx == -1 {
		return fmt.Errorf("target section not found: %s", targetSection)
	}

	// Insert at target section (right after the header)
	insertIdx++
	finalLines := append(newLines[:insertIdx], append(goalLines, newLines[insertIdx:]...)...)

	// Update timestamp
	r.updateTimestamp(finalLines)

	return os.WriteFile(r.roadmapPath, []byte(strings.Join(finalLines, "\n")), 0644)
}

// AddMilestone adds a milestone to the roadmap
func (r *Roadmap) AddMilestone(milestone Milestone) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	content, err := os.ReadFile(r.roadmapPath)
	if err != nil {
		return fmt.Errorf("failed to read roadmap.md: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	sectionIdx := r.findSectionIndex(lines, "## 🏆 Milestones")
	if sectionIdx == -1 {
		return fmt.Errorf("milestones section not found")
	}

	formattedMilestone := r.formatMilestone(milestone)
	insertIdx := sectionIdx + 2

	newLines := append(lines[:insertIdx], append([]string{formattedMilestone, ""}, lines[insertIdx:]...)...)

	r.updateTimestamp(newLines)

	return os.WriteFile(r.roadmapPath, []byte(strings.Join(newLines, "\n")), 0644)
}

// Read returns the current roadmap content
func (r *Roadmap) Read() (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	content, err := os.ReadFile(r.roadmapPath)
	if err != nil {
		return "", fmt.Errorf("failed to read roadmap.md: %w", err)
	}

	return string(content), nil
}

// GetSummary returns a brief summary of the roadmap status
func (r *Roadmap) GetSummary() (string, error) {
	content, err := r.Read()
	if err != nil {
		return "", err
	}

	lines := strings.Split(content, "\n")

	highPriority := r.countGoalsInSection(lines, "### 🔴 High Priority")
	mediumPriority := r.countGoalsInSection(lines, "### 🟡 Medium Priority")
	lowPriority := r.countGoalsInSection(lines, "### 🟢 Low Priority")
	completed := r.countGoalsInSection(lines, "## ✅ Completed")
	blocked := r.countGoalsInSection(lines, "## 🚫 Blocked")

	summary := fmt.Sprintf(`📊 Roadmap Status:
- High Priority: %d objectives
- Medium Priority: %d objectives
- Low Priority: %d objectives
- Completed: %d objectives
- Blocked: %d objectives
Total Active: %d | Total General: %d`,
		highPriority, mediumPriority, lowPriority, completed, blocked,
		highPriority+mediumPriority+lowPriority,
		highPriority+mediumPriority+lowPriority+completed+blocked)

	return summary, nil
}

// Helper methods

func (r *Roadmap) getPrioritySection(priority string) string {
	switch strings.ToLower(priority) {
	case "high", "alta", "alto":
		return "### 🔴 High Priority"
	case "medium", "média", "medio":
		return "### 🟡 Medium Priority"
	case "low", "baixa", "baixo":
		return "### 🟢 Low Priority"
	default:
		return "### 🟡 Medium Priority"
	}
}

func (r *Roadmap) findSectionIndex(lines []string, marker string) int {
	for i, line := range lines {
		if strings.TrimSpace(line) == marker {
			return i
		}
	}
	return -1
}

func (r *Roadmap) formatGoal(goal Goal) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("#### %s\n", goal.Title))
	sb.WriteString(fmt.Sprintf("*ID: `%s`* | ", goal.ID))
	sb.WriteString(fmt.Sprintf("**Status:** %s | ", goal.Status))
	sb.WriteString(fmt.Sprintf("**Created:** %s\n\n", goal.CreatedAt.Format("2006-01-02")))
	sb.WriteString(goal.Description)

	if goal.DueDate != nil {
		sb.WriteString(fmt.Sprintf("\n\n**Deadline:** %s", goal.DueDate.Format("2006-01-02")))
	}

	if len(goal.Dependencies) > 0 {
		sb.WriteString("\n\n**Dependencies:** ")
		for i, dep := range goal.Dependencies {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("`" + dep + "`")
		}
	}

	if len(goal.Tags) > 0 {
		sb.WriteString("\n\n**Tags:** ")
		for i, tag := range goal.Tags {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("`" + tag + "`")
		}
	}

	return sb.String()
}

func (r *Roadmap) formatMilestone(milestone Milestone) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("### %s\n\n", milestone.Title))
	sb.WriteString(milestone.Description)

	if milestone.TargetDate != nil {
		sb.WriteString(fmt.Sprintf("\n\n**Target Date:** %s", milestone.TargetDate.Format("2006-01-02")))
	}

	if milestone.CompletedAt != nil {
		sb.WriteString(fmt.Sprintf(" | **Completed on:** %s ✅", milestone.CompletedAt.Format("2006-01-02")))
	}

	if len(milestone.Goals) > 0 {
		sb.WriteString("\n\n**Related Objectives:** ")
		for i, goalID := range milestone.Goals {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("`" + goalID + "`")
		}
	}

	return sb.String()
}

func (r *Roadmap) findGoal(lines []string, goalID string) (start, end int) {
	idMarker := fmt.Sprintf("*ID: `%s`*", goalID)

	for i, line := range lines {
		if strings.Contains(line, idMarker) {
			start = i - 1 // Include the title line

			// Find the end of this goal (next #### or section marker)
			for j := i + 1; j < len(lines); j++ {
				if strings.HasPrefix(lines[j], "####") ||
					strings.HasPrefix(lines[j], "###") ||
					strings.HasPrefix(lines[j], "##") {
					return start, j
				}
			}
			return start, len(lines)
		}
	}

	return -1, -1
}

func (r *Roadmap) updateGoalInPlace(goalLines []string, position int) error {
	// This is a simplified update - just rewrite the file
	content, _ := os.ReadFile(r.roadmapPath)
	lines := strings.Split(string(content), "\n")

	r.updateTimestamp(lines)
	return os.WriteFile(r.roadmapPath, []byte(strings.Join(lines, "\n")), 0644)
}

func (r *Roadmap) updateTimestamp(lines []string) {
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.HasPrefix(lines[i], "*Last update:") {
			lines[i] = "*Last update: " + time.Now().Format("2006-01-02 15:04:05") + "*"
			break
		}
	}
}

func (r *Roadmap) countGoalsInSection(lines []string, section string) int {
	count := 0
	inSection := false

	for _, line := range lines {
		if strings.TrimSpace(line) == section {
			inSection = true
			continue
		}

		if inSection {
			if strings.HasPrefix(line, "##") || strings.HasPrefix(line, "---") {
				break
			}
			if strings.HasPrefix(line, "####") {
				count++
			}
		}
	}

	return count
}
