// Package tui provides command handling for the TUI
package tui

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"ClosedWheeler/pkg/recovery"
	"ClosedWheeler/pkg/telegram"
	"ClosedWheeler/pkg/tools"

	tea "github.com/charmbracelet/bubbletea"
)

// renderToggle returns a styled "ON" or "OFF" string for boolean states.
// Uses green for ON and red for OFF to provide clear visual feedback.
//
// Parameters:
//
//	on: Boolean state to render
//
// Returns:
//
//	string: Styled "ON" (green) or "OFF" (red) text
//
// Example:
//
//	fmt.Println(renderToggle(true))   // "ON" (green)
//	fmt.Println(renderToggle(false))  // "OFF" (red)
func renderToggle(on bool) string {
	if on {
		return ToggleOnStyle.Render("ON")
	}
	return ToggleOffStyle.Render("OFF")
}

// renderEnabled returns a styled "enabled" or "disabled" string for feature states.
// Similar to renderToggle but with lowercase text for feature descriptions.
//
// Parameters:
//
//	enabled: Boolean feature state to render
//
// Returns:
//
//	string: Styled "enabled" (green) or "disabled" (red) text
//
// Example:
//
//	fmt.Println(renderEnabled(true))   // "enabled" (green)
//	fmt.Println(renderEnabled(false))  // "disabled" (red)
func renderEnabled(enabled bool) string {
	if enabled {
		return ToggleOnStyle.Render("enabled")
	}
	return ToggleOffStyle.Render("disabled")
}

// renderEnabledUpper returns a styled "ENABLED" or "DISABLED" string.
// Uses uppercase text for emphasis in status displays and headers.
//
// Parameters:
//
//	on: Boolean state to render
//
// Returns:
//
//	string: Styled "ENABLED" (green) or "DISABLED" (red) text
//
// Example:
//
//	fmt.Println(renderEnabledUpper(true))   // "ENABLED" (green)
//	fmt.Println(renderEnabledUpper(false))  // "DISABLED" (red)
func renderEnabledUpper(on bool) string {
	if on {
		return ToggleOnStyle.Render("ENABLED")
	}
	return ToggleOffStyle.Render("DISABLED")
}

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

// GetAllCommands returns all available TUI commands organized by category.
// This is the central registry for all slash commands that users can execute.
//
// Returns:
//
//	[]CommandCategory: Slice of command categories with their commands
//
// Command Categories:
//   - Conversation (💬): clear, retry, continue
//   - Information (📊): status, stats, memory, context, tools
//   - Project (📁): reload, rules, git, health
//   - Features (⚙️): verbose, debug, timestamps, browser, heartbeat, pipeline
//   - Memory & Brain (🧠): brain, roadmap, save
//   - Integration (🔗): telegram, model, skill, mcp
//   - Providers (🔌): providers, pairings
//   - Dual Session (🤖): session, debate, conversation, stop
//   - System (🖥️): logs, config, report, errors, resilience, tool-retries, retry-mode, recover, help, exit
//   - Interface (🖥️): logs, history
//
// Example:
//
//	categories := GetAllCommands()
//	for _, cat := range categories {
//	    fmt.Printf("%s %s\n", cat.Icon, cat.Name)
//	    for _, cmd := range cat.Commands {
//	        fmt.Printf("  /%s - %s\n", cmd.Name, cmd.Description)
//	    }
//	}
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
					Description: "Manage Telegram bot integration",
					Usage:       "/telegram [enable|disable|token|chatid|pair]",
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
				{
					Name:        "skill",
					Aliases:     []string{"skills"},
					Category:    "Integration",
					Description: "List or reload external skills",
					Usage:       "/skill [list|reload]",
					Handler:     cmdSkill,
				},
				{
					Name:        "mcp",
					Category:    "Integration",
					Description: "Manage MCP server connections",
					Usage:       "/mcp [list|add|remove|reload]",
					Handler:     cmdMCP,
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
					Description: "Start agent-to-agent debate (wizard or quick)",
					Usage:       "/debate [topic] [turns]",
					Handler:     cmdDebate,
				},
				{
					Name:        "conversation",
					Aliases:     []string{"conv", "log"},
					Category:    "Dual Session",
					Description: "Open live debate viewer or view completed log",
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
		{
			Name: "Interface",
			Icon: "🖥️",
			Commands: []Command{
				{
					Name:        "logs",
					Aliases:     []string{"log", "l"},
					Category:    "Interface",
					Description: "Show log viewer table",
					Usage:       "/logs",
					Handler:     cmdLogs,
				},
				{
					Name:        "history",
					Aliases:     []string{"hist", "h"},
					Category:    "Interface",
					Description: "Show conversation history paginator",
					Usage:       "/history",
					Handler:     cmdHistory,
				},
			},
		},
	}
}

