// Package brain provides a transparent knowledge base for the agent to learn from past experiences.
// All knowledge is stored in Markdown format in workplace/brain.md for visibility.
package brain

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Entry represents a single knowledge entry in the brain
type Entry struct {
	Timestamp   time.Time
	Category    string // "error", "pattern", "decision", "insight"
	Title       string
	Description string
	Tags        []string
}

// Brain manages the agent's knowledge base
type Brain struct {
	projectPath string
	brainPath   string
	mu          sync.RWMutex
}

// NewBrain creates a new brain instance
func NewBrain(projectPath string) *Brain {
	return &Brain{
		projectPath: projectPath,
		brainPath:   filepath.Join(projectPath, "brain.md"),
	}
}

// Initialize creates the brain.md file if it doesn't exist
func (b *Brain) Initialize() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Ensure parent directory exists
	parentDir := filepath.Dir(b.brainPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create brain directory: %w", err)
	}

	// Check if brain.md exists
	if _, err := os.Stat(b.brainPath); os.IsNotExist(err) {
		initialContent := `# 🧠 Agent Knowledge Base

This file is the agent's "brain", where it records lessons learned, patterns discovered, and important decisions.

## 📚 Index

- [Errors and Solutions](#errors-and-solutions)
- [Code Patterns](#code-patterns)
- [Architectural Decisions](#architectural-decisions)
- [Insights](#insights)

---

## Errors and Solutions

<!-- Errors found and how they were resolved -->

## Code Patterns

<!-- Patterns and conventions discovered in the project -->

## Architectural Decisions

<!-- Important technical decisions made -->

## Insights

<!-- General observations and discoveries -->

---

*Last update: ` + time.Now().Format("2006-01-02 15:04:05") + `*
`
		return os.WriteFile(b.brainPath, []byte(initialContent), 0644)
	}

	return nil
}

// AddEntry adds a new knowledge entry to the brain
func (b *Brain) AddEntry(entry Entry) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Read current content
	content, err := os.ReadFile(b.brainPath)
	if err != nil {
		return fmt.Errorf("failed to read brain.md: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	// Find the appropriate section
	sectionMarker := b.getSectionMarker(entry.Category)
	sectionIdx := b.findSectionIndex(lines, sectionMarker)

	if sectionIdx == -1 {
		return fmt.Errorf("section not found for category: %s", entry.Category)
	}

	// Format the new entry
	formattedEntry := b.formatEntry(entry)

	// Insert after section header (skip comment line)
	insertIdx := sectionIdx + 2

	// Insert the entry
	newLines := append(lines[:insertIdx], append([]string{formattedEntry, ""}, lines[insertIdx:]...)...)

	// Update timestamp
	for i := len(newLines) - 1; i >= 0; i-- {
		if strings.HasPrefix(newLines[i], "*Last update:") {
			newLines[i] = "*Last update: " + time.Now().Format("2006-01-02 15:04:05") + "*"
			break
		}
	}

	// Write back
	return os.WriteFile(b.brainPath, []byte(strings.Join(newLines, "\n")), 0644)
}

// AddError adds an error and its solution to the knowledge base
func (b *Brain) AddError(title, description, solution string, tags []string) error {
	entry := Entry{
		Timestamp:   time.Now(),
		Category:    "error",
		Title:       title,
		Description: fmt.Sprintf("%s\n\n**Solution:** %s", description, solution),
		Tags:        tags,
	}
	return b.AddEntry(entry)
}

// AddPattern adds a discovered code pattern
func (b *Brain) AddPattern(title, description string, tags []string) error {
	entry := Entry{
		Timestamp:   time.Now(),
		Category:    "pattern",
		Title:       title,
		Description: description,
		Tags:        tags,
	}
	return b.AddEntry(entry)
}

// AddDecision adds an architectural decision
func (b *Brain) AddDecision(title, description, rationale string, tags []string) error {
	entry := Entry{
		Timestamp:   time.Now(),
		Category:    "decision",
		Title:       title,
		Description: fmt.Sprintf("%s\n\n**Rationale:** %s", description, rationale),
		Tags:        tags,
	}
	return b.AddEntry(entry)
}

// AddInsight adds a general insight or observation
func (b *Brain) AddInsight(title, description string, tags []string) error {
	entry := Entry{
		Timestamp:   time.Now(),
		Category:    "insight",
		Title:       title,
		Description: description,
		Tags:        tags,
	}
	return b.AddEntry(entry)
}

// Read returns the current brain content
func (b *Brain) Read() (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	content, err := os.ReadFile(b.brainPath)
	if err != nil {
		return "", fmt.Errorf("failed to read brain.md: %w", err)
	}

	return string(content), nil
}

// Search finds entries matching a query
func (b *Brain) Search(query string) ([]string, error) {
	content, err := b.Read()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(content, "\n")
	var matches []string
	var currentEntry strings.Builder
	inEntry := false

	queryLower := strings.ToLower(query)

	for _, line := range lines {
		if strings.HasPrefix(line, "### ") {
			// New entry starts
			if inEntry && currentEntry.Len() > 0 {
				entry := currentEntry.String()
				if strings.Contains(strings.ToLower(entry), queryLower) {
					matches = append(matches, entry)
				}
			}
			currentEntry.Reset()
			currentEntry.WriteString(line + "\n")
			inEntry = true
		} else if inEntry {
			if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "---") {
				// End of current entry
				if currentEntry.Len() > 0 {
					entry := currentEntry.String()
					if strings.Contains(strings.ToLower(entry), queryLower) {
						matches = append(matches, entry)
					}
				}
				currentEntry.Reset()
				inEntry = false
			} else {
				currentEntry.WriteString(line + "\n")
			}
		}
	}

	return matches, nil
}

// Helper methods

func (b *Brain) getSectionMarker(category string) string {
	switch category {
	case "error":
		return "## Errors and Solutions"
	case "pattern":
		return "## Code Patterns"
	case "decision":
		return "## Architectural Decisions"
	case "insight":
		return "## Insights"
	default:
		return "## Insights"
	}
}

func (b *Brain) findSectionIndex(lines []string, marker string) int {
	for i, line := range lines {
		if strings.TrimSpace(line) == marker {
			return i
		}
	}
	return -1
}

func (b *Brain) formatEntry(entry Entry) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("### %s\n", entry.Title))
	sb.WriteString(fmt.Sprintf("*%s*\n\n", entry.Timestamp.Format("2006-01-02 15:04")))
	sb.WriteString(entry.Description)

	if len(entry.Tags) > 0 {
		sb.WriteString("\n\n**Tags:** ")
		for i, tag := range entry.Tags {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("`" + tag + "`")
		}
	}

	return sb.String()
}
