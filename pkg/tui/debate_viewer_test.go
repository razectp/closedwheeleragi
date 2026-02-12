package tui

import (
	"strings"
	"testing"
	"time"
)

// TestWrapText verifies basic word-wrapping behavior.
func TestWrapText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		maxWidth int
		wantMin  int // minimum number of lines
	}{
		{
			name:     "short line",
			text:     "hello",
			maxWidth: 40,
			wantMin:  1,
		},
		{
			name:     "exact width",
			text:     strings.Repeat("a", 40),
			maxWidth: 40,
			wantMin:  1,
		},
		{
			name:     "needs wrapping",
			text:     strings.Repeat("word ", 20),
			maxWidth: 30,
			wantMin:  2,
		},
		{
			name:     "multiline input",
			text:     "line one\nline two\nline three",
			maxWidth: 40,
			wantMin:  3,
		},
		{
			name:     "empty string",
			text:     "",
			maxWidth: 40,
			wantMin:  1, // one empty line
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := wrapText(tt.text, tt.maxWidth)
			if len(lines) < tt.wantMin {
				t.Errorf("wrapText(%q, %d) = %d lines, want >= %d",
					tt.text, tt.maxWidth, len(lines), tt.wantMin)
			}
			// No line should exceed maxWidth
			for i, line := range lines {
				if len(line) > tt.maxWidth+5 { // small tolerance for edge cases
					t.Errorf("line %d exceeds maxWidth: len=%d, max=%d, line=%q",
						i, len(line), tt.maxWidth, line)
				}
			}
		})
	}
}

// TestDebateViewerScrollBounds verifies scroll clamping.
func TestDebateViewerScrollBounds(t *testing.T) {
	m := EnhancedModel{
		width:  80,
		height: 40,
		dualSession: &DualSession{
			conversationLog: make([]DualMessage, 0),
			roleNameA:       "A",
			roleNameB:       "B",
		},
	}

	// Negative scroll should clamp to 0
	m.debateViewScroll = -5
	m.debateViewMaxScroll = 10
	m.recalcDebateViewScroll()
	if m.debateViewScroll < 0 {
		t.Errorf("scroll should not be negative, got %d", m.debateViewScroll)
	}

	// Scroll beyond max should clamp
	m.debateViewScroll = 999
	m.debateViewMaxScroll = 10
	m.recalcDebateViewScroll()
	if m.debateViewScroll > m.debateViewMaxScroll {
		t.Errorf("scroll %d should not exceed maxScroll %d",
			m.debateViewScroll, m.debateViewMaxScroll)
	}
}

// TestDebateViewerVisibleHeight verifies height calculation.
func TestDebateViewerVisibleHeight(t *testing.T) {
	m := EnhancedModel{height: 40}
	h := m.debateViewerVisibleHeight()
	if h <= 0 {
		t.Errorf("visibleHeight should be positive, got %d", h)
	}

	// Very small terminal should still give a minimum
	m.height = 10
	h = m.debateViewerVisibleHeight()
	if h < 5 {
		t.Errorf("visibleHeight should be >= 5 for small terminal, got %d", h)
	}
}

// TestDebateViewerEmptyState verifies rendering with no messages.
func TestDebateViewerEmptyState(t *testing.T) {
	m := EnhancedModel{
		width:           80,
		height:          40,
		debateViewActive: true,
		dualSession: &DualSession{
			conversationLog: make([]DualMessage, 0),
			roleNameA:       "Alice",
			roleNameB:       "Bob",
			topic:           "test topic",
		},
	}

	view := m.debateViewerView()
	if !strings.Contains(view, "Waiting for first message") {
		t.Error("empty state should show waiting message")
	}
	if !strings.Contains(view, "test topic") {
		t.Error("empty state should show topic")
	}
}

// TestDebateViewerSpeakerColors verifies Agent A gets blue icon and Agent B gets green.
func TestDebateViewerSpeakerColors(t *testing.T) {
	m := EnhancedModel{
		width:           80,
		height:          40,
		debateViewActive: true,
		dualSession: &DualSession{
			conversationLog: []DualMessage{
				{
					Speaker:   "Alice",
					Content:   "Hello from Agent A",
					Timestamp: time.Now(),
					Turn:      1,
					RoleName:  "Alice",
				},
				{
					Speaker:   "Bob",
					Content:   "Hello from Agent B",
					Timestamp: time.Now(),
					Turn:      2,
					RoleName:  "Bob",
				},
			},
			roleNameA: "Alice",
			roleNameB: "Bob",
			topic:     "colors test",
		},
	}

	view := m.debateViewerView()

	// Should contain both speaker indicators
	if !strings.Contains(view, "Alice") {
		t.Error("view should contain Agent A name")
	}
	if !strings.Contains(view, "Bob") {
		t.Error("view should contain Agent B name")
	}
}

