// Package llm provides the gollm adapter that bridges this project's canonical
// types with the gollm library. It consolidates provider-specific logic
// (endpoint URLs, HTTP headers, request building, response parsing, SSE
// streaming) that previously lived in provider_openai.go and
// provider_anthropic.go into a single file.
package llm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/teilomillet/gollm"
)

// ---------------------------------------------------------------------------
// Provider name mapping
// ---------------------------------------------------------------------------

// mapProviderName determines the canonical provider name from explicit name,
// model prefix, API key prefix, or base URL. Returns a lowercase string
// compatible with gollm provider names.
func mapProviderName(providerName, model, apiKey, baseURL string) string {
	name := strings.ToLower(strings.TrimSpace(providerName))

	switch name {
	case "anthropic":
		return "anthropic"
	case "openai":
		return "openai"
	case "google":
		return "google"
	case "deepseek":
		return "deepseek"
	case "groq":
		return "groq"
	case "mistral":
		return "mistral"
	case "cohere":
		return "cohere"
	case "ollama":
		return "ollama"
	case "bedrock":
		return "bedrock"
	case "openrouter":
		return "openrouter"
	case "azure", "azure_openai":
		return "azure_openai"
	case "lmstudio":
		return "lmstudio"
	case "vllm":
		return "vllm"
	case "lambda":
		return "lambda"
	case "moonshot", "kimi":
		return "openai" // Moonshot uses OpenAI-compatible API
	}

	// Auto-detect by model name prefix.
	lowerModel := strings.ToLower(model)
	switch {
	case strings.HasPrefix(lowerModel, "claude"):
		return "anthropic"
	case strings.HasPrefix(lowerModel, "gpt") ||
		strings.HasPrefix(lowerModel, "o1") ||
		strings.HasPrefix(lowerModel, "o3") ||
		strings.HasPrefix(lowerModel, "o4"):
		return "openai"
	case strings.HasPrefix(lowerModel, "gemini"):
		return "google"
	case strings.HasPrefix(lowerModel, "deepseek"):
		return "deepseek"
	case strings.HasPrefix(lowerModel, "kimi"):
		return "openai"
	case strings.HasPrefix(lowerModel, "llama") ||
		strings.HasPrefix(lowerModel, "codellama") ||
		strings.HasPrefix(lowerModel, "mistral") ||
		strings.HasPrefix(lowerModel, "phi") ||
		strings.HasPrefix(lowerModel, "qwen"):
		// Local models — check baseURL for Ollama
		if strings.Contains(baseURL, ":11434") {
			return "ollama"
		}
		return "openai" // generic OpenAI-compatible
	}

	// Auto-detect by API key prefix.
	if strings.HasPrefix(apiKey, "sk-ant-") {
		return "anthropic"
	}

	// Auto-detect by base URL.
	if strings.Contains(baseURL, ":11434") {
		return "ollama"
	}

	// Default: OpenAI-compatible.
	return "openai"
}

// ---------------------------------------------------------------------------
// gollm instance factory
// ---------------------------------------------------------------------------

// newGollmInstance creates a configured gollm.LLM. It is used for simple
// Generate calls (interview, simple queries) where structured tool calls
// and streaming are not required.
func newGollmInstance(baseURL, apiKey, model, providerName string) (gollm.LLM, error) {
	mapped := mapProviderName(providerName, model, apiKey, baseURL)

	opts := []gollm.ConfigOption{
		gollm.SetProvider(mapped),
		gollm.SetModel(model),
		gollm.SetAPIKey(apiKey),
		gollm.SetLogLevel(gollm.LogLevelOff),
		gollm.SetMaxRetries(0), // we handle retry ourselves
	}

	// Custom endpoints.
	switch mapped {
	case "ollama":
		if baseURL != "" {
			opts = append(opts, gollm.SetOllamaEndpoint(baseURL))
		}
	}

	instance, err := gollm.NewLLM(opts...)
	if err != nil {
		return nil, fmt.Errorf("gollm init [%s/%s]: %w", mapped, model, err)
	}

	// For non-Ollama custom endpoints, set via the LLM interface.
	if baseURL != "" && mapped != "ollama" {
		instance.SetEndpoint(endpointURL(baseURL, mapped))
	}

	return instance, nil
}

