// Package builtin provides built-in tools for the AGI agent.
package builtin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ClosedWheeler/pkg/security"
	"ClosedWheeler/pkg/tools"
)

// ReadFileTool creates a tool for reading files
func ReadFileTool(projectRoot string, auditor *security.Auditor) *tools.Tool {
	return &tools.Tool{
		Name:        "read_file",
		Description: "Read the contents of a file",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"path": {
					Type:        "string",
					Description: "Path to the file (relative to project root)",
				},
				"start_line": {
					Type:        "integer",
					Description: "Start line (1-indexed, optional)",
				},
				"end_line": {
					Type:        "integer",
					Description: "End line (1-indexed, optional)",
				},
			},
			Required: []string{"path"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			path, ok := args["path"].(string)
			if !ok {
				return tools.ToolResult{
					Success: false,
					Error:   "invalid path parameter: must be a string",
				}, fmt.Errorf("path parameter must be a string, got %T", args["path"])
			}
			fullPath := filepath.Join(projectRoot, path)

			// Security check using auditor
			if err := auditor.AuditPath(fullPath); err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   err.Error(),
				}, nil
			}

			content, err := os.ReadFile(fullPath)
			if err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   err.Error(),
				}, nil
			}

			lines := strings.Split(string(content), "\n")

			// Handle line range
			startLine := 1
			endLine := len(lines)

			if sl, ok := args["start_line"].(float64); ok {
				startLine = int(sl)
			}
			if el, ok := args["end_line"].(float64); ok {
				endLine = int(el)
			}

			// Validate bounds
			if startLine < 1 {
				startLine = 1
			}
			if endLine > len(lines) {
				endLine = len(lines)
			}
			if startLine > endLine {
				startLine = endLine
			}

			selectedLines := lines[startLine-1 : endLine]

			return tools.ToolResult{
				Success: true,
				Output:  strings.Join(selectedLines, "\n"),
				Data: map[string]any{
					"path":        path,
					"total_lines": len(lines),
					"start_line":  startLine,
					"end_line":    endLine,
				},
			}, nil
		},
	}
}

// WriteFileTool creates a tool for writing files
func WriteFileTool(projectRoot string, auditor *security.Auditor) *tools.Tool {
	return &tools.Tool{
		Name:        "write_file",
		Description: "Write content to a file. Creates the file if it doesn't exist.",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"path": {
					Type:        "string",
					Description: "Path to the file (relative to project root)",
				},
				"content": {
					Type:        "string",
					Description: "Content to write to the file",
				},
				"append": {
					Type:        "boolean",
					Description: "If true, append to file instead of overwriting",
				},
			},
			Required: []string{"path", "content"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			path, ok := args["path"].(string)
			if !ok {
				return tools.ToolResult{
					Success: false,
					Error:   "invalid path parameter: must be a string",
				}, fmt.Errorf("path parameter must be a string, got %T", args["path"])
			}

			content, ok := args["content"].(string)
			if !ok {
				return tools.ToolResult{
					Success: false,
					Error:   "invalid content parameter: must be a string",
				}, fmt.Errorf("content parameter must be a string, got %T", args["content"])
			}

			appendMode := false
			if a, ok := args["append"].(bool); ok {
				appendMode = a
			}

			fullPath := filepath.Join(projectRoot, path)

			// Security check using auditor
			if err := auditor.AuditPath(fullPath); err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   err.Error(),
				}, nil
			}

			// Create directory if needed
			dir := filepath.Dir(fullPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   err.Error(),
				}, nil
			}

			var err error
			if appendMode {
				f, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return tools.ToolResult{
						Success: false,
						Error:   err.Error(),
					}, nil
				}
				defer f.Close()
				_, err = f.WriteString(content)
			} else {
				err = os.WriteFile(fullPath, []byte(content), 0644)
			}

			if err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   err.Error(),
				}, nil
			}

			return tools.ToolResult{
				Success: true,
				Output:  fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path),
			}, nil
		},
	}
}

