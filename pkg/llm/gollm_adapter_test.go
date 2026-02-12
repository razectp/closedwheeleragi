package llm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// mapProviderName
// ---------------------------------------------------------------------------

func TestMapProviderName_Explicit(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     string
	}{
		{"anthropic", "anthropic", "anthropic"},
		{"openai", "openai", "openai"},
		{"google", "google", "google"},
		{"deepseek", "deepseek", "deepseek"},
		{"groq", "groq", "groq"},
		{"mistral", "mistral", "mistral"},
		{"ollama", "ollama", "ollama"},
		{"openrouter", "openrouter", "openrouter"},
		{"azure", "azure", "azure_openai"},
		{"azure_openai", "azure_openai", "azure_openai"},
		{"lmstudio", "lmstudio", "lmstudio"},
		{"vllm", "vllm", "vllm"},
		{"lambda", "lambda", "lambda"},
		{"moonshot", "moonshot", "openai"},
		{"kimi", "kimi", "openai"},
		{"case insensitive", "ANTHROPIC", "anthropic"},
		{"whitespace", "  openai  ", "openai"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapProviderName(tt.provider, "", "", "")
			if got != tt.want {
				t.Errorf("mapProviderName(%q) = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}

func TestMapProviderName_AutoDetectModel(t *testing.T) {
	tests := []struct {
		model string
		want  string
	}{
		{"claude-opus-4-6", "anthropic"},
		{"claude-sonnet-4-5-20250929", "anthropic"},
		{"gpt-4o", "openai"},
		{"o1-preview", "openai"},
		{"o3-mini", "openai"},
		{"gemini-2.5-pro", "google"},
		{"deepseek-chat", "deepseek"},
		{"kimi-k2.5", "openai"},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := mapProviderName("", tt.model, "", "")
			if got != tt.want {
				t.Errorf("mapProviderName(model=%q) = %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}

func TestMapProviderName_AutoDetectKey(t *testing.T) {
	got := mapProviderName("", "", "sk-ant-api-key", "")
	if got != "anthropic" {
		t.Errorf("expected anthropic for sk-ant- key, got %q", got)
	}
}

func TestMapProviderName_AutoDetectBaseURL(t *testing.T) {
	got := mapProviderName("", "llama3", "", "http://localhost:11434")
	if got != "ollama" {
		t.Errorf("expected ollama for localhost:11434, got %q", got)
	}
}

func TestMapProviderName_DefaultOpenAI(t *testing.T) {
	got := mapProviderName("", "unknown-model", "sk-abc123", "https://my-api.com")
	if got != "openai" {
		t.Errorf("expected openai default, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// endpointURL
// ---------------------------------------------------------------------------

func TestEndpointURL(t *testing.T) {
	tests := []struct {
		baseURL  string
		provider string
		want     string
	}{
		{"https://api.openai.com/v1", "openai", "https://api.openai.com/v1/chat/completions"},
		{"https://api.anthropic.com/v1", "anthropic", "https://api.anthropic.com/v1/messages"},
		{"http://localhost:11434/v1", "ollama", "http://localhost:11434/v1/chat/completions"},
		{"https://api.example.com/v1/", "openai", "https://api.example.com/v1/chat/completions"},
	}
	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			got := endpointURL(tt.baseURL, tt.provider)
			if got != tt.want {
				t.Errorf("endpointURL(%q, %q) = %q, want %q", tt.baseURL, tt.provider, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// setProviderHeaders
// ---------------------------------------------------------------------------

func TestSetProviderHeaders_OpenAI(t *testing.T) {
	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", nil)
	setProviderHeaders(req, "openai", "sk-test-key")

	if got := req.Header.Get("Authorization"); got != "Bearer sk-test-key" {
		t.Errorf("expected Bearer auth, got %q", got)
	}
	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("expected application/json, got %q", got)
	}
}

func TestSetProviderHeaders_Anthropic(t *testing.T) {
	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", nil)
	setProviderHeaders(req, "anthropic", "sk-ant-test")

	if got := req.Header.Get("x-api-key"); got != "sk-ant-test" {
		t.Errorf("expected x-api-key, got %q", got)
	}
	if got := req.Header.Get("anthropic-version"); got != "2023-06-01" {
		t.Errorf("expected anthropic-version, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// supportsModelListing
// ---------------------------------------------------------------------------

func TestSupportsModelListing(t *testing.T) {
	if supportsModelListing("anthropic") {
		t.Error("anthropic should not support model listing")
	}
	if supportsModelListing("google") {
		t.Error("google should not support model listing")
	}
	if !supportsModelListing("openai") {
		t.Error("openai should support model listing")
	}
	if !supportsModelListing("ollama") {
		t.Error("ollama should support model listing")
	}
}

// ---------------------------------------------------------------------------
// buildRequestBody — OpenAI format
// ---------------------------------------------------------------------------

func TestBuildRequestBody_OpenAI(t *testing.T) {
	temp := 0.7
	maxTok := 1024
	messages := []Message{
		{Role: "system", Content: "You are a helper."},
		{Role: "user", Content: "Hello"},
	}
	tools := []ToolDefinition{
		{
			Type: "function",
			Function: FunctionSchema{
				Name:        "test_tool",
				Description: "A test tool",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
	}

	body, err := buildRequestBody("openai", "gpt-4o", messages, tools, &temp, nil, &maxTok, false, "")
	if err != nil {
		t.Fatalf("buildRequestBody failed: %v", err)
	}

	var req ChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if req.Model != "gpt-4o" {
		t.Errorf("model = %q, want gpt-4o", req.Model)
	}
	if len(req.Messages) != 2 {
		t.Errorf("messages = %d, want 2", len(req.Messages))
	}
	if len(req.Tools) != 1 {
		t.Errorf("tools = %d, want 1", len(req.Tools))
	}
	if req.ToolChoice != "auto" {
		t.Errorf("tool_choice = %v, want auto", req.ToolChoice)
	}
	if req.Stream {
		t.Error("stream should be false")
	}
}

func TestBuildRequestBody_OpenAI_Streaming(t *testing.T) {
	body, err := buildRequestBody("openai", "gpt-4o", []Message{{Role: "user", Content: "hi"}}, nil, nil, nil, nil, true, "")
	if err != nil {
		t.Fatalf("buildRequestBody failed: %v", err)
	}

	var req ChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !req.Stream {
		t.Error("stream should be true")
	}
	if req.StreamOptions == nil || !req.StreamOptions.IncludeUsage {
		t.Error("stream_options.include_usage should be true")
	}
}

func TestBuildRequestBody_OpenAI_ReasoningEffort(t *testing.T) {
	body, err := buildRequestBody("openai", "o1-preview", []Message{{Role: "user", Content: "hi"}}, nil, nil, nil, nil, false, "high")
	if err != nil {
		t.Fatalf("buildRequestBody failed: %v", err)
	}

	var req ChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if req.ReasoningEffort != "high" {
		t.Errorf("reasoning_effort = %q, want high", req.ReasoningEffort)
	}
}

// ---------------------------------------------------------------------------
// buildRequestBody — Anthropic format
// ---------------------------------------------------------------------------

func TestBuildRequestBody_Anthropic(t *testing.T) {
	temp := 0.7
	maxTok := 2048
	messages := []Message{
		{Role: "system", Content: "Be helpful."},
		{Role: "user", Content: "Hello"},
	}

	body, err := buildRequestBody("anthropic", "claude-opus-4-6", messages, nil, &temp, nil, &maxTok, false, "")
	if err != nil {
		t.Fatalf("buildRequestBody failed: %v", err)
	}

	var req anthropicRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if req.Model != "claude-opus-4-6" {
		t.Errorf("model = %q, want claude-opus-4-6", req.Model)
	}
	if req.MaxTokens != 2048 {
		t.Errorf("max_tokens = %d, want 2048", req.MaxTokens)
	}
	// System should be extracted as separate field.
	if req.System == nil {
		t.Error("system prompt should be set")
	}
	// Only user message should remain in messages array.
	if len(req.Messages) != 1 {
		t.Errorf("messages = %d, want 1 (system extracted)", len(req.Messages))
	}
}

func TestBuildRequestBody_Anthropic_ExtendedThinking(t *testing.T) {
	maxTok := 4096
	body, err := buildRequestBody("anthropic", "claude-opus-4-6", []Message{{Role: "user", Content: "think"}}, nil, nil, nil, &maxTok, false, "high")
	if err != nil {
		t.Fatalf("buildRequestBody failed: %v", err)
	}

	var req anthropicRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if req.Thinking == nil {
		t.Fatal("thinking should be set for effort=high")
	}
	if req.Thinking.Type != "enabled" {
		t.Errorf("thinking.type = %q, want enabled", req.Thinking.Type)
	}
	if req.Thinking.BudgetTokens != 16384 {
		t.Errorf("budget_tokens = %d, want 16384", req.Thinking.BudgetTokens)
	}
}

// ---------------------------------------------------------------------------
// convertToAnthropicMessages
// ---------------------------------------------------------------------------

func TestConvertToAnthropicMessages_Basic(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "How are you?"},
	}

	result := convertToAnthropicMessages(messages)
	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(result))
	}
	if result[0].Role != "user" {
		t.Errorf("first message role = %q, want user", result[0].Role)
	}
}

func TestConvertToAnthropicMessages_ToolCalls(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "search for X"},
		{
			Role:    "assistant",
			Content: "Let me search.",
			ToolCalls: []ToolCall{
				{
					ID:   "call_123",
					Type: "function",
					Function: FunctionCall{
						Name:      "search",
						Arguments: `{"query":"X"}`,
					},
				},
			},
		},
		{
			Role:       "tool",
			Content:    "Found result Y",
			ToolCallID: "call_123",
		},
	}

	result := convertToAnthropicMessages(messages)
	// tool message maps to user role, and should merge with nothing (assistant before it).
	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(result))
	}

	// Check assistant message has text + tool_use blocks.
	assistantContent := result[1].Content
	if len(assistantContent) != 2 {
		t.Fatalf("assistant content blocks = %d, want 2", len(assistantContent))
	}

	// Check tool result maps to user role.
	if result[2].Role != "user" {
		t.Errorf("tool result role = %q, want user", result[2].Role)
	}
}

func TestConvertToAnthropicMessages_RoleMerge(t *testing.T) {
	// Two consecutive user messages should be merged.
	messages := []Message{
		{Role: "user", Content: "Part 1"},
		{Role: "user", Content: "Part 2"},
	}

	result := convertToAnthropicMessages(messages)
	if len(result) != 1 {
		t.Fatalf("expected 1 merged message, got %d", len(result))
	}
	if len(result[0].Content) != 2 {
		t.Errorf("expected 2 content blocks in merged message, got %d", len(result[0].Content))
	}
}

// ---------------------------------------------------------------------------
// parseResponseBody — OpenAI
// ---------------------------------------------------------------------------

func TestParseResponseBody_OpenAI(t *testing.T) {
	jsonResp := `{
		"id": "chatcmpl-123",
		"object": "chat.completion",
		"model": "gpt-4o",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "Hello!"
			},
			"finish_reason": "stop"
		}],
		"usage": {
			"prompt_tokens": 10,
			"completion_tokens": 5,
			"total_tokens": 15
		}
	}`

	resp, err := parseResponseBody("openai", []byte(jsonResp))
	if err != nil {
		t.Fatalf("parseResponseBody failed: %v", err)
	}

	if resp.ID != "chatcmpl-123" {
		t.Errorf("id = %q", resp.ID)
	}
	if resp.Choices[0].Message.Content != "Hello!" {
		t.Errorf("content = %q", resp.Choices[0].Message.Content)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("total_tokens = %d", resp.Usage.TotalTokens)
	}
}

func TestParseResponseBody_OpenAI_ToolCalls(t *testing.T) {
	jsonResp := `{
		"id": "chatcmpl-456",
		"object": "chat.completion",
		"model": "gpt-4o",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "",
				"tool_calls": [{
					"id": "call_abc",
					"type": "function",
					"function": {
						"name": "search",
						"arguments": "{\"query\":\"test\"}"
					}
				}]
			},
			"finish_reason": "tool_calls"
		}]
	}`

	resp, err := parseResponseBody("openai", []byte(jsonResp))
	if err != nil {
		t.Fatalf("parseResponseBody failed: %v", err)
	}

	if len(resp.Choices[0].Message.ToolCalls) != 1 {
		t.Fatalf("tool_calls = %d, want 1", len(resp.Choices[0].Message.ToolCalls))
	}
	tc := resp.Choices[0].Message.ToolCalls[0]
	if tc.Function.Name != "search" {
		t.Errorf("tool name = %q", tc.Function.Name)
	}
}

// ---------------------------------------------------------------------------
// parseResponseBody — Anthropic
// ---------------------------------------------------------------------------

func TestParseResponseBody_Anthropic(t *testing.T) {
	jsonResp := `{
		"id": "msg_123",
		"type": "message",
		"role": "assistant",
		"content": [
			{"type": "text", "text": "Hello from Claude!"}
		],
		"model": "claude-opus-4-6",
		"stop_reason": "end_turn",
		"usage": {"input_tokens": 10, "output_tokens": 5}
	}`

	resp, err := parseResponseBody("anthropic", []byte(jsonResp))
	if err != nil {
		t.Fatalf("parseResponseBody failed: %v", err)
	}

	if resp.ID != "msg_123" {
		t.Errorf("id = %q", resp.ID)
	}
	if resp.Choices[0].Message.Content != "Hello from Claude!" {
		t.Errorf("content = %q", resp.Choices[0].Message.Content)
	}
	if resp.Choices[0].FinishReason != "stop" {
		t.Errorf("finish_reason = %q", resp.Choices[0].FinishReason)
	}
	if resp.Usage.PromptTokens != 10 {
		t.Errorf("prompt_tokens = %d", resp.Usage.PromptTokens)
	}
}

func TestParseResponseBody_Anthropic_ToolUse(t *testing.T) {
	jsonResp := `{
		"id": "msg_456",
		"type": "message",
		"role": "assistant",
		"content": [
			{"type": "text", "text": "Let me search."},
			{"type": "tool_use", "id": "toolu_123", "name": "search", "input": {"query": "test"}}
		],
		"model": "claude-opus-4-6",
		"stop_reason": "tool_use",
		"usage": {"input_tokens": 20, "output_tokens": 15}
	}`

	resp, err := parseResponseBody("anthropic", []byte(jsonResp))
	if err != nil {
		t.Fatalf("parseResponseBody failed: %v", err)
	}

	if resp.Choices[0].Message.Content != "Let me search." {
		t.Errorf("content = %q", resp.Choices[0].Message.Content)
	}
	if resp.Choices[0].FinishReason != "tool_calls" {
		t.Errorf("finish_reason = %q, want tool_calls", resp.Choices[0].FinishReason)
	}
	if len(resp.Choices[0].Message.ToolCalls) != 1 {
		t.Fatalf("tool_calls = %d, want 1", len(resp.Choices[0].Message.ToolCalls))
	}
	tc := resp.Choices[0].Message.ToolCalls[0]
	if tc.ID != "toolu_123" {
		t.Errorf("tool_call id = %q", tc.ID)
	}
	if tc.Function.Name != "search" {
		t.Errorf("tool name = %q", tc.Function.Name)
	}
}

func TestParseResponseBody_Anthropic_Error(t *testing.T) {
	jsonResp := `{
		"type": "error",
		"error": {"type": "invalid_request_error", "message": "bad request"}
	}`

	_, err := parseResponseBody("anthropic", []byte(jsonResp))
	if err == nil {
		t.Fatal("expected error for Anthropic error response")
	}
	if !strings.Contains(err.Error(), "bad request") {
		t.Errorf("error = %q, should contain 'bad request'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// mapStopReason
// ---------------------------------------------------------------------------

func TestMapStopReason(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"end_turn", "stop"},
		{"max_tokens", "length"},
		{"tool_use", "tool_calls"},
		{"stop_sequence", "stop"},
		{"", "stop"},
		{"unknown", "stop"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := mapStopReason(tt.in); got != tt.want {
				t.Errorf("mapStopReason(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Rate limit parsing
// ---------------------------------------------------------------------------

func TestParseOpenAIRateLimits(t *testing.T) {
	h := http.Header{}
	h.Set("x-ratelimit-remaining-requests", "100")
	h.Set("x-ratelimit-remaining-tokens", "50000")

	rl := parseOpenAIRateLimits(h)
	if rl.RemainingRequests != 100 {
		t.Errorf("remaining_requests = %d", rl.RemainingRequests)
	}
	if rl.RemainingTokens != 50000 {
		t.Errorf("remaining_tokens = %d", rl.RemainingTokens)
	}
}

func TestParseAnthropicRateLimits(t *testing.T) {
	h := http.Header{}
	h.Set("anthropic-ratelimit-requests-remaining", "42")
	h.Set("anthropic-ratelimit-tokens-remaining", "10000")
	h.Set("anthropic-ratelimit-requests-reset", "2025-01-01T12:00:00Z")

	rl := parseAnthropicRateLimits(h)
	if rl.RemainingRequests != 42 {
		t.Errorf("remaining_requests = %d", rl.RemainingRequests)
	}
	if rl.ResetRequests.IsZero() {
		t.Error("reset_requests should be parsed")
	}
}

// ---------------------------------------------------------------------------
// SSE stream parsing — OpenAI
// ---------------------------------------------------------------------------

func TestParseOpenAISSEStream(t *testing.T) {
	sseData := `data: {"id":"chatcmpl-1","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}

data: {"id":"chatcmpl-1","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-1","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}

data: {"id":"chatcmpl-1","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}

data: [DONE]

`
	var chunks []string
	var doneCalled bool
	callback := func(content, thinking string, done bool) {
		if done {
			doneCalled = true
			return
		}
		if content != "" {
			chunks = append(chunks, content)
		}
	}

	resp, err := parseOpenAISSEStream(strings.NewReader(sseData), callback)
	if err != nil {
		t.Fatalf("parseOpenAISSEStream failed: %v", err)
	}

	if !doneCalled {
		t.Error("callback done=true was not called")
	}

	if resp.Choices[0].Message.Content != "Hello world" {
		t.Errorf("content = %q, want 'Hello world'", resp.Choices[0].Message.Content)
	}

	if resp.Choices[0].FinishReason != "stop" {
		t.Errorf("finish_reason = %q", resp.Choices[0].FinishReason)
	}

	if resp.Usage.TotalTokens != 7 {
		t.Errorf("total_tokens = %d, want 7", resp.Usage.TotalTokens)
	}

	if len(chunks) != 2 || chunks[0] != "Hello" || chunks[1] != " world" {
		t.Errorf("chunks = %v", chunks)
	}
}

func TestParseOpenAISSEStream_ToolCalls(t *testing.T) {
	sseData := `data: {"id":"chatcmpl-2","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":"","tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"search","arguments":""}}]},"finish_reason":null}]}

data: {"id":"chatcmpl-2","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"q\":"}}]},"finish_reason":null}]}

data: {"id":"chatcmpl-2","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"test\"}"}}]},"finish_reason":null}]}

data: {"id":"chatcmpl-2","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}

data: [DONE]

`

	resp, err := parseOpenAISSEStream(strings.NewReader(sseData), nil)
	if err != nil {
		t.Fatalf("parseOpenAISSEStream failed: %v", err)
	}

	toolCalls := resp.Choices[0].Message.ToolCalls
	if len(toolCalls) != 1 {
		t.Fatalf("tool_calls = %d, want 1", len(toolCalls))
	}
	if toolCalls[0].Function.Name != "search" {
		t.Errorf("tool name = %q", toolCalls[0].Function.Name)
	}
	if toolCalls[0].Function.Arguments != `{"q":"test"}` {
		t.Errorf("tool args = %q", toolCalls[0].Function.Arguments)
	}
}

// ---------------------------------------------------------------------------
// SSE stream parsing — Anthropic
// ---------------------------------------------------------------------------

func TestParseAnthropicSSEStream(t *testing.T) {
	sseData := `event: message_start
data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","content":[],"model":"claude-opus-4-6","usage":{"input_tokens":10,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":5}}

event: message_stop
data: {"type":"message_stop"}

`
	var chunks []string
	var doneCalled bool
	callback := func(content, thinking string, done bool) {
		if done {
			doneCalled = true
			return
		}
		if content != "" {
			chunks = append(chunks, content)
		}
	}

	resp, err := parseAnthropicSSEStream(strings.NewReader(sseData), callback)
	if err != nil {
		t.Fatalf("parseAnthropicSSEStream failed: %v", err)
	}

	if !doneCalled {
		t.Error("callback done=true was not called")
	}

	if resp.Choices[0].Message.Content != "Hello world" {
		t.Errorf("content = %q, want 'Hello world'", resp.Choices[0].Message.Content)
	}

	if resp.Choices[0].FinishReason != "stop" {
		t.Errorf("finish_reason = %q", resp.Choices[0].FinishReason)
	}

	if resp.Usage.PromptTokens != 10 {
		t.Errorf("prompt_tokens = %d", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 5 {
		t.Errorf("completion_tokens = %d", resp.Usage.CompletionTokens)
	}
}

func TestParseAnthropicSSEStream_ToolUse(t *testing.T) {
	sseData := `event: message_start
data: {"type":"message_start","message":{"id":"msg_2","type":"message","role":"assistant","content":[],"model":"claude-opus-4-6","usage":{"input_tokens":20,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_1","name":"search"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"query\":"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"\"test\"}"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":15}}

event: message_stop
data: {"type":"message_stop"}

`

	resp, err := parseAnthropicSSEStream(strings.NewReader(sseData), nil)
	if err != nil {
		t.Fatalf("parseAnthropicSSEStream failed: %v", err)
	}

	if resp.Choices[0].FinishReason != "tool_calls" {
		t.Errorf("finish_reason = %q, want tool_calls", resp.Choices[0].FinishReason)
	}

	toolCalls := resp.Choices[0].Message.ToolCalls
	if len(toolCalls) != 1 {
		t.Fatalf("tool_calls = %d, want 1", len(toolCalls))
	}
	if toolCalls[0].ID != "toolu_1" {
		t.Errorf("tool id = %q", toolCalls[0].ID)
	}
	if toolCalls[0].Function.Name != "search" {
		t.Errorf("tool name = %q", toolCalls[0].Function.Name)
	}
	if toolCalls[0].Function.Arguments != `{"query":"test"}` {
		t.Errorf("tool args = %q", toolCalls[0].Function.Arguments)
	}
}

func TestParseAnthropicSSEStream_Thinking(t *testing.T) {
	sseData := `event: message_start
data: {"type":"message_start","message":{"id":"msg_3","type":"message","role":"assistant","content":[],"model":"claude-opus-4-6","usage":{"input_tokens":5,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"Let me think..."}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: content_block_start
data: {"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"The answer is 42."}}

event: content_block_stop
data: {"type":"content_block_stop","index":1}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":10}}

event: message_stop
data: {"type":"message_stop"}

`
	var thinkingChunks []string
	var textChunks []string
	callback := func(content, thinking string, done bool) {
		if done {
			return
		}
		if thinking != "" {
			thinkingChunks = append(thinkingChunks, thinking)
		}
		if content != "" {
			textChunks = append(textChunks, content)
		}
	}

	resp, err := parseAnthropicSSEStream(strings.NewReader(sseData), callback)
	if err != nil {
		t.Fatalf("parseAnthropicSSEStream failed: %v", err)
	}

	if len(thinkingChunks) != 1 || thinkingChunks[0] != "Let me think..." {
		t.Errorf("thinking chunks = %v", thinkingChunks)
	}
	if len(textChunks) != 1 || textChunks[0] != "The answer is 42." {
		t.Errorf("text chunks = %v", textChunks)
	}
	if resp.Choices[0].Message.Content != "The answer is 42." {
		t.Errorf("content = %q", resp.Choices[0].Message.Content)
	}
}

// ---------------------------------------------------------------------------
// parseSSEStream dispatch
// ---------------------------------------------------------------------------

func TestParseSSEStream_Dispatch(t *testing.T) {
	// Minimal OpenAI SSE.
	sseData := "data: {\"id\":\"c1\",\"choices\":[{\"delta\":{\"content\":\"Hi\"},\"finish_reason\":null}]}\n\ndata: [DONE]\n\n"
	resp, err := parseSSEStream("openai", strings.NewReader(sseData), nil)
	if err != nil {
		t.Fatalf("dispatch to openai failed: %v", err)
	}
	if resp.Choices[0].Message.Content != "Hi" {
		t.Errorf("content = %q", resp.Choices[0].Message.Content)
	}
}

// ---------------------------------------------------------------------------
// Error classification
// ---------------------------------------------------------------------------

func TestParseAPIError(t *testing.T) {
	body := []byte(`{"error":{"type":"rate_limit_error","message":"Rate limit exceeded"}}`)
	err := parseAPIError(429, body)
	if !strings.Contains(err.Error(), "rate_limit_error") {
		t.Errorf("error = %q, should contain rate_limit_error", err)
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error = %q, should contain 429", err)
	}
}

func TestParseAPIError_Truncation(t *testing.T) {
	longMsg := strings.Repeat("x", 500)
	body := []byte(`{"error":{"type":"err","message":"` + longMsg + `"}}`)
	err := parseAPIError(500, body)
	if len(err.Error()) > 400 {
		t.Errorf("error message not truncated: len=%d", len(err.Error()))
	}
}

func TestIsContextLengthError(t *testing.T) {
	if IsContextLengthError(nil) {
		t.Error("nil should return false")
	}
	if IsContextLengthError(io.EOF) {
		t.Error("io.EOF should return false")
	}
	if IsContextLengthError(io.ErrUnexpectedEOF) {
		t.Error("ErrUnexpectedEOF should return false")
	}

	e := fmt.Errorf("context_length_exceeded: too long")
	if !IsContextLengthError(e) {
		t.Error("should detect context_length_exceeded")
	}
	e2 := fmt.Errorf("prompt is too long for this model")
	if !IsContextLengthError(e2) {
		t.Error("should detect 'prompt is too long'")
	}
}

func TestIsRateLimitError(t *testing.T) {
	e := fmt.Errorf("API error 429: rate limit")
	if !IsRateLimitError(e) {
		t.Error("should detect 429")
	}
	if IsRateLimitError(nil) {
		t.Error("nil should return false")
	}
}

// ---------------------------------------------------------------------------
// Client constructor
// ---------------------------------------------------------------------------

func TestNewClientWithProvider_ProviderDetection(t *testing.T) {
	c := NewClientWithProvider("https://api.openai.com/v1", "sk-test", "gpt-4o", "")
	if c.ProviderName() != "openai" {
		t.Errorf("expected openai, got %q", c.ProviderName())
	}

	c2 := NewClientWithProvider("https://api.anthropic.com/v1", "sk-ant-test", "claude-opus-4-6", "")
	if c2.ProviderName() != "anthropic" {
		t.Errorf("expected anthropic, got %q", c2.ProviderName())
	}
}

func TestClient_ReasoningEffort(t *testing.T) {
	c := NewClient("https://api.openai.com/v1", "sk-test", "gpt-4o")
	if c.GetReasoningEffort() != "" {
		t.Errorf("default reasoning effort should be empty")
	}

	c.SetReasoningEffort("high")
	if c.GetReasoningEffort() != "high" {
		t.Errorf("expected high, got %q", c.GetReasoningEffort())
	}
}

func TestClient_FallbackModels(t *testing.T) {
	c := NewClient("https://api.openai.com/v1", "sk-test", "gpt-4o")
	c.SetFallbackModels([]string{"gpt-3.5-turbo"}, 60)

	if len(c.fallbackModels) != 1 {
		t.Errorf("expected 1 fallback model, got %d", len(c.fallbackModels))
	}
	if c.fallbackTimeout != 60*time.Second {
		t.Errorf("expected 60s timeout, got %v", c.fallbackTimeout)
	}
}

// ---------------------------------------------------------------------------
// Response accessors
// ---------------------------------------------------------------------------

func TestResponseAccessors(t *testing.T) {
	c := NewClient("https://api.openai.com/v1", "sk-test", "gpt-4o")

	resp := &ChatResponse{
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: "Hello!",
					Thinking: "I should greet.",
					ToolCalls: []ToolCall{
						{ID: "call_1", Type: "function", Function: FunctionCall{Name: "greet", Arguments: "{}"}},
					},
				},
				FinishReason: "tool_calls",
			},
		},
	}

	if !c.HasToolCalls(resp) {
		t.Error("should have tool calls")
	}
	if len(c.GetToolCalls(resp)) != 1 {
		t.Error("should return 1 tool call")
	}
	if c.GetContent(resp) != "Hello!" {
		t.Error("content mismatch")
	}
	if c.GetThinking(resp) != "I should greet." {
		t.Error("thinking mismatch")
	}
	if c.GetFinishReason(resp) != "tool_calls" {
		t.Error("finish_reason mismatch")
	}

	// Empty response.
	empty := &ChatResponse{}
	if c.HasToolCalls(empty) {
		t.Error("empty response should not have tool calls")
	}
	if c.GetContent(empty) != "" {
		t.Error("empty response content should be empty")
	}
}

// ---------------------------------------------------------------------------
// ToolsToDefinitions
// ---------------------------------------------------------------------------

func TestToolsToDefinitions(t *testing.T) {
	tools := []map[string]any{
		{
			"function": map[string]any{
				"name":        "test_fn",
				"description": "A test function",
				"parameters":  map[string]any{"type": "object"},
			},
		},
		{
			"function": "invalid", // should be skipped
		},
		{
			"function": map[string]any{
				// missing "name" — should be skipped
				"description": "no name",
			},
		},
	}

	defs := ToolsToDefinitions(tools)
	if len(defs) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(defs))
	}
	if defs[0].Function.Name != "test_fn" {
		t.Errorf("name = %q", defs[0].Function.Name)
	}
}