// ---------------------------------------------------------------------------
// Endpoint URL
// ---------------------------------------------------------------------------

// endpointURL returns the full chat-completion endpoint for the given base URL
// and provider name.
func endpointURL(baseURL, providerName string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	switch providerName {
	case "anthropic":
		return baseURL + "/messages"
	default:
		return baseURL + "/chat/completions"
	}
}

// ---------------------------------------------------------------------------
// HTTP headers
// ---------------------------------------------------------------------------

// setProviderHeaders sets the required HTTP headers for the given provider.
func setProviderHeaders(req *http.Request, providerName, apiKey string) {
	req.Header.Set("Content-Type", "application/json")

	switch providerName {
	case "anthropic":
		req.Header.Set("anthropic-version", anthropicAPIVersion)
		req.Header.Set("accept", "application/json")
		req.Header.Set("x-api-key", apiKey)
	default:
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
}

// supportsModelListing returns true if the provider exposes a /models endpoint.
func supportsModelListing(providerName string) bool {
	switch providerName {
	case "anthropic", "google":
		return false
	default:
		return true
	}
}

// ---------------------------------------------------------------------------
// Anthropic constants and types (consolidated from provider_anthropic.go)
// ---------------------------------------------------------------------------

const anthropicAPIVersion = "2023-06-01"
const anthropicDefaultMaxTokens = 4096

// anthropicThinkingBudgets maps effort levels to budget_tokens.
var anthropicThinkingBudgets = map[string]int{
	"minimal": 1024,
	"low":     2048,
	"medium":  8192,
	"high":    16384,
	"xhigh":   16384,
}

// --- Anthropic request types ---

type anthropicRequest struct {
	Model       string                 `json:"model"`
	Messages    []anthropicMessage     `json:"messages"`
	System      interface{}            `json:"system,omitempty"`
	MaxTokens   int                    `json:"max_tokens"`
	Temperature *float64               `json:"temperature,omitempty"`
	TopP        *float64               `json:"top_p,omitempty"`
	Tools       []anthropicTool        `json:"tools,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Thinking    *anthropicThinking     `json:"thinking,omitempty"`
}

type anthropicThinking struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens"`
}

type anthropicMessage struct {
	Role    string        `json:"role"`
	Content []interface{} `json:"content"`
}

type anthropicTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicToolUseBlock struct {
	Type  string      `json:"type"`
	ID    string      `json:"id"`
	Name  string      `json:"name"`
	Input interface{} `json:"input"`
}

type anthropicToolResultBlock struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
}

type anthropicTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

// --- Anthropic response types ---

type anthropicResponse struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Role         string                 `json:"role"`
	Content      []anthropicContentItem `json:"content"`
	Model        string                 `json:"model"`
	StopReason   string                 `json:"stop_reason"`
	StopSequence *string                `json:"stop_sequence"`
	Usage        anthropicUsage         `json:"usage"`
}

type anthropicContentItem struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	Thinking string          `json:"thinking,omitempty"`
	ID       string          `json:"id,omitempty"`
	Name     string          `json:"name,omitempty"`
	Input    json.RawMessage `json:"input,omitempty"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type anthropicError struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// --- Anthropic SSE event types ---

type anthropicSSEMessageStart struct {
	Type    string            `json:"type"`
	Message anthropicResponse `json:"message"`
}

type anthropicSSEContentBlockStart struct {
	Type         string               `json:"type"`
	Index        int                  `json:"index"`
	ContentBlock anthropicContentItem `json:"content_block"`
}

type anthropicSSEContentBlockDelta struct {
	Type  string                `json:"type"`
	Index int                   `json:"index"`
	Delta anthropicDeltaContent `json:"delta"`
}

type anthropicDeltaContent struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
	Thinking    string `json:"thinking,omitempty"`
}

type anthropicSSEMessageDelta struct {
	Type  string `json:"type"`
	Delta struct {
		StopReason string `json:"stop_reason"`
	} `json:"delta"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// ---------------------------------------------------------------------------
