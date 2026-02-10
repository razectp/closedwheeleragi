// Package permissions provides authorization and access control.
package permissions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"ClosedWheeler/pkg/config"
)

// Manager handles permission checks and audit logging
type Manager struct {
	config    *config.PermissionsConfig
	auditFile *os.File
	mu        sync.Mutex
}

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"` // "command", "tool", "approval"
	Name      string `json:"name"`
	Allowed   bool   `json:"allowed"`
	Reason    string `json:"reason,omitempty"`
	UserID    int64  `json:"user_id,omitempty"`
}

// NewManager creates a new permissions manager
func NewManager(cfg *config.PermissionsConfig) (*Manager, error) {
	pm := &Manager{
		config: cfg,
	}

	// Open audit log file if enabled
	if cfg.EnableAuditLog {
		if err := os.MkdirAll(filepath.Dir(cfg.AuditLogPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create audit log directory: %w", err)
		}
		f, err := os.OpenFile(cfg.AuditLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open audit log: %w", err)
		}
		pm.auditFile = f
	}

	return pm, nil
}

// Close closes the audit log file
func (pm *Manager) Close() error {
	if pm.auditFile != nil {
		return pm.auditFile.Close()
	}
	return nil
}

// IsCommandAllowed checks if a command is permitted
func (pm *Manager) IsCommandAllowed(command string) bool {
	allowed := pm.checkAllowed(pm.config.AllowedCommands, command)
	pm.logAudit("command", command, allowed, "")
	return allowed
}

// IsToolAllowed checks if a tool is permitted
func (pm *Manager) IsToolAllowed(tool string) bool {
	allowed := pm.checkAllowed(pm.config.AllowedTools, tool)
	pm.logAudit("tool", tool, allowed, "")
	return allowed
}

// IsSensitiveTool checks if a tool requires approval
func (pm *Manager) IsSensitiveTool(tool string) bool {
	return pm.contains(pm.config.SensitiveTools, tool)
}

// RequiresApproval determines if a tool requires user approval
func (pm *Manager) RequiresApproval(tool string) bool {
	// If approval required for all, always return true
	if pm.config.RequireApprovalForAll {
		return true
	}

	// Check if tool is sensitive
	if pm.IsSensitiveTool(tool) {
		return true
	}

	// If auto-approve non-sensitive is disabled, require approval
	return !pm.config.AutoApproveNonSensitive
}

// LogApprovalDecision logs an approval decision to the audit log
func (pm *Manager) LogApprovalDecision(tool string, approved bool, userID int64) {
	reason := "approved by user"
	if !approved {
		reason = "denied by user"
	}
	pm.logAudit("approval", tool, approved, reason)
}

// LogApprovalTimeout logs when an approval request times out
func (pm *Manager) LogApprovalTimeout(tool string) {
	pm.logAudit("approval", tool, false, "timeout")
}

// checkAllowed checks if an item is in the allowed list
// "*" means all items are allowed
func (pm *Manager) checkAllowed(allowedList []string, item string) bool {
	// If list contains "*", everything is allowed
	if pm.contains(allowedList, "*") {
		return true
	}

	// Otherwise check if item is explicitly in the list
	return pm.contains(allowedList, item)
}

// contains checks if a slice contains a string
func (pm *Manager) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// logAudit writes an entry to the audit log
func (pm *Manager) logAudit(action, name string, allowed bool, reason string) {
	if !pm.config.EnableAuditLog || pm.auditFile == nil {
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	entry := AuditEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Action:    action,
		Name:      name,
		Allowed:   allowed,
		Reason:    reason,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return // Silent failure for audit logging
	}

	if _, err := pm.auditFile.Write(data); err != nil {
		return // Silent failure for audit logging — do not crash on log errors
	}
	pm.auditFile.WriteString("\n") //nolint:errcheck
}

// GetApprovalTimeout returns the configured timeout duration for Telegram approvals
func (pm *Manager) GetApprovalTimeout() time.Duration {
	return time.Duration(pm.config.TelegramApprovalTimeout) * time.Second
}

// UpdateConfig updates the permissions configuration at runtime
func (pm *Manager) UpdateConfig(cfg *config.PermissionsConfig) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.config = cfg
}

// GetConfig returns a copy of the current configuration
func (pm *Manager) GetConfig() config.PermissionsConfig {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return *pm.config
}
