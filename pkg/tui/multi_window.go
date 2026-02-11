// Package tui provides multi-window view for dual agent conversations
package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// MultiWindowManager manages separate terminal windows for each agent
type MultiWindowManager struct {
	windows map[string]*AgentWindow
	enabled bool
	agiDir  string // Path to .agi/ directory for log files
}

// AgentWindow represents a terminal window for a specific agent
type AgentWindow struct {
	speaker string
	logFile string
	cmd     *exec.Cmd
	enabled bool
	color   string // "Blue" or "Green"
}

// NewMultiWindowManager creates a new multi-window manager.
// appPath is the application root (where .agi/ lives).
func NewMultiWindowManager(appPath string) *MultiWindowManager {
	return &MultiWindowManager{
		windows: make(map[string]*AgentWindow),
		enabled: false,
		agiDir:  filepath.Join(appPath, ".agi"),
	}
}

// OpenWindows opens separate terminal windows for each agent
func (mwm *MultiWindowManager) OpenWindows(speakers []string) error {
	if mwm.enabled {
		return fmt.Errorf("windows already open")
	}

	// Create a window for each speaker
	for _, speaker := range speakers {
		var color string
		if speaker == "Agent A" {
			color = "Blue"
		} else {
			color = "Green"
		}

		window := &AgentWindow{
			speaker: speaker,
			logFile: filepath.Join(mwm.agiDir, strings.ToLower(strings.ReplaceAll(speaker, " ", "_"))+".txt"),
			enabled: false,
			color:   color,
		}

		// Create/clear log file
		header := generateAgentWindowHeader(speaker, color)
		if err := os.WriteFile(window.logFile, []byte(header), 0644); err != nil {
			// Clean up any previously created files
			for prevSpeaker, prevWindow := range mwm.windows {
				os.Remove(prevWindow.logFile)
				delete(mwm.windows, prevSpeaker)
			}
			return fmt.Errorf("failed to create log file for %s: %w", speaker, err)
		}

		// Open terminal based on OS
		cmd, err := openTerminalForAgent(window)
		if err != nil {
			// Clean up previously opened windows
			_ = mwm.CloseWindows()
			return fmt.Errorf("failed to open window for %s: %w", speaker, err)
		}

		window.cmd = cmd
		window.enabled = true
		mwm.windows[speaker] = window
	}

	mwm.enabled = true
	return nil
}

// openTerminalForAgent opens a terminal window for a specific agent
func openTerminalForAgent(window *AgentWindow) (*exec.Cmd, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Windows: PowerShell with color-coded title
		psCommand := fmt.Sprintf(
			"$host.ui.RawUI.WindowTitle='%s (%s)'; Get-Content '%s' -Wait -Tail 100",
			window.speaker,
			window.color,
			window.logFile,
		)
		cmd = exec.Command("cmd", "/c", "start", "powershell", "-NoExit", "-Command", psCommand)

	case "darwin":
		// macOS: Terminal.app
		script := fmt.Sprintf(`tell application "Terminal"
			do script "tail -f %s"
			set custom title of front window to "%s (%s)"
			activate
		end tell`, window.logFile, window.speaker, window.color)
		cmd = exec.Command("osascript", "-e", script)

	case "linux":
		// Linux: Try common terminals
		terminals := []string{"gnome-terminal", "xterm", "konsole", "xfce4-terminal"}
		found := false

		for _, term := range terminals {
			if _, err := exec.LookPath(term); err == nil {
				switch term {
				case "gnome-terminal":
					cmd = exec.Command(term, "--title", fmt.Sprintf("%s (%s)", window.speaker, window.color),
						"--", "tail", "-f", window.logFile)
				case "xterm":
					cmd = exec.Command(term, "-T", fmt.Sprintf("%s (%s)", window.speaker, window.color),
						"-e", "tail", "-f", window.logFile)
				case "konsole", "xfce4-terminal":
					cmd = exec.Command(term, "-e", "tail", "-f", window.logFile)
				}
				found = true
				break
			}
		}

		if !found {
			return nil, fmt.Errorf("no supported terminal emulator found")
		}

	default:
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start terminal: %w", err)
	}

	return cmd, nil
}

// WriteMessage writes a message to the appropriate agent's window
func (mwm *MultiWindowManager) WriteMessage(speaker, content string, turn int) error {
	if !mwm.enabled {
		return nil
	}

	window, ok := mwm.windows[speaker]
	if !ok {
		return fmt.Errorf("no window for speaker: %s", speaker)
	}

	if !window.enabled {
		return nil
	}

	// Format message
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", 80) + "\n")
	sb.WriteString(fmt.Sprintf("Turn %d - %s\n", turn, time.Now().Format("15:04:05")))
	sb.WriteString(strings.Repeat("─", 80) + "\n")
	sb.WriteString(content)
	sb.WriteString("\n")

	// Append to file
	f, err := os.OpenFile(window.logFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}
	defer f.Close()

	_, err = f.WriteString(sb.String())
	return err
}

// IsEnabled returns whether windows are enabled
func (mwm *MultiWindowManager) IsEnabled() bool {
	return mwm.enabled
}

// CloseWindows closes all agent windows
func (mwm *MultiWindowManager) CloseWindows() error {
	if !mwm.enabled {
		return nil
	}

	for _, window := range mwm.windows {
		if window.enabled {
			// Write closing message
			footer := "\n\n" + strings.Repeat("═", 80) + "\n"
			footer += "🏁 Debate Ended\n"
			footer += "You can close this window now.\n"
			footer += strings.Repeat("═", 80) + "\n"

			f, err := os.OpenFile(window.logFile, os.O_APPEND|os.O_WRONLY, 0644)
			if err == nil {
				f.WriteString(footer)
				f.Close()
			}

			window.enabled = false
		}
	}

	mwm.enabled = false
	mwm.windows = make(map[string]*AgentWindow)

	return nil
}

// generateAgentWindowHeader generates the header for an agent window
func generateAgentWindowHeader(speaker, _ string) string {
	var sb strings.Builder

	var emoji string
	if speaker == "Agent A" {
		emoji = "🔵"
	} else {
		emoji = "🟢"
	}

	sb.WriteString(strings.Repeat("═", 80) + "\n")
	sb.WriteString("╔══════════════════════════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║                                                                              ║\n")
	sb.WriteString(fmt.Sprintf("║                       %s  %s  WINDOW  %s                            ║\n", emoji, speaker, emoji))
	sb.WriteString("║                                                                              ║\n")
	sb.WriteString("╚══════════════════════════════════════════════════════════════════════════════╝\n")
	sb.WriteString(strings.Repeat("═", 80) + "\n")
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("📺 This window shows only %s messages\n", speaker))
	sb.WriteString("\n")
	sb.WriteString("⏳ Waiting for debate to start...\n")
	sb.WriteString(strings.Repeat("═", 80) + "\n")

	return sb.String()
}
