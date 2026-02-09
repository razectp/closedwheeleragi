// Package tui provides a terminal user interface for the AGI agent.
package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"ClosedWheeler/pkg/agent"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Theme colors
var (
	// Primary colors
	primaryColor   = lipgloss.Color("#7C3AED") // Purple
	secondaryColor = lipgloss.Color("#06B6D4") // Cyan
	accentColor    = lipgloss.Color("#F59E0B") // Amber
	successColor   = lipgloss.Color("#10B981") // Green
	errorColor     = lipgloss.Color("#EF4444") // Red
	mutedColor     = lipgloss.Color("#6B7280") // Gray

	// Background
	bgDark   = lipgloss.Color("#1F2937")
	bgDarker = lipgloss.Color("#111827")

	// Text
	textPrimary   = lipgloss.Color("#F9FAFB")
	textSecondary = lipgloss.Color("#9CA3AF")
)

// Styles
var (
	// Header
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textPrimary).
			Background(primaryColor).
			Padding(0, 2).
			MarginBottom(1)

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
			Foreground(textSecondary).
			Background(bgDark).
			Padding(0, 1)

	statusItemStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	// Messages
	userBubbleStyle = lipgloss.NewStyle().
			Foreground(textPrimary).
			Background(lipgloss.Color("#374151")).
			Padding(0, 1).
			MarginBottom(1)

	userLabelStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	assistantLabelStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)

	assistantTextStyle = lipgloss.NewStyle().
				Foreground(textPrimary)

	systemMsgStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			MarginBottom(1)

	errorMsgStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	successMsgStyle = lipgloss.NewStyle().
			Foreground(successColor)

	// Code blocks
	codeBlockStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A5F3FC")).
			Background(bgDarker).
			Padding(0, 1)

	// Input area
	inputBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(0, 1)

	// Help
	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)

	// Divider
	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#374151"))

	// Badges
	badgeStyle = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1).
			MarginRight(1)

	idleBadgeStyle = badgeStyle.Copy().
			Background(mutedColor).
			Foreground(textPrimary)

	thinkingBadgeStyle = badgeStyle.Copy().
				Background(accentColor).
				Foreground(bgDarker)

	workingBadgeStyle = badgeStyle.Copy().
				Background(successColor).
				Foreground(bgDarker)

	memStatsStyle = lipgloss.NewStyle().
			Foreground(textSecondary).
			Faint(true)

	// Thinking/Reasoning styles
	thinkingStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			Faint(true)

	thinkingHeaderStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true).
				MarginBottom(1)
)

// Message represents a chat message
type Message struct {
	Role      string
	Content   string
	Timestamp time.Time
}

// Model represents the TUI state
type Model struct {
	agent          *agent.Agent
	viewport       viewport.Model
	textarea       textarea.Model
	spinner        spinner.Model
	messages       []Message
	width          int
	height         int
	ready          bool
	loading        bool
	streamingText  string
	err            error
	showTimestamps bool
	status         string
	verbose        bool

	// Model picker state
	pickerActive   bool
	pickerStep     int
	pickerCursor   int
	pickerSelected ProviderOption
	pickerInput    textinput.Model
	pickerNewKey   string
	pickerNewURL   string

	// OAuth login state
	loginActive   bool
	loginVerifier string
	loginAuthURL  string
	loginInput    textinput.Model
}

// NewModel creates a new TUI model
func NewModel(ag *agent.Agent) Model {
	// Text input
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Focus()
	ta.CharLimit = 4096
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Base = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor)

	// Spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(primaryColor)

	return Model{
		agent:          ag,
		textarea:       ta,
		spinner:        sp,
		showTimestamps: true,
		verbose:        ag.Config().UI.Verbose,
		messages: []Message{
			{
				Role:      "system",
				Content:   "🤖 Coder AGI ready! Type /help for commands.",
				Timestamp: time.Now(),
			},
		},
	}
}

// Init initializes the TUI
func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick)
}

