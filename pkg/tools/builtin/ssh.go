package builtin

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"ClosedWheeler/pkg/config"
	"ClosedWheeler/pkg/tools"

	"golang.org/x/crypto/ssh"
)

// sshSessionManager holds active SSH sessions.
var sshSessionManager = &sshManager{
	sessions: make(map[string]*sshSession),
}

type sshSession struct {
	client       *ssh.Client
	host         string
	user         string
	created      time.Time
	logFile      *os.File // nil if visual mode is off
	visual       bool     // whether monitor window is open
	denyCommands []string // per-host deny patterns
}

// logCommand writes a timestamped entry to the session log file.
func (s *sshSession) logCommand(command, output string, cmdErr error) {
	if s.logFile == nil {
		return
	}
	ts := time.Now().Format("15:04:05")
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[%s] $ %s\n", ts, command))
	if output != "" {
		sb.WriteString(output)
		if !strings.HasSuffix(output, "\n") {
			sb.WriteString("\n")
		}
	}
	if cmdErr != nil {
		sb.WriteString(fmt.Sprintf("[error] %v\n", cmdErr))
	}
	sb.WriteString("---\n")
	_, _ = s.logFile.WriteString(sb.String())
}

type sshManager struct {
	mu       sync.RWMutex
	sessions map[string]*sshSession // label -> session
}

func (m *sshManager) get(label string) (*sshSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[label]
	return s, ok
}

func (m *sshManager) put(label string, s *sshSession) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[label] = s
}

func (m *sshManager) remove(label string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[label]; ok {
		if s.logFile != nil {
			_ = s.logFile.Close()
		}
		_ = s.client.Close()
		delete(m.sessions, label)
	}
}

func (m *sshManager) list() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0, len(m.sessions))
	for label, s := range m.sessions {
		mode := "hidden"
		if s.visual {
			mode = "visual"
		}
		out = append(out, fmt.Sprintf("%s (%s@%s, %s, since %s)", label, s.user, s.host, mode, s.created.Format("15:04:05")))
	}
	return out
}

func (m *sshManager) closeAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for label, s := range m.sessions {
		if s.logFile != nil {
			_ = s.logFile.Close()
		}
		_ = s.client.Close()
		delete(m.sessions, label)
	}
}

// CloseSSHSessions closes all active SSH sessions. Called during agent shutdown.
func CloseSSHSessions() {
	sshSessionManager.closeAll()
}

// RegisterSSHTools registers SSH tools to the registry.
// sshCfg provides host configs and deny command patterns.
func RegisterSSHTools(registry *tools.Registry, appPath string, sshCfg *config.SSHConfig) {
	registry.Register(sshConnectTool(appPath, sshCfg))
	registry.Register(sshExecTool(sshCfg.DenyCommands))
	registry.Register(sshDisconnectTool())
	registry.Register(sshListTool())
	registry.Register(sshUploadTool())
	registry.Register(sshDownloadTool())
}

// sshDial creates a programmatic SSH connection.
func sshDial(addr, user, password, keyFile string) (*ssh.Client, error) {
	cfg := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // Agent use case, user explicitly enables SSH
		Timeout:         15 * time.Second,
	}

	if keyFile != "" {
		key, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read key file: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("failed to parse key file: %w", err)
		}
		cfg.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	} else if password != "" {
		cfg.Auth = []ssh.AuthMethod{ssh.Password(password)}
	} else {
		return nil, fmt.Errorf("either password or key_file is required")
	}

	return ssh.Dial("tcp", addr, cfg)
}

// isDeniedCommand checks if command matches any deny pattern.
// Returns true and the matched pattern if denied.
func isDeniedCommand(command string, denyPatterns []string) (bool, string) {
	lower := strings.ToLower(command)
	for _, pattern := range denyPatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return true, pattern
		}
	}
	return false, ""
}

