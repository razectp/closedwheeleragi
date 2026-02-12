package tui

import (
	"context"
	"strconv"
	"strings"
	"time"

	"ClosedWheeler/pkg/browser"
	"ClosedWheeler/pkg/llm"
	"ClosedWheeler/pkg/telegram"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Step 0: Welcome ---

func (m SetupWizardModel) updateWelcome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		val := strings.TrimSpace(m.nameInput.Value())
		if val != "" {
			m.agentName = val
		}
		m.step = wizStepAPI
		m.apiSubStep = 0
		return m, nil
	case tea.KeyEsc:
		m.quitting = true
		return m, tea.Quit
	}
	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

// --- Step 1: API Configuration ---
// apiSubStep: 0=URL list, 1=custom URL input, 2=key input

func (m SetupWizardModel) updateAPI(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.apiSubStep {
	case 0:
		return m.updateAPIURLSelection(msg)
	case 1:
		return m.updateAPICustomURL(msg)
	case 2:
		return m.updateAPIKeyInput(msg)
	}
	return m, nil
}

// updateAPIURLSelection handles the preset URL list + "Custom URL" option.
func (m SetupWizardModel) updateAPIURLSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.apiURLCursor > 0 {
			m.apiURLCursor--
		}
	case "down", "j":
		if m.apiURLCursor < len(wizAPIURLs) {
			m.apiURLCursor++
		}
	case "enter":
		if m.apiURLCursor < len(wizAPIURLs) {
			// Preset URL selected
			m.selectedURL = wizAPIURLs[m.apiURLCursor].URL
			m.apiURL = m.selectedURL
			m.detectedProvider = detectProvider(m.apiURL)
			// Skip custom URL input, go straight to key
			m.apiSubStep = 2
			m.apiKeyInput.Focus()
			return m, textinput.Blink
		}
		// "Custom URL" selected — show the custom URL text input
		m.selectedURL = ""
		m.apiSubStep = 1
		m.customURLInput.SetValue("")
		m.customURLInput.Focus()
		return m, textinput.Blink
	case "esc":
		m.step = wizStepWelcome
		m.nameInput.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

// updateAPICustomURL handles the custom URL text input sub-step.
func (m SetupWizardModel) updateAPICustomURL(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		url := strings.TrimSpace(m.customURLInput.Value())
		if url == "" {
			return m, nil // don't advance on empty
		}
		m.selectedURL = url
		m.apiURL = url
		m.detectedProvider = detectProvider(url)
		m.customURLInput.Blur()
		// Advance to key input
		m.apiSubStep = 2
		m.apiKeyInput.Focus()
		return m, textinput.Blink
	case tea.KeyEsc:
		// Go back to URL list
		m.customURLInput.Blur()
		m.apiSubStep = 0
		return m, nil
	}

	var cmd tea.Cmd
	m.customURLInput, cmd = m.customURLInput.Update(msg)
	return m, cmd
}

// updateAPIKeyInput handles the API key text input sub-step.
func (m SetupWizardModel) updateAPIKeyInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		key := strings.TrimSpace(m.apiKeyInput.Value())
		m.apiKey = key
		if strings.HasPrefix(key, "sk-ant-") && m.detectedProvider == "" {
			m.detectedProvider = "anthropic"
		}
		m.apiKeyInput.Blur()
		// Advance to model selection, start fetching
		m.step = wizStepModel
		m.modelLoading = true
		return m, m.fetchModels()
	case tea.KeyEsc:
		m.apiKeyInput.Blur()
		// Go back: if custom URL was used, go to custom URL input; otherwise URL list
		if m.apiURLCursor >= len(wizAPIURLs) {
			m.apiSubStep = 1
			m.customURLInput.Focus()
			return m, textinput.Blink
		}
		m.apiSubStep = 0
		return m, nil
	}

	var cmd tea.Cmd
	m.apiKeyInput, cmd = m.apiKeyInput.Update(msg)
	return m, cmd
}

// detectProvider returns the detected provider based on URL.
func detectProvider(url string) string {
	switch {
	case strings.Contains(url, "anthropic.com"):
		return "anthropic"
	case strings.Contains(url, "openai.com"):
		return "openai"
	default:
		return ""
	}
}

func (m SetupWizardModel) fetchModels() tea.Cmd {
	return func() tea.Msg {
		models, err := llm.ListModelsWithProvider(m.apiURL, m.apiKey, m.detectedProvider)
		return modelsFetchedMsg{models: models, err: err}
	}
}

// --- Step 2: Model Selection ---

