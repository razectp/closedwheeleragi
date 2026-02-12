package tui

import (
	"fmt"
	"strings"
)

// debateWizStepInfo holds step icons and titles for the debate wizard.
// When model steps are skipped (only 1 model), the progress bar still shows
// the correct current/total because we dynamically compute visible steps.
var debateWizStepInfo = [debateWizTotalSteps][2]string{
	{"💬", "Topic"},
	{"🧠", "Agent A Model"},
	{"🔵", "Agent A Role"},
	{"🧠", "Agent B Model"},
	{"🟢", "Agent B Role"},
	{"🔢", "Turns"},
	{"🔒", "Session Rules"},
	{"✅", "Confirm"},
}

// debateWizardView renders the debate wizard overlay.
func (m EnhancedModel) debateWizardView() string {
	boxWidth := m.width - 4
	if boxWidth < 50 {
		boxWidth = 50
	}

	var s strings.Builder

	s.WriteString(WizardTitleStyle.Render("Debate Setup"))
	s.WriteString("\n\n")
	s.WriteString(m.debateWizRenderProgress(boxWidth - 8))
	s.WriteString("\n\n")

	switch m.debateWizStep {
	case debateWizStepTopic:
		s.WriteString(m.debateWizViewTopic())
	case debateWizStepModelA:
		s.WriteString(m.debateWizViewModel(true))
	case debateWizStepRoleA:
		s.WriteString(m.debateWizViewRole(true))
	case debateWizStepModelB:
		s.WriteString(m.debateWizViewModel(false))
	case debateWizStepRoleB:
		s.WriteString(m.debateWizViewRole(false))
	case debateWizStepTurns:
		s.WriteString(m.debateWizViewTurns())
	case debateWizStepRules:
		s.WriteString(m.debateWizViewRules())
	case debateWizStepConfirm:
		s.WriteString(m.debateWizViewConfirm())
	}

	return WizardBoxStyle.Width(boxWidth).Render(s.String())
}

// debateWizVisibleSteps returns the number of visible wizard steps and the
// 1-based index of the current step among visible steps.
func (m EnhancedModel) debateWizVisibleSteps() (current, total int) {
	skipModels := len(m.debateWizModels) <= 1

	// Map each real step to a visible index
	visible := 0
	for step := 0; step < debateWizTotalSteps; step++ {
		if skipModels && (step == debateWizStepModelA || step == debateWizStepModelB) {
			continue
		}
		visible++
		if step == m.debateWizStep {
			current = visible
		}
	}
	total = visible
	return
}

// debateWizRenderProgress renders the step progress bar for the debate wizard.
func (m EnhancedModel) debateWizRenderProgress(width int) string {
	icon := debateWizStepInfo[m.debateWizStep][0]
	title := debateWizStepInfo[m.debateWizStep][1]

	currentVisible, totalVisible := m.debateWizVisibleSteps()

	label := fmt.Sprintf("Step %d/%d — %s %s", currentVisible, totalVisible, icon, title)

	barWidth := width - 4
	if barWidth < 10 {
		barWidth = 10
	}
	filled := barWidth * currentVisible / totalVisible
	empty := barWidth - filled

	bar := WizardProgressBarFillStyle.Render(strings.Repeat("█", filled)) +
		WizardProgressBarEmptyStyle.Render(strings.Repeat("░", empty))

	return label + "  " + bar
}

// debateWizViewTopic renders the topic input step.
func (m EnhancedModel) debateWizViewTopic() string {
	var s strings.Builder
	s.WriteString(WizardStepTitleStyle.Render("What topic should the agents discuss?"))
	s.WriteString("\n\n")
	s.WriteString(m.debateWizTopic.View())
	s.WriteString("\n\n")
	s.WriteString(WizardFooterStyle.Render("Enter → Next | Esc → Cancel"))
	return s.String()
}

