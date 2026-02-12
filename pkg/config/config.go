// Package config provides configuration management.
package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"ClosedWheeler/pkg/ignore"
)

// Config holds all configuration settings
type Config struct {
	// API settings
	APIBaseURL      string `json:"api_base_url"`
	APIKey          string `json:"api_key"`
	Model           string `json:"model"`
	Provider        string `json:"provider,omitempty"`         // "openai", "anthropic", or "" for auto-detect
	ReasoningEffort string `json:"reasoning_effort,omitempty"` // "low", "medium", "high", "xhigh" for reasoning models

	// Fallback configuration
	FallbackModels  []string `json:"fallback_models,omitempty"`
	FallbackTimeout int      `json:"fallback_timeout,omitempty"` // Seconds before trying fallback

	// Documentation fields (Optional/Hidden in struct but present in JSON)
	BehaviorDoc    string `json:"// behavior_settings,omitempty"`
	TemperatureDoc string `json:"// temperature,omitempty"`
	TopPDoc        string `json:"// top_p,omitempty"`
	MaxTokensDoc   string `json:"// max_tokens,omitempty"`
	FallbackDoc    string `json:"// fallback_models,omitempty"`
	AutomationDoc  string `json:"// automation_settings,omitempty"`
	AnalysisDoc    string `json:"// analysis_settings,omitempty"`
	UIDoc          string `json:"// ui_settings,omitempty"`
	TelegramDoc    string `json:"// telegram_settings,omitempty"`
	PermissionsDoc string `json:"// permissions_settings,omitempty"`
	MemoryDoc      string `json:"// memory_settings,omitempty"`
	HeartbeatDoc   string `json:"// heartbeat_settings,omitempty"`

	// LLM behavior settings
	MaxTokens      *int     `json:"max_tokens,omitempty"`
	Temperature    *float64 `json:"temperature,omitempty"`
	TopP           *float64 `json:"top_p,omitempty"`
	MaxContextSize int      `json:"max_context_size"`

	// Memory settings
	Memory MemoryConfig `json:"memory"`

	// Improvement settings
	MinConfidenceScore float64 `json:"min_confidence_score"`
	MaxFilesPerBatch   int     `json:"max_files_per_batch"`
	BackupEnabled      bool    `json:"backup_enabled"`
	BackupPath         string  `json:"backup_path"`

	// Testing settings
	RunTestsBeforeApply bool   `json:"run_tests_before_apply"`
	TestCommand         string `json:"test_command"`

	// Analysis settings
	EnableCodeMetrics      bool     `json:"enable_code_metrics"`
	EnableSecurityAnalysis bool     `json:"enable_security_analysis"`
	EnablePerformanceCheck bool     `json:"enable_performance_check"`
	IgnorePatterns         []string `json:"ignore_patterns"`

	// UI settings
	UI UIConfig `json:"ui"`

	// Telegram settings
	Telegram TelegramConfig `json:"telegram"`

	// Permissions settings
	Permissions PermissionsConfig `json:"permissions"`

	// Heartbeat settings
	HeartbeatInterval int `json:"heartbeat_interval"` // Seconds between heartbeat checks

	// Debug settings
	DebugTools bool `json:"debug_tools"` // Enable detailed tool execution debugging

	// Git tools settings
	EnableGitTools bool `json:"enable_git_tools"` // Enable git tools (off by default, enable manually)

	// Browser settings
	Browser BrowserConfig `json:"browser"`

	// Model-specific parameters (for switching models)
	ModelParameters map[string]ModelParams `json:"model_parameters,omitempty"`
}

// BrowserConfig holds browser automation configuration
type BrowserConfig struct {
	Headless            bool `json:"headless"`
	SlowMo              int  `json:"slow_mo,omitempty"`
	RemoteDebuggingPort int  `json:"remote_debugging_port,omitempty"` // Port for remote debugging (0 = disabled/exec allocator)
}

// ModelParams holds parameters specific to a model
type ModelParams struct {
	Temperature   float64 `json:"temperature"`
	TopP          float64 `json:"top_p"`
	MaxTokens     int     `json:"max_tokens"`
	ContextWindow int     `json:"context_window"`
}

// MemoryConfig holds memory system configuration
type MemoryConfig struct {
	MaxShortTermItems  int    `json:"max_short_term_items"`
	MaxWorkingItems    int    `json:"max_working_items"`
	MaxLongTermItems   int    `json:"max_long_term_items"`
	CompressionTrigger int    `json:"compression_trigger"`
	StoragePath        string `json:"storage_path"`
}

