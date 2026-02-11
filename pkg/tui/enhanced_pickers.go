package tui

// Enhanced TUI bridge: model picker and OAuth login support for EnhancedModel.
// Reuses types, data, and view helpers from model_picker.go and login_picker.go.

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Model Picker for EnhancedModel ---

func (m *EnhancedModel) initPicker() {
	m.pickerActive = true
	m.pickerStep = pickerStepProvider
	m.pickerCursor = 0
	m.pickerNewKey = ""
	m.pickerNewURL = ""
	m.pickerModelID = ""

	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 50
	m.pickerInput = ti

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

// initPickerForOAuthProvider opens the model picker directly at the model-selection step

func (m EnhancedModel) enhancedPickerUpdate(msg tea.KeyMsg) (EnhancedModel, tea.Cmd) {
	if msg.Type == tea.KeyEsc {
		m.pickerActive = false
		return m, nil
	}

	switch m.pickerStep {
	case pickerStepProvider:
		return m.enhancedPickerUpdateProvider(msg)
	case pickerStepAPIKey:
		return m.enhancedPickerUpdateAPIKey(msg)
	case pickerStepModel:
		return m.enhancedPickerUpdateModel(msg)
	case pickerStepCustomModel:
		return m.enhancedPickerUpdateCustomModel(msg)
	case pickerStepEffort:
		return m.enhancedPickerUpdateEffort(msg)
	}
	return m, nil
}

func (m EnhancedModel) enhancedPickerUpdateProvider(msg tea.KeyMsg) (EnhancedModel, tea.Cmd) {
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

		if selected.BaseURL == "" {
			// Custom URL: ask for URL first (normal echo mode)
			m.pickerStep = pickerStepAPIKey
			m.pickerInput.Placeholder = "Enter custom base URL..."
			m.pickerInput.EchoMode = textinput.EchoNormal
			m.pickerInput.Focus()
			return m, textinput.Blink
		}

		if !selected.NeedsKey {
			m.pickerStep = pickerStepModel
			m.pickerCursor = 0
			return m, nil
		}

		// Ask for API key with password masking
		m.pickerStep = pickerStepAPIKey
		m.pickerInput.Placeholder = "Enter API key (or press Enter to keep current)..."
		m.pickerInput.EchoMode = textinput.EchoPassword
		m.pickerInput.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

func (m EnhancedModel) enhancedPickerUpdateAPIKey(msg tea.KeyMsg) (EnhancedModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		if m.pickerSelected.Label == "Custom URL" && m.pickerNewURL == "" {
			// First enter for Custom: this is the URL
			url := m.pickerInput.Value()
			if url == "" {
				return m, nil // Don't allow empty URL
			}
			m.pickerNewURL = url
			m.pickerSelected.BaseURL = url
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

func (m EnhancedModel) enhancedPickerUpdateModel(msg tea.KeyMsg) (EnhancedModel, tea.Cmd) {
	models := providerModels[m.pickerSelected.Label]

	switch msg.String() {
	case "up", "k":
		if m.pickerCursor > 0 {
			m.pickerCursor--
		}
	case "down", "j":
		if m.pickerCursor < len(models) {
			m.pickerCursor++
		}
	case "enter":
		if m.pickerCursor < len(models) {
			selectedModel := models[m.pickerCursor].ID
			if modelSupportsReasoning(selectedModel) {
				m.pickerModelID = selectedModel
				m.pickerStep = pickerStepEffort
				m.pickerCursor = 1
				return m, nil
			}
			return m.enhancedApplyPicker(selectedModel, "")
		}
		m.pickerStep = pickerStepCustomModel
		m.pickerInput.Placeholder = "Enter model name..."
		m.pickerInput.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

func (m EnhancedModel) enhancedPickerUpdateCustomModel(msg tea.KeyMsg) (EnhancedModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		modelID := strings.TrimSpace(m.pickerInput.Value())
		if modelID == "" {
			return m, nil
		}
		if modelSupportsReasoning(modelID) {
			m.pickerModelID = modelID
			m.pickerStep = pickerStepEffort
			m.pickerCursor = 1
			return m, nil
		}
		return m.enhancedApplyPicker(modelID, "")
	}

	var cmd tea.Cmd
	m.pickerInput, cmd = m.pickerInput.Update(msg)
	return m, cmd
}

func (m EnhancedModel) enhancedPickerUpdateEffort(msg tea.KeyMsg) (EnhancedModel, tea.Cmd) {
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
		return m.enhancedApplyPicker(m.pickerModelID, effort)
	}
	return m, nil
}

func (m EnhancedModel) enhancedApplyPicker(modelID, reasoningEffort string) (EnhancedModel, tea.Cmd) {
	selected := m.pickerSelected
	apiKey := m.pickerNewKey
	if apiKey == "" {
		apiKey = m.agent.Config().APIKey
	}
	baseURL := selected.BaseURL
	if m.pickerNewURL != "" {
		baseURL = m.pickerNewURL
	}

	m.pickerActive = false

	if err := m.agent.SwitchModel(selected.Provider, baseURL, apiKey, modelID, reasoningEffort); err != nil {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   fmt.Sprintf("Failed to switch model: %v", err),
			Timestamp: time.Now(),
			Complete:  true,
		})
	} else {
		msg := fmt.Sprintf("Switched to **%s** (provider: %s)", modelID, selected.Label)
		if reasoningEffort != "" {
			msg += fmt.Sprintf(" · reasoning: %s", reasoningEffort)
		}
		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   msg,
			Timestamp: time.Now(),
			Complete:  true,
		})
	}
	m.updateViewport()
	return m, nil
}

