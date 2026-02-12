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
	projectPath    string
	workplaceDir   string            // Workplace directory name (default: "workplace")
	agentName      string            // Agent display name from config
	userName       string            // User display name from config
	maxFileSize    int64             // Max rule file size in bytes
	rules          map[string]string // filename -> content
}

// NewRulesManager creates a new RulesManager instance.
// workplaceDir is the sandbox directory name (e.g. "workplace").
// maxFileSizeKB is the max size per rule file in KB (0 uses default 50KB).
func NewRulesManager(projectPath, workplaceDir string, maxFileSizeKB int) *RulesManager {
	maxSize := int64(50 * 1024)
	if maxFileSizeKB > 0 {
		maxSize = int64(maxFileSizeKB) * 1024
	}
	if workplaceDir == "" {
		workplaceDir = "workplace"
	}
	return &RulesManager{
		projectPath:  projectPath,
		workplaceDir: workplaceDir,
		maxFileSize:  maxSize,
		rules:        make(map[string]string),
	}
}

// SetIdentity sets the agent and user display names for prompt injection.
func (rm *RulesManager) SetIdentity(agentName, userName string) {
	rm.agentName = agentName
	rm.userName = userName
}

// LoadRules loads rules from workplace directory
func (rm *RulesManager) LoadRules() error {
	rm.rules = make(map[string]string)

	workplacePath := rm.projectPath
	if filepath.Base(rm.projectPath) != rm.workplaceDir {
		workplacePath = filepath.Join(rm.projectPath, rm.workplaceDir)
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

	if info.Size() > rm.maxFileSize {
		return
	}

	data, err := os.ReadFile(path)
	if err == nil {
		rm.rules[key] = string(data)
	}
}

// GetFormattedRules returns all active rules formatted for a system prompt.
func (rm *RulesManager) GetFormattedRules() string {
	if len(rm.rules) == 0 && rm.agentName == "" {
		return ""
	}

	var sb strings.Builder

	// 0. Agent Identity (from config)
	if rm.agentName != "" {
		sb.WriteString("## Agent Identity\n")
		sb.WriteString(fmt.Sprintf("Your name is **%s**. ", rm.agentName))
		if rm.userName != "" {
			sb.WriteString(fmt.Sprintf("The user's name is **%s**. ", rm.userName))
		}
		sb.WriteString("Always use your name when identifying yourself.\n\n")
	}

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

	// 5. Custom rules awareness
	sb.WriteString("## Custom Rules\n")
	sb.WriteString("The user may place additional instructions in these files inside the workplace directory:\n")
	sb.WriteString("- `personality.md` — defines your personality and communication style\n")
	sb.WriteString("- `expertise.md` — defines your technical expertise and domain knowledge\n")
	sb.WriteString("- `task.md` / `todo.md` / `tasks.md` — current project tasks and priorities\n")
	sb.WriteString("- `.agirules` — core agent rules and constraints\n")
	sb.WriteString("If any of these files exist, their contents have been loaded above.\n")
	sb.WriteString("Respect and follow all instructions from these files.\n\n")

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
