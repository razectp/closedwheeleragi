// Package skills handles loading and auditing of external skill scripts.
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
	"sync"
	"time"
)

// SkillMetadata represents the metadata for a skill.
type SkillMetadata struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Script      string            `json:"script"` // Filename of the script in the skill folder
	Parameters  *tools.JSONSchema `json:"parameters"`
}

// SkillInfo holds runtime information about a loaded skill.
type SkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Script      string `json:"script"`
	Folder      string `json:"folder"`
}

// Manager handles loading and auditing of external skills.
type Manager struct {
	appPath   string
	skillsDir string
	auditor   *security.Auditor
	registry  *tools.Registry
	mu        sync.RWMutex
	loaded    []SkillInfo // currently loaded skills
}

// NewManager creates a new skill manager.
// Skills are stored in appPath/.agi/skills/ (application-level, not workspace).
func NewManager(appPath string, auditor *security.Auditor, registry *tools.Registry) *Manager {
	skillsDir := filepath.Join(appPath, ".agi", "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		log.Printf("[WARN] Failed to create skills directory: %v", err)
	}

	return &Manager{
		appPath:   appPath,
		skillsDir: skillsDir,
		auditor:   auditor,
		registry:  registry,
	}
}

// LoadSkills scans the skills directory and registers safe skills.
// On reload, previously loaded skills are unregistered first to avoid duplicates.
func (m *Manager) LoadSkills() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Unregister previously loaded skills before reloading
	for _, s := range m.loaded {
		m.registry.Unregister(s.Name)
	}
	m.loaded = nil

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
				log.Printf("[WARN] Failed to load skill %s: %v", entry.Name(), err)
			}
		}
	}

	return nil
}

// ListSkills returns information about all loaded skills.
func (m *Manager) ListSkills() []SkillInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]SkillInfo, len(m.loaded))
	copy(out, m.loaded)
	return out
}

// Count returns the number of loaded skills.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.loaded)
}

// SkillsDir returns the path to the skills directory.
func (m *Manager) SkillsDir() string {
	return m.skillsDir
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

	if meta.Name == "" {
		return fmt.Errorf("skill name is required in skill.json")
	}
	if meta.Script == "" {
		return fmt.Errorf("skill script is required in skill.json")
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
	absScriptPath, _ := filepath.Abs(scriptPath)
	appRoot := m.appPath

	tool := &tools.Tool{
		Name:        meta.Name,
		Description: meta.Description,
		Parameters:  meta.Parameters,
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			argStrings := make([]string, 0, len(args))
			for k, v := range args {
				argStrings = append(argStrings, fmt.Sprintf("--%s=%v", k, v))
			}

			cmdTool := builtin.ExecCommandTool(appRoot, 30*time.Second, m.auditor)
			return cmdTool.Handler(map[string]any{
				"command": absScriptPath,
				"args":    strings.Join(argStrings, " "),
			})
		},
	}

	if err := m.registry.Register(tool); err != nil {
		return fmt.Errorf("failed to register skill tool: %w", err)
	}

	m.loaded = append(m.loaded, SkillInfo{
		Name:        meta.Name,
		Description: meta.Description,
		Script:      meta.Script,
		Folder:      skillFolderName,
	})

	return nil
}
