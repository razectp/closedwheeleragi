// Package config provides configuration management.
package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"ClosedWheeler/pkg/ignore"
)

// Config holds all configuration settings
type Config struct {
	// Agent identity
	AgentName string `json:"agent_name,omitempty"` // Display name for the agent
	UserName  string `json:"user_name,omitempty"`  // Display name for the user

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
	HeartbeatInterval      int `json:"heartbeat_interval"`                 // Seconds between heartbeat checks
	HeartbeatIdleThreshold int `json:"heartbeat_idle_threshold,omitempty"` // Seconds of inactivity before heartbeat can act (default: 30)

	// Debug settings
	DebugTools bool `json:"debug_tools"` // Enable detailed tool execution debugging

	// Git tools settings
	EnableGitTools bool `json:"enable_git_tools"` // Enable git tools (off by default, enable manually)

	// SSH settings
	SSH SSHConfig `json:"ssh"`

	// Browser settings
	Browser BrowserConfig `json:"browser"`

	// Advanced settings (optional, zero-values use defaults)
	WorkplaceDir       string `json:"workplace_dir,omitempty"`           // Sandbox directory name (default: "workplace")
	BrowserViewportW   int    `json:"browser_viewport_width,omitempty"`  // Browser viewport width (default: 1920)
	BrowserViewportH   int    `json:"browser_viewport_height,omitempty"` // Browser viewport height (default: 1080)
	MaxRuleFileSize    int    `json:"max_rule_file_size,omitempty"`      // Max size per rule file in KB (default: 50)
	PipelineRoleDelay  int    `json:"pipeline_role_delay_ms,omitempty"`  // Delay between pipeline roles in ms (default: 1500)
	SessionMaxMessages int    `json:"session_max_messages,omitempty"`    // Max messages per session (default: 1000)

	// MCP (Model Context Protocol) servers
	MCPServers []MCPServerConfig `json:"mcp_servers,omitempty"`

	// Model-specific parameters (for switching models)
	ModelParameters map[string]ModelParams `json:"model_parameters,omitempty"`
}

// MCPServerConfig describes a single MCP server connection in the config file.
type MCPServerConfig struct {
	Name      string   `json:"name"`
	Transport string   `json:"transport"` // "stdio" or "sse"
	Command   string   `json:"command,omitempty"`
	Args      []string `json:"args,omitempty"`
	Env       []string `json:"env,omitempty"`
	URL       string   `json:"url,omitempty"`
	Enabled   bool     `json:"enabled"`
}

// SSHConfig holds all SSH-related configuration.
type SSHConfig struct {
	Enabled      bool            `json:"enabled"`                 // Enable SSH tools (default: false)
	VisualMode   bool            `json:"visual_mode"`             // Open monitor window (default: true)
	Hosts        []SSHHostConfig `json:"hosts,omitempty"`         // Pre-configured hosts
	DenyCommands []string        `json:"deny_commands,omitempty"` // Global SSH command deny patterns
}

// SSHHostConfig describes a pre-configured SSH host.
type SSHHostConfig struct {
	Label        string   `json:"label"`
	Host         string   `json:"host"`
	Port         string   `json:"port,omitempty"` // Default: "22"
	User         string   `json:"user,omitempty"`
	Password     string   `json:"password,omitempty"`
	KeyFile      string   `json:"key_file,omitempty"`
	DenyCommands []string `json:"deny_commands,omitempty"` // Per-host deny patterns (merged with global)
}

// BrowserConfig holds browser automation configuration
type BrowserConfig struct {
	Headless            bool `json:"headless"`
	Stealth             bool `json:"stealth"`
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
		AgentName:       "ClosedWheeler",
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
				"ssh_connect",
				"ssh_exec",
				"ssh_upload",
			},
			AutoApproveNonSensitive: false, // Require approval for all by default
			RequireApprovalForAll:   false, // Only sensitive tools require approval
			TelegramApprovalTimeout: 300,   // 5 minutes timeout
			EnableAuditLog:          true,  // Enable audit logging by default
			AuditLogPath:            ".agi/audit.log",
		},

		HeartbeatInterval: 0, // Disabled by default

		DebugTools: false, // Disabled by default

		SSH: SSHConfig{
			Enabled:    false,
			VisualMode: false, // Secure by default - no visual window
			DenyCommands: []string{
				"rm -rf /",
				"mkfs",
				"dd if=",
				"> /dev/sda",
				"shutdown",
				"reboot",
				"init 0",
				"halt",
			},
		},

		Browser: BrowserConfig{
			Headless:            true, // Secure by default - run in background
			Stealth:             true, // Enable stealth mode by default
			SlowMo:              0,
			RemoteDebuggingPort: 0, // Disabled by default for security
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
				return nil, path, fmt.Errorf("invalid JSON in config file %s: %w", path, err)
			}
			// Apply overrides from env (this now includes .env variables)
			applyEnvOverrides(cfg)
			// Validate the configuration
			if err := cfg.Validate(); err != nil {
				return nil, path, fmt.Errorf("configuration validation failed in %s: %w", path, err)
			}
			return cfg, path, nil
		}
	}

	// If no config found, use default and return the primary project-local path for saving
	defaultPath := ".agi/config.json"
	cfg := DefaultConfig()
	applyEnvOverrides(cfg)

	// Validate the default configuration
	if err := cfg.Validate(); err != nil {
		return nil, defaultPath, fmt.Errorf("default configuration validation failed: %w", err)
	}

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

// GetWorkplaceDir returns the workplace directory name, defaulting to "workplace".
func (c *Config) GetWorkplaceDir() string {
	if c.WorkplaceDir != "" {
		return c.WorkplaceDir
	}
	return "workplace"
}

