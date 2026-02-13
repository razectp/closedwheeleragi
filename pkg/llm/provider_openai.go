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

// OpenAIProvider implements the Provider interface for OpenAI-compatible APIs.
type OpenAIProvider struct {
	reasoningEffort string // "low", "medium", "high", "xhigh"
}

func (p *OpenAIProvider) Name() string { return "openai" }

func (p *OpenAIProvider) Endpoint(baseURL string) string {
	return baseURL + "/chat/completions"
}

func (p *OpenAIProvider) SetHeaders(req *http.Request, apiKey string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
}

func (p *OpenAIProvider) BuildRequestBody(model string, messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int, stream bool) ([]byte, error) {
	reqBody := ChatRequest{
		Model:       model,
		Messages:    messages,
		Tools:       tools,
		Temperature: temperature,
		TopP:        topP,
		MaxTokens:   maxTokens,
		Stream:      stream,
	}

	// Request usage data in the final streaming chunk so token counts are available.
	if stream {
		reqBody.StreamOptions = &StreamOptions{IncludeUsage: true}
	}

	if p.reasoningEffort != "" {
		reqBody.ReasoningEffort = p.reasoningEffort
	}

	if len(tools) > 0 {
		reqBody.ToolChoice = "auto"
	}

	return json.Marshal(reqBody)
}

// SetReasoningEffort sets the reasoning effort level for reasoning models.
func (p *OpenAIProvider) SetReasoningEffort(effort string) { p.reasoningEffort = effort }

// GetReasoningEffort returns the current reasoning effort level.
func (p *OpenAIProvider) GetReasoningEffort() string { return p.reasoningEffort }

func (p *OpenAIProvider) ParseResponseBody(body []byte) (*ChatResponse, error) {
	var resp ChatResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &resp, nil
}

func (p *OpenAIProvider) ParseRateLimits(h http.Header) RateLimits {
	rl := RateLimits{}
	if v := h.Get("x-ratelimit-remaining-requests"); v != "" {
		val, err := strconv.Atoi(v)
		if err != nil {
			log.Printf("[WARN] Failed to parse x-ratelimit-remaining-requests: %v (value: %s)", err, v)
		} else {
			rl.RemainingRequests = val
		}
	}
	if v := h.Get("x-ratelimit-remaining-tokens"); v != "" {
		val, err := strconv.Atoi(v)
		if err != nil {
			log.Printf("[WARN] Failed to parse x-ratelimit-remaining-tokens: %v (value: %s)", err, v)
		} else {
			rl.RemainingTokens = val
		}
	}
	if v := h.Get("x-ratelimit-reset-requests"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			rl.ResetRequests = time.Now().Add(d).Unix()
		}
	}
	if v := h.Get("x-ratelimit-reset-tokens"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			rl.ResetTokens = time.Now().Add(d).Unix()
		}
	}
	return rl
}

// openAIStreamChunk is an extended streaming response that may include usage data.
// OpenAI sends a final chunk with choices=[] and usage populated when
// stream_options.include_usage is true.
type openAIStreamChunk struct {
	StreamingResponse
	Usage *Usage `json:"usage,omitempty"`
}

func (p *OpenAIProvider) ParseSSEStream(body io.Reader, callback StreamingCallback) (*ChatResponse, error) {
	reader := bufio.NewReader(body)

	var fullContent strings.Builder
	var toolCalls []ToolCall
	var lastResponse StreamingResponse
	var finishReason string
	var usage Usage // accumulated usage from the final chunk

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
			log.Printf("[WARN] Skipping malformed streaming chunk: %v (data: %s)", err, data)
			continue
		}

		lastResponse = chunk.StreamingResponse

		// Capture usage from the final chunk (stream_options.include_usage).
		if chunk.Usage != nil {
			usage = *chunk.Usage
		}

		if len(chunk.Choices) > 0 {
			choice := chunk.Choices[0]

			// Capture finish_reason from the stream
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

	finalResponse := &ChatResponse{
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
	}

	return finalResponse, nil
}

func (p *OpenAIProvider) SupportsModelListing() bool { return true }
