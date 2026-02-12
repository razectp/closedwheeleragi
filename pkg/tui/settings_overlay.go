package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// SettingsItem represents a single toggleable or informational setting.
type SettingsItem struct {
	Label       string
	Description string
	Key         string
	GetValue    func(*EnhancedModel) string
	Toggle      func(*EnhancedModel)
	IsBool      bool
}

// openSettings activates the settings overlay and optionally focuses the item matching focusKey.
func (m *EnhancedModel) openSettings(focusKey string) {
	m.settingsItems = m.buildSettingsItems()
	m.settingsCursor = 0

	for i, item := range m.settingsItems {
		if item.Key == focusKey {
			m.settingsCursor = i
			break
		}
	}

	m.settingsActive = true
}

// buildSettingsItems returns the full list of toggleable settings.
func (m *EnhancedModel) buildSettingsItems() []SettingsItem {
	return []SettingsItem{
		{
			Label:       "Verbose Mode",
			Description: "Show reasoning/thinking content in responses",
			Key:         "verbose",
			IsBool:      true,
			GetValue: func(em *EnhancedModel) string {
				if em.verbose {
					return "ON"
				}
				return "OFF"
			},
			Toggle: func(em *EnhancedModel) {
				em.verbose = !em.verbose
				em.agent.Config().UI.Verbose = em.verbose
				_ = em.agent.SaveConfig()
			},
		},
		{
			Label:       "Debug Mode",
			Description: "Enable debug output for tool execution",
			Key:         "debug",
			IsBool:      true,
			GetValue: func(em *EnhancedModel) string {
				if em.agent.Config().DebugTools {
					return "ON"
				}
				return "OFF"
			},
			Toggle: func(em *EnhancedModel) {
				em.agent.Config().DebugTools = !em.agent.Config().DebugTools
				_ = em.agent.SaveConfig()
			},
		},
		{
			Label:       "Timestamps",
			Description: "Show timestamps on messages",
			Key:         "timestamps",
			IsBool:      true,
			GetValue: func(em *EnhancedModel) string {
				if em.showTimestamps {
					return "ON"
				}
				return "OFF"
			},
			Toggle: func(em *EnhancedModel) {
				em.showTimestamps = !em.showTimestamps
			},
		},
		{
			Label:       "Multi-Agent Pipeline",
			Description: "Planner -> Researcher -> Executor -> Critic",
			Key:         "pipeline",
			IsBool:      true,
			GetValue: func(em *EnhancedModel) string {
				if em.agent.PipelineEnabled() {
					return "ON"
				}
				return "OFF"
			},
			Toggle: func(em *EnhancedModel) {
				em.agent.EnablePipeline(!em.agent.PipelineEnabled())
			},
		},
		{
			Label:       "Browser Headless",
			Description: "Run browser automation without visible window",
			Key:         "browser_headless",
			IsBool:      true,
			GetValue: func(em *EnhancedModel) string {
				if em.agent.Config().Browser.Headless {
					return "ON"
				}
				return "OFF"
			},
			Toggle: func(em *EnhancedModel) {
				em.agent.Config().Browser.Headless = !em.agent.Config().Browser.Headless
				_ = em.agent.SaveConfig()
			},
		},
		{
			Label:       "Dual Session",
			Description: "Enable agent-to-agent debate mode",
			Key:         "dual_session",
			IsBool:      true,
			GetValue: func(em *EnhancedModel) string {
				if em.dualSession.IsEnabled() {
					return "ON"
				}
				return "OFF"
			},
			Toggle: func(em *EnhancedModel) {
				if em.dualSession.IsEnabled() {
					em.dualSession.Disable()
				} else {
					em.dualSession.Enable()
				}
			},
		},
		{
			Label:       "Telegram",
			Description: "Enable/disable Telegram bot integration",
			Key:         "telegram",
			IsBool:      true,
			GetValue: func(em *EnhancedModel) string {
				if em.agent.Config().Telegram.Enabled {
					return "ON"
				}
				return "OFF"
			},
			Toggle: func(em *EnhancedModel) {
				cfg := em.agent.Config()
				cfg.Telegram.Enabled = !cfg.Telegram.Enabled
				_ = em.agent.SaveConfig()
			},
		},
		{
			Label:       "Telegram Notify",
			Description: "Notify on Telegram when a tool starts",
			Key:         "telegram_notify",
			IsBool:      true,
			GetValue: func(em *EnhancedModel) string {
				if em.agent.Config().Telegram.NotifyOnToolStart {
					return "ON"
				}
				return "OFF"
			},
			Toggle: func(em *EnhancedModel) {
				cfg := em.agent.Config()
				cfg.Telegram.NotifyOnToolStart = !cfg.Telegram.NotifyOnToolStart
				_ = em.agent.SaveConfig()
			},
		},
		{
			Label:       "Heartbeat",
			Description: "Periodic heartbeat interval",
			Key:         "heartbeat",
			IsBool:      false,
			GetValue: func(em *EnhancedModel) string {
				interval := em.agent.Config().HeartbeatInterval
				if interval <= 0 {
					return "disabled"
				}
				return fmt.Sprintf("%ds", interval)
			},
			Toggle: func(em *EnhancedModel) {
				// Heartbeat is not a simple toggle; show info instead
			},
		},
	}
}

// settingsUpdate handles keyboard input while the settings overlay is active.
func (m EnhancedModel) settingsUpdate(msg tea.KeyMsg) (EnhancedModel, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.settingsActive = false
		return m, nil

	case "up", "k":
		if m.settingsCursor > 0 {
			m.settingsCursor--
		}
		return m, nil

	case "down", "j":
		if m.settingsCursor < len(m.settingsItems)-1 {
			m.settingsCursor++
		}
		return m, nil

	case "enter", " ":
		if m.settingsCursor < len(m.settingsItems) {
			item := m.settingsItems[m.settingsCursor]
			if item.Toggle != nil {
				item.Toggle(&m)
			}
		}
		return m, nil
	}

	return m, nil
}

// settingsView renders the settings overlay.
func (m EnhancedModel) settingsView() string {
	boxWidth := m.width - 6
	if boxWidth < 40 {
		boxWidth = 40
	}

	var s strings.Builder

	// Title
	s.WriteString(SettingsTitleStyle.Render("Settings"))
	s.WriteString("\n\n")

	// Items
	for i, item := range m.settingsItems {
		cursor := "  "
		labelStyle := SettingsNormalStyle
		if i == m.settingsCursor {
			cursor = "▸ "
			labelStyle = SettingsSelectedStyle
		}

		value := item.GetValue(&m)

		var valueStyled string
		if item.IsBool {
			if value == "ON" {
				valueStyled = SettingsOnStyle.Render("[ON]")
			} else {
				valueStyled = SettingsOffStyle.Render("[OFF]")
			}
		} else {
			valueStyled = SettingsValueStyle.Render("[" + value + "]")
		}

		s.WriteString(labelStyle.Render(cursor+item.Label) + "  " + valueStyled)
		s.WriteString("\n")

		// Show description for selected item
		if i == m.settingsCursor {
			s.WriteString(PanelFooterStyle.Render("    " + item.Description))
			s.WriteString("\n")
		}
	}

	// Footer
	s.WriteString("\n")
	s.WriteString(PanelFooterStyle.Render("↑/↓ Navigate | Enter Toggle | Esc Close"))

	return SettingsBoxStyle.Width(boxWidth).Render(s.String())
}
