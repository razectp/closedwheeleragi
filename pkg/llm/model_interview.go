// Package llm provides model self-configuration via interview
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// ModelSelfConfig represents a model's self-reported configuration
type ModelSelfConfig struct {
	ModelName         string   `json:"model_name"`
	ContextWindow     int      `json:"context_window"`
	RecommendedTemp   float64  `json:"recommended_temperature"`
	RecommendedTopP   float64  `json:"recommended_top_p"`
	RecommendedMaxTok int      `json:"recommended_max_tokens"`
	SupportsTemp      bool     `json:"supports_temperature"`
	SupportsTopP      bool     `json:"supports_top_p"`
	SupportsMaxTokens bool     `json:"supports_max_tokens"`
	BestForAgentWork  bool     `json:"best_for_agent_work"`
	Reasoning         string   `json:"reasoning"`
	Warnings          []string `json:"warnings,omitempty"`
}

// InterviewPrompt is the question asked to models for self-configuration
const InterviewPrompt = `You are being configured as an AI agent assistant. Please analyze your own capabilities and provide optimal configuration parameters.

**Your Task**: Return ONLY a JSON object (no markdown, no explanations) with this exact structure:

{
  "model_name": "your-model-id",
  "context_window": 128000,
  "recommended_temperature": 0.7,
  "recommended_top_p": 0.9,
  "recommended_max_tokens": 4096,
  "supports_temperature": true,
  "supports_top_p": true,
  "supports_max_tokens": true,
  "best_for_agent_work": true,
  "reasoning": "Brief explanation of why these parameters work best for you as an agent",
  "warnings": ["Optional array of any limitations or warnings"]
}

**Guidelines**:
- context_window: Your maximum context size in tokens
- recommended_temperature: Best temperature for agent work (0.0-1.0). Lower = more deterministic, higher = more creative. For agents, 0.6-0.8 is usually optimal.
- recommended_top_p: Best top_p for agent work (0.0-1.0). 0.9 is typically good for agents.
- recommended_max_tokens: Maximum tokens per response. Calculate as: min(context_window * 0.5, 8192). This ensures you can handle context + generate response.
- supports_*: Set to true if you support that parameter, false if not
- best_for_agent_work: Are you suitable for autonomous agent tasks requiring reasoning, tool use, and multi-step problem solving?
- reasoning: Explain in 1-2 sentences why these parameters work for you
- warnings: Array of strings with any important limitations (e.g., "May struggle with very long contexts", "Best for text, not code")

**Important**:
- Return ONLY valid JSON, no other text
- Be honest about your capabilities
- If unsure about a parameter, use conservative defaults
- For max_tokens, never exceed 50% of context_window
- For agent work, temperature 0.6-0.8 is usually best (balanced creativity + consistency)

Return your JSON now:`

// InterviewModel asks the model to configure itself
func (c *Client) InterviewModel(ctx context.Context) (*ModelSelfConfig, error) {
	log.Printf("[INFO] 🎤 Interviewing model: %s", c.model)
	log.Printf("[INFO] Asking model to self-configure for agent work...")

	messages := []Message{
		{Role: "user", Content: InterviewPrompt},
	}

	// Use minimal parameters for the interview itself
	temp := float64(0.3) // Low temp for structured output
	maxTok := int(2000)  // Enough for JSON response

	resp, err := c.chatWithModel(c.model, messages, nil, &temp, nil, &maxTok, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("interview failed: %w", err)
	}

	content := c.GetContent(resp)
	log.Printf("[DEBUG] Model response length: %d chars", len(content))

	// Clean response - remove markdown code blocks if present
	content = cleanJSONResponse(content)

	// Parse JSON
	var config ModelSelfConfig
	if err := json.Unmarshal([]byte(content), &config); err != nil {
		log.Printf("[ERROR] Failed to parse model response as JSON")
		log.Printf("[DEBUG] Response: %s", content)
		return nil, fmt.Errorf("model returned invalid JSON: %w", err)
	}

	// Validate and apply safety limits
	config = validateAndAdjustConfig(config)

	log.Printf("[INFO] ✅ Interview complete!")
	log.Printf("[INFO] Model self-reported config:")
	log.Printf("  Context Window:  %d tokens", config.ContextWindow)
	log.Printf("  Temperature:     %.2f", config.RecommendedTemp)
	log.Printf("  Top-P:           %.2f", config.RecommendedTopP)
	log.Printf("  Max Tokens:      %d", config.RecommendedMaxTok)
	log.Printf("  Agent-Ready:     %v", config.BestForAgentWork)
	if config.Reasoning != "" {
		log.Printf("  Reasoning:       %s", config.Reasoning)
	}
	if len(config.Warnings) > 0 {
		log.Printf("  Warnings:        %v", config.Warnings)
	}

	return &config, nil
}

