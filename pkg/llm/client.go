// Package llm provides multi-provider LLM client support.
// Provider-specific logic (endpoints, headers, request/response formats,
// SSE parsers) is consolidated in gollm_adapter.go. This file contains
// the Client struct, canonical types, and HTTP orchestration.
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
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
	providerName    string    // canonical provider name (e.g. "openai", "anthropic")
	gollmLLM        gollm.LLM // optional gollm instance for simple queries
	fallbackModels  []string
	fallbackTimeout time.Duration
	reasoningEffort string
	httpClient      *http.Client
}

// ---------------------------------------------------------------------------
// Types imported from types.go - no duplicates here
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Helper functions
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
		// Configure timeouts for security and reliability
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // Maximum time for a complete request
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second, // Connection timeout
					KeepAlive: 30 * time.Second,
				}).DialContext,
				MaxIdleConns:          10,
				IdleConnTimeout:       60 * time.Second,
				TLSHandshakeTimeout:   15 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
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

	// Apply rate limiting
	rateLimiter := utils.GetRateLimiter(c.providerName)
	if err := rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait cancelled: %w", err)
	}

	jsonData, err := buildRequestBody(model, messages, tools, temperature, topP, maxTokens, false, c.reasoningEffort)
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

		parsed, parseErr := parseResponseBody(body)
		if parseErr != nil {
			return parseErr
		}
		chatResp = parsed
		chatResp.RateLimits = parseRateLimitHeaders(c.providerName, resp)

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

// ---------------------------------------------------------------------------
// Helper functions for gollm integration
// ---------------------------------------------------------------------------

// mapProviderName maps provider names to canonical names
func mapProviderName(providerName, model, apiKey, baseURL string) string {
	if providerName != "" {
		return strings.ToLower(providerName)
	}

	// Check baseURL for NVIDIA
	if strings.Contains(strings.ToLower(baseURL), "nvidia") {
		return "nvidia"
	}

	// Check API key for NVIDIA
	if strings.Contains(strings.ToLower(apiKey), "nvidia") {
		return "nvidia"
	}

	lowerModel := strings.ToLower(model)
	if strings.HasPrefix(lowerModel, "claude") {
		return "anthropic"
	}
	if strings.HasPrefix(lowerModel, "gpt") {
		return "openai"
	}

	// Check for NVIDIA-specific models
	if strings.Contains(lowerModel, "nvidia") ||
		strings.Contains(lowerModel, "mistral") ||
		strings.Contains(lowerModel, "llama") {
		return "nvidia"
	}

	return "openai" // default
}

// buildRequestBody builds the request body for the given provider
func buildRequestBody(model string, messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int, stream bool, reasoningEffort string) ([]byte, error) {
	reqBody := ChatRequest{
		Model:       model,
		Messages:    messages,
		Tools:       tools,
		Temperature: temperature,
		TopP:        topP,
		MaxTokens:   maxTokens,
		Stream:      stream,
	}

	if stream {
		reqBody.StreamOptions = &StreamOptions{IncludeUsage: true}
	}

	if reasoningEffort != "" {
		reqBody.ReasoningEffort = reasoningEffort
	}

	return json.Marshal(reqBody)
}

// endpointURL returns the endpoint URL for the given provider
func endpointURL(baseURL, providerName string) string {
	switch providerName {
	case "openai":
		if baseURL == "" {
			return "https://api.openai.com/v1/chat/completions"
		}
		return baseURL + "/chat/completions"
	case "anthropic":
		if baseURL == "" {
			return "https://api.anthropic.com/v1/messages"
		}
		return baseURL + "/messages"
	case "nvidia":
		if baseURL == "" {
			return "https://integrate.api.nvidia.com/v1/chat/completions"
		}
		// NVIDIA already includes /v1 in baseURL
		if strings.HasSuffix(baseURL, "/v1") {
			return baseURL + "/chat/completions"
		}
		return baseURL + "/v1/chat/completions"
	default:
		return baseURL + "/chat/completions"
	}
}