// UIConfig holds UI configuration
type UIConfig struct {
	Theme         string `json:"theme"` // "dark", "light", "auto"
	ShowTokens    bool   `json:"show_tokens"`
	ShowTimestamp bool   `json:"show_timestamp"`
	Verbose       bool   `json:"verbose"`
}

// TelegramConfig holds Telegram bot configuration
type TelegramConfig struct {
	Enabled           bool   `json:"enabled"`
	BotToken          string `json:"bot_token"`
	ChatID            int64  `json:"chat_id"`
	NotifyOnToolStart bool   `json:"notify_on_tool_start"`
}

// PermissionsConfig holds global permissions configuration
type PermissionsConfig struct {
	// AllowedCommands defines which commands are permitted
	// Use "*" for all commands, or list specific commands
	AllowedCommands []string `json:"allowed_commands"`

	// AllowedTools defines which tools can be executed
	// Use "*" for all tools, or list specific tools
	AllowedTools []string `json:"allowed_tools"`

	// SensitiveTools defines tools requiring explicit approval
	SensitiveTools []string `json:"sensitive_tools"`

	// AutoApproveNonSensitive automatically approves non-sensitive tools
	AutoApproveNonSensitive bool `json:"auto_approve_non_sensitive"`

	// RequireApprovalForAll requires approval for all tool executions
	RequireApprovalForAll bool `json:"require_approval_for_all"`

	// TelegramApprovalTimeout defines timeout in seconds for Telegram approvals
	TelegramApprovalTimeout int `json:"telegram_approval_timeout"`

	// EnableAuditLog enables logging of all permission checks
	EnableAuditLog bool `json:"enable_audit_log"`

	// AuditLogPath defines where audit logs are stored
	AuditLogPath string `json:"audit_log_path"`
}

// DefaultConfig returns sensible defaults
func DefaultConfig() *Config {
	cfg := &Config{
		APIBaseURL:      "https://api.openai.com/v1",
		APIKey:          os.Getenv("OPENAI_API_KEY"),
		Model:           "gpt-4o-mini",
		FallbackModels:  []string{}, // Empty by default
		FallbackTimeout: 30,         // 30 seconds default timeout
		MaxTokens:       nil,
		Temperature:     nil,
		TopP:            nil,
		MaxContextSize:  128000,

		BehaviorDoc:    "Advanced LLM tuning (Optional)",
		TemperatureDoc: "Range 0.0 to 2.0. (Default: 1.0 or provider-specific)",
		TopPDoc:        "Nucleus sampling. (Default: 1.0)",
		MaxTokensDoc:   "Limit response size. (Default: No limit or model-specific)",
		FallbackDoc:    "List of fallback models to try if primary is slow/fails (Optional)",
		AutomationDoc:  "Automated backup and testing settings",
		AnalysisDoc:    "Code analysis, security and performance metrics",
		UIDoc:          "Terminal UI theme and verbosity settings",
		TelegramDoc:    "Telegram bot settings for remote monitoring and approval",
		PermissionsDoc: "Tool execution and security permissions",
		MemoryDoc:      "Tiered memory limits and context compression logic",
		HeartbeatDoc:   "Internal tick interval for self-correction (seconds)",

		Memory: MemoryConfig{
			MaxShortTermItems:  20,
			MaxWorkingItems:    50,
			MaxLongTermItems:   100,
			CompressionTrigger: 15,
			StoragePath:        ".agi/memory.json",
		},

		MinConfidenceScore: 0.7,
		MaxFilesPerBatch:   10,
		BackupEnabled:      true,
		BackupPath:         ".agi/backups",

		RunTestsBeforeApply:    true,
		TestCommand:            "go test ./...",
		EnableCodeMetrics:      true,
		EnableSecurityAnalysis: true,
		EnablePerformanceCheck: true,

		UI: UIConfig{
			Theme:         "dark",
			ShowTokens:    true,
			ShowTimestamp: true,
			Verbose:       false,
		},

		Telegram: TelegramConfig{
			Enabled:           false,
			BotToken:          "",
			ChatID:            0,
			NotifyOnToolStart: true,
		},

		Permissions: PermissionsConfig{
			AllowedCommands: []string{"*"}, // Allow all commands by default
			AllowedTools:    []string{"*"}, // Allow all tools by default
			SensitiveTools: []string{
				"git_commit",
				"git_push",
				"git_checkpoint",
				"exec_command",
				"write_file",
				"delete_file",
				"rollback_edits",
				"complete_edit",
				"install_skill",
			},
			AutoApproveNonSensitive: false, // Require approval for all by default
			RequireApprovalForAll:   false, // Only sensitive tools require approval
			TelegramApprovalTimeout: 300,   // 5 minutes timeout
			EnableAuditLog:          true,  // Enable audit logging by default
			AuditLogPath:            ".agi/audit.log",
		},

		HeartbeatInterval: 0, // Disabled by default

		DebugTools: false, // Disabled by default

		Browser: BrowserConfig{
			Headless:            false, // Run in background (user requested "navegador em background")
			SlowMo:              0,
			RemoteDebuggingPort: 9222, // Enable remote debugging port for "Launch & Connect" mode
		},
	}

	// Load patterns from .agiignore if it exists
	patterns := ignore.Load(".")
	cfg.IgnorePatterns = patterns.List()

	return cfg
}

