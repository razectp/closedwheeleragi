// Package tui provides command handling for the TUI
package tui

import (
	"fmt"
	"strings"
	"time"

	"ClosedWheeler/pkg/tools"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Command represents a TUI command
type Command struct {
	Name        string
	Aliases     []string
	Category    string
	Description string
	Usage       string
	Handler     func(*EnhancedModel, []string) (tea.Model, tea.Cmd)
}

// CommandCategory represents a command category
type CommandCategory struct {
	Name     string
	Icon     string
	Commands []Command
}

// GetAllCommands returns all available commands organized by category
func GetAllCommands() []CommandCategory {
	return []CommandCategory{
		{
			Name: "Conversation",
			Icon: "💬",
			Commands: []Command{
				{
					Name:        "clear",
					Aliases:     []string{"c", "cls"},
					Category:    "Conversation",
					Description: "Clear conversation history",
					Usage:       "/clear",
					Handler:     cmdClear,
				},
				{
					Name:        "retry",
					Aliases:     []string{"r"},
					Category:    "Conversation",
					Description: "Retry last message",
					Usage:       "/retry",
					Handler:     cmdRetry,
				},
				{
					Name:        "continue",
					Aliases:     []string{"cont"},
					Category:    "Conversation",
					Description: "Continue last response",
					Usage:       "/continue",
					Handler:     cmdContinue,
				},
			},
		},
		{
			Name: "Information",
			Icon: "📊",
			Commands: []Command{
				{
					Name:        "status",
					Aliases:     []string{"s", "info"},
					Category:    "Information",
					Description: "Show project and system status",
					Usage:       "/status [detailed]",
					Handler:     cmdStatus,
				},
				{
					Name:        "stats",
					Aliases:     []string{"statistics"},
					Category:    "Information",
					Description: "Show API usage statistics",
					Usage:       "/stats",
					Handler:     cmdStats,
				},
				{
					Name:        "memory",
					Aliases:     []string{"mem"},
					Category:    "Information",
					Description: "Show memory system details",
					Usage:       "/memory [clear]",
					Handler:     cmdMemory,
				},
				{
					Name:        "context",
					Aliases:     []string{"ctx"},
					Category:    "Information",
					Description: "Show context cache status",
					Usage:       "/context [reset]",
					Handler:     cmdContext,
				},
				{
					Name:        "tools",
					Aliases:     []string{"t"},
					Category:    "Information",
					Description: "List available tools",
					Usage:       "/tools [category]",
					Handler:     cmdTools,
				},
			},
		},
		{
			Name: "Project",
			Icon: "📁",
			Commands: []Command{
				{
					Name:        "reload",
					Aliases:     []string{"refresh"},
					Category:    "Project",
					Description: "Reload project files and rules",
					Usage:       "/reload",
					Handler:     cmdReload,
				},
				{
					Name:        "rules",
					Aliases:     []string{"agirules"},
					Category:    "Project",
					Description: "Show active project rules",
					Usage:       "/rules",
					Handler:     cmdRules,
				},
				{
					Name:        "git",
					Aliases:     []string{"g"},
					Category:    "Project",
					Description: "Show git status",
					Usage:       "/git [status|diff|log]",
					Handler:     cmdGit,
				},
				{
					Name:        "health",
					Aliases:     []string{"check"},
					Category:    "Project",
					Description: "Run health check",
					Usage:       "/health",
					Handler:     cmdHealth,
				},
			},
		},
		{
			Name: "Features",
			Icon: "⚙️",
			Commands: []Command{
				{
					Name:        "verbose",
					Aliases:     []string{"v"},
					Category:    "Features",
					Description: "Toggle verbose mode (shows reasoning)",
					Usage:       "/verbose [on|off]",
					Handler:     cmdVerbose,
				},
				{
					Name:        "debug",
					Aliases:     []string{"d"},
					Category:    "Features",
					Description: "Toggle debug mode for tools",
					Usage:       "/debug [on|off|level]",
					Handler:     cmdDebug,
				},
				{
					Name:        "timestamps",
					Aliases:     []string{"time"},
					Category:    "Features",
					Description: "Toggle message timestamps",
					Usage:       "/timestamps [on|off]",
					Handler:     cmdTimestamps,
				},
				{
					Name:        "browser",
					Aliases:     []string{"b"},
					Category:    "Features",
					Description: "Configure browser automation",
					Usage:       "/browser [headless|stealth|slowmo] [value]",
					Handler:     cmdBrowser,
				},
				{
					Name:        "heartbeat",
					Aliases:     []string{"hb"},
					Category:    "Features",
					Description: "Configure heartbeat interval",
					Usage:       "/heartbeat [seconds|off]",
					Handler:     cmdHeartbeat,
				},
				{
					Name:        "pipeline",
					Aliases:     []string{"multi-agent", "ma"},
					Category:    "Features",
					Description: "Toggle multi-agent pipeline (Planner→Researcher→Executor→Critic)",
					Usage:       "/pipeline [on|off|status]",
					Handler:     cmdPipeline,
				},
			},
		},
		{
			Name: "Memory & Brain",
			Icon: "🧠",
			Commands: []Command{
				{
					Name:        "brain",
					Aliases:     []string{"knowledge"},
					Category:    "Memory & Brain",
					Description: "View or search knowledge base",
					Usage:       "/brain [search <query>|recent]",
					Handler:     cmdBrain,
				},
				{
					Name:        "roadmap",
					Aliases:     []string{"goals"},
					Category:    "Memory & Brain",
					Description: "View strategic roadmap",
					Usage:       "/roadmap [summary]",
					Handler:     cmdRoadmap,
				},
				{
					Name:        "save",
					Aliases:     []string{"persist"},
					Category:    "Memory & Brain",
					Description: "Save memory state to disk",
					Usage:       "/save",
					Handler:     cmdSave,
				},
			},
		},
		{
			Name: "Integration",
			Icon: "🔗",
			Commands: []Command{
				{
					Name:        "telegram",
					Aliases:     []string{"tg"},
					Category:    "Integration",
					Description: "Show Telegram bot status",
					Usage:       "/telegram",
					Handler:     cmdTelegram,
				},
				{
					Name:        "model",
					Aliases:     []string{"m"},
					Category:    "Integration",
					Description: "Interactive model/provider picker",
					Usage:       "/model [model-name [effort]]",
					Handler:     cmdModel,
				},
			},
		},
		{
			Name: "Providers",
			Icon: "🔌",
			Commands: []Command{
				{
					Name:        "providers",
					Aliases:     []string{"provider", "prov"},
					Category:    "Providers",
					Description: "Manage LLM providers",
					Usage:       "/providers [list|add|remove|enable|disable|set-primary|stats|examples]",
					Handler:     cmdProviders,
				},
				{
					Name:        "pairings",
					Aliases:     []string{"pairs"},
					Category:    "Providers",
					Description: "Show suggested provider pairings for debates",
					Usage:       "/pairings",
					Handler:     cmdPairings,
				},
			},
		},
		{
			Name: "Dual Session",
			Icon: "🤖",
			Commands: []Command{
				{
					Name:        "session",
					Aliases:     []string{"dual"},
					Category:    "Dual Session",
					Description: "Enable/disable dual session mode",
					Usage:       "/session [on|off|status]",
					Handler:     cmdSession,
				},
				{
					Name:        "debate",
					Aliases:     []string{"converse", "discuss"},
					Category:    "Dual Session",
					Description: "Start agent-to-agent conversation",
					Usage:       "/debate <topic> [turns]",
					Handler:     cmdDebate,
				},
				{
					Name:        "conversation",
					Aliases:     []string{"conv", "log"},
					Category:    "Dual Session",
					Description: "View dual session conversation log (live updates)",
					Usage:       "/conversation",
					Handler:     cmdConversation,
				},
				{
					Name:        "stop",
					Aliases:     []string{"end"},
					Category:    "Dual Session",
					Description: "Stop the current debate/conversation",
					Usage:       "/stop",
					Handler:     cmdStop,
				},
			},
		},
		{
			Name: "System",
			Icon: "🖥️",
			Commands: []Command{
				{
					Name:        "logs",
					Aliases:     []string{"log"},
					Category:    "System",
					Description: "Show recent logs",
					Usage:       "/logs [n]",
					Handler:     cmdLogs,
				},
				{
					Name:        "config",
					Aliases:     []string{"cfg"},
					Category:    "System",
					Description: "Show or reload configuration",
					Usage:       "/config [reload|show]",
					Handler:     cmdConfig,
				},
				{
					Name:        "report",
					Aliases:     []string{"debug-report"},
					Category:    "System",
					Description: "Generate debug report",
					Usage:       "/report",
					Handler:     cmdReport,
				},
				{
					Name:        "errors",
					Aliases:     []string{"errs"},
					Category:    "System",
					Description: "Show recent errors",
					Usage:       "/errors [n|clear]",
					Handler:     cmdErrors,
				},
				{
					Name:        "resilience",
					Aliases:     []string{"recovery"},
					Category:    "System",
					Description: "Show error resilience system status",
					Usage:       "/resilience",
					Handler:     cmdResilience,
				},
				{
					Name:        "tool-retries",
					Category:    "System",
					Description: "Show intelligent tool retry statistics",
					Usage:       "/tool-retries",
					Handler:     cmdToolRetries,
				},
				{
					Name:        "retry-mode",
					Category:    "System",
					Description: "Toggle intelligent retry feedback mode",
					Usage:       "/retry-mode [on|off]",
					Handler:     cmdRetryMode,
				},
				{
					Name:        "recover",
					Aliases:     []string{"heal"},
					Category:    "System",
					Description: "Run system recovery procedures",
					Usage:       "/recover",
					Handler:     cmdRecover,
				},
				{
					Name:        "help",
					Aliases:     []string{"h", "?"},
					Category:    "System",
					Description: "Show this help",
					Usage:       "/help [command]",
					Handler:     cmdHelp,
				},
				{
					Name:        "exit",
					Aliases:     []string{"quit", "q"},
					Category:    "System",
					Description: "Exit the program",
					Usage:       "/exit",
					Handler:     cmdExit,
				},
			},
		},
	}
}

// FindCommand finds a command by name or alias
func FindCommand(name string) *Command {
	name = strings.ToLower(strings.TrimPrefix(name, "/"))

	categories := GetAllCommands()
	for _, cat := range categories {
		for _, cmd := range cat.Commands {
			if cmd.Name == name {
				return &cmd
			}
			for _, alias := range cmd.Aliases {
				if alias == name {
					return &cmd
				}
			}
		}
	}
	return nil
}

// Command Handlers

func cmdClear(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	m.messageQueue.Clear()
	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   "✨ Conversation cleared.",
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}

func cmdRetry(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	messages := m.messageQueue.GetAll()
	if len(messages) < 2 {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   "No previous message to retry.",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	// Find last user message
	var lastUserMsg string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			lastUserMsg = messages[i].Content
			break
		}
	}

	if lastUserMsg == "" {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   "No user message found to retry.",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	// Add retry indicator
	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   "🔄 Retrying last message...",
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

	m.processing = true
	m.status = "Retrying..."
	m.requestStartTime = time.Now()
	m.requestBeforeUsage = m.agent.GetUsageStats()
	m.updateViewport()

	return *m, tea.Batch(
		m.sendMessage(lastUserMsg, m.requestBeforeUsage, m.requestStartTime),
		m.spinner.Tick,
	)
}

func cmdContinue(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	// Add continue message and trigger new response
	m.messageQueue.Add(QueuedMessage{
		Role:      "user",
		Content:   "Please continue from where you left off.",
		Timestamp: time.Now(),
		Complete:  true,
	})

	m.messageQueue.Add(QueuedMessage{
		Role:      "assistant",
		Content:   "",
		Streaming: true,
		Timestamp: time.Now(),
		Complete:  false,
	})

	m.processing = true
	m.status = "Continuing..."
	m.requestStartTime = time.Now()
	m.requestBeforeUsage = m.agent.GetUsageStats()
	m.updateViewport()

	return *m, tea.Batch(
		m.sendMessage("Continue from where you left off.", m.requestBeforeUsage, m.requestStartTime),
		m.spinner.Tick,
	)
}

func cmdStatus(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	detailed := len(args) > 0 && args[0] == "detailed"

	projectInfo := m.agent.GetProjectInfo()
	usage := m.agent.GetUsageStats()
	contextStats := m.agent.GetContextStats()

	var content strings.Builder
	content.WriteString("📊 **System Status**\n\n")
	content.WriteString(projectInfo)

	if detailed {
		content.WriteString("\n\n**Context:**\n")
		content.WriteString(fmt.Sprintf("- Messages: %d\n", contextStats.MessageCount))
		content.WriteString(fmt.Sprintf("- API Calls: %d\n", contextStats.CompletionCount))
		content.WriteString(fmt.Sprintf("- Context Cached: %v\n", contextStats.ContextSent))

		content.WriteString("\n**Memory:**\n")
		stats := m.agent.GetMemoryStats()
		content.WriteString(fmt.Sprintf("- Short-term: %d items\n", stats["short_term"]))
		content.WriteString(fmt.Sprintf("- Working: %d items\n", stats["working"]))
		content.WriteString(fmt.Sprintf("- Long-term: %d items\n", stats["long_term"]))

		content.WriteString("\n**Features:**\n")
		content.WriteString(fmt.Sprintf("- Verbose: %v\n", m.verbose))
		content.WriteString(fmt.Sprintf("- Debug Tools: %v\n", m.agent.Config().DebugTools))
		content.WriteString(fmt.Sprintf("- Heartbeat: %ds\n", m.agent.Config().HeartbeatInterval))
	}

	content.WriteString("\n**API Usage:**\n")
	content.WriteString(fmt.Sprintf("- Total Tokens: %v\n", usage["total_tokens"]))
	content.WriteString(fmt.Sprintf("- Prompt Tokens: %v\n", usage["prompt_tokens"]))
	content.WriteString(fmt.Sprintf("- Completion Tokens: %v\n", usage["completion_tokens"]))

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   content.String(),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}

func cmdStats(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	usage := m.agent.GetUsageStats()
	contextStats := m.agent.GetContextStats()

	var content strings.Builder
	content.WriteString("📈 **API Usage Statistics**\n\n")
	content.WriteString("**Tokens:**\n")
	content.WriteString(fmt.Sprintf("- Total: %v\n", usage["total_tokens"]))
	content.WriteString(fmt.Sprintf("- Prompt: %v\n", usage["prompt_tokens"]))
	content.WriteString(fmt.Sprintf("- Completion: %v\n", usage["completion_tokens"]))

	content.WriteString("\n**Rate Limits:**\n")
	content.WriteString(fmt.Sprintf("- Remaining Tokens: %v\n", usage["remaining_tokens"]))
	content.WriteString(fmt.Sprintf("- Remaining Requests: %v\n", usage["remaining_requests"]))

	content.WriteString("\n**Session:**\n")
	content.WriteString(fmt.Sprintf("- Messages: %d\n", contextStats.MessageCount))
	content.WriteString(fmt.Sprintf("- API Calls: %d\n", contextStats.CompletionCount))

	avgTokensPerMsg := 0
	if contextStats.MessageCount > 0 {
		totalTokens := usage["total_tokens"]
		if tokens, ok := totalTokens.(int); ok {
			avgTokensPerMsg = tokens / contextStats.MessageCount
		}
	}
	content.WriteString(fmt.Sprintf("- Avg Tokens/Message: %d\n", avgTokensPerMsg))

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   content.String(),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}

func cmdMemory(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) > 0 && args[0] == "clear" {
		// Clear specific tier or all
		tier := "all"
		if len(args) > 1 {
			tier = args[1]
		}

		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   fmt.Sprintf("🗑️ Memory cleared: %s", tier),
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	stats := m.agent.GetMemoryStats()
	var content strings.Builder
	content.WriteString("🧠 **Memory System**\n\n")
	content.WriteString(fmt.Sprintf("**Short-term Memory:** %d items\n", stats["short_term"]))
	content.WriteString("Recent conversation messages\n\n")
	content.WriteString(fmt.Sprintf("**Working Memory:** %d items\n", stats["working"]))
	content.WriteString("Currently active files and functions\n\n")
	content.WriteString(fmt.Sprintf("**Long-term Memory:** %d items\n", stats["long_term"]))
	content.WriteString("Compressed summaries and decisions\n")

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   content.String(),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}

func cmdContext(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) > 0 && args[0] == "reset" {
		// Reset context cache
		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   "🔄 Context cache reset. Next message will send full context.",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	contextStats := m.agent.GetContextStats()
	var content strings.Builder
	content.WriteString("📦 **Context Status**\n\n")

	if contextStats.ContextSent {
		content.WriteString("**Status:** ● Cached (green)\n")
		content.WriteString("Context has been sent and is cached by the model.\n")
	} else {
		content.WriteString("**Status:** ○ Not Cached (orange)\n")
		content.WriteString("Next message will send full context.\n")
	}

	content.WriteString(fmt.Sprintf("\n**Messages:** %d\n", contextStats.MessageCount))
	content.WriteString(fmt.Sprintf("**API Calls:** %d\n", contextStats.CompletionCount))

	if contextStats.MessageCount > 15 {
		content.WriteString("\n⚠️ **Warning:** High message count. Context may be compressed soon.\n")
	}

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   content.String(),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}

func cmdTools(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	var content strings.Builder
	content.WriteString("🔧 **Available Tools**\n\n")

	categories := map[string][]string{
		"File Operations":        {"read_file", "write_file", "edit_file", "list_files"},
		"Browser":                {"browser_navigate", "browser_click", "browser_type", "browser_screenshot"},
		"Git (enable_git_tools)": {"git_status", "git_diff", "git_commit", "git_push"},
		"Analysis":               {"analyze_code", "security_scan", "run_diagnostics"},
		"Tasks":                  {"list_tasks", "complete_task"},
	}

	filter := ""
	if len(args) > 0 {
		filter = strings.ToLower(args[0])
	}

	for category, tools := range categories {
		if filter != "" && !strings.Contains(strings.ToLower(category), filter) {
			continue
		}

		content.WriteString(fmt.Sprintf("**%s:**\n", category))
		for _, tool := range tools {
			content.WriteString(fmt.Sprintf("- `%s`\n", tool))
		}
		content.WriteString("\n")
	}

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   content.String(),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}

func cmdReload(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if err := m.agent.ReloadProject(); err != nil {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   fmt.Sprintf("❌ Failed to reload: %v", err),
			Timestamp: time.Now(),
			Complete:  true,
		})
	} else {
		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   "🔄 Project files and rules reloaded successfully.",
			Timestamp: time.Now(),
			Complete:  true,
		})
	}
	m.updateViewport()
	return *m, nil
}

