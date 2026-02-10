package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Picker steps
const (
	pickerStepProvider = iota
	pickerStepAPIKey
	pickerStepModel
	pickerStepCustomModel
	pickerStepEffort // reasoning effort selection
)

// ProviderOption represents a selectable provider
type ProviderOption struct {
	Label    string
	Provider string // "openai" or "anthropic"
	BaseURL  string
	NeedsKey bool
}

// ModelOption represents a selectable model
type ModelOption struct {
	ID   string
	Hint string
}

// Known providers
var pickerProviders = []ProviderOption{
	{Label: "Anthropic", Provider: "anthropic", BaseURL: "https://api.anthropic.com/v1", NeedsKey: true},
	{Label: "OpenAI", Provider: "openai", BaseURL: "https://api.openai.com/v1", NeedsKey: true},
	{Label: "DeepSeek", Provider: "openai", BaseURL: "https://api.deepseek.com", NeedsKey: true},
	{Label: "Moonshot", Provider: "openai", BaseURL: "https://api.moonshot.ai/v1", NeedsKey: true},
	{Label: "Google Gemini", Provider: "openai", BaseURL: "https://generativelanguage.googleapis.com/v1beta", NeedsKey: true},
	{Label: "Local (Ollama)", Provider: "openai", BaseURL: "http://localhost:11434/v1", NeedsKey: false},
	{Label: "Custom URL", Provider: "openai", BaseURL: "", NeedsKey: true},
}

// Known models per provider label
var providerModels = map[string][]ModelOption{
	"Anthropic": {
		{ID: "claude-opus-4-6", Hint: "200K · Latest flagship · Opus 4.6"},
		{ID: "claude-sonnet-4-5-20250929", Hint: "200K · Fast + capable · Sonnet 4.5"},
		{ID: "claude-haiku-4-5-20251001", Hint: "200K · Fastest + cheapest · Haiku 4.5"},
		{ID: "claude-opus-4-20250514", Hint: "200K · Previous flagship · Opus 4"},
		{ID: "claude-sonnet-4-20250514", Hint: "200K · Balanced · Sonnet 4"},
		{ID: "claude-3-5-sonnet-20241022", Hint: "200K · Legacy · Sonnet 3.5"},
		{ID: "claude-3-5-haiku-20241022", Hint: "200K · Legacy fast · Haiku 3.5"},
		{ID: "claude-3-opus-20240229", Hint: "200K · Legacy flagship · Opus 3"},
	},
	"OpenAI": {
		{ID: "gpt-4.1", Hint: "1M · Latest flagship"},
		{ID: "gpt-4.1-mini", Hint: "1M · Fast + cheap"},
		{ID: "gpt-4.1-nano", Hint: "1M · Lightest"},
		{ID: "o3", Hint: "200K · Advanced reasoning"},
		{ID: "o4-mini", Hint: "200K · Fast reasoning"},
		{ID: "gpt-4o", Hint: "128K · Multimodal"},
		{ID: "gpt-4o-mini", Hint: "128K · Fast multimodal"},
	},
	"DeepSeek": {
		{ID: "deepseek-chat", Hint: "128K · DeepSeek-V3 · General purpose"},
		{ID: "deepseek-reasoner", Hint: "128K · DeepSeek-R1 · Reasoning"},
	},
	"Moonshot": {
		{ID: "kimi-k2-0711-preview", Hint: "128K · Kimi K2 · Agent-optimized"},
		{ID: "moonshot-v1-128k", Hint: "128K · General purpose"},
		{ID: "moonshot-v1-32k", Hint: "32K · Fast"},
	},
	"Google Gemini": {
		{ID: "gemini-2.5-pro", Hint: "1M · Latest pro · Best reasoning"},
		{ID: "gemini-2.5-flash", Hint: "1M · Latest fast · Thinking"},
		{ID: "gemini-2.5-flash-lite", Hint: "1M · Lightweight"},
		{ID: "gemini-2.0-flash", Hint: "1M · Stable fast"},
		{ID: "gemini-2.0-flash-thinking-exp", Hint: "1M · Experimental thinking"},
		{ID: "gemini-1.5-pro", Hint: "2M · Long context pro"},
		{ID: "gemini-1.5-flash", Hint: "1M · Long context fast"},
	},
	"Local (Ollama)": {
		{ID: "llama3.3", Hint: "128K · Meta Llama 3.3 · General"},
		{ID: "qwen2.5-coder:32b", Hint: "128K · Qwen code specialist"},
		{ID: "deepseek-coder-v2", Hint: "128K · Code specialist"},
		{ID: "codellama", Hint: "16K · Code specialist"},
		{ID: "mistral", Hint: "32K · Fast general"},
		{ID: "phi4", Hint: "16K · Microsoft small model"},
	},
}

