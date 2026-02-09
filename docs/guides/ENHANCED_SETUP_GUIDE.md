# 🚀 Enhanced Setup - Complete Guide

**Date**: 2026-02-08
**Status**: ✅ **IMPLEMENTED**

---

## 🎯 What's New

The first-time setup wizard now includes:

✅ **Agent Name** - Give your AI a custom name
✅ **Multiple Models** - Select primary + fallback models
✅ **Permissions Presets** - Full/Restricted/Read-Only
✅ **Rules Presets** - Code Quality/Security/Performance
✅ **Memory Presets** - Minimal/Balanced/Extended
✅ **Dynamic Model Switching** - Change models via `/model` command

---

## 🎨 Setup Wizard Flow

### Step 1: Agent Name

```
Give your agent a name [ClosedWheeler]: MyCodeAssistant
✅ Agent name: MyCodeAssistant
```

### Step 2: API Configuration

```
📡 API Configuration

Examples:
  1. OpenAI     - https://api.openai.com/v1
  2. NVIDIA     - https://integrate.api.nvidia.com/v1
  3. Anthropic  - https://api.anthropic.com/v1
  4. Local      - http://localhost:11434/v1

API Base URL [https://api.openai.com/v1]:
API Key []: sk-proj-...
```

### Step 3: Model Selection

```
🔍 Fetching available models...
✅ Found 15 models

  1. gpt-4o
  2. gpt-4o-mini
  3. gpt-3.5-turbo
  ... and 12 more

Select primary model (1-15 or name): 2
```

**Fallback Models** (optional):

```
Add fallback models? (y/N): y

Enter model numbers/names (comma-separated):
Fallbacks: 3, gpt-3.5-turbo-16k
```

### Step 4: Permissions Preset

```
🔐 Permissions Preset

Presets:
  1. Full Access    - All commands and tools (recommended for solo dev)
  2. Restricted     - Only read, edit, write files (safe for teams)
  3. Read-Only      - Only read operations (maximum safety)

Select preset (1-3) [1]: 1
```

**Presets Explained**:

| Preset | Allowed Tools | Use Case |
|--------|--------------|----------|
| **Full Access** | All tools (`*`) | Solo developer, full trust |
| **Restricted** | read, edit, write files only | Team environment, needs approval |
| **Read-Only** | read, list, search files only | Analysis only, maximum safety |

### Step 5: Project Rules

```
📜 Project Rules

Presets:
  1. None           - No predefined rules
  2. Code Quality   - Focus on clean, maintainable code
  3. Security First - Emphasize security best practices
  4. Performance    - Optimize for speed and efficiency

Select preset (1-4) [1]: 2
```

**Rules Created**:

- **None**: No `.agirules` file created
- **Code Quality**: SOLID principles, clean code, tests
- **Security**: OWASP guidelines, input validation, encryption
- **Performance**: Complexity optimization, caching, async operations

### Step 6: Memory Configuration

```
🧠 Memory Configuration

Presets:
  1. Balanced  - 20/50/100 items (recommended)
  2. Minimal   - 10/25/50 items (lightweight)
  3. Extended  - 30/100/200 items (maximum context)

Select preset (1-3) [1]: 1
```

### Step 7: Telegram Integration (Optional)

```
📱 Telegram Integration (Optional)

Telegram allows you to control the agent remotely:
  • Chat with the agent from anywhere
  • Execute commands (/status, /logs, /model)
  • Approve sensitive operations

To get a bot token:
  1. Open Telegram and find @BotFather
  2. Send: /newbot
  3. Follow instructions to create your bot
  4. Copy the token (looks like: 1234567890:ABC...)

Configure Telegram now? (y/N): y

Enter Telegram Bot Token []: 1234567890:ABCdefGHIjklMNOpqrsTUVwxyz
✅ Telegram token saved!
Note: You'll need to complete pairing after starting the agent
```

**Or skip for now**:
```
Configure Telegram now? (y/N): n
⏭️  Skipping Telegram setup (you can configure it later)
```

**Memory Presets**:

| Preset | STM | WM | LTM | Use Case |
|--------|-----|-----|-----|----------|
| **Minimal** | 10 | 25 | 50 | Simple projects, resource-constrained |
| **Balanced** | 20 | 50 | 100 | Most projects (recommended) |
| **Extended** | 30 | 100 | 200 | Complex projects, maximum context |