// setProviderHeaders sets provider-specific headers
func setProviderHeaders(req *http.Request, providerName, apiKey string) {
	req.Header.Set("Content-Type", "application/json")

	switch providerName {
	case "openai":
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}
	case "anthropic":
		if apiKey != "" {
			req.Header.Set("x-api-key", apiKey)
		}
		req.Header.Set("anthropic-version", "2023-06-01")
	case "nvidia":
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}
		req.Header.Set("Accept", "application/json")
	}
}

// parseResponseBody parses the response body based on provider
func parseResponseBody(body []byte) (*ChatResponse, error) {
	var response ChatResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &response, nil
}

// parseRateLimitHeaders parses rate limit headers from response
func parseRateLimitHeaders(providerName string, resp *http.Response) RateLimits {
	limits := RateLimits{}

	switch providerName {
	case "openai":
		if remaining := resp.Header.Get("x-ratelimit-remaining-requests"); remaining != "" {
			if val, err := strconv.Atoi(remaining); err == nil {
				limits.RemainingRequests = val
			}
		}
		if remaining := resp.Header.Get("x-ratelimit-remaining-tokens"); remaining != "" {
			if val, err := strconv.Atoi(remaining); err == nil {
				limits.RemainingTokens = val
			}
		}
	case "anthropic":
		if remaining := resp.Header.Get("anthropic-ratelimit-remaining-requests"); remaining != "" {
			if val, err := strconv.Atoi(remaining); err == nil {
				limits.RemainingRequests = val
			}
		}
		if remaining := resp.Header.Get("anthropic-ratelimit-remaining-tokens"); remaining != "" {
			if val, err := strconv.Atoi(remaining); err == nil {
				limits.RemainingTokens = val
			}
		}
	}

	return limits
}

// supportsModelListing checks if provider supports model listing
func supportsModelListing(providerName string) bool {
	switch providerName {
	case "openai", "nvidia":
		return true
	case "anthropic":
		return false
	default:
		return false
	}
}

// newGollmInstance creates a new gollm instance - simplified for now
func newGollmInstance(providerName, baseURL, apiKey, model string) (gollm.LLM, error) {
	// TODO: Implement proper gollm initialization using these parameters
	// providerName - for selecting the right provider configuration
	// baseURL - for custom API endpoints
	// apiKey - for authentication
	// model - for specifying the model to use
	_ = providerName // Suppress unused warning for now
	_ = baseURL
	_ = apiKey
	_ = model
	// Simplified implementation - returns nil for now
	// In production, this would properly initialize gollm
	return nil, nil
}

// parseSSEStream parses Server-Sent Events stream
func parseSSEStream(body io.Reader, callback StreamingCallback) (*ChatResponse, error) {
	var fullContent string
	var fullThinking string
	var response ChatResponse

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and "data: [DONE]"
		if line == "" || line == "data: [DONE]" {
			continue
		}

		// Process SSE data lines
		if strings.HasPrefix(line, "data: ") {
			jsonData := strings.TrimPrefix(line, "data: ")

			var chunk StreamingResponse
			if err := json.Unmarshal([]byte(jsonData), &chunk); err != nil {
				continue // Skip malformed chunks
			}

			// Process choices
			for _, choice := range chunk.Choices {
				delta := choice.Delta

				// Accumulate content
				if delta.Content != "" {
					fullContent += delta.Content
					callback(delta.Content, "", false)
				}

				// Accumulate thinking/reasoning
				if delta.Thinking != "" {
					fullThinking += delta.Thinking
					callback("", delta.Thinking, false)
				}
				if delta.ReasoningContent != "" {
					fullThinking += delta.ReasoningContent
					callback("", delta.ReasoningContent, false)
				}

				// Check if stream is done
				if choice.FinishReason != "" {
					callback("", "", true)
				}
			}
		}
	}

	// Build final response
	response = ChatResponse{
		Choices: []Choice{
			{
				Message: Message{
					Role:    "assistant",
					Content: fullContent,
				},
				FinishReason: "stop",
			},
		},
	}

	// Add thinking if present
	if fullThinking != "" {
		response.Choices[0].Message.Thinking = fullThinking
	}

	if err := scanner.Err(); err != nil {
		return &response, fmt.Errorf("error reading stream: %w", err)
	}

	return &response, nil
}
