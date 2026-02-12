// Package tui provides an enhanced terminal user interface inspired by Claude Code
package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ClosedWheeler/pkg/agent"
	"ClosedWheeler/pkg/providers"
	"ClosedWheeler/pkg/tools"
	"ClosedWheeler/pkg/utils"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
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
	Stats       *MessageStats // token/elapsed stats for completed assistant messages
}

// ToolExecution represents a tool being executed
type ToolExecution struct {
	Name      string
	Args      string
	Status    string // "pending", "running", "success", "failed"
	StartTime time.Time
	EndTime   time.Time
	Result    string
	Error     string
}

// MessageStats holds per-response usage stats displayed after assistant messages.
type MessageStats struct {
	PromptTokens     int
	CompletionTokens int
	Elapsed          time.Duration
}

// maxQueueMessages is the maximum number of messages kept in the queue.
// Older messages are pruned to prevent unbounded memory growth.
const maxQueueMessages = 200

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

// Len returns the current number of messages in the queue.
func (mq *MessageQueue) Len() int {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	return len(mq.messages)
}

// Prune trims the queue to keep only the most recent maxMessages entries.
func (mq *MessageQueue) Prune(maxMessages int) {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	if len(mq.messages) > maxMessages {
		mq.messages = mq.messages[len(mq.messages)-maxMessages:]
	}
}

