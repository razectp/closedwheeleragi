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

// AnthropicKnownModels is a hardcoded list of Anthropic models since the API
// does not provide a /models listing endpoint.
var AnthropicKnownModels = []ModelInfo{
	{ID: "claude-opus-4-20250514", Object: "model", OwnedBy: "anthropic"},
	{ID: "claude-sonnet-4-20250514", Object: "model", OwnedBy: "anthropic"},
	{ID: "claude-sonnet-4-5-20250929", Object: "model", OwnedBy: "anthropic"},
	{ID: "claude-haiku-4-5-20251001", Object: "model", OwnedBy: "anthropic"},
	{ID: "claude-3-5-sonnet-20241022", Object: "model", OwnedBy: "anthropic"},
	{ID: "claude-3-5-haiku-20241022", Object: "model", OwnedBy: "anthropic"},
	{ID: "claude-3-opus-20240229", Object: "model", OwnedBy: "anthropic"},
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
		return AnthropicKnownModels, nil
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
