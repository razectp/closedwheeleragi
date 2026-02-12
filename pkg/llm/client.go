// Package llm provides multi-provider LLM client support.
// Provider-specific logic (endpoints, headers, request/response formats,
// SSE parsers) is consolidated in gollm_adapter.go. This file contains
// the Client struct, canonical types, and HTTP orchestration.
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

	"ClosedWheeler/pkg/utils"

	"github.com/teilomillet/gollm"
)

// ---------------------------------------------------------------------------
// Error classification helpers
// ---------------------------------------------------------------------------

// parseAPIError extracts a clean error message from an API error response body.
func parseAPIError(statusCode int, body []byte) error {
	var errBody struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
		Message string `json:"message"`
		Code    string `json:"code"`
	}
	if json.Unmarshal(body, &errBody) == nil {
		if errBody.Error.Message != "" {
			msg := errBody.Error.Message
			if len(msg) > 300 {
				msg = msg[:300] + "..."
			}
			if errBody.Error.Type != "" {
				return fmt.Errorf("API error %d [%s]: %s", statusCode, errBody.Error.Type, msg)
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

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

// Client handles communication with the LLM API. Internally it delegates
// provider detection to gollm (via mapProviderName) and uses the adapter
// layer for request building, response parsing, and SSE streaming.
type Client struct {
	baseURL         string
	apiKey          string
	model           string
	providerName    string   // canonical provider name (e.g. "openai", "anthropic")
	gollmLLM        gollm.LLM // optional gollm instance for simple queries
	fallbackModels  []string
	fallbackTimeout time.Duration
	reasoningEffort string
	httpClient      *http.Client
}

// ---------------------------------------------------------------------------
// Canonical types (unchanged — consumed by pkg/agent, pkg/tui, etc.)
// ---------------------------------------------------------------------------

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

// StreamOptions controls additional data returned during streaming.
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
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

// ChatResponse represents a chat completion response.
type ChatResponse struct {
	ID         string     `json:"id"`
	Object     string     `json:"object"`
	Created    int64      `json:"created"`
	Model      string     `json:"model"`
	Choices    []Choice   `json:"choices"`
	Usage      Usage      `json:"usage"`
	RateLimits RateLimits `json:"-"`
}

// Choice represents a response choice.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// RateLimits represents API rate limit information from headers.
type RateLimits struct {
	RemainingRequests int       `json:"remaining_requests"`
	RemainingTokens   int       `json:"remaining_tokens"`
	ResetRequests     time.Time `json:"reset_requests"`
	ResetTokens       time.Time `json:"reset_tokens"`
}

// ---------------------------------------------------------------------------
// Constructors
// ---------------------------------------------------------------------------

// NewClient creates a new LLM client with auto-detected provider (backward compatible).
func NewClient(baseURL, apiKey, model string) *Client {
	return NewClientWithProvider(baseURL, apiKey, model, "")
}

// NewClientWithProvider creates a new LLM client with an explicit provider name.
// An empty providerName triggers auto-detection based on model name and API key.
func NewClientWithProvider(baseURL, apiKey, model, providerName string) *Client {
	mapped := mapProviderName(providerName, model, apiKey, baseURL)

	// Create a gollm instance for simple queries. Non-critical: if it fails
	// we fall back to direct HTTP for everything.
	// NOTE: gollm's validator rejects API keys that don't match standard
	// provider formats (e.g. sk-... for OpenAI). This is expected for
	// third-party OpenAI-compatible endpoints, so we silently skip.
	g, _ := newGollmInstance(baseURL, apiKey, model, providerName)

	return &Client{
		baseURL:         baseURL,
		apiKey:          apiKey,
		model:           model,
		providerName:    mapped,
		gollmLLM:        g,
		fallbackModels:  []string{},
		fallbackTimeout: 30 * time.Second,
		// No Timeout on the http.Client: LLM responses can legitimately take
		// many minutes. Cancellation is handled via request context.
		httpClient: &http.Client{},
	}
}

// ---------------------------------------------------------------------------
// Accessors
// ---------------------------------------------------------------------------

// ProviderName returns the name of the active provider.
func (c *Client) ProviderName() string { return c.providerName }

// SetFallbackModels configures fallback models and timeout.
func (c *Client) SetFallbackModels(models []string, timeoutSeconds int) {
	c.fallbackModels = models
	if timeoutSeconds > 0 {
		c.fallbackTimeout = time.Duration(timeoutSeconds) * time.Second
	}
}

// SetReasoningEffort sets the reasoning effort level.
func (c *Client) SetReasoningEffort(effort string) {
	c.reasoningEffort = effort
}

// GetReasoningEffort returns the current reasoning effort level.
func (c *Client) GetReasoningEffort() string {
	return c.reasoningEffort
}

// ---------------------------------------------------------------------------
// Chat methods
// ---------------------------------------------------------------------------

// Chat sends a chat completion request.
func (c *Client) Chat(messages []Message, temperature *float64, topP *float64, maxTokens *int) (*ChatResponse, error) {
	return c.ChatWithTools(messages, nil, temperature, topP, maxTokens)
}

// ChatWithTools sends a chat completion request with function calling.
func (c *Client) ChatWithTools(messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int) (*ChatResponse, error) {
	return c.ChatWithToolsContext(context.Background(), messages, tools, temperature, topP, maxTokens)
}

// ChatWithToolsContext is like ChatWithTools but honours ctx for cancellation.
func (c *Client) ChatWithToolsContext(ctx context.Context, messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int) (*ChatResponse, error) {
	if len(c.fallbackModels) > 0 {
		return c.chatWithFallbackCtx(ctx, messages, tools, temperature, topP, maxTokens)
	}
	return c.chatWithModelCtx(ctx, c.model, messages, tools, temperature, topP, maxTokens, 0)
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

// chatWithModel is the legacy entry point (no context).
func (c *Client) chatWithModel(model string, messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int, timeout time.Duration) (*ChatResponse, error) {
	return c.chatWithModelCtx(context.Background(), model, messages, tools, temperature, topP, maxTokens, timeout)
}

// chatWithModelCtx is the core HTTP request logic, cancellable via ctx.
func (c *Client) chatWithModelCtx(ctx context.Context, model string, messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int, timeout time.Duration) (*ChatResponse, error) {

	jsonData, err := buildRequestBody(c.providerName, model, messages, tools, temperature, topP, maxTokens, false, c.reasoningEffort)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	reqCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var chatResp *ChatResponse
	operation := func() error {
		req, reqErr := http.NewRequestWithContext(reqCtx, "POST", endpointURL(c.baseURL, c.providerName), bytes.NewBuffer(jsonData))
		if reqErr != nil {
			return fmt.Errorf("failed to create request: %w", reqErr)
		}

		setProviderHeaders(req, c.providerName, c.apiKey)

		resp, doErr := c.httpClient.Do(req)
		if doErr != nil {
			return fmt.Errorf("failed to send request: %w", doErr)
		}
		defer resp.Body.Close()

		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("failed to read response: %w", readErr)
		}

		if resp.StatusCode != http.StatusOK {
			apiErr := parseAPIError(resp.StatusCode, body)
			if utils.IsRetryableError(resp.StatusCode) {
				if resp.StatusCode == http.StatusTooManyRequests {
					retryAfter := resp.Header.Get("retry-after")
					wait := 30 * time.Second
					if retryAfter != "" {
						var secs int
						if _, scanErr := fmt.Sscanf(retryAfter, "%d", &secs); scanErr == nil && secs > 0 {
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

		parsed, parseErr := parseResponseBody(c.providerName, body)
		if parseErr != nil {
			return parseErr
		}
		chatResp = parsed
		chatResp.RateLimits = parseRateLimitHeaders(c.providerName, resp.Header)

		return nil
	}

	retryConfig := utils.DefaultRetryConfig()
	if retryErr := utils.ExecuteWithRetry(operation, retryConfig); retryErr != nil {
		return nil, retryErr
	}

	return chatResp, nil
}

// ---------------------------------------------------------------------------
// Convenience methods
// ---------------------------------------------------------------------------

// SimpleQuery sends a simple chat query (no tools).
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

// QueryWithSystem sends a query with a system message.
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

// ---------------------------------------------------------------------------
// Response accessors
// ---------------------------------------------------------------------------

// HasToolCalls checks if the response contains tool calls.
func (c *Client) HasToolCalls(resp *ChatResponse) bool {
	if len(resp.Choices) == 0 {
		return false
	}
	return len(resp.Choices[0].Message.ToolCalls) > 0
}

// GetToolCalls extracts tool calls from response.
func (c *Client) GetToolCalls(resp *ChatResponse) []ToolCall {
	if len(resp.Choices) == 0 {
		return nil
	}
	return resp.Choices[0].Message.ToolCalls
}

// GetFinishReason returns the finish reason of the first choice.
func (c *Client) GetFinishReason(resp *ChatResponse) string {
	if len(resp.Choices) == 0 {
		return ""
	}
	return resp.Choices[0].FinishReason
}

// GetContent extracts the text content from response.
func (c *Client) GetContent(resp *ChatResponse) string {
	if len(resp.Choices) == 0 {
		return ""
	}
	return resp.Choices[0].Message.Content
}

// GetThinking extracts the reasoning content from response.
func (c *Client) GetThinking(resp *ChatResponse) string {
	if len(resp.Choices) == 0 {
		return ""
	}
	return resp.Choices[0].Message.Thinking
}

// ---------------------------------------------------------------------------
// Tool conversion
// ---------------------------------------------------------------------------

// ToolsToDefinitions converts tool registry format to LLM format.
func ToolsToDefinitions(tools []map[string]any) []ToolDefinition {
	defs := make([]ToolDefinition, 0, len(tools))
	for _, t := range tools {
		funcMap, ok := t["function"].(map[string]any)
		if !ok {
			continue
		}

		name, ok := funcMap["name"].(string)
		if !ok {
			continue
		}

		description, _ := funcMap["description"].(string)
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
