package trpcbridge

import (
	"testing"

	"ClosedWheeler/pkg/llm"
	"ClosedWheeler/pkg/tools"

	"trpc.group/trpc-go/trpc-agent-go/model"
	trpctool "trpc.group/trpc-go/trpc-agent-go/tool"
)

func TestTrpcMessagesToLLM_AllRoles(t *testing.T) {
	msgs := []model.Message{
		{Role: model.RoleSystem, Content: "you are a helper"},
		{Role: model.RoleUser, Content: "hello"},
		{Role: model.RoleAssistant, Content: "hi there", ReasoningContent: "thinking..."},
		{
			Role:   model.RoleTool,
			ToolID: "call_123",
			Content: "tool output",
		},
		{
			Role: model.RoleAssistant,
			ToolCalls: []model.ToolCall{
				{
					ID:   "tc_1",
					Type: "function",
					Function: model.FunctionDefinitionParam{
						Name:      "read_file",
						Arguments: []byte(`{"path":"main.go"}`),
					},
				},
			},
		},
	}

	out := trpcMessagesToLLM(msgs)

	if len(out) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(out))
	}

	// System
	if out[0].Role != "system" || out[0].Content != "you are a helper" {
		t.Errorf("system message mismatch: %+v", out[0])
	}

	// User
	if out[1].Role != "user" || out[1].Content != "hello" {
		t.Errorf("user message mismatch: %+v", out[1])
	}

	// Assistant with thinking
	if out[2].Role != "assistant" || out[2].Content != "hi there" || out[2].Thinking != "thinking..." {
		t.Errorf("assistant message mismatch: %+v", out[2])
	}

	// Tool result
	if out[3].Role != "tool" || out[3].ToolCallID != "call_123" || out[3].Content != "tool output" {
		t.Errorf("tool message mismatch: %+v", out[3])
	}

	// Assistant with tool calls
	if len(out[4].ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(out[4].ToolCalls))
	}
	tc := out[4].ToolCalls[0]
	if tc.ID != "tc_1" || tc.Function.Name != "read_file" || tc.Function.Arguments != `{"path":"main.go"}` {
		t.Errorf("tool call mismatch: %+v", tc)
	}
}

func TestTrpcMessagesToLLM_Empty(t *testing.T) {
	out := trpcMessagesToLLM(nil)
	if len(out) != 0 {
		t.Errorf("expected empty, got %d", len(out))
	}
}

func TestLLMResponseToTrpc(t *testing.T) {
	resp := &llm.ChatResponse{
		ID:      "resp-1",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "gpt-4",
		Choices: []llm.Choice{
			{
				Index: 0,
				Message: llm.Message{
					Role:    "assistant",
					Content: "Hello world",
				},
				FinishReason: "stop",
			},
		},
		Usage: llm.Usage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
	}

	trpcResp := llmResponseToTrpc(resp)

	if trpcResp.ID != "resp-1" {
		t.Errorf("ID mismatch: %s", trpcResp.ID)
	}
	if trpcResp.Model != "gpt-4" {
		t.Errorf("Model mismatch: %s", trpcResp.Model)
	}
	if len(trpcResp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(trpcResp.Choices))
	}
	if trpcResp.Choices[0].Message.Content != "Hello world" {
		t.Errorf("content mismatch: %s", trpcResp.Choices[0].Message.Content)
	}
	if trpcResp.Choices[0].FinishReason == nil || *trpcResp.Choices[0].FinishReason != "stop" {
		t.Errorf("finish_reason mismatch")
	}
	if trpcResp.Usage == nil || trpcResp.Usage.TotalTokens != 150 {
		t.Errorf("usage mismatch: %+v", trpcResp.Usage)
	}
}

func TestLLMResponseToTrpc_Nil(t *testing.T) {
	trpcResp := llmResponseToTrpc(nil)
	if !trpcResp.Done {
		t.Error("nil response should produce Done=true")
	}
}

