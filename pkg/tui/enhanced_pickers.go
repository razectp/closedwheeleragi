package tui

// Enhanced TUI bridge: model picker and OAuth login support for EnhancedModel.
// Reuses types, data, and view helpers from model_picker.go and login_picker.go.

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"ClosedWheeler/pkg/llm"

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
// for the given OAuth provider (skipping provider/key steps since OAuth is already active).
func (m *EnhancedModel) initPickerForOAuthProvider(provider string) {
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
		m.pickerNewKey = ""
		m.pickerStep = pickerStepModel
		m.pickerCursor = 0
		return
	}
	// Fallback: open full picker
	m.initPicker()
}

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

		oauthSkip := false
		if selected.Label == "Anthropic" && m.agent.HasOAuthFor("anthropic") {
			oauthSkip = true
		} else if selected.Label == "OpenAI" && m.agent.HasOAuthFor("openai") {
			oauthSkip = true
		} else if selected.Label == "Google Gemini" && m.agent.HasOAuthFor("google") {
			oauthSkip = true
		}

		if !selected.NeedsKey || oauthSkip {
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
		if p.Label == "Anthropic" && m.agent.HasOAuthFor("anthropic") {
			label += " (OAuth " + m.agent.GetOAuthExpiryFor("anthropic") + ")"
		} else if p.Label == "OpenAI" && m.agent.HasOAuthFor("openai") {
			label += " (OAuth " + m.agent.GetOAuthExpiryFor("openai") + ")"
		} else if p.Label == "Google Gemini" && m.agent.HasOAuthFor("google") {
			label += " (OAuth " + m.agent.GetOAuthExpiryFor("google") + ")"
		}
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

func (m *EnhancedModel) closeEnhancedLogin() {
	m.loginActive = false
	m.loginStep = loginStepPickProvider
	m.loginCursor = 0
	m.loginProvider = ""
	m.loginVerifier = ""
	m.loginAuthURL = ""
	m.loginClipboard = false
	_ = removeLoginURL()
	if m.loginCancel != nil {
		m.loginCancel()
		m.loginCancel = nil
	}
}

func (m EnhancedModel) enhancedLoginUpdate(msg tea.KeyMsg) (EnhancedModel, tea.Cmd) {
	if msg.Type == tea.KeyEsc {
		m.closeEnhancedLogin()
		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   "OAuth login cancelled.",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return m, nil
	}

	switch m.loginStep {
	case loginStepPickProvider:
		return m.enhancedLoginUpdateProvider(msg)
	case loginStepAnthropicPaste:
		return m.enhancedLoginUpdatePaste(msg, "anthropic")
	case loginStepOpenAIWaiting:
		// While waiting for callback, 'P' switches to manual paste mode
		if msg.String() == "p" || msg.String() == "P" {
			m.loginStep = loginStepOpenAIPaste
			m.loginInput.Focus()
			return m, textinput.Blink
		}
		return m, nil
	case loginStepOpenAIPaste:
		return m.enhancedLoginUpdatePaste(msg, "openai")
	case loginStepGoogleWaiting:
		if msg.String() == "p" || msg.String() == "P" {
			m.loginStep = loginStepGooglePaste
			m.loginInput.Focus()
			return m, textinput.Blink
		}
		return m, nil
	case loginStepGooglePaste:
		return m.enhancedLoginUpdatePaste(msg, "google")
	}
	return m, nil
}

func (m EnhancedModel) enhancedLoginUpdateProvider(msg tea.KeyMsg) (EnhancedModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.loginCursor > 0 {
			m.loginCursor--
		}
	case "down", "j":
		if m.loginCursor < len(loginProviders)-1 {
			m.loginCursor++
		}
	case "enter":
		selected := loginProviders[m.loginCursor]
		m.loginProvider = selected.Provider

		switch selected.Provider {
		case "anthropic":
			return m.enhancedStartAnthropicLogin()
		case "openai":
			return m.enhancedStartCallbackLogin("openai")
		case "google":
			return m.enhancedStartCallbackLogin("google")
		default:
			m.closeEnhancedLogin()
			m.messageQueue.Add(QueuedMessage{
				Role:      "system",
				Content:   fmt.Sprintf("%s only supports API key authentication. Set it via /model.", selected.Label),
				Timestamp: time.Now(),
				Complete:  true,
			})
			m.updateViewport()
			return m, nil
		}
	}
	return m, nil
}