// EffortOption represents a selectable reasoning effort level
type EffortOption struct {
	ID   string // "low", "medium", "high", "xhigh"
	Hint string
}

// Models that support xhigh effort level
var xhighModels = map[string]bool{
	"gpt-5.3-codex": true,
	"gpt-5.2-codex": true,
	"gpt-5.1-codex": true,
	"gpt-5.2":       true,
}

// modelSupportsReasoning returns true if the model supports reasoning effort levels.
func modelSupportsReasoning(modelID string) bool {
	lower := strings.ToLower(modelID)
	// OpenAI reasoning models
	if strings.HasPrefix(lower, "o1") ||
		strings.HasPrefix(lower, "o3") ||
		strings.HasPrefix(lower, "gpt-5") ||
		strings.Contains(lower, "codex") {
		return true
	}
	// Anthropic extended thinking models (Claude 4+ and Sonnet 4.5+)
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

// Picker styles
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

// initPicker initializes the model picker state
func (m *Model) initPicker() {
	ti := textinput.New()
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))
	ti.CharLimit = 256
	ti.Width = 60

	m.pickerActive = true
	m.pickerStep = pickerStepProvider
	m.pickerCursor = 0
	m.pickerInput = ti
	m.pickerNewKey = ""
	m.pickerNewURL = ""

	// Pre-select current provider
	currentProvider := m.agent.Config().Provider
	currentURL := m.agent.Config().APIBaseURL
	for i, p := range pickerProviders {
		if p.Provider == currentProvider && p.BaseURL == currentURL {
			m.pickerCursor = i
			break
		}
		if p.Provider == currentProvider {
			m.pickerCursor = i
		}
	}
}

// closePicker exits picker mode and resets all picker state
func (m *Model) closePicker() {
	m.pickerActive = false
	m.pickerStep = pickerStepProvider
	m.pickerCursor = 0
	m.pickerSelected = ProviderOption{}
	m.pickerNewKey = ""
	m.pickerNewURL = ""
	m.pickerModelID = ""
}

// pickerUpdate handles key events when picker is active
func (m Model) pickerUpdate(msg tea.KeyMsg) (Model, tea.Cmd) {
	// Global: Esc cancels picker
	if msg.Type == tea.KeyEsc {
		m.closePicker()
		m.messages = append(m.messages, Message{
			Role:      "system",
			Content:   "Model selection cancelled.",
			Timestamp: time.Now(),
		})
		m.updateViewport()
		return m, nil
	}

	switch m.pickerStep {
	case pickerStepProvider:
		return m.pickerUpdateProvider(msg)
	case pickerStepAPIKey:
		return m.pickerUpdateAPIKey(msg)
	case pickerStepModel:
		return m.pickerUpdateModel(msg)
	case pickerStepCustomModel:
		return m.pickerUpdateCustomModel(msg)
	case pickerStepEffort:
		return m.pickerUpdateEffort(msg)
	}

	return m, nil
}

