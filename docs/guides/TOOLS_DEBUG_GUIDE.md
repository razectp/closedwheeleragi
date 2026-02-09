# 🔧 Tools & Debug - Complete Guide

## 📖 Overview

This guide covers the enhanced tool execution system with:
1. **🐛 Detailed Debug Logging** - Track every tool execution with stack traces
2. **🌐 Anti-Detection Browser** - Bypass CAPTCHA and bot detection
3. **✅ Error Transparency** - See exactly what went wrong and why

---

## 🐛 Debug System

### What Is It?

The debug system provides detailed logging of ALL tool executions, showing:
- ✅ Tool name and arguments
- ✅ Execution duration
- ✅ Success/failure status
- ✅ Error messages with types
- ✅ Stack traces for crashes
- ✅ Output previews

### Enable Debug Mode

#### Via Config

`.agi/config.json`:
```json
{
  "debug_tools": true
}
```

#### Via Code

```go
tools.SetGlobalDebugLevel(tools.DebugVerbose)
```

### Debug Levels

| Level | Value | Output |
|-------|-------|--------|
| **Off** | `0` | No debug output |
| **Basic** | `1` | Tool name, duration, success/fail |
| **Verbose** | `2` | + Arguments, output preview, metadata |
| **Trace** | `3` | + Full stack traces on errors |

### Example Debug Output

#### Basic Level (default when enabled)
```
╔══════════════════════════════════════════════════════════════
║ 🔧 TOOL EXECUTION START
║ Tool: browser_navigate
║ Time: 2026-02-09 14:30:15.123
╚══════════════════════════════════════════════════════════════

╔══════════════════════════════════════════════════════════════
║ ✅ TOOL EXECUTION SUCCESS
║ Tool: browser_navigate
║ Duration: 2.3s
╚══════════════════════════════════════════════════════════════
```

#### Verbose Level
```
╔══════════════════════════════════════════════════════════════
║ 🔧 TOOL EXECUTION START
║ Tool: browser_navigate
║ Time: 2026-02-09 14:30:15.123
║ Arguments:
║    {
║      "task_id": "research-ai",
║      "url": "https://example.com"
║    }
╚══════════════════════════════════════════════════════════════

╔══════════════════════════════════════════════════════════════
║ ✅ TOOL EXECUTION SUCCESS
║ Tool: browser_navigate
║ Duration: 2.3s
║ Output Preview:
║    Navigated to: https://example.com
║    Title: Example Domain
║    Status: 200
║    Content length: 1256 chars
║ Metadata:
║    tool_description: Navigate to a URL in a browser
╚══════════════════════════════════════════════════════════════
```

#### Error Example (Trace Level)
```
╔══════════════════════════════════════════════════════════════
║ ❌ TOOL EXECUTION FAILED
║ Tool: browser_click
║ Duration: 150ms
║ Error Type: execution
║ Error: timeout waiting for selector: button.submit
║ Details: Element not found within 60s timeout
║ Stack Trace:
║    goroutine 45 [running]:
║    ClosedWheeler/pkg/browser.(*Manager).Click(...)
║        /path/to/browser.go:234
║    ClosedWheeler/pkg/tools/builtin.RegisterBrowserTools.func2(...)
║        /path/to/browser_tools.go:98
║    ... (15 more lines)
╚══════════════════════════════════════════════════════════════
```

### Error Types

| Type | Description | Example |
|------|-------------|---------|
| **validation** | Invalid arguments | Missing required parameter |
| **execution** | Tool execution failed | Selector not found, timeout |
| **panic** | Unexpected crash | Nil pointer dereference |
| **timeout** | Operation timed out | Network request exceeded limit |

### Debug Reports

Generate summary reports:

```go
executor := tools.NewExecutor(registry)
report := executor.GetDebugReport()
fmt.Println(report)
```

Output:
```
╔══════════════════════════════════════════════════════════════
║ 📊 TOOL EXECUTION REPORT
╠══════════════════════════════════════════════════════════════
║ Total Executions: 45
║ Successful: 42 (93.3%)
║ Failed: 3 (6.7%)
║ Average Duration: 1.2s
╠══════════════════════════════════════════════════════════════
║ Errors by Type:
║   - execution: 2
║   - timeout: 1
╠══════════════════════════════════════════════════════════════
║ Tool Usage:
║   - browser_navigate: 15
║   - browser_click: 10
║   - read_file: 12
║   - write_file: 8
╚══════════════════════════════════════════════════════════════
```

### Get Recent Failures

```go
failures := executor.GetRecentFailures()
for _, failure := range failures {
    fmt.Printf("Tool: %s\n", failure.ToolName)
    fmt.Printf("Error: %v\n", failure.Error)
    fmt.Printf("Duration: %v\n", failure.Duration)
}
```

