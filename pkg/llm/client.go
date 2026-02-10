// Package llm provides multi-provider LLM client support.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"ClosedWheeler/pkg/config"
	"ClosedWheeler/pkg/utils"
)

// parseAPIError extracts a clean error message from an API error response body.
// Instead of dumping the full JSON, it returns just the error type + message.
func parseAPIError(statusCode int, body []byte) error {
	// Try to extract a structured error message from JSON
	var errBody struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
		// OpenAI style
		Message string `json:"message"`
		Code    string `json:"code"`
	}
	if json.Unmarshal(body, &errBody) == nil {
		if errBody.Error.Message != "" {
			msg := errBody.Error.Message
			// Truncate very long messages (e.g. full prompt echo)
			if len(msg) > 300 {
				msg = msg[:300] + "..."
			}
			errType := errBody.Error.Type
			if errType != "" {
				return fmt.Errorf("API error %d [%s]: %s", statusCode, errType, msg)
			}
			return fmt.Errorf("API error %d: %s", statusCode, msg)
		}
		if errBody.Message != "" {
			msg := errBody.Message
			if len(msg) > 300 {
				msg = msg[:300] + "..."
			}
			return fmt.Errorf("API error %d: %s", statusCode, msg)
		}
	}
	// Fallback: truncate raw body
	raw := strings.TrimSpace(string(body))
	if len(raw) > 300 {
		raw = raw[:300] + "..."
	}
	return fmt.Errorf("API error %d: %s", statusCode, raw)
}

// IsContextLengthError returns true if the error is a context-length-exceeded error.
func IsContextLengthError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "context_length_exceeded") ||
		strings.Contains(s, "context window") ||
		strings.Contains(s, "prompt is too long") ||
		strings.Contains(s, "tokens_exceeded") ||
		strings.Contains(s, "maximum context length")
}

// IsRateLimitError returns true if the error is a rate-limit (429) error.
func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "429") ||
		strings.Contains(s, "rate_limit") ||
		strings.Contains(s, "too many requests")
}

// Client handles communication with the LLM API
type Client struct {
	baseURL         string
	apiKey          string
	model           string
	provider        Provider
	fallbackModels  []string
	fallbackTimeout time.Duration
	httpClient      *http.Client
}

// Message represents a chat message
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall represents a function call from the LLM
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall contains the function name and arguments
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolDefinition defines a tool for the LLM
type ToolDefinition struct {
	Type     string         `json:"type"`
	Function FunctionSchema `json:"function"`
}

// FunctionSchema defines a function's schema
type FunctionSchema struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model       string           `json:"model"`
	Messages    []Message        `json:"messages"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	ToolChoice  interface{}      `json:"tool_choice,omitempty"`
	Temperature *float64         `json:"temperature,omitempty"`
	TopP        *float64         `json:"top_p,omitempty"`
	MaxTokens       *int             `json:"max_tokens,omitempty"`
	Stream          bool             `json:"stream,omitempty"`
	ReasoningEffort string           `json:"reasoning_effort,omitempty"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	ID         string     `json:"id"`
	Object     string     `json:"object"`
	Created    int64      `json:"created"`
	Model      string     `json:"model"`
	Choices    []Choice   `json:"choices"`
	Usage      Usage      `json:"usage"`
	RateLimits RateLimits `json:"-"`
}

// Choice represents a response choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// RateLimits represents API rate limit information from headers
type RateLimits struct {
	RemainingRequests int       `json:"remaining_requests"`
	RemainingTokens   int       `json:"remaining_tokens"`
	ResetRequests     time.Time `json:"reset_requests"`
	ResetTokens       time.Time `json:"reset_tokens"`
}

// NewClient creates a new LLM client with auto-detected provider (backward compatible).
func NewClient(baseURL, apiKey, model string) *Client {
	return NewClientWithProvider(baseURL, apiKey, model, "")
}

// NewClientWithProvider creates a new LLM client with an explicit provider name.
// An empty providerName triggers auto-detection based on model name and API key.
func NewClientWithProvider(baseURL, apiKey, model, providerName string) *Client {
	return &Client{
		baseURL:         baseURL,
		apiKey:          apiKey,
		model:           model,
		provider:        DetectProvider(providerName, model, apiKey),
		fallbackModels:  []string{},
		fallbackTimeout: 30 * time.Second,
		// No Timeout on the http.Client: LLM responses can legitimately take many
		// minutes (deep research, long tool chains). Cancellation is handled via
		// request context (passed by the agent). Network-level connection timeout
		// is enforced by the OS TCP stack.
		httpClient: &http.Client{},
	}
}

// ProviderName returns the name of the active provider.
func (c *Client) ProviderName() string {
	return c.provider.Name()
}