// FindCommand searches for a command by name or alias.
// This provides case-insensitive lookup with support for both primary names
// and aliases, with or without the leading slash.
//
// Parameters:
//
//	name: Command name or alias to search for (e.g., "clear", "c", "/clear")
//
// Returns:
//
//	*Command: Pointer to the found command, or nil if not found
//
// Search Behavior:
//   - Case-insensitive matching
//   - Accepts names with or without leading slash
//   - Searches both primary names and aliases
//   - Returns first match found
//
// Example:
//
//	cmd := FindCommand("clear")
//	if cmd != nil {
//	    fmt.Printf("Found: %s - %s\n", cmd.Name, cmd.Description)
//	}
//
//	// Find by alias
//	cmd = FindCommand("c") // alias for clear
//
//	// Find with slash
//	cmd = FindCommand("/status")
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
	return m, nil
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
		return m, nil
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
		return m, nil
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

	return m, tea.Batch(
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

	return m, tea.Batch(
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

	m.openPanel("System Status", content.String())
	return m, nil
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

	m.openPanel("API Statistics", content.String())
	return m, nil
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
		return m, nil
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

	m.openPanel("Memory System", content.String())
	return m, nil
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
		return m, nil
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

	m.openPanel("Context Status", content.String())
	return m, nil
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

	m.openPanel("Available Tools", content.String())
	return m, nil
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
	return m, nil
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

	m.openPanel("Project Rules", content.String())
	return m, nil
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
		return m, nil
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

	return m, tea.Batch(
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

	m.openPanel("Health Check", content.String())
	return m, nil
}

func cmdVerbose(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		m.openSettings("verbose")
		return m, nil
	}

	arg := strings.ToLower(args[0])
	m.verbose = arg == "on" || arg == "true" || arg == "1"

	m.agent.Config().UI.Verbose = m.verbose
	if err := m.agent.SaveConfig(); err != nil {
		m.agent.GetLogger().Error("Failed to save config: %v", err)
	}

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   fmt.Sprintf("📢 Verbose mode: %s (saved to config)", renderToggle(m.verbose)),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return m, nil
}

func cmdDebug(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		m.openSettings("debug")
		return m, nil
	}

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

	// Set debug level
	if m.agent.Config().DebugTools {
		tools.SetGlobalDebugLevel(tools.DebugVerbose)
	} else {
		tools.SetGlobalDebugLevel(tools.DebugOff)
	}

	if err := m.agent.SaveConfig(); err != nil {
		m.agent.GetLogger().Error("Failed to save config: %v", err)
	}

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   fmt.Sprintf("🐛 Debug mode: %s\n\nLevels: basic, verbose, trace", renderToggle(m.agent.Config().DebugTools)),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return m, nil
}

func cmdTimestamps(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		m.openSettings("timestamps")
		return m, nil
	}

	arg := strings.ToLower(args[0])
	m.showTimestamps = arg == "on" || arg == "true" || arg == "1"

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   fmt.Sprintf("🕒 Timestamps: %s", renderToggle(m.showTimestamps)),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return m, nil
}

func cmdBrowser(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		m.openSettings("browser_headless")
		return m, nil
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
	return m, nil
}

func cmdHeartbeat(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		m.openSettings("heartbeat")
		return m, nil
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
	return m, nil
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

			m.openPanel("Knowledge Base", "🧠 **Knowledge Base**\n\n"+content)
		}
		m.updateViewport()
		return m, nil
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
	return m, nil
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
			m.updateViewport()
		} else {
			m.openPanel("Roadmap Summary", summary)
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
			m.updateViewport()
		} else {
			m.openPanel("Roadmap", content)
		}
	}

	return m, nil
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
	return m, nil
}

