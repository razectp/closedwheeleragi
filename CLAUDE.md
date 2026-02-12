# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ClosedWheelerAGI is a terminal-based AI assistant written in Go. It provides a Bubble Tea TUI for interacting with multiple LLM providers, with browser automation (Playwright), persistent memory, multi-agent pipelines, and tool execution.

Module name: `ClosedWheeler` (in go.mod). Entry point: `cmd/agi/main.go`.

## Build & Development Commands

```bash
# Build
make build                  # Build binary (ClosedWheeler.exe on Windows)
go build -ldflags="-s -w" -o ClosedWheeler.exe ./cmd/agi

# Run
make run                    # Run with default project dir "workplace"
go run ./cmd/agi -project . # Run against current directory

# Test
make test                   # go test -v ./...
make test-coverage          # Generate coverage.out and coverage.html
go test -v ./pkg/brain/...  # Run a single package's tests

# Lint & Format
make lint                   # golangci-lint run ./...
make fmt                    # go fmt ./...

# Dependencies
make deps                   # go mod download && go mod tidy
```

## Architecture

### Startup Flow
`main.go` loads config → runs interactive setup if no API key → creates `agent.Agent` → redirects `log` output to `.agi/debug.log` → starts Telegram bridge + heartbeat → launches TUI via `tui.RunEnhanced()`.

### Key Packages

| Package | Role |
|---------|------|
| `pkg/agent` | Core agent: chat loop, tool dispatch, memory integration, multi-agent pipeline (`pipeline.go`), session/context optimization (`session.go`) |
| `pkg/llm` | Multi-provider LLM client with streaming, rate limiting, and model self-configuration ("model interview"). Provider implementations: `provider_openai.go` (OpenAI-compatible), `provider_anthropic.go` (native Anthropic with OAuth/vision/extended thinking) |
| `pkg/tui` | Bubble Tea TUI — the largest package. `tui.go` is the main model; `styles.go` centralizes all colors and styles; `commands.go` handles 50+ slash commands; `interactive_setup.go` / `setup_wizard*.go` handle first-run setup; `dual_session.go` handles agent-to-agent debates |
| `pkg/tools` | Tool registry (`registry.go`), error enhancement, intelligent retry with exponential backoff. Built-in tools live in `tools/builtin/` (files, git, browser, shell, analysis, tasks) |
| `pkg/config` | Config loading from `.agi/config.json`, `.env`, or CLI flags (priority: flags > env > file > defaults) |
| `pkg/browser` | Playwright-go wrapper for browser automation with tab management, screenshot capture, JS evaluation. Cross-platform browser detection in `browser.go`, fast HTTP fetch in `fetch.go` |
| `pkg/memory` | Three-tier persistent memory: short-term (20 items) → working (50) → long-term (100). Stored in `.agi/memory.json` |
| `pkg/providers` | Multi-provider management with fallback, priority, statistics tracking, and runtime switching |
| `pkg/brain` | Knowledge base — stores learned patterns/errors, updated during reflection cycles |
| `pkg/recovery` | Error recovery/resilience system with fallback strategies |
| `pkg/prompts` | System prompt generation and `.agirules` management |
| `pkg/security` | Path validation, permission checking, operation auditing |

### Multi-Agent Pipeline (optional, toggled via `/pipeline`)
User request flows through 4 roles: **Planner** → **Researcher** → **Executor** → **Critic**. Defined in `agent/pipeline.go` with prompts in `agent/pipeline_prompts.go`.

### Tool Execution Flow
User message → LLM returns tool call → `tools.Registry` lookup → security/permission check → execute handler → error enhancement → intelligent retry if needed → result back to LLM.

### Runtime Data
All runtime state lives in `.agi/`: `config.json`, `memory.json`, `debug.log`, `audit.log`, `skills/`.

## Linting Notes

The `.golangci.yml` has intentional exclusions:
- `unused` linter is disabled globally (TUI false positives)
- `goconst` is disabled (common string false positives)
- Max cyclomatic complexity is 50 (some TUI/agent functions are legitimately complex)
- `errcheck` is excluded for Bubble Tea `.Send`/`.Quit`, `WriteString`, `os.WriteFile`, `.Load`, `.Close`, `fmt.Fprint*`, and Telegram async calls

## TUI Styling System

All TUI colors and styles are centralized in `pkg/tui/styles.go`. No file outside `styles.go` should define `lipgloss.Color(...)` literals or construct styles with hardcoded color values.

**Theme palette** (13 named colors): `PrimaryColor`, `SecondaryColor`, `AccentColor`, `SuccessColor`, `ErrorColor`, `MutedColor`, `HeadingColor`, `BgBase`, `BgDark`, `BgDarker`, `TextPrimary`, `TextSecondary`, `TextMuted`.

**Style naming convention**: `<Component><Element>Style` (e.g., `PickerTitleStyle`, `ToolSuccessStyle`, `DebateViewFooterStyle`).

**Toggle helpers** (in `commands.go`): `renderToggle(bool)` → "ON"/"OFF", `renderEnabled(bool)` → "enabled"/"disabled", `renderEnabledUpper(bool)` → "ENABLED"/"DISABLED". All use `ToggleOnStyle`/`ToggleOffStyle`.

**Overlay pattern**: Each overlay (help, panel, settings, wizard, debate viewer, picker) has its own `*Style` group in `styles.go` (e.g., `HelpBoxStyle`, `PanelBoxStyle`, `SettingsBoxStyle`).

## Conventions

- Go module is `ClosedWheeler` — imports use `ClosedWheeler/pkg/...`
- TUI uses Charmbracelet stack: `bubbletea` (framework), `bubbles` (components), `lipgloss` (styling)
- Standard `log` output is redirected to `.agi/debug.log` at runtime to avoid corrupting the TUI's alternate screen
- Shell commands execute via `cmd.exe` on Windows — use Windows-native commands (`dir`, `type`, `findstr`)
- Browser automation requires Playwright browsers installed: `go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps`
- Config priority: CLI flags > environment variables > `.agi/config.json` > built-in defaults

## Testing

Tests exist in: `pkg/brain/`, `pkg/git/`, `pkg/health/`, `pkg/security/`, `pkg/tools/builtin/`. Run individual package tests with `go test -v ./pkg/<package>/...`.