func cmdRules(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	summary := m.agent.GetRulesSummary()
	fullRules := m.agent.GetFormattedRules()

	var content strings.Builder
	content.WriteString("📖 **Project Rules & Context**\n\n")
	content.WriteString(summary)

	if len(args) > 0 && args[0] == "full" {
		if fullRules != "" {
			content.WriteString("\n\n" + fullRules)
		}
	}

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   content.String(),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}

func cmdGit(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	subcommand := "status"
	if len(args) > 0 {
		subcommand = args[0]
	}

	var prompt string
	switch subcommand {
	case "status":
		prompt = "Show git status"
	case "diff":
		prompt = "Show git diff"
	case "log":
		prompt = "Show git log (last 5 commits)"
	default:
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   fmt.Sprintf("Unknown git subcommand: %s", subcommand),
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	m.messageQueue.Add(QueuedMessage{
		Role:      "user",
		Content:   prompt,
		Timestamp: time.Now(),
		Complete:  true,
	})

	m.messageQueue.Add(QueuedMessage{
		Role:      "assistant",
		Content:   "",
		Streaming: true,
		Timestamp: time.Now(),
		Complete:  false,
	})

	m.processing = true
	m.status = "Running git command..."
	m.requestStartTime = time.Now()
	m.requestBeforeUsage = m.agent.GetUsageStats()
	m.updateViewport()

	return *m, tea.Batch(
		m.sendMessage(prompt, m.requestBeforeUsage, m.requestStartTime),
		m.spinner.Tick,
	)
}

func cmdHealth(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	status := m.agent.PerformHealthCheck()

	var content strings.Builder
	content.WriteString("🏥 **Project Health Check**\n\n")
	content.WriteString(fmt.Sprintf("**Build:** %s\n", status.BuildStatus))
	content.WriteString(fmt.Sprintf("**Tests:** %s\n", status.TestStatus))
	content.WriteString(fmt.Sprintf("**Git:** %s\n", status.GitStatus))
	content.WriteString(fmt.Sprintf("**Pending Tasks:** %d\n", status.PendingTasks))

	if len(status.Warnings) > 0 {
		content.WriteString("\n⚠️ **Warnings:**\n")
		for _, warning := range status.Warnings {
			content.WriteString(fmt.Sprintf("- %s\n", warning))
		}
	}

	if len(status.Recommendations) > 0 {
		content.WriteString("\n💡 **Recommendations:**\n")
		for _, rec := range status.Recommendations {
			content.WriteString(fmt.Sprintf("- %s\n", rec))
		}
	}

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   content.String(),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}

func cmdVerbose(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) > 0 {
		arg := strings.ToLower(args[0])
		m.verbose = arg == "on" || arg == "true" || arg == "1"
	} else {
		m.verbose = !m.verbose
	}

	m.agent.Config().UI.Verbose = m.verbose
	if err := m.agent.SaveConfig(); err != nil {
		m.agent.GetLogger().Error("Failed to save config: %v", err)
	}

	state := lipgloss.NewStyle().Foreground(ErrorColor).Render("OFF")
	if m.verbose {
		state = lipgloss.NewStyle().Foreground(SuccessColor).Render("ON")
	}

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   fmt.Sprintf("📢 Verbose mode: %s (saved to config)", state),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}