### Step 8: Summary

```
💾 Saving configuration...
🎉 Setup Complete!

Configuration Summary:
  Agent:       MyCodeAssistant
  Model:       gpt-4o-mini
  Fallbacks:   gpt-3.5-turbo, gpt-3.5-turbo-16k
  Permissions: full
  Rules:       code-quality
  Memory:      balanced
  Telegram:    true

📱 Telegram Pairing Instructions

To complete Telegram setup:
  1. Start the ClosedWheeler agent
  2. Open Telegram and find your bot
  3. Send: /start
  4. Copy your Chat ID from the bot's response
  5. Edit .agi/config.json and set 'chat_id' field
  6. Restart the agent

💡 Tip: You can also configure Telegram later by editing .agi/config.json
```

---

## 📂 Files Created

After setup, these files are created:

### 1. `.env` (API Credentials)

```bash
# ClosedWheelerAGI Configuration
# Agent: MyCodeAssistant
# Generated: 2026-02-08

API_BASE_URL=https://api.openai.com/v1
API_KEY=sk-proj-...
MODEL=gpt-4o-mini
```

### 2. `.agi/config.json` (Full Configuration)

```json
{
  "// agent_name": "MyCodeAssistant",
  "model": "gpt-4o-mini",
  "fallback_models": ["gpt-3.5-turbo", "gpt-3.5-turbo-16k"],
  "fallback_timeout": 30,
  "permissions": {
    "allowed_commands": ["*"],
    "allowed_tools": ["*"],
    ...
  },
  "memory": {
    "max_short_term_items": 20,
    "max_working_items": 50,
    "max_long_term_items": 100,
    ...
  }
}
```

### 3. `.agirules` (Project Rules - if selected)

Created only if you select a rules preset (Code Quality, Security, or Performance).

---

## 🔄 Changing Models Dynamically

### Via Telegram

```
/model                    # Show current model
/model gpt-4o            # Switch to gpt-4o
/model gpt-3.5-turbo     # Switch to gpt-3.5-turbo
```

**Example**:

```
You: /model
Bot: 🤖 Current Model
     Primary: gpt-4o-mini
     Fallbacks: gpt-3.5-turbo, gpt-3.5-turbo-16k

You: /model gpt-4o
Bot: ✅ Model changed to: gpt-4o
```

### Via TUI

The `/model` command works the same way in the terminal interface!

### Programmatically

Edit `.agi/config.json`:

```json
{
  "model": "new-model-name"
}
```

Restart the agent to apply.

---

## 🔐 Permissions Presets Details

### Full Access (Preset 1)

```json
{
  "allowed_commands": ["*"],
  "allowed_tools": ["*"],
  "sensitive_tools": ["git_commit", "git_push", "exec_command", "write_file", "delete_file"],
  "require_approval_for_all": false
}
```

**Behavior**:
- All commands allowed
- All tools allowed
- Only sensitive tools require approval (if Telegram enabled)

### Restricted (Preset 2)

```json
{
  "allowed_commands": ["*"],
  "allowed_tools": ["read_file", "list_files", "search_files", "edit_file", "write_file"],
  "require_approval_for_all": true
}
```

**Behavior**:
- Only file read/write tools allowed
- ALL tool executions require approval
- Cannot run commands, git operations, or delete files

### Read-Only (Preset 3)

```json
{
  "allowed_commands": ["/status", "/logs", "/help"],
  "allowed_tools": ["read_file", "list_files", "search_files"]
}
```

**Behavior**:
- Only analysis commands allowed
- Cannot modify anything
- Maximum safety for code review/audit scenarios

---

## 📜 Rules Presets Content

### Code Quality Rules

```markdown
# Code Quality Rules

## Principles
- Write clean, readable, and maintainable code
- Follow SOLID principles
- Prefer composition over inheritance
- Keep functions small and focused
- Use meaningful names for variables and functions

## Standards
- Add comments for complex logic
- Write unit tests for new code
- Refactor duplicated code
- Follow language-specific best practices
```

### Security First Rules

```markdown
# Security First Rules

## Principles
- Never commit secrets to version control
- Validate all user inputs
- Use parameterized queries for SQL
- Sanitize output to prevent XSS
- Implement proper authentication and authorization

## Standards
- Follow OWASP Top 10 guidelines
- Use encryption for sensitive data
- Keep dependencies updated
- Log security-relevant events
```

