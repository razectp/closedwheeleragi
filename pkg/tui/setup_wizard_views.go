package tui

import (
	"fmt"
	"strings"
)

// --- Step views ---

func (m SetupWizardModel) viewWelcome() string {
	var s strings.Builder
	s.WriteString(WizardStepTitleStyle.Render("🚀 Welcome to ClosedWheelerAGI"))
	s.WriteString("\n\n")
	s.WriteString(HelpDetailValueStyle.Render("Let's get you up and running in under 2 minutes."))
	s.WriteString("\n\n")
	s.WriteString(HelpDetailLabelStyle.Render("Agent Name:"))
	s.WriteString("\n")
	s.WriteString(m.nameInput.View())
	s.WriteString("\n\n")
	s.WriteString(WizardFooterStyle.Render("Enter Confirm | Esc Quit"))
	return s.String()
}

func (m SetupWizardModel) viewAPI() string {
	var s strings.Builder
	s.WriteString(WizardStepTitleStyle.Render("🔑 API Configuration"))
	s.WriteString("\n\n")

	switch m.apiSubStep {
	case 0:
		// URL list selection
		s.WriteString(HelpDetailValueStyle.Render("Select your API provider:"))
		s.WriteString("\n\n")
		for i, url := range wizAPIURLs {
			cursor := "  "
			style := WizardUnselectedStyle
			if i == m.apiURLCursor {
				cursor = "▸ "
				style = WizardSelectedStyle
			}
			s.WriteString(style.Render(fmt.Sprintf("%s%s", cursor, url.Label)))
			s.WriteString("\n")
			s.WriteString(WizardDescStyle.Render("    " + url.URL))
			s.WriteString("\n")
		}
		// Custom option
		cursor := "  "
		style := WizardUnselectedStyle
		if m.apiURLCursor == len(wizAPIURLs) {
			cursor = "▸ "
			style = WizardSelectedStyle
		}
		s.WriteString(style.Render(cursor + "Custom URL"))
		s.WriteString("\n")
		s.WriteString(WizardDescStyle.Render("    Enter your own API endpoint"))
		s.WriteString("\n\n")
		s.WriteString(WizardFooterStyle.Render("↑/↓ Navigate | Enter Select | Esc Back"))

	case 1:
		// Custom URL text input
		s.WriteString(HelpDetailValueStyle.Render("Enter your custom API endpoint:"))
		s.WriteString("\n\n")
		s.WriteString(HelpDetailLabelStyle.Render("URL:"))
		s.WriteString("\n")
		s.WriteString(m.customURLInput.View())
		s.WriteString("\n\n")
		s.WriteString(WizardFooterStyle.Render("Enter Confirm | Esc Back"))

	case 2:
		// API key input
		provider := m.selectedURL
		if m.detectedProvider != "" {
			provider += fmt.Sprintf(" (detected: %s)", m.detectedProvider)
		}
		s.WriteString(HelpDetailValueStyle.Render(fmt.Sprintf("URL: %s", provider)))
		s.WriteString("\n\n")
		s.WriteString(HelpDetailLabelStyle.Render("API Key:"))
		s.WriteString("\n")
		s.WriteString(m.apiKeyInput.View())
		s.WriteString("\n\n")
		s.WriteString(WizardFooterStyle.Render("Enter Confirm | Esc Back"))
	}

	return s.String()
}

