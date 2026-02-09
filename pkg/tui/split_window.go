// Package tui provides split window view for dual agent conversations
package tui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// SplitWindowManager manages a separate terminal window for conversation viewing
type SplitWindowManager struct {
	enabled   bool
	cmd       *exec.Cmd
	logFile   string
	lastCheck time.Time
}

// NewSplitWindowManager creates a new split window manager
func NewSplitWindowManager() *SplitWindowManager {
	return &SplitWindowManager{
		enabled:   false,
		logFile:   ".agi/conversation_live.txt",
		lastCheck: time.Now(),
	}
}

// OpenSplitWindow opens a new terminal window showing the conversation
func (swm *SplitWindowManager) OpenSplitWindow() error {
	if swm.enabled {
		return fmt.Errorf("split window already open")
	}

	// Create/clear log file
	if err := os.WriteFile(swm.logFile, []byte(""), 0644); err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	// Write initial content
	header := generateSplitWindowHeader()
	if err := os.WriteFile(swm.logFile, []byte(header), 0644); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Open terminal based on OS
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// Windows: Use PowerShell to tail the file
		psCommand := fmt.Sprintf("$host.ui.RawUI.WindowTitle='Debate View'; Get-Content '%s' -Wait -Tail 100", swm.logFile)
		cmd = exec.Command("cmd", "/c", "start", "powershell", "-NoExit", "-Command", psCommand)

	case "darwin":
		// macOS: Use osascript to open new Terminal
		script := fmt.Sprintf(`tell application "Terminal"
			do script "tail -f %s"
			activate
		end tell`, swm.logFile)
		cmd = exec.Command("osascript", "-e", script)

	case "linux":
		// Linux: Try common terminal emulators
		terminals := []string{"gnome-terminal", "xterm", "konsole", "xfce4-terminal"}
		found := false

		for _, term := range terminals {
			if _, err := exec.LookPath(term); err == nil {
				switch term {
				case "gnome-terminal":
					cmd = exec.Command(term, "--", "tail", "-f", swm.logFile)
				case "xterm", "konsole", "xfce4-terminal":
					cmd = exec.Command(term, "-e", "tail", "-f", swm.logFile)
				}
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("no supported terminal emulator found")
		}

	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open split window: %w", err)
	}

	swm.cmd = cmd
	swm.enabled = true

	return nil
}

// CloseSplitWindow closes the split window
func (swm *SplitWindowManager) CloseSplitWindow() error {
	if !swm.enabled {
		return nil
	}

	// Write closing message
	footer := "\n\n" + strings.Repeat("═", 80) + "\n"
	footer += "🏁 Debate Ended - Window will remain open\n"
	footer += "You can close this window now.\n"
	footer += strings.Repeat("═", 80) + "\n"

	f, err := os.OpenFile(swm.logFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		f.WriteString(footer)
		f.Close()
	}

	// Don't kill the process - let user close manually
	swm.enabled = false
	swm.cmd = nil

	return nil
}

// IsEnabled returns whether split window is enabled
func (swm *SplitWindowManager) IsEnabled() bool {
	return swm.enabled
}

// WriteMessage writes a message to the split window
func (swm *SplitWindowManager) WriteMessage(speaker, content string, turn int) error {
	if !swm.enabled {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", 80) + "\n")

	// Speaker header with color indicator
	if speaker == "Agent A" {
		sb.WriteString(fmt.Sprintf("🔵 AGENT A (Turn %d) - %s\n",
			turn, time.Now().Format("15:04:05")))
	} else {
		sb.WriteString(fmt.Sprintf("🟢 AGENT B (Turn %d) - %s\n",
			turn, time.Now().Format("15:04:05")))
	}

	sb.WriteString(strings.Repeat("─", 80) + "\n")
	sb.WriteString(content)
	sb.WriteString("\n")

	// Append to file
	f, err := os.OpenFile(swm.logFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}
	defer f.Close()

	_, err = f.WriteString(sb.String())
	return err
}

// UpdateProgress updates the progress indicator
func (swm *SplitWindowManager) UpdateProgress(current, max int) error {
	if !swm.enabled {
		return nil
	}

	// Only update every 5 seconds to avoid spam
	if time.Since(swm.lastCheck) < 5*time.Second {
		return nil
	}
	swm.lastCheck = time.Now()

	percent := 0
	if max > 0 {
		percent = (current * 100) / max
	}

	progress := fmt.Sprintf("\n[Progress: %d/%d turns (%d%%)]\n", current, max, percent)

	f, err := os.OpenFile(swm.logFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(progress)
	return err
}

// generateSplitWindowHeader generates the header for the split window
func generateSplitWindowHeader() string {
	var sb strings.Builder

	// ASCII art header
	sb.WriteString(strings.Repeat("═", 80) + "\n")
	sb.WriteString("╔══════════════════════════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║                                                                              ║\n")
	sb.WriteString("║              🤖 LIVE DEBATE VIEW - Agent A vs Agent B 🤖                    ║\n")
	sb.WriteString("║                                                                              ║\n")
	sb.WriteString("╚══════════════════════════════════════════════════════════════════════════════╝\n")
	sb.WriteString(strings.Repeat("═", 80) + "\n")
	sb.WriteString("\n")
	sb.WriteString("📺 This window shows the debate in real-time\n")
	sb.WriteString("🔵 Blue messages = Agent A\n")
	sb.WriteString("🟢 Green messages = Agent B\n")
	sb.WriteString("\n")
	sb.WriteString("⏳ Waiting for debate to start...\n")
	sb.WriteString(strings.Repeat("═", 80) + "\n")

	return sb.String()
}

// FormatConversationForSplitView formats the entire conversation for split view
func FormatConversationForSplitView(messages []DualMessage) string {
	var sb strings.Builder

	sb.WriteString(generateSplitWindowHeader())
	sb.WriteString("\n🎬 DEBATE STARTED\n\n")

	for _, msg := range messages {
		sb.WriteString(strings.Repeat("─", 80) + "\n")

		if msg.Speaker == "Agent A" {
			sb.WriteString(fmt.Sprintf("🔵 AGENT A (Turn %d) - %s\n",
				msg.Turn, msg.Timestamp.Format("15:04:05")))
		} else {
			sb.WriteString(fmt.Sprintf("🟢 AGENT B (Turn %d) - %s\n",
				msg.Turn, msg.Timestamp.Format("15:04:05")))
		}

		sb.WriteString(strings.Repeat("─", 80) + "\n")
		sb.WriteString(msg.Content)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// Styles for split window (not used in terminal but kept for consistency)
var (
	splitHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#00FF00")).
				Border(lipgloss.DoubleBorder())

	splitAgentAStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#0000FF"))

	splitAgentBStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00FF00"))
)
