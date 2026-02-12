package skills

import (
	"ClosedWheeler/pkg/security"
	"ClosedWheeler/pkg/tools"
	"ClosedWheeler/pkg/tools/builtin"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SkillMetadata represents the metadata for a skill
type SkillMetadata struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Script      string            `json:"script"` // Filename of the script in the skill folder
	Parameters  *tools.JSONSchema `json:"parameters"`
}

// Manager handles loading and auditing of external skills
type Manager struct {
	projectRoot string
	skillsDir   string
	auditor     *security.Auditor
	registry    *tools.Registry
}

// NewManager creates a new skill manager
func NewManager(projectRoot string, auditor *security.Auditor, registry *tools.Registry) *Manager {
	skillsDir := filepath.Join(projectRoot, ".agi", "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		log.Printf("[WARN] Failed to create skills directory: %v", err)
	}

	return &Manager{
		projectRoot: projectRoot,
		skillsDir:   skillsDir,
		auditor:     auditor,
		registry:    registry,
	}
}

// LoadSkills scans the skills directory and registers safe skills
func (m *Manager) LoadSkills() error {
	entries, err := os.ReadDir(m.skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read skills directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if err := m.loadSkill(entry.Name()); err != nil {
				// Log error but continue with other skills
				log.Printf("[WARN] Failed to load skill %s: %v", entry.Name(), err)
			}
		}
	}

	return nil
}

func (m *Manager) loadSkill(skillFolderName string) error {
	folderPath := filepath.Join(m.skillsDir, skillFolderName)

	// 1. Read metadata
	metaPath := filepath.Join(folderPath, "skill.json")
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return fmt.Errorf("failed to read skill.json: %w", err)
	}

	var meta SkillMetadata
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return fmt.Errorf("failed to parse skill.json: %w", err)
	}

	// 2. Read and audit script
	scriptPath := filepath.Join(folderPath, meta.Script)
	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read script file %s: %w", meta.Script, err)
	}

	if err := m.auditor.AuditScript(string(scriptContent)); err != nil {
		return fmt.Errorf("security audit failed for skill %s: %w", meta.Name, err)
	}

	// 3. Register as tool
	tool := &tools.Tool{
		Name:        meta.Name,
		Description: meta.Description,
		Parameters:  meta.Parameters,
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			// Convert args to space separated string or handle as JSON
			// For simplicity, we'll try to execute the script with the args

			// We can use the existing ExecCommandTool logic but pointed to this script
			argStrings := []string{}
			for k, v := range args {
				argStrings = append(argStrings, fmt.Sprintf("--%s=%v", k, v))
			}

			// Build absolute path to script
			absScriptPath, _ := filepath.Abs(scriptPath)

			// We use a wrapper to execute the script safely
			cmdTool := builtin.ExecCommandTool(m.projectRoot, 30*time.Second, m.auditor)
			return cmdTool.Handler(map[string]any{
				"command": absScriptPath,
				"args":    strings.Join(argStrings, " "),
			})
		},
	}

	return m.registry.Register(tool)
}
