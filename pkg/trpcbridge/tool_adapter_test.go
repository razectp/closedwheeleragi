package trpcbridge

import (
	"context"
	"encoding/json"
	"testing"

	"ClosedWheeler/pkg/tools"
)

func TestToolAdapterCall_Success(t *testing.T) {
	tool := &tools.Tool{
		Name:        "echo",
		Description: "Echoes input",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"text": {Type: "string", Description: "text to echo"},
			},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			text, _ := args["text"].(string)
			return tools.ToolResult{
				Success: true,
				Output:  "echo: " + text,
			}, nil
		},
	}

	registry := tools.NewRegistry()
	if err := registry.Register(tool); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	executor := tools.NewExecutor(registry)
	adapter := NewToolAdapter(tool, executor)

	result, err := adapter.Call(context.Background(), []byte(`{"text":"hello"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "echo: hello" {
		t.Errorf("expected 'echo: hello', got %v", result)
	}
}

func TestToolAdapterCall_Error(t *testing.T) {
	tool := &tools.Tool{
		Name:        "failing",
		Description: "Always fails",
		Parameters: &tools.JSONSchema{
			Type:       "object",
			Properties: map[string]tools.Property{},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			return tools.ToolResult{
				Success: false,
				Error:   "something went wrong",
			}, nil
		},
	}

	registry := tools.NewRegistry()
	if err := registry.Register(tool); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	executor := tools.NewExecutor(registry)
	adapter := NewToolAdapter(tool, executor)

	_, err := adapter.Call(context.Background(), []byte(`{}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestToolAdapterCall_InvalidJSON(t *testing.T) {
	tool := &tools.Tool{
		Name:        "noop",
		Description: "No-op",
		Parameters: &tools.JSONSchema{
			Type:       "object",
			Properties: map[string]tools.Property{},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			return tools.ToolResult{Success: true, Output: "ok"}, nil
		},
	}

	registry := tools.NewRegistry()
	if err := registry.Register(tool); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	executor := tools.NewExecutor(registry)
	adapter := NewToolAdapter(tool, executor)

	_, err := adapter.Call(context.Background(), []byte(`{invalid json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestToolAdapterCall_EmptyArgs(t *testing.T) {
	var receivedArgs map[string]any
	tool := &tools.Tool{
		Name:        "noargs",
		Description: "No arguments needed",
		Parameters: &tools.JSONSchema{
			Type:       "object",
			Properties: map[string]tools.Property{},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			receivedArgs = args
			return tools.ToolResult{Success: true, Output: "done"}, nil
		},
	}

	registry := tools.NewRegistry()
	if err := registry.Register(tool); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	executor := tools.NewExecutor(registry)
	adapter := NewToolAdapter(tool, executor)

	result, err := adapter.Call(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "done" {
		t.Errorf("expected 'done', got %v", result)
	}
	if receivedArgs == nil {
		t.Error("expected non-nil args map")
	}
}

func TestAdaptAllTools(t *testing.T) {
	registry := tools.NewRegistry()
	for _, name := range []string{"tool_a", "tool_b", "tool_c"} {
		n := name
		err := registry.Register(&tools.Tool{
			Name:        n,
			Description: "desc " + n,
			Parameters:  &tools.JSONSchema{Type: "object"},
			Handler: func(args map[string]any) (tools.ToolResult, error) {
				return tools.ToolResult{Success: true}, nil
			},
		})
		if err != nil {
			t.Fatalf("register %s: %v", n, err)
		}
	}
	executor := tools.NewExecutor(registry)

	adapted := AdaptAllTools(registry, executor)
	if len(adapted) != 3 {
		t.Errorf("expected 3 tools, got %d", len(adapted))
	}

	// Verify each has a valid declaration.
	for _, a := range adapted {
		decl := a.Declaration()
		if decl == nil || decl.Name == "" {
			t.Error("adapted tool has nil/empty declaration")
		}
	}
}

func TestAdaptToolsFiltered(t *testing.T) {
	registry := tools.NewRegistry()
	for _, name := range []string{"read_file", "write_file", "list_files", "shell_exec"} {
		n := name
		err := registry.Register(&tools.Tool{
			Name:        n,
			Description: "desc " + n,
			Parameters:  &tools.JSONSchema{Type: "object"},
			Handler: func(args map[string]any) (tools.ToolResult, error) {
				return tools.ToolResult{Success: true}, nil
			},
		})
		if err != nil {
			t.Fatalf("register %s: %v", n, err)
		}
	}
	executor := tools.NewExecutor(registry)

	// Filter to read-only tools.
	adapted := AdaptToolsFiltered(registry, executor, IsReadOnlyTool)

	names := map[string]bool{}
	for _, a := range adapted {
		names[a.Declaration().Name] = true
	}

	if !names["read_file"] || !names["list_files"] {
		t.Error("expected read_file and list_files to be included")
	}
	if names["write_file"] || names["shell_exec"] {
		t.Error("write_file and shell_exec should be excluded by read-only filter")
	}
}

func TestIsReadOnlyTool(t *testing.T) {
	readOnly := []string{
		"read_file", "list_files", "search_code", "get_code_outline",
		"get_project_metrics", "get_system_info", "manage_tasks",
		"git_diff", "git_status", "git_log",
		"web_fetch", "browser_screenshot", "browser_get_page_text",
	}
	for _, name := range readOnly {
		if !IsReadOnlyTool(name) {
			t.Errorf("%s should be read-only", name)
		}
	}

	writable := []string{"write_file", "exec_command", "go_build", "run_tests"}
	for _, name := range writable {
		if IsReadOnlyTool(name) {
			t.Errorf("%s should NOT be read-only", name)
		}
	}
}

func TestJSONSchemaToTrpc(t *testing.T) {
	js := &tools.JSONSchema{
		Type: "object",
		Properties: map[string]tools.Property{
			"path": {
				Type:        "string",
				Description: "file path",
			},
			"mode": {
				Type:        "string",
				Description: "open mode",
				Enum:        []string{"read", "write"},
				Default:     "read",
			},
		},
		Required: []string{"path"},
	}

	s := jsonSchemaToTrpc(js)

	if s.Type != "object" {
		t.Errorf("type mismatch: %s", s.Type)
	}
	if len(s.Properties) != 2 {
		t.Fatalf("expected 2 properties, got %d", len(s.Properties))
	}
	if s.Properties["path"].Type != "string" {
		t.Error("path type mismatch")
	}
	if s.Properties["mode"].Default != "read" {
		t.Error("mode default mismatch")
	}
	if len(s.Properties["mode"].Enum) != 2 {
		t.Error("mode enum mismatch")
	}
	if len(s.Required) != 1 || s.Required[0] != "path" {
		t.Error("required mismatch")
	}
}

func TestJSONSchemaToTrpc_Nil(t *testing.T) {
	s := jsonSchemaToTrpc(nil)
	if s != nil {
		t.Error("expected nil for nil input")
	}
}

// TestToolAdapterDeclaration_RoundTrip verifies Declaration output can be JSON-serialized.
func TestToolAdapterDeclaration_RoundTrip(t *testing.T) {
	tool := &tools.Tool{
		Name:        "roundtrip",
		Description: "Round trip test",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"x": {Type: "number", Description: "x value"},
			},
			Required: []string{"x"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			return tools.ToolResult{Success: true}, nil
		},
	}

	registry := tools.NewRegistry()
	if err := registry.Register(tool); err != nil {
		t.Fatal(err)
	}
	executor := tools.NewExecutor(registry)
	adapter := NewToolAdapter(tool, executor)

	decl := adapter.Declaration()
	data, err := json.Marshal(decl)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("empty JSON")
	}
}
