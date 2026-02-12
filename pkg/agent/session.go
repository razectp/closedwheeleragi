// Package agent provides session management for context optimization
package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"ClosedWheeler/pkg/llm"
)

// Session tracks conversation state to optimize context usage
type Session struct {
	ID                string
	SystemPromptHash  string // Hash of system prompt to detect changes
	RulesHash         string // Hash of rules to detect changes
	ProjectHash       string // Hash of project info to detect changes
	Messages          []llm.Message
	ContextSent       bool // True if initial context was sent
	LastActivity      time.Time
	TotalPromptTokens int
	TotalCompletions  int
	mu                sync.RWMutex
}

// SessionManager manages conversation sessions
type SessionManager struct {
	currentSession *Session
	maxMessages    int // configurable max messages per session
	mu             sync.RWMutex
}

// NewSessionManager creates a new session manager.
// maxMessages sets the cap on session history (0 uses default 1000).
func NewSessionManager(maxMessages ...int) *SessionManager {
	max := 1000
	if len(maxMessages) > 0 && maxMessages[0] > 0 {
		max = maxMessages[0]
	}
	return &SessionManager{
		currentSession: newSession(),
		maxMessages:    max,
	}
}

// newSession creates a fresh session
func newSession() *Session {
	return &Session{
		ID:           generateSessionID(),
		Messages:     make([]llm.Message, 0),
		ContextSent:  false,
		LastActivity: time.Now(),
	}
}

// generateSessionID creates a unique session ID
func generateSessionID() string {
	h := sha256.New()
	h.Write([]byte(time.Now().String()))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// hashContent creates a hash of content for change detection
func hashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:8]) // Use first 8 bytes for efficiency
}

// NeedsContextRefresh checks if context needs to be resent
func (sm *SessionManager) NeedsContextRefresh(systemPrompt, rules, projectInfo string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	s := sm.currentSession

	// First interaction - always send context
	if !s.ContextSent {
		return true
	}

	// Check for changes in any component
	sysHash := hashContent(systemPrompt)
	rulesHash := hashContent(rules)
	projHash := hashContent(projectInfo)

	return s.SystemPromptHash != sysHash ||
		s.RulesHash != rulesHash ||
		s.ProjectHash != projHash
}

// MarkContextSent marks context as sent and stores hashes
func (sm *SessionManager) MarkContextSent(systemPrompt, rules, projectInfo string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s := sm.currentSession
	s.SystemPromptHash = hashContent(systemPrompt)
	s.RulesHash = hashContent(rules)
	s.ProjectHash = hashContent(projectInfo)
	s.ContextSent = true
	s.LastActivity = time.Now()
}

// AddMessage adds a message to session history
// Prevents memory leaks by limiting message history to maxMessages
func (sm *SessionManager) AddMessage(msg llm.Message) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s := sm.currentSession
	s.Messages = append(s.Messages, msg)

	// Trim old messages if limit exceeded
	if len(s.Messages) > sm.maxMessages {
		s.Messages = s.Messages[len(s.Messages)-sm.maxMessages:]
	}

	s.LastActivity = time.Now()
}

// GetMessages returns all session messages
func (sm *SessionManager) GetMessages() []llm.Message {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Return copy to prevent external modification
	msgs := make([]llm.Message, len(sm.currentSession.Messages))
	copy(msgs, sm.currentSession.Messages)
	return msgs
}

// UpdateTokenUsage updates token statistics
func (sm *SessionManager) UpdateTokenUsage(promptTokens int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.currentSession.TotalPromptTokens += promptTokens
	sm.currentSession.TotalCompletions++
}

// GetContextStats returns context usage statistics
func (sm *SessionManager) GetContextStats() ContextStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	s := sm.currentSession
	return ContextStats{
		MessageCount:      len(s.Messages),
		TotalPromptTokens: s.TotalPromptTokens,
		ContextSent:       s.ContextSent,
		SessionAge:        time.Since(s.LastActivity),
		CompletionCount:   s.TotalCompletions,
	}
}

// ResetSession creates a new session (used after compression)
func (sm *SessionManager) ResetSession() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.currentSession = newSession()
}

// CompressSession prepares session for compression
func (sm *SessionManager) CompressSession(keepLast int) []llm.Message {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s := sm.currentSession
	if len(s.Messages) <= keepLast {
		return nil // Nothing to compress
	}

	// Extract messages to compress
	toCompress := s.Messages[:len(s.Messages)-keepLast]

	// Keep only last N messages
	s.Messages = s.Messages[len(s.Messages)-keepLast:]

	// Mark context as needing refresh (since we compressed history)
	s.ContextSent = false

	return toCompress
}

// ContextStats provides session context statistics
type ContextStats struct {
	MessageCount      int
	TotalPromptTokens int
	ContextSent       bool
	SessionAge        time.Duration
	CompletionCount   int
}

// EstimateContextSize estimates context size in tokens (rough estimate)
func (cs *ContextStats) EstimateContextSize() int {
	// Rough estimate: average 4 chars per token
	// This is a heuristic - actual tokenization varies
	avgTokensPerMessage := 100 // Conservative estimate
	return cs.MessageCount * avgTokensPerMessage
}

// ShouldCompress determines if context should be compressed
func (cs *ContextStats) ShouldCompress(threshold int) bool {
	return cs.MessageCount > threshold
}