func cmdDebug(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		// Toggle
		m.agent.Config().DebugTools = !m.agent.Config().DebugTools
	} else {
		switch arg := strings.ToLower(args[0]); arg {
		case "on", "true", "1":
			m.agent.Config().DebugTools = true
		case "off", "false", "0":
			m.agent.Config().DebugTools = false
		case "basic":
			m.agent.Config().DebugTools = true
			tools.SetGlobalDebugLevel(tools.DebugBasic)
		case "verbose":
			m.agent.Config().DebugTools = true
			tools.SetGlobalDebugLevel(tools.DebugVerbose)
		case "trace":
			m.agent.Config().DebugTools = true
			tools.SetGlobalDebugLevel(tools.DebugTrace)
		}
	}

	// Set debug level
	if m.agent.Config().DebugTools {
		tools.SetGlobalDebugLevel(tools.DebugVerbose)
	} else {
		tools.SetGlobalDebugLevel(tools.DebugOff)
	}

	if err := m.agent.SaveConfig(); err != nil {
		m.agent.GetLogger().Error("Failed to save config: %v", err)
	}

	state := lipgloss.NewStyle().Foreground(ErrorColor).Render("OFF")
	if m.agent.Config().DebugTools {
		state = lipgloss.NewStyle().Foreground(SuccessColor).Render("ON")
	}

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   fmt.Sprintf("🐛 Debug mode: %s\n\nLevels: basic, verbose, trace", state),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}

