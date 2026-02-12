package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Debate wizard steps
const (
	debateWizStepTopic   = 0 // Topic input
	debateWizStepModelA  = 1 // Model picker for Agent A
	debateWizStepRoleA   = 2 // Role picker for Agent A
	debateWizStepModelB  = 3 // Model picker for Agent B
	debateWizStepRoleB   = 4 // Role picker for Agent B
	debateWizStepTurns   = 5 // Turns number input
	debateWizStepRules   = 6 // Session rules (tool permissions)
	debateWizStepConfirm = 7 // Summary + launch
	debateWizTotalSteps  = 8
)

// DebateToolMode defines the tool permission level for a debate session.
const (
	DebateToolModeFull = "full" // All tools enabled
	DebateToolModeSafe = "safe" // Read-only tools only
	DebateToolModeNone = "none" // No tools — pure conversation
)

// debateToolModePresets returns the available tool permission modes.
func debateToolModePresets() []struct {
	Mode        string
	Icon        string
	Name        string
	Description string
} {
	return []struct {
		Mode        string
		Icon        string
		Name        string
		Description string
	}{
		{
			Mode:        DebateToolModeSafe,
			Icon:        "🛡️",
			Name:        "Safe Mode",
			Description: "Read-only tools only (read files, analyze, git status). Cannot modify anything.",
		},
		{
			Mode:        DebateToolModeFull,
			Icon:        "🔓",
			Name:        "Full Access",
			Description: "All tools enabled — agents can read, write, execute shell commands, and browse.",
		},
		{
			Mode:        DebateToolModeNone,
			Icon:        "💬",
			Name:        "Conversation Only",
			Description: "No tools at all. Pure discussion between the two agents.",
		},
	}
}

// initDebateWizard initializes the debate setup wizard overlay.
// If topic is non-empty it will be pre-filled.
func (m *EnhancedModel) initDebateWizard(topic string) {
	ti := textinput.New()
	ti.Placeholder = "e.g. artificial consciousness"
	ti.CharLimit = 256
	ti.Width = 50
	ti.SetValue(topic)
	ti.Focus()

	ca := textinput.New()
	ca.Placeholder = "Enter custom system prompt for Agent A..."
	ca.CharLimit = 1024
	ca.Width = 50

	cb := textinput.New()
	cb.Placeholder = "Enter custom system prompt for Agent B..."
	cb.CharLimit = 1024
	cb.Width = 50

	tn := textinput.New()
	tn.Placeholder = "20"
	tn.CharLimit = 4
	tn.Width = 10
	tn.SetValue("20")

	m.debateWizActive = true
	m.debateWizStep = debateWizStepTopic
	m.debateWizTopic = ti
	m.debateWizRoleA = 0
	m.debateWizRoleB = 0
	m.debateWizCustomA = ca
	m.debateWizCustomB = cb
	m.debateWizTurns = tn
	m.debateWizModelA = 0
	m.debateWizModelB = 0
	m.debateWizModels = m.buildAvailableModels()
	m.debateWizRulesCursor = 0 // Default: Safe Mode (index 0)
}

// buildAvailableModels builds the list of available model names for the wizard.
// The current config model is always first. Additional models come from enabled providers.
func (m *EnhancedModel) buildAvailableModels() []string {
	currentModel := m.agent.Config().Model
	models := []string{currentModel}
	seen := map[string]bool{currentModel: true}

	if m.providerManager != nil {
		for _, p := range m.providerManager.GetEnabledProviders() {
			if p.Model != "" && !seen[p.Model] {
				models = append(models, p.Model)
				seen[p.Model] = true
			}
		}
	}

	return models
}

// debateWizardUpdate handles keyboard input for the debate wizard.
func (m EnhancedModel) debateWizardUpdate(msg tea.KeyMsg) (EnhancedModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.debateWizActive = false
		return m, nil
	}

	switch m.debateWizStep {
	case debateWizStepTopic:
		return m.debateWizUpdateTopic(msg)
	case debateWizStepModelA:
		return m.debateWizUpdateModel(msg, true)
	case debateWizStepRoleA:
		return m.debateWizUpdateRole(msg, true)
	case debateWizStepModelB:
		return m.debateWizUpdateModel(msg, false)
	case debateWizStepRoleB:
		return m.debateWizUpdateRole(msg, false)
	case debateWizStepTurns:
		return m.debateWizUpdateTurns(msg)
	case debateWizStepRules:
		return m.debateWizUpdateRules(msg)
	case debateWizStepConfirm:
		return m.debateWizUpdateConfirm(msg)
	}
	return m, nil
}

