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
	"time"
)

const anthropicAPIVersion = "2023-06-01"
const anthropicDefaultMaxTokens = 4096

// AnthropicProvider implements the Provider interface for the Anthropic Messages API.
type AnthropicProvider struct{}

func (p *AnthropicProvider) Name() string { return "anthropic" }

func (p *AnthropicProvider) Endpoint(baseURL string) string {
	return baseURL + "/messages"
}

func (p *AnthropicProvider) SetHeaders(req *http.Request, apiKey string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", anthropicAPIVersion)

	if IsSetupToken(apiKey) {
		// OAuth/setup tokens use Bearer auth + beta header
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	} else {
		// Regular API keys use x-api-key
		req.Header.Set("x-api-key", apiKey)
	}
}

// --- Anthropic request types ---

type anthropicRequest struct {
	Model       string                 `json:"model"`
	Messages    []anthropicMessage     `json:"messages"`
	System      string                 `json:"system,omitempty"`
	MaxTokens   int                    `json:"max_tokens"`
	Temperature *float64               `json:"temperature,omitempty"`
	TopP        *float64               `json:"top_p,omitempty"`
	Tools       []anthropicTool        `json:"tools,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type anthropicMessage struct {
	Role    string        `json:"role"`
	Content []interface{} `json:"content"` // Can be text blocks, tool_use blocks, tool_result blocks
}

type anthropicTextBlock struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

type anthropicToolUseBlock struct {
	Type  string      `json:"type"` // "tool_use"
	ID    string      `json:"id"`
	Name  string      `json:"name"`
	Input interface{} `json:"input"`
}

type anthropicToolResultBlock struct {
	Type      string `json:"type"` // "tool_result"
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
	Type         string                 `json:"type"` // "message"
	Role         string                 `json:"role"`
	Content      []anthropicContentItem `json:"content"`
	Model        string                 `json:"model"`
	StopReason   string                 `json:"stop_reason"`
	StopSequence *string                `json:"stop_sequence"`
	Usage        anthropicUsage         `json:"usage"`
}

type anthropicContentItem struct {
	Type  string          `json:"type"` // "text" or "tool_use"
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
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
	Type  string                  `json:"type"`
	Index int                     `json:"index"`
	Delta anthropicDeltaContent   `json:"delta"`
}

type anthropicDeltaContent struct {
	Type         string `json:"type"` // "text_delta" or "input_json_delta"
	Text         string `json:"text,omitempty"`
	PartialJSON  string `json:"partial_json,omitempty"`
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

// --- Provider interface implementation ---

func (p *AnthropicProvider) BuildRequestBody(model string, messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int, stream bool) ([]byte, error) {
	// Extract and concatenate system messages
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

	// Convert messages to Anthropic format with role alternation enforcement
	anthropicMsgs := convertToAnthropicMessages(chatMessages)

	// Determine max_tokens (required by Anthropic)
	mt := anthropicDefaultMaxTokens
	if maxTokens != nil && *maxTokens > 0 {
		mt = *maxTokens
	}

	req := anthropicRequest{
		Model:       model,
		Messages:    anthropicMsgs,
		System:      systemPrompt,
		MaxTokens:   mt,
		Temperature: temperature,
		TopP:        topP,
		Stream:      stream,
	}

	// Convert tools
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
// handling tool_calls -> tool_use content blocks, tool results -> tool_result blocks,
// and merging consecutive same-role messages (Anthropic requires strict alternation).
func convertToAnthropicMessages(messages []Message) []anthropicMessage {
	var result []anthropicMessage

	for _, msg := range messages {
		var content []interface{}

		switch {
		case msg.Role == "assistant" && len(msg.ToolCalls) > 0:
			// Assistant message with tool calls -> tool_use content blocks
			if msg.Content != "" {
				content = append(content, anthropicTextBlock{Type: "text", Text: msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				// Parse arguments JSON to interface{} for Anthropic's input field
				var input interface{}
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err != nil {
					input = map[string]interface{}{} // fallback to empty object
				}
				content = append(content, anthropicToolUseBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: input,
				})
			}

		case msg.Role == "tool":
			// Tool result -> user message with tool_result content block
			content = append(content, anthropicToolResultBlock{
				Type:      "tool_result",
				ToolUseID: msg.ToolCallID,
				Content:   msg.Content,
			})

		default:
			// Regular text message
			text := msg.Content
			if text == "" {
				text = " " // Anthropic requires non-empty content
			}
			content = append(content, anthropicTextBlock{Type: "text", Text: text})
		}

		// Map role: tool -> user (Anthropic doesn't have a "tool" role)
		role := msg.Role
		if role == "tool" {
			role = "user"
		}

		amsg := anthropicMessage{
			Role:    role,
			Content: content,
		}

		// Merge with previous message if same role (Anthropic requires alternation)
		if len(result) > 0 && result[len(result)-1].Role == role {
			result[len(result)-1].Content = append(result[len(result)-1].Content, content...)
		} else {
			result = append(result, amsg)
		}
	}

	return result
}

func (p *AnthropicProvider) ParseResponseBody(body []byte) (*ChatResponse, error) {
	// Check top-level "type" field to distinguish error from success
	var resp anthropicResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Anthropic response: %w", err)
	}

	// Anthropic error responses have "type": "error"
	if resp.Type == "error" {
		var errResp anthropicError
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, fmt.Errorf("Anthropic API error (%s): %s", errResp.Error.Type, errResp.Error.Message)
		}
		return nil, fmt.Errorf("Anthropic API returned an error response: %s", string(body))
	}

	return p.convertResponse(&resp), nil
}

// convertResponse translates an Anthropic response to the canonical ChatResponse.
func (p *AnthropicProvider) convertResponse(resp *anthropicResponse) *ChatResponse {
	var content string
	var toolCalls []ToolCall

	for _, item := range resp.Content {
		switch item.Type {
		case "text":
			content += item.Text
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

	// Map stop_reason to finish_reason
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

func (p *AnthropicProvider) ParseRateLimits(h http.Header) RateLimits {
	rl := RateLimits{}
	// Anthropic uses anthropic-ratelimit-* headers
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

func (p *AnthropicProvider) ParseSSEStream(body io.Reader, callback StreamingCallback) (*ChatResponse, error) {
	reader := bufio.NewReader(body)

	var fullContent strings.Builder
	var toolCalls []ToolCall
	blockToToolIndex := make(map[int]int) // maps Anthropic block index -> toolCalls slice index
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

		// Parse event type
		if strings.HasPrefix(line, "event: ") {
			eventType := strings.TrimPrefix(line, "event: ")

			// Read the data line, skipping blank lines and comments
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
					blockToToolIndex[evt.Index] = len(toolCalls)
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
						callback(evt.Delta.Text, false)
					}
				case "input_json_delta":
					// Accumulate tool call arguments using block index mapping
					if idx, ok := blockToToolIndex[evt.Index]; ok && idx < len(toolCalls) {
						toolCalls[idx].Function.Arguments += evt.Delta.PartialJSON
					}
				}

			case "content_block_stop":
				// No action needed; block index mapping handles routing

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
					callback("", true)
				}

			case "ping":
				// Keepalive, ignore

			case "error":
				log.Printf("[ERROR] Anthropic stream error: %s", data)
				return nil, fmt.Errorf("Anthropic stream error: %s", data)
			}
			continue
		}

		// Fallback: handle bare "data: " lines (shouldn't happen with Anthropic but be safe)
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

func (p *AnthropicProvider) SupportsModelListing() bool { return false }
