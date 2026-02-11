package tui

import (
	"fmt"
	"strings"
	"time"

	"ClosedWheeler/pkg/providers"

	tea "github.com/charmbracelet/bubbletea"
)

// Provider management commands

func cmdProviders(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if m.providerManager == nil {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   "❌ Provider manager not initialized.",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	if len(args) == 0 {
		// List all providers
		return cmdListProviders(m, args)
	}

	subCmd := args[0]
	switch subCmd {
	case "list":
		return cmdListProviders(m, args[1:])
	case "add":
		return cmdAddProvider(m, args[1:])
	case "remove":
		return cmdRemoveProvider(m, args[1:])
	case "enable":
		return cmdEnableProvider(m, args[1:])
	case "disable":
		return cmdDisableProvider(m, args[1:])
	case "set-primary":
		return cmdSetPrimaryProvider(m, args[1:])
	case "stats":
		return cmdProviderStats(m, args[1:])
	case "test":
		return cmdTestProvider(m, args[1:])
	case "examples":
		return cmdProviderExamples(m, args[1:])
	default:
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   fmt.Sprintf("❌ Unknown subcommand: %s\n\nUse /providers list, add, remove, enable, disable, set-primary, stats, test, examples", subCmd),
			Timestamp: time.Now(),
			Complete:  true,
		})
	}

	m.updateViewport()
	return *m, nil
}

func cmdListProviders(m *EnhancedModel, _ []string) (tea.Model, tea.Cmd) {
	providers := m.providerManager.ListProviders()

	var content strings.Builder
	content.WriteString("🔌 **Available Providers**\n\n")

	if len(providers) == 0 {
		content.WriteString("No providers configured.\n\n")
		content.WriteString("Add a provider with: `/providers add`\n")
		content.WriteString("See examples: `/providers examples`")
	} else {
		primary, _ := m.providerManager.GetPrimaryProvider()

		for _, p := range providers {
			isPrimary := primary != nil && p.ID == primary.ID
			status := "🔴 Disabled"
			if p.Enabled {
				if p.IsHealthy() {
					status = "🟢 Healthy"
				} else {
					status = "🟡 Unhealthy"
				}
			}

			primaryBadge := ""
			if isPrimary {
				primaryBadge = " ⭐ PRIMARY"
			}

			content.WriteString(fmt.Sprintf("**%s** %s%s\n", p.Name, status, primaryBadge))
			content.WriteString(fmt.Sprintf("  ID: `%s`\n", p.ID))
			content.WriteString(fmt.Sprintf("  Type: %s | Model: %s\n", p.Type, p.Model))
			content.WriteString(fmt.Sprintf("  Priority: %d | Cost: $%.4f/1K tokens\n", p.Priority, p.CostPerToken))

			stats := p.GetStats()
			content.WriteString(fmt.Sprintf("  Requests: %v | Success: %.1f%% | Latency: %vms\n",
				stats["total_requests"],
				stats["success_rate"],
				stats["avg_latency_ms"]))
			content.WriteString("\n")
		}

		content.WriteString("\n**Commands:**\n")
		content.WriteString("- `/providers enable <id>` - Enable provider\n")
		content.WriteString("- `/providers disable <id>` - Disable provider\n")
		content.WriteString("- `/providers set-primary <id>` - Set as primary\n")
		content.WriteString("- `/providers stats <id>` - Detailed stats\n")
		content.WriteString("- `/providers test <id>` - Test provider")
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

func cmdAddProvider(m *EnhancedModel, _ []string) (tea.Model, tea.Cmd) {
	// Interactive provider addition would go here
	// For now, show usage
	var content strings.Builder
	content.WriteString("➕ **Add Provider**\n\n")
	content.WriteString("To add a provider, edit your `.agi/providers.json` file.\n\n")
	content.WriteString("See example configurations: `/providers examples`\n\n")
	content.WriteString("After editing, reload with: `/config reload`")

	m.messageQueue.Add(QueuedMessage{
		Role:      "system",
		Content:   content.String(),
		Timestamp: time.Now(),
		Complete:  true,
	})
	m.updateViewport()
	return *m, nil
}

func cmdRemoveProvider(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   "❌ Usage: /providers remove <provider-id>",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	id := args[0]
	if err := m.providerManager.RemoveProvider(id); err != nil {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   fmt.Sprintf("❌ Failed to remove provider: %v", err),
			Timestamp: time.Now(),
			Complete:  true,
		})
	} else {
		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   fmt.Sprintf("✅ Provider '%s' removed successfully.", id),
			Timestamp: time.Now(),
			Complete:  true,
		})
	}

	m.updateViewport()
	return *m, nil
}