// TestDebateViewerAutoScroll verifies auto-scroll behavior.
func TestDebateViewerAutoScroll(t *testing.T) {
	m := EnhancedModel{
		width:  80,
		height: 40,
	}

	// Auto-scroll should be enabled by default when opening
	m.dualSession = &DualSession{
		conversationLog: make([]DualMessage, 0),
		roleNameA:       "A",
		roleNameB:       "B",
	}
	m.openDebateViewer()

	if !m.debateViewAutoScroll {
		t.Error("auto-scroll should be enabled after openDebateViewer")
	}
	if !m.debateViewActive {
		t.Error("viewer should be active after openDebateViewer")
	}
}

// TestDebateViewerContentLines verifies line counting with messages.
func TestDebateViewerContentLines(t *testing.T) {
	m := EnhancedModel{
		width:  80,
		height: 40,
		dualSession: &DualSession{
			conversationLog: []DualMessage{
				{
					Speaker:   "Alice",
					Content:   "Short message",
					Timestamp: time.Now(),
					Turn:      1,
				},
			},
			roleNameA: "Alice",
			roleNameB: "Bob",
		},
	}

	lines := m.debateViewerContentLines()
	// At minimum: header(1) + divider(1) + content(1) + blank(1) = 4
	if lines < 4 {
		t.Errorf("expected at least 4 lines for one message, got %d", lines)
	}
}

// TestDebateViewerThinkingIndicator verifies the thinking indicator when debate is running with no messages.
func TestDebateViewerThinkingIndicator(t *testing.T) {
	m := EnhancedModel{
		width:           80,
		height:          40,
		debateViewActive: true,
		dualSession: &DualSession{
			conversationLog: make([]DualMessage, 0),
			roleNameA:       "Alice",
			roleNameB:       "Bob",
			topic:           "test topic",
			running:         true,
			startedAt:       time.Now(),
		},
	}

	view := m.debateViewerView()
	if !strings.Contains(view, "Alice is thinking") {
		t.Error("running debate with no messages should show thinking indicator for first speaker")
	}
	if !strings.Contains(view, "generating a response") {
		t.Error("running debate with no messages should show helper text")
	}

	// Content lines should be 3 (thinking + blank + helper)
	lines := m.debateViewerContentLines()
	if lines != 3 {
		t.Errorf("expected 3 content lines for thinking state, got %d", lines)
	}
}

// TestDebateViewerNextSpeakerThinking verifies the thinking indicator between turns.
func TestDebateViewerNextSpeakerThinking(t *testing.T) {
	m := EnhancedModel{
		width:           80,
		height:          40,
		debateViewActive: true,
		dualSession: &DualSession{
			conversationLog: []DualMessage{
				{
					Speaker:   "Alice",
					Content:   "First message",
					Timestamp: time.Now(),
					Turn:      1,
				},
			},
			roleNameA: "Alice",
			roleNameB: "Bob",
			topic:     "test",
			running:   true,
			startedAt: time.Now(),
		},
	}

	view := m.debateViewerView()
	// After Alice spoke, Bob should be thinking
	if !strings.Contains(view, "Bob is thinking") {
		t.Error("after Agent A speaks, should show Agent B thinking indicator")
	}
}

// TestRenderDebateMessage verifies that system messages render differently.
func TestRenderDebateMessage(t *testing.T) {
	m := EnhancedModel{
		width:  80,
		height: 40,
		dualSession: &DualSession{
			roleNameA: "Alice",
			roleNameB: "Bob",
		},
	}

	// Agent message
	agentMsg := DualMessage{
		Speaker:   "Alice",
		Content:   "Agent message",
		Timestamp: time.Now(),
		Turn:      1,
	}
	agentLines := m.renderDebateMessage(agentMsg, "Alice", 60)
	if len(agentLines) < 4 {
		t.Errorf("agent message should have >= 4 lines (header, divider, content, blank), got %d", len(agentLines))
	}

	// System message
	sysMsg := DualMessage{
		Speaker:   "System",
		Content:   "Rate limit hit",
		Timestamp: time.Now(),
		Turn:      1,
	}
	sysLines := m.renderDebateMessage(sysMsg, "Alice", 60)
	// System messages: just header + blank = 2
	if len(sysLines) != 2 {
		t.Errorf("system message should have 2 lines, got %d", len(sysLines))
	}
}