// Update handles TUI events
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Quit keys — checked first, always active regardless of state
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlD, tea.KeyCtrlBackslash:
			return m, tea.Quit
		}

		// Model picker intercepts all keys when active
		if m.pickerActive {
			updated, cmd := m.pickerUpdate(msg)
			return updated, cmd
		}

		// Login mode intercepts all keys when active
		if m.loginActive {
			return m.loginUpdate(msg)
		}

		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			return m.sendCurrentMessage()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Dynamically calculate component heights
		headerH := lipgloss.Height(headerStyle.Render("AGI"))
		statusH := lipgloss.Height(m.renderStatusBar())
		inputH := 7 // Textarea height + borders
		helpH := lipgloss.Height(helpStyle.Render("H"))

		viewportHeight := m.height - headerH - statusH - inputH - helpH
		if viewportHeight < 0 {
			viewportHeight = 1
		}

		if !m.ready {
			m.viewport = viewport.New(msg.Width-4, viewportHeight)
			m.viewport.YPosition = headerH + statusH
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 4
			m.viewport.Height = viewportHeight
		}
		m.textarea.SetWidth(msg.Width - 6)
		m.updateViewport()

	case responseMsg:
		m.loading = false
		m.streamingText = ""
		if msg.err != nil {
			m.messages = append(m.messages, Message{
				Role:      "error",
				Content:   msg.err.Error(),
				Timestamp: time.Now(),
			})
		} else {
			m.messages = append(m.messages, Message{
				Role:      "assistant",
				Content:   msg.content,
				Timestamp: time.Now(),
			})
		}
		m.updateViewport()

	case streamChunkMsg:
		m.streamingText += msg.chunk
		m.updateViewport()

	case statusUpdateMsg:
		m.status = msg.status
		if m.verbose {
			m.messages = append(m.messages, Message{
				Role:      "system",
				Content:   fmt.Sprintf("DEBUG: %s", msg.status),
				Timestamp: time.Now(),
			})
			m.updateViewport()
		}
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Update textarea
	if !m.loading {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update viewport
	var cmd tea.Cmd
	viewportMsg := msg
	if !m.loading && m.textarea.Focused() {
		// If focused on textarea, don't pass single-character keys to viewport
		// to prevent navigation shortcuts (like 'u', 'd') from triggering.
		if k, ok := msg.(tea.KeyMsg); ok {
			if k.Type == tea.KeyRunes || (k.Type == tea.KeySpace && len(k.Runes) > 0) {
				viewportMsg = nil
			}
		}
	}

	if viewportMsg != nil {
		m.viewport, cmd = m.viewport.Update(viewportMsg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// sendCurrentMessage extracts input and sends it
func (m Model) sendCurrentMessage() (tea.Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}

	input := strings.TrimSpace(m.textarea.Value())
	if input == "" {
		return m, nil
	}

	// Handle commands
	if strings.HasPrefix(input, "/") {
		return m.handleCommand(input)
	}

	// Add user message
	m.messages = append(m.messages, Message{
		Role:      "user",
		Content:   input,
		Timestamp: time.Now(),
	})
	m.textarea.Reset()
	m.loading = true
	m.status = "Thinking..."
	m.updateViewport()

	return m, tea.Batch(m.sendMessage(input), m.spinner.Tick)
}

// View renders the TUI
func (m Model) View() string {
	if !m.ready {
		return m.spinner.View() + " Initializing..."
	}

	// Model picker overlay
	if m.pickerActive {
		var sb strings.Builder
		header := headerStyle.Render("🤖 Coder AGI")
		sb.WriteString(header)
		sb.WriteString("\n")
		sb.WriteString(m.renderStatusBar())
		sb.WriteString("\n")
		sb.WriteString(m.pickerView())
		return sb.String()
	}

	// Login overlay
	if m.loginActive {
		var sb strings.Builder
		header := headerStyle.Render("🤖 Coder AGI")
		sb.WriteString(header)
		sb.WriteString("\n")
		sb.WriteString(m.renderStatusBar())
		sb.WriteString("\n")
		sb.WriteString(m.loginView())
		return sb.String()
	}

	var sb strings.Builder

	// Header
	header := headerStyle.Render("🤖 Coder AGI")
	sb.WriteString(header)
	sb.WriteString("\n")

	// Status bar
	sb.WriteString(m.renderStatusBar())
	sb.WriteString("\n")

	// Divider
	divider := dividerStyle.Render(strings.Repeat("─", m.width-2))
	sb.WriteString(divider)
	sb.WriteString("\n")

	// Messages viewport
	sb.WriteString(m.viewport.View())
	sb.WriteString("\n")

	// Divider
	sb.WriteString(divider)
	sb.WriteString("\n")

	// Input area
	if m.loading {
		statusText := m.status
		if statusText == "" {
			statusText = "Thinking..."
		}
		loadingText := m.spinner.View() + " " + statusText
		if m.streamingText != "" {
			// Show streaming text preview
			preview := m.streamingText
			if len(preview) > 50 {
				preview = "..." + preview[len(preview)-47:]
			}
			loadingText = m.spinner.View() + " " + preview
		}
		sb.WriteString(lipgloss.NewStyle().Foreground(primaryColor).Render(loadingText))
	} else {
		sb.WriteString(m.textarea.View())
	}
	sb.WriteString("\n")

	// Help bar
	help := helpStyle.Render("Enter: Send │ /help: Commands │ /login: OAuth │ /model: Switch │ Ctrl+C/D: Quit")
	sb.WriteString(help)

	return sb.String()
}

func (m Model) renderStatusBar() string {
	// Status Badge
	var badge string
	if m.loading {
		if strings.Contains(m.status, "🔧") {
			badge = workingBadgeStyle.Render(" WORKING ")
		} else {
			badge = thinkingBadgeStyle.Render(" THINKING ")
		}
	} else {
		badge = idleBadgeStyle.Render(" IDLE ")
	}

	// Context Stats
	contextStats := m.agent.GetContextStats()
	var contextIndicator string
	if contextStats.ContextSent {
		contextIndicator = lipgloss.NewStyle().Foreground(successColor).Render("●") // Green dot - context cached
	} else {
		contextIndicator = lipgloss.NewStyle().Foreground(accentColor).Render("○") // Orange circle - needs refresh
	}

	// Memory Stats with context info
	stats := m.agent.GetMemoryStats()
	verboseStr := ""
	if m.verbose {
		verboseStr = lipgloss.NewStyle().Foreground(accentColor).Render(" [VERBOSE]")
	}

	// Show context message count and estimated size
	contextInfo := fmt.Sprintf(" │ CTX: %d msgs", contextStats.MessageCount)
	if contextStats.MessageCount > 15 { // Warning threshold
		contextInfo = lipgloss.NewStyle().Foreground(accentColor).Render(contextInfo)
	}

	mem := memStatsStyle.Render(fmt.Sprintf("%s STM: %d │ WM: %d │ LTM: %d%s",
		contextIndicator, stats["short_term"], stats["working"], stats["long_term"], contextInfo)) + verboseStr

	// Active Agent + Model
	modelName := m.agent.Config().Model
	agentInfo := lipgloss.NewStyle().Foreground(secondaryColor).Bold(true).Render("🦅 ClosedWheelerAGI") +
		lipgloss.NewStyle().Foreground(textSecondary).Render(" │ ") +
		lipgloss.NewStyle().Foreground(accentColor).Render(modelName)

	// Token Usage with session info
	usage := m.agent.GetUsageStats()
	sessionInfo := ""
	if contextStats.CompletionCount > 0 {
		sessionInfo = fmt.Sprintf(" (%d API calls)", contextStats.CompletionCount)
	}
	tokensStr := fmt.Sprintf("Tokens: %v%s", usage["total_tokens"], sessionInfo)
	tokens := lipgloss.NewStyle().Foreground(textSecondary).Faint(true).Render(tokensStr)

	left := lipgloss.JoinHorizontal(lipgloss.Center, badge, agentInfo, lipgloss.NewStyle().MarginLeft(2).Render(tokens))
	right := mem

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 4
	if gap < 0 {
		gap = 0
	}

	return statusBarStyle.Width(m.width - 2).Render(
		lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", gap), right),
	)
}

// handleCommand handles special slash commands
func (m Model) handleCommand(input string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])

	m.textarea.Reset()

	switch cmd {
	case "/help", "/h":
		m.messages = append(m.messages, Message{
			Role: "system",
			Content: `📚 Commands:
  /help, /h     - Show this help
  /login        - Anthropic OAuth login (Claude Pro/Max)
  /model        - Switch provider & model (interactive)
  /model <name> - Quick switch to a model
  /clear, /c    - Clear conversation
  /status, /s   - Show project status
  /reload, /r   - Reload project files
  /save         - Save memory state
  /verbose      - Toggle verbose mode (on/off)
  /telegram     - Show Telegram bot status & pairing
  /rules        - Show active project rules & context
  /git          - Show git status
  /logs         - Show debug logs
  /exit, /q     - Exit the program`,
			Timestamp: time.Now(),
		})

	case "/clear", "/c":
		m.messages = []Message{{
			Role:      "system",
			Content:   "✨ Conversation cleared.",
			Timestamp: time.Now(),
		}}

	case "/status", "/s":
		stats := m.agent.GetUsageStats()
		usageStr := fmt.Sprintf("\n\n📈 *API Usage:*\n- Tokens: %v (Prompt: %v, Comp: %v)\n- Remaining: %v tokens / %v requests",
			stats["total_tokens"], stats["prompt_tokens"], stats["completion_tokens"],
			stats["remaining_tokens"], stats["remaining_requests"])

		m.messages = append(m.messages, Message{
			Role:      "system",
			Content:   "📊 " + m.agent.GetProjectInfo() + usageStr,
			Timestamp: time.Now(),
		})

	case "/reload", "/r":
		if err := m.agent.ReloadProject(); err != nil {
			m.messages = append(m.messages, Message{
				Role:      "error",
				Content:   err.Error(),
				Timestamp: time.Now(),
			})
		} else {
			m.messages = append(m.messages, Message{
				Role:      "system",
				Content:   "🔄 Project reloaded.",
				Timestamp: time.Now(),
			})
		}

	case "/telegram":
		cfg := m.agent.Config().Telegram
		status := "❌ Disabled"
		if cfg.Enabled {
			status = "✅ Enabled"
		}

		pairingStatus := "⚠️ Not Paired"
		if cfg.ChatID != 0 {
			pairingStatus = fmt.Sprintf("🔗 Paired (Chat ID: %d)", cfg.ChatID)
		}

		m.messages = append(m.messages, Message{
			Role:      "system",
			Content:   fmt.Sprintf("🤖 *Telegram Bridge*\nStatus: %s\nPairing: %s\n\nInstruções:\n1. Mande /start para seu bot no Telegram\n2. Configure o Chat ID retornado no config.json", status, pairingStatus),
			Timestamp: time.Now(),
		})

	case "/verbose":
		m.verbose = !m.verbose
		if len(parts) > 1 {
			arg := strings.ToLower(parts[1])
			m.verbose = arg == "on" || arg == "true" || arg == "1"
		}

		// Update and save config
		m.agent.Config().UI.Verbose = m.verbose
		m.agent.SaveConfig()

		state := "OFF"
		if m.verbose {
			state = "ON"
		}
		m.messages = append(m.messages, Message{
			Role:      "system",
			Content:   fmt.Sprintf("📢 Verbose mode: %s (salvo em config.json)", state),
			Timestamp: time.Now(),
		})

	case "/rules":
		summary := m.agent.GetRulesSummary()
		fullRules := m.agent.GetFormattedRules()
		content := fmt.Sprintf("📖 *Project Rules & Context*\n\n%s", summary)
		if fullRules != "" {
			content += "\n\n---\n\n" + fullRules
		}
		m.messages = append(m.messages, Message{
			Role:      "system",
			Content:   content,
			Timestamp: time.Now(),
		})

	case "/save":
		if err := m.agent.Save(); err != nil {
			m.messages = append(m.messages, Message{
				Role:      "error",
				Content:   err.Error(),
				Timestamp: time.Now(),
			})
		} else {
			m.messages = append(m.messages, Message{
				Role:      "system",
				Content:   "💾 Memory saved.",
				Timestamp: time.Now(),
			})
		}

	case "/git":
		m.messages = append(m.messages, Message{
			Role:      "user",
			Content:   "Show git status",
			Timestamp: time.Now(),
		})
		m.loading = true
		m.updateViewport()
		return m, tea.Batch(m.sendMessage("Show me the current git status"), m.spinner.Tick)

	case "/logs":
		logs := m.agent.GetLogger().GetLastLines(20)
		m.messages = append(m.messages, Message{
			Role:      "system",
			Content:   "📋 Recent Logs:\n" + logs,
			Timestamp: time.Now(),
		})

	case "/model", "/m":
		if len(parts) > 1 {
			// Quick switch: /model <model-name>
			newModel := parts[1]
			cfg := m.agent.Config()
			if err := m.agent.SwitchModel(cfg.Provider, cfg.APIBaseURL, cfg.APIKey, newModel); err != nil {
				m.messages = append(m.messages, Message{
					Role:      "error",
					Content:   fmt.Sprintf("Failed to switch model: %v", err),
					Timestamp: time.Now(),
				})
			} else {
				m.messages = append(m.messages, Message{
					Role:      "system",
					Content:   fmt.Sprintf("Model switched to: %s", newModel),
					Timestamp: time.Now(),
				})
			}
		} else {
			// Interactive picker
			m.initPicker()
			return m, textinput.Blink
		}

	case "/login":
		m.messages = append(m.messages, Message{
			Role:      "system",
			Content:   "Use ./ClosedWheeler -login from the terminal for OAuth login.\nThe TUI login is not reliable on remote servers.",
			Timestamp: time.Now(),
		})
		m.updateViewport()
		return m, nil

	case "/exit", "/q":
		return m, tea.Quit

	default:
		m.messages = append(m.messages, Message{
			Role:      "error",
			Content:   fmt.Sprintf("Unknown command: %s. Type /help for available commands.", cmd),
			Timestamp: time.Now(),
		})
	}

	m.updateViewport()
	return m, nil
}