### Performance Rules

```markdown
# Performance Optimization Rules

## Principles
- Optimize for time and space complexity
- Use appropriate data structures
- Avoid premature optimization
- Profile before optimizing
- Cache expensive operations

## Standards
- Minimize database queries
- Use async/await for I/O operations
- Implement pagination for large datasets
- Lazy load resources when possible
```

---

## 🧠 Memory Presets Comparison

| Metric | Minimal | Balanced | Extended |
|--------|---------|----------|----------|
| Short-Term | 10 | 20 | 30 |
| Working | 25 | 50 | 100 |
| Long-Term | 50 | 100 | 200 |
| Compression Trigger | 15 | 15 | 15 |
| **Total Items** | 85 | 170 | 330 |
| **Best For** | Small projects | Most projects | Large codebases |
| **Memory Usage** | Low | Medium | High |

---

## 🎯 Use Case Examples

### Example 1: Solo Developer, Full Access

```
Agent Name: DevAssistant
Model: gpt-4o-mini
Fallbacks: gpt-3.5-turbo
Permissions: Full Access
Rules: Code Quality
Memory: Balanced
```

**Perfect for**: Personal projects where you want maximum capability.

### Example 2: Team Project, Safety First

```
Agent Name: TeamHelper
Model: gpt-4o
Fallbacks: gpt-4o-mini
Permissions: Restricted
Rules: Security First
Memory: Balanced
```

**Perfect for**: Team environments where changes need approval.

### Example 3: Code Review Only

```
Agent Name: CodeReviewer
Model: gpt-4o
Fallbacks: none
Permissions: Read-Only
Rules: Code Quality
Memory: Extended
```

**Perfect for**: Auditing and analysis without modification risk.

### Example 4: Performance Optimization

```
Agent Name: Optimizer
Model: gpt-4o
Fallbacks: gpt-4o-mini, gpt-3.5-turbo
Permissions: Full Access
Rules: Performance
Memory: Extended
```

**Perfect for**: Optimizing large codebases with context.

---

## 🔧 Customizing After Setup

All settings can be modified after initial setup:

### Change Permissions

Edit `.agi/config.json`:

```json
{
  "permissions": {
    "allowed_tools": ["read_file", "write_file", "git_status"]
  }
}
```

### Add/Remove Fallback Models

```json
{
  "fallback_models": ["model-1", "model-2", "model-3"]
}
```

### Change Memory Settings

```json
{
  "memory": {
    "max_short_term_items": 25,
    "max_working_items": 75,
    "max_long_term_items": 150
  }
}
```

### Modify Rules

Edit `.agirules` file directly or create a new one.

---

## 📊 Command Reference

### Setup Commands

```bash
# First time setup
./ClosedWheeler

# Re-run setup (creates new config)
rm .env .agi/config.json
./ClosedWheeler
```

### Runtime Commands (Telegram/TUI)

```
/status                  # Show agent status
/logs                    # View recent logs
/diff                    # Show git diff
/model                   # Show current model
/model <name>            # Switch to different model
/help                    # Show all commands
```

---

## 🐛 Troubleshooting

### Issue: Setup fails to fetch models

**Symptom**: "Failed to fetch models" error

**Solution**:
1. Verify API key is correct
2. Check internet connection
3. Manually enter model name when prompted

### Issue: Permissions too restrictive

**Symptom**: "Tool not allowed" errors

**Solution**:
Edit `.agi/config.json` and change permissions preset or add specific tools to `allowed_tools`.

### Issue: Model switch doesn't work

**Symptom**: `/model` command fails

**Solution**:
Ensure the new model name is correct and available with your API key.

---

## 🎉 Summary

The enhanced setup provides:

✅ **Personalization** - Custom agent name
✅ **Reliability** - Automatic fallback models
✅ **Security** - Granular permission controls
✅ **Guidance** - Predefined rules for best practices
✅ **Flexibility** - Memory configurations for any project size
✅ **Dynamic Control** - Change models on the fly

All configurable through a simple, interactive wizard! 🚀

---

**Status**: ✅ **PRODUCTION READY**
**Build**: ✅ **11MB**
**Setup Time**: ⏱️ **< 2 minutes**

*Your AI agent, your way!* 🎨