// SetFallbackModels configures fallback models and timeout
func (c *Client) SetFallbackModels(models []string, timeoutSeconds int) {
	c.fallbackModels = models
	if timeoutSeconds > 0 {
		c.fallbackTimeout = time.Duration(timeoutSeconds) * time.Second
	}
}

// SetOAuthCredentials sets OAuth credentials on the underlying provider.
// Supports both Anthropic and OpenAI providers.
func (c *Client) SetOAuthCredentials(creds *config.OAuthCredentials) {
	switch p := c.provider.(type) {
	case *AnthropicProvider:
		p.SetOAuth(creds)
	case *OpenAIProvider:
		p.SetOAuth(creds)
	}
}

// GetOAuthCredentials returns the current OAuth credentials from the provider.
func (c *Client) GetOAuthCredentials() *config.OAuthCredentials {
	switch p := c.provider.(type) {
	case *AnthropicProvider:
		return p.GetOAuth()
	case *OpenAIProvider:
		return p.GetOAuth()
	}
	return nil
}

// SetReasoningEffort sets the reasoning effort level on the provider.
func (c *Client) SetReasoningEffort(effort string) {
	switch p := c.provider.(type) {
	case *OpenAIProvider:
		p.SetReasoningEffort(effort)
	case *AnthropicProvider:
		p.SetReasoningEffort(effort)
	}
}

// GetReasoningEffort returns the current reasoning effort level.
func (c *Client) GetReasoningEffort() string {
	switch p := c.provider.(type) {
	case *OpenAIProvider:
		return p.GetReasoningEffort()
	case *AnthropicProvider:
		return p.GetReasoningEffort()
	}
	return ""
}

// RefreshOAuthIfNeeded refreshes OAuth token if it's close to expiry.
// Called once before the request loop, not inside SetHeaders.
func (c *Client) RefreshOAuthIfNeeded() {
	switch p := c.provider.(type) {
	case *AnthropicProvider:
		p.RefreshIfNeeded()
	case *OpenAIProvider:
		p.RefreshIfNeeded()
	}
}

// Chat sends a chat completion request
func (c *Client) Chat(messages []Message, temperature *float64, topP *float64, maxTokens *int) (*ChatResponse, error) {
	return c.ChatWithTools(messages, nil, temperature, topP, maxTokens)
}

// ChatWithTools sends a chat completion request with function calling.
func (c *Client) ChatWithTools(messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int) (*ChatResponse, error) {
	return c.ChatWithToolsContext(context.Background(), messages, tools, temperature, topP, maxTokens)
}

// ChatWithToolsContext is like ChatWithTools but honours ctx for cancellation.
// Cancel the context to abort the in-flight HTTP request immediately.
func (c *Client) ChatWithToolsContext(ctx context.Context, messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int) (*ChatResponse, error) {
	if len(c.fallbackModels) > 0 {
		return c.chatWithFallbackCtx(ctx, messages, tools, temperature, topP, maxTokens)
	}
	return c.chatWithModelCtx(ctx, c.model, messages, tools, temperature, topP, maxTokens, 0)
}

// chatWithFallback attempts primary model with timeout, then fallback models
func (c *Client) chatWithFallback(messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int) (*ChatResponse, error) {
	return c.chatWithFallbackCtx(context.Background(), messages, tools, temperature, topP, maxTokens)
}

func (c *Client) chatWithFallbackCtx(ctx context.Context, messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int) (*ChatResponse, error) {
	resp, err := c.chatWithModelCtx(ctx, c.model, messages, tools, temperature, topP, maxTokens, c.fallbackTimeout)
	if err == nil {
		return resp, nil
	}

	log.Printf("[INFO] Primary model %s failed or timed out: %v. Trying fallback models...", c.model, err)

	for i, fallbackModel := range c.fallbackModels {
		log.Printf("[INFO] Attempting fallback model %d/%d: %s", i+1, len(c.fallbackModels), fallbackModel)
		resp, fallbackErr := c.chatWithModelCtx(ctx, fallbackModel, messages, tools, temperature, topP, maxTokens, c.fallbackTimeout)
		if fallbackErr == nil {
			log.Printf("[INFO] Fallback model %s succeeded!", fallbackModel)
			return resp, nil
		}
		log.Printf("[WARN] Fallback model %s failed: %v", fallbackModel, fallbackErr)
	}

	return nil, fmt.Errorf("all models failed, primary error: %w", err)
}

// chatWithModel is the legacy entry point (no context); delegates to chatWithModelCtx.
func (c *Client) chatWithModel(model string, messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int, timeout time.Duration) (*ChatResponse, error) {
	return c.chatWithModelCtx(context.Background(), model, messages, tools, temperature, topP, maxTokens, timeout)
}