func (m Model) pickerUpdateProvider(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.pickerCursor > 0 {
			m.pickerCursor--
		}
	case "down", "j":
		if m.pickerCursor < len(pickerProviders)-1 {
			m.pickerCursor++
		}
	case "enter":
		selected := pickerProviders[m.pickerCursor]
		m.pickerSelected = selected

		// Check if provider needs API key and is different from current
		currentProvider := m.agent.Config().Provider
		currentURL := m.agent.Config().APIBaseURL
		sameProvider := selected.Provider == currentProvider && selected.BaseURL == currentURL

		if selected.Label == "Custom URL" {
			// Custom: always ask for URL and key
			m.pickerStep = pickerStepAPIKey
			m.pickerInput.Placeholder = "Enter base URL (e.g., https://api.example.com/v1)"
			m.pickerInput.EchoMode = textinput.EchoNormal
			m.pickerInput.SetValue(currentURL)
			m.pickerInput.Focus()
			m.pickerNewURL = "" // Will be set when user confirms
			return m, textinput.Blink
		}

		m.pickerNewURL = selected.BaseURL

		// If OAuth is active for this specific provider+endpoint, skip API key step
		// Only skip for the actual OAuth provider (Anthropic API, OpenAI API), not for
		// other providers that share the "openai" protocol (DeepSeek, Gemini, Ollama).
		oauthSkip := false
		if selected.Label == "Anthropic" && m.agent.HasOAuthFor("anthropic") {
			oauthSkip = true
		} else if selected.Label == "OpenAI" && m.agent.HasOAuthFor("openai") {
			oauthSkip = true
		} else if selected.Label == "Google Gemini" && m.agent.HasOAuthFor("google") {
			oauthSkip = true
		}
		if oauthSkip {
			m.pickerNewKey = m.agent.Config().APIKey
			m.pickerStep = pickerStepModel
			m.pickerCursor = 0
			return m, nil
		}

		if !sameProvider && selected.NeedsKey {
			// Different provider: ask for API key
			m.pickerStep = pickerStepAPIKey
			m.pickerInput.Placeholder = "Paste your API key..."
			m.pickerInput.EchoMode = textinput.EchoPassword
			m.pickerInput.SetValue("")
			m.pickerInput.Focus()
			return m, textinput.Blink
		}

		// Same provider or no key needed: go to model selection
		m.pickerNewKey = m.agent.Config().APIKey
		m.pickerStep = pickerStepModel
		m.pickerCursor = 0
	}

	return m, nil
}

func (m Model) pickerUpdateAPIKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		if m.pickerSelected.Label == "Custom URL" && m.pickerNewURL == "" {
			// First enter for Custom: this is the URL
			url := m.pickerInput.Value()
			if url == "" {
				return m, nil // Don't allow empty URL
			}
			m.pickerNewURL = url
			// Now ask for API key
			m.pickerInput.Reset()
			m.pickerInput.Placeholder = "Paste your API key (or Enter to skip)..."
			m.pickerInput.EchoMode = textinput.EchoPassword
			m.pickerInput.SetValue("")
			m.pickerInput.Focus()
			return m, textinput.Blink
		}

		// API key entry
		key := m.pickerInput.Value()
		if key == "" {
			// Keep current key
			m.pickerNewKey = m.agent.Config().APIKey
		} else {
			m.pickerNewKey = key
		}

		// Auto-detect provider from key
		if strings.HasPrefix(m.pickerNewKey, "sk-ant-") && m.pickerSelected.Label == "Custom URL" {
			m.pickerSelected.Provider = "anthropic"
		}

		// Skip model list if no known models for this provider (e.g. Custom URL)
		models := providerModels[m.pickerSelected.Label]
		if len(models) == 0 {
			m.pickerStep = pickerStepCustomModel
			m.pickerInput.Reset()
			m.pickerInput.Placeholder = "Enter model ID..."
			m.pickerInput.EchoMode = textinput.EchoNormal
			m.pickerInput.SetValue("")
			m.pickerInput.Focus()
			return m, textinput.Blink
		}
		m.pickerStep = pickerStepModel
		m.pickerCursor = 0
		return m, nil
	}

	var cmd tea.Cmd
	m.pickerInput, cmd = m.pickerInput.Update(msg)
	return m, cmd
}