// InterviewMultipleModels interviews all available models and returns their configs.
// providerName can be empty for auto-detection based on model name and API key.
func InterviewMultipleModels(baseURL, apiKey, providerName string, modelIDs []string) (map[string]*ModelSelfConfig, []string) {
	log.Printf("[INFO] 🎤 Interviewing %d models...", len(modelIDs))

	configs := make(map[string]*ModelSelfConfig)
	errors := make([]string, 0)

	for i, modelID := range modelIDs {
		log.Printf("[INFO] Interviewing model %d/%d: %s", i+1, len(modelIDs), modelID)

		client := NewClientWithProvider(baseURL, apiKey, modelID, providerName)
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)

		config, err := client.InterviewModel(ctx)
		cancel()

		if err != nil {
			errMsg := fmt.Sprintf("Model '%s' failed: %v", modelID, err)
			log.Printf("[WARN] ❌ %s", errMsg)
			errors = append(errors, errMsg)
			continue
		}

		configs[modelID] = config
		log.Printf("[INFO] ✅ Model '%s' configured successfully", modelID)
	}

	log.Printf("[INFO] Interview complete: %d succeeded, %d failed", len(configs), len(errors))
	return configs, errors
}

// cleanJSONResponse removes markdown code blocks and whitespace
func cleanJSONResponse(content string) string {
	// Remove markdown code blocks
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	// Find JSON object boundaries
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")

	if start != -1 && end != -1 && end > start {
		content = content[start : end+1]
	}

	return content
}

// validateAndAdjustConfig applies safety limits and validation
func validateAndAdjustConfig(config ModelSelfConfig) ModelSelfConfig {
	// Validate context window (min 1000, max 2M)
	if config.ContextWindow < 1000 {
		log.Printf("[WARN] Context window too small (%d), setting to 8000", config.ContextWindow)
		config.ContextWindow = 8000
	}
	if config.ContextWindow > 2000000 {
		log.Printf("[WARN] Context window suspiciously large (%d), capping at 2M", config.ContextWindow)
		config.ContextWindow = 2000000
	}

	// Validate temperature (0.0-1.0)
	if config.RecommendedTemp < 0.0 {
		config.RecommendedTemp = 0.0
	}
	if config.RecommendedTemp > 1.0 {
		config.RecommendedTemp = 1.0
	}

	// Validate top_p (0.0-1.0)
	if config.RecommendedTopP < 0.0 {
		config.RecommendedTopP = 0.0
	}
	if config.RecommendedTopP > 1.0 {
		config.RecommendedTopP = 1.0
	}

	// Calculate safe max_tokens (50% of context, max 8192)
	safeMaxTokens := config.ContextWindow / 2
	if safeMaxTokens > 8192 {
		safeMaxTokens = 8192
	}
	if safeMaxTokens < 512 {
		safeMaxTokens = 512
	}

	// Validate max_tokens doesn't exceed safe limit
	if config.RecommendedMaxTok > safeMaxTokens {
		log.Printf("[WARN] Max tokens (%d) exceeds safe limit (%d), adjusting",
			config.RecommendedMaxTok, safeMaxTokens)
		config.RecommendedMaxTok = safeMaxTokens
	}
	if config.RecommendedMaxTok < 256 {
		config.RecommendedMaxTok = 256
	}

	// Add warning if temperature too high for agent work
	if config.RecommendedTemp > 0.9 && config.BestForAgentWork {
		warning := "Temperature > 0.9 may cause inconsistent agent behavior"
		config.Warnings = append(config.Warnings, warning)
	}

	// Add warning if max_tokens very low
	if config.RecommendedMaxTok < 1024 {
		warning := "Low max_tokens may limit complex responses"
		config.Warnings = append(config.Warnings, warning)
	}

	return config
}

