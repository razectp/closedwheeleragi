package tui

import (
	"fmt"
	"strings"
	"time"

	"ClosedWheeler/pkg/config"
	agimcp "ClosedWheeler/pkg/mcp"

	tea "github.com/charmbracelet/bubbletea"
)

// cmdSkill handles /skill [list|reload]
func cmdSkill(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	sub := "list"
	if len(args) > 0 {
		sub = strings.ToLower(args[0])
	}

	switch sub {
	case "list", "ls":
		sm := m.agent.GetSkillManager()
		skills := sm.ListSkills()

		var content strings.Builder
		content.WriteString("**Skills**\n\n")
		content.WriteString(fmt.Sprintf("**Directory:** `%s`\n\n", sm.SkillsDir()))

		if len(skills) == 0 {
			content.WriteString("No skills loaded.\n\n")
			content.WriteString("To add a skill, create a folder in the skills directory\n")
			content.WriteString("with a `skill.json` metadata file and a script.\n\n")
			content.WriteString("Example structure:\n")
			content.WriteString("```\n")
			content.WriteString(".agi/skills/\n")
			content.WriteString("  my-skill/\n")
			content.WriteString("    skill.json\n")
			content.WriteString("    run.cmd\n")
			content.WriteString("```\n")
		} else {
			content.WriteString(fmt.Sprintf("**Loaded:** %d skill(s)\n\n", len(skills)))
			for i, s := range skills {
				content.WriteString(fmt.Sprintf("%d. **%s** (`%s/%s`)\n", i+1, s.Name, s.Folder, s.Script))
				if s.Description != "" {
					content.WriteString(fmt.Sprintf("   %s\n", s.Description))
				}
			}
		}

		m.openPanel("Skills", content.String())
		return m, nil

	case "reload":
		sm := m.agent.GetSkillManager()
		if err := sm.LoadSkills(); err != nil {
			m.messageQueue.Add(QueuedMessage{
				Role:      "error",
				Content:   fmt.Sprintf("Failed to reload skills: %v", err),
				Timestamp: time.Now(),
				Complete:  true,
			})
		} else {
			count := sm.Count()
			m.messageQueue.Add(QueuedMessage{
				Role:      "system",
				Content:   fmt.Sprintf("Skills reloaded. %d skill(s) loaded.", count),
				Timestamp: time.Now(),
				Complete:  true,
			})
		}
		m.updateViewport()
		return m, nil

	default:
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   fmt.Sprintf("Unknown subcommand: %s\n\nUsage: /skill [list|reload]", sub),
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return m, nil
	}
}

// cmdMCP handles /mcp [list|add|remove|reload]
func cmdMCP(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	sub := "list"
	if len(args) > 0 {
		sub = strings.ToLower(args[0])
	}

	mcpMgr := m.agent.GetMCPManager()

	switch sub {
	case "list", "ls", "status":
		servers := mcpMgr.ListServers()

		var content strings.Builder
		content.WriteString("**MCP Servers**\n\n")

		if len(servers) == 0 {
			content.WriteString("No MCP servers configured.\n\n")
			content.WriteString("**Add via config:**\n")
			content.WriteString("Edit `.agi/config.json` and add to `mcp_servers`:\n")
			content.WriteString("```json\n")
			content.WriteString("\"mcp_servers\": [\n")
			content.WriteString("  {\n")
			content.WriteString("    \"name\": \"my-server\",\n")
			content.WriteString("    \"transport\": \"stdio\",\n")
			content.WriteString("    \"command\": \"npx\",\n")
			content.WriteString("    \"args\": [\"-y\", \"@modelcontextprotocol/server-filesystem\", \".\"],\n")
			content.WriteString("    \"enabled\": true\n")
			content.WriteString("  }\n")
			content.WriteString("]\n")
			content.WriteString("```\n\n")
			content.WriteString("**Or add via command:**\n")
			content.WriteString("`/mcp add <name> stdio <command> [args...]`\n")
			content.WriteString("`/mcp add <name> sse <url>`\n")
		} else {
			content.WriteString(fmt.Sprintf("**Configured:** %d server(s) | **Tools:** %d\n\n", mcpMgr.ServerCount(), mcpMgr.ToolCount()))
			for _, s := range servers {
				status := ToggleOffStyle.Render("disconnected")
				if s.Connected {
					status = ToggleOnStyle.Render("connected")
				}
				content.WriteString(fmt.Sprintf("**%s** [%s] %s\n", s.Name, s.Transport, status))
				if s.Error != "" {
					content.WriteString(fmt.Sprintf("  Error: %s\n", s.Error))
				}
				if len(s.Tools) > 0 {
					content.WriteString(fmt.Sprintf("  Tools: %s\n", strings.Join(s.Tools, ", ")))
				}
				content.WriteString("\n")
			}
		}

		content.WriteString("**Commands:**\n")
		content.WriteString("  `/mcp list`                         - Show servers\n")
		content.WriteString("  `/mcp add <name> stdio <cmd> [args]` - Add stdio server\n")
		content.WriteString("  `/mcp add <name> sse <url>`          - Add SSE server\n")
		content.WriteString("  `/mcp remove <name>`                 - Remove server\n")
		content.WriteString("  `/mcp reload`                        - Reconnect all\n")

		m.openPanel("MCP Servers", content.String())
		return m, nil

	case "add":
		return cmdMCPAdd(m, args[1:])

	case "remove", "rm", "delete":
		if len(args) < 2 {
			m.messageQueue.Add(QueuedMessage{
				Role:      "error",
				Content:   "Usage: /mcp remove <name>",
				Timestamp: time.Now(),
				Complete:  true,
			})
			m.updateViewport()
			return m, nil
		}

		name := args[1]
		if err := mcpMgr.RemoveServer(name); err != nil {
			m.messageQueue.Add(QueuedMessage{
				Role:      "error",
				Content:   fmt.Sprintf("Failed to remove server: %v", err),
				Timestamp: time.Now(),
				Complete:  true,
			})
		} else {
			// Update config
			saveMCPConfig(m)
			m.messageQueue.Add(QueuedMessage{
				Role:      "system",
				Content:   fmt.Sprintf("MCP server %q removed and config saved.", name),
				Timestamp: time.Now(),
				Complete:  true,
			})
		}
		m.updateViewport()
		return m, nil

	case "reload", "reconnect":
		mcpMgr.Reload()
		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   fmt.Sprintf("MCP servers reconnected. %d tool(s) available.", mcpMgr.ToolCount()),
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return m, nil

	default:
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   fmt.Sprintf("Unknown subcommand: %s\n\nUsage: /mcp [list|add|remove|reload]", sub),
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return m, nil
	}
}