func (m Model) pickerUpdateModel(msg tea.KeyMsg) (Model, tea.Cmd) {
	models := m.getPickerModels()
	totalItems := len(models) + 1 // +1 for "Custom model ID" option

	switch msg.String() {
	case "up", "k":
		if m.pickerCursor > 0 {
			m.pickerCursor--
		}
	case "down", "j":
		if m.pickerCursor < totalItems-1 {
			m.pickerCursor++
		}
	case "enter":
		if m.pickerCursor == len(models) {
			// "Custom model ID" selected
			m.pickerStep = pickerStepCustomModel
			m.pickerInput.Reset()
			m.pickerInput.Placeholder = "Enter model ID..."
			m.pickerInput.EchoMode = textinput.EchoNormal
			m.pickerInput.SetValue("")
			m.pickerInput.Focus()
			return m, textinput.Blink
		}

		// Check if model supports reasoning effort
		selectedModel := models[m.pickerCursor].ID
		if modelSupportsReasoning(selectedModel) {
			m.pickerModelID = selectedModel
			m.pickerStep = pickerStepEffort
			m.pickerCursor = 1 // default to "medium"
			return m, nil
		}
		return m.applyPickerSelection(selectedModel, "")
	}

	return m, nil
}

func (m Model) pickerUpdateCustomModel(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		modelID := strings.TrimSpace(m.pickerInput.Value())
		if modelID == "" {
			return m, nil
		}
		if modelSupportsReasoning(modelID) {
			m.pickerModelID = modelID
			m.pickerStep = pickerStepEffort
			m.pickerCursor = 1 // default to "medium"
			return m, nil
		}
		return m.applyPickerSelection(modelID, "")
	}

	var cmd tea.Cmd
	m.pickerInput, cmd = m.pickerInput.Update(msg)
	return m, cmd
}

// pickerUpdateEffort handles key events during effort level selection
func (m Model) pickerUpdateEffort(msg tea.KeyMsg) (Model, tea.Cmd) {
	opts := getEffortOptions(m.pickerModelID)

	switch msg.String() {
	case "up", "k":
		if m.pickerCursor > 0 {
			m.pickerCursor--
		}
	case "down", "j":
		if m.pickerCursor < len(opts)-1 {
			m.pickerCursor++
		}
	case "enter":
		effort := opts[m.pickerCursor].ID
		return m.applyPickerSelection(m.pickerModelID, effort)
	}

	return m, nil
}

// applyPickerSelection applies the final model/provider selection
func (m Model) applyPickerSelection(modelID, reasoningEffort string) (Model, tea.Cmd) {
	provider := m.pickerSelected.Provider
	baseURL := m.pickerNewURL
	apiKey := m.pickerNewKey

	// Apply via agent
	if err := m.agent.SwitchModel(provider, baseURL, apiKey, modelID, reasoningEffort); err != nil {
		m.closePicker()
		m.messages = append(m.messages, Message{
			Role:      "error",
			Content:   fmt.Sprintf("Failed to switch model: %v", err),
			Timestamp: time.Now(),
		})
		m.updateViewport()
		return m, nil
	}

	info := fmt.Sprintf("Model switched!\n  Provider: %s\n  Model:    %s\n  Base URL: %s", provider, modelID, baseURL)
	if reasoningEffort != "" {
		info += fmt.Sprintf("\n  Effort:   %s", reasoningEffort)
	}

	m.closePicker()
	m.messages = append(m.messages, Message{
		Role:      "system",
		Content:   info,
		Timestamp: time.Now(),
	})
	m.updateViewport()
	return m, nil
}

