package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// SettingsItem represents a single toggleable or editable setting.
type SettingsItem struct {
	Label       string
	Description string
	Key         string
	GetValue    func(*EnhancedModel) string
	Toggle      func(*EnhancedModel)           // For bool items
	SetValue    func(*EnhancedModel, string)    // For editable items
	IsBool      bool
	IsEditable  bool                            // true for text/number input
}

// openSettings activates the settings overlay and optionally focuses the item matching focusKey.
func (m *EnhancedModel) openSettings(focusKey string) {
	m.settingsItems = m.buildSettingsItems()
	m.settingsCursor = 0
	m.settingsEditing = false
	m.settingsEditBuffer = ""

	for i, item := range m.settingsItems {
		if item.Key == focusKey {
			m.settingsCursor = i
			break
		}
	}

	m.settingsActive = true
}

// buildSettingsItems returns the full list of toggleable and editable settings.
func (m *EnhancedModel) buildSettingsItems() []SettingsItem {
	return []SettingsItem{
		// ── Identity ──────────────────────────────────────
		{
			Label:       "Agent Name",
			Description: "Display name used in system prompt and identity",
			Key:         "agent_name",
			IsEditable:  true,
			GetValue: func(em *EnhancedModel) string {
				name := em.agent.Config().AgentName
				if name == "" {
					return "(not set)"
				}
				return name
			},
			SetValue: func(em *EnhancedModel, v string) {
				em.agent.Config().AgentName = v
				_ = em.agent.SaveConfig()
			},
		},
		{
			Label:       "User Name",
			Description: "Your display name (injected into system prompt)",
			Key:         "user_name",
			IsEditable:  true,
			GetValue: func(em *EnhancedModel) string {
				name := em.agent.Config().UserName
				if name == "" {
					return "(not set)"
				}
				return name
			},
			SetValue: func(em *EnhancedModel, v string) {
				em.agent.Config().UserName = v
				_ = em.agent.SaveConfig()
			},
		},
		// ── Model ─────────────────────────────────────────
		{
			Label:       "Model",
			Description: "Active LLM model identifier",
			Key:         "model",
			IsEditable:  true,
			GetValue: func(em *EnhancedModel) string {
				return em.agent.Config().Model
			},
			SetValue: func(em *EnhancedModel, v string) {
				em.agent.Config().Model = v
				_ = em.agent.SaveConfig()
			},
		},
		{
			Label:       "API Base URL",
			Description: "LLM provider API endpoint",
			Key:         "api_base_url",
			IsEditable:  true,
			GetValue: func(em *EnhancedModel) string {
				return em.agent.Config().APIBaseURL
			},
			SetValue: func(em *EnhancedModel, v string) {
				em.agent.Config().APIBaseURL = v
				_ = em.agent.SaveConfig()
			},
		},
		// ── Toggles ───────────────────────────────────────
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
			Label:       "Git Tools",
			Description: "Enable git tools (commit, push, diff, etc.)",
			Key:         "git_tools",
			IsBool:      true,
			GetValue: func(em *EnhancedModel) string {
				if em.agent.Config().EnableGitTools {
					return "ON"
				}
				return "OFF"
			},
			Toggle: func(em *EnhancedModel) {
				em.agent.Config().EnableGitTools = !em.agent.Config().EnableGitTools
				_ = em.agent.SaveConfig()
			},
		},
		{
			Label:       "SSH Tools",
			Description: "Enable SSH tools (connect, exec, upload, download). Restart required.",
			Key:         "ssh_tools",
			IsBool:      true,
			GetValue: func(em *EnhancedModel) string {
				if em.agent.Config().SSH.Enabled {
					return "ON"
				}
				return "OFF"
			},
			Toggle: func(em *EnhancedModel) {
				em.agent.Config().SSH.Enabled = !em.agent.Config().SSH.Enabled
				_ = em.agent.SaveConfig()
			},
		},
		{
			Label:       "SSH Visual Mode",
			Description: "Open monitor window to watch SSH commands (credentials from config, not model)",
			Key:         "ssh_visual",
			IsBool:      true,
			GetValue: func(em *EnhancedModel) string {
				if em.agent.Config().SSH.VisualMode {
					return "ON"
				}
				return "OFF"
			},
			Toggle: func(em *EnhancedModel) {
				em.agent.Config().SSH.VisualMode = !em.agent.Config().SSH.VisualMode
				_ = em.agent.SaveConfig()
			},
		},
		// ── Editable numbers ──────────────────────────────
		{
			Label:       "Heartbeat Interval",
			Description: "Seconds between heartbeat checks (0 = disabled)",
			Key:         "heartbeat",
			IsEditable:  true,
			GetValue: func(em *EnhancedModel) string {
				interval := em.agent.Config().HeartbeatInterval
				if interval <= 0 {
					return "0"
				}
				return fmt.Sprintf("%d", interval)
			},
			SetValue: func(em *EnhancedModel, v string) {
				if n, err := strconv.Atoi(v); err == nil && n >= 0 {
					em.agent.Config().HeartbeatInterval = n
					_ = em.agent.SaveConfig()
				}
			},
		},
		{
			Label:       "Idle Threshold",
			Description: "Seconds of inactivity before heartbeat can act (default: 30)",
			Key:         "heartbeat_idle",
			IsEditable:  true,
			GetValue: func(em *EnhancedModel) string {
				return fmt.Sprintf("%d", em.agent.Config().GetHeartbeatIdleThreshold())
			},
			SetValue: func(em *EnhancedModel, v string) {
				if n, err := strconv.Atoi(v); err == nil && n >= 0 {
					em.agent.Config().HeartbeatIdleThreshold = n
					_ = em.agent.SaveConfig()
				}
			},
		},
		{
			Label:       "Workplace Dir",
			Description: "Sandbox directory name for agent file operations",
			Key:         "workplace_dir",
			IsEditable:  true,
			GetValue: func(em *EnhancedModel) string {
				return em.agent.Config().GetWorkplaceDir()
			},
			SetValue: func(em *EnhancedModel, v string) {
				v = strings.TrimSpace(v)
				if v != "" {
					em.agent.Config().WorkplaceDir = v
					_ = em.agent.SaveConfig()
				}
			},
		},
		{
			Label:       "Viewport Width",
			Description: "Browser automation viewport width in pixels",
			Key:         "viewport_w",
			IsEditable:  true,
			GetValue: func(em *EnhancedModel) string {
				return fmt.Sprintf("%d", em.agent.Config().GetBrowserViewportW())
			},
			SetValue: func(em *EnhancedModel, v string) {
				if n, err := strconv.Atoi(v); err == nil && n > 0 {
					em.agent.Config().BrowserViewportW = n
					_ = em.agent.SaveConfig()
				}
			},
		},
		{
			Label:       "Viewport Height",
			Description: "Browser automation viewport height in pixels",
			Key:         "viewport_h",
			IsEditable:  true,
			GetValue: func(em *EnhancedModel) string {
				return fmt.Sprintf("%d", em.agent.Config().GetBrowserViewportH())
			},
			SetValue: func(em *EnhancedModel, v string) {
				if n, err := strconv.Atoi(v); err == nil && n > 0 {
					em.agent.Config().BrowserViewportH = n
					_ = em.agent.SaveConfig()
				}
			},
		},
	}
}

