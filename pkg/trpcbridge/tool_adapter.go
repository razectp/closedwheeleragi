package trpcbridge

import (
	"context"
	"encoding/json"
	"fmt"

	"ClosedWheeler/pkg/tools"

	trpctool "trpc.group/trpc-go/trpc-agent-go/tool"
)

// readOnlyTools is the set of tool names allowed for the Researcher role.
// These must match the exact names registered in pkg/tools/builtin/.
var readOnlyTools = map[string]bool{
	"read_file":           true,
	"list_files":          true,
	"search_code":         true,
	"get_code_outline":    true,
	"get_project_metrics": true,
	"get_system_info":     true,
	"manage_tasks":        true,
	"git_diff":            true,
	"git_status":          true,
	"git_log":             true,
	"web_fetch":           true,
	"browser_screenshot":  true,
	"browser_get_page_text": true,
}

// ToolAdapter wraps a tools.Tool and its Executor as a trpc-agent-go CallableTool.
type ToolAdapter struct {
	inner    *tools.Tool
	executor *tools.Executor
}

// NewToolAdapter creates a ToolAdapter for the given tool and executor.
func NewToolAdapter(t *tools.Tool, exec *tools.Executor) *ToolAdapter {
	return &ToolAdapter{inner: t, executor: exec}
}

// Declaration converts the tool's JSONSchema to a trpc-agent-go Declaration.
func (ta *ToolAdapter) Declaration() *trpctool.Declaration {
	decl := &trpctool.Declaration{
		Name:        ta.inner.Name,
		Description: ta.inner.Description,
	}
	if ta.inner.Parameters != nil {
		decl.InputSchema = jsonSchemaToTrpc(ta.inner.Parameters)
	}
	return decl
}

// Call unmarshals the JSON arguments and delegates to the Executor.
func (ta *ToolAdapter) Call(_ context.Context, jsonArgs []byte) (any, error) {
	var args map[string]any
	if len(jsonArgs) > 0 {
		if err := json.Unmarshal(jsonArgs, &args); err != nil {
			return nil, fmt.Errorf("invalid JSON arguments for tool %s: %w", ta.inner.Name, err)
		}
	}
	if args == nil {
		args = map[string]any{}
	}

	result, err := ta.executor.Execute(tools.ToolCall{
		Name:      ta.inner.Name,
		Arguments: args,
	})
	if err != nil {
		return nil, fmt.Errorf("tool %s execution error: %w", ta.inner.Name, err)
	}
	if !result.Success {
		errMsg := result.Error
		if errMsg == "" {
			errMsg = result.Output
		}
		return nil, fmt.Errorf("tool %s failed: %s", ta.inner.Name, errMsg)
	}

	return result.Output, nil
}

// AdaptAllTools converts every tool in the registry into a trpc-agent-go Tool.
func AdaptAllTools(registry *tools.Registry, executor *tools.Executor) []trpctool.Tool {
	list := registry.List()
	out := make([]trpctool.Tool, 0, len(list))
	for _, t := range list {
		out = append(out, NewToolAdapter(t, executor))
	}
	return out
}

// AdaptToolsFiltered converts tools matching the filter predicate.
func AdaptToolsFiltered(registry *tools.Registry, executor *tools.Executor, filter func(string) bool) []trpctool.Tool {
	list := registry.List()
	out := make([]trpctool.Tool, 0)
	for _, t := range list {
		if filter(t.Name) {
			out = append(out, NewToolAdapter(t, executor))
		}
	}
	return out
}

// IsReadOnlyTool returns true if the tool name is in the read-only set.
func IsReadOnlyTool(name string) bool {
	return readOnlyTools[name]
}

// jsonSchemaToTrpc converts a tools.JSONSchema to a trpc-agent-go Schema.
func jsonSchemaToTrpc(js *tools.JSONSchema) *trpctool.Schema {
	if js == nil {
		return nil
	}

	s := &trpctool.Schema{
		Type:     js.Type,
		Required: js.Required,
	}

	if len(js.Properties) > 0 {
		s.Properties = make(map[string]*trpctool.Schema, len(js.Properties))
		for name, prop := range js.Properties {
			ps := &trpctool.Schema{
				Type:        prop.Type,
				Description: prop.Description,
			}
			if len(prop.Enum) > 0 {
				ps.Enum = make([]any, len(prop.Enum))
				for i, v := range prop.Enum {
					ps.Enum[i] = v
				}
			}
			if prop.Default != nil {
				ps.Default = prop.Default
			}
			s.Properties[name] = ps
		}
	}

	return s
}
