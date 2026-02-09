package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ProvidersConfig holds configuration for all providers
type ProvidersConfig struct {
	Providers       []*Provider          `json:"providers"`
	PrimaryProvider string               `json:"primary_provider"`
	FallbackEnabled bool                 `json:"fallback_enabled"`
	AutoSwitch      bool                 `json:"auto_switch"` // Auto switch on failure
	DebateConfig    DebateConfiguration  `json:"debate_config"`
	Presets         map[string][]string  `json:"presets"` // Named provider groups
}

// DebateConfiguration holds debate-specific settings
type DebateConfiguration struct {
	AllowCrossProvider bool     `json:"allow_cross_provider"` // Allow debates between different providers
	DefaultPairings    []Pairing `json:"default_pairings"`     // Pre-configured pairings
	BalanceByModel     bool     `json:"balance_by_model"`     // Try to balance model capabilities
}

// Pairing represents a provider pairing for debates
type Pairing struct {
	Name        string `json:"name"`        // Display name
	ProviderA   string `json:"provider_a"`  // First provider ID
	ProviderB   string `json:"provider_b"`  // Second provider ID
	Description string `json:"description"` // What makes this pairing interesting
}

// DefaultProvidersConfig returns a default configuration
func DefaultProvidersConfig() *ProvidersConfig {
	return &ProvidersConfig{
		Providers: []*Provider{
			{
				ID:           "openai-gpt4",
				Name:         "OpenAI GPT-4",
				Type:         ProviderOpenAI,
				BaseURL:      "https://api.openai.com/v1",
				Model:        "gpt-4",
				Description:  "OpenAI's most capable model",
				MaxTokens:    8192,
				Temperature:  0.7,
				TopP:         1.0,
				Priority:     1,
				CostPerToken: 0.03,
				RateLimit:    60,
				Capabilities: []string{"streaming", "functions", "vision"},
				Enabled:      true,
			},
			{
				ID:           "openai-gpt35",
				Name:         "OpenAI GPT-3.5 Turbo",
				Type:         ProviderOpenAI,
				BaseURL:      "https://api.openai.com/v1",
				Model:        "gpt-3.5-turbo",
				Description:  "Fast and cost-effective",
				MaxTokens:    4096,
				Temperature:  0.7,
				TopP:         1.0,
				Priority:     2,
				CostPerToken: 0.002,
				RateLimit:    200,
				Capabilities: []string{"streaming", "functions"},
				Enabled:      true,
			},
		},
		PrimaryProvider: "openai-gpt4",
		FallbackEnabled: true,
		AutoSwitch:      true,
		DebateConfig: DebateConfiguration{
			AllowCrossProvider: true,
			DefaultPairings: []Pairing{
				{
					Name:        "GPT-4 vs GPT-3.5",
					ProviderA:   "openai-gpt4",
					ProviderB:   "openai-gpt35",
					Description: "Capability difference - advanced vs efficient",
				},
			},
			BalanceByModel: true,
		},
		Presets: map[string][]string{
			"all-openai": {"openai-gpt4", "openai-gpt35"},
			"fast":       {"openai-gpt35"},
			"powerful":   {"openai-gpt4"},
		},
	}
}