// Request body building
// ---------------------------------------------------------------------------

// buildRequestBody creates the JSON request body for the specified provider.
func buildRequestBody(providerName, model string, messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int, stream bool, reasoningEffort string) ([]byte, error) {
	switch providerName {
	case "anthropic":
		return buildAnthropicRequestBody(model, messages, tools, temperature, topP, maxTokens, stream, reasoningEffort)
	default:
		return buildOpenAIRequestBody(model, messages, tools, temperature, topP, maxTokens, stream, reasoningEffort)
	}
}

// buildOpenAIRequestBody creates an OpenAI-compatible request body.
func buildOpenAIRequestBody(model string, messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int, stream bool, reasoningEffort string) ([]byte, error) {
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

	if len(tools) > 0 {
		reqBody.ToolChoice = "auto"
	}

	return json.Marshal(reqBody)
}

// buildAnthropicRequestBody creates an Anthropic Messages API request body.
func buildAnthropicRequestBody(model string, messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int, stream bool, reasoningEffort string) ([]byte, error) {
	// Extract and concatenate system messages.
	var systemParts []string
	var chatMessages []Message
	for _, msg := range messages {
		if msg.Role == "system" {
			if msg.Content != "" {
				systemParts = append(systemParts, msg.Content)
			}
		} else {
			chatMessages = append(chatMessages, msg)
		}
	}
	systemPrompt := strings.Join(systemParts, "\n\n")

	// Convert messages to Anthropic format with role alternation enforcement.
	anthropicMsgs := convertToAnthropicMessages(chatMessages)

	mt := anthropicDefaultMaxTokens
	if maxTokens != nil && *maxTokens > 0 {
		mt = *maxTokens
	}

	var systemField interface{}
	if systemPrompt != "" {
		systemField = systemPrompt
	}

	req := anthropicRequest{
		Model:       model,
		Messages:    anthropicMsgs,
		System:      systemField,
		MaxTokens:   mt,
		Temperature: temperature,
		TopP:        topP,
		Stream:      stream,
	}

	// Extended thinking.
	if reasoningEffort != "" && reasoningEffort != "off" {
		if budget, ok := anthropicThinkingBudgets[reasoningEffort]; ok {
			const modelMaxTokens = 128000
			const minOutputTokens = 1024
			newMax := req.MaxTokens + budget
			if newMax > modelMaxTokens {
				newMax = modelMaxTokens
			}
			if newMax <= budget {
				budget = newMax - minOutputTokens
				if budget < 0 {
					budget = 0
				}
			}
			if budget > 0 {
				req.Thinking = &anthropicThinking{
					Type:         "enabled",
					BudgetTokens: budget,
				}
				req.MaxTokens = newMax
			}
		}
	}

	if len(tools) > 0 {
		for _, t := range tools {
			req.Tools = append(req.Tools, anthropicTool{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				InputSchema: t.Function.Parameters,
			})
		}
	}

	return json.Marshal(req)
}