func (m EnhancedModel) enhancedPickerView() string {
	// Reuse styles from model_picker.go
	switch m.pickerStep {
	case pickerStepProvider:
		return m.enhancedPickerViewProvider()
	case pickerStepAPIKey:
		return m.enhancedPickerViewAPIKey()
	case pickerStepModel:
		return m.enhancedPickerViewModel()
	case pickerStepCustomModel:
		return m.enhancedPickerViewCustomModel()
	case pickerStepEffort:
		return m.enhancedPickerViewEffort()
	}
	return ""
}

func (m EnhancedModel) enhancedPickerViewProvider() string {
	var s strings.Builder
	s.WriteString(pickerTitleStyle.Render("Switch Provider & Model"))
	s.WriteString("\n\n")
	s.WriteString(pickerSubtitleStyle.Render("Select a provider:"))
	s.WriteString("\n\n")

	for i, p := range pickerProviders {
		cursor := "  "
		style := pickerUnselectedStyle
		if m.pickerCursor == i {
			cursor = "▸ "
			style = pickerSelectedStyle
		}
		label := p.Label
		s.WriteString(style.Render(cursor + label))

		s.WriteString("\n")
	}
	s.WriteString("\n")
	s.WriteString(pickerHintStyle.Render("  ↑/↓ Navigate · Enter Select · Esc Cancel"))

	boxWidth := m.width - 6
	if boxWidth < 20 {
		boxWidth = 20
	}
	return pickerBoxStyle.Width(boxWidth).Render(s.String())
}

func (m EnhancedModel) enhancedPickerViewAPIKey() string {
	var s strings.Builder
	s.WriteString(pickerTitleStyle.Render(fmt.Sprintf("API Key for %s", m.pickerSelected.Label)))
	s.WriteString("\n\n")
	s.WriteString(m.pickerInput.View())
	s.WriteString("\n\n")
	s.WriteString(pickerHintStyle.Render("  Enter to continue · Esc Cancel"))

	boxWidth := m.width - 6
	if boxWidth < 20 {
		boxWidth = 20
	}
	return pickerBoxStyle.Width(boxWidth).Render(s.String())
}

func (m EnhancedModel) enhancedPickerViewModel() string {
	var s strings.Builder
	models := providerModels[m.pickerSelected.Label]

	s.WriteString(pickerTitleStyle.Render(fmt.Sprintf("Select Model (%s)", m.pickerSelected.Label)))
	s.WriteString("\n\n")

	for i, model := range models {
		cursor := "  "
		style := pickerUnselectedStyle
		if m.pickerCursor == i {
			cursor = "▸ "
			style = pickerSelectedStyle
		}
		s.WriteString(style.Render(fmt.Sprintf("%s%s", cursor, model.ID)))
		if model.Hint != "" {
			s.WriteString(pickerHintStyle.Render("  " + model.Hint))
		}
		s.WriteString("\n")
	}

	cursor := "  "
	style := pickerUnselectedStyle
	if m.pickerCursor == len(models) {
		cursor = "▸ "
		style = pickerSelectedStyle
	}
	s.WriteString(style.Render(cursor + "Custom model..."))
	s.WriteString("\n\n")
	s.WriteString(pickerHintStyle.Render("  ↑/↓ Navigate · Enter Select · Esc Cancel"))

	boxWidth := m.width - 6
	if boxWidth < 20 {
		boxWidth = 20
	}
	return pickerBoxStyle.Width(boxWidth).Render(s.String())
}

func (m EnhancedModel) enhancedPickerViewCustomModel() string {
	var s strings.Builder
	s.WriteString(pickerTitleStyle.Render("Enter Model Name"))
	s.WriteString("\n\n")
	s.WriteString(m.pickerInput.View())
	s.WriteString("\n\n")
	s.WriteString(pickerHintStyle.Render("  Enter to confirm · Esc Cancel"))

	boxWidth := m.width - 6
	if boxWidth < 20 {
		boxWidth = 20
	}
	return pickerBoxStyle.Width(boxWidth).Render(s.String())
}

func (m EnhancedModel) enhancedPickerViewEffort() string {
	var s strings.Builder
	opts := getEffortOptions(m.pickerModelID)
	currentEffort := m.agent.Config().ReasoningEffort

	s.WriteString(pickerTitleStyle.Render(fmt.Sprintf("Reasoning Effort (%s)", m.pickerModelID)))
	s.WriteString("\n\n")

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
		s.WriteString(style.Render(fmt.Sprintf("%s%-8s", cursor, label)))
		s.WriteString(pickerHintStyle.Render("  " + opt.Hint))
		s.WriteString("\n")
	}
	s.WriteString("\n")
	s.WriteString(pickerHintStyle.Render("  ↑/↓ Navigate · Enter Select · Esc Cancel"))

	boxWidth := m.width - 6
	if boxWidth < 20 {
		boxWidth = 20
	}
	return pickerBoxStyle.Width(boxWidth).Render(s.String())
}

// --- Login Picker for EnhancedModel ---