// ListFilesTool creates a tool for listing files
func ListFilesTool(projectRoot string, auditor *security.Auditor) *tools.Tool {
	return &tools.Tool{
		Name:        "list_files",
		Description: "List files in a directory",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"path": {
					Type:        "string",
					Description: "Path to directory (relative to project root, default: '.')",
				},
				"recursive": {
					Type:        "boolean",
					Description: "If true, list files recursively",
				},
				"pattern": {
					Type:        "string",
					Description: "Glob pattern to filter files (e.g., '*.go')",
				},
			},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			path := "."
			if p, ok := args["path"].(string); ok && p != "" {
				path = p
			}
			recursive := false
			if r, ok := args["recursive"].(bool); ok {
				recursive = r
			}
			pattern := "*"
			if p, ok := args["pattern"].(string); ok && p != "" {
				pattern = p
			}

			fullPath := filepath.Join(projectRoot, path)
			// Security check using auditor
			if err := auditor.AuditPath(fullPath); err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   err.Error(),
				}, nil
			}

			var files []string

			if recursive {
				filepath.WalkDir(fullPath, func(p string, d os.DirEntry, err error) error {
					if err != nil || d == nil {
						return nil
					}
					if d.IsDir() {
						return nil
					}

					// Skip hidden and common ignore patterns
					base := filepath.Base(p)
					if strings.HasPrefix(base, ".") {
						return nil
					}
					relPath, _ := filepath.Rel(projectRoot, p)
					if strings.Contains(relPath, "node_modules") ||
						strings.Contains(relPath, "vendor") ||
						strings.Contains(relPath, ".git") {
						return nil
					}

					matched, _ := filepath.Match(pattern, base)
					if matched || pattern == "*" {
						files = append(files, relPath)
					}
					return nil
				})
			} else {
				entries, err := os.ReadDir(fullPath)
				if err != nil {
					return tools.ToolResult{
						Success: false,
						Error:   err.Error(),
					}, nil
				}

				for _, entry := range entries {
					name := entry.Name()
					if strings.HasPrefix(name, ".") {
						continue
					}

					matched, _ := filepath.Match(pattern, name)
					if matched || pattern == "*" {
						if entry.IsDir() {
							files = append(files, name+"/")
						} else {
							files = append(files, name)
						}
					}
				}
			}

			return tools.ToolResult{
				Success: true,
				Output:  strings.Join(files, "\n"),
				Data: map[string]any{
					"count": len(files),
					"files": files,
				},
			}, nil
		},
	}
}

// SearchCodeTool creates a tool for searching code
func SearchCodeTool(projectRoot string, auditor *security.Auditor) *tools.Tool {
	return &tools.Tool{
		Name:        "search_code",
		Description: "Search for text or patterns in code files",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"query": {
					Type:        "string",
					Description: "Text or pattern to search for",
				},
				"file_pattern": {
					Type:        "string",
					Description: "Glob pattern for files to search (e.g., '*.go')",
				},
				"case_sensitive": {
					Type:        "boolean",
					Description: "If true, search is case sensitive",
				},
			},
			Required: []string{"query"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			// Search always starts at projectRoot and is internal,
			// but we can still audit the projectRoot just in case.
			if err := auditor.AuditPath(projectRoot); err != nil {
				return tools.ToolResult{Success: false, Error: err.Error()}, nil
			}

			query, ok := args["query"].(string)
			if !ok {
				return tools.ToolResult{
					Success: false,
					Error:   "invalid query parameter: must be a string",
				}, fmt.Errorf("query parameter must be a string, got %T", args["query"])
			}
			filePattern := "*"
			if p, ok := args["file_pattern"].(string); ok && p != "" {
				filePattern = p
			}
			caseSensitive := true
			if cs, ok := args["case_sensitive"].(bool); ok {
				caseSensitive = cs
			}

			searchQuery := query
			if !caseSensitive {
				searchQuery = strings.ToLower(query)
			}

			var results []string

			filepath.WalkDir(projectRoot, func(path string, d os.DirEntry, err error) error {
				if err != nil || d == nil || d.IsDir() {
					return nil
				}

				// Skip common ignores
				relPath, _ := filepath.Rel(projectRoot, path)
				if strings.Contains(relPath, "node_modules") ||
					strings.Contains(relPath, "vendor") ||
					strings.Contains(relPath, ".git") {
					return nil
				}

				// Check file pattern
				matched, _ := filepath.Match(filePattern, filepath.Base(path))
				if !matched && filePattern != "*" {
					return nil
				}

				content, err := os.ReadFile(path)
				if err != nil {
					return nil
				}

				lines := strings.Split(string(content), "\n")
				for i, line := range lines {
					searchLine := line
					if !caseSensitive {
						searchLine = strings.ToLower(line)
					}

					if strings.Contains(searchLine, searchQuery) {
						results = append(results, fmt.Sprintf("%s:%d: %s",
							relPath, i+1, strings.TrimSpace(line)))
					}
				}

				return nil
			})

			if len(results) == 0 {
				return tools.ToolResult{
					Success: true,
					Output:  "No matches found",
				}, nil
			}

			// Limit results
			if len(results) > 50 {
				results = results[:50]
				results = append(results, "... and more (showing first 50)")
			}

			return tools.ToolResult{
				Success: true,
				Output:  strings.Join(results, "\n"),
				Data: map[string]any{
					"count": len(results),
				},
			}, nil
		},
	}
}

// RegisterBuiltinTools registers all builtin tools to a registry
func RegisterBuiltinTools(registry *tools.Registry, projectRoot string, appPath string, auditor *security.Auditor) {
	registry.Register(ReadFileTool(projectRoot, auditor))
	registry.Register(WriteFileTool(projectRoot, auditor))
	registry.Register(ListFilesTool(projectRoot, auditor))
	registry.Register(SearchCodeTool(projectRoot, auditor))

	// Register Git tools
	RegisterGitTools(registry, projectRoot, auditor)

	// Register Diagnostics tools
	RegisterDiagnosticsTools(registry)

	// Register Analysis tools
	RegisterAnalysisTools(registry, projectRoot)

	// Register Command tools
	RegisterCommandTools(registry, projectRoot, auditor)

	// Register Task Management tools
	registry.Register(TaskManagerTool(projectRoot, auditor))

	// Register Browser tools
	_ = RegisterBrowserTools(registry, appPath)
}
