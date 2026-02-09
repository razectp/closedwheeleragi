// Package llm provides streaming support for LLM APIs.
package llm

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

// StreamingCallback is called for each chunk of the response
type StreamingCallback func(chunk string, done bool)

// StreamingDelta represents a streaming response delta
type StreamingDelta struct {
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// StreamingChoice represents a streaming choice
type StreamingChoice struct {
	Index        int            `json:"index"`
	Delta        StreamingDelta `json:"delta"`
	FinishReason string         `json:"finish_reason"`
}

// StreamingResponse represents a streaming response chunk
type StreamingResponse struct {
	ID      string            `json:"id"`
	Object  string            `json:"object"`
	Created int64             `json:"created"`
	Model   string            `json:"model"`
	Choices []StreamingChoice `json:"choices"`
}

// ChatWithStreaming sends a chat request and streams the response
func (c *Client) ChatWithStreaming(messages []Message, tools []ToolDefinition, temperature *float64, topP *float64, maxTokens *int, callback StreamingCallback) (*ChatResponse, error) {
	jsonData, err := c.provider.BuildRequestBody(c.model, messages, tools, temperature, topP, maxTokens, true)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.provider.Endpoint(c.baseURL), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.provider.SetHeaders(req, c.apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Delegate SSE parsing to the provider
	return c.provider.ParseSSEStream(resp.Body, callback)
}

// SimpleQueryStreaming sends a simple query with streaming
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