// GetBrowserViewportW returns the browser viewport width, defaulting to 1920.
func (c *Config) GetBrowserViewportW() int {
	if c.BrowserViewportW > 0 {
		return c.BrowserViewportW
	}
	return 1920
}

// GetBrowserViewportH returns the browser viewport height, defaulting to 1080.
func (c *Config) GetBrowserViewportH() int {
	if c.BrowserViewportH > 0 {
		return c.BrowserViewportH
	}
	return 1080
}

// GetMaxRuleFileSize returns the max rule file size in KB, defaulting to 50.
func (c *Config) GetMaxRuleFileSize() int {
	if c.MaxRuleFileSize > 0 {
		return c.MaxRuleFileSize
	}
	return 50
}

// GetPipelineRoleDelay returns the pipeline role delay in ms, defaulting to 1500.
func (c *Config) GetPipelineRoleDelay() int {
	if c.PipelineRoleDelay > 0 {
		return c.PipelineRoleDelay
	}
	return 1500
}

// GetSessionMaxMessages returns the max messages per session, defaulting to 1000.
func (c *Config) GetSessionMaxMessages() int {
	if c.SessionMaxMessages > 0 {
		return c.SessionMaxMessages
	}
	return 1000
}

// GetHeartbeatIdleThreshold returns seconds of inactivity required before heartbeat acts, defaulting to 30.
func (c *Config) GetHeartbeatIdleThreshold() int {
	if c.HeartbeatIdleThreshold > 0 {
		return c.HeartbeatIdleThreshold
	}
	return 30
}

// FindSSHHost returns the host config matching label or host, or nil.
func (c *Config) FindSSHHost(labelOrHost string) *SSHHostConfig {
	for i := range c.SSH.Hosts {
		if c.SSH.Hosts[i].Label == labelOrHost || c.SSH.Hosts[i].Host == labelOrHost {
			return &c.SSH.Hosts[i]
		}
	}
	return nil
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

// Validate validates the configuration and returns any errors
func (c *Config) Validate() error {
	// Validate API key
	if err := validateAPIKey(c.APIKey, c.Provider); err != nil {
		return fmt.Errorf("invalid API key: %w", err)
	}

	// Validate API base URL
	if err := validateURL(c.APIBaseURL); err != nil {
		return fmt.Errorf("invalid API base URL: %w", err)
	}

	// Validate LLM parameters
	if err := validateLLMParams(c.Temperature, c.TopP, c.MaxTokens); err != nil {
		return fmt.Errorf("invalid LLM parameters: %w", err)
	}

	// Validate numeric ranges
	if c.MaxContextSize < 1 {
		return fmt.Errorf("max_context_size must be at least 1")
	}
	if c.MinConfidenceScore < 0 || c.MinConfidenceScore > 1 {
		return fmt.Errorf("min_confidence_score must be between 0.0 and 1.0")
	}
	if c.MaxFilesPerBatch < 1 {
		return fmt.Errorf("max_files_per_batch must be at least 1")
	}

	return nil
}

// validateAPIKey validates API key format based on provider
func validateAPIKey(key, provider string) error {
	if key == "" {
		return fmt.Errorf("API key is required")
	}

	// Auto-detect provider if not specified
	if provider == "" {
		if strings.HasPrefix(key, "sk-ant-") {
			provider = "anthropic"
		} else if strings.HasPrefix(key, "sk-") {
			provider = "openai"
		} else if strings.HasPrefix(key, "nvapi-") {
			provider = "nvidia"
		}
	}

	switch provider {
	case "openai":
		if !strings.HasPrefix(key, "sk-") || len(key) < 20 {
			return fmt.Errorf("OpenAI API key must start with 'sk-' and be at least 20 characters")
		}
	case "anthropic":
		if !strings.HasPrefix(key, "sk-ant-") || len(key) < 20 {
			return fmt.Errorf("Anthropic API key must start with 'sk-ant-' and be at least 20 characters")
		}
	case "nvidia":
		if !strings.HasPrefix(key, "nvapi-") || len(key) < 20 {
			return fmt.Errorf("NVIDIA API key must start with 'nvapi-' and be at least 20 characters")
		}
	case "":
		return fmt.Errorf("could not determine provider from API key format")
	default:
		// For unknown providers, just check basic requirements
		if len(key) < 10 {
			return fmt.Errorf("API key must be at least 10 characters")
		}
	}

	return nil
}

// validateURL validates that a URL is properly formatted
func validateURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("API base URL is required")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL must use http or https scheme")
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("URL must have a valid host")
	}

	// Recommend HTTPS for security
	if parsedURL.Scheme != "https" {
		fmt.Printf("Warning: Using HTTP instead of HTTPS is not recommended for API communications\n")
	}

	return nil
}

// validateLLMParams validates LLM parameter ranges
func validateLLMParams(temp, topP *float64, maxTokens *int) error {
	if temp != nil {
		if *temp < 0.0 || *temp > 2.0 {
			return fmt.Errorf("temperature must be between 0.0 and 2.0, got %f", *temp)
		}
	}

	if topP != nil {
		if *topP < 0.0 || *topP > 1.0 {
			return fmt.Errorf("top_p must be between 0.0 and 1.0, got %f", *topP)
		}
	}

	if maxTokens != nil {
		if *maxTokens < 1 {
			return fmt.Errorf("max_tokens must be at least 1, got %d", *maxTokens)
		}
		if *maxTokens > 1000000 {
			return fmt.Errorf("max_tokens seems too large (%d), please verify", *maxTokens)
		}
	}

	return nil
}
