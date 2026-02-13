package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// Init initializes the enhanced TUI
// Following Bubble Tea best practices: use tea.Sequence for ordered commands
func (m EnhancedModel) Init() tea.Cmd {
	return tea.Sequence(
		textarea.Blink,     // Start cursor blinking
		m.spinner.Tick,     // Start spinner animation
		tickAnimation(),   // Start animation ticker
	)
}

// tickAnimation creates a periodic command for animation updates
// Using 250ms interval for smooth animations (4 FPS)
func tickAnimation() tea.Cmd {
	return tea.Tick(time.Millisecond*250, func(t time.Time) tea.Msg {
		return animationTickMsg{}
	})
}