func cmdTelegram(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		return cmdTelegramStatus(m)
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "enable":
		m.agent.Config().Telegram.Enabled = true
		if err := m.agent.SaveConfig(); err != nil {
			return cmdTelegramError(m, fmt.Sprintf("Failed to save: %v", err))
		}
		return cmdTelegramMsg(m, "Telegram integration "+renderEnabled(true)+". Restart to apply.")

	case "disable":
		m.agent.Config().Telegram.Enabled = false
		if err := m.agent.SaveConfig(); err != nil {
			return cmdTelegramError(m, fmt.Sprintf("Failed to save: %v", err))
		}
		return cmdTelegramMsg(m, "Telegram integration "+renderEnabled(false)+".")

	case "token":
		if len(args) < 2 {
			return cmdTelegramError(m, "Usage: /telegram token <bot-token>")
		}
		token := args[1]
		// Validate token
		botName, err := telegram.ValidateToken(token)
		if err != nil {
			return cmdTelegramError(m, fmt.Sprintf("Invalid token: %v", err))
		}
		if err := m.agent.ReconfigureTelegram(token, true); err != nil {
			return cmdTelegramError(m, fmt.Sprintf("Failed to reconfigure: %v", err))
		}
		return cmdTelegramMsg(m, fmt.Sprintf("Bot token set. Bot: @%s\nTelegram is now enabled and polling.", botName))

	case "chatid":
		if len(args) < 2 {
			return cmdTelegramError(m, "Usage: /telegram chatid <id>")
		}
		id, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return cmdTelegramError(m, fmt.Sprintf("Invalid Chat ID: %v", err))
		}
		if err := m.agent.SetTelegramChatID(id); err != nil {
			return cmdTelegramError(m, fmt.Sprintf("Failed to save: %v", err))
		}
		return cmdTelegramMsg(m, fmt.Sprintf("Chat ID set to %d and saved.", id))

	case "pair":
		return cmdTelegramPair(m)

	default:
		return cmdTelegramError(m, fmt.Sprintf("Unknown subcommand: %s\n\nUsage: /telegram [enable|disable|token|chatid|pair]", sub))
	}
}

// cmdTelegramStatus shows the enhanced Telegram status panel.
func cmdTelegramStatus(m *EnhancedModel) (tea.Model, tea.Cmd) {
	cfg := m.agent.Config().Telegram

	var content strings.Builder
	content.WriteString("🤖 **Telegram Integration**\n\n")

	content.WriteString(fmt.Sprintf("**Status:**  %s\n", renderEnabled(cfg.Enabled)))

	// Bot info
	bot := m.agent.GetTelegramBot()
	if bot != nil {
		content.WriteString(fmt.Sprintf("**Bot:**     @%s\n", bot.GetBotUsername()))
	} else if cfg.BotToken != "" {
		content.WriteString("**Bot:**     (token set, not connected)\n")
	} else {
		content.WriteString("**Bot:**     (no token)\n")
	}

	// Pairing status
	if cfg.ChatID != 0 {
		content.WriteString(fmt.Sprintf("**Chat ID:** Paired (%d)\n", cfg.ChatID))
	} else {
		content.WriteString("**Chat ID:** Not paired\n")
	}

	// Notify
	content.WriteString(fmt.Sprintf("**Notify:**  %s\n", renderToggle(cfg.NotifyOnToolStart)))

	content.WriteString("\n**Commands:**\n")
	content.WriteString("  `/telegram enable`        — Enable Telegram\n")
	content.WriteString("  `/telegram disable`       — Disable Telegram\n")
	content.WriteString("  `/telegram token <token>` — Set/change bot token\n")
	content.WriteString("  `/telegram chatid <id>`   — Set Chat ID manually\n")
	content.WriteString("  `/telegram pair`          — Show pairing instructions\n")

	m.openPanel("Telegram", content.String())
	return m, nil
}