// debateWizViewModel renders a model selection step.
func (m EnhancedModel) debateWizViewModel(isAgentA bool) string {
	cursor := m.debateWizModelB
	agentLabel := "Agent B"
	hint := "This agent responds to Agent A."
	if isAgentA {
		cursor = m.debateWizModelA
		agentLabel = "Agent A"
		hint = "This agent sends the first message each turn."
	}

	var s strings.Builder
	s.WriteString(WizardStepTitleStyle.Render(fmt.Sprintf("Choose a model for %s:", agentLabel)))
	s.WriteString("\n\n")

	for i, model := range m.debateWizModels {
		prefix := "  "
		style := WizardUnselectedStyle
		if i == cursor {
			prefix = "▸ "
			style = WizardSelectedStyle
		}

		label := model
		if i == 0 {
			label += " (current)"
		}
		// For Agent B, annotate if same as Agent A
		if !isAgentA && i == m.debateWizModelA {
			label += " (same as Agent A)"
		}

		s.WriteString(style.Render(fmt.Sprintf("%s%s", prefix, label)))
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(SetupInfoStyle.Render(hint))
	s.WriteString("\n\n")

	footer := "↑/↓ Navigate | Enter → Next | Backspace → Back | Esc → Cancel"
	if isAgentA {
		footer = "↑/↓ Navigate | Enter → Next | Esc → Cancel"
	}
	s.WriteString(WizardFooterStyle.Render(footer))
	return s.String()
}

// debateWizViewRole renders a role selection step.
func (m EnhancedModel) debateWizViewRole(isAgentA bool) string {
	presets := DebateRolePresets()
	cursor := m.debateWizRoleB
	agentLabel := "Agent B"
	customInput := m.debateWizCustomB
	if isAgentA {
		cursor = m.debateWizRoleA
		agentLabel = "Agent A"
		customInput = m.debateWizCustomA
	}

	var s strings.Builder
	s.WriteString(WizardStepTitleStyle.Render(fmt.Sprintf("Choose a role for %s:", agentLabel)))
	s.WriteString("\n\n")

	for i, role := range presets {
		prefix := "  "
		style := WizardUnselectedStyle
		if i == cursor {
			prefix = "▸ "
			style = WizardSelectedStyle
		}
		s.WriteString(style.Render(fmt.Sprintf("%s%s %s", prefix, role.Icon, role.Name)))
		s.WriteString("\n")
		if i == cursor {
			s.WriteString(WizardDescStyle.Render(role.Description))
			s.WriteString("\n")
		}
	}

	// Show custom input when Custom is selected
	if cursor == len(presets)-1 {
		s.WriteString("\n")
		s.WriteString(SetupPromptStyle.Render("Custom prompt:"))
		s.WriteString("\n")
		s.WriteString(customInput.View())
		s.WriteString("\n")
		if customInput.Focused() {
			s.WriteString(WizardFooterStyle.Render("Enter → Next | Tab → Back to list | Esc → Cancel"))
		} else {
			s.WriteString(WizardFooterStyle.Render("Tab → Edit prompt | Enter → Next | Backspace → Back | Esc → Cancel"))
		}
	} else {
		s.WriteString("\n")
		s.WriteString(WizardFooterStyle.Render("↑/↓ Navigate | Enter → Next | Backspace → Back | Esc → Cancel"))
	}

	return s.String()
}

// debateWizViewTurns renders the turns input step.
func (m EnhancedModel) debateWizViewTurns() string {
	var s strings.Builder
	s.WriteString(WizardStepTitleStyle.Render("How many turns?"))
	s.WriteString("\n\n")
	s.WriteString(SetupInfoStyle.Render("Each turn is one agent response. 20 turns = 10 exchanges."))
	s.WriteString("\n\n")
	s.WriteString(m.debateWizTurns.View())
	s.WriteString("\n\n")
	s.WriteString(WizardFooterStyle.Render("Enter → Next | Backspace (empty) → Back | Esc → Cancel"))
	return s.String()
}

// debateWizViewRules renders the session rules (permissions) step.
func (m EnhancedModel) debateWizViewRules() string {
	modes := debateToolModePresets()

	var s strings.Builder
	s.WriteString(WizardStepTitleStyle.Render("Session Rules — What can the agents do?"))
	s.WriteString("\n\n")
	s.WriteString(SetupInfoStyle.Render("Control which tools the debate agents can use during the conversation."))
	s.WriteString("\n\n")

	for i, mode := range modes {
		prefix := "  "
		style := WizardUnselectedStyle
		if i == m.debateWizRulesCursor {
			prefix = "▸ "
			style = WizardSelectedStyle
		}
		s.WriteString(style.Render(fmt.Sprintf("%s%s %s", prefix, mode.Icon, mode.Name)))
		s.WriteString("\n")
		if i == m.debateWizRulesCursor {
			s.WriteString(WizardDescStyle.Render(mode.Description))
			s.WriteString("\n")
		}
	}

	s.WriteString("\n")
	s.WriteString(WizardFooterStyle.Render("↑/↓ Navigate | Enter → Next | Backspace → Back | Esc → Cancel"))
	return s.String()
}

// debateWizViewConfirm renders the confirmation/summary step.
func (m EnhancedModel) debateWizViewConfirm() string {
	presets := DebateRolePresets()
	roleA := presets[m.debateWizRoleA]
	roleB := presets[m.debateWizRoleB]

	var s strings.Builder
	s.WriteString(WizardStepTitleStyle.Render("Ready to launch!"))
	s.WriteString("\n\n")

	s.WriteString(SetupPromptStyle.Render("Topic: "))
	s.WriteString(WizardSelectedStyle.Render(m.debateWizTopic.Value()))
	s.WriteString("\n\n")

	// Agent A summary
	modelA := m.debateWizResolveModel(m.debateWizModelA)
	s.WriteString(DebateAgentAStyle.Render(fmt.Sprintf("🔵 Agent A: %s %s", roleA.Icon, roleA.Name)))
	s.WriteString("\n")
	if m.debateWizRoleA == len(presets)-1 {
		prompt := m.debateWizCustomA.Value()
		if len(prompt) > 60 {
			prompt = prompt[:57] + "..."
		}
		s.WriteString(WizardDescStyle.Render(prompt))
	} else {
		s.WriteString(WizardDescStyle.Render(roleA.Description))
	}
	s.WriteString("\n")
	s.WriteString(SetupInfoStyle.Render(fmt.Sprintf("   Model: %s", modelA)))
	s.WriteString("\n\n")

	// Agent B summary
	modelB := m.debateWizResolveModel(m.debateWizModelB)
	s.WriteString(DebateAgentBStyle.Render(fmt.Sprintf("🟢 Agent B: %s %s", roleB.Icon, roleB.Name)))
	s.WriteString("\n")
	if m.debateWizRoleB == len(presets)-1 {
		prompt := m.debateWizCustomB.Value()
		if len(prompt) > 60 {
			prompt = prompt[:57] + "..."
		}
		s.WriteString(WizardDescStyle.Render(prompt))
	} else {
		s.WriteString(WizardDescStyle.Render(roleB.Description))
	}
	s.WriteString("\n")
	s.WriteString(SetupInfoStyle.Render(fmt.Sprintf("   Model: %s", modelB)))
	s.WriteString("\n\n")

	s.WriteString(SetupPromptStyle.Render("Turns: "))
	s.WriteString(WizardSelectedStyle.Render(m.debateWizTurns.Value()))
	s.WriteString("\n\n")

	// Session rules summary
	modes := debateToolModePresets()
	selectedMode := modes[m.debateWizRulesCursor]
	s.WriteString(SetupPromptStyle.Render("Rules: "))
	s.WriteString(WizardSelectedStyle.Render(fmt.Sprintf("%s %s", selectedMode.Icon, selectedMode.Name)))
	s.WriteString("\n")
	s.WriteString(WizardDescStyle.Render(selectedMode.Description))
	s.WriteString("\n\n")

	s.WriteString(SetupInfoStyle.Render("Debate will open in separate terminal windows. Main TUI stays interactive."))
	s.WriteString("\n\n")

	s.WriteString(WizardFooterStyle.Render("Enter → Launch | Backspace → Back | Esc → Cancel"))
	return s.String()
}
