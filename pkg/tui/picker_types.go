package tui

// picker_types.go — shared types, constants, and data for the model picker
// and OAuth login flow used by enhanced_pickers.go.

import (
	"strings"

	"ClosedWheeler/pkg/llm"

	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// Model picker
// ---------------------------------------------------------------------------

// Picker steps
const (
	pickerStepProvider = iota
	pickerStepAPIKey
	pickerStepModel
	pickerStepCustomModel
	pickerStepEffort // reasoning effort selection
)

// ProviderOption represents a selectable provider.
type ProviderOption struct {
	Label    string
	Provider string // "openai" or "anthropic"
	BaseURL  string
	NeedsKey bool
}

// ModelOption represents a selectable model.
type ModelOption struct {
	ID   string
	Hint string
}

// EffortOption represents a selectable reasoning effort level.
type EffortOption struct {
	ID   string // "low", "medium", "high", "xhigh"
	Hint string
}

// Known providers shown in the picker.
var pickerProviders = []ProviderOption{
	{Label: "Anthropic", Provider: "anthropic", BaseURL: "https://api.anthropic.com/v1", NeedsKey: true},
	{Label: "OpenAI", Provider: "openai", BaseURL: "https://api.openai.com/v1", NeedsKey: true},
	{Label: "DeepSeek", Provider: "openai", BaseURL: "https://api.deepseek.com", NeedsKey: true},
	{Label: "Moonshot", Provider: "openai", BaseURL: "https://api.moonshot.ai/v1", NeedsKey: true},
	{Label: "Google Gemini", Provider: "openai", BaseURL: "https://generativelanguage.googleapis.com/v1beta", NeedsKey: true},
	{Label: "Local (Ollama)", Provider: "openai", BaseURL: "http://localhost:11434/v1", NeedsKey: false},
	{Label: "Custom URL", Provider: "openai", BaseURL: "", NeedsKey: true},
}

// knownToModelOptions converts llm.KnownModel slice to ModelOption slice.
func knownToModelOptions(km []llm.KnownModel) []ModelOption {
	out := make([]ModelOption, len(km))
	for i, m := range km {
		out[i] = ModelOption{ID: m.ID, Hint: m.Hint}
	}
	return out
}

// providerModels is the single source of truth for model lists per provider.
// All lists come from pkg/llm/models.go.
var providerModels = map[string][]ModelOption{
	"Anthropic":      knownToModelOptions(llm.AnthropicKnownModels),
	"OpenAI":         knownToModelOptions(llm.OpenAIKnownModels),
	"DeepSeek":       knownToModelOptions(llm.DeepSeekKnownModels),
	"Moonshot":       knownToModelOptions(llm.MoonshotKnownModels),
	"Google Gemini":  knownToModelOptions(llm.GoogleKnownModels),
	"Local (Ollama)": knownToModelOptions(llm.OllamaKnownModels),
}

// Models that support xhigh effort level.
var xhighModels = map[string]bool{
	"gpt-5.3-codex": true,
	"gpt-5.2-codex": true,
	"gpt-5.1-codex": true,
	"gpt-5.2":       true,
}

// modelSupportsReasoning returns true if the model supports reasoning effort levels.
func modelSupportsReasoning(modelID string) bool {
	lower := strings.ToLower(modelID)
	if strings.HasPrefix(lower, "o1") ||
		strings.HasPrefix(lower, "o3") ||
		strings.HasPrefix(lower, "gpt-5") ||
		strings.Contains(lower, "codex") {
		return true
	}
	if strings.Contains(lower, "claude-opus-4") ||
		strings.Contains(lower, "claude-sonnet-4") {
		return true
	}
	return false
}

// getEffortOptions returns the available reasoning effort levels for a model.
func getEffortOptions(modelID string) []EffortOption {
	opts := []EffortOption{
		{ID: "low", Hint: "Faster, less thorough"},
		{ID: "medium", Hint: "Balanced (default)"},
		{ID: "high", Hint: "Slower, more thorough"},
	}
	if xhighModels[strings.ToLower(modelID)] {
		opts = append(opts, EffortOption{ID: "xhigh", Hint: "Maximum reasoning depth"})
	}
	return opts
}

// Picker styles.
var (
	pickerTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Bold(true).
				MarginBottom(1)

	pickerSubtitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#9CA3AF")).
				MarginBottom(1)

	pickerSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#10B981")).
				Bold(true)

	pickerUnselectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F9FAFB"))

	pickerHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Faint(true)

	pickerCurrentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F59E0B")).
				Bold(true)

	pickerBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			Padding(1, 2).
			Margin(1, 1)

	pickerFooterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280")).
				MarginTop(1)
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