// sshConnectTool creates a tool for establishing an SSH connection.
// In visual mode, it connects programmatically AND opens a monitor window.
// In hidden mode, it connects programmatically only.
func sshConnectTool(appPath string, sshCfg *config.SSHConfig) *tools.Tool {
	visualMode := sshCfg.VisualMode

	desc := "Connect to a remote server via SSH. "
	if visualMode {
		desc += "Connects programmatically and opens a monitor window so the user can watch commands. " +
			"If the host is pre-configured, credentials come from config (the model never sees them). " +
			"Use ssh_exec to run commands on the session."
	} else {
		desc += "Connects programmatically with provided credentials or pre-configured host. " +
			"Use ssh_exec to run commands on the session."
	}

	// In both modes we accept host/port/label.
	// In hidden mode (and when host is not pre-configured), we also accept user/password/key_file.
	props := map[string]tools.Property{
		"host": {
			Type:        "string",
			Description: "Remote host address, IP, or label of a pre-configured host",
		},
		"port": {
			Type:        "string",
			Description: "SSH port (default: 22, ignored for pre-configured hosts with port set)",
		},
		"label": {
			Type:        "string",
			Description: "Session label for referencing this connection later (default: same as host)",
		},
	}

	if !visualMode {
		props["user"] = tools.Property{
			Type:        "string",
			Description: "SSH username (not needed if host is pre-configured)",
		}
		props["password"] = tools.Property{
			Type:        "string",
			Description: "SSH password (not needed if host is pre-configured)",
		}
		props["key_file"] = tools.Property{
			Type:        "string",
			Description: "Path to SSH private key file (not needed if host is pre-configured)",
		}
	}

	return &tools.Tool{
		Name:        "ssh_connect",
		Description: desc,
		Parameters: &tools.JSONSchema{
			Type:       "object",
			Properties: props,
			Required:   []string{"host"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			hostArg, _ := args["host"].(string)
			port := "22"
			if p, ok := args["port"].(string); ok && p != "" {
				port = p
			}
			label := hostArg
			if l, ok := args["label"].(string); ok && l != "" {
				label = l
			}

			// Check for existing session with same label
			if _, exists := sshSessionManager.get(label); exists {
				return tools.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("session %q already exists. Disconnect first with ssh_disconnect.", label),
				}, nil
			}

			// Look up host in pre-configured hosts
			var hostCfg *config.SSHHostConfig
			for i := range sshCfg.Hosts {
				if sshCfg.Hosts[i].Label == hostArg || sshCfg.Hosts[i].Host == hostArg {
					hostCfg = &sshCfg.Hosts[i]
					break
				}
			}

			var (
				host     string
				user     string
				password string
				keyFile  string
				hostDeny []string
			)

			if hostCfg != nil {
				// Use pre-configured credentials
				host = hostCfg.Host
				if hostCfg.Port != "" {
					port = hostCfg.Port
				}
				user = hostCfg.User
				password = hostCfg.Password
				keyFile = hostCfg.KeyFile
				hostDeny = hostCfg.DenyCommands
				if label == hostArg && hostCfg.Label != "" {
					label = hostCfg.Label
				}
			} else if visualMode {
				// Visual mode requires pre-configured host (credentials not from model)
				return tools.ToolResult{
					Success: false,
					Error: fmt.Sprintf(
						"host %q is not pre-configured. In visual mode, add it to ssh.hosts in config.json "+
							"with credentials so the model never handles them. "+
							"Or switch to hidden mode (ssh.visual_mode: false).", hostArg),
				}, nil
			} else {
				// Hidden mode: credentials from model args
				host = hostArg
				user, _ = args["user"].(string)
				password, _ = args["password"].(string)
				keyFile, _ = args["key_file"].(string)
			}

			if user == "" {
				return tools.ToolResult{
					Success: false,
					Error:   "SSH user is required (provide via args or pre-configure the host in ssh.hosts)",
				}, nil
			}

			addr := net.JoinHostPort(host, port)

			// Connect programmatically (both modes)
			client, err := sshDial(addr, user, password, keyFile)
			if err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("SSH connection failed: %v", err),
				}, nil
			}

			sess := &sshSession{
				client:       client,
				host:         addr,
				user:         user,
				created:      time.Now(),
				visual:       visualMode,
				denyCommands: hostDeny,
			}

			// In visual mode, create log file and open monitor window
			if visualMode {
				agiDir := filepath.Join(appPath, ".agi")
				_ = os.MkdirAll(agiDir, 0755)

				logPath := filepath.Join(agiDir, "ssh_"+label+".log")
				logFile, logErr := os.Create(logPath)
				if logErr == nil {
					sess.logFile = logFile
					// Write header
					_, _ = logFile.WriteString(fmt.Sprintf("=== SSH Monitor: %s@%s (session: %s) ===\n", user, addr, label))
					_, _ = logFile.WriteString(fmt.Sprintf("=== Connected at %s ===\n\n", time.Now().Format("2006-01-02 15:04:05")))
				}

				openMonitorWindow(appPath, label, logPath)
			}

			sshSessionManager.put(label, sess)

			output := fmt.Sprintf("Connected to %s@%s as session %q.", user, addr, label)
			if visualMode {
				output += "\nMonitor window opened. Use ssh_exec to run commands."
			} else {
				output += " Use ssh_exec to run commands."
			}

			return tools.ToolResult{
				Success: true,
				Output:  output,
			}, nil
		},
	}
}