// ExportConfigsToJSON exports interviewed configs to JSON format
func ExportConfigsToJSON(configs map[string]*ModelSelfConfig) (string, error) {
	output := make(map[string]interface{})

	for modelID, config := range configs {
		output[modelID] = map[string]interface{}{
			"context_window":       config.ContextWindow,
			"recommended_temp":     config.RecommendedTemp,
			"recommended_top_p":    config.RecommendedTopP,
			"recommended_max_tok":  config.RecommendedMaxTok,
			"supports_temperature": config.SupportsTemp,
			"supports_top_p":       config.SupportsTopP,
			"supports_max_tokens":  config.SupportsMaxTokens,
			"best_for_agent_work":  config.BestForAgentWork,
			"reasoning":            config.Reasoning,
			"warnings":             config.Warnings,
		}
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

// ImportConfigsFromJSON imports model configs from JSON
func ImportConfigsFromJSON(jsonData string) (map[string]*ModelSelfConfig, error) {
	var rawConfigs map[string]map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &rawConfigs); err != nil {
		return nil, fmt.Errorf("invalid JSON format: %w", err)
	}

	configs := make(map[string]*ModelSelfConfig)

	for modelID, rawConfig := range rawConfigs {
		config := &ModelSelfConfig{
			ModelName:     modelID,
			ContextWindow: int(rawConfig["context_window"].(float64)),
		}

		if temp, ok := rawConfig["recommended_temp"].(float64); ok {
			config.RecommendedTemp = temp
		}
		if topP, ok := rawConfig["recommended_top_p"].(float64); ok {
			config.RecommendedTopP = topP
		}
		if maxTok, ok := rawConfig["recommended_max_tok"].(float64); ok {
			config.RecommendedMaxTok = int(maxTok)
		}
		if supportsTemp, ok := rawConfig["supports_temperature"].(bool); ok {
			config.SupportsTemp = supportsTemp
		}
		if supportsTopP, ok := rawConfig["supports_top_p"].(bool); ok {
			config.SupportsTopP = supportsTopP
		}
		if supportsMaxTok, ok := rawConfig["supports_max_tokens"].(bool); ok {
			config.SupportsMaxTokens = supportsMaxTok
		}
		if bestForAgent, ok := rawConfig["best_for_agent_work"].(bool); ok {
			config.BestForAgentWork = bestForAgent
		}
		if reasoning, ok := rawConfig["reasoning"].(string); ok {
			config.Reasoning = reasoning
		}
		if warnings, ok := rawConfig["warnings"].([]interface{}); ok {
			for _, w := range warnings {
				if warning, ok := w.(string); ok {
					config.Warnings = append(config.Warnings, warning)
				}
			}
		}

		configs[modelID] = config
	}

	return configs, nil
}

// CalculateDynamicMaxTokens calculates max_tokens as percentage of context window
func CalculateDynamicMaxTokens(contextWindow int, percentageMax float64) int {
	if percentageMax <= 0 || percentageMax > 1.0 {
		percentageMax = 0.5 // Default to 50%
	}

	maxTokens := int(float64(contextWindow) * percentageMax)

	// Apply safety limits
	if maxTokens < 256 {
		maxTokens = 256
	}
	if maxTokens > 8192 {
		maxTokens = 8192
	}

	return maxTokens
}
