package llm

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapProviderName(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		model        string
		apiKey       string
		baseURL      string
		want         string
	}{
		{"explicit provider", "Anthropic", "gpt-4", "", "", "anthropic"},
		{"detect anthropic by model", "", "claude-3-5-sonnet", "", "", "anthropic"},
		{"detect openai by model", "", "gpt-4o", "", "", "openai"},
		{"detect ollama by port", "", "llama3", "", "http://localhost:11434", "ollama"},
		{"default to openai", "", "unknown-model", "", "", "openai"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapProviderName(tt.providerName, tt.model, tt.apiKey, tt.baseURL)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_Config(t *testing.T) {
	c := NewClient("https://api.openai.com/v1", "sk-test", "gpt-4o")
	require.NotNil(t, c)
	assert.Equal(t, "openai", c.ProviderName())

	// Reasoning effort
	c.SetReasoningEffort("high")
	assert.Equal(t, "high", c.GetReasoningEffort())

	// Fallback models
	fallbacks := []string{"gpt-4-turbo"}
	c.SetFallbackModels(fallbacks, 45)
	assert.Equal(t, fallbacks, c.fallbackModels)
	assert.Equal(t, 45*time.Second, c.fallbackTimeout)
}

func TestChatResponse_Accessors(t *testing.T) {
	c := &Client{}
	resp := &ChatResponse{
		Choices: []Choice{
			{
				Message: Message{
					Role:    "assistant",
					Content: "Hello world",
					ToolCalls: []ToolCall{
						{ID: "call_1", Type: "function"},
					},
				},
				FinishReason: "stop",
			},
		},
	}

	assert.True(t, c.HasToolCalls(resp))
	assert.Len(t, c.GetToolCalls(resp), 1)
	assert.Equal(t, "Hello world", c.GetContent(resp))
	assert.Equal(t, "stop", c.GetFinishReason(resp))

	// Empty response
	empty := &ChatResponse{}
	assert.False(t, c.HasToolCalls(empty))
	assert.Nil(t, c.GetToolCalls(empty))
	assert.Empty(t, c.GetContent(empty))
	assert.Empty(t, c.GetFinishReason(empty))
}

func TestErrorClassification(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		isCtxLen bool
		isRate   bool
	}{
		{"nil error", nil, false, false},
		{"context length error", fmt.Errorf("context_length_exceeded"), true, false},
		{"rate limit error", fmt.Errorf("429 Too Many Requests"), false, true},
		{"other error", fmt.Errorf("timeout"), false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isCtxLen, IsContextLengthError(tt.err))
			assert.Equal(t, tt.isRate, IsRateLimitError(tt.err))
		})
	}
}