// GetConfigPaths returns a prioritized list of configuration file paths
func GetConfigPaths(cliPath string) []string {
	var paths []string

	// 1. CLI Override
	if cliPath != "" {
		paths = append(paths, cliPath)
		return paths // If explicit, only use that
	}

	// 2. Project local paths
	paths = append(paths, ".agi/config.json")
	paths = append(paths, "configs/config.json")
	paths = append(paths, "config.json")

	// 3. User global path
	if homeDir, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(homeDir, ".agi", "config.json"))
		paths = append(paths, filepath.Join(homeDir, ".ClosedWheeler", "config.json"))
	}

	return paths
}

// Load loads configuration from the first available path in the prioritized list
func Load(cliPath string) (*Config, string, error) {
	// First, load .env file if it exists
	loadDotEnv()

	paths := GetConfigPaths(cliPath)

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			cfg := DefaultConfig()
			if err := json.Unmarshal(data, cfg); err != nil {
				return nil, path, err
			}
			// Apply overrides from env (this now includes .env variables)
			applyEnvOverrides(cfg)
			return cfg, path, nil
		}
	}

	// If no config found, use default and return the primary project-local path for saving
	defaultPath := ".agi/config.json"
	cfg := DefaultConfig()
	applyEnvOverrides(cfg)

	return cfg, defaultPath, cfg.Save(defaultPath)
}

// allowedEnvVars is a whitelist of environment variable names that may be set from .env
var allowedEnvVars = map[string]bool{
	"API_KEY":            true,
	"OPENAI_API_KEY":     true,
	"NVIDIA_API_KEY":     true,
	"ANTHROPIC_API_KEY":  true,
	"API_BASE_URL":       true,
	"OPENAI_BASE_URL":    true,
	"MODEL":              true,
	"OPENAI_MODEL":       true,
	"TELEGRAM_BOT_TOKEN": true,
	"TELEGRAM_CHAT_ID":   true,
	"VERBOSE":            true,
}

// loadDotEnv loads environment variables from .env file
func loadDotEnv() {
	envFile := ".env"
	file, err := os.Open(envFile)
	if err != nil {
		return // .env doesn't exist, that's ok
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Only allow whitelisted keys to prevent env injection
		if !allowedEnvVars[key] {
			continue
		}

		// Remove quotes if present
		value = strings.Trim(value, `"'`)

		// Set environment variable (only if not already set)
		if os.Getenv(key) == "" {
			if err := os.Setenv(key, value); err != nil {
				fmt.Printf("Warning: failed to set environment variable %s: %v\n", key, err)
			}
		}
	}
}

func applyEnvOverrides(cfg *Config) {
	// Support multiple environment variable names for different providers
	if apiKey := os.Getenv("API_KEY"); apiKey != "" {
		cfg.APIKey = apiKey
	} else if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		cfg.APIKey = apiKey
	} else if apiKey := os.Getenv("NVIDIA_API_KEY"); apiKey != "" {
		cfg.APIKey = apiKey
	} else if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		cfg.APIKey = apiKey
	}

	if baseURL := os.Getenv("API_BASE_URL"); baseURL != "" {
		cfg.APIBaseURL = baseURL
	} else if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
		cfg.APIBaseURL = baseURL
	}

	if model := os.Getenv("MODEL"); model != "" {
		cfg.Model = model
	} else if model := os.Getenv("OPENAI_MODEL"); model != "" {
		cfg.Model = model
	}

	if provider := os.Getenv("PROVIDER"); provider != "" {
		cfg.Provider = provider
	}

	// Telegram environment variables
	if botToken := os.Getenv("TELEGRAM_BOT_TOKEN"); botToken != "" {
		cfg.Telegram.BotToken = botToken
	}
	if chatIDStr := os.Getenv("TELEGRAM_CHAT_ID"); chatIDStr != "" {
		if chatID, err := strconv.ParseInt(chatIDStr, 10, 64); err == nil {
			cfg.Telegram.ChatID = chatID
		}
	}
}

// Save saves configuration to a file
func (c *Config) Save(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600) // 0600: owner read/write only (protects api_key)
}
