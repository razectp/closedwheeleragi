package builtin

import (
	"fmt"
	"path/filepath"
	"strings"

	"ClosedWheeler/pkg/context"
	"ClosedWheeler/pkg/tools"
)

// GetCodeOutlineTool creates a tool for getting a high-level outline of a code file
func GetCodeOutlineTool(projectRoot string) *tools.Tool {
	return &tools.Tool{
		Name:        "get_code_outline",
		Description: "Get a high-level outline of a code file (functions, methods, classes)",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"path": {
					Type:        "string",
					Description: "Path to the file (relative to project root)",
				},
			},
			Required: []string{"path"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			path, ok := args["path"].(string)
			if !ok || path == "" {
				return tools.ToolResult{Success: false, Error: "missing required parameter: path"}, nil
			}
			fullPath := filepath.Join(projectRoot, path)

			// Use project context for analysis if possible
			pc := context.NewProjectContext(projectRoot)
			if err := pc.Load([]string{}); err != nil {
				return tools.ToolResult{Success: false, Error: err.Error()}, nil
			}

			fi, ok := pc.GetFile(fullPath)
			if !ok {
				return tools.ToolResult{Success: false, Error: "file not found in project"}, nil
			}

			if len(fi.Functions) == 0 {
				return tools.ToolResult{
					Success: true,
					Output:  fmt.Sprintf("Outline for %s:\nNo functions or methods detected.", path),
				}, nil
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Outline for %s (Lang: %s):\n\n", path, fi.Language))
			sb.WriteString(fmt.Sprintf("%-30s | %-10s | %s\n", "Symbol", "Lines", "Signature"))
			sb.WriteString(strings.Repeat("-", 80) + "\n")

			for _, fn := range fi.Functions {
				lines := fmt.Sprintf("%d-%d", fn.StartLine, fn.EndLine)
				sb.WriteString(fmt.Sprintf("%-30s | %-10s | %s\n", fn.Name, lines, fn.Signature))
			}

			return tools.ToolResult{
				Success: true,
				Output:  sb.String(),
				Data: map[string]any{
					"functions": fi.Functions,
					"language":  fi.Language,
				},
			}, nil
		},
	}
}

// GetProjectMetricsTool creates a tool for getting project-wide metrics
func GetProjectMetricsTool(projectRoot string) *tools.Tool {
	return &tools.Tool{
		Name:        "get_project_metrics",
		Description: "Get summary metrics for the entire project",
		Parameters: &tools.JSONSchema{
			Type:       "object",
			Properties: map[string]tools.Property{},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			pc := context.NewProjectContext(projectRoot)
			// Minimal load for metrics (might need refined ignore patterns)
			if err := pc.Load([]string{".git", "node_modules", "vendor"}); err != nil {
				return tools.ToolResult{Success: false, Error: err.Error()}, nil
			}

			return tools.ToolResult{
				Success: true,
				Output:  pc.GetSummary(),
				Data:    pc.Metrics,
			}, nil
		},
	}
}

// RegisterAnalysisTools registers analysis-related tools
func RegisterAnalysisTools(registry *tools.Registry, projectRoot string) {
	registry.Register(GetCodeOutlineTool(projectRoot))
	registry.Register(GetProjectMetricsTool(projectRoot))
}