// getPickerModels returns the model list for the selected provider
func (m *Model) getPickerModels() []ModelOption {
	if models, ok := providerModels[m.pickerSelected.Label]; ok {
		return models
	}
	// For custom/unknown providers, return empty (user types model ID)
	return nil
}

// initPickerForOAuthProvider opens the model picker directly at the model-selection step
// for the given OAuth provider (skipping provider/key steps since OAuth is already active).
func (m *Model) initPickerForOAuthProvider(provider string) {
	// Find the matching pickerProvider entry
	for _, p := range pickerProviders {
		oauthProvider := ""
		switch p.Label {
		case "Anthropic":
			oauthProvider = "anthropic"
		case "OpenAI":
			oauthProvider = "openai"
		case "Google Gemini":
			oauthProvider = "google"
		}
		if oauthProvider != provider {
			continue
		}
		m.pickerActive = true
		m.pickerSelected = p
		m.pickerNewURL = p.BaseURL
		m.pickerNewKey = "" // OAuth — no key needed
		m.pickerStep = pickerStepModel
		m.pickerCursor = 0
		return
	}
	// Fallback: open full picker
	m.initPicker()
}

// pickerView renders the model picker overlay
func (m Model) pickerView() string {
	var s strings.Builder

	// Header
	s.WriteString(pickerTitleStyle.Render("🔧 Model Configuration"))
	s.WriteString("\n")

	// Current config
	cfg := m.agent.Config()
	currentInfo := fmt.Sprintf("Current: %s @ %s",
		pickerCurrentStyle.Render(cfg.Model),
		pickerHintStyle.Render(cfg.APIBaseURL))
	s.WriteString(currentInfo)
	s.WriteString("\n\n")

	switch m.pickerStep {
	case pickerStepProvider:
		s.WriteString(m.pickerViewProvider())
	case pickerStepAPIKey:
		s.WriteString(m.pickerViewAPIKey())
	case pickerStepModel:
		s.WriteString(m.pickerViewModel())
	case pickerStepCustomModel:
		s.WriteString(m.pickerViewCustomModel())
	case pickerStepEffort:
		s.WriteString(m.pickerViewEffort())
	}

	// Footer
	s.WriteString("\n")
	s.WriteString(pickerFooterStyle.Render("↑↓ Navigate · Enter Select · Esc Cancel"))

	boxWidth := m.width - 6
	if boxWidth < 20 {
		boxWidth = 20
	}
	return pickerBoxStyle.Width(boxWidth).Render(s.String())
}

func (m Model) pickerViewProvider() string {
	var s strings.Builder
	s.WriteString(pickerSubtitleStyle.Render("Select Provider:"))
	s.WriteString("\n\n")

	currentProvider := m.agent.Config().Provider
	currentURL := m.agent.Config().APIBaseURL

	for i, p := range pickerProviders {
		cursor := "  "
		style := pickerUnselectedStyle
		if m.pickerCursor == i {
			cursor = "▸ "
			style = pickerSelectedStyle
		}

		label := p.Label
		// Mark current provider
		if p.Provider == currentProvider && (p.BaseURL == currentURL || p.Label == "Custom URL") {
			label += " ✓"
		}

		hint := ""
		if models, ok := providerModels[p.Label]; ok {
			hint = fmt.Sprintf(" (%d models)", len(models))
		}

		// Show OAuth status for providers that support OAuth
		oauthProvider := ""
		switch p.Label {
		case "Anthropic":
			oauthProvider = "anthropic"
		case "OpenAI":
			oauthProvider = "openai"
		case "Google Gemini":
			oauthProvider = "google"
		}
		if oauthProvider != "" {
			if oauthExpiry := m.agent.GetOAuthExpiryFor(oauthProvider); oauthExpiry != "" {
				hint += " [OAuth: " + oauthExpiry + "]"
			}
		}

		line := style.Render(cursor+label) + pickerHintStyle.Render(hint)
		s.WriteString(line)
		s.WriteString("\n")
	}

	return s.String()
}