// openMonitorWindow opens a terminal window that tails the SSH log file.
func openMonitorWindow(appPath, label, logPath string) {
	agiDir := filepath.Join(appPath, ".agi")

	switch runtime.GOOS {
	case "windows":
		// PowerShell Get-Content -Wait tails the log file
		scriptPath := filepath.Join(agiDir, "ssh_monitor_"+label+".cmd")
		script := fmt.Sprintf(
			"@echo off\r\ntitle SSH Monitor: %s\r\n"+
				"powershell -NoProfile -Command \"Get-Content -Path '%s' -Wait -Tail 50\"\r\n",
			label, logPath)
		if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
			return
		}

		wtPath, wtErr := exec.LookPath("wt.exe")
		if wtErr == nil {
			cmd := exec.Command(wtPath, "new-tab", "--title", "SSH Monitor: "+label, "--", "cmd", "/c", scriptPath)
			_ = cmd.Start()
		} else {
			cmd := exec.Command("cmd", "/c", "start", "SSH Monitor: "+label, scriptPath)
			_ = cmd.Start()
		}

	case "darwin":
		scriptPath := filepath.Join(agiDir, "ssh_monitor_"+label+".sh")
		script := fmt.Sprintf("#!/bin/bash\necho \"SSH Monitor: %s\"\ntail -f \"%s\"\n", label, logPath)
		if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
			return
		}
		cmd := exec.Command("open", "-a", "Terminal", scriptPath)
		_ = cmd.Start()

	default:
		// Linux: try common terminal emulators
		scriptPath := filepath.Join(agiDir, "ssh_monitor_"+label+".sh")
		script := fmt.Sprintf("#!/bin/bash\necho \"SSH Monitor: %s\"\ntail -f \"%s\"\n", label, logPath)
		if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
			return
		}

		terminals := []struct {
			cmd  string
			args []string
		}{
			{"gnome-terminal", []string{"--", "bash", scriptPath}},
			{"konsole", []string{"-e", "bash", scriptPath}},
			{"xfce4-terminal", []string{"-e", "bash " + scriptPath}},
			{"alacritty", []string{"-e", "bash", scriptPath}},
			{"kitty", []string{"bash", scriptPath}},
			{"wezterm", []string{"start", "bash", scriptPath}},
			{"foot", []string{"bash", scriptPath}},
			{"tilix", []string{"-e", "bash " + scriptPath}},
			{"xterm", []string{"-e", "bash", scriptPath}},
		}

		for _, t := range terminals {
			if path, err := exec.LookPath(t.cmd); err == nil {
				cmd := exec.Command(path, t.args...)
				if err := cmd.Start(); err == nil {
					break
				}
			}
		}
	}
}