// chatWithModelCtx is the core HTTP request logic, cancellable via ctx.
// When ctx is cancelled (e.g. user pressed Escape), the in-flight HTTP request
// is aborted immediately and the error is propagated as a cancellation.
func (c *Client) chatWithModelCtx(ctx context.Context, model string, messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int, timeout time.Duration) (*ChatResponse, error) {
	c.RefreshOAuthIfNeeded()

	jsonData, err := c.provider.BuildRequestBody(model, messages, tools, temperature, topP, maxTokens, false)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Wrap ctx with a per-fallback deadline only when the fallback timeout is set.
	// The outer http.Client has no Timeout — cancellation is exclusively via ctx.
	reqCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var chatResp *ChatResponse
	operation := func() error {
		req, err := http.NewRequestWithContext(reqCtx, "POST", c.provider.Endpoint(c.baseURL), bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		c.provider.SetHeaders(req, c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			apiErr := parseAPIError(resp.StatusCode, body)
			if utils.IsRetryableError(resp.StatusCode) {
				// For 429, wait longer before the retry kicks in
				if resp.StatusCode == http.StatusTooManyRequests {
					retryAfter := resp.Header.Get("retry-after")
					wait := 30 * time.Second
					if retryAfter != "" {
						var secs int
						if _, err2 := fmt.Sscanf(retryAfter, "%d", &secs); err2 == nil && secs > 0 {
							wait = time.Duration(secs) * time.Second
						}
					}
					log.Printf("[LLM] Rate limited (429). Waiting %s...", wait.Round(time.Second))
					time.Sleep(wait)
				}
				return apiErr
			}
			return apiErr
		}

		parsed, err := c.provider.ParseResponseBody(body)
		if err != nil {
			return err
		}
		chatResp = parsed

		// Parse rate limits from headers
		chatResp.RateLimits = c.provider.ParseRateLimits(resp.Header)

		return nil
	}

	retryConfig := utils.DefaultRetryConfig()
	if err := utils.ExecuteWithRetry(operation, retryConfig); err != nil {
		return nil, err
	}

	return chatResp, nil
}

// SimpleQuery sends a simple chat query (no tools)
func (c *Client) SimpleQuery(prompt string, temperature *float64, topP *float64, maxTokens *int) (string, error) {
	messages := []Message{
		{Role: "user", Content: prompt},
	}

	resp, err := c.Chat(messages, temperature, topP, maxTokens)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return resp.Choices[0].Message.Content, nil
}

// QueryWithSystem sends a query with a system message
func (c *Client) QueryWithSystem(systemPrompt, userPrompt string, temperature *float64, topP *float64, maxTokens *int) (string, error) {
	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	resp, err := c.Chat(messages, temperature, topP, maxTokens)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return resp.Choices[0].Message.Content, nil
}

// HasToolCalls checks if the response contains tool calls
func (c *Client) HasToolCalls(resp *ChatResponse) bool {
	if len(resp.Choices) == 0 {
		return false
	}
	return len(resp.Choices[0].Message.ToolCalls) > 0
}

// GetToolCalls extracts tool calls from response
func (c *Client) GetToolCalls(resp *ChatResponse) []ToolCall {
	if len(resp.Choices) == 0 {
		return nil
	}
	return resp.Choices[0].Message.ToolCalls
}

// GetFinishReason returns the finish reason of the first choice
func (c *Client) GetFinishReason(resp *ChatResponse) string {
	if len(resp.Choices) == 0 {
		return ""
	}
	return resp.Choices[0].FinishReason
}

// GetContent extracts the text content from response
func (c *Client) GetContent(resp *ChatResponse) string {
	if len(resp.Choices) == 0 {
		return ""
	}
	return resp.Choices[0].Message.Content
}

// ToolsToDefinitions converts tool registry format to LLM format
func ToolsToDefinitions(tools []map[string]any) []ToolDefinition {
	defs := make([]ToolDefinition, 0, len(tools))
	for _, t := range tools {
		// Safe type assertion for "function" field
		funcMap, ok := t["function"].(map[string]any)
		if !ok {
			// Skip malformed tool definition
			continue
		}

		// Safe type assertion for "name" field
		name, ok := funcMap["name"].(string)
		if !ok {
			continue
		}

		// Safe type assertion for "description" field
		description, ok := funcMap["description"].(string)
		if !ok {
			description = "" // Use empty string as fallback
		}

		// Parameters can be any type, so just assign directly
		parameters := funcMap["parameters"]

		def := ToolDefinition{
			Type: "function",
			Function: FunctionSchema{
				Name:        name,
				Description: description,
				Parameters:  parameters,
			},
		}
		defs = append(defs, def)
	}
	return defs
}