func (m Model) pickerViewAPIKey() string {
	var s strings.Builder

	if m.pickerSelected.Label == "Custom URL" && m.pickerNewURL == "" {
		s.WriteString(pickerSubtitleStyle.Render("Enter Base URL:"))
		s.WriteString("\n\n")
		s.WriteString(m.pickerInput.View())
		s.WriteString("\n\n")
		s.WriteString(pickerHintStyle.Render("The API endpoint URL (e.g., https://api.openai.com/v1)"))
	} else {
		providerLabel := m.pickerSelected.Label
		s.WriteString(pickerSubtitleStyle.Render(fmt.Sprintf("Enter API Key for %s:", providerLabel)))
		s.WriteString("\n\n")
		s.WriteString(m.pickerInput.View())
		s.WriteString("\n\n")

		hints := []string{"Paste your API key and press Enter"}
		if m.pickerSelected.Provider == "anthropic" {
			hints = append(hints,
				"Supports: API keys (sk-ant-api03-...) and setup tokens (sk-ant-oat01-...)",
				"Auth type is detected automatically from the key prefix")
		}
		hints = append(hints, "Press Enter with empty field to keep current key")

		for _, h := range hints {
			s.WriteString(pickerHintStyle.Render("  " + h))
			s.WriteString("\n")
		}
	}

	return s.String()
}

func (m Model) pickerViewModel() string {
	var s strings.Builder
	s.WriteString(pickerSubtitleStyle.Render(fmt.Sprintf("Select Model (%s):", m.pickerSelected.Label)))
	s.WriteString("\n\n")

	models := m.getPickerModels()
	currentModel := m.agent.Config().Model

	for i, mo := range models {
		cursor := "  "
		style := pickerUnselectedStyle
		if m.pickerCursor == i {
			cursor = "▸ "
			style = pickerSelectedStyle
		}

		label := mo.ID
		if label == currentModel {
			label += " ✓"
		}

		line := style.Render(cursor+label) + "  " + pickerHintStyle.Render(mo.Hint)
		s.WriteString(line)
		s.WriteString("\n")
	}

	// Custom model option
	customCursor := "  "
	customStyle := pickerUnselectedStyle
	if m.pickerCursor == len(models) {
		customCursor = "▸ "
		customStyle = pickerSelectedStyle
	}
	s.WriteString(customStyle.Render(customCursor + "Enter custom model ID..."))
	s.WriteString("\n")

	return s.String()
}

func (m Model) pickerViewCustomModel() string {
	var s strings.Builder
	s.WriteString(pickerSubtitleStyle.Render("Enter Model ID:"))
	s.WriteString("\n\n")
	s.WriteString(m.pickerInput.View())
	s.WriteString("\n\n")
	s.WriteString(pickerHintStyle.Render("  Type the exact model identifier and press Enter"))
	return s.String()
}

func (m Model) pickerViewEffort() string {
	var s strings.Builder
	s.WriteString(pickerSubtitleStyle.Render(fmt.Sprintf("Reasoning Effort for %s:", m.pickerModelID)))
	s.WriteString("\n\n")

	opts := getEffortOptions(m.pickerModelID)
	currentEffort := m.agent.Config().ReasoningEffort

	for i, opt := range opts {
		cursor := "  "
		style := pickerUnselectedStyle
		if m.pickerCursor == i {
			cursor = "▸ "
			style = pickerSelectedStyle
		}

		label := opt.ID
		if label == currentEffort {
			label += " ✓"
		}

		line := style.Render(cursor+label) + "  " + pickerHintStyle.Render(opt.Hint)
		s.WriteString(line)
		s.WriteString("\n")
	}

	return s.String()
}