// sshExecTool creates a tool for executing commands on an active SSH session.
// globalDeny is the list of globally denied command patterns.
func sshExecTool(globalDeny []string) *tools.Tool {
	return &tools.Tool{
		Name:        "ssh_exec",
		Description: "Execute a command on an active SSH session. Returns stdout and stderr.",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"label": {
					Type:        "string",
					Description: "Session label (from ssh_connect). Use ssh_list to see active sessions.",
				},
				"command": {
					Type:        "string",
					Description: "Shell command to execute on the remote server",
				},
				"timeout": {
					Type:        "string",
					Description: "Timeout in seconds (default: 30)",
				},
			},
			Required: []string{"label", "command"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			label, _ := args["label"].(string)
			command, _ := args["command"].(string)
			timeoutStr, _ := args["timeout"].(string)

			timeout := 30 * time.Second
			if timeoutStr != "" {
				if secs, err := time.ParseDuration(timeoutStr + "s"); err == nil {
					timeout = secs
				}
			}

			// Check global deny patterns
			if denied, pattern := isDeniedCommand(command, globalDeny); denied {
				return tools.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("command denied by policy: matches global pattern %q", pattern),
				}, nil
			}

			sess, ok := sshSessionManager.get(label)
			if !ok {
				return tools.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("no active session %q. Use ssh_connect first or ssh_list to see sessions.", label),
				}, nil
			}

			// Check per-host deny patterns
			if denied, pattern := isDeniedCommand(command, sess.denyCommands); denied {
				return tools.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("command denied by policy: matches host pattern %q", pattern),
				}, nil
			}

			session, err := sess.client.NewSession()
			if err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("failed to create SSH session: %v", err),
				}, nil
			}
			defer session.Close()

			var stdout, stderr bytes.Buffer
			session.Stdout = &stdout
			session.Stderr = &stderr

			// Run with timeout
			done := make(chan error, 1)
			go func() {
				done <- session.Run(command)
			}()

			select {
			case err := <-done:
				output := stdout.String()
				if stderr.Len() > 0 {
					output += "\n[stderr]:\n" + stderr.String()
				}

				// Log to monitor file
				sess.logCommand(command, output, err)

				if err != nil {
					return tools.ToolResult{
						Success: false,
						Output:  output,
						Error:   fmt.Sprintf("command failed: %v", err),
					}, nil
				}
				return tools.ToolResult{
					Success: true,
					Output:  output,
				}, nil

			case <-time.After(timeout):
				// Send signal to close session (kills command)
				_ = session.Signal(ssh.SIGTERM)
				partial := stdout.String()

				// Log timeout
				sess.logCommand(command, partial, fmt.Errorf("timed out after %v", timeout))

				if partial != "" {
					partial = "[partial output]:\n" + partial + "\n"
				}
				return tools.ToolResult{
					Success: false,
					Output:  partial,
					Error:   fmt.Sprintf("command timed out after %v", timeout),
				}, nil
			}
		},
	}
}

// sshDisconnectTool creates a tool for closing an SSH session.
func sshDisconnectTool() *tools.Tool {
	return &tools.Tool{
		Name:        "ssh_disconnect",
		Description: "Close an active SSH session by label. Use ssh_list to see active sessions.",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"label": {
					Type:        "string",
					Description: "Session label to disconnect",
				},
			},
			Required: []string{"label"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			label, _ := args["label"].(string)

			if _, ok := sshSessionManager.get(label); !ok {
				return tools.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("no active session %q", label),
				}, nil
			}

			sshSessionManager.remove(label)

			return tools.ToolResult{
				Success: true,
				Output:  fmt.Sprintf("Session %q disconnected.", label),
			}, nil
		},
	}
}

