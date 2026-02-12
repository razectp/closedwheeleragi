package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// openPanel activates the read-only panel overlay with the given title and content.
func (m *EnhancedModel) openPanel(title, content string) {
	m.panelTitle = title
	m.panelLines = strings.Split(content, "\n")
	m.panelScroll = 0

	visibleHeight := m.panelVisibleHeight()
	m.panelMaxScroll = len(m.panelLines) - visibleHeight
	if m.panelMaxScroll < 0 {
		m.panelMaxScroll = 0
	}

	m.panelActive = true
}

// panelVisibleHeight returns how many content lines fit in the panel.
func (m *EnhancedModel) panelVisibleHeight() int {
	// total height minus: border(2) + margin(2) + padding(2) + title(1) + blank(1) + scroll indicator(1) + footer(1)
	h := m.height - 10
	if h < 5 {
		h = 5
	}
	return h
}

// panelUpdate handles keyboard input while the panel overlay is active.
func (m EnhancedModel) panelUpdate(msg tea.KeyMsg) (EnhancedModel, tea.Cmd) {
	pageSize := m.panelVisibleHeight()

	switch msg.String() {
	case "esc", "q":
		m.panelActive = false
		return m, nil

	case "up", "k":
		if m.panelScroll > 0 {
			m.panelScroll--
		}
		return m, nil

	case "down", "j":
		if m.panelScroll < m.panelMaxScroll {
			m.panelScroll++
		}
		return m, nil

	case "pgup", "b":
		m.panelScroll -= pageSize
		if m.panelScroll < 0 {
			m.panelScroll = 0
		}
		return m, nil

	case "pgdown", "f":
		m.panelScroll += pageSize
		if m.panelScroll > m.panelMaxScroll {
			m.panelScroll = m.panelMaxScroll
		}
		return m, nil

	case "home", "g":
		m.panelScroll = 0
		return m, nil

	case "end", "G":
		m.panelScroll = m.panelMaxScroll
		return m, nil
	}

	return m, nil
}

// panelView renders the panel overlay.
func (m EnhancedModel) panelView() string {
	boxWidth := m.width - 6
	if boxWidth < 40 {
		boxWidth = 40
	}

	visibleHeight := m.panelVisibleHeight()

	var s strings.Builder

	// Title
	s.WriteString(PanelTitleStyle.Render(m.panelTitle))
	s.WriteString("\n\n")

	// Scroll-up indicator
	if m.panelScroll > 0 {
		s.WriteString(PanelScrollStyle.Render(fmt.Sprintf("  ▲ %d more", m.panelScroll)))
		s.WriteString("\n")
	}

	// Visible lines
	end := m.panelScroll + visibleHeight
	if end > len(m.panelLines) {
		end = len(m.panelLines)
	}

	for i := m.panelScroll; i < end; i++ {
		line := m.panelLines[i]
		// Truncate long lines to avoid wrapping issues
		if len(line) > boxWidth-6 {
			line = line[:boxWidth-9] + "..."
		}
		s.WriteString(line)
		s.WriteString("\n")
	}

	// Scroll-down indicator
	remaining := len(m.panelLines) - end
	if remaining > 0 {
		s.WriteString(PanelScrollStyle.Render(fmt.Sprintf("  ▼ %d more", remaining)))
		s.WriteString("\n")
	}

	// Footer
	s.WriteString("\n")
	s.WriteString(PanelFooterStyle.Render("↑/↓ Scroll | PgUp/PgDn Page | Home/End | Esc Close"))

	return PanelBoxStyle.Width(boxWidth).Render(s.String())
}
