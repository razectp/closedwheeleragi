// Package prompts provides system prompt templates and rules management.
package prompts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RulesManager handles loading and managing project-specific rules and context.
type RulesManager struct {
	projectPath string
	rules       map[string]string // filename -> content
}

// NewRulesManager creates a new RulesManager instance
func NewRulesManager(projectPath string) *RulesManager {
	return &RulesManager{
		projectPath: projectPath,
		rules:       make(map[string]string),
	}
}

const maxRuleFileSize = 50 * 1024 // 50KB limit per rule file

// LoadRules loads rules from workplace directory
func (rm *RulesManager) LoadRules() error {
	rm.rules = make(map[string]string)

	workplacePath := rm.projectPath
	if filepath.Base(rm.projectPath) != "workplace" {
		workplacePath = filepath.Join(rm.projectPath, "workplace")
	}

	// Load workplace/.agirules (main configuration)
	rm.loadFile(filepath.Join(workplacePath, ".agirules"), ".agirules")

	// Load personality and expertise files
	rm.loadFile(filepath.Join(workplacePath, "personality.md"), "personality.md")
	rm.loadFile(filepath.Join(workplacePath, "expertise.md"), "expertise.md")

	// Load task lists from workplace (task.md, todo.md, tasks.md)
	taskFiles := []string{"task.md", "todo.md", "tasks.md"}
	for _, tf := range taskFiles {
		rm.loadFile(filepath.Join(workplacePath, tf), "TASK:"+tf)
	}

	return nil
}

func (rm *RulesManager) loadFile(path string, key string) {
	info, err := os.Stat(path)
	if err != nil {
		return // File doesn't exist
	}

	if info.Size() > maxRuleFileSize {
		// Log error to stderr or ignore
		return
	}

	data, err := os.ReadFile(path)
	if err == nil {
		rm.rules[key] = string(data)
	}
}

// GetFormattedRules returns all active rules formatted for a system prompt.
func (rm *RulesManager) GetFormattedRules() string {
	if len(rm.rules) == 0 {
		return ""
	}

	var sb strings.Builder

	// 1. Agent Identity (personality and expertise)
	if personality, ok := rm.rules["personality.md"]; ok {
		sb.WriteString("## 🤖 Your Personality\n")
		sb.WriteString(personality)
		sb.WriteString("\n\n")
	}

	if expertise, ok := rm.rules["expertise.md"]; ok {
		sb.WriteString("## 🎯 Your Expertise\n")
		sb.WriteString(expertise)
		sb.WriteString("\n\n")
	}

	// 2. Main Rules (.agirules)
	if content, ok := rm.rules[".agirules"]; ok {
		sb.WriteString("## 🛡️ Agent Configuration\n")
		sb.WriteString(content)
		sb.WriteString("\n\n")
	}

	// 3. Current Tasks (Highest Priority)
	hasTasks := false
	for name, content := range rm.rules {
		if strings.HasPrefix(name, "TASK:") {
			if !hasTasks {
				sb.WriteString("## 📋 Current Project Tasks\n")
				sb.WriteString("Follow the progress and instructions in these task lists:\n\n")
				hasTasks = true
			}
			displayName := strings.TrimPrefix(name, "TASK:")
			sb.WriteString(fmt.Sprintf("### %s\n", displayName))
			sb.WriteString(content)
			sb.WriteString("\n\n")
		}
	}

	// 4. Other rules
	for name, content := range rm.rules {
		if name == ".agirules" || name == "personality.md" || name == "expertise.md" || strings.HasPrefix(name, "TASK:") {
			continue
		}
		sb.WriteString(fmt.Sprintf("### %s\n", name))
		sb.WriteString(content)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// GetRulesSummary returns a brief summary of loaded rules (for TUI)
func (rm *RulesManager) GetRulesSummary() string {
	if len(rm.rules) == 0 {
		return "No specific rules loaded."
	}

	var names []string
	for name := range rm.rules {
		names = append(names, name)
	}
	return "Active files: " + strings.Join(names, ", ")
}
