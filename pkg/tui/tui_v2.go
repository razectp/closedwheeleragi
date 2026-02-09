// Package tui provides an enhanced terminal user interface inspired by Claude Code
package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"ClosedWheeler/pkg/agent"
	"ClosedWheeler/pkg/providers"
	"ClosedWheeler/pkg/tools"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MessageQueue handles queued messages with streaming support
type MessageQueue struct {
	messages []QueuedMessage
	mu       sync.RWMutex
}

// QueuedMessage represents a message in the queue
type QueuedMessage struct {
	Role        string
	Content     string
	Streaming   bool
	StreamChunk string
	Timestamp   time.Time
	ToolUse     *ToolExecution
	Thinking    string // Reasoning/thinking content
	Complete    bool
}

// ToolExecution represents a tool being executed
type ToolExecution struct {
	Name      string
	Status    string // "pending", "running", "success", "failed"
	StartTime time.Time
	EndTime   time.Time
	Result    string
	Error     string
}

// NewMessageQueue creates a new message queue
func NewMessageQueue() *MessageQueue {
	return &MessageQueue{
		messages: make([]QueuedMessage, 0),
	}
}

// Add adds a message to the queue
func (mq *MessageQueue) Add(msg QueuedMessage) {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.messages = append(mq.messages, msg)
}

// UpdateLast updates the last message
func (mq *MessageQueue) UpdateLast(update func(*QueuedMessage)) {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	if len(mq.messages) > 0 {
		update(&mq.messages[len(mq.messages)-1])
	}
}

// GetAll returns all messages
func (mq *MessageQueue) GetAll() []QueuedMessage {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	result := make([]QueuedMessage, len(mq.messages))
	copy(result, mq.messages)
	return result
}

// Clear clears all messages
func (mq *MessageQueue) Clear() {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.messages = make([]QueuedMessage, 0)
}

// EnhancedModel represents the enhanced TUI state
type EnhancedModel struct {
	agent             *agent.Agent
	viewport          viewport.Model
	textarea          textarea.Model
	spinner           spinner.Model
	messageQueue      *MessageQueue
	width             int
	height            int
	ready             bool
	processing        bool
	currentTool       *ToolExecution
	activeTools       []ToolExecution
	showTimestamps    bool
	verbose           bool
	status            string
	thinkingAnimation int
	contextStats      agent.ContextStats
	dualSession       *DualSession       // Dual session for agent-to-agent conversations
	providerManager   *providers.ProviderManager // Multi-provider support
	toolRetryWrapper  *tools.IntelligentRetryWrapper // Intelligent tool retry system
	conversationView  *ConversationView  // Live conversation view
	multiWindow       *MultiWindowManager // Multi-window for debate viewing (one per agent)
}

// NewEnhancedModel creates a new enhanced TUI model
func NewEnhancedModel(ag *agent.Agent) EnhancedModel {
	// Text input
	ta := textarea.New()
	ta.Placeholder = "Message ClosedWheelerAGI..."
	ta.Focus()
	ta.CharLimit = 8192
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Base = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor).
		Padding(0, 1)

	// Spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(primaryColor)

	// Initialize message queue with welcome message
	mq := NewMessageQueue()
	mq.Add(QueuedMessage{
		Role:      "system",
		Content:   "🚀 ClosedWheelerAGI v2.1 initialized. Type /help for commands.",
		Timestamp: time.Now(),
		Complete:  true,
	})

	// Initialize provider manager
	providerConfig, _ := providers.LoadProvidersConfig("")
	var pm *providers.ProviderManager
	if providerConfig != nil {
		pm, _ = providers.InitializeFromConfig(providerConfig)
	}
	if pm == nil {
		pm = providers.NewProviderManager()
	}

	// Initialize intelligent retry wrapper
	// Note: The wrapper will be used by TUI commands to show stats
	// The agent will use the original executor
	var retryWrapper *tools.IntelligentRetryWrapper
	if executor := ag.GetToolExecutor(); executor != nil {
		retryWrapper = tools.NewIntelligentRetryWrapper(executor)
		retryWrapper.EnableFeedbackMode(true) // Enable by default

		// TODO: Integrate retry wrapper with agent tool execution
		// This requires modifying how the agent executes tools
	}

	return EnhancedModel{
		agent:            ag,
		textarea:         ta,
		spinner:          sp,
		messageQueue:     mq,
		showTimestamps:   true,
		verbose:          ag.Config().UI.Verbose,
		activeTools:      make([]ToolExecution, 0),
		dualSession:      NewDualSession(ag, ag), // Same agent for now, can be different
		providerManager:  pm,
		toolRetryWrapper: retryWrapper,
		conversationView: NewConversationView(),
		multiWindow:      NewMultiWindowManager(),
	}
}