func (m SetupWizardModel) viewModel() string {
	var s strings.Builder
	s.WriteString(WizardStepTitleStyle.Render("🤖 Model Selection"))
	s.WriteString("\n\n")

	if m.modelLoading {
		s.WriteString(m.selfConfigSpinner.View())
		s.WriteString(" Fetching available models...")
		return s.String()
	}

	if m.modelErr != nil || len(m.models) == 0 {
		s.WriteString(SetupErrorStyle.Render("Could not fetch models."))
		s.WriteString("\n")
		s.WriteString(HelpDetailValueStyle.Render("Default model will be used: gpt-4o-mini"))
		s.WriteString("\n\n")
		s.WriteString(WizardFooterStyle.Render("Enter Continue | Esc Back"))
		return s.String()
	}

	pageSize := 10
	totalModels := len(m.models)
	totalPages := (totalModels + pageSize - 1) / pageSize
	start := m.modelPage * pageSize
	end := start + pageSize
	if end > totalModels {
		end = totalModels
	}

	s.WriteString(HelpDetailValueStyle.Render(fmt.Sprintf("Found %d models (page %d/%d):", totalModels, m.modelPage+1, totalPages)))
	s.WriteString("\n\n")

	for i := start; i < end; i++ {
		cursor := "  "
		style := WizardUnselectedStyle
		if i == m.modelCursor {
			cursor = "▸ "
			style = WizardSelectedStyle
		}
		s.WriteString(style.Render(fmt.Sprintf("%s%s", cursor, m.models[i].ID)))
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(WizardFooterStyle.Render("↑/↓ Navigate | n/p Page | Enter Select | Esc Back"))

	return s.String()
}

func (m SetupWizardModel) viewSelfConfig() string {
	var s strings.Builder
	s.WriteString(WizardStepTitleStyle.Render("🎤 Model Self-Configuration"))
	s.WriteString("\n\n")
	s.WriteString(HelpDetailValueStyle.Render(fmt.Sprintf("Asking '%s' to configure itself...", m.primaryModel)))
	s.WriteString("\n\n")

	if !m.selfConfigDone {
		s.WriteString(m.selfConfigSpinner.View())
		s.WriteString(" Interviewing model (up to 45s)...")
		return s.String()
	}

	if m.selfConfigErr != nil {
		s.WriteString(SetupErrorStyle.Render(fmt.Sprintf("Self-config failed: %v", m.selfConfigErr)))
		s.WriteString("\n")
		s.WriteString(HelpDetailValueStyle.Render("Fallback defaults will be used."))
	} else if m.selfConfigResult != nil {
		s.WriteString(SetupSuccessStyle.Render("Model configured itself successfully!"))
		s.WriteString("\n\n")
		s.WriteString(fmt.Sprintf("  Temperature: %.2f\n", m.selfConfigResult.RecommendedTemp))
		s.WriteString(fmt.Sprintf("  Top P:       %.2f\n", m.selfConfigResult.RecommendedTopP))
		s.WriteString(fmt.Sprintf("  Max Tokens:  %d\n", m.selfConfigResult.RecommendedMaxTok))
	}

	s.WriteString("\n")
	s.WriteString(WizardFooterStyle.Render("Enter Continue | Esc Back"))

	return s.String()
}

func (m SetupWizardModel) viewPermissions() string {
	return m.viewListStep("🛡️ Permissions Configuration",
		"Select a permissions preset:", wizPermOptions, m.permCursor)
}

func (m SetupWizardModel) viewRules() string {
	return m.viewListStep("📜 Rules & Personality",
		"Select a rules preset:", wizRulesOptions, m.rulesCursor)
}

func (m SetupWizardModel) viewMemory() string {
	return m.viewListStep("🧠 Memory Configuration",
		"Select memory storage preset:", wizMemOptions, m.memCursor)
}

// viewListStep renders a generic list selection step.
func (m SetupWizardModel) viewListStep(title, subtitle string, options []struct {
	Label string
	Desc  string
	Value string
}, cursor int) string {
	var s strings.Builder
	s.WriteString(WizardStepTitleStyle.Render(title))
	s.WriteString("\n\n")
	s.WriteString(HelpDetailValueStyle.Render(subtitle))
	s.WriteString("\n\n")

	for i, opt := range options {
		cur := "  "
		style := WizardUnselectedStyle
		if i == cursor {
			cur = "▸ "
			style = WizardSelectedStyle
		}
		s.WriteString(style.Render(fmt.Sprintf("%s%s", cur, opt.Label)))
		s.WriteString("\n")
		s.WriteString(WizardDescStyle.Render("    " + opt.Desc))
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(WizardFooterStyle.Render("↑/↓ Navigate | Enter Select | Esc Back"))

	return s.String()
}

func (m SetupWizardModel) viewTelegram() string {
	var s strings.Builder
	s.WriteString(WizardStepTitleStyle.Render("📱 Telegram Integration (Optional)"))
	s.WriteString("\n\n")
	s.WriteString(HelpDetailValueStyle.Render("Configure Telegram for remote agent control?"))
	s.WriteString("\n\n")

	switch m.telegramSubStep {
	case 0:
		// Yes/No toggle
		yesStyle := WizardUnselectedStyle
		noStyle := WizardUnselectedStyle
		yesCursor := "  "
		noCursor := "  "
		if m.telegramEnabled {
			yesCursor = "▸ "
			yesStyle = WizardSelectedStyle
		} else {
			noCursor = "▸ "
			noStyle = WizardSelectedStyle
		}
		s.WriteString(yesStyle.Render(yesCursor + "Yes"))
		s.WriteString("\n")
		s.WriteString(noStyle.Render(noCursor + "No"))
		s.WriteString("\n\n")
		s.WriteString(WizardFooterStyle.Render("↑/↓ Toggle | Enter Confirm | Esc Back"))

	case 1:
		// Token input
		s.WriteString(HelpDetailLabelStyle.Render("Bot Token:"))
		s.WriteString("\n")
		s.WriteString(m.telegramInput.View())
		s.WriteString("\n")
		if m.telegramValidErr != nil {
			s.WriteString("\n")
			s.WriteString(SetupErrorStyle.Render(fmt.Sprintf("Token invalid: %v", m.telegramValidErr)))
			s.WriteString("\n")
			s.WriteString(HelpDetailValueStyle.Render("Please enter a valid bot token."))
		}
		s.WriteString("\n")
		s.WriteString(WizardFooterStyle.Render("Enter Validate | Esc Back"))

	case 2:
		// Validating spinner
		s.WriteString(m.selfConfigSpinner.View())
		s.WriteString(" Validating bot token...")

	case 3:
		// Chat ID input + pairing instructions
		s.WriteString(SetupSuccessStyle.Render(fmt.Sprintf("Bot validated: @%s", m.telegramBotName)))
		s.WriteString("\n\n")
		s.WriteString(HelpDetailLabelStyle.Render("Chat ID (optional):"))
		s.WriteString("\n")
		s.WriteString(m.telegramChatInput.View())
		s.WriteString("\n\n")
		s.WriteString(WizardDescStyle.Render("To get your Chat ID:"))
		s.WriteString("\n")
		s.WriteString(WizardDescStyle.Render(fmt.Sprintf("  1. Open Telegram and send /start to @%s", m.telegramBotName)))
		s.WriteString("\n")
		s.WriteString(WizardDescStyle.Render("  2. The bot will auto-pair on first /start"))
		s.WriteString("\n")
		s.WriteString(WizardDescStyle.Render("  3. Or enter a known Chat ID above"))
		s.WriteString("\n\n")
		s.WriteString(WizardFooterStyle.Render("Enter Continue | Esc Back"))
	}

	return s.String()
}

func (m SetupWizardModel) viewBrowser() string {
	var s strings.Builder
	s.WriteString(WizardStepTitleStyle.Render("🌐 Browser Dependencies"))
	s.WriteString("\n\n")

	if m.browserDepsOK {
		s.WriteString(SetupSuccessStyle.Render("Playwright browsers already installed!"))
		s.WriteString("\n\n")
		s.WriteString(WizardFooterStyle.Render("Enter Continue | Esc Back"))
		return s.String()
	}

	if m.browserInstalling {
		s.WriteString(m.browserSpinner.View())
		s.WriteString(" Installing Playwright and Chromium...")
		s.WriteString("\n")
		s.WriteString(WizardDescStyle.Render("This may take a minute (~150 MB download)"))
		return s.String()
	}

	if m.browserDone {
		if m.browserErr != nil {
			s.WriteString(SetupErrorStyle.Render(fmt.Sprintf("Install failed: %v", m.browserErr)))
			s.WriteString("\n")
			s.WriteString(HelpDetailValueStyle.Render("Browser tools will auto-install on first use."))
		} else {
			s.WriteString(SetupSuccessStyle.Render("Browser dependencies installed successfully!"))
		}
		s.WriteString("\n\n")
		s.WriteString(WizardFooterStyle.Render("Enter Continue | Esc Back"))
		return s.String()
	}

	s.WriteString(HelpDetailValueStyle.Render("Install Playwright Chromium for browser automation?"))
	s.WriteString("\n")
	s.WriteString(WizardDescStyle.Render("Required for browser tools (~150 MB download)"))
	s.WriteString("\n\n")

	yesStyle := WizardUnselectedStyle
	noStyle := WizardUnselectedStyle
	yesCursor := "  "
	noCursor := "  "
	if m.browserCursor == 0 {
		yesCursor = "▸ "
		yesStyle = WizardSelectedStyle
	} else {
		noCursor = "▸ "
		noStyle = WizardSelectedStyle
	}
	s.WriteString(yesStyle.Render(yesCursor + "Yes, install now"))
	s.WriteString("\n")
	s.WriteString(noStyle.Render(noCursor + "Skip (auto-install later)"))
	s.WriteString("\n\n")
	s.WriteString(WizardFooterStyle.Render("↑/↓ Toggle | Enter Confirm | Esc Back"))

	return s.String()
}

func (m SetupWizardModel) viewSummary() string {
	var s strings.Builder
	s.WriteString(WizardStepTitleStyle.Render("✅ Review & Save"))
	s.WriteString("\n\n")

	if m.saving {
		s.WriteString(m.selfConfigSpinner.View())
		s.WriteString(" Saving configuration...")
		return s.String()
	}

	if m.saveDone {
		if m.saveErr != nil {
			s.WriteString(SetupErrorStyle.Render(fmt.Sprintf("Save failed: %v", m.saveErr)))
			s.WriteString("\n\n")
			s.WriteString(WizardFooterStyle.Render("Esc Back to retry"))
		} else {
			s.WriteString(SetupSuccessStyle.Render("Setup Complete!"))
			s.WriteString("\n\n")
			s.WriteString(HelpDetailValueStyle.Render("Configuration saved to .env and .agi/config.json"))
			s.WriteString("\n")
			s.WriteString(HelpDetailValueStyle.Render("Press any key to continue..."))
		}
		return s.String()
	}

	// Show summary
	telegramSummary := fmt.Sprintf("%v", m.telegramEnabled)
	if m.telegramEnabled && m.telegramBotName != "" {
		telegramSummary = fmt.Sprintf("@%s", m.telegramBotName)
		if m.telegramChatID != 0 {
			telegramSummary += fmt.Sprintf(" (Chat ID: %d)", m.telegramChatID)
		} else {
			telegramSummary += " (auto-pair on /start)"
		}
	}

	items := [][2]string{
		{"Agent", m.agentName},
		{"API URL", m.apiURL},
		{"Provider", m.detectedProvider},
		{"Model", m.primaryModel},
		{"Permissions", m.permPreset},
		{"Rules", m.rulesPreset},
		{"Memory", m.memPreset},
		{"Telegram", telegramSummary},
	}

	for _, item := range items {
		s.WriteString(HelpDetailLabelStyle.Render(fmt.Sprintf("  %-14s", item[0]+":")))
		s.WriteString(HelpDetailValueStyle.Render(item[1]))
		s.WriteString("\n")
	}

	if m.primaryConfig != nil {
		s.WriteString("\n")
		s.WriteString(HelpDetailLabelStyle.Render("  Model Config:"))
		s.WriteString("\n")
		s.WriteString(fmt.Sprintf("    Temperature: %.2f\n", m.primaryConfig.RecommendedTemp))
		s.WriteString(fmt.Sprintf("    Top P:       %.2f\n", m.primaryConfig.RecommendedTopP))
		s.WriteString(fmt.Sprintf("    Max Tokens:  %d\n", m.primaryConfig.RecommendedMaxTok))
	}

	s.WriteString("\n")
	s.WriteString(WizardFooterStyle.Render("Enter Save & Finish | Esc Back"))

	return s.String()
}