func cmdEnableProvider(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   "❌ Usage: /providers enable <provider-id>",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	id := args[0]
	provider, err := m.providerManager.GetProvider(id)
	if err != nil {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   fmt.Sprintf("❌ Provider not found: %v", err),
			Timestamp: time.Now(),
			Complete:  true,
		})
	} else {
		provider.Enabled = true
		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   fmt.Sprintf("✅ Provider '%s' enabled.", provider.Name),
			Timestamp: time.Now(),
			Complete:  true,
		})
	}

	m.updateViewport()
	return *m, nil
}

func cmdDisableProvider(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   "❌ Usage: /providers disable <provider-id>",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	id := args[0]
	provider, err := m.providerManager.GetProvider(id)
	if err != nil {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   fmt.Sprintf("❌ Provider not found: %v", err),
			Timestamp: time.Now(),
			Complete:  true,
		})
	} else {
		provider.Enabled = false
		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   fmt.Sprintf("⏸️ Provider '%s' disabled.", provider.Name),
			Timestamp: time.Now(),
			Complete:  true,
		})
	}

	m.updateViewport()
	return *m, nil
}

func cmdSetPrimaryProvider(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   "❌ Usage: /providers set-primary <provider-id>",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	id := args[0]
	if err := m.providerManager.SetPrimaryProvider(id); err != nil {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   fmt.Sprintf("❌ Failed to set primary: %v", err),
			Timestamp: time.Now(),
			Complete:  true,
		})
	} else {
		provider, _ := m.providerManager.GetProvider(id)
		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   fmt.Sprintf("⭐ Primary provider set to: **%s**", provider.Name),
			Timestamp: time.Now(),
			Complete:  true,
		})
	}

	m.updateViewport()
	return *m, nil
}

func cmdProviderStats(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		// Show total stats
		stats := m.providerManager.GetTotalStats()

		var content strings.Builder
		content.WriteString("📊 **Provider Statistics (All)**\n\n")
		content.WriteString(fmt.Sprintf("Total Providers: %v\n", stats["total_providers"]))
		content.WriteString(fmt.Sprintf("Active Providers: %v\n", stats["active_providers"]))
		content.WriteString(fmt.Sprintf("Total Requests: %v\n", stats["total_requests"]))
		content.WriteString(fmt.Sprintf("Total Tokens: %v\n", stats["total_tokens"]))
		content.WriteString(fmt.Sprintf("Total Cost: $%.4f\n", stats["total_cost"]))

		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   content.String(),
			Timestamp: time.Now(),
			Complete:  true,
		})
	} else {
		// Show specific provider stats
		id := args[0]
		provider, err := m.providerManager.GetProvider(id)
		if err != nil {
			m.messageQueue.Add(QueuedMessage{
				Role:      "error",
				Content:   fmt.Sprintf("❌ Provider not found: %v", err),
				Timestamp: time.Now(),
				Complete:  true,
			})
		} else {
			stats := provider.GetStats()

			var content strings.Builder
			content.WriteString(fmt.Sprintf("📊 **Statistics: %s**\n\n", provider.Name))
			content.WriteString("**Configuration:**\n")
			content.WriteString(fmt.Sprintf("- Model: %s\n", provider.Model))
			content.WriteString(fmt.Sprintf("- Type: %s\n", provider.Type))
			content.WriteString(fmt.Sprintf("- Priority: %d\n", provider.Priority))
			content.WriteString(fmt.Sprintf("- Cost: $%.4f per 1K tokens\n", provider.CostPerToken))
			content.WriteString("\n**Performance:**\n")
			content.WriteString(fmt.Sprintf("- Total Requests: %v\n", stats["total_requests"]))
			content.WriteString(fmt.Sprintf("- Failed Requests: %v\n", stats["failed_requests"]))
			content.WriteString(fmt.Sprintf("- Success Rate: %.1f%%\n", stats["success_rate"]))
			content.WriteString(fmt.Sprintf("- Avg Latency: %vms\n", stats["avg_latency_ms"]))
			content.WriteString("\n**Usage:**\n")
			content.WriteString(fmt.Sprintf("- Total Tokens: %v\n", stats["total_tokens"]))
			content.WriteString(fmt.Sprintf("- Total Cost: $%.4f\n", stats["total_cost"]))
			if stats["last_used"].(time.Time).IsZero() {
				content.WriteString("- Last Used: Never\n")
			} else {
				content.WriteString(fmt.Sprintf("- Last Used: %v\n", stats["last_used"].(time.Time).Format("2006-01-02 15:04:05")))
			}
			content.WriteString(fmt.Sprintf("- Health: %v\n", stats["healthy"]))

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

func cmdTestProvider(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   "❌ Usage: /providers test <provider-id>",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	id := args[0]
	provider, err := m.providerManager.GetProvider(id)
	if err != nil {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   fmt.Sprintf("❌ Provider not found: %v", err),
			Timestamp: time.Now(),
			Complete:  true,
		})
	} else {
		m.messageQueue.Add(QueuedMessage{
			Role:      "system",
			Content:   fmt.Sprintf("🧪 Testing provider: **%s**\n\nThis would send a test request...\n(Feature coming soon!)", provider.Name),
			Timestamp: time.Now(),
			Complete:  true,
		})
	}

	m.updateViewport()
	return *m, nil
}