func cmdTimestamps(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) > 0 {
		arg := strings.ToLower(args[0])
		m.showTimestamps = arg == "on" || arg == "true" || arg == "1"
	} else {
		m.showTimestamps = !m.showTimestamps
	}

	state := lipgloss.NewStyle().Foreground(ErrorColor).Render("OFF")
	if m.showTimestamps {
		state = lipgloss.NewStyle().Foreground(SuccessColor).Render("ON")
	}

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   fmt.Sprintf("🕒 Timestamps: %s", state),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}

func cmdBrowser(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		// Show current settings
		cfg := m.agent.Config().Browser
		var content strings.Builder
		content.WriteString("🌐 **Browser Configuration**\n\n")
		content.WriteString(fmt.Sprintf("**Headless:** %v\n", cfg.Headless))
		content.WriteString(fmt.Sprintf("**SlowMo:** %dms\n", cfg.SlowMo))

		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   content.String(),
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	setting := strings.ToLower(args[0])
	value := ""
	if len(args) > 1 {
		value = strings.ToLower(args[1])
	}

	switch setting {
	case "headless":
		switch value {
		case "on", "true":
			m.agent.Config().Browser.Headless = true
		case "off", "false":
			m.agent.Config().Browser.Headless = false
		default:
			m.agent.Config().Browser.Headless = !m.agent.Config().Browser.Headless
		}

	case "slowmo":
		// Parse value as int
		var slowmo int
		fmt.Sscanf(value, "%d", &slowmo)
		m.agent.Config().Browser.SlowMo = slowmo
	}

	if err := m.agent.SaveConfig(); err != nil {
		m.agent.GetLogger().Error("Failed to save config: %v", err)
	}

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   "🌐 Browser configuration updated and saved.",
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}