// settingsUpdate handles keyboard input while the settings overlay is active.
func (m EnhancedModel) settingsUpdate(msg tea.KeyMsg) (EnhancedModel, tea.Cmd) {
	// ── Edit mode ─────────────────────────────────────────
	if m.settingsEditing {
		switch msg.Type {
		case tea.KeyEsc:
			// Cancel edit
			m.settingsEditing = false
			m.settingsEditBuffer = ""
			return m, nil

		case tea.KeyEnter:
			// Confirm edit
			if m.settingsCursor < len(m.settingsItems) {
				item := m.settingsItems[m.settingsCursor]
				if item.SetValue != nil {
					item.SetValue(&m, m.settingsEditBuffer)
				}
			}
			m.settingsEditing = false
			m.settingsEditBuffer = ""
			return m, nil

		case tea.KeyBackspace:
			if len(m.settingsEditBuffer) > 0 {
				m.settingsEditBuffer = m.settingsEditBuffer[:len(m.settingsEditBuffer)-1]
			}
			return m, nil

		case tea.KeyDelete:
			m.settingsEditBuffer = ""
			return m, nil

		default:
			// Append typed runes
			if msg.Type == tea.KeyRunes {
				m.settingsEditBuffer += string(msg.Runes)
			} else if msg.Type == tea.KeySpace {
				m.settingsEditBuffer += " "
			}
			return m, nil
		}
	}

	// ── Normal navigation mode ────────────────────────────
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
			if item.IsBool && item.Toggle != nil {
				item.Toggle(&m)
			} else if item.IsEditable {
				// Enter edit mode
				m.settingsEditing = true
				val := item.GetValue(&m)
				if val == "(not set)" {
					val = ""
				}
				m.settingsEditBuffer = val
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

		var valueStyled string

		if m.settingsEditing && i == m.settingsCursor {
			// Show edit buffer with cursor
			buf := m.settingsEditBuffer
			valueStyled = SettingsEditStyle.Render("[" + buf) +
				SettingsEditCursorStyle.Render("_") +
				SettingsEditStyle.Render("]")
		} else if item.IsBool {
			value := item.GetValue(&m)
			if value == "ON" {
				valueStyled = SettingsOnStyle.Render("[ON]")
			} else {
				valueStyled = SettingsOffStyle.Render("[OFF]")
			}
		} else {
			value := item.GetValue(&m)
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
	if m.settingsEditing {
		s.WriteString(PanelFooterStyle.Render("Enter Confirm | Esc Cancel | Del Clear"))
	} else {
		s.WriteString(PanelFooterStyle.Render("↑/↓ Navigate | Enter Edit/Toggle | Esc Close"))
	}

	return SettingsBoxStyle.Width(boxWidth).Render(s.String())
}
