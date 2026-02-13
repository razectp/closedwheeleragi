// Package tui provides an in-TUI lipgloss-based debate viewer overlay.
// It replaces the external terminal window approach (multi_window.go) with a
// cross-platform overlay that renders inside the existing Bubble Tea TUI.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// DebateView holds the viewport and state for the debate viewer overlay.
type DebateView struct {
	Viewport viewport.Model
}

// debateViewerTickMsg triggers a periodic poll for new debate messages.
type debateViewerTickMsg struct{}

// openDebateViewer activates the debate viewer overlay.
// It reads the current conversation log and enables auto-scroll.
func (m *EnhancedModel) openDebateViewer() {
	m.debateViewActive = true
	m.debateViewScroll = 0
	m.debateViewAutoScroll = true

	log := m.dualSession.GetConversationLog()
	m.debateViewLastCount = len(log)

	// Calculate initial scroll bounds
	m.recalcDebateViewScroll()

	// Auto-scroll to bottom
	if m.debateViewAutoScroll && m.debateViewMaxScroll > 0 {
		m.debateViewScroll = m.debateViewMaxScroll
	}
}

// recalcDebateViewScroll recalculates scroll bounds based on content and terminal size.
func (m *EnhancedModel) recalcDebateViewScroll() {
	lines := m.debateViewerContentLines()
	visible := m.debateViewerVisibleHeight()
	m.debateViewMaxScroll = lines - visible
	if m.debateViewMaxScroll < 0 {
		m.debateViewMaxScroll = 0
	}
	if m.debateViewScroll < 0 {
		m.debateViewScroll = 0
	}
	if m.debateViewScroll > m.debateViewMaxScroll {
		m.debateViewScroll = m.debateViewMaxScroll
	}
}

// debateViewerVisibleHeight returns how many content lines fit in the overlay.
func (m *EnhancedModel) debateViewerVisibleHeight() int {
	// total height minus: border(2) + margin(2) + padding(2) + title(1) + progress(1)
	// + blank(1) + scroll-up indicator(1) + scroll-down indicator(1) + footer(1) + blank(1)
	h := m.height - 13
	if h < 5 {
		h = 5
	}
	return h
}

// debateViewerContentLines returns the total number of rendered lines.
func (m *EnhancedModel) debateViewerContentLines() int {
	log := m.dualSession.GetConversationLog()
	if len(log) == 0 {
		if m.dualSession.IsRunning() {
			return 3 // thinking indicator + blank + helper text
		}
		return 1 // "Waiting for first message..." placeholder
	}

	boxWidth := m.width - 10
	if boxWidth < 30 {
		boxWidth = 30
	}

	total := 0
	for _, msg := range log {
		total += m.debateViewerMessageLineCount(msg, boxWidth)
	}
	return total
}

// debateViewerMessageLineCount returns the number of rendered lines for a single message.
func (m *EnhancedModel) debateViewerMessageLineCount(msg DualMessage, boxWidth int) int {
	lines := 0

	// Header line (speaker + turn + time)
	lines++

	// Divider line
	lines++

	// Content lines (word-wrapped)
	contentWidth := boxWidth - 4
	if contentWidth < 20 {
		contentWidth = 20
	}
	wrapped := wrapText(msg.Content, contentWidth)
	lines += len(wrapped)

	// Blank separator
	lines++

	return lines
}

// wrapText splits text into lines, each no wider than maxWidth.
func wrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		maxWidth = 40
	}

	var result []string
	for _, line := range strings.Split(text, "\n") {
		if len(line) == 0 {
			result = append(result, "")
			continue
		}
		for len(line) > maxWidth {
			// Try to break at a space
			breakAt := maxWidth
			for i := maxWidth; i > maxWidth/2; i-- {
				if line[i] == ' ' {
					breakAt = i
					break
				}
			}
			result = append(result, line[:breakAt])
			line = line[breakAt:]
			// Skip leading space after break
			if len(line) > 0 && line[0] == ' ' {
				line = line[1:]
			}
		}
		result = append(result, line)
	}
	return result
}

