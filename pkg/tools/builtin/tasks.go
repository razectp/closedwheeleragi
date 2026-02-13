package builtin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ClosedWheeler/pkg/security"
	"ClosedWheeler/pkg/tools"
)

// TaskManagerTool provides tools to manage the project's task.md file
func TaskManagerTool(projectRoot string, auditor *security.Auditor) *tools.Tool {
	return &tools.Tool{
		Name:        "manage_tasks",
		Description: "Manages the project's task.md file. Use to add, update, or list tasks.",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"action": {
					Type:        "string",
					Enum:        []string{"add", "update", "list", "sync"},
					Description: "Action to perform",
				},
				"task": {
					Type:        "string",
					Description: "Description of the task (for add/update)",
				},
				"status": {
					Type:        "string",
					Enum:        []string{"todo", "in_progress", "done"},
					Description: "Status of the task",
				},
			},
			Required: []string{"action"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			action, ok := args["action"].(string)
			if !ok || action == "" {
				return tools.ToolResult{Success: false, Error: "missing required parameter: action"}, nil
			}
			workplacePath := projectRoot
			if filepath.Base(projectRoot) != "workplace" {
				workplacePath = filepath.Join(projectRoot, "workplace")
			}
			taskPath := filepath.Join(workplacePath, "task.md")

			// Security check using auditor
			if err := auditor.AuditPath(taskPath); err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   err.Error(),
				}, nil
			}

			switch action {
			case "list":
				content, err := os.ReadFile(taskPath)
				if err != nil {
					if os.IsNotExist(err) {
						return tools.ToolResult{Success: true, Output: "No task.md found."}, nil
					}
					return tools.ToolResult{Success: false, Error: err.Error()}, nil
				}
				return tools.ToolResult{Success: true, Output: string(content)}, nil

			case "sync":
				// This is a special internal call to ensure task.md exists
				if _, err := os.Stat(taskPath); os.IsNotExist(err) {
					initialContent := "# 📋 Project Tasks\n\n- [ ] Initial project audit\n"
					if wErr := os.WriteFile(taskPath, []byte(initialContent), 0644); wErr != nil {
						return tools.ToolResult{Success: false, Error: fmt.Sprintf("failed to create task.md: %v", wErr)}, nil
					}
					return tools.ToolResult{Success: true, Output: "Created initial task.md"}, nil
				}
				return tools.ToolResult{Success: true, Output: "task.md already exists"}, nil

			case "add":
				task, ok := args["task"].(string)
				if !ok || task == "" {
					return tools.ToolResult{Success: false, Error: "missing required parameter: task"}, nil
				}
				f, err := os.OpenFile(taskPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return tools.ToolResult{Success: false, Error: err.Error()}, nil
				}
				defer f.Close()

				if _, err := f.WriteString(fmt.Sprintf("- [ ] %s\n", task)); err != nil {
					return tools.ToolResult{Success: false, Error: err.Error()}, nil
				}
				return tools.ToolResult{Success: true, Output: "Task added successfully."}, nil

			case "update":
				task, ok := args["task"].(string)
				if !ok || task == "" {
					return tools.ToolResult{Success: false, Error: "missing required parameter: task"}, nil
				}
				status, ok := args["status"].(string)
				if !ok || status == "" {
					return tools.ToolResult{Success: false, Error: "missing required parameter: status"}, nil
				}

				content, err := os.ReadFile(taskPath)
				if err != nil {
					return tools.ToolResult{Success: false, Error: err.Error()}, nil
				}

				lines := strings.Split(string(content), "\n")
				found := false
				for i, line := range lines {
					if strings.Contains(line, task) {
						char := " "
						switch status {
						case "in_progress":
							char = "/"
						case "done":
							char = "x"
						}

						// Replace early part of line (e.g. - [ ] or - [/])
						idx := strings.Index(line, "[")
						if idx != -1 && len(line) > idx+2 {
							lines[i] = line[:idx+1] + char + line[idx+2:]
							found = true
						}
					}
				}

				if !found {
					return tools.ToolResult{Success: false, Error: "Task not found."}, nil
				}

				err = os.WriteFile(taskPath, []byte(strings.Join(lines, "\n")), 0644)
				if err != nil {
					return tools.ToolResult{Success: false, Error: err.Error()}, nil
				}
				return tools.ToolResult{Success: true, Output: "Task updated successfully."}, nil
			}

			return tools.ToolResult{Success: false, Error: "Invalid action"}, nil
		},
	}
}