func cmdHeartbeat(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		// Show current setting
		interval := m.agent.Config().HeartbeatInterval
		status := "disabled"
		if interval > 0 {
			status = fmt.Sprintf("%d seconds", interval)
		}

		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   fmt.Sprintf("💓 Heartbeat interval: %s", status),
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	arg := strings.ToLower(args[0])
	if arg == "off" || arg == "disable" {
		m.agent.Config().HeartbeatInterval = 0
	} else {
		var interval int
		fmt.Sscanf(arg, "%d", &interval)
		m.agent.Config().HeartbeatInterval = interval
	}

	if err := m.agent.SaveConfig(); err != nil {
		m.agent.GetLogger().Error("Failed to save config: %v", err)
	}

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   "💓 Heartbeat configuration updated. Restart required for changes to take effect.",
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}

func cmdBrain(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	brain := m.agent.GetBrain()

	if len(args) == 0 || args[0] == "show" {
		content, err := brain.Read()
		if err != nil {
			m.messageQueue.Add(QueuedMessage{
				Role:      "error",
				Content:   fmt.Sprintf("❌ Failed to read brain: %v", err),
				Timestamp: time.Now(),
				Complete:  true,
			})
		} else {
			// Show first 1000 chars
			if len(content) > 1000 {
				content = content[:1000] + "\n\n... (truncated, use /brain search <query> to find specific entries)"
			}

			m.messageQueue.Add(QueuedMessage{
				Role:      "system",
				Content:   "🧠 **Knowledge Base**\n\n" + content,
				Timestamp: time.Now(),
				Complete:  true,
			})
		}
	} else if args[0] == "search" && len(args) > 1 {
		query := strings.Join(args[1:], " ")
		matches, err := brain.Search(query)
		if err != nil {
			m.messageQueue.Add(QueuedMessage{
				Role:      "error",
				Content:   fmt.Sprintf("❌ Search failed: %v", err),
				Timestamp: time.Now(),
				Complete:  true,
			})
		} else {
			var content strings.Builder
			content.WriteString(fmt.Sprintf("🔍 **Search results for '%s'**\n\n", query))

			if len(matches) == 0 {
				content.WriteString("No matches found.")
			} else {
				for i, match := range matches {
					content.WriteString(fmt.Sprintf("**Result %d:**\n%s\n\n", i+1, match))
					if i >= 4 { // Limit to 5 results
						content.WriteString(fmt.Sprintf("... (%d more results)", len(matches)-5))
						break
					}
				}
			}

			m.messageQueue.Add(QueuedMessage{
				Role:      "system",
				Content:   content.String(),
				Timestamp: time.Now(),
				Complete:  true,
			})
		}
	}

	m.updateViewport()
	return *m, nil
}

