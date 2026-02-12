// Package tui provides live conversation viewing for dual sessions
package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ConversationView represents a live view of the dual session conversation
type ConversationView struct {
	enabled        bool
	lastMessageIdx int
	autoScroll     bool
}

// NewConversationView creates a new conversation view
func NewConversationView() *ConversationView {
	return &ConversationView{
		enabled:        false,
		lastMessageIdx: 0,
		autoScroll:     true,
	}
}

// Enable enables live conversation view
func (cv *ConversationView) Enable() {
	cv.enabled = true
	cv.lastMessageIdx = 0
}

// Disable disables live conversation view
func (cv *ConversationView) Disable() {
	cv.enabled = false
	cv.lastMessageIdx = 0
}

// IsEnabled returns whether conversation view is enabled
func (cv *ConversationView) IsEnabled() bool {
	return cv.enabled
}

// conversationUpdateMsg is sent when there's a new message in the conversation
type conversationUpdateMsg struct {
	speaker string
	content string
	turn    int
}

// conversationCompleteMsg is sent when the conversation ends
type conversationCompleteMsg struct {
	reason string
}

// checkConversationUpdates checks for new messages and returns a command to poll
func checkConversationUpdates(m *EnhancedModel) tea.Cmd {
	// If conversation view is not enabled, don't poll
	if m.conversationView == nil || !m.conversationView.enabled {
		return nil
	}

	// If dual session is not running, don't poll
	if !m.dualSession.IsRunning() {
		return nil
	}

	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return pollConversationMsg{}
	})
}

// pollConversationMsg triggers a check for new conversation messages
type pollConversationMsg struct{}

// handleConversationUpdate handles new conversation messages
func (m *EnhancedModel) handleConversationUpdate() tea.Cmd {
	if m.conversationView == nil || !m.conversationView.enabled {
		return nil
	}

	if !m.dualSession.IsRunning() {
		// Conversation ended
		m.conversationView.Disable()
		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   "🏁 **Conversation Ended**\n\nThe dual session debate has completed.",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return nil
	}

	log := m.dualSession.GetConversationLog()
	if len(log) > m.conversationView.lastMessageIdx {
		// New messages available
		for i := m.conversationView.lastMessageIdx; i < len(log); i++ {
			msg := log[i]

			// Format the message with dynamic role names
			label := msg.Speaker
			if msg.RoleName != "" {
				label = msg.RoleName
			}
			// First speaker (Agent A) gets blue icon, others get green
			icon := "🟢"
			if msg.Speaker == m.dualSession.GetRoleNameA() {
				icon = "🔵"
			}

			header := fmt.Sprintf("%s **%s** (Turn %d)", icon, label, msg.Turn)

			m.messageQueue.Add(QueuedMessage{
				Role:      "assistant",
				Content:   fmt.Sprintf("%s\n%s", header, msg.Content),
				Timestamp: msg.Timestamp,
				Complete:  true,
			})
		}
		m.conversationView.lastMessageIdx = len(log)
		m.updateViewport()
	}

	// Continue polling
	return checkConversationUpdates(m)
}

// formatConversationHeader formats a header for the conversation view
func formatConversationHeader(m *EnhancedModel) string {
	current, max := m.dualSession.GetProgress()
	percent := 0
	if max > 0 {
		percent = (current * 100) / max
	}

	var sb strings.Builder
	sb.WriteString("🤖 **Live Conversation View**\n\n")
	sb.WriteString(fmt.Sprintf("Progress: Turn %d/%d (%d%%)\n", current, max, percent))
	sb.WriteString(fmt.Sprintf("Status: %s\n\n", func() string {
		if m.dualSession.IsRunning() {
			return "🔄 Active"
		}
		return "⏸️ Paused"
	}()))
	sb.WriteString(strings.Repeat("═", 60) + "\n")
	sb.WriteString("**Tip:** New messages will appear here in real-time.\n")
	sb.WriteString("Use `/stop` to end the debate early.\n")
	sb.WriteString(strings.Repeat("═", 60) + "\n\n")

	return sb.String()
}