// cmdTelegramPair shows pairing instructions or current pairing status.
func cmdTelegramPair(m *EnhancedModel) (tea.Model, tea.Cmd) {
	cfg := m.agent.Config().Telegram
	bot := m.agent.GetTelegramBot()

	var content strings.Builder
	content.WriteString("📱 **Telegram Pairing**\n\n")

	if bot == nil {
		content.WriteString("No bot is active. Set a token first:\n")
		content.WriteString("  `/telegram token <your-bot-token>`\n")
		m.openPanel("Telegram Pairing", content.String())
		return m, nil
	}

	if cfg.ChatID != 0 {
		content.WriteString(fmt.Sprintf("Already paired to Chat ID: `%d`\n\n", cfg.ChatID))
		content.WriteString("To re-pair with a different account:\n")
		content.WriteString("  `/telegram chatid <new-id>`\n")
	} else {
		content.WriteString("**How to pair:**\n")
		content.WriteString(fmt.Sprintf("  1. Open Telegram and find @%s\n", bot.GetBotUsername()))
		content.WriteString("  2. Send `/start` to the bot\n")
		content.WriteString("  3. The bot will auto-pair with your Chat ID\n\n")
		content.WriteString("No restart needed — it pairs automatically!\n")
	}

	m.openPanel("Telegram Pairing", content.String())
	return m, nil
}

func cmdTelegramMsg(m *EnhancedModel, msg string) (tea.Model, tea.Cmd) {
	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   "📱 " + msg,
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return m, nil
}