// updateViewport updates the viewport content
func (m *Model) updateViewport() {
	wasAtBottom := m.viewport.AtBottom()

	var sb strings.Builder

	for _, msg := range m.messages {
		// Timestamp (optional)
		timestamp := ""
		if m.showTimestamps && !msg.Timestamp.IsZero() {
			timestamp = lipgloss.NewStyle().Foreground(mutedColor).Render(msg.Timestamp.Format("15:04") + " ")
		}

		switch msg.Role {
		case "user":
			sb.WriteString(timestamp)
			sb.WriteString(userLabelStyle.Render("You: "))
			sb.WriteString(userBubbleStyle.Render(msg.Content))

		case "assistant":
			sb.WriteString(timestamp)
			sb.WriteString(assistantLabelStyle.Render("🤖 Assistant: "))
			sb.WriteString("\n")
			// Render with code block detection
			sb.WriteString(m.renderContent(msg.Content))

		case "system":
			sb.WriteString(systemMsgStyle.Render(msg.Content))

		case "error":
			sb.WriteString(errorMsgStyle.Render("❌ " + msg.Content))
		}
		sb.WriteString("\n\n")
	}

	// Show streaming text if loading
	if m.loading && m.streamingText != "" {
		sb.WriteString(assistantLabelStyle.Render("🤖 Assistant: "))
		sb.WriteString("\n")
		sb.WriteString(m.renderContent(m.streamingText))
		sb.WriteString(lipgloss.NewStyle().Foreground(primaryColor).Render("▌"))
	}

	m.viewport.SetContent(sb.String())

	if wasAtBottom || m.streamingText != "" {
		m.viewport.GotoBottom()
	}
}