func cmdRoadmap(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	roadmap := m.agent.GetRoadmap()

	if len(args) > 0 && args[0] == "summary" {
		summary, err := roadmap.GetSummary()
		if err != nil {
			m.messageQueue.Add(QueuedMessage{
				Role:      "error",
				Content:   fmt.Sprintf("❌ Failed to read roadmap: %v", err),
				Timestamp: time.Now(),
				Complete:  true,
			})
		} else {
			m.messageQueue.Add(QueuedMessage{
				Role:      "system",
				Content:   summary,
				Timestamp: time.Now(),
				Complete:  true,
			})
		}
	} else {
		content, err := roadmap.Read()
		if err != nil {
			m.messageQueue.Add(QueuedMessage{
				Role:      "error",
				Content:   fmt.Sprintf("❌ Failed to read roadmap: %v", err),
				Timestamp: time.Now(),
				Complete:  true,
			})
		} else {
			// Show first 1000 chars
			if len(content) > 1000 {
				content = content[:1000] + "\n\n... (truncated, use /roadmap summary for overview)"
			}

			m.messageQueue.Add(QueuedMessage{
				Role:      "system",
				Content:   content,
				Timestamp: time.Now(),
				Complete:  true,
			})
		}
	}

	m.updateViewport()
	return *m, nil
}

func cmdSave(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if err := m.agent.Save(); err != nil {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   fmt.Sprintf("❌ Save failed: %v", err),
			Timestamp: time.Now(),
			Complete:  true,
		})
	} else {
		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   "💾 Memory state saved successfully.",
			Timestamp: time.Now(),
			Complete:  true,
		})
	}
	m.updateViewport()
	return *m, nil
}

func cmdTelegram(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	cfg := m.agent.Config().Telegram

	var content strings.Builder
	content.WriteString("🤖 **Telegram Integration**\n\n")

	status := lipgloss.NewStyle().Foreground(ErrorColor).Render("❌ Disabled")
	if cfg.Enabled {
		status = lipgloss.NewStyle().Foreground(SuccessColor).Render("✅ Enabled")
	}
	content.WriteString(fmt.Sprintf("**Status:** %s\n", status))

	pairingStatus := "⚠️ Not Paired"
	if cfg.ChatID != 0 {
		pairingStatus = fmt.Sprintf("🔗 Paired (Chat ID: %d)", cfg.ChatID)
	}
	content.WriteString(fmt.Sprintf("**Pairing:** %s\n", pairingStatus))

	content.WriteString("\n**Setup Instructions:**\n")
	content.WriteString("1. Send /start to your bot on Telegram\n")
	content.WriteString("2. Copy the Chat ID returned\n")
	content.WriteString("3. Update .agi/config.json with the Chat ID\n")
	content.WriteString("4. Restart the agent\n")

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   content.String(),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}

func cmdModel(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		// Launch interactive picker
		m.initPicker()
		return *m, nil
	}

	// Quick switch: /model <name>
	newModel := args[0]
	reasoningEffort := ""
	if len(args) > 1 {
		reasoningEffort = args[1]
	}

	cfg := m.agent.Config()
	if err := m.agent.SwitchModel(cfg.Provider, cfg.APIBaseURL, cfg.APIKey, newModel, reasoningEffort); err != nil {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   fmt.Sprintf("Failed to switch model: %v", err),
			Timestamp: time.Now(),
			Complete:  true,
		})
	} else {
		msg := fmt.Sprintf("🧠 Switched to **%s**", newModel)
		if reasoningEffort != "" {
			msg += fmt.Sprintf(" · reasoning: %s", reasoningEffort)
		}
		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   msg,
			Timestamp: time.Now(),
			Complete:  true,
		})
	}
	m.updateViewport()
	return *m, nil
}

func cmdLogs(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	n := 20
	if len(args) > 0 {
		fmt.Sscanf(args[0], "%d", &n)
	}

	logs := m.agent.GetLogger().GetLastLines(n)

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   fmt.Sprintf("📋 **Recent Logs (%d lines)**\n\n```\n%s\n```", n, logs),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}

func cmdConfig(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) > 0 && args[0] == "reload" {
		// Reload config
		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   "🔄 Configuration reloaded from disk.",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	// Show config summary
	cfg := m.agent.Config()
	var content strings.Builder
	content.WriteString("⚙️ **Configuration Summary**\n\n")
	content.WriteString(fmt.Sprintf("**Model:** %s\n", cfg.Model))
	content.WriteString(fmt.Sprintf("**Verbose:** %v\n", cfg.UI.Verbose))
	content.WriteString(fmt.Sprintf("**Debug Tools:** %v\n", cfg.DebugTools))
	content.WriteString(fmt.Sprintf("**Heartbeat:** %ds\n", cfg.HeartbeatInterval))

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   content.String(),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}

func cmdReport(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   "📊 Generating debug report...",
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()

	// TODO: Generate comprehensive debug report
	// For now, show a placeholder

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   "📊 Debug report generation coming soon!",
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}