// debateWizUpdateTopic handles input on the topic step.
func (m EnhancedModel) debateWizUpdateTopic(msg tea.KeyMsg) (EnhancedModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		topic := strings.TrimSpace(m.debateWizTopic.Value())
		if topic == "" {
			return m, nil
		}
		m.debateWizTopic.Blur()
		// If only 1 model available, skip model steps
		if len(m.debateWizModels) <= 1 {
			m.debateWizStep = debateWizStepRoleA
		} else {
			m.debateWizStep = debateWizStepModelA
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.debateWizTopic, cmd = m.debateWizTopic.Update(msg)
	return m, cmd
}

// debateWizUpdateModel handles input on a model selection step.
func (m EnhancedModel) debateWizUpdateModel(msg tea.KeyMsg, isAgentA bool) (EnhancedModel, tea.Cmd) {
	cursor := &m.debateWizModelB
	if isAgentA {
		cursor = &m.debateWizModelA
	}

	switch msg.String() {
	case "up", "k":
		if *cursor > 0 {
			*cursor--
		}
		return m, nil

	case "down", "j":
		if *cursor < len(m.debateWizModels)-1 {
			*cursor++
		}
		return m, nil

	case "enter":
		if isAgentA {
			m.debateWizStep = debateWizStepRoleA
		} else {
			m.debateWizStep = debateWizStepRoleB
		}
		return m, nil

	case "backspace":
		if isAgentA {
			m.debateWizStep = debateWizStepTopic
			m.debateWizTopic.Focus()
		} else {
			m.debateWizStep = debateWizStepRoleA
		}
		return m, nil
	}

	return m, nil
}

// debateWizUpdateRole handles input on a role selection step.
func (m EnhancedModel) debateWizUpdateRole(msg tea.KeyMsg, isAgentA bool) (EnhancedModel, tea.Cmd) {
	presets := DebateRolePresets()
	cursor := &m.debateWizRoleB
	customInput := &m.debateWizCustomB
	if isAgentA {
		cursor = &m.debateWizRoleA
		customInput = &m.debateWizCustomA
	}

	isCustom := *cursor == len(presets)-1

	switch msg.String() {
	case "up", "k":
		if *cursor > 0 {
			*cursor--
		}
		return m, nil

	case "down", "j":
		if *cursor < len(presets)-1 {
			*cursor++
		}
		return m, nil

	case "tab":
		// Toggle focus between list and custom input when Custom is selected
		if isCustom {
			if customInput.Focused() {
				customInput.Blur()
			} else {
				customInput.Focus()
			}
		}
		return m, nil

	case "enter":
		// If custom is selected and input is empty, don't advance
		if isCustom && strings.TrimSpace(customInput.Value()) == "" && !customInput.Focused() {
			customInput.Focus()
			return m, nil
		}
		if isAgentA {
			// Skip model B step if only 1 model
			if len(m.debateWizModels) <= 1 {
				m.debateWizStep = debateWizStepRoleB
			} else {
				m.debateWizStep = debateWizStepModelB
			}
		} else {
			m.debateWizStep = debateWizStepTurns
			m.debateWizTurns.Focus()
		}
		customInput.Blur()
		return m, nil

	case "backspace":
		if isCustom && customInput.Focused() {
			var cmd tea.Cmd
			*customInput, cmd = customInput.Update(msg)
			return m, cmd
		}
		// Go back a step
		if isAgentA {
			if len(m.debateWizModels) <= 1 {
				m.debateWizStep = debateWizStepTopic
				m.debateWizTopic.Focus()
			} else {
				m.debateWizStep = debateWizStepModelA
			}
		} else {
			if len(m.debateWizModels) <= 1 {
				m.debateWizStep = debateWizStepRoleA
			} else {
				m.debateWizStep = debateWizStepModelB
			}
		}
		return m, nil
	}

	// If custom input is focused, forward all keys to it
	if isCustom && customInput.Focused() {
		var cmd tea.Cmd
		*customInput, cmd = customInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// debateWizUpdateTurns handles input on the turns step.
func (m EnhancedModel) debateWizUpdateTurns(msg tea.KeyMsg) (EnhancedModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		m.debateWizStep = debateWizStepRules
		m.debateWizTurns.Blur()
		return m, nil
	}

	switch msg.String() {
	case "backspace":
		if m.debateWizTurns.Value() == "" {
			m.debateWizStep = debateWizStepRoleB
			m.debateWizTurns.Blur()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.debateWizTurns, cmd = m.debateWizTurns.Update(msg)
	return m, cmd
}

// debateWizUpdateRules handles input on the session rules (permissions) step.
func (m EnhancedModel) debateWizUpdateRules(msg tea.KeyMsg) (EnhancedModel, tea.Cmd) {
	modes := debateToolModePresets()

	switch msg.String() {
	case "up", "k":
		if m.debateWizRulesCursor > 0 {
			m.debateWizRulesCursor--
		}
		return m, nil

	case "down", "j":
		if m.debateWizRulesCursor < len(modes)-1 {
			m.debateWizRulesCursor++
		}
		return m, nil

	case "enter":
		m.debateWizStep = debateWizStepConfirm
		return m, nil

	case "backspace":
		m.debateWizStep = debateWizStepTurns
		m.debateWizTurns.Focus()
		return m, nil
	}

	return m, nil
}

// debateWizUpdateConfirm handles input on the confirmation step.
func (m EnhancedModel) debateWizUpdateConfirm(msg tea.KeyMsg) (EnhancedModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		return m.debateWizardLaunch()
	}

	switch msg.String() {
	case "backspace":
		m.debateWizStep = debateWizStepRules
		return m, nil
	}

	return m, nil
}

// debateWizResolveModel returns the model name for the given wizard cursor index.
func (m *EnhancedModel) debateWizResolveModel(idx int) string {
	if idx >= 0 && idx < len(m.debateWizModels) {
		return m.debateWizModels[idx]
	}
	return m.agent.Config().Model
}

// debateWizResolveToolMode returns the tool mode string for the current wizard selection.
func (m *EnhancedModel) debateWizResolveToolMode() string {
	modes := debateToolModePresets()
	if m.debateWizRulesCursor >= 0 && m.debateWizRulesCursor < len(modes) {
		return modes[m.debateWizRulesCursor].Mode
	}
	return DebateToolModeSafe
}

// debateWizardLaunch resolves wizard state and starts the debate.
func (m EnhancedModel) debateWizardLaunch() (EnhancedModel, tea.Cmd) {
	presets := DebateRolePresets()

	// Resolve prompts
	promptA := presets[m.debateWizRoleA].Prompt
	if m.debateWizRoleA == len(presets)-1 { // Custom
		promptA = strings.TrimSpace(m.debateWizCustomA.Value())
	}
	promptB := presets[m.debateWizRoleB].Prompt
	if m.debateWizRoleB == len(presets)-1 { // Custom
		promptB = strings.TrimSpace(m.debateWizCustomB.Value())
	}

	roleNameA := presets[m.debateWizRoleA].Name
	roleNameB := presets[m.debateWizRoleB].Name

	topic := strings.TrimSpace(m.debateWizTopic.Value())
	turns := 20
	if v, err := strconv.Atoi(strings.TrimSpace(m.debateWizTurns.Value())); err == nil && v > 0 {
		turns = v
	}

	// Resolve models
	modelA := m.debateWizResolveModel(m.debateWizModelA)
	modelB := m.debateWizResolveModel(m.debateWizModelB)

	// Resolve tool mode
	toolMode := m.debateWizResolveToolMode()

	// Auto-enable dual session
	if !m.dualSession.IsEnabled() {
		m.dualSession.Enable()
	}

	m.dualSession.SetMaxTurns(turns)
	m.dualSession.SetTopic(topic)
	m.dualSession.SetModels(modelA, modelB)
	m.dualSession.SetToolMode(toolMode)

	initialPrompt := fmt.Sprintf(
		"Let's have a thoughtful discussion about: %s\n\nShare your perspective and insights.", topic)

	if err := m.dualSession.StartConversationWithRoles(
		initialPrompt, roleNameA, promptA, roleNameB, promptB,
	); err != nil {
		m.debateWizActive = false
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   fmt.Sprintf("Failed to start debate: %v", err),
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return m, nil
	}

	// Close wizard and open the in-TUI debate viewer
	m.debateWizActive = false
	m.openDebateViewer()

	modelInfo := ""
	if modelA != modelB {
		modelInfo = fmt.Sprintf("\n   Models: %s vs %s", modelA, modelB)
	} else {
		modelInfo = fmt.Sprintf("\n   Model: %s", modelA)
	}

	// Tool mode label for the system message
	modes := debateToolModePresets()
	rulesLabel := "Safe Mode"
	for _, mode := range modes {
		if mode.Mode == toolMode {
			rulesLabel = fmt.Sprintf("%s %s", mode.Icon, mode.Name)
			break
		}
	}

	m.messageQueue.Add(QueuedMessage{
		Role: "system",
		Content: fmt.Sprintf("🤖 Debate started: %s\n"+
			"   🔵 %s vs 🟢 %s — %d turns%s\n"+
			"   Rules: %s\n"+
			"   Esc to close viewer. Use /stop to end early.",
			topic, roleNameA, roleNameB, turns, modelInfo, rulesLabel),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()

	return m, debateViewerTick()
}