// convertToAnthropicMessages translates canonical messages to Anthropic format,
// handling tool_calls → tool_use content blocks, tool results → tool_result
// blocks, and merging consecutive same-role messages.
func convertToAnthropicMessages(messages []Message) []anthropicMessage {
	var result []anthropicMessage

	for _, msg := range messages {
		var content []interface{}

		switch {
		case msg.Role == "assistant" && len(msg.ToolCalls) > 0:
			if msg.Content != "" {
				content = append(content, anthropicTextBlock{Type: "text", Text: msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				var input interface{}
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err != nil {
					input = map[string]interface{}{}
				}
				content = append(content, anthropicToolUseBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: input,
				})
			}

		case msg.Role == "tool":
			content = append(content, anthropicToolResultBlock{
				Type:      "tool_result",
				ToolUseID: msg.ToolCallID,
				Content:   msg.Content,
			})

		default:
			text := msg.Content
			if text == "" {
				text = " " // Anthropic requires non-empty content.
			}
			content = append(content, anthropicTextBlock{Type: "text", Text: text})
		}

		role := msg.Role
		if role == "tool" {
			role = "user"
		}

		amsg := anthropicMessage{Role: role, Content: content}

		if len(result) > 0 && result[len(result)-1].Role == role {
			result[len(result)-1].Content = append(result[len(result)-1].Content, content...)
		} else {
			result = append(result, amsg)
		}
	}

	return result
}

// ---------------------------------------------------------------------------
// Response parsing
// ---------------------------------------------------------------------------

// parseResponseBody parses the raw JSON response from the provider into a
// canonical ChatResponse.
func parseResponseBody(providerName string, body []byte) (*ChatResponse, error) {
	switch providerName {
	case "anthropic":
		return parseAnthropicResponseBody(body)
	default:
		return parseOpenAIResponseBody(body)
	}
}

func parseOpenAIResponseBody(body []byte) (*ChatResponse, error) {
	var resp ChatResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &resp, nil
}

func parseAnthropicResponseBody(body []byte) (*ChatResponse, error) {
	var resp anthropicResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Anthropic response: %w", err)
	}

	if resp.Type == "error" {
		var errResp anthropicError
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, fmt.Errorf("Anthropic API error (%s): %s", errResp.Error.Type, errResp.Error.Message)
		}
		return nil, fmt.Errorf("Anthropic API returned an error response: %s", string(body))
	}

	return convertAnthropicResponse(&resp), nil
}

func convertAnthropicResponse(resp *anthropicResponse) *ChatResponse {
	var content string
	var thinking string
	var toolCalls []ToolCall

	for _, item := range resp.Content {
		switch item.Type {
		case "text":
			content += item.Text
		case "thinking":
			thinking += item.Thinking
		case "tool_use":
			argsJSON, err := json.Marshal(item.Input)
			if err != nil {
				log.Printf("[WARN] Failed to marshal tool_use input for %s: %v", item.Name, err)
				argsJSON = []byte("{}")
			}
			toolCalls = append(toolCalls, ToolCall{
				ID:   item.ID,
				Type: "function",
				Function: FunctionCall{
					Name:      item.Name,
					Arguments: string(argsJSON),
				},
			})
		}
	}

	finishReason := mapStopReason(resp.StopReason)

	return &ChatResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   resp.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:      "assistant",
					Content:   content,
					Thinking:  thinking,
					ToolCalls: toolCalls,
				},
				FinishReason: finishReason,
			},
		},
		Usage: Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
}

func mapStopReason(stopReason string) string {
	switch stopReason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "tool_use":
		return "tool_calls"
	case "stop_sequence":
		return "stop"
	default:
		return "stop"
	}
}

// ---------------------------------------------------------------------------
// Rate limit parsing
// ---------------------------------------------------------------------------

// parseRateLimitHeaders extracts rate limit information from response headers.
func parseRateLimitHeaders(providerName string, h http.Header) RateLimits {
	switch providerName {
	case "anthropic":
		return parseAnthropicRateLimits(h)
	default:
		return parseOpenAIRateLimits(h)
	}
}

func parseOpenAIRateLimits(h http.Header) RateLimits {
	rl := RateLimits{}
	if v := h.Get("x-ratelimit-remaining-requests"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			rl.RemainingRequests = val
		}
	}
	if v := h.Get("x-ratelimit-remaining-tokens"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			rl.RemainingTokens = val
		}
	}
	if v := h.Get("x-ratelimit-reset-requests"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			rl.ResetRequests = time.Now().Add(d)
		}
	}
	if v := h.Get("x-ratelimit-reset-tokens"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			rl.ResetTokens = time.Now().Add(d)
		}
	}
	return rl
}

