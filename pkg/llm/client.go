// Package llm provides multi-provider LLM client support.
package llm

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"ClosedWheeler/pkg/utils"
)

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
	MaxTokens   *int             `json:"max_tokens,omitempty"`
	Stream      bool             `json:"stream,omitempty"`
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
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
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

// Chat sends a chat completion request
func (c *Client) Chat(messages []Message, temperature *float64, topP *float64, maxTokens *int) (*ChatResponse, error) {
	return c.ChatWithTools(messages, nil, temperature, topP, maxTokens)
}

// ChatWithTools sends a chat completion request with function calling
func (c *Client) ChatWithTools(messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int) (*ChatResponse, error) {
	// If we have fallback models, use timeout for primary model
	if len(c.fallbackModels) > 0 {
		return c.chatWithFallback(messages, tools, temperature, topP, maxTokens)
	}

	// No fallback configured, use normal flow
	return c.chatWithModel(c.model, messages, tools, temperature, topP, maxTokens, 0)
}

// chatWithFallback attempts primary model with timeout, then fallback models
func (c *Client) chatWithFallback(messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int) (*ChatResponse, error) {
	// Try primary model with timeout
	resp, err := c.chatWithModel(c.model, messages, tools, temperature, topP, maxTokens, c.fallbackTimeout)
	if err == nil {
		return resp, nil
	}

	log.Printf("[INFO] Primary model %s failed or timed out: %v. Trying fallback models...", c.model, err)

	// Try each fallback model
	for i, fallbackModel := range c.fallbackModels {
		log.Printf("[INFO] Attempting fallback model %d/%d: %s", i+1, len(c.fallbackModels), fallbackModel)

		// Use same timeout for fallback models
		resp, fallbackErr := c.chatWithModel(fallbackModel, messages, tools, temperature, topP, maxTokens, c.fallbackTimeout)
		if fallbackErr == nil {
			log.Printf("[INFO] Fallback model %s succeeded!", fallbackModel)
			return resp, nil
		}

		log.Printf("[WARN] Fallback model %s failed: %v", fallbackModel, fallbackErr)
	}

	// All models failed, return original error
	return nil, fmt.Errorf("all models failed, primary error: %w", err)
}

// chatWithModel attempts to chat with a specific model, with optional timeout
func (c *Client) chatWithModel(model string, messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int, timeout time.Duration) (*ChatResponse, error) {
	jsonData, err := c.provider.BuildRequestBody(model, messages, tools, temperature, topP, maxTokens, false)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create a temporary HTTP client with custom timeout if specified
	httpClient := c.httpClient
	if timeout > 0 {
		httpClient = &http.Client{
			Timeout: timeout,
		}
	}

	var chatResp *ChatResponse
	operation := func() error {
		req, err := http.NewRequest("POST", c.provider.Endpoint(c.baseURL), bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		c.provider.SetHeaders(req, c.apiKey)

		resp, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			apiErr := fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
			if utils.IsRetryableError(resp.StatusCode) {
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