---

## 🌐 Anti-Detection Browser

### The Problem

Many websites use bot detection that triggers CAPTCHAs when they detect:
- Selenium/WebDriver
- Automated browser patterns
- Missing browser features
- Unnatural timing

### The Solution

Our browser now includes comprehensive anti-detection features:

#### 1. **Stealth Mode** (Enabled by Default)

Injects JavaScript to hide automation:
```javascript
// Hide webdriver property
Object.defineProperty(navigator, 'webdriver', {
  get: () => undefined
});

// Mock chrome runtime
window.chrome = { runtime: {} };

// Mock plugins and permissions
Object.defineProperty(navigator, 'plugins', {
  get: () => [1, 2, 3, 4, 5]
});
```

#### 2. **Realistic User Agent**

```
Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36
(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36
```

#### 3. **Human-Like Timing**

```go
SlowMo: 100 // Adds 100ms delay between operations
```

#### 4. **Launch Arguments**

```go
args := []string{
    "--disable-blink-features=AutomationControlled",
    "--disable-dev-shm-usage",
    "--no-sandbox",
    "--disable-setuid-sandbox",
    "--disable-web-security",
    "--disable-infobars",
    "--window-size=1920,1080",
    "--start-maximized",
}
```

#### 5. **Locale & Timezone**

```go
Locale:     "en-US"
TimezoneId: "America/New_York"
```

### Configuration

`.agi/config.json`:
```json
{
  "browser": {
    "headless": false,
    "stealth": true,
    "slow_mo": 100
  }
}
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `headless` | bool | `false` | Run browser without GUI |
| `stealth` | bool | `true` | Enable anti-detection |
| `slow_mo` | int | `100` | Delay in ms (0 = no delay) |

### Testing Anti-Detection

Visit these sites to test:
- https://bot.sannysoft.com/ - Comprehensive bot detection tests
- https://arh.antoinevastel.com/bots/areyouheadless - Headless detection
- https://intoli.com/blog/not-possible-to-block-chrome-headless/chrome-headless-test.html

**Before stealth:**
- ❌ webdriver: true
- ❌ Chrome: false
- ❌ Plugins: 0

**After stealth:**
- ✅ webdriver: undefined
- ✅ Chrome: true
- ✅ Plugins: 5

### Advanced: Persistent Sessions

The browser automatically uses persistent sessions to maintain:
- ✅ Cookies
- ✅ Local storage
- ✅ Session data
- ✅ Browser cache

Location: `.agi/browsers/`

This helps avoid repeated logins and builds browser "reputation".

---

## 🛠️ All Available Tools

### File Operations

| Tool | Description |
|------|-------------|
| `read_file` | Read file contents |
| `write_file` | Write/create file |
| `edit_file` | Edit existing file |
| `list_files` | List directory contents |
| `search_files` | Search for files by pattern |

### Browser Automation

| Tool | Description | Anti-Detection |
|------|-------------|----------------|
| `browser_navigate` | Navigate to URL | ✅ Stealth enabled |
| `browser_click` | Click element (CSS selector) | ✅ Human-like timing |
| `browser_type` | Type text into input | ✅ Natural typing speed |
| `browser_get_text` | Extract text from element | ✅ |
| `browser_screenshot` | Capture page screenshot | ✅ Optimized for AI |
| `browser_get_elements` | Map all interactive elements | ✅ |
| `browser_click_coords` | Click at X,Y coordinates | ✅ |
| `browser_close_tab` | Close specific tab | ✅ |
| `browser_list_tabs` | List all open tabs | ✅ |

### Git Operations

| Tool | Description |
|------|-------------|
| `git_status` | Show repo status |
| `git_diff` | Show changes |
| `git_commit` | Commit changes |
| `git_push` | Push to remote |
| `git_checkpoint` | Create checkpoint |

### Analysis

| Tool | Description |
|------|-------------|
| `analyze_code` | Analyze code quality |
| `run_diagnostics` | Run project diagnostics |
| `security_scan` | Scan for vulnerabilities |

### Tasks

| Tool | Description |
|------|-------------|
| `list_tasks` | List pending tasks |
| `complete_task` | Mark task complete |

---

## 🔍 Debugging Common Issues

### Browser Issues

#### CAPTCHA Still Appearing

**Causes:**
1. Stealth mode disabled
2. Too fast operations (SlowMo = 0)
3. Headless mode enabled
4. New/empty browser profile

**Solutions:**
```json
{
  "browser": {
    "headless": false,
    "stealth": true,
    "slow_mo": 200  // Increase delay
  }
}
```

#### Browser Won't Start

**Check debug output:**
```
╔══════════════════════════════════════════════════════════════
║ ❌ TOOL EXECUTION FAILED
║ Tool: browser_navigate
║ Error Type: execution
║ Error: failed to initialize browser: ...
╚══════════════════════════════════════════════════════════════
```

**Common fixes:**
- Install Playwright browsers: `go run github.com/playwright-community/playwright-go/cmd/playwright install`
- Check `.agi/browsers/` permissions
- Close existing browser instances

#### Element Not Found

**Debug shows:**
```
║ Error: timeout waiting for selector: button.submit
```

**Solutions:**
1. Use `browser_get_elements` first to see all available selectors
2. Wait for page load before clicking
3. Check if element is in iframe
4. Try clicking by coordinates instead

### Tool Execution Issues

#### Tool Not Found

```
║ Error Type: validation
║ Error: tool not found: browser_navigte
```

Fix typo in tool name: `browser_navigate`

#### Missing Arguments

```
║ Error Type: validation
║ Error: task_id and url are required
```

Provide all required arguments.

#### Timeout

```
║ Error Type: timeout
║ Error: operation exceeded 60s timeout
```

Increase timeout or optimize operation.

---

## 📊 Performance Tuning

### Browser Performance

| Setting | Fast | Balanced | Stealthy |
|---------|------|----------|----------|
| `headless` | true | false | false |
| `stealth` | false | true | true |
| `slow_mo` | 0 | 100 | 300 |

**Fast:** Fastest execution, high bot detection risk
**Balanced:** Good speed, moderate detection avoidance
**Stealthy:** Slower, best for avoiding detection

### Debug Performance

| Level | Performance Impact |
|-------|-------------------|
| Off | 0% (no overhead) |
| Basic | <1% |
| Verbose | ~2-3% |
| Trace | ~5-10% |

**Recommendation:** Use Verbose during development, Off in production.

---

## 🧪 Testing Tools

### Test All Tools

Create a test script:

```go
tools := []string{
    "read_file",
    "browser_navigate",
    "git_status",
}