// renderContent renders content with code block and thinking tag highlighting
func (m *Model) renderContent(content string) string {
	var result strings.Builder
	inCodeBlock := false
	inThinkingBlock := false

	// Handle <think> tags before line splitting to support multi-line blocks
	// DeepSeek and other models use these tags
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Code Block Detection
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

		// Thinking Tag Detection
		if strings.Contains(line, "<think>") {
			inThinkingBlock = true
			if m.verbose {
				result.WriteString(thinkingHeaderStyle.Render("💭 Reasoning Process:"))
				result.WriteString("\n")
			} else {
				result.WriteString(thinkingStyle.Render("💭 [IA está raciocinando... ative /verbose para ver o processo completo]"))
				result.WriteString("\n")
			}

			// If tag is on its own line, skip
			if trimmed == "<think>" {
				continue
			}
			// If there's content after <think>, strip the tag and keep processing
			line = strings.Replace(line, "<think>", "", 1)
		}

		if strings.Contains(line, "</think>") {
			if inThinkingBlock && m.verbose {
				result.WriteString(thinkingHeaderStyle.Render("────────────────────"))
				result.WriteString("\n")
			}
			inThinkingBlock = false

			// If tag is on its own line, skip
			if trimmed == "</think>" {
				continue
			}
			// If there's content before </think>, strip the tag and keep processing
			line = strings.Replace(line, "</think>", "", 1)
		}

		// Rendering logic
		if inCodeBlock {
			result.WriteString(codeBlockStyle.Render(line))
		} else if inThinkingBlock {
			if m.verbose {
				result.WriteString(thinkingStyle.Render(line))
			} else {
				// Don't render content inside thinking block if not verbose
				continue
			}
		} else {
			result.WriteString(assistantTextStyle.Render(line))
		}

		// Add newline except for the very last line if it's empty
		if i < len(lines)-1 || line != "" {
			if !inThinkingBlock || m.verbose {
				result.WriteString("\n")
			}
		}
	}

	return result.String()
}