func (m SetupWizardModel) updateModel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.modelLoading {
		return m, nil
	}

	pageSize := 10
	totalModels := len(m.models)

	switch msg.String() {
	case "up", "k":
		if m.modelCursor > 0 {
			m.modelCursor--
			if m.modelCursor < m.modelPage*pageSize {
				m.modelPage--
			}
		}
	case "down", "j":
		if m.modelCursor < totalModels-1 {
			m.modelCursor++
			if m.modelCursor >= (m.modelPage+1)*pageSize {
				m.modelPage++
			}
		}
	case "n":
		maxPage := (totalModels - 1) / pageSize
		if m.modelPage < maxPage {
			m.modelPage++
			m.modelCursor = m.modelPage * pageSize
		}
	case "p":
		if m.modelPage > 0 {
			m.modelPage--
			m.modelCursor = m.modelPage * pageSize
		}
	case "enter":
		if totalModels > 0 && m.modelCursor < totalModels {
			m.primaryModel = m.models[m.modelCursor].ID
		} else if totalModels == 0 {
			m.primaryModel = "gpt-4o-mini"
		}
		m.step = wizStepSelfConfig
		m.selfConfigDone = false
		return m, tea.Batch(m.selfConfigSpinner.Tick, m.runSelfConfig())
	case "esc":
		m.step = wizStepAPI
		m.apiSubStep = 0
		return m, nil
	}
	return m, nil
}

func (m SetupWizardModel) runSelfConfig() tea.Cmd {
	return func() tea.Msg {
		client := llm.NewClientWithProvider(m.apiURL, m.apiKey, m.primaryModel, m.detectedProvider)
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		config, err := client.InterviewModel(ctx)
		return selfConfigDoneMsg{config: config, err: err}
	}
}

// --- Step 3: Self-Config ---

func (m SetupWizardModel) updateSelfConfig(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if !m.selfConfigDone {
		return m, nil
	}

	switch msg.String() {
	case "enter":
		if m.selfConfigErr != nil && m.primaryConfig == nil {
			temp, topP, maxTok := llm.ApplyProfileToConfig(m.primaryModel)
			pTemp, pTopP, pMaxTok := 0.7, 1.0, 4096
			if temp != nil {
				pTemp = *temp
			}
			if topP != nil {
				pTopP = *topP
			}
			if maxTok != nil {
				pMaxTok = *maxTok
			}
			m.primaryConfig = &llm.ModelSelfConfig{
				ModelName:         m.primaryModel,
				RecommendedTemp:   pTemp,
				RecommendedTopP:   pTopP,
				RecommendedMaxTok: pMaxTok,
				ContextWindow:     128000,
			}
		}
		m.step = wizStepPermissions
		return m, nil
	case "esc":
		m.step = wizStepModel
		m.selfConfigDone = false
		return m, nil
	}
	return m, nil
}

// --- Step 4: Permissions ---

func (m SetupWizardModel) updatePermissions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.permCursor > 0 {
			m.permCursor--
		}
	case "down", "j":
		if m.permCursor < len(wizPermOptions)-1 {
			m.permCursor++
		}
	case "enter":
		m.permPreset = wizPermOptions[m.permCursor].Value
		m.step = wizStepRules
		return m, nil
	case "esc":
		m.step = wizStepSelfConfig
		return m, nil
	}
	return m, nil
}

// --- Step 5: Rules ---

func (m SetupWizardModel) updateRules(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.rulesCursor > 0 {
			m.rulesCursor--
		}
	case "down", "j":
		if m.rulesCursor < len(wizRulesOptions)-1 {
			m.rulesCursor++
		}
	case "enter":
		m.rulesPreset = wizRulesOptions[m.rulesCursor].Value
		m.step = wizStepMemory
		return m, nil
	case "esc":
		m.step = wizStepPermissions
		return m, nil
	}
	return m, nil
}

// --- Step 6: Memory ---

func (m SetupWizardModel) updateMemory(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.memCursor > 0 {
			m.memCursor--
		}
	case "down", "j":
		if m.memCursor < len(wizMemOptions)-1 {
			m.memCursor++
		}
	case "enter":
		m.memPreset = wizMemOptions[m.memCursor].Value
		m.step = wizStepTelegram
		m.telegramSubStep = 0
		return m, nil
	case "esc":
		m.step = wizStepRules
		return m, nil
	}
	return m, nil
}

// --- Step 7: Telegram ---
// Sub-steps: 0=yes/no, 1=token input, 2=validating (spinner), 3=chatID+pairing