func parseAnthropicRateLimits(h http.Header) RateLimits {
	rl := RateLimits{}
	if v := h.Get("anthropic-ratelimit-requests-remaining"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			rl.RemainingRequests = val
		}
	}
	if v := h.Get("anthropic-ratelimit-tokens-remaining"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			rl.RemainingTokens = val
		}
	}
	if v := h.Get("anthropic-ratelimit-requests-reset"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			rl.ResetRequests = t
		}
	}
	if v := h.Get("anthropic-ratelimit-tokens-reset"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			rl.ResetTokens = t
		}
	}
	return rl
}

// ---------------------------------------------------------------------------
// SSE stream parsing
// ---------------------------------------------------------------------------

// openAIStreamChunk is an extended streaming response that may include usage.
type openAIStreamChunk struct {
	StreamingResponse
	Usage *Usage `json:"usage,omitempty"`
}

// parseOpenAISSEStream parses an OpenAI-compatible SSE stream.
func parseOpenAISSEStream(body io.Reader, callback StreamingCallback) (*ChatResponse, error) {
	reader := bufio.NewReader(body)

	var fullContent strings.Builder
	var toolCalls []ToolCall
	var lastResponse StreamingResponse
	var finishReason string
	var usage Usage

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			if callback != nil {
				callback("", "", true)
			}
			break
		}

		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			log.Printf("[WARN] Skipping malformed streaming chunk: %v", err)
			continue
		}

		lastResponse = chunk.StreamingResponse

		if chunk.Usage != nil {
			usage = *chunk.Usage
		}

		if len(chunk.Choices) > 0 {
			choice := chunk.Choices[0]

			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}

			if choice.Delta.Content != "" {
				fullContent.WriteString(choice.Delta.Content)
				if callback != nil {
					callback(choice.Delta.Content, "", false)
				}
			}

			if choice.Delta.ReasoningContent != "" {
				if callback != nil {
					callback("", choice.Delta.ReasoningContent, false)
				}
			}

			if len(choice.Delta.ToolCalls) > 0 {
				for _, tc := range choice.Delta.ToolCalls {
					if tc.ID != "" {
						toolCalls = append(toolCalls, tc)
					} else if len(toolCalls) > 0 {
						last := &toolCalls[len(toolCalls)-1]
						last.Function.Arguments += tc.Function.Arguments
					}
				}
			}
		}
	}

	if finishReason == "" {
		finishReason = "stop"
	}

	return &ChatResponse{
		ID:      lastResponse.ID,
		Object:  "chat.completion",
		Created: lastResponse.Created,
		Model:   lastResponse.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:      "assistant",
					Content:   fullContent.String(),
					ToolCalls: toolCalls,
				},
				FinishReason: finishReason,
			},
		},
		Usage: usage,
	}, nil
}

// anthropicBlockState tracks tool call state during Anthropic SSE parsing.
type anthropicBlockState struct {
	mu               sync.Mutex
	blockToToolIndex map[int]int
}

