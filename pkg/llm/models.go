// Package llm provides model discovery functionality
package llm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ModelInfo represents information about an available model
type ModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created,omitempty"`
	OwnedBy string `json:"owned_by,omitempty"`
}

// ModelsResponse represents the API response for listing models
type ModelsResponse struct {
	Object string      `json:"object"`
	Data   []ModelInfo `json:"data"`
}

// KnownModel extends ModelInfo with a human-readable hint for UI display.
type KnownModel struct {
	ModelInfo
	Hint string
}

// AnthropicKnownModels is a hardcoded list of Anthropic models since the API
// does not provide a /models listing endpoint.
var AnthropicKnownModels = []KnownModel{
	{ModelInfo: ModelInfo{ID: "claude-opus-4-6", Object: "model", OwnedBy: "anthropic"}, Hint: "200K · Latest flagship · Opus 4.6"},
	{ModelInfo: ModelInfo{ID: "claude-sonnet-4-5-20250929", Object: "model", OwnedBy: "anthropic"}, Hint: "200K · Fast + capable · Sonnet 4.5"},
	{ModelInfo: ModelInfo{ID: "claude-haiku-4-5-20251001", Object: "model", OwnedBy: "anthropic"}, Hint: "200K · Fast + cheap · Haiku 4.5"},
	{ModelInfo: ModelInfo{ID: "claude-opus-4-20250514", Object: "model", OwnedBy: "anthropic"}, Hint: "200K · Previous flagship · Opus 4"},
	{ModelInfo: ModelInfo{ID: "claude-sonnet-4-20250514", Object: "model", OwnedBy: "anthropic"}, Hint: "200K · Balanced · Sonnet 4"},
	{ModelInfo: ModelInfo{ID: "claude-3-5-sonnet-20241022", Object: "model", OwnedBy: "anthropic"}, Hint: "200K · Legacy · Sonnet 3.5"},
	{ModelInfo: ModelInfo{ID: "claude-3-5-haiku-20241022", Object: "model", OwnedBy: "anthropic"}, Hint: "200K · Legacy fast · Haiku 3.5"},
	{ModelInfo: ModelInfo{ID: "claude-3-opus-20240229", Object: "model", OwnedBy: "anthropic"}, Hint: "200K · Legacy flagship · Opus 3"},
}

// GoogleKnownModels is a hardcoded list of Google Gemini models.
var GoogleKnownModels = []KnownModel{
	{ModelInfo: ModelInfo{ID: "gemini-2.5-pro", Object: "model", OwnedBy: "google"}, Hint: "1M · Latest pro · Best reasoning"},
	{ModelInfo: ModelInfo{ID: "gemini-2.5-flash", Object: "model", OwnedBy: "google"}, Hint: "1M · Latest fast · Thinking"},
	{ModelInfo: ModelInfo{ID: "gemini-2.5-flash-lite", Object: "model", OwnedBy: "google"}, Hint: "1M · Lightweight"},
	{ModelInfo: ModelInfo{ID: "gemini-2.0-flash-001", Object: "model", OwnedBy: "google"}, Hint: "1M · Stable fast"},
	{ModelInfo: ModelInfo{ID: "gemini-2.0-flash-thinking-exp-01-21", Object: "model", OwnedBy: "google"}, Hint: "1M · Experimental thinking"},
	{ModelInfo: ModelInfo{ID: "gemini-1.5-pro", Object: "model", OwnedBy: "google"}, Hint: "2M · Long context pro"},
	{ModelInfo: ModelInfo{ID: "gemini-1.5-flash", Object: "model", OwnedBy: "google"}, Hint: "1M · Long context fast"},
}

// OpenAIKnownModels is a list of well-known OpenAI models for UI display.
var OpenAIKnownModels = []KnownModel{
	{ModelInfo: ModelInfo{ID: "gpt-5.3-codex", Object: "model", OwnedBy: "openai"}, Hint: "200K · Codex flagship · Reasoning"},
	{ModelInfo: ModelInfo{ID: "gpt-4o", Object: "model", OwnedBy: "openai"}, Hint: "128K · Fast multimodal"},
}

// DeepSeekKnownModels is a list of well-known DeepSeek models for UI display.
var DeepSeekKnownModels = []KnownModel{
	{ModelInfo: ModelInfo{ID: "deepseek-chat", Object: "model", OwnedBy: "deepseek"}, Hint: "128K · General purpose"},
	{ModelInfo: ModelInfo{ID: "deepseek-coder", Object: "model", OwnedBy: "deepseek"}, Hint: "128K · Code specialist"},
	{ModelInfo: ModelInfo{ID: "deepseek-reasoner", Object: "model", OwnedBy: "deepseek"}, Hint: "128K · Reasoning (R1)"},
}

// MoonshotKnownModels is a list of well-known Moonshot/Kimi models for UI display.
var MoonshotKnownModels = []KnownModel{
	{ModelInfo: ModelInfo{ID: "kimi-k2.5", Object: "model", OwnedBy: "moonshot"}, Hint: "256K · Kimi flagship · Free"},
}

// OllamaKnownModels is a list of popular Ollama local models for UI display.
var OllamaKnownModels = []KnownModel{
	{ModelInfo: ModelInfo{ID: "llama3", Object: "model", OwnedBy: "ollama"}, Hint: "8K · Meta general purpose"},
	{ModelInfo: ModelInfo{ID: "codellama", Object: "model", OwnedBy: "ollama"}, Hint: "16K · Code specialist"},
	{ModelInfo: ModelInfo{ID: "mistral", Object: "model", OwnedBy: "ollama"}, Hint: "32K · Fast general"},
	{ModelInfo: ModelInfo{ID: "deepseek-coder-v2", Object: "model", OwnedBy: "ollama"}, Hint: "128K · Code"},
	{ModelInfo: ModelInfo{ID: "phi3", Object: "model", OwnedBy: "ollama"}, Hint: "128K · Microsoft small"},
	{ModelInfo: ModelInfo{ID: "qwen2.5-coder", Object: "model", OwnedBy: "ollama"}, Hint: "128K · Code specialist"},
}

// knownToModelInfo converts a []KnownModel slice to []ModelInfo.
func knownToModelInfo(km []KnownModel) []ModelInfo {
	out := make([]ModelInfo, len(km))
	for i, m := range km {
		out[i] = m.ModelInfo
	}
	return out
}

// ListModels fetches available models from the API.
// For providers that don't support model listing (Anthropic), returns a hardcoded list.
func ListModels(baseURL, apiKey string) ([]ModelInfo, error) {
	return ListModelsWithProvider(baseURL, apiKey, "")
}

// ListModelsWithProvider fetches models using the specified provider.
func ListModelsWithProvider(baseURL, apiKey, providerName string) ([]ModelInfo, error) {
	provider := DetectProvider(providerName, "", apiKey)

	if !provider.SupportsModelListing() {
		return knownToModelInfo(AnthropicKnownModels), nil
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	url := baseURL + "/models"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	provider.SetHeaders(req, apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var modelsResp ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return modelsResp.Data, nil
}

// GetModelIDs extracts just the model IDs from ModelInfo list
func GetModelIDs(models []ModelInfo) []string {
	ids := make([]string, len(models))
	for i, m := range models {
		ids[i] = m.ID
	}
	return ids
}
