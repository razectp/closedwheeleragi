// Package tui provides dual session support for agent-to-agent conversations
package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"ClosedWheeler/pkg/agent"
)

// DualSession manages two agents conversing with each other
type DualSession struct {
	agentA          *agent.Agent
	agentB          *agent.Agent
	enabled         bool
	running         bool
	conversationLog []DualMessage
	maxTurns        int
	currentTurn     int
	mu              sync.RWMutex
	stopChan        chan struct{}
	multiWindow     *MultiWindowManager // Optional multi-window for viewing (one per agent)
}

// DualMessage represents a message in the dual session
type DualMessage struct {
	Speaker   string    // "Agent A" or "Agent B"
	Content   string
	Timestamp time.Time
	Turn      int
}

// NewDualSession creates a new dual session manager
func NewDualSession(agentA, agentB *agent.Agent) *DualSession {
	return &DualSession{
		agentA:          agentA,
		agentB:          agentB,
		enabled:         false,
		running:         false,
		conversationLog: make([]DualMessage, 0),
		maxTurns:        20, // Default: 20 turns (10 exchanges)
		stopChan:        make(chan struct{}),
		multiWindow:     nil,
	}
}

// SetMultiWindow sets the multi-window manager
func (ds *DualSession) SetMultiWindow(mw *MultiWindowManager) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.multiWindow = mw
}

// Enable enables dual session mode
func (ds *DualSession) Enable() {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.enabled = true
}

// Disable disables dual session mode
func (ds *DualSession) Disable() {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.enabled = false
	if ds.running {
		close(ds.stopChan)
		ds.running = false
	}
}

// IsEnabled returns whether dual session is enabled
func (ds *DualSession) IsEnabled() bool {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.enabled
}

// IsRunning returns whether a conversation is currently running
func (ds *DualSession) IsRunning() bool {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.running
}

// SetMaxTurns sets the maximum number of turns for a conversation
func (ds *DualSession) SetMaxTurns(turns int) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.maxTurns = turns
}

// StartConversation starts a conversation between the two agents
func (ds *DualSession) StartConversation(initialPrompt string) error {
	ds.mu.Lock()
	if !ds.enabled {
		ds.mu.Unlock()
		return fmt.Errorf("dual session is not enabled")
	}
	if ds.running {
		ds.mu.Unlock()
		return fmt.Errorf("conversation already running")
	}
	ds.running = true
	ds.currentTurn = 0
	ds.conversationLog = make([]DualMessage, 0)
	ds.stopChan = make(chan struct{})
	ds.mu.Unlock()

	// Run conversation in background
	go ds.runConversation(initialPrompt)

	return nil
}

// StopConversation stops the current conversation
func (ds *DualSession) StopConversation() {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if ds.running {
		close(ds.stopChan)
		ds.running = false
	}
}

// runConversation runs the actual conversation loop
func (ds *DualSession) runConversation(initialPrompt string) {
	defer func() {
		ds.mu.Lock()
		ds.running = false
		ds.mu.Unlock()
	}()

	// Start with Agent A receiving the initial prompt
	currentMessage := initialPrompt
	currentAgent := ds.agentA
	currentSpeaker := "Agent A"

	for {
		// Check if we should stop
		select {
		case <-ds.stopChan:
			return
		default:
		}

		// Check turn limit
		ds.mu.Lock()
		if ds.currentTurn >= ds.maxTurns {
			ds.mu.Unlock()
			return
		}
		ds.currentTurn++
		turnNum := ds.currentTurn
		ds.mu.Unlock()

		// Get response from current agent
		response, err := currentAgent.Chat(currentMessage)
		if err != nil {
			// Log error and stop
			ds.addMessage(currentSpeaker, fmt.Sprintf("[ERROR: %v]", err), turnNum)
			return
		}

		// Add to conversation log
		ds.addMessage(currentSpeaker, response, turnNum)

		// Check for stop conditions
		if ds.shouldStopConversation(response) {
			return
		}

		// Switch agents
		if currentAgent == ds.agentA {
			currentAgent = ds.agentB
			currentSpeaker = "Agent B"
		} else {
			currentAgent = ds.agentA
			currentSpeaker = "Agent A"
		}
		currentMessage = response

		// Small delay between turns
		time.Sleep(500 * time.Millisecond)
	}
}

// addMessage adds a message to the conversation log
func (ds *DualSession) addMessage(speaker, content string, turn int) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	msg := DualMessage{
		Speaker:   speaker,
		Content:   content,
		Timestamp: time.Now(),
		Turn:      turn,
	}

	ds.conversationLog = append(ds.conversationLog, msg)

	// Write to multi-window if enabled
	if ds.multiWindow != nil && ds.multiWindow.IsEnabled() {
		ds.multiWindow.WriteMessage(speaker, content, turn)
	}
}

// shouldStopConversation checks if the conversation should stop based on content
func (ds *DualSession) shouldStopConversation(content string) bool {
	lowerContent := strings.ToLower(content)

	// Stop if agents agree to end or say goodbye
	stopPhrases := []string{
		"goodbye",
		"end conversation",
		"nothing more to discuss",
		"we've covered everything",
		"let's end here",
	}

	for _, phrase := range stopPhrases {
		if strings.Contains(lowerContent, phrase) {
			return true
		}
	}

	return false
}

// GetConversationLog returns the full conversation log
func (ds *DualSession) GetConversationLog() []DualMessage {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	// Return a copy
	log := make([]DualMessage, len(ds.conversationLog))
	copy(log, ds.conversationLog)
	return log
}

// GetLastMessage returns the last message in the conversation
func (ds *DualSession) GetLastMessage() *DualMessage {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if len(ds.conversationLog) == 0 {
		return nil
	}

	msg := ds.conversationLog[len(ds.conversationLog)-1]
	return &msg
}

// GetProgress returns current progress (turn/maxTurns)
func (ds *DualSession) GetProgress() (int, int) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.currentTurn, ds.maxTurns
}

// FormatConversation formats the conversation log as a readable string
func (ds *DualSession) FormatConversation() string {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if len(ds.conversationLog) == 0 {
		return "No messages in conversation."
	}

	var sb strings.Builder
	sb.WriteString("🤖 Dual Session Conversation Log\n")
	sb.WriteString(strings.Repeat("═", 60) + "\n\n")

	for _, msg := range ds.conversationLog {
		// Speaker header
		if msg.Speaker == "Agent A" {
			sb.WriteString(fmt.Sprintf("🔵 %s (Turn %d) - %s\n",
				msg.Speaker, msg.Turn, msg.Timestamp.Format("15:04:05")))
		} else {
			sb.WriteString(fmt.Sprintf("🟢 %s (Turn %d) - %s\n",
				msg.Speaker, msg.Turn, msg.Timestamp.Format("15:04:05")))
		}

		// Content
		sb.WriteString(strings.Repeat("─", 60) + "\n")
		sb.WriteString(msg.Content)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// GetStats returns statistics about the conversation
func (ds *DualSession) GetStats() map[string]interface{} {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	agentACount := 0
	agentBCount := 0
	totalChars := 0

	for _, msg := range ds.conversationLog {
		if msg.Speaker == "Agent A" {
			agentACount++
		} else {
			agentBCount++
		}
		totalChars += len(msg.Content)
	}

	return map[string]interface{}{
		"total_messages":   len(ds.conversationLog),
		"agent_a_messages": agentACount,
		"agent_b_messages": agentBCount,
		"current_turn":     ds.currentTurn,
		"max_turns":        ds.maxTurns,
		"total_chars":      totalChars,
		"is_running":       ds.running,
	}
}
