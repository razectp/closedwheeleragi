// Package llm provides streaming support for LLM APIs.
// SSE parsing is delegated to the adapter layer (gollm_adapter.go).
package llm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// StreamingCallback is called for each chunk of the response.
// chunk is for text content, thinking is for reasoning/thoughts.
type StreamingCallback func(content string, thinking string, done bool)

// StreamingDelta represents a streaming response delta.
type StreamingDelta struct {
	Content          string     `json:"content,omitempty"`
	Thinking         string     `json:"thinking,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

// StreamingChoice represents a streaming choice.
type StreamingChoice struct {
	Index        int            `json:"index"`
	Delta        StreamingDelta `json:"delta"`
	FinishReason string         `json:"finish_reason"`
}

// StreamingResponse represents a streaming response chunk.
type StreamingResponse struct {
	ID      string            `json:"id"`
	Object  string            `json:"object"`
	Created int64             `json:"created"`
	Model   string            `json:"model"`
	Choices []StreamingChoice `json:"choices"`
}

// ChatWithStreaming sends a chat request and streams the response.
func (c *Client) ChatWithStreaming(messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int, callback StreamingCallback) (*ChatResponse, error) {
	return c.ChatWithStreamingContext(context.Background(), messages, tools, temperature, topP, maxTokens, callback)
}

// ChatWithStreamingContext is like ChatWithStreaming but cancellable via ctx.
func (c *Client) ChatWithStreamingContext(ctx context.Context, messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int, callback StreamingCallback) (*ChatResponse, error) {

	jsonData, err := buildRequestBody(c.providerName, c.model, messages, tools, temperature, topP, maxTokens, true, c.reasoningEffort)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpointURL(c.baseURL, c.providerName), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	setProviderHeaders(req, c.providerName, c.apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		apiErr := parseAPIError(resp.StatusCode, body)
		if resp.StatusCode == http.StatusTooManyRequests {
			wait := 30 * time.Second
			if ra := resp.Header.Get("retry-after"); ra != "" {
				var secs int
				if _, scanErr := fmt.Sscanf(ra, "%d", &secs); scanErr == nil && secs > 0 {
					wait = time.Duration(secs) * time.Second
				}
			}
			log.Printf("[LLM] Rate limited (429, streaming). Waiting %s...", wait.Round(time.Second))
			time.Sleep(wait)
		}
		return nil, apiErr
	}

	return parseSSEStream(c.providerName, resp.Body, callback)
}

// SimpleQueryStreaming sends a simple query with streaming.
func (c *Client) SimpleQueryStreaming(prompt string, temperature *float64, topP *float64, maxTokens *int, callback StreamingCallback) (string, error) {
	messages := []Message{
		{Role: "user", Content: prompt},
	}

	resp, err := c.ChatWithStreaming(messages, nil, temperature, topP, maxTokens, callback)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return resp.Choices[0].Message.Content, nil
}