// LoadProvidersConfig loads provider configuration from file
func LoadProvidersConfig(path string) (*ProvidersConfig, error) {
	// If no path provided, use default location
	if path == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(homeDir, ".agi", "providers.json")
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Return default config if file doesn't exist
		return DefaultProvidersConfig(), nil
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// Parse JSON
	var config ProvidersConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// SaveProvidersConfig saves provider configuration to file
func SaveProvidersConfig(config *ProvidersConfig, path string) error {
	// If no path provided, use default location
	if path == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(homeDir, ".agi", "providers.json")
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// InitializeFromConfig initializes a ProviderManager from configuration
func InitializeFromConfig(config *ProvidersConfig) (*ProviderManager, error) {
	pm := NewProviderManager()

	// Add all providers
	for _, p := range config.Providers {
		if err := pm.AddProvider(p); err != nil {
			return nil, fmt.Errorf("failed to add provider %s: %w", p.ID, err)
		}
	}

	// Set primary provider
	if config.PrimaryProvider != "" {
		if err := pm.SetPrimaryProvider(config.PrimaryProvider); err != nil {
			return nil, fmt.Errorf("failed to set primary provider: %w", err)
		}
	}

	return pm, nil
}

// ExampleConfigs returns example configurations for different providers
func ExampleConfigs() map[string]*Provider {
	return map[string]*Provider{
		"openai-gpt4": {
			ID:           "openai-gpt4",
			Name:         "OpenAI GPT-4",
			Type:         ProviderOpenAI,
			BaseURL:      "https://api.openai.com/v1",
			Model:        "gpt-4",
			Description:  "Most capable OpenAI model",
			MaxTokens:    8192,
			Priority:     1,
			CostPerToken: 0.03,
			Capabilities: []string{"streaming", "functions", "vision"},
		},
		"openai-gpt4-turbo": {
			ID:           "openai-gpt4-turbo",
			Name:         "OpenAI GPT-4 Turbo",
			Type:         ProviderOpenAI,
			BaseURL:      "https://api.openai.com/v1",
			Model:        "gpt-4-turbo-preview",
			Description:  "Faster GPT-4 with 128k context",
			MaxTokens:    128000,
			Priority:     1,
			CostPerToken: 0.01,
			Capabilities: []string{"streaming", "functions", "vision"},
		},
		"anthropic-claude3-opus": {
			ID:           "anthropic-claude3-opus",
			Name:         "Claude 3 Opus",
			Type:         ProviderAnthropic,
			BaseURL:      "https://api.anthropic.com/v1",
			Model:        "claude-3-opus-20240229",
			Description:  "Anthropic's most intelligent model",
			MaxTokens:    200000,
			Priority:     1,
			CostPerToken: 0.015,
			Capabilities: []string{"streaming", "vision", "long-context"},
		},
		"anthropic-claude3-sonnet": {
			ID:           "anthropic-claude3-sonnet",
			Name:         "Claude 3 Sonnet",
			Type:         ProviderAnthropic,
			BaseURL:      "https://api.anthropic.com/v1",
			Model:        "claude-3-sonnet-20240229",
			Description:  "Balanced performance and speed",
			MaxTokens:    200000,
			Priority:     2,
			CostPerToken: 0.003,
			Capabilities: []string{"streaming", "vision", "long-context"},
		},
		"google-gemini-pro": {
			ID:           "google-gemini-pro",
			Name:         "Google Gemini Pro",
			Type:         ProviderGoogle,
			BaseURL:      "https://generativelanguage.googleapis.com/v1",
			Model:        "gemini-pro",
			Description:  "Google's multimodal AI",
			MaxTokens:    32000,
			Priority:     2,
			CostPerToken: 0.00025,
			Capabilities: []string{"streaming", "vision", "multimodal"},
		},
		"local-ollama": {
			ID:           "local-ollama",
			Name:         "Local Ollama",
			Type:         ProviderLocal,
			BaseURL:      "http://localhost:11434",
			Model:        "llama2",
			Description:  "Local model via Ollama",
			MaxTokens:    4096,
			Priority:     10,
			CostPerToken: 0.0, // Free!
			Capabilities: []string{"streaming"},
		},
	}
}

// SuggestPairingsForDebate suggests interesting provider pairings
func SuggestPairingsForDebate(pm *ProviderManager) []Pairing {
	providers := pm.GetEnabledProviders()
	pairings := []Pairing{}

	// Cross-provider pairings (different companies)
	for i, pA := range providers {
		for j, pB := range providers[i+1:] {
			_ = j // Use j to avoid unused variable warning
			if pA.Type != pB.Type {
				pairings = append(pairings, Pairing{
					Name:        fmt.Sprintf("%s vs %s", pA.Name, pB.Name),
					ProviderA:   pA.ID,
					ProviderB:   pB.ID,
					Description: fmt.Sprintf("Cross-provider: %s vs %s", pA.Type, pB.Type),
				})
			}
		}
	}

	// Same provider, different models (capability comparison)
	for i, pA := range providers {
		for j, pB := range providers[i+1:] {
			_ = j
			if pA.Type == pB.Type && pA.Model != pB.Model {
				pairings = append(pairings, Pairing{
					Name:        fmt.Sprintf("%s vs %s", pA.Model, pB.Model),
					ProviderA:   pA.ID,
					ProviderB:   pB.ID,
					Description: fmt.Sprintf("Model comparison within %s", pA.Type),
				})
			}
		}
	}

	return pairings
}