for _, toolName := range tools {
    fmt.Printf("\nTesting: %s\n", toolName)

    result, err := executor.Execute(tools.ToolCall{
        Name: toolName,
        Arguments: getTestArgs(toolName),
    })

    if err != nil || !result.Success {
        fmt.Printf("❌ FAILED: %v\n", err)
    } else {
        fmt.Printf("✅ SUCCESS\n")
    }
}
```

### Browser Test Sequence

1. **Navigate** to test page
2. **Screenshot** initial state
3. **Get elements** map
4. **Click** button using selector
5. **Type** text into input
6. **Get text** from result element
7. **Screenshot** final state
8. **Close tab**

---

## 🎯 Best Practices

### Debug

1. ✅ **Enable during development** - Catch issues early
2. ✅ **Use Verbose level** - Good balance of detail vs noise
3. ✅ **Review failed traces** - Learn from errors
4. ✅ **Generate reports** - Track tool usage patterns
5. ✅ **Disable in production** - Unless investigating issues

### Browser

1. ✅ **Always use stealth mode** - Unless speed is critical
2. ✅ **Add delays for suspicious sites** - Increase SlowMo
3. ✅ **Use persistent sessions** - Builds reputation
4. ✅ **Get elements first** - Before clicking blind
5. ✅ **Handle errors gracefully** - Retry with different selectors

### Error Handling

1. ✅ **Check error type** - Validation vs execution vs panic
2. ✅ **Log stack traces** - For unexpected panics
3. ✅ **Retry transient errors** - Timeouts, network issues
4. ✅ **Fail fast on validation** - Don't retry bad arguments
5. ✅ **Add metadata** - Context helps debugging

---

## 🚀 Quick Reference

### Enable Debug
```json
{"debug_tools": true}
```

### Configure Browser
```json
{
  "browser": {
    "headless": false,
    "stealth": true,
    "slow_mo": 100
  }
}
```

### Get Debug Report
```go
report := executor.GetDebugReport()
```

### Check Recent Failures
```go
failures := executor.GetRecentFailures()
```

### Set Debug Level Programmatically
```go
tools.SetGlobalDebugLevel(tools.DebugVerbose)
```

---

## 📝 Configuration Examples

### Development (Max Debug)
```json
{
  "debug_tools": true,
  "browser": {
    "headless": false,
    "stealth": true,
    "slow_mo": 200
  }
}
```

### Production (No Debug, Fast)
```json
{
  "debug_tools": false,
  "browser": {
    "headless": true,
    "stealth": true,
    "slow_mo": 50
  }
}
```

### Testing (Visible, Detailed)
```json
{
  "debug_tools": true,
  "browser": {
    "headless": false,
    "stealth": false,
    "slow_mo": 0
  }
}
```

---

**All tools now have comprehensive debugging and browsers avoid bot detection! 🎉**