func (m EnhancedModel) enhancedStartAnthropicLogin() (EnhancedModel, tea.Cmd) {
	verifier, challenge, err := llm.GeneratePKCE()
	if err != nil {
		m.closeEnhancedLogin()
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   fmt.Sprintf("Failed to generate PKCE: %v", err),
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return m, nil
	}

	authURL := llm.BuildAuthURL(challenge, verifier)
	writeLoginURL(authURL)
	openBrowser(authURL)
	copied := copyToClipboard(authURL)

	m.loginStep = loginStepAnthropicPaste
	m.loginVerifier = verifier
	m.loginAuthURL = authURL
	m.loginClipboard = copied

	ti := textinput.New()
	ti.Placeholder = "Paste the code#state here..."
	ti.CharLimit = 512
	ti.Width = 60
	ti.Focus()
	m.loginInput = ti

	return m, textinput.Blink
}

func (m EnhancedModel) enhancedStartCallbackLogin(provider string) (EnhancedModel, tea.Cmd) {
	verifier, challenge, err := llm.GeneratePKCE()
	if err != nil {
		m.closeEnhancedLogin()
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   fmt.Sprintf("Failed to generate PKCE: %v", err),
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return m, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	m.loginCancel = cancel

	var authURL string
	var waitCmd tea.Cmd

	switch provider {
	case "openai":
		resultCh, err := llm.StartOpenAICallbackServer(ctx)
		if err != nil {
			cancel()
			m.closeEnhancedLogin()
			m.messageQueue.Add(QueuedMessage{
				Role:      "error",
				Content:   fmt.Sprintf("Failed to start callback server: %v", err),
				Timestamp: time.Now(),
				Complete:  true,
			})
			m.updateViewport()
			return m, nil
		}
		authURL = llm.BuildOpenAIAuthURL(challenge, verifier)
		m.loginStep = loginStepOpenAIWaiting
		ag := m.agent
		waitCmd = func() tea.Msg {
			select {
			case result := <-resultCh:
				if result.Err != nil {
					return oauthExchangeMsg{provider: "openai", err: result.Err}
				}
				err := ag.LoginOAuth("openai", result.Code, verifier)
				return oauthExchangeMsg{provider: "openai", err: err}
			case <-ctx.Done():
				return oauthExchangeMsg{provider: "openai", err: fmt.Errorf("cancelled")}
			}
		}
	case "google":
		resultCh, err := llm.StartGoogleCallbackServer(ctx)
		if err != nil {
			cancel()
			m.closeEnhancedLogin()
			m.messageQueue.Add(QueuedMessage{
				Role:      "error",
				Content:   fmt.Sprintf("Failed to start callback server: %v", err),
				Timestamp: time.Now(),
				Complete:  true,
			})
			m.updateViewport()
			return m, nil
		}
		authURL = llm.BuildGoogleAuthURL(challenge, verifier)
		m.loginStep = loginStepGoogleWaiting
		ag := m.agent
		waitCmd = func() tea.Msg {
			select {
			case result := <-resultCh:
				if result.Err != nil {
					return oauthExchangeMsg{provider: "google", err: result.Err}
				}
				err := ag.LoginOAuth("google", result.Code, verifier)
				return oauthExchangeMsg{provider: "google", err: err}
			case <-ctx.Done():
				return oauthExchangeMsg{provider: "google", err: fmt.Errorf("cancelled")}
			}
		}
	}

	m.loginVerifier = verifier
	m.loginAuthURL = authURL
	writeLoginURL(authURL)
	openBrowser(authURL)
	copied := copyToClipboard(authURL)
	m.loginClipboard = copied

	ti := textinput.New()
	ti.Placeholder = "Paste redirect URL here..."
	ti.CharLimit = 2048
	ti.Width = 60
	ti.Focus()
	m.loginInput = ti

	return m, waitCmd
}

func (m EnhancedModel) enhancedLoginUpdatePaste(msg tea.KeyMsg, provider string) (EnhancedModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		input := strings.TrimSpace(m.loginInput.Value())
		if input == "" {
			return m, nil
		}

		var code string
		if provider == "anthropic" {
			code = input // Anthropic uses code#state format directly
		} else {
			var err error
			code, err = extractCodeFromURL(input)
			if err != nil {
				m.messageQueue.Add(QueuedMessage{
					Role:      "error",
					Content:   fmt.Sprintf("Invalid URL: %v", err),
					Timestamp: time.Now(),
					Complete:  true,
				})
				m.updateViewport()
				return m, nil
			}
		}

		verifier := m.loginVerifier
		m.closeEnhancedLogin()
		err := m.agent.LoginOAuth(provider, code, verifier)
		if err != nil {
			m.messageQueue.Add(QueuedMessage{
				Role:      "error",
				Content:   fmt.Sprintf("OAuth login failed: %v", err),
				Timestamp: time.Now(),
				Complete:  true,
			})
		} else {
			labels := map[string]string{"anthropic": "Anthropic", "openai": "OpenAI", "google": "Google Gemini"}
			m.messageQueue.Add(QueuedMessage{
				Role:      "system",
				Content:   fmt.Sprintf("%s OAuth login successful! Token %s. Select a model below.", labels[provider], m.agent.GetOAuthExpiry()),
				Timestamp: time.Now(),
				Complete:  true,
			})
			m.initPickerForOAuthProvider(provider)
		}
		m.updateViewport()
		return m, nil
	}

	var cmd tea.Cmd
	m.loginInput, cmd = m.loginInput.Update(msg)
	return m, cmd
}