// Init initializes the enhanced TUI
func (m EnhancedModel) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
		tickAnimation(),
	)
}

// Update handles TUI events
func (m EnhancedModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit

		case tea.KeyEnter:
			if !m.processing {
				return m.sendCurrentMessage()
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate viewport height with correct measurements
		viewportHeight := m.calculateViewportHeight()

		// Calculate YPosition for viewport
		headerH := 1        // Header: 1 line
		statusH := 1        // Status bar: 1 line
		toolsH := m.calculateToolsHeight()  // Active tools: 0 or 1 line
		dividerH := 1       // One divider before viewport

		yPosition := headerH + statusH + toolsH + dividerH

		if !m.ready {
			m.viewport = viewport.New(msg.Width-4, viewportHeight)
			m.viewport.YPosition = yPosition
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 4
			m.viewport.Height = viewportHeight
			m.viewport.YPosition = yPosition
		}
		m.textarea.SetWidth(msg.Width - 8)
		m.updateViewport()

	case responseCompleteMsg:
		m.processing = false
		m.status = ""
		m.messageQueue.UpdateLast(func(qm *QueuedMessage) {
			qm.Complete = true
			qm.Streaming = false
			if msg.err != nil {
				qm.Role = "error"
				qm.Content = "❌ " + msg.err.Error()
			} else {
				qm.Content = msg.content
			}
		})
		m.updateViewport()
		// Re-focus textarea for new input
		m.textarea.Focus()

	case streamChunkMsg:
		m.messageQueue.UpdateLast(func(qm *QueuedMessage) {
			qm.StreamChunk += msg.chunk
			qm.Content = qm.StreamChunk
		})
		m.updateViewport()
		return m, waitForStream()

	case toolStartMsg:
		tool := ToolExecution{
			Name:      msg.toolName,
			Status:    "running",
			StartTime: time.Now(),
		}
		m.currentTool = &tool
		m.activeTools = append(m.activeTools, tool)
		m.status = fmt.Sprintf("🔧 %s", msg.toolName)
		return m, nil

	case toolCompleteMsg:
		if m.currentTool != nil {
			m.currentTool.Status = "success"
			m.currentTool.EndTime = time.Now()
			m.currentTool.Result = msg.result

			// Update in activeTools
			for i := range m.activeTools {
				if m.activeTools[i].Name == m.currentTool.Name &&
				   m.activeTools[i].StartTime == m.currentTool.StartTime {
					m.activeTools[i] = *m.currentTool
					break
				}
			}
		}
		m.currentTool = nil
		return m, nil

	case toolErrorMsg:
		if m.currentTool != nil {
			m.currentTool.Status = "failed"
			m.currentTool.EndTime = time.Now()
			m.currentTool.Error = msg.err.Error()

			// Update in activeTools
			for i := range m.activeTools {
				if m.activeTools[i].Name == m.currentTool.Name &&
				   m.activeTools[i].StartTime == m.currentTool.StartTime {
					m.activeTools[i] = *m.currentTool
					break
				}
			}
		}
		m.currentTool = nil
		return m, nil

	case statusUpdateMsg:
		m.status = msg.status
		return m, nil

	case thinkingMsg:
		m.messageQueue.UpdateLast(func(qm *QueuedMessage) {
			qm.Thinking = msg.content
		})
		m.updateViewport()
		return m, nil

	case animationTickMsg:
		m.thinkingAnimation = (m.thinkingAnimation + 1) % 4
		m.updateViewport()
		return m, tickAnimation()

	case pollConversationMsg:
		// Check for new conversation messages
		return m, m.handleConversationUpdate()

	case spinner.TickMsg:
		if m.processing {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Update context stats
	m.contextStats = m.agent.GetContextStats()

	// Update textarea
	if !m.processing {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update viewport
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the enhanced TUI
func (m EnhancedModel) View() string {
	if !m.ready {
		return m.spinner.View() + " Initializing ClosedWheelerAGI..."
	}

	var sections []string

	// Header
	sections = append(sections, m.renderHeader())

	// Status bar
	sections = append(sections, m.renderStatusBar())

	// Active tools section
	if len(m.activeTools) > 0 {
		sections = append(sections, m.renderActiveTools())
	}

	// Divider
	sections = append(sections, m.renderDivider())

	// Messages viewport
	sections = append(sections, m.viewport.View())

	// Divider
	sections = append(sections, m.renderDivider())

	// Input or processing area
	if m.processing {
		sections = append(sections, m.renderProcessingArea())
	} else {
		sections = append(sections, m.textarea.View())
	}

	// Help bar
	sections = append(sections, m.renderHelpBar())

	return strings.Join(sections, "\n")
}

// renderHeader renders the header
func (m EnhancedModel) renderHeader() string {
	title := "ClosedWheelerAGI"
	version := "v2.1"

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(textPrimary).
		Background(primaryColor).
		Padding(0, 2)

	versionStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Background(bgDark).
		Padding(0, 1)

	left := lipgloss.JoinHorizontal(lipgloss.Center,
		titleStyle.Render("🤖 "+title),
		versionStyle.Render(version),
	)

	// Model info
	modelInfo := m.agent.Config().Model
	modelStyle := lipgloss.NewStyle().
		Foreground(secondaryColor).
		Background(bgDark).
		Padding(0, 1)

	right := modelStyle.Render("🧠 " + modelInfo)

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 4
	if gap < 0 {
		// Not enough space - truncate model info
		availableWidth := m.width - lipgloss.Width(left) - 8
		if availableWidth < 10 {
			// Too small, drop model info
			right = ""
			gap = m.width - lipgloss.Width(left) - 2
		} else {
			modelInfo = truncateText(modelInfo, availableWidth-6)  // 6 for "🧠 " + padding
			right = modelStyle.Render("🧠 " + modelInfo)
			gap = m.width - lipgloss.Width(left) - lipgloss.Width(right) - 4
		}
		if gap < 0 {
			gap = 0
		}
	}

	header := lipgloss.NewStyle().
		Background(bgDark).
		Width(m.width).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", gap), right))

	return header
}

// renderStatusBar renders an enhanced status bar
func (m EnhancedModel) renderStatusBar() string {
	// Status badge
	var badge string
	var badgeStyle lipgloss.Style

	if m.processing {
		if m.currentTool != nil {
			badge = " TOOL "
			badgeStyle = workingBadgeStyle
		} else {
			badge = " THINKING "
			badgeStyle = thinkingBadgeStyle
		}
	} else {
		badge = " READY "
		badgeStyle = idleBadgeStyle
	}

	// Context indicator with detailed info
	var contextIndicator string
	if m.contextStats.ContextSent {
		contextIndicator = lipgloss.NewStyle().Foreground(successColor).Render("●")
	} else {
		contextIndicator = lipgloss.NewStyle().Foreground(accentColor).Render("○")
	}

	// Memory stats
	stats := m.agent.GetMemoryStats()
	memInfo := fmt.Sprintf(" %s STM:%d WM:%d LTM:%d",
		contextIndicator,
		stats["short_term"],
		stats["working"],
		stats["long_term"])

	// Context info with percentage
	contextInfo := fmt.Sprintf("CTX:%d", m.contextStats.MessageCount)
	var contextPercent string
	if m.contextStats.MessageCount > 15 {
		percent := int((float64(m.contextStats.MessageCount) / 50.0) * 100)
		if percent > 100 {
			percent = 100
		}
		contextPercent = fmt.Sprintf("(%d%%)", percent)
		contextInfo = lipgloss.NewStyle().Foreground(accentColor).Render(contextInfo + contextPercent)
	} else {
		contextInfo = lipgloss.NewStyle().Foreground(textSecondary).Render(contextInfo)
	}

	// Usage stats with prompt/completion breakdown
	usage := m.agent.GetUsageStats()
	promptTokens := usage["prompt_tokens"]
	completionTokens := usage["completion_tokens"]
	totalTokens := usage["total_tokens"]

	tokensInfo := fmt.Sprintf("TOK:%v", totalTokens)
	if promptTokens != nil && completionTokens != nil {
		tokensInfo = fmt.Sprintf("TOK:%v(↑%v/↓%v)", totalTokens, promptTokens, completionTokens)
	}

	// Session stats
	sessionInfo := ""
	if m.contextStats.CompletionCount > 0 {
		sessionInfo = fmt.Sprintf(" │ CMPL:%d", m.contextStats.CompletionCount)
	}

	left := badgeStyle.Render(badge)
	middle := memStatsStyle.Render(memInfo + " │ " + contextInfo + " │ " + tokensInfo + sessionInfo)

	// Status message
	right := ""
	if m.status != "" {
		right = lipgloss.NewStyle().
			Foreground(accentColor).
			Italic(true).
			Render(m.status)
	}

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(middle) - lipgloss.Width(right) - 6
	if gap < 0 {
		// Not enough space - prioritize: left (status) > middle (stats) > right (message)
		availableWidth := m.width - lipgloss.Width(left) - 8

		if availableWidth < 20 {
			// Too small - drop right and truncate middle severely
			right = ""
			middle = truncateText(middle, availableWidth)
			gap = 0
		} else {
			// Truncate components intelligently
			rightWidth := lipgloss.Width(right)
			middleMaxWidth := availableWidth - rightWidth - 4

			if middleMaxWidth < 30 {
				// Not enough for both - drop right
				right = ""
				middle = truncateText(middle, availableWidth)
			} else {
				// Both fit with truncation
				middle = truncateText(middle, middleMaxWidth)
			}
			gap = 0
		}
	}

	statusBar := statusBarStyle.Width(m.width - 2).Render(
		lipgloss.JoinHorizontal(lipgloss.Top,
			left,
			" ",
			middle,
			strings.Repeat(" ", gap),
			right,
		),
	)

	return statusBar
}

// renderActiveTools renders the active tools section
func (m EnhancedModel) renderActiveTools() string {
	if len(m.activeTools) == 0 {
		return ""
	}

	var toolItems []string

	// Show last 3 tools
	start := len(m.activeTools) - 3
	if start < 0 {
		start = 0
	}

	for i := start; i < len(m.activeTools); i++ {
		tool := m.activeTools[i]

		var icon string
		var statusStyle lipgloss.Style

		switch tool.Status {
		case "running":
			icon = m.spinner.View()
			statusStyle = lipgloss.NewStyle().Foreground(accentColor)
		case "success":
			icon = "✓"
			statusStyle = lipgloss.NewStyle().Foreground(successColor)
		case "failed":
			icon = "✗"
			statusStyle = lipgloss.NewStyle().Foreground(errorColor)
		default:
			icon = "○"
			statusStyle = lipgloss.NewStyle().Foreground(mutedColor)
		}

		duration := ""
		if !tool.EndTime.IsZero() {
			duration = fmt.Sprintf(" (%s)", tool.EndTime.Sub(tool.StartTime).Round(time.Millisecond))
		}

		durationStyled := lipgloss.NewStyle().Foreground(mutedColor).Render(duration)

		toolItem := fmt.Sprintf("%s %s%s",
			statusStyle.Render(icon),
			tool.Name,
			durationStyled)

		toolItems = append(toolItems, toolItem)
	}

	toolsSection := lipgloss.NewStyle().
		Foreground(textSecondary).
		Background(bgDarker).
		Padding(0, 1).
		Width(m.width - 2).
		Render("🔧 " + strings.Join(toolItems, " │ "))

	return toolsSection
}

// renderProcessingArea renders the processing/thinking area
func (m EnhancedModel) renderProcessingArea() string {
	dots := strings.Repeat(".", m.thinkingAnimation)

	var statusText string
	if m.currentTool != nil {
		statusText = fmt.Sprintf("%s Executing %s%s",
			m.spinner.View(),
			m.currentTool.Name,
			dots)
	} else {
		statusText = fmt.Sprintf("%s Thinking%s",
			m.spinner.View(),
			dots)
	}

	// Show streaming preview if available
	messages := m.messageQueue.GetAll()
	if len(messages) > 0 {
		lastMsg := messages[len(messages)-1]
		if lastMsg.Streaming && len(lastMsg.StreamChunk) > 0 {
			preview := lastMsg.StreamChunk
			if len(preview) > 60 {
				preview = "..." + preview[len(preview)-57:]
			}
			statusText = fmt.Sprintf("%s %s",
				m.spinner.View(),
				preview)
		}
	}

	processingStyle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(0, 1).
		Width(m.width - 8)

	return processingStyle.Render(statusText)
}

// renderHelpBar renders the help bar
func (m EnhancedModel) renderHelpBar() string {
	helpText := "↵ Send │ /help Commands │ ^C Quit"

	if m.processing {
		helpText = "⏳ Processing... │ Please wait"
	}

	return helpStyle.Render(helpText)
}

// renderDivider renders a divider
func (m EnhancedModel) renderDivider() string {
	return dividerStyle.Render(strings.Repeat("─", m.width-2))
}

// calculateToolsHeight calculates the height needed for tools section
func (m EnhancedModel) calculateToolsHeight() int {
	if len(m.activeTools) > 0 {
		return 1
	}
	return 0
}

// calculateViewportHeight calculates the correct viewport height based on fixed components
func (m *EnhancedModel) calculateViewportHeight() int {
	// Fixed components with actual measurements
	headerH := 1        // Header line
	statusH := 1        // Status bar line
	toolsH := m.calculateToolsHeight()  // 0 or 1 line
	dividers := 2       // 2 dividers (before and after viewport)
	inputH := 4         // Input area: 3 lines textarea + 1 line padding
	helpH := 1          // Help bar line

	fixedHeight := headerH + statusH + toolsH + dividers + inputH + helpH

	viewportHeight := m.height - fixedHeight

	// Ensure minimum height
	if viewportHeight < 5 {
		viewportHeight = 5
	}

	// Ensure we don't exceed available space
	maxHeight := m.height - 10  // Reserve 10 lines for UI
	if maxHeight < 5 {
		maxHeight = 5
	}
	if viewportHeight > maxHeight {
		viewportHeight = maxHeight
	}

	return viewportHeight
}

// truncateText truncates text to fit within maxWidth, adding ellipsis if needed
func truncateText(text string, maxWidth int) string {
	if maxWidth < 4 {
		return "..."
	}

	width := lipgloss.Width(text)
	if width <= maxWidth {
		return text
	}

	// Binary search for optimal truncation point
	runes := []rune(text)
	left, right := 0, len(runes)

	for left < right {
		mid := (left + right + 1) / 2
		candidate := string(runes[:mid]) + "..."
		if lipgloss.Width(candidate) <= maxWidth {
			left = mid
		} else {
			right = mid - 1
		}
	}

	if left == 0 {
		return "..."
	}

	return string(runes[:left]) + "..."
}

// updateViewport updates the viewport content
func (m *EnhancedModel) updateViewport() {
	wasAtBottom := m.viewport.AtBottom()

	var sb strings.Builder

	messages := m.messageQueue.GetAll()
	for i, msg := range messages {
		// Timestamp
		timestamp := ""
		if m.showTimestamps && !msg.Timestamp.IsZero() {
			timestamp = lipgloss.NewStyle().
				Foreground(mutedColor).
				Faint(true).
				Render(msg.Timestamp.Format("15:04") + " ")
		}

		switch msg.Role {
		case "user":
			sb.WriteString(timestamp)
			sb.WriteString(userLabelStyle.Render("You"))
			sb.WriteString("\n")
			sb.WriteString(userBubbleStyle.Render(msg.Content))

		case "assistant":
			sb.WriteString(timestamp)
			sb.WriteString(assistantLabelStyle.Render("🤖 Assistant"))

			// Show streaming cursor
			if msg.Streaming && !msg.Complete {
				sb.WriteString(lipgloss.NewStyle().Foreground(primaryColor).Render(" ▌"))
			}

			sb.WriteString("\n")

			// Show thinking if verbose
			if msg.Thinking != "" && m.verbose {
				sb.WriteString(thinkingHeaderStyle.Render("💭 Reasoning:"))
				sb.WriteString("\n")
				sb.WriteString(thinkingStyle.Render(msg.Thinking))
				sb.WriteString("\n")
				sb.WriteString(dividerStyle.Render("─────────────"))
				sb.WriteString("\n")
			}

			// Render content
			sb.WriteString(m.renderContent(msg.Content))

		case "system":
			sb.WriteString(systemMsgStyle.Render("ℹ " + msg.Content))

		case "error":
			sb.WriteString(errorMsgStyle.Render(msg.Content))
		}

		// Add spacing between messages
		if i < len(messages)-1 {
			sb.WriteString("\n\n")
		}
	}

	m.viewport.SetContent(sb.String())

	if wasAtBottom {
		m.viewport.GotoBottom()
	}
}

// sendCurrentMessage sends the current input
func (m EnhancedModel) sendCurrentMessage() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.textarea.Value())
	if input == "" {
		return m, nil
	}

	// Handle commands
	if strings.HasPrefix(input, "/") {
		return m.handleCommand(input)
	}

	// Add user message
	m.messageQueue.Add(QueuedMessage{
		Role:      "user",
		Content:   input,
		Timestamp: time.Now(),
		Complete:  true,
	})

	// Add assistant placeholder
	m.messageQueue.Add(QueuedMessage{
		Role:      "assistant",
		Content:   "",
		Streaming: true,
		Timestamp: time.Now(),
		Complete:  false,
	})

	m.textarea.Reset()
	m.processing = true
	m.status = "Processing request..."
	m.activeTools = []ToolExecution{} // Clear old tools
	m.updateViewport()

	return m, tea.Batch(
		m.sendMessage(input),
		m.spinner.Tick,
	)
}

// renderContent renders content with code block highlighting
func (m *EnhancedModel) renderContent(content string) string {
	var result strings.Builder
	inCodeBlock := false

	lines := strings.Split(content, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Code block detection
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			if inCodeBlock {
				lang := strings.TrimPrefix(trimmed, "```")
				if lang == "" {
					lang = "code"
				}
				result.WriteString(dividerStyle.Render(fmt.Sprintf("─── %s ───", lang)))
			} else {
				result.WriteString(dividerStyle.Render("────────────"))
			}
			result.WriteString("\n")
			continue
		}

		// Render line
		if inCodeBlock {
			result.WriteString(codeBlockStyle.Render(line))
		} else {
			result.WriteString(assistantTextStyle.Render(line))
		}

		// Add newline except for last empty line
		if i < len(lines)-1 || line != "" {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// sendMessage sends a message to the agent
func (m EnhancedModel) sendMessage(input string) tea.Cmd {
	return func() tea.Msg {
		response, err := m.agent.Chat(input)
		return responseCompleteMsg{content: response, err: err}
	}
}

// handleCommand handles slash commands (reuse from original)
func (m EnhancedModel) handleCommand(input string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return m, nil
	}

	cmdName := strings.ToLower(strings.TrimPrefix(parts[0], "/"))
	args := parts[1:]

	m.textarea.Reset()

	// Find command by name or alias
	foundCmd := FindCommand(cmdName)

	if foundCmd == nil {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   fmt.Sprintf("❌ Unknown command: /%s\n\nType `/help` for available commands.", cmdName),
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return m, nil
	}

	// Execute command
	return foundCmd.Handler(&m, args)
}

// Message types
type responseCompleteMsg struct {
	content string
	err     error
}

type toolStartMsg struct {
	toolName string
}

type toolCompleteMsg struct {
	result string
}

type toolErrorMsg struct {
	err error
}

type thinkingMsg struct {
	content string
}

type animationTickMsg struct{}

// Helper commands
func waitForStream() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return streamChunkMsg{}
	})
}

func tickAnimation() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return animationTickMsg{}
	})
}


// RunEnhanced starts the enhanced TUI
func RunEnhanced(ag *agent.Agent) error {
	p := tea.NewProgram(
		NewEnhancedModel(ag),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Set status callback
	ag.SetStatusCallback(func(s string) {
		p.Send(statusUpdateMsg{status: s})
	})

	_, err := p.Run()
	return err
}