// sendMessage sends a message to the agent
func (m Model) sendMessage(input string) tea.Cmd {
	return func() tea.Msg {
		response, err := m.agent.Chat(input)
		return responseMsg{content: response, err: err}
	}
}

// Message types
type responseMsg struct {
	content string
	err     error
}

type streamChunkMsg struct {
	chunk string
}

type statusUpdateMsg struct {
	status string
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
					fmt.Print("\033[?1000l\033[?1002l\033[?1003l\033[?1006l")
					fmt.Print("\033[?25h\033[?1049l")
					fmt.Fprintln(os.Stderr, "\nForce quit.")
					os.Exit(1)
				}
			default:
				ctrlCCount = 0
			}
		}
		return msg
	}
}

// loginUpdate handles key events during OAuth login flow.
func (m Model) loginUpdate(msg tea.KeyMsg) (Model, tea.Cmd) {
	if msg.Type == tea.KeyEsc {
		m.loginActive = false
		m.messages = append(m.messages, Message{
			Role:      "system",
			Content:   "OAuth login cancelled.",
			Timestamp: time.Now(),
		})
		m.updateViewport()
		return m, tea.EnableMouseCellMotion
	}

	if msg.Type == tea.KeyEnter {
		code := strings.TrimSpace(m.loginInput.Value())
		if code == "" {
			return m, nil
		}

		m.loginActive = false
		err := m.agent.LoginOAuth(code, m.loginVerifier)
		if err != nil {
			m.messages = append(m.messages, Message{
				Role:      "error",
				Content:   fmt.Sprintf("OAuth login failed: %v", err),
				Timestamp: time.Now(),
			})
		} else {
			m.messages = append(m.messages, Message{
				Role:      "system",
				Content:   fmt.Sprintf("OAuth login successful! Token %s.\nYou can now use Anthropic models.", m.agent.GetOAuthExpiry()),
				Timestamp: time.Now(),
			})
		}
		m.updateViewport()
		return m, tea.EnableMouseCellMotion
	}

	var cmd tea.Cmd
	m.loginInput, cmd = m.loginInput.Update(msg)
	return m, cmd
}

