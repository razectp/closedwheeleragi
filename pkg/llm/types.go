package llm

// Message represents a chat message.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	Thinking   string     `json:"thinking,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall represents a function call from the LLM.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall contains the function name and arguments.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolDefinition defines a tool for the LLM.
type ToolDefinition struct {
	Type     string         `json:"type"`
	Function FunctionSchema `json:"function"`
}

// FunctionSchema defines a function's schema.
type FunctionSchema struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// Usage contains token usage information for a request.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// RateLimits contains rate limit information.
type RateLimits struct {
	RemainingRequests int   `json:"remaining_requests"`
	RemainingTokens   int   `json:"remaining_tokens"`
	ResetRequests     int64 `json:"reset_requests"`
	ResetTokens       int64 `json:"reset_tokens"`
}

// ChatResponse represents a chat completion response.
type ChatResponse struct {
	ID         string     `json:"id"`
	Object     string     `json:"object"`
	Created    int64      `json:"created"`
	Model      string     `json:"model"`
	Choices    []Choice   `json:"choices"`
	Usage      Usage      `json:"usage"`
	RateLimits RateLimits `json:"rate_limits"`
}

// Choice represents a response choice.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// ChatRequest represents a chat completion request.
type ChatRequest struct {
	Model           string           `json:"model"`
	Messages        []Message        `json:"messages"`
	Tools           []ToolDefinition `json:"tools,omitempty"`
	ToolChoice      interface{}      `json:"tool_choice,omitempty"`
	Temperature     *float64         `json:"temperature,omitempty"`
	TopP            *float64         `json:"top_p,omitempty"`
	MaxTokens       *int             `json:"max_tokens,omitempty"`
	Stream          bool             `json:"stream,omitempty"`
	StreamOptions   *StreamOptions   `json:"stream_options,omitempty"`
	ReasoningEffort string           `json:"reasoning_effort,omitempty"`
}

// StreamOptions controls streaming behavior.
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// LLM Provider health info (used by model_profiles.go)
type ProviderHealth struct {
	ProviderName string  `json:"provider"`
	Latency      float64 `json:"latency_ms"`
	SuccessRate  float64 `json:"success_rate"`
	ErrorCount   int     `json:"error_count"`
	LastTested   int64   `json:"last_tested"`
}
