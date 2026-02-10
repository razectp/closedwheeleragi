package builtin

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"ClosedWheeler/pkg/security"
	"ClosedWheeler/pkg/tools"
)

// execDescription returns a runtime-accurate description for exec_command.
func execDescription() string {
	if runtime.GOOS == "windows" {
		return "Execute a shell command via cmd.exe in the workplace directory. " +
			"IMPORTANT: This is Windows — use Windows commands only: " +
			"'dir' (not ls), 'type' (not cat), 'move' (not mv), 'copy' (not cp), 'del' (not rm), 'mkdir'. " +
			"Unix commands (ls, find, grep, head, tail, cat) are NOT available. " +
			"The $PATH is the system PATH — all installed programs (git, go, node, python, etc.) are accessible."
	}
	return "Execute a shell command via sh in the workplace directory. " +
		"The $PATH is the system PATH — all installed programs are accessible."
}

// ExecCommandTool creates a tool for executing shell commands with a security auditor
func ExecCommandTool(projectRoot string, timeout time.Duration, auditor *security.Auditor) *tools.Tool {
	return &tools.Tool{
		Name:        "exec_command",
		Description: execDescription(),
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"command": {
					Type:        "string",
					Description: "The command to execute",
				},
				"args": {
					Type:        "string",
					Description: "Command arguments (space-separated)",
				},
			},
			Required: []string{"command"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			fullCmd, ok := args["command"].(string)
			if !ok {
				return tools.ToolResult{
					Success: false,
					Error:   "invalid command parameter: must be a string",
				}, fmt.Errorf("command parameter must be a string, got %T", args["command"])
			}
			if cmdArgs, ok := args["args"].(string); ok {
				fullCmd += " " + cmdArgs
			}

			// Security: Use centralized auditor
			if err := auditor.AuditCommand(fullCmd); err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("security block: %v", err),
				}, nil
			}

			// Build command
			var cmd *exec.Cmd
			if runtime.GOOS == "windows" {
				cmd = exec.Command("cmd", "/c", fullCmd)
			} else {
				cmd = exec.Command("sh", "-c", fullCmd)
			}

			cmd.Dir = projectRoot
			cmd.Env = os.Environ() // Inherit full system PATH and environment

			// Capture output
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			// Run with timeout
			// Use buffered channel to prevent goroutine leak on timeout
			done := make(chan error, 1)
			go func() {
				done <- cmd.Run()
			}()

			select {
			case err := <-done:
				if err != nil {
					return tools.ToolResult{
						Success: false,
						Output:  stdout.String(),
						Error:   fmt.Sprintf("%v\n%s", err, stderr.String()),
					}, nil
				}
			case <-time.After(timeout):
				if cmd.Process != nil {
					cmd.Process.Kill()
				}
				partialOut := stdout.String()
				if partialOut != "" {
					partialOut = "[partial output before timeout]:\n" + partialOut + "\n"
				}
				return tools.ToolResult{
					Success: false,
					Output:  partialOut,
					Error:   "command timed out",
				}, nil
			}

			output := stdout.String()
			if stderr.Len() > 0 {
				output += "\n[stderr]:\n" + stderr.String()
			}

			return tools.ToolResult{
				Success: true,
				Output:  output,
			}, nil
		},
	}
}

// RunTestsTool creates a tool for running tests
func RunTestsTool(projectRoot string) *tools.Tool {
	return &tools.Tool{
		Name:        "run_tests",
		Description: "Run tests for the project",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"path": {
					Type:        "string",
					Description: "Path to test (default: ./...)",
				},
				"verbose": {
					Type:        "boolean",
					Description: "Run in verbose mode",
				},
			},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			testPath := "./..."
			if p, ok := args["path"].(string); ok && p != "" {
				testPath = p
			}
			verbose := false
			if v, ok := args["verbose"].(bool); ok {
				verbose = v
			}

			cmdArgs := []string{"test"}
			if verbose {
				cmdArgs = append(cmdArgs, "-v")
			}
			cmdArgs = append(cmdArgs, testPath)

			cmd := exec.Command("go", cmdArgs...)
			cmd.Dir = projectRoot

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			output := stdout.String()
			if stderr.Len() > 0 {
				output += "\n" + stderr.String()
			}

			success := err == nil

			var testErr string
			if err != nil {
				testErr = err.Error()
			}
			return tools.ToolResult{
				Success: success,
				Output:  output,
				Error:   testErr,
				Data: map[string]any{
					"passed": success,
				},
			}, nil
		},
	}
}

// GoBuildTool creates a tool for building Go projects
func GoBuildTool(projectRoot string) *tools.Tool {
	return &tools.Tool{
		Name:        "go_build",
		Description: "Build the Go project",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"output": {
					Type:        "string",
					Description: "Output binary name",
				},
			},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			cmdArgs := []string{"build"}

			if output, ok := args["output"].(string); ok && output != "" {
				cmdArgs = append(cmdArgs, "-o", output)
			}

			cmdArgs = append(cmdArgs, ".")

			cmd := exec.Command("go", cmdArgs...)
			cmd.Dir = projectRoot

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			if err != nil {
				out := stdout.String()
				if stderr.Len() > 0 {
					out += "\n[stderr]:\n" + stderr.String()
				}
				return tools.ToolResult{
					Success: false,
					Output:  out,
					Error:   err.Error(),
				}, nil
			}

			return tools.ToolResult{
				Success: true,
				Output:  "Build successful",
			}, nil
		},
	}
}

// RegisterCommandTools registers command-related tools
func RegisterCommandTools(registry *tools.Registry, projectRoot string, auditor *security.Auditor) {
	registry.Register(ExecCommandTool(projectRoot, 60*time.Second, auditor))
	registry.Register(RunTestsTool(projectRoot))
	registry.Register(GoBuildTool(projectRoot))
}