func (m EnhancedModel) enhancedLoginView() string {
	switch m.loginStep {
	case loginStepPickProvider:
		return enhancedLoginViewProvider(m)
	case loginStepAnthropicPaste:
		return enhancedLoginViewPaste(m, "Anthropic", "Paste the code#state below:")
	case loginStepOpenAIWaiting:
		return enhancedLoginViewWaiting(m, "OpenAI", 1455)
	case loginStepOpenAIPaste:
		return enhancedLoginViewPaste(m, "OpenAI", "Paste the redirect URL below:")
	case loginStepGoogleWaiting:
		return enhancedLoginViewWaiting(m, "Google Gemini", 8085)
	case loginStepGooglePaste:
		return enhancedLoginViewPaste(m, "Google Gemini", "Paste the redirect URL below:")
	}
	return ""
}

func enhancedLoginViewProvider(m EnhancedModel) string {
	var s strings.Builder
	s.WriteString(pickerTitleStyle.Render("OAuth Login"))
	s.WriteString("\n\n")
	s.WriteString(pickerSubtitleStyle.Render("Select provider:"))
	s.WriteString("\n\n")

	for i, p := range loginProviders {
		cursor := "  "
		style := pickerUnselectedStyle
		if m.loginCursor == i {
			cursor = "▸ "
			style = pickerSelectedStyle
		}
		label := p.Label
		// Show OAuth status for providers that support it
		expiry := m.agent.GetOAuthExpiryFor(p.Provider)
		if expiry != "" {
			label += " [" + expiry + "]"
		}
		hint := ""
		if p.Hint != "" {
			hint = pickerHintStyle.Render("  " + p.Hint)
		}
		s.WriteString(style.Render(cursor+label) + hint)
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

func enhancedLoginViewWaiting(m EnhancedModel, provider string, port int) string {
	var s strings.Builder
	s.WriteString(pickerTitleStyle.Render(fmt.Sprintf("%s OAuth Login", provider)))
	s.WriteString("\n\n")
	s.WriteString(pickerSubtitleStyle.Render("1. Open this URL in your browser:"))
	s.WriteString("\n\n")
	if m.loginClipboard {
		s.WriteString(pickerHintStyle.Render("   URL copied to clipboard!"))
	} else {
		s.WriteString(pickerHintStyle.Render("   Run in another terminal:  cat .agi/login-url.txt"))
	}
	s.WriteString("\n\n")
	s.WriteString(pickerHintStyle.Render("  After authorizing, the login will complete automatically."))
	s.WriteString("\n\n")
	s.WriteString(pickerHintStyle.Render(fmt.Sprintf("  VPS? SSH tunnel: ssh -L %d:localhost:%d user@server", port, port)))
	s.WriteString("\n\n")
	s.WriteString(pickerFooterStyle.Render("P = Paste URL manually  |  Esc Cancel"))

	boxWidth := m.width - 6
	if boxWidth < 20 {
		boxWidth = 20
	}
	return pickerBoxStyle.Width(boxWidth).Render(s.String())
}

func enhancedLoginViewPaste(m EnhancedModel, provider, prompt string) string {
	var s strings.Builder
	s.WriteString(pickerTitleStyle.Render(fmt.Sprintf("%s OAuth Login", provider)))
	s.WriteString("\n\n")
	s.WriteString(pickerSubtitleStyle.Render(prompt))
	s.WriteString("\n\n")
	s.WriteString(m.loginInput.View())
	s.WriteString("\n\n")
	s.WriteString(pickerHintStyle.Render("  Enter Submit  |  Esc Cancel"))

	boxWidth := m.width - 6
	if boxWidth < 20 {
		boxWidth = 20
	}
	return pickerBoxStyle.Width(boxWidth).Render(s.String())
}

// removeLoginURL removes the login URL file.
func removeLoginURL() error {
	return os.Remove(".agi/login-url.txt")
}
