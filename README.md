# ClosedWheelerAGI

> An open-source, terminal-based AI assistant with multi-provider LLM support, browser automation, multi-agent pipelines, persistent memory, and agent-to-agent debates.

**Version 2.1** | Created by Cezar Trainotti Paiva

[![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/go-1.25-blue)](https://go.dev/)

---

## Table of Contents

- [What Is This?](#what-is-this)
- [Installation](#installation)
- [First-Time Setup](#first-time-setup)
- [Configuration](#configuration)
- [Using the TUI](#using-the-tui)
- [Commands Reference](#commands-reference)
- [Features](#features)
  - [Multi-Provider Support](#multi-provider-support)
  - [Streaming & Reasoning](#streaming--reasoning)
  - [Browser Automation](#browser-automation)
  - [Multi-Agent Pipeline](#multi-agent-pipeline)
  - [Agent-to-Agent Debate](#agent-to-agent-debate)
  - [Persistent Memory](#persistent-memory)
  - [Brain & Roadmap](#brain--roadmap)
  - [Telegram Integration](#telegram-integration)
  - [Model Self-Configuration](#model-self-configuration-model-interview)
  - [Security & Permissions](#security--permissions)
- [Architecture](#architecture)
- [Building from Source](#building-from-source)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [Support](#support)

---

## What Is This?

ClosedWheelerAGI is a full-featured terminal AI assistant written in Go. It connects to 15+ LLM providers and lets you interact with AI models directly from your terminal through a rich TUI (Terminal User Interface).

**Key capabilities:**

- Real-time streaming responses with token usage tracking
- 15+ LLM providers (OpenAI, Anthropic, Google, DeepSeek, Groq, Mistral, Ollama, and more)
- Browser automation via Playwright for web research and interaction
- Multi-agent pipeline: Planner, Researcher, Executor, Critic
- Agent-to-agent debate mode with customizable roles
- Three-tier persistent memory across sessions
- Built-in tools: file operations, git, shell commands, code search, task management
- Telegram bot integration for remote access
- Self-configuring model parameters via model interview
- Interactive setup wizard for first-time configuration

---

## Installation

### Pre-built Binary (Windows)

Download the latest `ClosedWheeler.exe` from the [Releases page](https://github.com/razectp/closedwheeleragi/releases) and place it in your desired directory.

### Build from Source

**Requirements:** Go 1.25 or later

```bash
git clone https://github.com/razectp/closedwheeleragi.git
cd closedwheeleragi
go build -ldflags="-s -w" -o ClosedWheeler.exe ./cmd/agi
```

### Install Browser Automation (Optional)

Browser tools require Playwright browsers. Install them with:

```bash
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps
```

This installs a bundled Chromium. Alternatively, the agent can use your system Chrome or Edge.

---

## First-Time Setup

Run the executable. If no configuration is found, an interactive setup wizard starts automatically:

```
.\ClosedWheeler.exe
```

The wizard walks you through 10 steps:

| Step | What happens |
|------|-------------|
| 1. Welcome | Introduction and overview |
| 2. API Provider | Choose OpenAI, Anthropic, Google, DeepSeek, Moonshot, Ollama, or custom endpoint |
| 3. API Key | Enter your API key (stored locally in `.agi/config.json`) |
| 4. Model | Select from available models or enter a custom model ID |
| 5. Self-Configuration | Model interviews itself to determine optimal parameters (temperature, tokens, top-p) |
| 6. Permissions | Choose how permissive the agent is (all, safe, or ask-per-action) |
| 7. Rules Preset | Pick a behavior preset (coding, general, strict, creative) |
| 8. Memory | Configure memory tiers and compression |
| 9. Telegram | Optionally set up a Telegram bot for remote access |
| 10. Browser | Check Playwright browser availability |

After setup, the TUI launches and your configuration is saved to `.agi/config.json`.

---

## Configuration

Configuration is loaded from multiple sources with this priority: **CLI flags > environment variables > `.agi/config.json` > built-in defaults**.

### config.json

```json
{
  "agent_name": "ClosedWheeler",
  "api_base_url": "https://api.openai.com/v1",
  "api_key": "sk-...",
  "model": "gpt-4o",
  "provider_name": "openai",
  "max_context_size": 128000,
  "reasoning_effort": "medium",
  "fallback_models": ["gpt-4o-mini"],
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
  },
  "permissions": {
    "auto_approve": ["read", "write", "execute", "network"]
  }
}
```

### Environment Variables (.env)

Create a `.env` file in the project root:

```env
API_KEY=sk-...
API_BASE_URL=https://api.openai.com/v1
MODEL=gpt-4o
PROVIDER_NAME=openai
TELEGRAM_BOT_TOKEN=123456:ABC-DEF...
TELEGRAM_CHAT_ID=123456789
```

### CLI Flags

```
.\ClosedWheeler.exe -config path/to/config.json
.\ClosedWheeler.exe -project path/to/workspace
.\ClosedWheeler.exe -version
.\ClosedWheeler.exe -help
```

### Provider Examples

| Provider | Base URL | API Key Format |
|----------|----------|----------------|
| OpenAI | `https://api.openai.com/v1` | `sk-...` |
| Anthropic | `https://api.anthropic.com/v1` | `sk-ant-...` |
| Google Gemini | `https://generativelanguage.googleapis.com/v1beta/openai` | `AIza...` |
| DeepSeek | `https://api.deepseek.com` | `sk-...` |
| Groq | `https://api.groq.com/openai/v1` | `gsk_...` |
| Mistral | `https://api.mistral.ai/v1` | `...` |
| Moonshot/Kimi | `https://api.moonshot.cn/v1` | `sk-...` |
| Ollama (local) | `http://localhost:11434/api` | (none) |
| OpenRouter | `https://openrouter.ai/api/v1` | `sk-or-...` |
| LM Studio | `http://localhost:1234/v1` | `lm-studio` |

---

## Using the TUI

### Keyboard Shortcuts

| Action | Key |
|--------|-----|
| Send message | `Enter` |
| New line in input | `Shift+Enter` |
| Scroll up | `Up` or `PageUp` |
| Scroll down | `Down` or `PageDown` |
| Scroll to top | `Home` |
| Scroll to bottom | `End` |
| Stop current request | `Esc` |
| Open help menu | `Ctrl+H` |
| Open settings | `Ctrl+S` |
| Quit | `Ctrl+C` (when idle) |

### Status Bar

The header shows: agent name, status (idle/thinking/working), active tool count, token usage, and current model.

### Overlays

The TUI includes several overlay panels accessible via keyboard shortcuts or commands:

- **Help Menu** (`/help` or `Ctrl+H`) - Searchable command reference with categories
- **Settings** (`Ctrl+S`) - Toggle features and configure options
- **Debate Viewer** (`/conversation`) - Live view of agent-to-agent debates
- **Model Picker** (`/model`) - Browse and switch models interactively

---

## Commands Reference

Type `/help` in the input box to see all commands. Type `/help <command>` for details on a specific command.

### Conversation

| Command | Aliases | Description |
|---------|---------|-------------|
| `/clear` | `/c`, `/cls` | Clear conversation history |
| `/retry` | `/r` | Retry the last message |
| `/continue` | `/cont` | Continue the model's last response |

### Information

| Command | Aliases | Description |
|---------|---------|-------------|
| `/status [detailed]` | `/s`, `/info` | Show project and system status |
| `/stats` | `/statistics` | Show API usage statistics (tokens, cost) |
| `/memory [clear]` | `/mem` | View or clear the memory system |
| `/context [reset]` | `/ctx` | Show context cache status; `reset` invalidates it |
| `/tools [category]` | `/t` | List all available tools |

### Project

| Command | Aliases | Description |
|---------|---------|-------------|
| `/reload` | `/refresh` | Reload project files and `.agirules` |
| `/rules` | `/agirules` | Show the active project rules |
| `/git [status\|diff\|log]` | `/g` | Run git commands on the workspace |
| `/health` | `/check` | Run a health check on the project |

### Features

| Command | Aliases | Description |
|---------|---------|-------------|
| `/verbose [on\|off]` | `/v` | Toggle verbose mode (shows model reasoning) |
| `/debug [on\|off\|level]` | `/d` | Toggle debug mode for tool execution |
| `/timestamps [on\|off]` | `/time` | Toggle message timestamps |
| `/browser [headless\|stealth\|slowmo] [value]` | `/b` | Configure browser automation |
| `/heartbeat [seconds\|off]` | `/hb` | Configure background task interval |
| `/pipeline [on\|off\|status]` | `/multi-agent`, `/ma` | Toggle multi-agent pipeline |

### Memory & Brain

| Command | Aliases | Description |
|---------|---------|-------------|
| `/brain [search <query>\|recent]` | `/knowledge` | View or search the knowledge base |
| `/roadmap [summary]` | `/goals` | View the strategic roadmap |
| `/save` | `/persist` | Manually save memory to disk |

### Integration

| Command | Aliases | Description |
|---------|---------|-------------|
| `/model [name [effort]]` | `/m` | Open interactive model/provider picker |
| `/telegram [enable\|disable\|token\|chatid\|pair]` | `/tg` | Manage Telegram bot integration |

### Providers

| Command | Aliases | Description |
|---------|---------|-------------|
| `/providers [list\|add\|remove\|enable\|disable\|set-primary\|stats\|examples]` | `/prov` | Manage LLM providers |
| `/pairings` | `/pairs` | Show suggested provider pairings for debates |

### Dual Session (Agent-to-Agent)

| Command | Aliases | Description |
|---------|---------|-------------|
| `/session [on\|off\|status]` | `/dual` | Enable or disable dual session mode |
| `/debate [topic] [turns]` | `/discuss` | Start an agent-to-agent debate (opens wizard if no topic given) |
| `/conversation` | `/conv`, `/log` | Open the live debate viewer |
| `/stop` | `/end` | Stop the current debate |

### System

| Command | Aliases | Description |
|---------|---------|-------------|
| `/config [reload\|show]` | `/cfg` | Show or reload configuration |
| `/logs [n]` | `/log` | Show last `n` debug log entries |
| `/errors [n\|clear]` | `/errs` | Show recent errors |
| `/resilience` | `/recovery` | Show error resilience system status |
| `/tool-retries` | | Show intelligent tool retry statistics |
| `/retry-mode [on\|off]` | | Toggle intelligent retry feedback |
| `/recover` | `/heal` | Run system recovery procedures |
| `/report` | `/debug-report` | Generate a full debug report |
| `/help [command]` | `/h`, `/?` | Show help |
| `/exit` | `/quit`, `/q` | Exit the program |

---

## Features

### Multi-Provider Support

ClosedWheelerAGI connects to 15+ LLM providers through a unified adapter layer. Provider detection is automatic based on model name, API key, or base URL.

**Supported providers:**

| Provider | Models | Notes |
|----------|--------|-------|
| **OpenAI** | GPT-4o, GPT-5.3 Codex, o1, o3, o4 | Full tool calling + streaming |
| **Anthropic** | Claude Opus 4.6, Sonnet 4.5, Haiku 4.5 | Extended thinking + vision |
| **Google** | Gemini 2.5 Pro, 2.5 Flash | Via OpenAI-compatible endpoint |
| **DeepSeek** | deepseek-chat, deepseek-coder, deepseek-reasoner | |
| **Groq** | Llama, Mixtral (fast inference) | |
| **Mistral** | Mistral models | |
| **Moonshot/Kimi** | kimi-k2.5 | OpenAI-compatible |
| **Ollama** | Any local model (Llama, Phi, Qwen, etc.) | No API key needed |
| **OpenRouter** | Any model via OpenRouter | |
| **Azure OpenAI** | Azure-hosted OpenAI models | |
| **LM Studio** | Any local model | |
| **vLLM** | Self-hosted models | |
| **Any OpenAI-compatible** | Custom endpoints | |

Switch providers at runtime with `/model` or `/providers`.

### Streaming & Reasoning

- Real-time token streaming with live content display
- Extended thinking support for Anthropic Claude (reasoning tokens displayed separately)
- Reasoning effort configuration (`low`, `medium`, `high`) for models that support it
- Token usage tracking per response and per session

### Browser Automation

The agent controls a real Chrome/Chromium browser via Playwright for web tasks:

| Tool | Description |
|------|-------------|
| `web_fetch` | Fast HTTP fetch without browser (articles, docs, APIs) |
| `browser_navigate` | Open URL in Chrome (JS-rendered pages, SPAs) |
| `browser_get_page_text` | Get full visible text of current page |
| `browser_click` | Click element by CSS selector |
| `browser_type` | Type text into form inputs |
| `browser_get_text` | Extract text from specific element |
| `browser_screenshot` | Take page screenshot (full or optimized for LLM vision) |
| `browser_get_elements` | Discover interactive elements with selectors and coordinates |
| `browser_click_coords` | Click at exact X,Y pixel coordinates |
| `browser_eval` | Execute JavaScript on the page |
| `browser_close_tab` | Close browser session |
| `browser_list_tabs` | List open browser sessions |

**Quick start:**
```
You: Search for the latest Go release notes and summarize them
Agent: [uses web_fetch to get golang.org/doc, summarizes content]
```

### Multi-Agent Pipeline

Enable with `/pipeline on`. Complex requests are processed through four specialized agents:

```
User Request
    |
    v
[Planner] --> Breaks task into steps
    |
    v
[Researcher] --> Gathers needed information
    |
    v
[Executor] --> Implements the solution
    |
    v
[Critic] --> Reviews quality and correctness
    |
    v
Final Response
```

Each agent role has its own system prompt and can use all available tools. Toggle with `/pipeline on|off`, check with `/pipeline status`.

### Agent-to-Agent Debate

Two independent agent instances debate a topic with configurable roles.

**Quick start:**
```
/debate "Should AI be open-source?" 5
```

**Wizard mode** (interactive step-by-step):
```
/debate
```

The debate wizard lets you configure:
1. Topic
2. Model for Agent A (can be different from Agent B)
3. Role for Agent A (Proponent, Critic, Devil's Advocate, etc.)
4. Model for Agent B
5. Role for Agent B
6. Number of turns
7. Ground rules
8. Tool access level (Full, Safe, None)

**Watch live** with `/conversation` (opens the debate viewer overlay). **Stop** with `/stop`.

**Built-in role presets:**
- Proponent / Advocate
- Critic / Skeptic
- Devil's Advocate
- Neutral Analyst
- Domain Expert
- Creative Thinker
- Pragmatist
- Custom role (free text)

### Persistent Memory

Memory persists across sessions in `.agi/memory.json` with three tiers:

| Tier | Capacity | Purpose |
|------|----------|---------|
| Short-term | 20 items | Recent conversation context |
| Working | 50 items | Active task information |
| Long-term | 100 items | Persistent facts and preferences |

Items have relevance scores that decay over time. When capacity is reached, lowest-relevance items are dropped. Memory is automatically saved on shutdown and can be manually saved with `/save`.

### Brain & Roadmap

The agent maintains two persistent knowledge files in `workplace/`:

- **`brain.md`** - Knowledge base of learned patterns, errors, and decisions. The agent updates it during reflection cycles. Search with `/brain search <query>`.
- **`roadmap.md`** - Strategic objectives and milestones. View with `/roadmap`.

Every 5 heartbeats, the agent runs a deep reflection cycle that may update these files.

### Telegram Integration

Control the agent remotely via Telegram:

1. Create a bot via [@BotFather](https://t.me/botfather)
2. Get your chat ID (send a message to the bot, then check `/telegram pair`)
3. Configure in the setup wizard or via commands:

```
/telegram token YOUR_BOT_TOKEN
/telegram chatid YOUR_CHAT_ID
/telegram enable
```

Features:
- Send messages to chat with the agent
- Receive responses with Markdown formatting
- Long messages are automatically split for Telegram's character limit
- Sensitive actions trigger approval requests in the terminal

### Model Self-Configuration (Model Interview)

When adding a new model, ClosedWheelerAGI can ask the model about itself to auto-configure parameters:

- Optimal temperature
- Max output tokens
- Top-p value
- Supported features (streaming, vision, tool calling)
- Context window size

This runs automatically during setup or can be triggered from `/model`. Pre-tested profiles are available as fallback for known models (GPT-4o, Claude, Gemini, etc.).

### Security & Permissions

The permission system controls what the agent can do:

| Permission | Controls |
|-----------|----------|
| `read` | Reading files from disk |
| `write` | Writing/editing files |
| `execute` | Running shell commands |
| `network` | Making network requests |

**Security features:**
- Path traversal prevention (files restricted to project root)
- Command auditing (dangerous patterns blocked: `rm -rf`, fork bombs, etc.)
- Script validation before execution
- API key sanitization in logs
- Audit trail in `.agi/audit.log`

---

## Architecture

```
closedwheeleragi/
├── cmd/agi/main.go              # Entry point
├── pkg/
│   ├── agent/                   # Core agent: chat loop, tool dispatch, pipeline
│   │   ├── agent.go             # Main agent logic (1900+ lines)
│   │   ├── session.go           # Session & context management
│   │   ├── pipeline.go          # Multi-agent pipeline orchestration
│   │   └── pipeline_prompts.go  # Role-specific system prompts
│   ├── llm/                     # LLM client (multi-provider)
│   │   ├── client.go            # Client struct, canonical types, HTTP orchestration
│   │   ├── gollm_adapter.go     # Provider-specific logic (endpoints, headers, SSE)
│   │   ├── streaming.go         # Streaming chat methods
│   │   ├── models.go            # Model discovery and known model lists
│   │   ├── model_interview.go   # Model self-configuration system
│   │   └── model_profiles.go    # Pre-tested parameter profiles
│   ├── tui/                     # Terminal UI (Bubble Tea)
│   │   ├── tui.go               # Main TUI model
│   │   ├── commands.go          # 40 slash commands
│   │   ├── styles.go            # Centralized theme (13 colors, 80+ styles)
│   │   ├── setup_wizard*.go     # 10-step first-run wizard
│   │   ├── debate_*.go          # Debate wizard, viewer, roles
│   │   ├── dual_session.go      # Agent-to-agent conversation engine
│   │   ├── help_menu*.go        # Searchable help overlay
│   │   ├── settings_overlay.go  # Settings panel
│   │   └── panel_overlay.go     # Generic scrollable panel
│   ├── tools/                   # Tool system
│   │   ├── registry.go          # Thread-safe tool registry
│   │   ├── intelligent_retry.go # Smart retry with error classification
│   │   ├── error_enhancer.go    # Error enhancement with suggestions
│   │   └── builtin/             # Built-in tools
│   │       ├── files.go         # read_file, write_file, list_files, search_code
│   │       ├── commands.go      # exec_command (shell execution)
│   │       ├── git.go           # git_status, git_diff, git_commit, git_log
│   │       ├── browser_tools.go # 12 browser/web tools
│   │       ├── tasks.go         # manage_tasks (task.md)
│   │       ├── analysis.go      # analyze_code
│   │       └── diagnostics.go   # get_system_info
│   ├── browser/                 # Playwright wrapper
│   ├── config/                  # Configuration loading
│   ├── memory/                  # Three-tier persistent memory
│   ├── providers/               # Multi-provider management
│   ├── brain/                   # Knowledge base
│   ├── recovery/                # Error recovery & resilience
│   ├── prompts/                 # System prompt templates
│   ├── security/                # Path validation & command auditing
│   ├── telegram/                # Telegram bot
│   ├── logger/                  # File-based logging with key sanitization
│   └── ...                      # context, editor, git, health, ignore, etc.
├── .agi/                        # Runtime data (auto-created)
│   ├── config.json              # Configuration
│   ├── memory.json              # Persistent memory
│   ├── debug.log                # Debug output
│   ├── audit.log                # Security audit trail
│   └── debates/                 # Saved debate logs
└── workplace/                   # Agent workspace
    ├── brain.md                 # Knowledge base
    ├── roadmap.md               # Strategic roadmap
    └── task.md                  # Project tasks
```

### Startup Flow

```
main.go
  → Load config (flags → env → file → defaults)
  → Run setup wizard if no API key
  → Create agent.Agent with LLM client
  → Redirect log output to .agi/debug.log
  → Start Telegram bridge (if enabled)
  → Start heartbeat goroutine
  → Launch TUI via tui.RunEnhanced()
```

### Tool Execution Flow

```
User message
  → LLM returns tool_calls in response
  → tools.Registry lookup by name
  → security.Auditor validates operation
  → Execute handler (parallel for non-sensitive tools)
  → error_enhancer classifies failures
  → intelligent_retry with exponential backoff if transient
  → Result sent back to LLM for next response
```

---

## Building from Source

```bash
# Clone
git clone https://github.com/razectp/closedwheeleragi.git
cd closedwheeleragi

# Install dependencies
go mod download

# Build
go build -ldflags="-s -w" -o ClosedWheeler.exe ./cmd/agi

# Build with version info
go build -ldflags="-s -w -X main.Version=2.1" -o ClosedWheeler.exe ./cmd/agi
```

### Makefile Targets

```bash
make build          # Build binary
make run            # Run with default project dir
make test           # go test -v ./...
make test-coverage  # Generate coverage report
make lint           # golangci-lint run
make fmt            # go fmt ./...
make deps           # go mod download && go mod tidy
```

---

## Testing

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run a specific package
go test -v ./pkg/llm/...
go test -v ./pkg/tools/builtin/...
go test -v ./pkg/tui/...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

**Test coverage includes:**
- `pkg/llm/` - 40+ tests for provider detection, request/response building, SSE streaming
- `pkg/tools/builtin/` - 17 tests for file ops, commands, search, tasks, security
- `pkg/tui/` - Chat rendering tests
- `pkg/brain/` - Knowledge base operations
- `pkg/security/` - Command and path auditing
- `pkg/telegram/` - Bot message handling
- `pkg/health/` - Project health checks
- `pkg/git/` - Git operations

---

## Troubleshooting

### Setup wizard not appearing

The wizard runs when no `.agi/config.json` exists. To re-run it, delete the config:
```bash
del .agi\config.json
.\ClosedWheeler.exe
```

### API errors / rate limits

- Verify your API key in `.agi/config.json`
- Check the base URL matches your provider (see [Provider Examples](#provider-examples))
- Use `/errors` to see recent error details
- Use `/config reload` to reload config without restarting
- Check `.agi/debug.log` for detailed logs
- The agent auto-retries on 429 (rate limit) with exponential backoff

### Model not responding

- Try `/model` to switch to a different model
- Check `/stats` for token usage (you may have hit quota)
- For Anthropic: ensure `anthropic-version` header is correct (handled automatically)
- For local models (Ollama): make sure the server is running (`ollama serve`)

### Browser automation not working

1. Install Playwright browsers:
   ```bash
   go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps
   ```
2. Or ensure Chrome/Edge is installed on your system
3. Use `/browser headless off` to see the browser window for debugging
4. Check `/errors` for Playwright error messages

### Context length exceeded

- Use `/clear` to start a fresh conversation
- Use `/context reset` to clear the context cache
- Lower `max_context_size` in config
- The agent automatically trims 30% of oldest messages on context overflow

### TUI rendering issues

- Use **Windows Terminal** or **PowerShell 7+** on Windows
- Ensure your terminal supports true color (24-bit) and Unicode
- If colors look wrong, try a different terminal emulator
- Standard `cmd.exe` has limited rendering support

### Windows-specific notes

Shell commands run via `cmd.exe`. The agent knows to use Windows commands:
- `dir` instead of `ls`
- `type` instead of `cat`
- `findstr` instead of `grep`

---

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Write tests for new functionality
4. Run `go test ./...` and `go vet ./...`
5. Submit a Pull Request

**Code style:** Follow Go conventions. Run `go fmt` and `golangci-lint` before committing. See `CLAUDE.md` for detailed coding guidelines.

---

## Support

If this project is useful to you:

- **Bitcoin (BTC)**: `bc1px38hyrc4kufzxdz9207rsy5cn0hau2tfhf3678wz3uv9fpn2m0msre98w7`
- **Solana (SOL)**: `3pPpEcGEmtjCYokm8sRUu6jzjjkmfpv3qnz2pGdVYnKH`
- **Ethereum (ETH)**: `0xF465cc2d41b2AA66393ae110396263C20746CfC9`

---

**License:** [MIT](LICENSE)