// ClearStreamChunks zeroes the StreamChunk field on all completed messages
// to reclaim the duplicate memory held alongside Content.
func (mq *MessageQueue) ClearStreamChunks() {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	for i := range mq.messages {
		if mq.messages[i].Complete {
			mq.messages[i].StreamChunk = ""
		}
	}
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
	dualSession       *DualSession                   // Dual session for agent-to-agent conversations
	providerManager   *providers.ProviderManager     // Multi-provider support
	toolRetryWrapper  *tools.IntelligentRetryWrapper // Intelligent tool retry system
	conversationView *ConversationView // Live conversation view

	// Model picker state (from tui.go)
	pickerActive   bool
	pickerStep     int
	pickerCursor   int
	pickerSelected ProviderOption
	pickerInput    textinput.Model
	pickerNewKey   string
	pickerNewURL   string
	pickerModelID  string

	// Help menu overlay state
	helpActive         bool
	helpCategoryCursor int
	helpCommandCursor  int
	helpSearchMode     bool
	helpSearchInput    textinput.Model
	helpSearchResults  []helpFlatCommand

	// Panel overlay state (read-only info panels)
	panelActive    bool
	panelTitle     string
	panelLines     []string
	panelScroll    int
	panelMaxScroll int

	// Settings overlay state (interactive toggle menu)
	settingsActive bool
	settingsCursor int
	settingsItems  []SettingsItem

	// Pipeline status map for multi-agent workflows
	pipelineStatus map[agent.AgentRole]string // "thinking", "done", "error", ""

	// Request timing and before-usage snapshot (for per-response stats)
	requestStartTime   time.Time
	requestBeforeUsage map[string]any

	// Input queue: messages submitted while agent is processing
	inputQueue []string

	// Scroll tracking: auto-scroll stays on unless user scrolls up
	userScrolledAway bool
	// Stream rendering throttle: limits viewport rebuilds to once per 50ms
	lastStreamRender time.Time

	// Debate wizard overlay state
	debateWizActive  bool
	debateWizStep    int
	debateWizTopic   textinput.Model // topic input
	debateWizRoleA   int             // preset index for Agent A
	debateWizRoleB   int             // preset index for Agent B
	debateWizCustomA textinput.Model // custom prompt for A (when Custom selected)
	debateWizCustomB textinput.Model // custom prompt for B (when Custom selected)
	debateWizTurns   textinput.Model // turns number
	debateWizModelA      int      // cursor index for model A selection
	debateWizModelB      int      // cursor index for model B selection
	debateWizModels      []string // available model names (built lazily)
	debateWizRulesCursor int      // cursor index for session rules (tool permission mode)

	// Debate viewer overlay state (lipgloss-based in-TUI viewer)
	debateViewActive     bool // overlay is visible
	debateViewScroll     int  // current scroll position
	debateViewMaxScroll  int  // max scroll position
	debateViewAutoScroll bool // auto-scroll to bottom on new messages
	debateViewLastCount  int  // last seen message count (for detecting new messages)
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
		BorderForeground(PrimaryColor).
		Padding(0, 1)

	// Spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(PrimaryColor)

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
		dualSession:      NewDualSession(ag.CloneForDebate("Agent A"), ag.CloneForDebate("Agent B"), ag.GetLogger()),
		providerManager:  pm,
		toolRetryWrapper: retryWrapper,
		conversationView: NewConversationView(),
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
		// Intercept keys when picker is active
		if m.pickerActive {
			newM, cmd := m.enhancedPickerUpdate(msg)
			return newM, cmd
		}

		// Intercept keys when help menu is active
		if m.helpActive {
			newM, cmd := m.helpMenuUpdate(msg)
			return newM, cmd
		}

		// Intercept keys when panel overlay is active
		if m.panelActive {
			newM, cmd := m.panelUpdate(msg)
			return newM, cmd
		}

		// Intercept keys when settings overlay is active
		if m.settingsActive {
			newM, cmd := m.settingsUpdate(msg)
			return newM, cmd
		}

		// Intercept keys when debate wizard is active
		if m.debateWizActive {
			newM, cmd := m.debateWizardUpdate(msg)
			return newM, cmd
		}

		// Intercept keys when debate viewer overlay is active
		if m.debateViewActive {
			newM, cmd := m.debateViewerUpdate(msg)
			return newM, cmd
		}

		switch msg.Type {
		case tea.KeyCtrlC:
			if m.processing {
				// First Ctrl+C while processing: cancel the current request, don't quit.
				// The user can press Ctrl+C again (or a third time via quitKeyFilter) to exit.
				m.agent.StopCurrentRequest()
				return m, nil
			}
			return m, tea.Quit

		case tea.KeyEsc:
			if m.processing {
				m.agent.StopCurrentRequest()
				return m, nil
			}

		case tea.KeyEnter:
			if m.processing {
				// Queue the input for later processing
				queued := strings.TrimSpace(m.textarea.Value())
				if queued != "" {
					m.inputQueue = append(m.inputQueue, queued)
					m.textarea.Reset()
					m.messageQueue.Add(QueuedMessage{
						Role:      "system",
						Content:   fmt.Sprintf("📋 Queued: %s", queued),
						Timestamp: time.Now(),
						Complete:  true,
					})
					m.updateViewport()
				}
				return m, nil
			}
			return m.sendCurrentMessage()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.recalculateLayout()
		m.updateViewport()

		// Recalculate debate viewer scroll bounds on resize
		if m.debateViewActive {
			m.recalcDebateViewScroll()
		}

		// Recalculate panel scroll bounds on resize
		if m.panelActive {
			visibleH := m.panelVisibleHeight()
			m.panelMaxScroll = len(m.panelLines) - visibleH
			if m.panelMaxScroll < 0 {
				m.panelMaxScroll = 0
			}
			if m.panelScroll > m.panelMaxScroll {
				m.panelScroll = m.panelMaxScroll
			}
		}

	case responseCompleteMsg:
		m.processing = false
		m.status = ""
		m.pipelineStatus = nil            // reset pipeline indicators
		m.activeTools = []ToolExecution{} // clear stale tools
		m.currentTool = nil
		m.userScrolledAway = false         // resume auto-scroll for new content
		m.lastStreamRender = time.Time{}   // force full render on next stream
		m.messageQueue.UpdateLast(func(qm *QueuedMessage) {
			qm.Complete = true
			qm.Streaming = false
			if msg.err != nil {
				// Distinguish user-initiated stop from actual errors
				if isCancelledError(msg.err) {
					qm.Role = "system"
					qm.Content = "Request stopped."
				} else {
					qm.Role = "error"
					qm.Content = "Error: " + msg.err.Error()
				}
			} else {
				qm.Content = msg.content
				if msg.deltaPrompt > 0 || msg.deltaCompletion > 0 {
					qm.Stats = &MessageStats{
						PromptTokens:     msg.deltaPrompt,
						CompletionTokens: msg.deltaCompletion,
						Elapsed:          msg.elapsed,
					}
				}
			}
		})
		// Reclaim duplicate stream memory and prune old messages
		m.messageQueue.ClearStreamChunks()
		m.messageQueue.Prune(maxQueueMessages)
		// Recalculate layout: tools cleared + processing area gone
		if m.ready {
			m.recalculateLayout()
		}
		m.updateViewport()
		// Re-focus textarea for new input
		m.textarea.Focus()

		// Drain input queue: if there are queued messages, send the next one
		if len(m.inputQueue) > 0 {
			next := m.inputQueue[0]
			m.inputQueue = m.inputQueue[1:]
			m.textarea.SetValue(next)
			return m.sendCurrentMessage()
		}

	case streamChunkMsg:
		m.messageQueue.UpdateLast(func(qm *QueuedMessage) {
			if msg.chunk != "" {
				qm.StreamChunk += msg.chunk
				qm.Content = qm.StreamChunk
			}
			if msg.thinking != "" {
				qm.Thinking += msg.thinking
			}
		})
		// Throttle viewport rebuilds: max once per 50ms during streaming
		if time.Since(m.lastStreamRender) >= 50*time.Millisecond {
			m.updateViewport()
			m.lastStreamRender = time.Now()
		}
		return m, nil

	case toolStartMsg:
		wasEmpty := len(m.activeTools) == 0
		tool := ToolExecution{
			Name:      msg.toolName,
			Args:      msg.args,
			Status:    "running",
			StartTime: time.Now(),
		}
		m.currentTool = &tool
		m.activeTools = append(m.activeTools, tool)
		m.status = fmt.Sprintf("🔧 %s", msg.toolName)
		// Tools section just appeared — recalculate layout to shrink viewport
		if wasEmpty && m.ready {
			m.recalculateLayout()
		}
		return m, nil

	case toolCompleteMsg:
		if m.currentTool != nil {
			m.currentTool.Status = "success"
			m.currentTool.EndTime = time.Now()
			m.currentTool.Result = msg.result

			// Update in activeTools
			for i := range m.activeTools {
				if m.activeTools[i].Name == m.currentTool.Name &&
					m.activeTools[i].StartTime.Equal(m.currentTool.StartTime) {
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
					m.activeTools[i].StartTime.Equal(m.currentTool.StartTime) {
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

	case pipelineStatusMsg:
		if m.pipelineStatus == nil {
			m.pipelineStatus = make(map[agent.AgentRole]string)
		}
		m.pipelineStatus[msg.role] = msg.status
		m.updateViewport()
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
		return m, m.handleConversationUpdate()

	case debateViewerTickMsg:
		cmd := m.handleDebateViewerTick()
		return m, cmd

	case spinner.TickMsg:
		if m.processing {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Update context stats
	m.contextStats = m.agent.GetContextStats()

	// Update textarea — always allow typing so users can queue instructions
	{
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update viewport
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	// Detect explicit user scroll intent to pause/resume auto-scroll
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "pgup", "ctrl+u":
			if !m.viewport.AtBottom() {
				m.userScrolledAway = true
			}
		case "end":
			m.userScrolledAway = false
		case "down", "pgdown", "ctrl+d":
			if m.viewport.AtBottom() {
				m.userScrolledAway = false
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the enhanced TUI
func (m EnhancedModel) View() string {
	if !m.ready {
		return m.spinner.View() + " Initializing ClosedWheelerAGI..."
	}

	// Render picker overlay if active (replaces main view)
	if m.pickerActive {
		return m.enhancedPickerView()
	}

	// Render help menu overlay if active (replaces main view)
	if m.helpActive {
		return m.helpMenuView()
	}

	// Render panel overlay if active (replaces main view)
	if m.panelActive {
		return m.panelView()
	}

	// Render settings overlay if active (replaces main view)
	if m.settingsActive {
		return m.settingsView()
	}

	// Render debate wizard overlay if active (replaces main view)
	if m.debateWizActive {
		return m.debateWizardView()
	}

	// Render debate viewer overlay if active (replaces main view)
	if m.debateViewActive {
		return m.debateViewerView()
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

	// Processing indicator (shown above input when active)
	if m.processing {
		sections = append(sections, m.renderProcessingArea())
	}

	// Input area — always visible so users can queue instructions
	sections = append(sections, m.textarea.View())

	// Help bar
	sections = append(sections, m.renderHelpBar())

	return strings.Join(sections, "\n")
}

// renderHeader renders a sleek modern header
func (m EnhancedModel) renderHeader() string {
	title := "CLOSED WHEELER AGI"
	version := "2.1"

	left := TitleStyle.Render("◈ " + title)
	versionText := HelpStyle.Render("v" + version)

	// Model info
	modelInfo := m.agent.Config().Model
	modelBadge := BadgeStyle.Copy().
		Foreground(SecondaryColor).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(MutedColor).
		Render("λ " + modelInfo)

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(versionText) - lipgloss.Width(modelBadge) - 10
	if gap < 0 {
		gap = 0
	}

	header := HeaderStyle.Width(m.width).Render(
		lipgloss.JoinHorizontal(lipgloss.Center,
			left,
			strings.Repeat(" ", 2),
			versionText,
			strings.Repeat(" ", gap),
			modelBadge,
		),
	)

	return header
}

// renderStatusBar renders a high-information, low-noise status bar
func (m EnhancedModel) renderStatusBar() string {
	var badge string
	var bStyle lipgloss.Style

	if m.processing {
		if m.currentTool != nil {
			badge = " WORKING "
			bStyle = WorkingBadgeStyle
		} else {
			badge = " THINKING "
			bStyle = ThinkingBadgeStyle
		}
	} else {
		badge = " IDLE "
		bStyle = IdleBadgeStyle
	}

	// Status message from agent
	statusMsg := ""
	if m.status != "" {
		statusMsg = " " + ThinkingStyle.Render(m.status)
	}

	// Stats section
	usage := m.agent.GetUsageStats()
	tok := formatK(toInt(usage["total_tokens"]))
	mem := m.agent.GetMemoryStats()
	stats := fmt.Sprintf("STM:%d WM:%d CTX:%d TOK:%s",
		mem["short_term"],
		mem["working"],
		m.contextStats.MessageCount,
		tok)

	statsItem := StatusItemStyle.Render(stats)

	left := lipgloss.JoinHorizontal(lipgloss.Center, bStyle.Render(badge), statusMsg)
	right := statsItem

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 4
	if gap < 0 {
		gap = 1
	}

	return StatusBarStyle.Width(m.width - 2).Render(
		lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", gap), right),
	)
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
			statusStyle = ToolRunningStyle
		case "success":
			icon = "✓"
			statusStyle = ToolSuccessStyle
		case "failed":
			icon = "✗"
			statusStyle = ToolFailedStyle
		default:
			icon = "○"
			statusStyle = ToolPendingStyle
		}

		duration := ""
		if !tool.EndTime.IsZero() {
			duration = fmt.Sprintf(" (%s)", tool.EndTime.Sub(tool.StartTime).Round(time.Millisecond))
		}

		durationStyled := ToolDurationStyle.Render(duration)

		toolItem := fmt.Sprintf("%s %s%s",
			statusStyle.Render(icon),
			tool.Name,
			durationStyled)

		toolItems = append(toolItems, toolItem)
	}

	toolsSection := ToolsSectionStyle.
		Width(m.width - 2).
		Render("🔧 " + strings.Join(toolItems, " │ "))

	return toolsSection
}

// renderProcessingArea renders the processing/thinking area.
// It must occupy exactly the same terminal height as the textarea (5 lines:
// rounded border top + 3 inner lines + rounded border bottom).
func (m EnhancedModel) renderProcessingArea() string {
	dots := strings.Repeat(".", m.thinkingAnimation)

	// Elapsed time since request started
	elapsedStr := ""
	if !m.requestStartTime.IsZero() {
		elapsed := time.Since(m.requestStartTime)
		elapsedStr = fmt.Sprintf(" [%.1fs]", elapsed.Seconds())
	}

	var line1 string
	if m.currentTool != nil {
		line1 = fmt.Sprintf("%s WORKING: %s%s%s",
			m.spinner.View(),
			strings.ToUpper(m.currentTool.Name),
			dots,
			elapsedStr)
	} else {
		line1 = fmt.Sprintf("%s THINKING%s%s",
			m.spinner.View(),
			dots,
			elapsedStr)
	}

	line2 := ""
	if m.agent.PipelineEnabled() && len(m.pipelineStatus) > 0 {
		line2 = renderPipelineBar(m.pipelineStatus)
	} else if m.currentTool != nil {
		if arg := extractToolArg(m.currentTool.Args); arg != "" {
			line2 = HelpStyle.Render("   ↳ " + arg)
		}
	} else {
		// Show thinking preview if streaming
		messages := m.messageQueue.GetAll()
		if len(messages) > 0 {
			last := messages[len(messages)-1]
			if last.Thinking != "" && !last.Complete {
				trimmed := strings.TrimSpace(last.Thinking)
				lines := strings.Split(trimmed, "\n")
				lastLine := lines[len(lines)-1]
				if len(lastLine) > m.width-15 {
					lastLine = "..." + lastLine[len(lastLine)-(m.width-18):]
				}
				line2 = ThinkingStyle.Render(lastLine)
			}
		}
	}

	inner := line1 + "\n" + line2

	processingStyle := ProcessingStyle.
		Width(m.width - 10)

	return processingStyle.Render(inner)
}

// extractToolArg attempts to extract a meaningful argument (like a filename) from tool JSON
func extractToolArg(argsJSON string) string {
	if argsJSON == "" {
		return ""
	}
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return ""
	}

	// Priority keys for display
	keys := []string{"TargetFile", "Path", "AbsolutePath", "SearchPath", "FileName", "Url", "Query", "CommandLine"}
	for _, k := range keys {
		if v, ok := args[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				// Special case for paths: show just the base or a compact version
				if strings.Contains(s, string(os.PathSeparator)) || strings.Contains(s, "/") {
					return filepath.Base(s)
				}
				if len(s) > 40 {
					return s[:37] + "..."
				}
				return s
			}
		}
	}
	return ""
}

// renderPipelineBar renders the 4-role status line for the multi-agent pipeline.
func renderPipelineBar(status map[agent.AgentRole]string) string {
	type roleInfo struct {
		role  agent.AgentRole
		icon  string
		label string
	}
	roles := []roleInfo{
		{agent.RolePlanner, "🧠", "Planner"},
		{agent.RoleResearcher, "🔍", "Research"},
		{agent.RoleExecutor, "⚙", "Executor"},
		{agent.RoleCritic, "🎯", "Critic"},
	}

	parts := make([]string, 0, len(roles))
	for _, r := range roles {
		s := status[r.role]
		var indicator string
		switch s {
		case "thinking":
			indicator = "●"
		case "done":
			indicator = "✓"
		case "error":
			indicator = "✗"
		default:
			indicator = "…"
		}
		parts = append(parts, fmt.Sprintf("%s %s %s", r.icon, r.label, indicator))
	}
	return strings.Join(parts, "  ")
}

// renderHelpBar renders the help bar
func (m EnhancedModel) renderHelpBar() string {
	helpText := "↵ Send │ /help Commands │ ^C Quit"

	if m.processing {
		helpText = "Esc / Ctrl+C — Stop request"
	}

	return HelpStyle.Render(helpText)
}

// renderDivider renders a divider with heavy horizontal line
func (m EnhancedModel) renderDivider() string {
	return DividerStyle.Render(strings.Repeat("━", m.width-2))
}

// calculateToolsHeight calculates the height needed for tools section
func (m EnhancedModel) calculateToolsHeight() int {
	if len(m.activeTools) > 0 {
		return 1
	}
	return 0
}

func (m EnhancedModel) calculateProcessingHeight() int {
	if m.processing {
		return 3
	}
	return 0
}

// calculateViewportHeight calculates the correct viewport height based on fixed components.
// View() uses strings.Join(sections, "\n"). Each "\n" separator only terminates the
// previous section's last line — it does NOT add an extra visual row. Therefore only
// the actual rendered line-heights of each section count toward fixedHeight.
// Section order: header | statusBar | [tools] | divider | viewport | divider | [processing] | input | helpBar
func (m *EnhancedModel) calculateViewportHeight() int {
	toolsH := m.calculateToolsHeight()           // 0 or 1
	processingH := m.calculateProcessingHeight() // 0 or 5

	// Rendered line heights of each non-viewport section:
	headerH := 1  // header: 1 line
	statusH := 1  // status bar: 1 line
	dividers := 2 // two dividers × 1 line each
	inputH := 5   // textarea with border: always present
	helpH := 1    // help bar: 1 line

	fixedHeight := headerH + statusH + toolsH + dividers + processingH + inputH + helpH

	viewportHeight := m.height - fixedHeight

	if viewportHeight < 5 {
		viewportHeight = 5
	}

	return viewportHeight
}

// recalculateLayout recalculates all layout dimensions consistently.
// Must be called whenever m.width, m.height, or m.activeTools changes.
func (m *EnhancedModel) recalculateLayout() {
	viewportHeight := m.calculateViewportHeight()
	vpWidth := m.width - 2

	// YPosition: rows above the viewport in the rendered output.
	// "\n" separators from strings.Join only terminate the previous line — no extra rows.
	// Above viewport: header(1) + status(1) + [tools(1)] + divider(1)
	toolsH := m.calculateToolsHeight()
	yPosition := 1 + 1 + toolsH + 1 // = 3 + toolsH

	if !m.ready {
		m.viewport = viewport.New(vpWidth, viewportHeight)
		m.viewport.YPosition = yPosition
		m.ready = true
	} else {
		m.viewport.Width = vpWidth
		m.viewport.Height = viewportHeight
		m.viewport.YPosition = yPosition
	}

	// Textarea: same width logic; height fixed at 3 inner lines (border adds 2, total 5)
	m.textarea.SetWidth(m.width - 8)
	m.textarea.SetHeight(3)
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

// maxDisplayContentLen caps the display length of a single message's Content
// to prevent viewport/wordwrap from choking on extremely long responses.
const maxDisplayContentLen = 50_000

// updateViewport updates the viewport content
func (m *EnhancedModel) updateViewport() {
	var sb strings.Builder

	messages := m.messageQueue.GetAll()

	// Rich layout only when the terminal is wide enough
	useRichLayout := m.width >= 50

	// Available width for message content (viewport minus 1-char margins each side)
	availWidth := m.width - 4
	if availWidth < 20 {
		availWidth = 20
	}

	for i, msg := range messages {
		// Timestamp string
		timestamp := ""
		if m.showTimestamps && !msg.Timestamp.IsZero() {
			timestamp = TimestampStyle.Render(msg.Timestamp.Format("15:04"))
		}

		// Truncate very long content for display safety
		displayContent := msg.Content
		if len(displayContent) > maxDisplayContentLen {
			displayContent = displayContent[:maxDisplayContentLen] + "\n...[truncated for display]..."
		}

		if !useRichLayout {
			// ── Narrow terminal fallback (< 50 cols) ─────────────
			m.renderNarrowMessage(&sb, msg, displayContent, timestamp, availWidth)
		} else {
			// ── Rich visual layout ──────────────────────────────
			switch msg.Role {
			case "user":
				m.renderRichUser(&sb, displayContent, timestamp, availWidth)
			case "assistant":
				m.renderRichAssistant(&sb, msg, displayContent, timestamp, availWidth)
			case "system":
				m.renderRichSystem(&sb, displayContent, availWidth)
			case "error":
				m.renderRichError(&sb, displayContent, availWidth)
			}
		}

		// Add spacing between messages
		if i < len(messages)-1 {
			sb.WriteString("\n\n")
		}
	}

	m.viewport.SetContent(sb.String())

	// Auto-scroll to bottom unless the user has explicitly scrolled away
	if !m.userScrolledAway {
		m.viewport.GotoBottom()
	}
}

// renderNarrowMessage renders a message for narrow terminals (< 50 cols).
func (m *EnhancedModel) renderNarrowMessage(sb *strings.Builder, msg QueuedMessage, content, timestamp string, availWidth int) {
	switch msg.Role {
	case "user":
		if timestamp != "" {
			sb.WriteString(timestamp + " ")
		}
		sb.WriteString(UserLabelStyle.Render("YOU"))
		sb.WriteString(" ")
		wrapped := wordwrap.String(content, availWidth-10)
		sb.WriteString(AssistantTextStyle.Render(wrapped))

	case "assistant":
		if timestamp != "" {
			sb.WriteString(timestamp + " ")
		}
		sb.WriteString(AssistantLabelStyle.Render("AGI"))
		if msg.Streaming && !msg.Complete {
			sb.WriteString(StreamCursorStyle.Render(" ▌"))
		}
		sb.WriteString("\n")

		if msg.Thinking != "" && m.verbose {
			sb.WriteString(ThinkingHeaderStyle.Render("   thoughts"))
			sb.WriteString("\n")
			wrapped := wordwrap.String(msg.Thinking, availWidth-4)
			sb.WriteString(ThinkingStyle.Render(wrapped))
			sb.WriteString("\n")
		}

		rendered := m.renderContent(content, availWidth-4)
		sb.WriteString(rendered)

		if msg.Complete && msg.Stats != nil {
			statsLine := fmt.Sprintf("   %s · %s · %.1fs",
				formatK(msg.Stats.PromptTokens),
				formatK(msg.Stats.CompletionTokens),
				msg.Stats.Elapsed.Seconds())
			sb.WriteString(HelpStyle.Render(statsLine))
			sb.WriteString("\n")
		}

	case "system":
		wrapped := wordwrap.String("   "+content, availWidth-3)
		sb.WriteString(SystemMsgStyle.Render(wrapped))

	case "error":
		sb.WriteString(ErrorMsgStyle.Render("   ERROR"))
		sb.WriteString("\n")
		wrapped := wordwrap.String(content, availWidth-3)
		sb.WriteString(ErrorMsgStyle.Render("   " + wrapped))
	}
}

// renderRichUser renders a user message as a right-aligned chat bubble.
// Frame-aware: uses GetHorizontalFrameSize to compute inner content width.
func (m *EnhancedModel) renderRichUser(sb *strings.Builder, content, timestamp string, availWidth int) {
	bubbleWidth := chatBubbleMaxWidth(availWidth)

	// Frame-aware inner width (border + padding)
	frameW := UserBubbleStyle.GetHorizontalFrameSize()
	innerWidth := bubbleWidth - frameW
	if innerWidth < 10 {
		innerWidth = 10
	}

	wrapped := wordwrap.String(content, innerWidth)
	bubble := UserBubbleStyle.Width(bubbleWidth).Render(wrapped)

	// Badge line: badge + timestamp
	badgeLine := UserBadgeStyle.Render(" YOU ")
	if timestamp != "" {
		badgeLine += "  " + timestamp
	}

	// Stack bubble and badge, right-aligned
	block := lipgloss.JoinVertical(lipgloss.Right, bubble, badgeLine)
	aligned := lipgloss.PlaceHorizontal(availWidth, lipgloss.Right, block)
	sb.WriteString(aligned)
}

// renderRichAssistant renders an assistant message with left border, thinking box, and stats pill.
// Frame-aware: uses GetHorizontalFrameSize to compute content width inside border.
func (m *EnhancedModel) renderRichAssistant(sb *strings.Builder, msg QueuedMessage, content, timestamp string, availWidth int) {
	// Badge line
	badge := AssistantBadgeStyle.Render(" AGI ")
	cursor := ""
	if msg.Streaming && !msg.Complete {
		cursor = StreamCursorStyle.Render(" ▌")
	}
	badgeLine := badge + cursor
	if timestamp != "" {
		badgeLine += "  " + timestamp
	}
	sb.WriteString(badgeLine)
	sb.WriteString("\n")

	// Thinking box (same width as bordered content)
	if msg.Thinking != "" && m.verbose {
		thinkBox := renderThinkingBox(msg.Thinking, availWidth)
		sb.WriteString(thinkBox)
		sb.WriteString("\n")
	}

	// Content inside left-border style with frame-aware width
	borderFrame := AssistantBorderStyle.GetHorizontalFrameSize()
	contentWidth := availWidth - borderFrame
	if contentWidth < 20 {
		contentWidth = 20
	}
	rendered := m.renderContent(content, contentWidth)
	bordered := AssistantBorderStyle.Width(availWidth).Render(rendered)
	sb.WriteString(bordered)
	sb.WriteString("\n")

	// Stats pill
	if msg.Complete && msg.Stats != nil {
		statsText := fmt.Sprintf("⚡ %s in · %s out · %.1fs",
			formatK(msg.Stats.PromptTokens),
			formatK(msg.Stats.CompletionTokens),
			msg.Stats.Elapsed.Seconds())
		sb.WriteString("  " + StatsPillStyle.Render(statsText))
		sb.WriteString("\n")
	}
}

// renderRichSystem renders a centered system message with decorative stars.
func (m *EnhancedModel) renderRichSystem(sb *strings.Builder, content string, availWidth int) {
	star := lipgloss.NewStyle().Foreground(AccentColor).Bold(true).Render("✦")
	text := SystemMsgStyle.Render(content)
	textW := lipgloss.Width(text)
	starW := lipgloss.Width(star)

	// Compute dash budget for each side
	occupied := textW + 2*starW + 6 // text + 2 stars + spaces around
	dashBudget := (availWidth - occupied) / 2
	if dashBudget < 1 {
		dashBudget = 1
	}
	dashes := DividerStyle.Render(strings.Repeat("━", dashBudget))

	line := dashes + " " + star + " " + text + " " + star + " " + dashes
	centered := lipgloss.PlaceHorizontal(availWidth, lipgloss.Center, line)
	sb.WriteString(centered)
}

// renderRichError renders an error message with a coral thick left border.
// Frame-aware: uses GetHorizontalFrameSize for inner content width.
func (m *EnhancedModel) renderRichError(sb *strings.Builder, content string, availWidth int) {
	header := ErrorBoxHeaderStyle.Render("✘ ERROR")
	borderFrame := ErrorBoxStyle.GetHorizontalFrameSize()
	innerWidth := availWidth - borderFrame
	if innerWidth < 10 {
		innerWidth = 10
	}
	body := wordwrap.String(content, innerWidth)
	inner := header + "\n" + body
	box := ErrorBoxStyle.Width(availWidth).Render(inner)
	sb.WriteString(box)
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
	m.userScrolledAway = false // new message → resume auto-scroll
	m.status = "Processing... (Esc or Ctrl+C to stop)"
	m.activeTools = []ToolExecution{} // Clear old tools
	m.requestStartTime = time.Now()
	m.requestBeforeUsage = m.agent.GetUsageStats()
	if m.ready {
		m.recalculateLayout()
	}
	m.updateViewport()

	return m, tea.Batch(
		m.sendMessage(input, m.requestBeforeUsage, m.requestStartTime),
		m.spinner.Tick,
	)
}

// chatBubbleMaxWidth returns the max bubble/content width for chat messages.
// 80% of available width, floored at 30, capped at 120.
func chatBubbleMaxWidth(totalWidth int) int {
	w := totalWidth * 80 / 100
	if w < 30 {
		w = 30
	}
	if w > 120 {
		w = 120
	}
	return w
}

// renderCodeBox renders a bordered code block with optional language label.
// Uses GetHorizontalFrameSize for frame-aware width calculation.
func renderCodeBox(lang string, lines []string, boxWidth int) string {
	if boxWidth < 10 {
		boxWidth = 10
	}

	// Frame-aware inner width (border + padding)
	frameW := CodeBoxStyle.GetHorizontalFrameSize()
	innerWidth := boxWidth - frameW
	if innerWidth < 6 {
		innerWidth = 6
	}

	var content strings.Builder
	for i, line := range lines {
		wrapped := wordwrap.String(line, innerWidth)
		content.WriteString(wrapped)
		if i < len(lines)-1 {
			content.WriteString("\n")
		}
	}

	box := CodeBoxStyle.Width(boxWidth).Render(content.String())

	if lang != "" && lang != "code" {
		label := CodeLangLabelStyle.Render(strings.ToUpper(lang))
		return label + "\n" + box
	}
	return box
}

// renderThinkingBox renders a bordered thinking/reasoning accordion.
// Uses GetHorizontalFrameSize for frame-aware width calculation.
func renderThinkingBox(thinking string, boxWidth int) string {
	if boxWidth < 10 {
		boxWidth = 10
	}

	// Frame-aware inner width (border + padding)
	frameW := ThinkingBoxStyle.GetHorizontalFrameSize()
	innerWidth := boxWidth - frameW
	if innerWidth < 6 {
		innerWidth = 6
	}

	header := ThinkingBoxHeaderStyle.Render("💭 Reasoning")
	wrapped := wordwrap.String(thinking, innerWidth)
	return ThinkingBoxStyle.Width(boxWidth).Render(header + "\n" + wrapped)
}

// renderMarkdownLine strips inline markdown markers (**, *, ") and returns
// plain text safe for wordwrap.String() and subsequent lipgloss rendering.
// Structural elements (headings, bullets, hr) are handled by renderContent directly.
func renderMarkdownLine(s string) string {
	var out strings.Builder
	runes := []rune(s)
	i := 0
	for i < len(runes) {
		// Bold: **text**
		if i+1 < len(runes) && runes[i] == '*' && runes[i+1] == '*' {
			i += 2
			for i < len(runes) {
				if i+1 < len(runes) && runes[i] == '*' && runes[i+1] == '*' {
					i += 2
					break
				}
				out.WriteRune(runes[i])
				i++
			}
			continue
		}
		// Italic: *text* (single asterisk, skip lone *)
		if runes[i] == '*' {
			i++
			for i < len(runes) {
				if runes[i] == '*' {
					i++
					break
				}
				out.WriteRune(runes[i])
				i++
			}
			continue
		}
		// Inline code: `code`
		if runes[i] == '`' {
			i++
			out.WriteRune('[')
			for i < len(runes) {
				if runes[i] == '`' {
					i++
					break
				}
				out.WriteRune(runes[i])
				i++
			}
			out.WriteRune(']')
			continue
		}
		out.WriteRune(runes[i])
		i++
	}
	return out.String()
}

// renderContent renders assistant content with markdown support.
// Handles: fenced code blocks (bordered), headings (#), bold (**), inline code (`), bullet lists.
// Applies word-wrap to prevent terminal layout corruption.
// contentWidth is the available width for the rendered content area.
func (m *EnhancedModel) renderContent(content string, contentWidth int) string {
	if contentWidth < 20 {
		contentWidth = 20
	}

	var result strings.Builder
	inCodeBlock := false
	var codeLang string
	var codeLines []string

	lines := strings.Split(content, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// ── tool error separator ────────────────────────────────
		if trimmed == "[error]:" {
			result.WriteString("\n" + ErrorMsgStyle.Render("━━━ FAILURE ━━━") + "\n")
			continue
		}

		// ── fenced code block toggle ──────────────────────────────
		if strings.HasPrefix(trimmed, "```") {
			if !inCodeBlock {
				// Opening fence
				inCodeBlock = true
				codeLang = strings.TrimPrefix(trimmed, "```")
				codeLines = nil
			} else {
				// Closing fence — render collected code as bordered box
				inCodeBlock = false
				codeBox := renderCodeBox(codeLang, codeLines, contentWidth-2)
				result.WriteString(codeBox)
				result.WriteString("\n")
				codeLines = nil
				codeLang = ""
			}
			continue
		}

		if inCodeBlock {
			codeLines = append(codeLines, line)
			continue
		}

		// ── horizontal rule ───────────────────────────────────────
		if trimmed == "---" || trimmed == "***" || trimmed == "___" {
			result.WriteString(DividerStyle.Render(strings.Repeat("━", contentWidth-6)))
			result.WriteString("\n")
			continue
		}

		// ── headings ──────────────────────────────────────────────
		if strings.HasPrefix(trimmed, "# ") {
			text := renderMarkdownLine(strings.TrimPrefix(trimmed, "# "))
			wrapped := wordwrap.String(text, contentWidth-6)
			result.WriteString(ContentHeadingStyle.Render("█ " + wrapped))
			result.WriteString("\n")
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			text := renderMarkdownLine(strings.TrimPrefix(trimmed, "## "))
			wrapped := wordwrap.String(text, contentWidth-6)
			result.WriteString(ContentHeadingStyle.Render("▸ " + wrapped))
			result.WriteString("\n")
			continue
		}

		// ── bullet list ───────────────────────────────────────────
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			text := renderMarkdownLine(trimmed[2:])
			wrapped := wordwrap.String(text, contentWidth-6)
			result.WriteString("  " + AssistantTextStyle.Render("◆ "+wrapped))
			result.WriteString("\n")
			continue
		}

		// ── regular text (inline markdown stripped) ───────────────
		plain := renderMarkdownLine(trimmed)
		wrapped := wordwrap.String(plain, contentWidth-2)
		result.WriteString(AssistantTextStyle.Render(wrapped))

		// Add newline except for last empty line
		if i < len(lines)-1 || line != "" {
			result.WriteString("\n")
		}
	}

	// Handle unclosed code fence (streaming case): render partial box
	if inCodeBlock && len(codeLines) > 0 {
		codeBox := renderCodeBox(codeLang, codeLines, contentWidth-2)
		result.WriteString(codeBox)
		result.WriteString("\n")
	}

	return result.String()
}

// sendMessage sends a message to the agent and captures per-response usage stats.
func (m EnhancedModel) sendMessage(input string, beforeUsage map[string]any, startTime time.Time) tea.Cmd {
	return func() tea.Msg {
		response, err := m.agent.Chat(input)
		elapsed := time.Since(startTime)
		afterUsage := m.agent.GetUsageStats()
		deltaPrompt := toInt(afterUsage["prompt_tokens"]) - toInt(beforeUsage["prompt_tokens"])
		deltaCompletion := toInt(afterUsage["completion_tokens"]) - toInt(beforeUsage["completion_tokens"])
		return responseCompleteMsg{
			content:         response,
			err:             err,
			elapsed:         elapsed,
			deltaPrompt:     deltaPrompt,
			deltaCompletion: deltaCompletion,
		}
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
	content         string
	err             error
	elapsed         time.Duration
	deltaPrompt     int
	deltaCompletion int
}

type toolStartMsg struct {
	toolName string
	args     string
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

// toInt safely converts an interface{} value (typically int from GetUsageStats) to int.
func toInt(v any) int {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return 0
}

// formatK formats a token count with K/M suffix for compact display.
func formatK(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

// isCancelledError returns true when the error is a user-initiated cancellation
// (context.Canceled or context.DeadlineExceeded wrapping "context canceled").
func isCancelledError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "context canceled") ||
		strings.Contains(s, "request canceled") ||
		strings.Contains(s, "operation was canceled")
}

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

// RunEnhanced starts the enhanced TUI with optional context for cancellation.
func RunEnhanced(ag *agent.Agent, ctx ...context.Context) error {
	opts := []tea.ProgramOption{
		tea.WithAltScreen(),
		tea.WithFilter(quitKeyFilter()),
	}
	if len(ctx) > 0 && ctx[0] != nil {
		opts = append(opts, tea.WithContext(ctx[0]))
	}

	p := tea.NewProgram(NewEnhancedModel(ag), opts...)

	// Set status callback
	ag.SetStatusCallback(func(s string) {
		p.Send(statusUpdateMsg{status: s})
	})

	// Set tool callbacks
	ag.SetToolCallbacks(func(name, args string) {
		p.Send(toolStartMsg{toolName: name, args: args})
	}, func(name, result string) {
		p.Send(toolCompleteMsg{result: result})
	}, func(name string, err error) {
		p.Send(toolErrorMsg{err: err})
	})

	// Set streaming callback — sends each chunk to the TUI for live display
	ag.SetStreamCallback(func(content string, thinking string, done bool) {
		if done {
			return // responseCompleteMsg will handle the final state
		}
		p.Send(streamChunkMsg{chunk: content, thinking: thinking})
	})

	// Set pipeline status callback — updates role indicators in the processing area
	ag.SetPipelineStatusCallback(func(role agent.AgentRole, status string) {
		p.Send(pipelineStatusMsg{role: role, status: status})
	})

	_, err := p.Run()

	// Clear callbacks to prevent sends after program exits
	ag.SetStatusCallback(nil)
	ag.SetStreamCallback(nil)
	ag.SetPipelineStatusCallback(nil)

	return err
}

// Shared message types used by both the main loop and helpers.

type streamChunkMsg struct {
	chunk    string
	thinking string
}

type statusUpdateMsg struct {
	status string
}

// pipelineStatusMsg is sent when a pipeline role changes status.
type pipelineStatusMsg struct {
	role   agent.AgentRole
	status string // "thinking" | "done" | "error"
}

// quitKeyFilter is a program-level filter that catches quit keys
// even if the model's Update somehow doesn't process them.
// It counts consecutive Ctrl+C presses and force-exits on the third.
func quitKeyFilter() func(tea.Model, tea.Msg) tea.Msg {
	ctrlCCount := 0
	return func(m tea.Model, msg tea.Msg) tea.Msg {
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.Type {
			case tea.KeyCtrlC, tea.KeyCtrlD, tea.KeyCtrlBackslash:
				ctrlCCount++
				if ctrlCCount >= 3 {
					// Nuclear option: force exit after 3 attempts
					// Restore terminal first, then print (safe because TUI is dead)
					fmt.Print("\033[?1000l\033[?1002l\033[?1003l\033[?1006l")
					fmt.Print("\033[?25h\033[?1049l")
					fmt.Println("\nForce quit.")
					os.Exit(1)
				}
			default:
				ctrlCCount = 0
			}
		}
		return msg
	}
}

// openBrowser attempts to open a URL in the default browser.
func openBrowser(url string) {
	_ = utils.OpenBrowser(url)
}