func TestLLMResponseToTrpc_ToolCalls(t *testing.T) {
	resp := &llm.ChatResponse{
		Choices: []llm.Choice{
			{
				Message: llm.Message{
					Role: "assistant",
					ToolCalls: []llm.ToolCall{
						{
							ID:   "tc_42",
							Type: "function",
							Function: llm.FunctionCall{
								Name:      "shell_exec",
								Arguments: `{"command":"ls"}`,
							},
						},
					},
				},
				FinishReason: "tool_calls",
			},
		},
	}

	trpcResp := llmResponseToTrpc(resp)

	if len(trpcResp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(trpcResp.Choices))
	}
	tcs := trpcResp.Choices[0].Message.ToolCalls
	if len(tcs) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(tcs))
	}
	if tcs[0].ID != "tc_42" || tcs[0].Function.Name != "shell_exec" {
		t.Errorf("tool call mismatch: %+v", tcs[0])
	}
	if string(tcs[0].Function.Arguments) != `{"command":"ls"}` {
		t.Errorf("arguments mismatch: %s", string(tcs[0].Function.Arguments))
	}
}

func TestTrpcToolsToLLMDefs(t *testing.T) {
	schema := &trpctool.Schema{
		Type: "object",
		Properties: map[string]*trpctool.Schema{
			"path": {Type: "string", Description: "file path"},
		},
		Required: []string{"path"},
	}

	adapter := &mockTool{
		decl: &trpctool.Declaration{
			Name:        "read_file",
			Description: "Read a file",
			InputSchema: schema,
		},
	}

	trpcTools := map[string]trpctool.Tool{
		"read_file": adapter,
	}

	defs := trpcToolsToLLMDefs(trpcTools)

	if len(defs) != 1 {
		t.Fatalf("expected 1 def, got %d", len(defs))
	}
	if defs[0].Function.Name != "read_file" {
		t.Errorf("name mismatch: %s", defs[0].Function.Name)
	}
	if defs[0].Type != "function" {
		t.Errorf("type mismatch: %s", defs[0].Type)
	}
}

func TestTrpcToolsToLLMDefs_Empty(t *testing.T) {
	defs := trpcToolsToLLMDefs(nil)
	if defs != nil {
		t.Errorf("expected nil, got %v", defs)
	}
}

func TestSchemaToMap(t *testing.T) {
	schema := &trpctool.Schema{
		Type: "object",
		Properties: map[string]*trpctool.Schema{
			"name": {Type: "string", Description: "a name"},
			"age":  {Type: "integer", Default: 25},
		},
		Required: []string{"name"},
	}

	m := schemaToMap(schema)

	if m["type"] != "object" {
		t.Errorf("type mismatch: %v", m["type"])
	}
	req, ok := m["required"].([]string)
	if !ok || len(req) != 1 || req[0] != "name" {
		t.Errorf("required mismatch: %v", m["required"])
	}
	props, ok := m["properties"].(map[string]any)
	if !ok || len(props) != 2 {
		t.Errorf("properties mismatch: %v", m["properties"])
	}
}

func TestSchemaToMap_Nil(t *testing.T) {
	m := schemaToMap(nil)
	if m != nil {
		t.Errorf("expected nil, got %v", m)
	}
}

func TestModelAdapterInfo(t *testing.T) {
	adapter := NewModelAdapter(nil, "test-model", nil)
	info := adapter.Info()
	if info.Name != "test-model" {
		t.Errorf("expected test-model, got %s", info.Name)
	}
}

// mockTool implements trpctool.Tool for testing.
type mockTool struct {
	decl *trpctool.Declaration
}

func (m *mockTool) Declaration() *trpctool.Declaration {
	return m.decl
}

// Verify we can create a ToolAdapter from a real tools.Tool without panic.
func TestToolAdapterDeclaration(t *testing.T) {
	tool := &tools.Tool{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"input": {
					Type:        "string",
					Description: "some input",
				},
			},
			Required: []string{"input"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			return tools.ToolResult{Success: true}, nil
		},
	}

	registry := tools.NewRegistry()
	if err := registry.Register(tool); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	executor := tools.NewExecutor(registry)

	adapter := NewToolAdapter(tool, executor)
	decl := adapter.Declaration()

	if decl.Name != "test_tool" {
		t.Errorf("name mismatch: %s", decl.Name)
	}
	if decl.Description != "A test tool" {
		t.Errorf("description mismatch: %s", decl.Description)
	}
	if decl.InputSchema == nil {
		t.Fatal("InputSchema is nil")
	}
	if decl.InputSchema.Type != "object" {
		t.Errorf("schema type mismatch: %s", decl.InputSchema.Type)
	}
	if len(decl.InputSchema.Properties) != 1 {
		t.Errorf("expected 1 property, got %d", len(decl.InputSchema.Properties))
	}
	if len(decl.InputSchema.Required) != 1 || decl.InputSchema.Required[0] != "input" {
		t.Errorf("required mismatch: %v", decl.InputSchema.Required)
	}
}