func cmdHelp(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) > 0 {
		// Show help for specific command
		cmd := FindCommand(args[0])
		if cmd == nil {
			m.messageQueue.Add(QueuedMessage{
				Role:      "error",
				Content:   fmt.Sprintf("Command not found: %s", args[0]),
				Timestamp: time.Now(),
				Complete:  true,
			})
			m.updateViewport()
			return *m, nil
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("**%s**\n\n", cmd.Name))
		content.WriteString(fmt.Sprintf("%s\n\n", cmd.Description))
		content.WriteString(fmt.Sprintf("**Usage:** `%s`\n", cmd.Usage))

		if len(cmd.Aliases) > 0 {
			content.WriteString(fmt.Sprintf("**Aliases:** %s\n", strings.Join(cmd.Aliases, ", ")))
		}

		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   content.String(),
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	// Show all commands
	var content strings.Builder
	content.WriteString("📚 **Available Commands**\n\n")

	categories := GetAllCommands()
	for _, cat := range categories {
		content.WriteString(fmt.Sprintf("**%s %s**\n", cat.Icon, cat.Name))
		for _, cmd := range cat.Commands {
			aliases := ""
			if len(cmd.Aliases) > 0 {
				aliases = fmt.Sprintf(" (/%s)", strings.Join(cmd.Aliases, ", /"))
			}
			content.WriteString(fmt.Sprintf("  `/%s`%s - %s\n", cmd.Name, aliases, cmd.Description))
		}
		content.WriteString("\n")
	}

	content.WriteString("💡 **Tip:** Use `/help <command>` for detailed help on a specific command.")

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   content.String(),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}

func cmdExit(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	return *m, tea.Quit
}

// Dual Session Commands

func cmdSession(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 || args[0] == "status" {
		// Show status
		var content strings.Builder
		content.WriteString("🤖 **Dual Session Status**\n\n")

		if m.dualSession.IsEnabled() {
			content.WriteString("✅ **Enabled**\n\n")

			if m.dualSession.IsRunning() {
				current, max := m.dualSession.GetProgress()
				content.WriteString("🔄 **Active Conversation**\n")
				content.WriteString(fmt.Sprintf("- Progress: %d/%d turns\n", current, max))

				stats := m.dualSession.GetStats()
				content.WriteString(fmt.Sprintf("- Total messages: %v\n", stats["total_messages"]))
				content.WriteString(fmt.Sprintf("- Agent A: %v messages\n", stats["agent_a_messages"]))
				content.WriteString(fmt.Sprintf("- Agent B: %v messages\n", stats["agent_b_messages"]))
				content.WriteString("\nUse `/conversation` to view the full log.")
			} else {
				content.WriteString("⏸️  No active conversation\n")
				content.WriteString("\nStart one with: `/debate <topic>`")
			}
		} else {
			content.WriteString("❌ **Disabled**\n\n")
			content.WriteString("Enable with: `/session on`\n\n")
			content.WriteString("**What is Dual Session?**\n")
			content.WriteString("Two AI agents can converse with each other, debating\n")
			content.WriteString("topics, exploring ideas, or solving problems together.")
		}

		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   content.String(),
			Timestamp: time.Now(),
			Complete:  true,
		})
	} else if args[0] == "on" || args[0] == "enable" {
		m.dualSession.Enable()
		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   "✅ Dual session mode enabled!\n\nNow you can use:\n- `/debate <topic>` to start a conversation\n- `/conversation` to view the log",
			Timestamp: time.Now(),
			Complete:  true,
		})
	} else if args[0] == "off" || args[0] == "disable" {
		m.dualSession.Disable()
		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   "❌ Dual session mode disabled.",
			Timestamp: time.Now(),
			Complete:  true,
		})
	} else {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   fmt.Sprintf("❌ Unknown action: %s\n\nUse: /session [on|off|status]", args[0]),
			Timestamp: time.Now(),
			Complete:  true,
		})
	}

	m.updateViewport()
	return *m, nil
}

func cmdDebate(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if !m.dualSession.IsEnabled() {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   "❌ Dual session is not enabled.\n\nEnable it with: `/session on`",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	if m.dualSession.IsRunning() {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   "❌ A conversation is already running.\n\nWait for it to finish or use `/session off` to stop.",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	if len(args) == 0 {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   "❌ Please provide a topic.\n\nUsage: `/debate <topic> [turns]`\n\nExamples:\n- `/debate artificial consciousness`\n- `/debate best programming language 10`",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	// Parse arguments
	topic := strings.Join(args, " ")
	turns := 20 // default

	// Check if last arg is a number (turns)
	if len(args) > 1 {
		if lastArg, err := fmt.Sscanf(args[len(args)-1], "%d", &turns); err == nil && lastArg == 1 {
			topic = strings.Join(args[:len(args)-1], " ")
		}
	}

	// Set max turns
	m.dualSession.SetMaxTurns(turns)

	// Set multi-window for dual session (one window per agent)
	m.dualSession.SetMultiWindow(m.multiWindow)

	// Create initial prompt
	initialPrompt := fmt.Sprintf("Let's have a thoughtful discussion about: %s\n\nShare your perspective and insights.", topic)

	// Start conversation
	if err := m.dualSession.StartConversation(initialPrompt); err != nil {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   fmt.Sprintf("❌ Failed to start conversation: %v", err),
			Timestamp: time.Now(),
			Complete:  true,
		})
	} else {
		var content strings.Builder
		content.WriteString(fmt.Sprintf("🤖 **Starting debate on: %s**\n\n", topic))
		content.WriteString(fmt.Sprintf("Max turns: %d\n\n", turns))
		content.WriteString("💡 **Tip:** Use `/conversation` to open separate windows for each agent!\n")
		content.WriteString("   🔵 Window 1 = Agent A only\n")
		content.WriteString("   🟢 Window 2 = Agent B only\n\n")
		content.WriteString("   The debate will run in the background while you continue working.")

		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   content.String(),
			Timestamp: time.Now(),
			Complete:  true,
		})
	}

	m.updateViewport()
	return *m, nil
}

