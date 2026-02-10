package builtin

import (
	"ClosedWheeler/pkg/git"
	"ClosedWheeler/pkg/security"
	"ClosedWheeler/pkg/tools"
)

// GitStatusTool creates a tool for checking git status
func GitStatusTool(projectRoot string) *tools.Tool {
	return &tools.Tool{
		Name:        "git_status",
		Description: "Get the current Git status of the project",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"short": {
					Type:        "boolean",
					Description: "If true, show short status",
				},
			},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			client := git.NewClient(projectRoot)
			
			if !client.IsRepo() {
				return tools.ToolResult{
					Success: true,
					Output:  "Not a git repository",
				}, nil
			}
			
			status, err := client.Status()
			if err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   err.Error(),
				}, nil
			}
			
			if status == "" {
				status = "Working tree clean"
			}
			
			branch, _ := client.Branch()
			
			return tools.ToolResult{
				Success: true,
				Output:  "Branch: " + branch + "\n\n" + status,
				Data: map[string]any{
					"branch": branch,
				},
			}, nil
		},
	}
}

// GitDiffTool creates a tool for showing git diff
func GitDiffTool(projectRoot string) *tools.Tool {
	return &tools.Tool{
		Name:        "git_diff",
		Description: "Show the diff of uncommitted changes",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"staged": {
					Type:        "boolean",
					Description: "If true, show staged changes only",
				},
			},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			client := git.NewClient(projectRoot)
			
			if !client.IsRepo() {
				return tools.ToolResult{
					Success: false,
					Error:   "Not a git repository",
				}, nil
			}
			
			staged := false
			if s, ok := args["staged"].(bool); ok {
				staged = s
			}
			
			var diff string
			var err error
			if staged {
				diff, err = client.DiffStaged()
			} else {
				diff, err = client.Diff()
			}
			
			if err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   err.Error(),
				}, nil
			}
			
			if diff == "" {
				diff = "No changes"
			}
			
			return tools.ToolResult{
				Success: true,
				Output:  diff,
			}, nil
		},
	}
}

// GitCommitTool creates a tool for making commits
func GitCommitTool(projectRoot string) *tools.Tool {
	return &tools.Tool{
		Name:        "git_commit",
		Description: "Stage all changes and create a commit",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"message": {
					Type:        "string",
					Description: "Commit message",
				},
			},
			Required: []string{"message"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			client := git.NewClient(projectRoot)
			
			if !client.IsRepo() {
				return tools.ToolResult{
					Success: false,
					Error:   "Not a git repository",
				}, nil
			}
			
			message, ok := args["message"].(string)
			if !ok || message == "" {
				return tools.ToolResult{Success: false, Error: "missing required parameter: message"}, nil
			}
			
			// Stage all changes
			if err := client.AddAll(); err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   "Failed to stage: " + err.Error(),
				}, nil
			}
			
			// Check if there's anything to commit
			if !client.HasUncommittedChanges() {
				return tools.ToolResult{
					Success: true,
					Output:  "Nothing to commit",
				}, nil
			}
			
			// Commit
			if err := client.CommitWithTimestamp(message); err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   "Failed to commit: " + err.Error(),
				}, nil
			}
			
			return tools.ToolResult{
				Success: true,
				Output:  "Committed: " + message,
			}, nil
		},
	}
}

// GitLogTool creates a tool for viewing commit history
func GitLogTool(projectRoot string) *tools.Tool {
	return &tools.Tool{
		Name:        "git_log",
		Description: "Show recent commit history",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"count": {
					Type:        "integer",
					Description: "Number of commits to show (default: 10)",
				},
			},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			client := git.NewClient(projectRoot)
			
			if !client.IsRepo() {
				return tools.ToolResult{
					Success: false,
					Error:   "Not a git repository",
				}, nil
			}
			
			count := 10
			if c, ok := args["count"].(float64); ok {
				count = int(c)
			}
			
			commits, err := client.Log(count)
			if err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   err.Error(),
				}, nil
			}
			
			var output string
			for _, c := range commits {
				output += c.Hash[:7] + " " + c.Message + " (" + c.Author + ")\n"
			}
			
			if output == "" {
				output = "No commits yet"
			}
			
			return tools.ToolResult{
				Success: true,
				Output:  output,
			}, nil
		},
	}
}

// GitCheckpointTool creates a checkpoint commit
func GitCheckpointTool(projectRoot string) *tools.Tool {
	return &tools.Tool{
		Name:        "git_checkpoint",
		Description: "Create a checkpoint commit to save current state",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"description": {
					Type:        "string",
					Description: "Brief description of the checkpoint",
				},
			},
			Required: []string{"description"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			client := git.NewClient(projectRoot)
			
			if !client.IsRepo() {
				// Initialize repo if needed
				if err := client.Init(); err != nil {
					return tools.ToolResult{
						Success: false,
						Error:   "Failed to init repo: " + err.Error(),
					}, nil
				}
			}
			
			description, ok := args["description"].(string)
			if !ok || description == "" {
				return tools.ToolResult{Success: false, Error: "missing required parameter: description"}, nil
			}
			
			hash, err := client.CreateCheckpoint(description)
			if err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   err.Error(),
				}, nil
			}
			
			if hash == "" {
				return tools.ToolResult{
					Success: true,
					Output:  "No changes to checkpoint",
				}, nil
			}
			
			return tools.ToolResult{
				Success: true,
				Output:  "Checkpoint created: " + hash[:7],
				Data: map[string]any{
					"hash": hash,
				},
			}, nil
		},
	}
}

// RegisterGitTools registers all git-related tools
func RegisterGitTools(registry *tools.Registry, projectRoot string, auditor *security.Auditor) {
	registry.Register(GitStatusTool(projectRoot))
	registry.Register(GitDiffTool(projectRoot))
	registry.Register(GitCommitTool(projectRoot))
	registry.Register(GitLogTool(projectRoot))
	registry.Register(GitCheckpointTool(projectRoot))
}