// loginView renders the OAuth login overlay.
func (m Model) loginView() string {
	var s strings.Builder
	boxWidth := m.width - 6
	if boxWidth < 20 {
		boxWidth = 20
	}

	s.WriteString(pickerTitleStyle.Render("🔑 Anthropic OAuth Login"))
	s.WriteString("\n\n")
	s.WriteString(pickerSubtitleStyle.Render("To authorize, download and open the HTML file locally:"))
	s.WriteString("\n\n")
	s.WriteString("  scp <this-server>:.agi/login.html /tmp/ && open /tmp/login.html")
	s.WriteString("\n\n")
	s.WriteString(pickerSubtitleStyle.Render("Or copy the raw URL (careful with line wrapping):"))
	s.WriteString("\n\n")
	s.WriteString("  cat .agi/login-url.txt")
	s.WriteString("\n\n")
	s.WriteString(pickerSubtitleStyle.Render("After authorizing, paste the code below:"))
	s.WriteString("\n\n")
	s.WriteString(m.loginInput.View())
	s.WriteString("\n\n")
	s.WriteString(pickerHintStyle.Render("  The code format is: code#state"))
	s.WriteString("\n")
	s.WriteString(pickerHintStyle.Render("  Press Enter to submit · Esc to cancel"))
	s.WriteString("\n")

	return pickerBoxStyle.Width(boxWidth).Render(s.String())
}

// openBrowser attempts to open a URL in the default browser.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	if err := cmd.Start(); err == nil {
		go func() { _ = cmd.Wait() }() // Reap process
	}
}

// Run starts the TUI with an optional context for cancellation.
func Run(ag *agent.Agent, ctx ...context.Context) error {
	opts := []tea.ProgramOption{
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithFilter(quitKeyFilter()),
	}
	if len(ctx) > 0 && ctx[0] != nil {
		opts = append(opts, tea.WithContext(ctx[0]))
	}

	p := tea.NewProgram(NewModel(ag), opts...)

	// Set status callback
	ag.SetStatusCallback(func(s string) {
		p.Send(statusUpdateMsg{status: s})
	})

	_, err := p.Run()

	// Clear status callback to prevent sends after program exits
	ag.SetStatusCallback(nil)

	return err
}
