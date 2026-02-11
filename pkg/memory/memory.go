// Package memory implements a tiered memory system for context management.
// It provides short-term, working, and long-term memory tiers to efficiently
// manage context for LLM interactions.
package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// MemoryTier represents the type of memory storage
type MemoryTier int

const (
	ShortTerm  MemoryTier = iota // Recent conversation messages
	WorkingMem                   // Currently relevant files/functions
	LongTerm                     // Compressed summaries and decisions
)

// MemoryItem represents a single item in memory
type MemoryItem struct {
	ID         string     `json:"id"`
	Tier       MemoryTier `json:"tier"`
	Type       string     `json:"type"` // "message", "file", "function", "decision", "summary"
	Content    string     `json:"content"`
	Metadata   Metadata   `json:"metadata"`
	Relevance  float64    `json:"relevance"` // 0.0 - 1.0
	CreatedAt  time.Time  `json:"created_at"`
	AccessedAt time.Time  `json:"accessed_at"`
}

// Metadata holds additional information about a memory item
type Metadata struct {
	FilePath     string   `json:"file_path,omitempty"`
	FunctionName string   `json:"function_name,omitempty"`
	LineStart    int      `json:"line_start,omitempty"`
	LineEnd      int      `json:"line_end,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Role         string   `json:"role,omitempty"` // "user", "assistant", "system"
}

// Manager handles all memory operations
type Manager struct {
	shortTerm []*MemoryItem
	working   map[string]*MemoryItem // keyed by file path or identifier
	longTerm  []*MemoryItem

	config      *Config
	storagePath string
	mu          sync.RWMutex
}

// Config holds memory configuration
type Config struct {
	MaxShortTermItems  int `json:"max_short_term_items"`
	MaxWorkingItems    int `json:"max_working_items"`
	MaxLongTermItems   int `json:"max_long_term_items"`
	MaxContextTokens   int `json:"max_context_tokens"`
	CompressionTrigger int `json:"compression_trigger"` // Compress when short-term exceeds this
}

// DefaultConfig returns sensible defaults
func DefaultConfig() *Config {
	return &Config{
		MaxShortTermItems:  20,
		MaxWorkingItems:    50,
		MaxLongTermItems:   100,
		MaxContextTokens:   8000,
		CompressionTrigger: 15,
	}
}

// NewManager creates a new memory manager
func NewManager(storagePath string, config *Config) *Manager {
	if config == nil {
		config = DefaultConfig()
	}

	return &Manager{
		shortTerm:   make([]*MemoryItem, 0),
		working:     make(map[string]*MemoryItem),
		longTerm:    make([]*MemoryItem, 0),
		config:      config,
		storagePath: storagePath,
	}
}

// AddMessage adds a conversation message to short-term memory
func (m *Manager) AddMessage(role, content string) *MemoryItem {
	m.mu.Lock()
	defer m.mu.Unlock()

	item := &MemoryItem{
		ID:         generateID(),
		Tier:       ShortTerm,
		Type:       "message",
		Content:    content,
		Metadata:   Metadata{Role: role},
		Relevance:  1.0,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
	}

	m.shortTerm = append(m.shortTerm, item)

	// Check if we need to compress
	if len(m.shortTerm) > m.config.CompressionTrigger {
		m.triggerCompression()
	}

	return item
}

// AddFile adds a file to working memory
func (m *Manager) AddFile(filePath, content string, relevance float64) *MemoryItem {
	m.mu.Lock()
	defer m.mu.Unlock()

	item := &MemoryItem{
		ID:         generateID(),
		Tier:       WorkingMem,
		Type:       "file",
		Content:    content,
		Metadata:   Metadata{FilePath: filePath},
		Relevance:  relevance,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
	}

	m.working[filePath] = item

	// Evict if over limit
	if len(m.working) > m.config.MaxWorkingItems {
		m.evictLeastRelevant()
	}

	return item
}

// AddFunction adds a function to working memory
func (m *Manager) AddFunction(filePath, funcName, content string, lineStart, lineEnd int) *MemoryItem {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s::%s", filePath, funcName)

	item := &MemoryItem{
		ID:      generateID(),
		Tier:    WorkingMem,
		Type:    "function",
		Content: content,
		Metadata: Metadata{
			FilePath:     filePath,
			FunctionName: funcName,
			LineStart:    lineStart,
			LineEnd:      lineEnd,
		},
		Relevance:  0.8,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
	}

	m.working[key] = item
	return item
}

// AddDecision adds an important decision to long-term memory
func (m *Manager) AddDecision(decision string, tags []string) *MemoryItem {
	m.mu.Lock()
	defer m.mu.Unlock()

	item := &MemoryItem{
		ID:         generateID(),
		Tier:       LongTerm,
		Type:       "decision",
		Content:    decision,
		Metadata:   Metadata{Tags: tags},
		Relevance:  1.0,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
	}

	m.longTerm = append(m.longTerm, item)
	return item
}

// AddSummary adds a compressed summary to long-term memory
func (m *Manager) AddSummary(summary string) *MemoryItem {
	m.mu.Lock()
	defer m.mu.Unlock()

	item := &MemoryItem{
		ID:         generateID(),
		Tier:       LongTerm,
		Type:       "summary",
		Content:    summary,
		Relevance:  0.9,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
	}

	m.longTerm = append(m.longTerm, item)
	return item
}

// GetMessages returns short-term memory as LLM messages
func (m *Manager) GetMessages() []map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	messages := make([]map[string]string, 0, len(m.shortTerm))
	for _, item := range m.shortTerm {
		messages = append(messages, map[string]string{
			"role":    item.Metadata.Role,
			"content": item.Content,
		})
	}
	return messages
}

// TrimOldest removes the oldest fraction of short-term messages to reduce context size.
// fraction should be between 0.0 and 1.0 (e.g. 0.3 = drop oldest 30%).
// System messages (role == "system") are always preserved.
func (m *Manager) TrimOldest(fraction float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.shortTerm) == 0 {
		return
	}

	// Separate system messages (keep) from conversation messages (trim)
	var sysItems []*MemoryItem
	var convItems []*MemoryItem
	for _, item := range m.shortTerm {
		if item.Metadata.Role == "system" {
			sysItems = append(sysItems, item)
		} else {
			convItems = append(convItems, item)
		}
	}

	dropCount := int(float64(len(convItems)) * fraction)
	if dropCount < 1 {
		dropCount = 1
	}
	if dropCount >= len(convItems) {
		dropCount = len(convItems) - 1
	}

	// Drop oldest conversation messages
	if dropCount > 0 {
		convItems = convItems[dropCount:]
	}

	m.shortTerm = append(sysItems, convItems...)
}

// UpdateRelevance updates the relevance score of an item and refreshes its timestamp
func (m *Manager) UpdateRelevance(id string, relevance float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, item := range m.working {
		if item.ID == id {
			item.Relevance = relevance
			item.AccessedAt = time.Now()
			return
		}
	}
}

// AgeWorkingMemory reduces relevance of items over time and removes expired ones
func (m *Manager) AgeWorkingMemory(decayFactor float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, item := range m.working {
		// Decay relevance based on time since last access
		hoursSinceAccess := time.Since(item.AccessedAt).Hours()
		item.Relevance *= (1.0 - (decayFactor * hoursSinceAccess))

		if item.Relevance < 0.1 {
			delete(m.working, key)
		}
	}
}

// Clear clears a specific memory tier
func (m *Manager) Clear(tier MemoryTier) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch tier {
	case ShortTerm:
		m.shortTerm = make([]*MemoryItem, 0)
	case WorkingMem:
		m.working = make(map[string]*MemoryItem)
	case LongTerm:
		m.longTerm = make([]*MemoryItem, 0)
	}
}

// Save persists long-term memory to disk
func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.storagePath == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(m.storagePath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m.longTerm, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.storagePath, data, 0644)
}

// Load loads long-term memory from disk
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.storagePath == "" {
		return nil
	}

	data, err := os.ReadFile(m.storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &m.longTerm)
}

// GetItemsToCompress returns items that should be compressed
func (m *Manager) GetItemsToCompress() []*MemoryItem {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.shortTerm) < m.config.CompressionTrigger {
		return nil
	}

	// Return older items (leave the last 5 for immediate context)
	keep := 5
	if len(m.shortTerm) <= keep {
		return nil
	}

	return m.shortTerm[:len(m.shortTerm)-keep]
}

// CompressItems removes old items and adds summary
func (m *Manager) CompressItems(summary string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	keep := 5
	if len(m.shortTerm) <= keep {
		return
	}

	m.shortTerm = m.shortTerm[len(m.shortTerm)-keep:]

	// Add summary to long-term
	m.longTerm = append(m.longTerm, &MemoryItem{
		ID:         generateID(),
		Tier:       LongTerm,
		Type:       "summary",
		Content:    summary,
		Relevance:  0.9,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
	})

	// Trim long-term if it's too big
	if len(m.longTerm) > m.config.MaxLongTermItems {
		m.longTerm = m.longTerm[1:] // Simple FIFO for now
	}
}

// GetContext builds a refined context focusing on highly relevant items
func (m *Manager) GetContext() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sb strings.Builder

	// 1. Long-Term Memory (Summaries of past interactions)
	if len(m.longTerm) > 0 {
		sb.WriteString("### 🧠 Long-Term Memory (Lessons & Summaries)\n")
		// Show last 3 summaries/decisions to keep it focused
		start := len(m.longTerm) - 3
		if start < 0 {
			start = 0
		}
		for _, item := range m.longTerm[start:] {
			prefix := "• "
			if item.Type == "decision" {
				prefix = "📍 [Decision] "
			}
			sb.WriteString(fmt.Sprintf("%s%s\n", prefix, item.Content))
		}
		sb.WriteString("\n")
	}

	// 2. Working Memory (Active Code Context)
	if len(m.working) > 0 {
		sb.WriteString("### 🛠️ Working Context (Files Analyzed)\n")
		// Only show items with relevance > 0.5
		for _, item := range m.working {
			if item.Relevance < 0.5 {
				continue
			}

			if item.Type == "file" {
				sb.WriteString(fmt.Sprintf("- File: `%s` (Relevance: %.2f)\n",
					item.Metadata.FilePath, item.Relevance))
			} else if item.Type == "function" {
				sb.WriteString(fmt.Sprintf("- Function: `%s` in `%s`\n",
					item.Metadata.FunctionName, item.Metadata.FilePath))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// Stats returns memory usage statistics
func (m *Manager) Stats() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]int{
		"short_term": len(m.shortTerm),
		"working":    len(m.working),
		"long_term":  len(m.longTerm),
	}
}

// triggerCompression is now handled by the Agent periodically
func (m *Manager) triggerCompression() {
	// Signal-only for now, logic lives in Agent.Chat
}

// evictLeastRelevant removes the least relevant item from working memory
func (m *Manager) evictLeastRelevant() {
	var minKey string
	var minRelevance float64 = 2.0

	for key, item := range m.working {
		if item.Relevance < minRelevance {
			minRelevance = item.Relevance
			minKey = key
		}
	}

	if minKey != "" {
		delete(m.working, minKey)
	}
}

// Helper functions

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