// debateViewerUpdate handles keyboard input while the debate viewer overlay is active.
func (m EnhancedModel) debateViewerUpdate(msg tea.KeyMsg) (EnhancedModel, tea.Cmd) {
	pageSize := m.debateViewerVisibleHeight()

	switch msg.String() {
	case "esc", "q":
		m.debateViewActive = false
		return m, nil

	case "up", "k":
		if m.debateViewScroll > 0 {
			m.debateViewScroll--
			m.debateViewAutoScroll = false
		}
		return m, nil

	case "down", "j":
		if m.debateViewScroll < m.debateViewMaxScroll {
			m.debateViewScroll++
		}
		// Re-enable auto-scroll if at bottom
		if m.debateViewScroll >= m.debateViewMaxScroll {
			m.debateViewAutoScroll = true
		}
		return m, nil

	case "pgup", "b":
		m.debateViewScroll -= pageSize
		if m.debateViewScroll < 0 {
			m.debateViewScroll = 0
		}
		m.debateViewAutoScroll = false
		return m, nil

	case "pgdown", "f":
		m.debateViewScroll += pageSize
		if m.debateViewScroll > m.debateViewMaxScroll {
			m.debateViewScroll = m.debateViewMaxScroll
		}
		if m.debateViewScroll >= m.debateViewMaxScroll {
			m.debateViewAutoScroll = true
		}
		return m, nil

	case "home", "g":
		m.debateViewScroll = 0
		m.debateViewAutoScroll = false
		return m, nil

	case "end", "G":
		m.debateViewScroll = m.debateViewMaxScroll
		m.debateViewAutoScroll = true
		return m, nil
	}

	return m, nil
}

// debateViewerTick returns a command that polls for new messages every 500ms.
func debateViewerTick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return debateViewerTickMsg{}
	})
}

// handleDebateViewerTick processes a tick: checks for new messages and refreshes scroll.
func (m *EnhancedModel) handleDebateViewerTick() tea.Cmd {
	if !m.debateViewActive {
		return nil
	}

	log := m.dualSession.GetConversationLog()
	newCount := len(log)

	if newCount != m.debateViewLastCount {
		m.debateViewLastCount = newCount
		m.recalcDebateViewScroll()

		// Auto-scroll to bottom when new messages arrive
		if m.debateViewAutoScroll {
			m.debateViewScroll = m.debateViewMaxScroll
		}
	}

	// Keep polling while the debate is running or viewer is open
	if m.dualSession.IsRunning() || m.debateViewActive {
		return debateViewerTick()
	}
	return nil
}