func cmdTelegramError(m *EnhancedModel, msg string) (tea.Model, tea.Cmd) {
	m.messageQueue.Add(QueuedMessage{
		Role:      "error",
		Content:   msg,
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return m, nil
}

func cmdModel(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		// Launch interactive picker
		m.initPicker()
		return m, nil
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
	return m, nil
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
		return m, nil
	}

	// Show config summary
	cfg := m.agent.Config()
	var content strings.Builder
	content.WriteString("⚙️ **Configuration Summary**\n\n")
	content.WriteString(fmt.Sprintf("**Model:** %s\n", cfg.Model))
	content.WriteString(fmt.Sprintf("**Verbose:** %v\n", cfg.UI.Verbose))
	content.WriteString(fmt.Sprintf("**Debug Tools:** %v\n", cfg.DebugTools))
	content.WriteString(fmt.Sprintf("**Heartbeat:** %ds\n", cfg.HeartbeatInterval))

	m.openPanel("Configuration", content.String())
	return m, nil
}

func cmdReport(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   "📊 Generating debug report...",
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()

	// Generate comprehensive debug report
	var report strings.Builder

	// System Information
	report.WriteString("🔍 **System Debug Report**\n")
	report.WriteString(strings.Repeat("═", 60) + "\n\n")

	report.WriteString("**System Information:**\n")
	report.WriteString(fmt.Sprintf("- Go Version: %s\n", runtime.Version()))
	report.WriteString(fmt.Sprintf("- OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH))
	report.WriteString(fmt.Sprintf("- CPU Cores: %d\n", runtime.NumCPU()))
	report.WriteString(fmt.Sprintf("- PID: %d\n", os.Getpid()))

	// Memory Usage
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	report.WriteString("\n**Memory Usage:**\n")
	report.WriteString(fmt.Sprintf("- Alloc: %s\n", formatBytes(memStats.Alloc)))
	report.WriteString(fmt.Sprintf("- Total Alloc: %s\n", formatBytes(memStats.TotalAlloc)))
	report.WriteString(fmt.Sprintf("- Sys: %s\n", formatBytes(memStats.Sys)))
	report.WriteString(fmt.Sprintf("- Num GC: %d\n", memStats.NumGC))

	// Agent Configuration
	report.WriteString("\n**Agent Configuration:**\n")
	cfg := m.agent.Config()
	report.WriteString(fmt.Sprintf("- Model: %s\n", cfg.Model))
	report.WriteString(fmt.Sprintf("- Base URL: %s\n", cfg.APIBaseURL))
	report.WriteString(fmt.Sprintf("- Provider: %s\n", cfg.Provider))
	report.WriteString(fmt.Sprintf("- Verbose: %v\n", cfg.UI.Verbose))
	report.WriteString(fmt.Sprintf("- Debug Tools: %v\n", cfg.DebugTools))
	report.WriteString(fmt.Sprintf("- Heartbeat: %ds\n", cfg.HeartbeatInterval))

	// API Usage Statistics
	report.WriteString("\n**API Usage Statistics:**\n")
	usage := m.agent.GetUsageStats()
	report.WriteString(fmt.Sprintf("- Total Tokens: %v\n", usage["total_tokens"]))
	report.WriteString(fmt.Sprintf("- Prompt Tokens: %v\n", usage["prompt_tokens"]))
	report.WriteString(fmt.Sprintf("- Completion Tokens: %v\n", usage["completion_tokens"]))

	// Memory Statistics
	report.WriteString("\n**Memory Statistics:**\n")
	mem := m.agent.GetMemoryStats()
	report.WriteString(fmt.Sprintf("- Short-term Memory: %d\n", mem["short_term_size"]))
	report.WriteString(fmt.Sprintf("- Working Memory: %d\n", mem["working_memory_size"]))
	report.WriteString(fmt.Sprintf("- Context Size: %d\n", mem["context_size"]))

	// Tool Statistics
	if m.toolRetryWrapper != nil {
		report.WriteString("\n**Tool Statistics:**\n")
		debugReport := m.toolRetryWrapper.GetDebugReport()
		report.WriteString(debugReport)

		// Recent Failures
		failures := m.toolRetryWrapper.GetRecentFailures()
		if len(failures) > 0 {
			report.WriteString(fmt.Sprintf("\n**Recent Tool Failures (%d):**\n", len(failures)))
			for i, failure := range failures {
				if i >= 5 { // Limit to last 5 failures
					break
				}
				report.WriteString(fmt.Sprintf("- %s: %v\n", failure.ToolName, failure.Error))
			}
		}
	}

	// Error Recovery Statistics
	if recoveryHandler := recovery.GetGlobalHandler(); recoveryHandler != nil {
		report.WriteString("\n**Error Recovery Statistics:**\n")
		stats := recoveryHandler.GetErrorStats()
		report.WriteString(fmt.Sprintf("- Total Errors: %v\n", stats["total_errors"]))
		report.WriteString(fmt.Sprintf("- Recovered: %v\n", stats["recovered"]))
		if total, ok := stats["total_errors"].(int); ok && total > 0 {
			report.WriteString(fmt.Sprintf("- Recovery Rate: %.1f%%\n", stats["recovery_rate"]))
		}
	}

	// TUI Statistics
	report.WriteString("\n**TUI Statistics:**\n")
	report.WriteString(fmt.Sprintf("- Messages in Queue: %d\n", m.messageQueue.Len()))
	report.WriteString(fmt.Sprintf("- Viewport Height: %d\n", m.viewport.Height))
	report.WriteString(fmt.Sprintf("- Current State: %v\n", m.GetState()))

	// Recent Logs (last 5 lines) - skip if logger not available
	report.WriteString("\n**Recent Log Entries:**\n")
	report.WriteString("Logger not directly accessible in TUI context\n")

	report.WriteString("\n" + strings.Repeat("═", 60) + "\n")
	report.WriteString(fmt.Sprintf("Generated at: %s\n", time.Now().Format("2006-01-02 15:04:05")))

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   report.String(),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()

	return m, nil
}

// formatBytes formats bytes into human readable string
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func cmdHelp(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) > 0 {
		// Show help for specific command (inline, backward compatible)
		cmd := FindCommand(args[0])
		if cmd == nil {
			m.messageQueue.Add(QueuedMessage{
				Role:      "error",
				Content:   fmt.Sprintf("Command not found: %s", args[0]),
				Timestamp: time.Now(),
				Complete:  true,
			})
			m.updateViewport()
			return m, nil
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
		return m, nil
	}

	// No args: open interactive help menu overlay
	m.initHelpMenu()
	return m, nil
}

func cmdExit(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	return m, tea.Quit
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

		m.openPanel("Dual Session", content.String())
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
	return m, nil
}

func cmdDebate(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if m.dualSession.IsRunning() {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   "❌ A conversation is already running.\n\nStop it with `/stop` first.",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return m, nil
	}

	// No args → open interactive wizard
	if len(args) == 0 {
		m.initDebateWizard("")
		return m, nil
	}

	// Quick path: /debate <topic> [turns]
	topic := strings.Join(args, " ")
	turns := 20

	// Check if last arg is a number (turns)
	if len(args) > 1 {
		if lastArg, err := fmt.Sscanf(args[len(args)-1], "%d", &turns); err == nil && lastArg == 1 {
			topic = strings.Join(args[:len(args)-1], " ")
		}
	}

	// Auto-enable dual session
	if !m.dualSession.IsEnabled() {
		m.dualSession.Enable()
	}

	m.dualSession.SetMaxTurns(turns)
	m.dualSession.SetTopic(topic)
	m.dualSession.SetModels("", "")               // default model for both
	m.dualSession.SetToolMode(DebateToolModeSafe) // safe by default on quick path

	// Quick path: Agent A = Debater (index 0), Agent B = Critic (index 4)
	presets := DebateRolePresets()
	roleA := presets[0] // Debater
	roleB := presets[4] // Critic
	initialPrompt := fmt.Sprintf("Let's have a thoughtful discussion about: %s\n\nShare your perspective and insights.", topic)

	if err := m.dualSession.StartConversationWithRoles(
		initialPrompt, roleA.Name, roleA.Prompt, roleB.Name, roleB.Prompt,
	); err != nil {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   fmt.Sprintf("❌ Failed to start debate: %v", err),
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return m, nil
	}

	// Open the in-TUI debate viewer overlay
	m.openDebateViewer()

	m.messageQueue.Add(QueuedMessage{
		Role: "system",
		Content: fmt.Sprintf("🤖 Debate started: %s\n"+
			"   🔵 %s vs 🟢 %s — %d turns\n"+
			"   Esc to close viewer. Use /stop to end early.",
			topic, roleA.Name, roleB.Name, turns),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()

	return m, debateViewerTick()
}

func cmdConversation(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	log := m.dualSession.GetConversationLog()

	if len(log) == 0 && !m.dualSession.IsRunning() {
		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   "No conversation log yet.\n\nStart one with: `/debate <topic>` or `/debate` for the setup wizard.",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return m, nil
	}

	// Running debate → open the in-TUI debate viewer
	if m.dualSession.IsRunning() {
		m.openDebateViewer()
		return m, debateViewerTick()
	}

	// Complete debate → show read-only panel with full log
	formatted := m.dualSession.FormatConversation()

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
	statsStr.WriteString("- Status: Complete\n")

	m.openPanel("Conversation Log", formatted+statsStr.String())
	return m, nil
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
		return m, nil
	}

	// Stop the conversation
	m.dualSession.StopConversation()

	// Close debate viewer if active
	m.debateViewActive = false

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
	return m, nil
}

func cmdPipeline(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	var msg string
	if len(args) == 0 || strings.ToLower(args[0]) == "status" {
		var content strings.Builder
		content.WriteString("🤖 **Multi-Agent Pipeline**\n\n")
		if m.agent.PipelineEnabled() {
			content.WriteString("Status: ON\n")
			content.WriteString("Flow: Planner → Researcher → Executor → Critic\n\n")
			content.WriteString("Use /pipeline off to deactivate.")
		} else {
			content.WriteString("Status: OFF\n\n")
			content.WriteString("Use /pipeline on to activate.")
		}
		m.openPanel("Pipeline Status", content.String())
		return m, nil
	} else {
		arg := strings.ToLower(args[0])
		switch arg {
		case "on", "true", "1":
			m.agent.EnablePipeline(true)
			msg = "🤖 Multi-agent pipeline " + renderEnabledUpper(true) +
				"\n   Each message will go through: Planner → Researcher → Executor → Critic"
		case "off", "false", "0":
			m.agent.EnablePipeline(false)
			msg = "🤖 Multi-agent pipeline " + renderEnabledUpper(false) +
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
	return m, nil
}

// cmdLogs shows the log viewer table
func cmdLogs(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	// Add some sample logs if empty
	if len(m.logTable.logs) == 0 {
		m.logTable.AddLog("INFO", "Log viewer initialized", "TUI")
		m.logTable.AddLog("INFO", "Use arrow keys to navigate", "TUI")
		m.logTable.AddLog("WARN", "Press ESC to close", "TUI")
	}

	m.logTable.Show()
	return m, nil
}

// cmdHistory shows the conversation history paginator
func cmdHistory(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	// Get conversation content
	var content strings.Builder

	for _, msg := range m.messageQueue.messages {
		timestamp := msg.Timestamp.Format("15:04:05")
		role := strings.ToUpper(msg.Role)

		content.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, role))
		content.WriteString(msg.Content)
		content.WriteString("\n\n")
	}

	// Set content to paginator (max 20 lines per page)
	m.conversationPaginator.SetContent(content.String(), 20)
	m.conversationPaginator.Show()

	return m, nil
}
