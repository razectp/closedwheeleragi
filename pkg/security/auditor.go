package security

import (
	"fmt"
	"path/filepath"
	"strings"
)

// DangerousPatterns is a list of patterns that are considered malicious or dangerous.
// Note: This is a defense-in-depth measure; the primary sandbox is AuditPath.
var DangerousPatterns = []string{
	"rm -rf /",
	"rm -rf ~",
	"dd if=",
	"mkfs.",
	"format ",
	":(){:|:&};:", // Fork bomb
	"chmod 777 /",
	"wget ",
	"curl ",
	"> /dev/sda",
	"cmd.exe",
	// Scripting / download + exec patterns
	"invoke-expression",
	"invoke-webrequest",
	"certutil -urlcache",
	"bitsadmin /transfer",
	// PowerShell with flags (bare "powershell" blocked to prevent -command/-enc use)
	"powershell",
	"pwsh -",
	// Encoded commands
	"encodedcommand",
	"-enc ",
}

// Auditor handles security checks for commands and scripts
type Auditor struct {
	projectPath   string
	customBlocked []string
}

// NewAuditor creates a new security auditor
func NewAuditor(projectPath string) *Auditor {
	return &Auditor{
		projectPath:   projectPath,
		customBlocked: []string{},
	}
}

// AuditPath ensures a path is within the allowed project directory
func (a *Auditor) AuditPath(path string) error {
	absProject, err := filepath.Abs(a.projectPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute project path: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute target path: %w", err)
	}

	// Add separator to prevent prefix collision (e.g. /workplace vs /workplaceMalicious)
	projectWithSep := absProject + string(filepath.Separator)
	if absPath != absProject && !strings.HasPrefix(absPath, projectWithSep) {
		return fmt.Errorf("access denied: path %s is outside project directory %s", path, a.projectPath)
	}

	return nil
}

// AuditCommand checks if a command string contains any dangerous patterns
func (a *Auditor) AuditCommand(command string) error {
	normalized := strings.Join(strings.Fields(strings.ToLower(command)), " ")

	for _, pattern := range DangerousPatterns {
		if strings.Contains(normalized, strings.ToLower(pattern)) {
			return fmt.Errorf("dangerous pattern detected: %s", pattern)
		}
	}

	for _, pattern := range a.customBlocked {
		if strings.Contains(normalized, strings.ToLower(pattern)) {
			return fmt.Errorf("custom blocked pattern detected: %s", pattern)
		}
	}

	return nil
}

// AuditScript checks the content of a script file for dangerous patterns
func (a *Auditor) AuditScript(content string) error {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if err := a.AuditCommand(line); err != nil {
			return fmt.Errorf("line %d: %w", i+1, err)
		}
	}
	return nil
}

// AddBlockedPattern adds a custom pattern to the blocked list
func (a *Auditor) AddBlockedPattern(pattern string) {
	a.customBlocked = append(a.customBlocked, pattern)
}