// cmdMCPAdd handles /mcp add <name> <transport> <command|url> [args...]
func cmdMCPAdd(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	// /mcp add <name> stdio <command> [args...]
	// /mcp add <name> sse <url>
	if len(args) < 3 {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   "Usage:\n  /mcp add <name> stdio <command> [args...]\n  /mcp add <name> sse <url>",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return m, nil
	}

	name := args[0]
	transport := strings.ToLower(args[1])

	mcpMgr := m.agent.GetMCPManager()

	switch transport {
	case "stdio":
		command := args[2]
		var cmdArgs []string
		if len(args) > 3 {
			cmdArgs = args[3:]
		}

		cfg := agimcp.ServerConfig{
			Name:      name,
			Transport: "stdio",
			Command:   command,
			Args:      cmdArgs,
			Enabled:   true,
		}

		if err := mcpMgr.AddServer(cfg, true); err != nil {
			m.messageQueue.Add(QueuedMessage{
				Role:      "error",
				Content:   fmt.Sprintf("Failed to add server: %v", err),
				Timestamp: time.Now(),
				Complete:  true,
			})
			m.updateViewport()
			return m, nil
		}

	case "sse":
		url := args[2]
		cfg := agimcp.ServerConfig{
			Name:      name,
			Transport: "sse",
			URL:       url,
			Enabled:   true,
		}

		if err := mcpMgr.AddServer(cfg, true); err != nil {
			m.messageQueue.Add(QueuedMessage{
				Role:      "error",
				Content:   fmt.Sprintf("Failed to add server: %v", err),
				Timestamp: time.Now(),
				Complete:  true,
			})
			m.updateViewport()
			return m, nil
		}

	default:
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   fmt.Sprintf("Unknown transport: %s (use 'stdio' or 'sse')", transport),
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return m, nil
	}

	// Save to config
	saveMCPConfig(m)

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   fmt.Sprintf("MCP server %q added (%s). %d tool(s) available.", name, transport, mcpMgr.ToolCount()),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return m, nil
}

// saveMCPConfig persists the current MCP server list to the agent config file.
func saveMCPConfig(m *EnhancedModel) {
	mcpMgr := m.agent.GetMCPManager()
	configs := mcpMgr.GetConfigs()

	cfg := m.agent.Config()
	cfg.MCPServers = make([]config.MCPServerConfig, len(configs))
	for i, c := range configs {
		cfg.MCPServers[i] = config.MCPServerConfig{
			Name:      c.Name,
			Transport: c.Transport,
			Command:   c.Command,
			Args:      c.Args,
			Env:       c.Env,
			URL:       c.URL,
			Enabled:   c.Enabled,
		}
	}

	if err := m.agent.SaveConfig(); err != nil {
		m.agent.GetLogger().Error("Failed to save MCP config: %v", err)
	}
}
