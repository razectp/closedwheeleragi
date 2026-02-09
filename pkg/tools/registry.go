// Package tools provides a registry and executor for agent tools.
package tools

import (
	"encoding/json"
	"fmt"
	"sync"
)

// Tool represents a callable tool/function
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  *JSONSchema `json:"parameters"`
	Handler     ToolHandler `json:"-"` // Function that executes the tool
}

// JSONSchema represents a JSON Schema for tool parameters
type JSONSchema struct {
	Type       string                `json:"type"`
	Properties map[string]Property   `json:"properties,omitempty"`
	Required   []string              `json:"required,omitempty"`
}

// Property represents a schema property
type Property struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
	Default     any      `json:"default,omitempty"`
}

// ToolHandler is the function signature for tool execution
type ToolHandler func(args map[string]any) (ToolResult, error)

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// ToolCall represents a request to execute a tool
type ToolCall struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// Registry manages available tools
type Registry struct {
	tools map[string]*Tool
	mu    sync.RWMutex
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]*Tool),
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(tool *Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if tool.Name == "" {
		return fmt.Errorf("tool name is required")
	}
	
	if tool.Handler == nil {
		return fmt.Errorf("tool handler is required")
	}
	
	r.tools[tool.Name] = tool
	return nil
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (*Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	tool, exists := r.tools[name]
	return tool, exists
}

// List returns all registered tools
func (r *Registry) List() []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	tools := make([]*Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// GetOpenAIFormat returns tools in OpenAI function calling format
func (r *Registry) GetOpenAIFormat() []map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	functions := make([]map[string]any, 0, len(r.tools))
	
	for _, tool := range r.tools {
		fn := map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  tool.Parameters,
			},
		}
		functions = append(functions, fn)
	}
	
	return functions
}

// Executor executes tool calls with detailed debug logging
type Executor struct {
	registry    *Registry
	debugLogger *DebugLogger
}

// NewExecutor creates a new tool executor
func NewExecutor(registry *Registry) *Executor {
	return &Executor{
		registry:    registry,
		debugLogger: GlobalDebugLogger,
	}
}

// SetDebugLevel sets the debug level for this executor
func (e *Executor) SetDebugLevel(level DebugLevel) {
	e.debugLogger.Level = level
}

// Execute runs a tool call with comprehensive error handling and debug logging
func (e *Executor) Execute(call ToolCall) (ToolResult, error) {
	// Start execution trace
	trace := e.debugLogger.StartTrace(call.Name, call.Arguments)

	// Defer panic recovery
	defer func() {
		if r := recover(); r != nil {
			panicErr := fmt.Errorf("panic during tool execution: %v", r)
			e.debugLogger.CaptureError(trace, panicErr, "panic")

			result := ToolResult{
				Success: false,
				Error:   fmt.Sprintf("PANIC: %v", r),
			}
			e.debugLogger.EndTrace(trace, result, panicErr)
		}
	}()

	// Validate tool exists
	tool, exists := e.registry.Get(call.Name)
	if !exists {
		notFoundErr := fmt.Errorf("tool not found: %s", call.Name)
		e.debugLogger.CaptureError(trace, notFoundErr, "validation")

		result := ToolResult{
			Success: false,
			Error:   fmt.Sprintf("tool not found: %s", call.Name),
		}
		e.debugLogger.EndTrace(trace, result, notFoundErr)

		return result, notFoundErr
	}

	// Add metadata
	e.debugLogger.AddMetadata(trace, "tool_description", tool.Description)

	// Execute tool
	result, err := tool.Handler(call.Arguments)

	// Capture error details if failed
	if err != nil {
		e.debugLogger.CaptureError(trace, err, "execution")
	} else if !result.Success && result.Error != "" {
		e.debugLogger.CaptureError(trace, fmt.Errorf("%s", result.Error), "execution")
	}

	// End trace with results
	e.debugLogger.EndTrace(trace, result, err)

	return result, err
}

// GetDebugReport generates a debug report
func (e *Executor) GetDebugReport() string {
	return e.debugLogger.GenerateReport()
}

// GetRecentFailures returns recent failed tool executions
func (e *Executor) GetRecentFailures() []ExecutionTrace {
	return e.debugLogger.GetFailedTraces()
}

// ExecuteFromJSON executes a tool call from JSON
func (e *Executor) ExecuteFromJSON(jsonStr string) (ToolResult, error) {
	var call ToolCall
	if err := json.Unmarshal([]byte(jsonStr), &call); err != nil {
		return ToolResult{
			Success: false,
			Error:   fmt.Sprintf("invalid JSON: %v", err),
		}, err
	}
	
	return e.Execute(call)
}

// ParseToolCalls extracts tool calls from LLM response
func ParseToolCalls(response map[string]any) ([]ToolCall, error) {
	calls := make([]ToolCall, 0)
	
	// Check for OpenAI format
	choices, ok := response["choices"].([]any)
	if !ok || len(choices) == 0 {
		return calls, nil
	}

	// Safe type assertion for first choice
	if len(choices) == 0 {
		return calls, nil
	}
	choice, ok := choices[0].(map[string]any)
	if !ok {
		return calls, nil
	}

	message, ok := choice["message"].(map[string]any)
	if !ok {
		return calls, nil
	}

	toolCalls, ok := message["tool_calls"].([]any)
	if !ok {
		return calls, nil
	}

	for _, tc := range toolCalls {
		// Safe type assertion for tool call
		tcMap, ok := tc.(map[string]any)
		if !ok {
			continue
		}

		// Safe type assertion for function
		fn, ok := tcMap["function"].(map[string]any)
		if !ok {
			continue
		}

		// Safe type assertion for arguments string
		argsStr, ok := fn["arguments"].(string)
		if !ok {
			continue
		}

		var args map[string]any
		if err := json.Unmarshal([]byte(argsStr), &args); err != nil {
			continue
		}

		// Safe type assertion for name
		name, ok := fn["name"].(string)
		if !ok {
			continue
		}

		calls = append(calls, ToolCall{
			Name:      name,
			Arguments: args,
		})
	}
	
	return calls, nil
}