// parseAnthropicSSEStream parses an Anthropic Messages API SSE stream.
func parseAnthropicSSEStream(body io.Reader, callback StreamingCallback) (*ChatResponse, error) {
	reader := bufio.NewReader(body)

	var fullContent strings.Builder
	var toolCalls []ToolCall
	state := &anthropicBlockState{blockToToolIndex: make(map[int]int)}
	var messageID, model string
	var inputTokens, outputTokens int
	var stopReason string

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		if strings.HasPrefix(line, "event: ") {
			eventType := strings.TrimPrefix(line, "event: ")

			var dataLine string
			for {
				dl, dlErr := reader.ReadString('\n')
				if dlErr != nil && dlErr != io.EOF {
					return nil, dlErr
				}
				dl = strings.TrimSpace(dl)
				if dl == "" || strings.HasPrefix(dl, ":") {
					if dlErr == io.EOF {
						break
					}
					continue
				}
				dataLine = dl
				break
			}
			if !strings.HasPrefix(dataLine, "data: ") {
				continue
			}
			data := strings.TrimPrefix(dataLine, "data: ")

			switch eventType {
			case "message_start":
				var evt anthropicSSEMessageStart
				if err := json.Unmarshal([]byte(data), &evt); err != nil {
					log.Printf("[WARN] Failed to parse message_start: %v", err)
					continue
				}
				messageID = evt.Message.ID
				model = evt.Message.Model
				inputTokens = evt.Message.Usage.InputTokens

			case "content_block_start":
				var evt anthropicSSEContentBlockStart
				if err := json.Unmarshal([]byte(data), &evt); err != nil {
					log.Printf("[WARN] Failed to parse content_block_start: %v", err)
					continue
				}
				if evt.ContentBlock.Type == "tool_use" {
					state.mu.Lock()
					state.blockToToolIndex[evt.Index] = len(toolCalls)
					state.mu.Unlock()
					toolCalls = append(toolCalls, ToolCall{
						ID:   evt.ContentBlock.ID,
						Type: "function",
						Function: FunctionCall{
							Name:      evt.ContentBlock.Name,
							Arguments: "",
						},
					})
				}

			case "content_block_delta":
				var evt anthropicSSEContentBlockDelta
				if err := json.Unmarshal([]byte(data), &evt); err != nil {
					log.Printf("[WARN] Failed to parse content_block_delta: %v", err)
					continue
				}
				switch evt.Delta.Type {
				case "text_delta":
					fullContent.WriteString(evt.Delta.Text)
					if callback != nil {
						callback(evt.Delta.Text, "", false)
					}
				case "thinking_delta":
					if callback != nil {
						callback("", evt.Delta.Thinking, false)
					}
				case "input_json_delta":
					state.mu.Lock()
					if idx, ok := state.blockToToolIndex[evt.Index]; ok && idx < len(toolCalls) {
						toolCalls[idx].Function.Arguments += evt.Delta.PartialJSON
					}
					state.mu.Unlock()
				}

			case "content_block_stop":
				// No action needed.

			case "message_delta":
				var evt anthropicSSEMessageDelta
				if err := json.Unmarshal([]byte(data), &evt); err != nil {
					log.Printf("[WARN] Failed to parse message_delta: %v", err)
					continue
				}
				stopReason = evt.Delta.StopReason
				outputTokens = evt.Usage.OutputTokens

			case "message_stop":
				if callback != nil {
					callback("", "", true)
				}

			case "ping":
				// Keepalive, ignore.

			case "error":
				log.Printf("[ERROR] Anthropic stream error: %s", data)
				return nil, fmt.Errorf("Anthropic stream error: %s", data)
			}
			continue
		}

		// Bare "data: " lines — skip.
		if strings.HasPrefix(line, "data: ") {
			continue
		}
	}

	finishReason := mapStopReason(stopReason)

	return &ChatResponse{
		ID:      messageID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:      "assistant",
					Content:   fullContent.String(),
					ToolCalls: toolCalls,
				},
				FinishReason: finishReason,
			},
		},
		Usage: Usage{
			PromptTokens:     inputTokens,
			CompletionTokens: outputTokens,
			TotalTokens:      inputTokens + outputTokens,
		},
	}, nil
}

// parseSSEStream dispatches to the correct SSE parser for the given provider.
func parseSSEStream(providerName string, body io.Reader, callback StreamingCallback) (*ChatResponse, error) {
	switch providerName {
	case "anthropic":
		return parseAnthropicSSEStream(body, callback)
	default:
		return parseOpenAISSEStream(body, callback)
	}
}
