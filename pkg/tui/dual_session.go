// Package tui provides dual session support for agent-to-agent conversations
package tui

import (
	"fmt"
	"os"
	"path/filepath"
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
	Speaker   string // "Agent A" or "Agent B"
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

// runConversation runs the actual conversation loop with enhanced robustness
func (ds *DualSession) runConversation(initialPrompt string) {
	defer func() {
		ds.mu.Lock()
		ds.running = false
		ds.mu.Unlock()

		// Save the conversation log automatically at the end
		filename := ds.saveConversationLog()
		if filename != "" {
			ds.addMessage("System", fmt.Sprintf("💾 Debate log saved to: %s", filename), ds.currentTurn)
		}
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

		// Robust Turn Loop: Retry if turn fails or returns empty,
		// but wait indefinitely while the agent is 'active' (thinking/working)
		var response string
		var err error
		maxRetries := 3

		for attempt := 0; attempt <= maxRetries; attempt++ {
			if attempt > 0 {
				ds.addMessage("System", fmt.Sprintf("🔄 [Attempt %d/%d] Nudging %s...", attempt+1, maxRetries+1, currentSpeaker), turnNum)
				// Exponential backoff: 5s, 15s, 45s
				backoff := time.Duration(5<<uint(attempt-1)) * time.Second
				if backoff > 60*time.Second {
					backoff = 60 * time.Second
				}
				time.Sleep(backoff)
			}

			// We use a channel to monitor the Chat execution
			type chatResult struct {
				resp string
				err  error
			}
			resultChan := make(chan chatResult, 1)

			go func() {
				// We call Chat. This is synchronous and will block until done.
				// It updates Agent.lastActivity internally during tool calls.
				resp, e := currentAgent.Chat(currentMessage)
				resultChan <- chatResult{resp, e}
			}()

			// Liveness Check Loop
			stuckThreshold := 3 * time.Minute // Nudge if silent for 3 mins
			ticker := time.NewTicker(30 * time.Second)
			activeWaiting := true

			for activeWaiting {
				select {
				case <-ds.stopChan:
					return
				case res := <-resultChan:
					response = res.resp
					err = res.err
					activeWaiting = false
					ticker.Stop()
				case <-ticker.C:
					// Check for 'dead air'
					lastAct := currentAgent.GetLastActivity()
					if time.Since(lastAct) > stuckThreshold {
						ds.addMessage("System",
							fmt.Sprintf("⚠️ %s seems stuck (no activity for %s). Attempting to wake up...",
								currentSpeaker, stuckThreshold.Round(time.Minute)), turnNum)
						// We can't safely kill the Chat goroutine, but we can break out
						// of this wait and try a retry if desired.
						// However, if it's still running, it might eventually finish.
						// For now, we continue waiting because the user said "aguardar".
					}
				}
			}

			if err == nil && response != "" {
				break
			}

			if err != nil {
				errStr := err.Error()
				if isRateLimitError(errStr) {
					wait := rateLimitWait(errStr)
					ds.addMessage("System", fmt.Sprintf("⏳ Rate limit hit. Waiting %s before retry...", wait.Round(time.Second)), turnNum)
					select {
					case <-ds.stopChan:
						return
					case <-time.After(wait):
					}
				} else if isContextLimitError(errStr) {
					ds.addMessage("System", "⚠️ Context window full. Truncating history and retrying...", turnNum)
					currentMessage = truncateForContext(currentMessage)
				} else {
					ds.addMessage("System", fmt.Sprintf("⚠️ Turn error: %v", err), turnNum)
				}
			} else if response == "" {
				ds.addMessage("System", "⚠️ Received empty response.", turnNum)
			}
		}

		if err != nil || response == "" {
			ds.addMessage("System", fmt.Sprintf("❌ Turn failed after %d retries. Stopping debate.", maxRetries), turnNum)
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

		// Delay between turns: respect API rate limits (minimum 1.5s)
		time.Sleep(1500 * time.Millisecond)
	}
}

// isRateLimitError returns true if the error indicates an API rate limit (429).
func isRateLimitError(s string) bool {
	return strings.Contains(s, "429") ||
		strings.Contains(s, "rate limit") ||
		strings.Contains(s, "rate_limit") ||
		strings.Contains(s, "too many requests")
}

// isContextLimitError returns true if the error indicates a context-length overflow.
func isContextLimitError(s string) bool {
	return strings.Contains(s, "context_length_exceeded") ||
		strings.Contains(s, "context window") ||
		strings.Contains(s, "tokens_exceeded") ||
		strings.Contains(s, "maximum context") ||
		strings.Contains(s, "prompt is too long")
}

// rateLimitWait returns how long to wait after a rate-limit error.
// Defaults to 30 seconds when no specific value is found.
func rateLimitWait(errMsg string) time.Duration {
	// Look for "retry after N" pattern
	lower := strings.ToLower(errMsg)
	idx := strings.Index(lower, "retry after ")
	if idx >= 0 {
		rest := errMsg[idx+len("retry after "):]
		// Try to parse seconds
		var secs int
		if n, _ := fmt.Sscanf(rest, "%d", &secs); n == 1 && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return 30 * time.Second
}

// truncateForContext truncates a message to fit within typical context limits.
// Keeps the last 4000 characters so the conversation stays meaningful.
func truncateForContext(msg string) string {
	const maxLen = 4000
	if len(msg) <= maxLen {
		return msg
	}
	return "[...truncated...]\n" + msg[len(msg)-maxLen:]
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
		_ = ds.multiWindow.WriteMessage(speaker, content, turn)
	}

	// Always append to a global debate log file
	ds.appendToGlobalLog(msg)
}

// saveConversationLog saves the full conversation log to a Markdown file
func (ds *DualSession) saveConversationLog() string {
	ds.mu.RLock()
	if len(ds.conversationLog) == 0 {
		ds.mu.RUnlock()
		return ""
	}

	content := ds.FormatConversation()
	workplacePath := ds.agentA.GetWorkplacePath()
	ds.mu.RUnlock()

	// Create debates directory in workplace
	debatesDir := filepath.Join(workplacePath, "debates")
	if err := os.MkdirAll(debatesDir, 0755); err != nil {
		ds.agentA.GetLogger().Error("Failed to create debates directory: %v", err)
	}

	// Generate filename based on timestamp and topic
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(debatesDir, fmt.Sprintf("debate_%s.md", timestamp))

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		ds.agentA.GetLogger().Error("Failed to save debate log: %v", err)
	}
	return filename
}

// appendToGlobalLog appends a single message to a persistent debate log
func (ds *DualSession) appendToGlobalLog(msg DualMessage) {
	workplacePath := ds.agentA.GetWorkplacePath()
	logFile := filepath.Join(workplacePath, "debate_history.log")

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	entry := fmt.Sprintf("[%s] %s (Turn %d): %s\n\n",
		msg.Timestamp.Format("2006-01-02 15:04:05"),
		msg.Speaker,
		msg.Turn,
		msg.Content)

	if _, err := f.WriteString(entry); err != nil {
		// Log write failure but don't crash — debate log is non-critical
		_ = err
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