// sshListTool creates a tool for listing active SSH sessions.
func sshListTool() *tools.Tool {
	return &tools.Tool{
		Name:        "ssh_list",
		Description: "List all active SSH sessions with their labels and connection details.",
		Parameters: &tools.JSONSchema{
			Type:       "object",
			Properties: map[string]tools.Property{},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			sessions := sshSessionManager.list()
			if len(sessions) == 0 {
				return tools.ToolResult{
					Success: true,
					Output:  "No active SSH sessions.",
				}, nil
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Active SSH sessions (%d):\n", len(sessions)))
			for _, s := range sessions {
				sb.WriteString("  - " + s + "\n")
			}
			return tools.ToolResult{
				Success: true,
				Output:  sb.String(),
			}, nil
		},
	}
}

// sshUploadTool creates a tool for uploading files via SCP/SFTP.
func sshUploadTool() *tools.Tool {
	return &tools.Tool{
		Name:        "ssh_upload",
		Description: "Upload a file to a remote server over an active SSH session (SFTP).",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"label": {
					Type:        "string",
					Description: "Session label",
				},
				"local_path": {
					Type:        "string",
					Description: "Local file path to upload",
				},
				"remote_path": {
					Type:        "string",
					Description: "Destination path on the remote server",
				},
			},
			Required: []string{"label", "local_path", "remote_path"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			label, _ := args["label"].(string)
			localPath, _ := args["local_path"].(string)
			remotePath, _ := args["remote_path"].(string)

			sess, ok := sshSessionManager.get(label)
			if !ok {
				return tools.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("no active session %q", label),
				}, nil
			}

			// Read local file
			data, err := os.ReadFile(localPath)
			if err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("failed to read local file: %v", err),
				}, nil
			}

			// Use SCP via an SSH session (simple approach, no SFTP library needed)
			session, err := sess.client.NewSession()
			if err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("failed to create session: %v", err),
				}, nil
			}
			defer session.Close()

			// Write file via stdin pipe to remote cat
			stdinPipe, err := session.StdinPipe()
			if err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("failed to get stdin pipe: %v", err),
				}, nil
			}

			var stderr bytes.Buffer
			session.Stderr = &stderr

			// Use cat to write to remote file
			if err := session.Start(fmt.Sprintf("cat > %q", remotePath)); err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("failed to start remote command: %v", err),
				}, nil
			}

			if _, err := io.Copy(stdinPipe, bytes.NewReader(data)); err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("failed to write data: %v", err),
				}, nil
			}
			stdinPipe.Close()

			if err := session.Wait(); err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("remote write failed: %v %s", err, stderr.String()),
				}, nil
			}

			result := fmt.Sprintf("Uploaded %s to %s:%s (%d bytes)", localPath, sess.host, remotePath, len(data))
			sess.logCommand(fmt.Sprintf("[upload] %s -> %s", localPath, remotePath), result, nil)

			return tools.ToolResult{
				Success: true,
				Output:  result,
			}, nil
		},
	}
}

// sshDownloadTool creates a tool for downloading files from a remote server.
func sshDownloadTool() *tools.Tool {
	return &tools.Tool{
		Name:        "ssh_download",
		Description: "Download a file from a remote server over an active SSH session.",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"label": {
					Type:        "string",
					Description: "Session label",
				},
				"remote_path": {
					Type:        "string",
					Description: "File path on the remote server",
				},
				"local_path": {
					Type:        "string",
					Description: "Local destination path",
				},
			},
			Required: []string{"label", "remote_path", "local_path"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			label, _ := args["label"].(string)
			remotePath, _ := args["remote_path"].(string)
			localPath, _ := args["local_path"].(string)

			sess, ok := sshSessionManager.get(label)
			if !ok {
				return tools.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("no active session %q", label),
				}, nil
			}

			session, err := sess.client.NewSession()
			if err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("failed to create session: %v", err),
				}, nil
			}
			defer session.Close()

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			session.Stdout = &stdout
			session.Stderr = &stderr

			if err := session.Run(fmt.Sprintf("cat %q", remotePath)); err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("failed to read remote file: %v %s", err, stderr.String()),
				}, nil
			}

			// Ensure parent directory exists
			if dir := filepath.Dir(localPath); dir != "" {
				_ = os.MkdirAll(dir, 0755)
			}

			if err := os.WriteFile(localPath, stdout.Bytes(), 0644); err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("failed to write local file: %v", err),
				}, nil
			}

			result := fmt.Sprintf("Downloaded %s:%s to %s (%d bytes)", sess.host, remotePath, localPath, stdout.Len())
			sess.logCommand(fmt.Sprintf("[download] %s -> %s", remotePath, localPath), result, nil)

			return tools.ToolResult{
				Success: true,
				Output:  result,
			}, nil
		},
	}
}