func cmdConversation(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if !m.dualSession.IsEnabled() {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   "❌ Dual session is not enabled.",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	// Check if there's a running or past conversation
	isRunning := m.dualSession.IsRunning()
	log := m.dualSession.GetConversationLog()

	if len(log) == 0 && !isRunning {
		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   "📝 No conversation log yet.\n\nStart one with: `/debate <topic>`",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	// If conversation is running, try to open multi-window (one per agent)
	if isRunning {
		// Check if windows are already open
		if m.multiWindow.IsEnabled() {
			m.messageQueue.Add(QueuedMessage{
				Role:      "system",
				Content:   "📺 **Agent windows already open!**\n\nThe debate is being shown in separate terminal windows (one per agent).",
				Timestamp: time.Now(),
				Complete:  true,
			})
			m.updateViewport()
			return *m, nil
		}

		// Try to open multi-window (one terminal per agent)
		speakers := []string{"Agent A", "Agent B"}
		if err := m.multiWindow.OpenWindows(speakers); err != nil {
			// Fallback to TUI view if multi-window fails
			m.messageQueue.Add(QueuedMessage{
				Role:      "error",
				Content:   fmt.Sprintf("❌ Failed to open agent windows: %v\n\nFalling back to TUI view...", err),
				Timestamp: time.Now(),
				Complete:  true,
			})

			// Enable live view in TUI
			m.conversationView.Enable()

			// Show header with instructions
			header := formatConversationHeader(m)
			m.messageQueue.Add(QueuedMessage{
				Role:      "system",
				Content:   header,
				Timestamp: time.Now(),
				Complete:  true,
			})

			// Show existing messages
			for _, msg := range log {
				var icon string
				var label string
				if msg.Speaker == "Agent A" {
					icon = "🔵"
					label = "Agent A"
				} else {
					icon = "🟢"
					label = "Agent B"
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
			return *m, checkConversationUpdates(m)
		}

		// Multi-window opened successfully
		// Write existing messages to the appropriate windows
		for _, msg := range log {
			_ = m.multiWindow.WriteMessage(msg.Speaker, msg.Content, msg.Turn)
		}

		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   "✅ **Agent windows opened!**\n\n📺 The debate is now shown in TWO separate terminal windows:\n\n🔵 **Window 1**: Agent A only\n🟢 **Window 2**: Agent B only\n\nYou can continue working here while watching the debate in real-time.",
			Timestamp: time.Now(),
			Complete:  true,
		})

		m.updateViewport()
		return *m, nil
	}

	// Conversation is complete, show full log
	formatted := m.dualSession.FormatConversation()

	// Add stats
	stats := m.dualSession.GetStats()
	var statsStr strings.Builder
	statsStr.WriteString("\n")
	statsStr.WriteString(strings.Repeat("═", 60) + "\n")
	statsStr.WriteString("📊 **Statistics**\n\n")
	statsStr.WriteString(fmt.Sprintf("- Total messages: %v\n", stats["total_messages"]))
	statsStr.WriteString(fmt.Sprintf("- Agent A: %v messages\n", stats["agent_a_messages"]))
	statsStr.WriteString(fmt.Sprintf("- Agent B: %v messages\n", stats["agent_b_messages"]))
	statsStr.WriteString(fmt.Sprintf("- Current turn: %v/%v\n", stats["current_turn"], stats["max_turns"]))
	statsStr.WriteString(fmt.Sprintf("- Total characters: %v\n", stats["total_chars"]))
	statsStr.WriteString("- Status: ⏸️ Complete\n")

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   formatted + statsStr.String(),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}

func cmdStop(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if !m.dualSession.IsRunning() {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   "❌ No active conversation to stop.",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	// Stop the conversation
	m.dualSession.StopConversation()

	// Close multi-window if open
	if m.multiWindow.IsEnabled() {
		_ = m.multiWindow.CloseWindows()
	}

	// Disable live view
	if m.conversationView.IsEnabled() {
		m.conversationView.Disable()
	}

	// Get final stats
	stats := m.dualSession.GetStats()

	var content strings.Builder
	content.WriteString("⏹️ **Conversation Stopped**\n\n")
	content.WriteString("The debate has been ended early.\n\n")
	content.WriteString("**Final Statistics:**\n")
	content.WriteString(fmt.Sprintf("- Total messages: %v\n", stats["total_messages"]))
	content.WriteString(fmt.Sprintf("- Turns completed: %v/%v\n", stats["current_turn"], stats["max_turns"]))
	content.WriteString(fmt.Sprintf("- Agent A: %v messages\n", stats["agent_a_messages"]))
	content.WriteString(fmt.Sprintf("- Agent B: %v messages\n", stats["agent_b_messages"]))
	content.WriteString("\nUse `/conversation` to view the full log.")

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   content.String(),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}

func cmdPipeline(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	var msg string
	if len(args) == 0 || strings.ToLower(args[0]) == "status" {
		if m.agent.PipelineEnabled() {
			msg = "🤖 Multi-agent pipeline: " + lipgloss.NewStyle().Foreground(SuccessColor).Render("ON") +
				"\n   Planner → Researcher → Executor → Critic"
		} else {
			msg = "🤖 Multi-agent pipeline: " + lipgloss.NewStyle().Foreground(ErrorColor).Render("OFF") +
				"\n   Use /pipeline on to activate."
		}
	} else {
		arg := strings.ToLower(args[0])
		switch arg {
		case "on", "true", "1":
			m.agent.EnablePipeline(true)
			msg = "🤖 Multi-agent pipeline " + lipgloss.NewStyle().Foreground(SuccessColor).Render("ENABLED") +
				"\n   Each message will go through: Planner → Researcher → Executor → Critic"
		case "off", "false", "0":
			m.agent.EnablePipeline(false)
			msg = "🤖 Multi-agent pipeline " + lipgloss.NewStyle().Foreground(ErrorColor).Render("DISABLED") +
				"\n   Returning to single-agent mode."
		default:
			msg = "Usage: /pipeline [on|off|status]"
		}
	}

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   msg,
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}