func (m SetupWizardModel) updateTelegram(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.telegramSubStep {
	case 0:
		// Yes/No toggle
		switch msg.String() {
		case "up", "k", "down", "j", "left", "right":
			m.telegramEnabled = !m.telegramEnabled
		case "enter":
			if !m.telegramEnabled {
				m.step = wizStepBrowser
				m.browserDepsOK = browser.CheckDeps()
				return m, nil
			}
			m.telegramSubStep = 1
			m.telegramInput.Focus()
			return m, textinput.Blink
		case "esc":
			m.step = wizStepMemory
			return m, nil
		}
		return m, nil

	case 1:
		// Token input
		switch msg.Type {
		case tea.KeyEnter:
			token := strings.TrimSpace(m.telegramInput.Value())
			if token == "" {
				m.telegramEnabled = false
				m.telegramInput.Blur()
				m.step = wizStepBrowser
				m.browserDepsOK = browser.CheckDeps()
				return m, nil
			}
			m.telegramToken = token
			m.telegramInput.Blur()
			// Start async validation
			m.telegramSubStep = 2
			m.telegramValidating = true
			m.telegramValidErr = nil
			m.telegramBotName = ""
			return m, tea.Batch(m.selfConfigSpinner.Tick, m.validateTelegramToken(token))
		case tea.KeyEsc:
			m.telegramSubStep = 0
			m.telegramInput.Blur()
			return m, nil
		}
		var cmd tea.Cmd
		m.telegramInput, cmd = m.telegramInput.Update(msg)
		return m, cmd

	case 2:
		// Validating spinner — block input until done
		return m, nil

	case 3:
		// Chat ID input with pairing instructions
		switch msg.Type {
		case tea.KeyEnter:
			chatStr := strings.TrimSpace(m.telegramChatInput.Value())
			if chatStr != "" {
				id, err := strconv.ParseInt(chatStr, 10, 64)
				if err != nil {
					// Invalid number — stay on step
					return m, nil
				}
				m.telegramChatID = id
			}
			m.telegramChatInput.Blur()
			m.step = wizStepBrowser
			m.browserDepsOK = browser.CheckDeps()
			return m, nil
		case tea.KeyEsc:
			m.telegramChatInput.Blur()
			m.telegramSubStep = 1
			m.telegramInput.Focus()
			return m, textinput.Blink
		}
		var cmd tea.Cmd
		m.telegramChatInput, cmd = m.telegramChatInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// validateTelegramToken returns a tea.Cmd that validates the bot token asynchronously.
func (m SetupWizardModel) validateTelegramToken(token string) tea.Cmd {
	return func() tea.Msg {
		botName, err := telegram.ValidateToken(token)
		return telegramValidatedMsg{botName: botName, err: err}
	}
}

// --- Step 8: Browser ---

func (m SetupWizardModel) updateBrowser(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.browserInstalling {
		return m, nil
	}

	if m.browserDepsOK || m.browserDone {
		switch msg.String() {
		case "enter":
			m.step = wizStepSummary
			return m, nil
		case "esc":
			m.step = wizStepTelegram
			m.telegramSubStep = 0
			return m, nil
		}
		return m, nil
	}

	switch msg.String() {
	case "up", "k", "down", "j", "left", "right":
		m.browserCursor = 1 - m.browserCursor
	case "enter":
		if m.browserCursor == 0 {
			m.browserInstalling = true
			return m, tea.Batch(m.browserSpinner.Tick, m.installBrowserDepsCmd())
		}
		m.step = wizStepSummary
		return m, nil
	case "esc":
		m.step = wizStepTelegram
		m.telegramSubStep = 0
		return m, nil
	}
	return m, nil
}

func (m SetupWizardModel) installBrowserDepsCmd() tea.Cmd {
	return func() tea.Msg {
		err := browser.InstallDeps()
		return browserInstallDoneMsg{err: err}
	}
}

// --- Step 9: Summary ---

func (m SetupWizardModel) updateSummary(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.saving {
		return m, nil
	}
	if m.saveDone {
		return m, tea.Quit
	}

	switch msg.String() {
	case "enter":
		m.saving = true
		return m, m.saveAll()
	case "esc":
		m.step = wizStepBrowser
		return m, nil
	}
	return m, nil
}

func (m SetupWizardModel) saveAll() tea.Cmd {
	return func() tea.Msg {
		err := saveConfiguration(
			m.agentName, m.apiURL, m.apiKey, m.primaryModel,
			m.detectedProvider, m.fallbackModels,
			m.permPreset, m.memPreset, m.telegramToken,
			m.telegramEnabled, m.telegramChatID, m.primaryConfig,
		)
		if err != nil {
			return saveDoneMsg{err: err}
		}
		err = saveRulesPreset(m.appRoot, m.rulesPreset)
		return saveDoneMsg{err: err}
	}
}
