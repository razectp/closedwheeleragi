package config

import (
	"os"
	"testing"
)

func TestValidateAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		provider string
		wantErr  bool
	}{
		{
			name:     "Valid OpenAI key",
			key:      "sk-proj-1234567890abcdefghijklmnopqrstuvwxyz",
			provider: "openai",
			wantErr:  false,
		},
		{
			name:     "Invalid OpenAI key - wrong prefix",
			key:      "abc-1234567890",
			provider: "openai",
			wantErr:  true,
		},
		{
			name:     "Invalid OpenAI key - too short",
			key:      "sk-123",
			provider: "openai",
			wantErr:  true,
		},
		{
			name:     "Valid Anthropic key",
			key:      "sk-ant-api03-1234567890abcdefghijklmnopqrstuvwxyz",
			provider: "anthropic",
			wantErr:  false,
		},
		{
			name:     "Invalid Anthropic key - wrong prefix",
			key:      "sk-1234567890",
			provider: "anthropic",
			wantErr:  true,
		},
		{
			name:     "Valid NVIDIA key",
			key:      "nvapi-1234567890abcdefghijklmnopqrstuvwxyz",
			provider: "nvidia",
			wantErr:  false,
		},
		{
			name:     "Auto-detect OpenAI",
			key:      "sk-proj-1234567890abcdefghijklmnopqrstuvwxyz",
			provider: "",
			wantErr:  false,
		},
		{
			name:     "Auto-detect Anthropic",
			key:      "sk-ant-api03-1234567890abcdefghijklmnopqrstuvwxyz",
			provider: "",
			wantErr:  false,
		},
		{
			name:     "Empty key",
			key:      "",
			provider: "openai",
			wantErr:  true,
		},
		{
			name:     "Unknown provider - basic validation",
			key:      "12345678901234567890",
			provider: "unknown",
			wantErr:  false,
		},
		{
			name:     "Unknown provider - too short",
			key:      "123",
			provider: "unknown",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAPIKey(tt.key, tt.provider)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAPIKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "Valid HTTPS URL",
			url:     "https://api.openai.com/v1",
			wantErr: false,
		},
		{
			name:    "Valid HTTP URL with warning",
			url:     "http://api.openai.com/v1",
			wantErr: false,
		},
		{
			name:    "Invalid URL - no scheme",
			url:     "api.openai.com/v1",
			wantErr: true,
		},
		{
			name:    "Invalid URL - unsupported scheme",
			url:     "ftp://api.openai.com/v1",
			wantErr: true,
		},
		{
			name:    "Invalid URL - no host",
			url:     "https://",
			wantErr: true,
		},
		{
			name:    "Empty URL",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateLLMParams(t *testing.T) {
	tests := []struct {
		name     string
		temp     *float64
		topP     *float64
		maxTokens *int
		wantErr  bool
	}{
		{
			name:      "Valid parameters",
			temp:      ptrFloat64(0.7),
			topP:      ptrFloat64(0.9),
			maxTokens: ptrInt(1000),
			wantErr:   false,
		},
		{
			name:      "Temperature too low",
			temp:      ptrFloat64(-0.1),
			topP:      ptrFloat64(0.9),
			maxTokens: ptrInt(1000),
			wantErr:   true,
		},
		{
			name:      "Temperature too high",
			temp:      ptrFloat64(2.1),
			topP:      ptrFloat64(0.9),
			maxTokens: ptrInt(1000),
			wantErr:   true,
		},
		{
			name:      "TopP too low",
			temp:      ptrFloat64(0.7),
			topP:      ptrFloat64(-0.1),
			maxTokens: ptrInt(1000),
			wantErr:   true,
		},
		{
			name:      "TopP too high",
			temp:      ptrFloat64(0.7),
			topP:      ptrFloat64(1.1),
			maxTokens: ptrInt(1000),
			wantErr:   true,
		},
		{
			name:      "MaxTokens too low",
			temp:      ptrFloat64(0.7),
			topP:      ptrFloat64(0.9),
			maxTokens: ptrInt(0),
			wantErr:   true,
		},
		{
			name:      "MaxTokens too large",
			temp:      ptrFloat64(0.7),
			topP:      ptrFloat64(0.9),
			maxTokens: ptrInt(2000000),
			wantErr:   true,
		},
		{
			name:      "Nil parameters",
			temp:      nil,
			topP:      nil,
			maxTokens: nil,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLLMParams(tt.temp, tt.topP, tt.maxTokens)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateLLMParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "Valid config",
			cfg: &Config{
				APIKey:          "sk-proj-1234567890abcdefghijklmnopqrstuvwxyz",
				APIBaseURL:      "https://api.openai.com/v1",
				Model:           "gpt-4o-mini",
				MaxContextSize:  128000,
				MinConfidenceScore: 0.7,
				MaxFilesPerBatch:   10,
			},
			wantErr: false,
		},
		{
			name: "Invalid API key",
			cfg: &Config{
				APIKey:          "invalid-key",
				APIBaseURL:      "https://api.openai.com/v1",
				Model:           "gpt-4o-mini",
				MaxContextSize:  128000,
				MinConfidenceScore: 0.7,
				MaxFilesPerBatch:   10,
			},
			wantErr: true,
		},
		{
			name: "Invalid URL",
			cfg: &Config{
				APIKey:          "sk-proj-1234567890abcdefghijklmnopqrstuvwxyz",
				APIBaseURL:      "invalid-url",
				Model:           "gpt-4o-mini",
				MaxContextSize:  128000,
				MinConfidenceScore: 0.7,
				MaxFilesPerBatch:   10,
			},
			wantErr: true,
		},
		{
			name: "Invalid confidence score",
			cfg: &Config{
				APIKey:          "sk-proj-1234567890abcdefghijklmnopqrstuvwxyz",
				APIBaseURL:      "https://api.openai.com/v1",
				Model:           "gpt-4o-mini",
				MaxContextSize:  128000,
				MinConfidenceScore: 1.5,
				MaxFilesPerBatch:   10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfigValidation(t *testing.T) {
	cfg := DefaultConfig()
	
	// Default config should be valid except for missing API key
	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for missing API key")
	}
	
	// Add a valid API key and it should pass
	cfg.APIKey = "sk-proj-1234567890abcdefghijklmnopqrstuvwxyz"
	err = cfg.Validate()
	if err != nil {
		t.Errorf("Expected valid config with API key, got error: %v", err)
	}
}

func TestSecureDefaults(t *testing.T) {
	cfg := DefaultConfig()
	
	// Test secure browser defaults
	if !cfg.Browser.Headless {
		t.Error("Browser should be headless by default for security")
	}
	if !cfg.Browser.Stealth {
		t.Error("Browser should have stealth enabled by default")
	}
	if cfg.Browser.RemoteDebuggingPort != 0 {
		t.Error("Remote debugging port should be disabled by default")
	}
	
	// Test secure SSH defaults
	if cfg.SSH.Enabled {
		t.Error("SSH should be disabled by default")
	}
	if cfg.SSH.VisualMode {
		t.Error("SSH visual mode should be disabled by default")
	}
}

// Helper functions
func ptrFloat64(f float64) *float64 {
	return &f
}

func ptrInt(i int) *int {
	return &i
}

func TestLoadWithValidation(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.json"
	
	// Test with invalid config
	invalidConfig := `{
		"api_key": "invalid-key",
		"api_base_url": "invalid-url",
		"max_context_size": -1
	}`
	
	err := os.WriteFile(configPath, []byte(invalidConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	
	_, _, err = Load(configPath)
	if err == nil {
		t.Error("Expected validation error when loading invalid config")
	}
	
	// Test with valid config
	validConfig := `{
		"api_key": "sk-proj-1234567890abcdefghijklmnopqrstuvwxyz",
		"api_base_url": "https://api.openai.com/v1",
		"model": "gpt-4o-mini",
		"max_context_size": 128000
	}`
	
	err = os.WriteFile(configPath, []byte(validConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	
	cfg, _, err := Load(configPath)
	if err != nil {
		t.Errorf("Expected success when loading valid config, got error: %v", err)
	}
	
	if cfg.APIKey != "sk-proj-1234567890abcdefghijklmnopqrstuvwxyz" {
		t.Error("API key not loaded correctly")
	}
}
