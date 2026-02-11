# ClosedWheelerAGI

> An open-source, terminal-based AI assistant with multi-provider support, browser automation, multi-agent pipelines, and persistent memory.

**Version 2.1** | Created by Cezar Trainotti Paiva

[![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/go-1.24-blue)](https://go.dev/)

---

## Table of Contents

- [What Is This?](#what-is-this)
- [Installation](#installation)
- [First-Time Setup](#first-time-setup)
- [Configuration](#configuration)
- [Using the TUI](#using-the-tui)
- [Commands Reference](#commands-reference)
- [Features](#features)
- [Project Structure](#project-structure)
- [Building from Source](#building-from-source)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [Support](#support)

---

## What Is This?

ClosedWheelerAGI is a terminal AI assistant that connects to any OpenAI-compatible API (OpenAI, Anthropic, Groq, Ollama, etc.) and lets you interact with AI models directly from your terminal. It includes:

- A full-featured terminal UI (TUI) with real-time streaming
- Persistent memory across sessions
- Browser automation for web tasks
- Multi-agent pipeline (Planner → Researcher → Executor → Critic)
- Agent-to-agent debate mode
- Telegram bot integration
- OAuth login for Anthropic, OpenAI, and Google
- Self-configuring model parameters via model interview

---

## Installation

### Pre-built Binary (Windows)

Download the latest `ClosedWheeler.exe` from the [releases page](https://github.com/Glucksberg/closedwheeleragi/releases) and place it in your desired directory.

### Build from Source

```bash
git clone https://github.com/Glucksberg/closedwheeleragi
cd closedwheeleragi
go build -o ClosedWheeler.exe ./cmd/agi
```

Requirements: Go 1.24+

---

## First-Time Setup

Run the executable. If no configuration is found, an interactive setup wizard starts automatically:

```
.\ClosedWheeler.exe
```

The wizard walks you through:

1. **Agent name** — Give your assistant a name (default: ClosedWheeler)
2. **API provider** — Choose OpenAI, Anthropic, or a custom OpenAI-compatible endpoint
3. **API key** — Enter your API key (stored in `.agi/config.json`)
4. **Model** — Select a model or let the wizard list available models
5. **Model interview** — The model interviews itself to set optimal parameters automatically
6. **Permissions** — Choose how permissive the agent is with file/system operations
7. **Rules preset** — Pick a behavior preset (coding, general, strict, etc.)
8. **Telegram** — Optionally configure a Telegram bot for remote access

After setup, the TUI launches. Your configuration is saved to `.agi/config.json`.

### OAuth Login (Alternative)

Instead of an API key, you can log in via OAuth:

```
/login
```

Supports Anthropic, OpenAI, and Google.

---

## Configuration

Configuration lives in `.agi/config.json`. You can also use a `.env` file in the project root.

### config.json fields

```json
{
  "api_base_url": "https://api.openai.com/v1",
  "api_key": "sk-...",
  "model": "gpt-4o-mini",
  "max_context_size": 128000,
  "memory": {
    "max_short_term_items": 20,
    "max_working_items": 50,
    "max_long_term_items": 100,
    "compression_trigger": 15,
    "storage_path": ".agi/memory.json"
  },
  "ui": {
    "theme": "dark",
    "show_tokens": true,
    "show_timestamp": true
  },
  "telegram": {
    "enabled": false,
    "bot_token": "",
    "chat_id": 0
  }
}
```

### .env file

```env
API_KEY=sk-...
API_BASE_URL=https://api.openai.com/v1
MODEL=gpt-4o-mini
```

### Command-line flags

```
.\ClosedWheeler.exe -config path/to/config.json
.\ClosedWheeler.exe -project path/to/workspace
.\ClosedWheeler.exe -version
.\ClosedWheeler.exe -help
```

### Multiple Providers

Use `/providers` to manage multiple LLM providers (OpenAI, Anthropic, Groq, local, etc.) and switch between them without restarting.

---

## Using the TUI

When the TUI is running:

| Action | Key |
|--------|-----|
| Send message | `Enter` |
| New line in input | `Shift+Enter` |
| Scroll messages up | `↑` or `PageUp` |
| Scroll messages down | `↓` or `PageDown` |
| Scroll to top | `Home` |
| Scroll to bottom | `End` |
| Stop current request | `Esc` or `Ctrl+C` |
| Quit | `Ctrl+C` (when idle) |

Type any `/command` in the input box and press Enter to run it.

### Status bar

The top status bar shows:
- Agent status (idle / thinking / working)
- Active tool executions
- Token usage for the current session
- Current model name

### Token display

After each assistant response, the number of prompt and completion tokens is shown along with elapsed time, e.g.:
```
TOK: 1.2k/320  1.4s
```

---

## Commands Reference

Type `/help` to see all commands, or `/help <command>` for details on a specific one.

### Conversation

| Command | Aliases | Description |
|---------|---------|-------------|
| `/clear` | `/c`, `/cls` | Clear the conversation history |
| `/retry` | `/r` | Retry the last message |
| `/continue` | `/cont` | Ask the model to continue its last response |

### Information

| Command | Aliases | Description |
|---------|---------|-------------|
| `/status [detailed]` | `/s`, `/info` | Show project and system status |
| `/stats` | `/statistics` | Show API usage statistics (tokens, cost estimate) |
| `/memory [clear]` | `/mem` | View or clear the memory system |
| `/context [reset]` | `/ctx` | Show context cache status; `reset` invalidates the cache |
| `/tools [category]` | `/t` | List all available tools |

### Project

| Command | Aliases | Description |
|---------|---------|-------------|
| `/reload` | `/refresh` | Reload project files and `.agirules` |
| `/rules` | `/agirules` | Show the active project rules |
| `/git [status\|diff\|log]` | `/g` | Run a git command on the workspace |
| `/health` | `/check` | Run a health check on the project |

### Features

| Command | Aliases | Description |
|---------|---------|-------------|
| `/verbose [on\|off]` | `/v` | Toggle verbose mode — shows model reasoning steps |
| `/debug [on\|off\|level]` | `/d` | Toggle debug mode for tool execution |
| `/timestamps [on\|off]` | `/time` | Toggle message timestamps |
| `/browser [headless\|stealth] [value]` | `/b` | Configure browser automation options |
| `/heartbeat [seconds\|off]` | `/hb` | Configure how often the agent runs background tasks |
| `/pipeline [on\|off\|status]` | `/multi-agent`, `/ma` | Toggle the multi-agent pipeline |

### Memory & Brain

| Command | Aliases | Description |
|---------|---------|-------------|
| `/brain [search <query>\|recent]` | `/knowledge` | View or search the knowledge base (`workplace/brain.md`) |
| `/roadmap [summary]` | `/goals` | View the strategic roadmap (`workplace/roadmap.md`) |
| `/save` | `/persist` | Manually save memory state to disk |

### Integration

| Command | Aliases | Description |
|---------|---------|-------------|
| `/model [name [effort]]` | `/m` | Open the interactive model/provider picker |
| `/login` | `/auth`, `/oauth` | OAuth login for Anthropic, OpenAI, or Google |
| `/telegram` | `/tg` | Show Telegram bot status |

### Providers

| Command | Aliases | Description |
|---------|---------|-------------|
| `/providers [list\|add\|remove\|enable\|disable\|set-primary\|stats\|examples]` | `/prov` | Manage LLM providers |
| `/pairings` | `/pairs` | Show suggested provider pairings for debates |

### Dual Session (Agent-to-Agent)

| Command | Aliases | Description |
|---------|---------|-------------|
| `/session [on\|off\|status]` | `/dual` | Enable or disable dual session mode |
| `/debate <topic> [turns]` | `/discuss` | Start an agent-to-agent debate on a topic |
| `/conversation` | `/conv`, `/log` | View the live agent-to-agent conversation log |
| `/stop` | `/end` | Stop the current debate or conversation |

### System

| Command | Aliases | Description |
|---------|---------|-------------|
| `/config [reload\|show]` | `/cfg` | Show or reload the configuration |
| `/logs [n]` | `/log` | Show the last `n` log entries |
| `/errors [n\|clear]` | `/errs` | Show recent errors |
| `/resilience` | `/recovery` | Show error resilience system status |
| `/tool-retries` | — | Show intelligent tool retry statistics |
| `/retry-mode [on\|off]` | — | Toggle intelligent retry feedback |
| `/recover` | `/heal` | Run system recovery procedures |
| `/report` | `/debug-report` | Generate a full debug report |
| `/help [command]` | `/h`, `/?` | Show help |
| `/exit` | `/quit`, `/q` | Exit the program |

---

## Features

### Multi-Provider Support

Connect to any OpenAI-compatible API. Use `/providers` to add multiple providers and switch between them at runtime. Supported:
- OpenAI (GPT-4o, GPT-4 Turbo, GPT-3.5, etc.)
- Anthropic (Claude 3.5 Sonnet, Claude 3 Opus, etc.)
- Groq (Llama, Mixtral)
- Google (Gemini via OpenAI-compatible endpoint)
- Ollama (local models)
- Any OpenAI-compatible endpoint

### Context Optimization

The context cache system reduces token usage by 60–80% on follow-up messages:
- First message: full system prompt + context is sent
- Subsequent messages: only the new user message is sent
- The system automatically compresses context when it grows large
- Use `/context reset` to force a cache invalidation

### Self-Configuring Models (Model Interview)

When adding a new model, it can interview itself to set optimal parameters (temperature, max tokens, top-p, etc.). This means you don't need to manually configure model parameters — the model knows itself best.

### Browser Automation

The agent can control a real browser (via chromedp) to perform web tasks:

```
Available tools:
  browser_navigate   — Open a URL in a browser tab
  browser_screenshot — Take a screenshot of the current page
  browser_click      — Click on a page element
  browser_type       — Type text into a field
  browser_scroll     — Scroll the page
  browser_evaluate   — Run JavaScript on the page
  browser_wait       — Wait for an element
  browser_close_tab  — Close a browser tab
  web_fetch          — Fetch a URL as text (fast, no browser needed)
```

Configure with `/browser headless on` or `/browser stealth on`.

### Multi-Agent Pipeline

Enable with `/pipeline on`. Requests are routed through four specialized agents:

1. **Planner** — Breaks the task into steps
2. **Researcher** — Gathers information needed
3. **Executor** — Implements the solution
4. **Critic** — Reviews the output for quality

Use `/pipeline status` to see the current state of each agent.

### Agent-to-Agent Debate

Two separate agent instances can debate a topic:

```
/debate "Is TDD better than BDD?" 5
```

This starts a 5-turn debate. Watch the live conversation with `/conversation`. Stop with `/stop`.

### Brain & Roadmap

The agent maintains two persistent files in `workplace/`:

- **`brain.md`** — The agent's knowledge base. It records errors, patterns, and decisions learned over time. Browse with `/brain` or search with `/brain search <query>`.
- **`roadmap.md`** — Strategic objectives. The agent updates this during deep reflection cycles. View with `/roadmap`.

Every 5 heartbeats, the agent runs a deep reflection cycle that may update these files.

### Persistent Memory

Memory is stored in `.agi/memory.json` across sessions. It has three tiers:
- **Short-term** — Recent conversation context (20 items)
- **Working** — Active task information (50 items)
- **Long-term** — Persistent facts and preferences (100 items)

Use `/memory` to inspect and `/memory clear` to reset.

### Telegram Integration

Control the agent remotely via Telegram:

1. Create a bot via [@BotFather](https://t.me/botfather)
2. Add `bot_token` and `chat_id` to your config
3. Set `telegram.enabled: true`

Available Telegram commands:
- Send any message to chat with the agent
- Sensitive actions trigger an approval request in the terminal

### Fallback Models

If the primary model is slow or returns an error, the agent can fall back to a backup model automatically. Configure via `/providers`.

### Security & Permissions

The permission system controls what the agent is allowed to do:
- `read` — Read files
- `write` — Write/edit files
- `execute` — Run shell commands
- `network` — Make network requests

The setup wizard lets you choose a permission level. Change at any time in the config.

---

## Project Structure

```
closedwheeleragi/
├── ClosedWheeler.exe       # Compiled binary
├── cmd/
│   └── agi/
│       └── main.go         # Entry point
├── pkg/
│   ├── agent/              # Core agent logic (Chat, pipeline, sessions)
│   ├── brain/              # Knowledge base system
│   ├── browser/            # Browser automation (chromedp)
│   ├── config/             # Configuration loading and management
│   ├── context/            # Project context handling
│   ├── editor/             # File editing capabilities
│   ├── git/                # Git integration
│   ├── health/             # Project health monitoring
│   ├── llm/                # LLM client (streaming, OAuth, model interview)
│   ├── logger/             # Logging
│   ├── memory/             # Persistent memory (short/working/long-term)
│   ├── permissions/        # Permission system
│   ├── prompts/            # System prompts and rules management
│   ├── providers/          # Multi-provider configuration
│   ├── recovery/           # Error recovery
│   ├── roadmap/            # Strategic roadmap
│   ├── security/           # Security auditing
│   ├── skills/             # Custom skill modules
│   ├── telegram/           # Telegram bot integration
│   ├── tools/              # Tool registry and execution
│   └── tui/                # Terminal UI (Bubble Tea)
├── .agi/                   # Runtime data (created automatically)
│   ├── config.json         # Active configuration
│   ├── memory.json         # Persistent memory
│   ├── audit.log           # Audit trail
│   └── debug.log           # Debug output
├── workplace/              # Agent workspace (created automatically)
│   ├── brain.md            # Agent knowledge base
│   ├── roadmap.md          # Strategic objectives
│   ├── task.md             # Current task
│   ├── personality.md      # Agent personality
│   └── .agirules           # Behavior rules for this workspace
├── docs/
│   └── guides/             # Deep-dive feature guides
├── config.json.example     # Configuration template
└── .env.example            # Environment variable template
```

---

## Building from Source

```bash
# Clone
git clone https://github.com/Glucksberg/closedwheeleragi
cd closedwheeleragi

# Build
go build -o ClosedWheeler.exe ./cmd/agi

# Run tests
go test ./...

# Build with version info
go build -ldflags "-X main.Version=2.1" -o ClosedWheeler.exe ./cmd/agi
```

### Makefile targets

```bash
make build      # Build binary
make test       # Run tests
make clean      # Remove build artifacts
```

---

## Troubleshooting

### "No configuration found" on startup

The setup wizard will run automatically. If it doesn't start, create `.agi/config.json` from the template:
```bash
cp config.json.example .agi/config.json
```

### API errors / rate limits

- Check your API key in `.agi/config.json`
- Use `/config reload` to reload config without restarting
- Use `/errors` to see recent error details
- Check `.agi/debug.log` for detailed logs

### Browser automation not working

- Make sure Google Chrome or Chromium is installed and in your PATH
- Use `/browser headless off` to see the browser window and debug
- Check `/errors` for chromedp error messages

### Context getting too long

- Use `/context reset` to clear the context cache
- Use `/clear` to start a fresh conversation
- Lower `max_context_size` in config if models complain about length

### Windows-specific issues

On Windows, shell commands run via `cmd.exe`. Use Windows commands:
- `dir` instead of `ls`
- `type` instead of `cat`
- `del` instead of `rm`
- `findstr` instead of `grep`

### TUI rendering issues

Make sure your terminal supports true color and Unicode. Recommended terminals:
- Windows Terminal
- PowerShell 7+
- Any modern Linux/macOS terminal

---

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## Support

If this project is useful to you:

- **Bitcoin (BTC)**: `bc1px38hyrc4kufzxdz9207rsy5cn0hau2tfhf3678wz3uv9fpn2m0msre98w7`
- **Solana (SOL)**: `3pPpEcGEmtjCYokm8sRUu6jzjjkmfpv3qnz2pGdVYnKH`
- **Ethereum (ETH)**: `0xF465cc2d41b2AA66393ae110396263C20746CfC9`