// debateViewerView renders the debate viewer overlay.
func (m EnhancedModel) debateViewerView() string {
	boxWidth := m.width - 6
	if boxWidth < 40 {
		boxWidth = 40
	}

	contentWidth := boxWidth - 6 // Account for padding and border
	if contentWidth < 30 {
		contentWidth = 30
	}

	visibleHeight := m.debateViewerVisibleHeight()

	var s strings.Builder

	// Title
	topic := m.dualSession.GetTopic()
	if topic == "" {
		topic = "Debate"
	}
	titleText := fmt.Sprintf("Live Debate: %s", topic)
	if len(titleText) > contentWidth-4 {
		titleText = titleText[:contentWidth-7] + "..."
	}
	s.WriteString(DebateViewTitleStyle.Render(titleText))
	s.WriteString("\n")

	// Progress line
	current, max := m.dualSession.GetProgress()
	percent := 0
	if max > 0 {
		percent = (current * 100) / max
	}
	statusIcon := DebateActiveStyle.Render("Active")
	if !m.dualSession.IsRunning() {
		statusIcon = DebateCompleteStyle.Render("Complete")
	}
	progressLine := fmt.Sprintf("Progress: Turn %d/%d (%d%%)  %s", current, max, percent, statusIcon)
	s.WriteString(DebateViewTurnStyle.Render(progressLine))
	s.WriteString("\n\n")

	// Build all content lines
	log := m.dualSession.GetConversationLog()

	var allLines []string
	if len(log) == 0 {
		if m.dualSession.IsRunning() {
			// Show animated thinking indicator with elapsed time
			dots := strings.Repeat(".", m.thinkingAnimation%4)
			elapsed := time.Since(m.dualSession.GetStartedAt()).Truncate(time.Second)
			speaker := m.dualSession.GetRoleNameA()
			allLines = []string{
				DebateViewThinkingStyle.Render(
					fmt.Sprintf("  %s is thinking%s (%s)", speaker, dots, elapsed)),
				"",
				DebateSystemStyle.Render("  The model is generating a response. This may take a moment."),
			}
		} else {
			allLines = []string{DebateSystemStyle.Render("Waiting for first message...")}
		}
	} else {
		roleNameA := m.dualSession.GetRoleNameA()
		roleNameB := m.dualSession.GetRoleNameB()
		for _, msg := range log {
			msgLines := m.renderDebateMessage(msg, roleNameA, contentWidth)
			allLines = append(allLines, msgLines...)
		}
		// Show thinking indicator for the next speaker while debate is running
		if m.dualSession.IsRunning() {
			lastMsg := log[len(log)-1]
			nextSpeaker := roleNameA
			if lastMsg.Speaker == roleNameA {
				nextSpeaker = roleNameB
			}
			dots := strings.Repeat(".", m.thinkingAnimation%4)
			allLines = append(allLines,
				DebateViewThinkingStyle.Render(
					fmt.Sprintf("  %s is thinking%s", nextSpeaker, dots)))
		}
	}

	// Scroll-up indicator
	if m.debateViewScroll > 0 {
		s.WriteString(DebateViewScrollStyle.Render(fmt.Sprintf("  ▲ %d more", m.debateViewScroll)))
		s.WriteString("\n")
	}

	// Visible slice
	end := m.debateViewScroll + visibleHeight
	if end > len(allLines) {
		end = len(allLines)
	}
	start := m.debateViewScroll
	if start < 0 {
		start = 0
	}
	if start > len(allLines) {
		start = len(allLines)
	}

	for i := start; i < end; i++ {
		line := allLines[i]
		s.WriteString(line)
		s.WriteString("\n")
	}

	// Pad remaining height if content is shorter than visible area
	rendered := end - start
	for i := rendered; i < visibleHeight; i++ {
		s.WriteString("\n")
	}

	// Scroll-down indicator
	remaining := len(allLines) - end
	if remaining > 0 {
		s.WriteString(DebateViewScrollStyle.Render(fmt.Sprintf("  ▼ %d more", remaining)))
		s.WriteString("\n")
	}

	// Footer
	autoIndicator := ""
	if m.debateViewAutoScroll {
		autoIndicator = " [auto-scroll]"
	}
	s.WriteString("\n")
	s.WriteString(DebateViewFooterStyle.Render(
		fmt.Sprintf("↑/↓ Scroll  PgUp/PgDn Page  Home/End  Esc Close%s", autoIndicator)))

	return DebateViewBoxStyle.Width(boxWidth).Render(s.String())
}

// renderDebateMessage renders a single debate message into styled lines.
func (m EnhancedModel) renderDebateMessage(msg DualMessage, roleNameA string, contentWidth int) []string {
	var lines []string

	// Speaker header with color
	icon := "🟢"
	speakerStyle := DebateAgentBStyle
	if msg.Speaker == roleNameA {
		icon = "🔵"
		speakerStyle = DebateAgentAStyle
	}

	// System messages get a different style
	if msg.Speaker != roleNameA && msg.Speaker != m.dualSession.GetRoleNameB() {
		header := DebateSystemStyle.Render(
			fmt.Sprintf("  [%s] %s", msg.Timestamp.Format("15:04:05"), msg.Content))
		lines = append(lines, header)
		lines = append(lines, "") // blank separator
		return lines
	}

	header := fmt.Sprintf("%s %s (Turn %d) — %s",
		icon,
		speakerStyle.Render(msg.Speaker),
		msg.Turn,
		msg.Timestamp.Format("15:04:05"))
	lines = append(lines, header)

	// Divider
	divLen := contentWidth - 2
	if divLen < 10 {
		divLen = 10
	}
	if divLen > 40 {
		divLen = 40
	}
	lines = append(lines, DebateViewDividerStyle.Render(strings.Repeat("─", divLen)))

	// Content (word-wrapped)
	wrapped := wrapText(msg.Content, contentWidth-2)
	lines = append(lines, wrapped...)

	// Blank separator
	lines = append(lines, "")

	return lines
}