func cmdProviderExamples(m *EnhancedModel, _ []string) (tea.Model, tea.Cmd) {
	examples := providers.ExampleConfigs()

	var content strings.Builder
	content.WriteString("📚 **Example Provider Configurations**\n\n")
	content.WriteString("Copy these to your `.agi/providers.json`:\n\n")

	for _, p := range examples {
		content.WriteString(fmt.Sprintf("**%s**\n", p.Name))
		content.WriteString("```json\n")
		content.WriteString(fmt.Sprintf(`{
  "id": "%s",
  "name": "%s",
  "type": "%s",
  "base_url": "%s",
  "model": "%s",
  "description": "%s",
  "max_tokens": %d,
  "temperature": %.1f,
  "priority": %d,
  "cost_per_token": %.4f,
  "enabled": true
}
`, p.ID, p.Name, p.Type, p.BaseURL, p.Model, p.Description,
			p.MaxTokens, p.Temperature, p.Priority, p.CostPerToken))
		content.WriteString("```\n\n")

		// Limit to 3 examples to avoid too much text
		if len(content.String()) > 2000 {
			content.WriteString("... (more examples available in docs)\n")
			break
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

// Debate pairing commands

func cmdPairings(m *EnhancedModel, args []string) (tea.Model, tea.Cmd) {
	if m.providerManager == nil {
		m.messageQueue.Add(QueuedMessage{
			Role:      "error",
			Content:   "❌ Provider manager not initialized.",
			Timestamp: time.Now(),
			Complete:  true,
		})
		m.updateViewport()
		return *m, nil
	}

	pairings := providers.SuggestPairingsForDebate(m.providerManager)

	var content strings.Builder
	content.WriteString("🤝 **Suggested Debate Pairings**\n\n")

	if len(pairings) == 0 {
		content.WriteString("No pairings available. Add more providers first!\n\n")
		content.WriteString("Use: `/providers examples` to see how to add providers.")
	} else {
		for i, pairing := range pairings {
			content.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, pairing.Name))
			content.WriteString(fmt.Sprintf("   %s\n", pairing.Description))
			content.WriteString(fmt.Sprintf("   Use: `/debate-cross %s %s <topic>`\n\n", pairing.ProviderA, pairing.ProviderB))
		}

		content.WriteString("\n💡 **Tip:** Use `/debate-cross` for cross-provider debates!")
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
