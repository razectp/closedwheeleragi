# 🌐 Browser Navigation - Quick Guide

**Status**: ✅ **IMPLEMENTED**
**Engine**: Playwright (Chromium)
**Tab Management**: ✅ **Task-Specific**

---

## 🎯 Overview

Legitimate web navigation with **intelligent tab management**. Each task gets its own dedicated tab - no more opening multiple tabs per request!

### Key Features

✅ **Task-Specific Tabs** - One tab per task_id
✅ **Reuse Tabs** - Same task_id reuses existing tab
✅ **Smart Management** - Automatic cleanup on close
✅ **Full Control** - Click, type, screenshot, extract text
✅ **Headless Mode** - Runs in background by default

---

## 🚀 Available Tools

### 1. `browser_navigate`

Navigate to a URL. Creates tab if needed, reuses if exists.

```json
{
  "task_id": "research-docs",
  "url": "https://example.com"
}
```

**Example**:
```
"Navigate to Python docs using task_id 'python-docs'"
→ Creates tab 'python-docs'
→ Goes to python.org
```

### 2. `browser_click`

Click an element by CSS selector.

```json
{
  "task_id": "research-docs",
  "selector": "button.login"
}
```

### 3. `browser_type`

Type text into input field.

```json
{
  "task_id": "research-docs",
  "selector": "#search",
  "text": "machine learning"
}
```

### 4. `browser_get_text`

Extract text from element.

```json
{
  "task_id": "research-docs",
  "selector": "h1.title"
}
```

### 5. `browser_screenshot`

Take screenshot of page.

```json
{
  "task_id": "research-docs",
  "path": "docs-screenshot.png"
}
```

### 6. `browser_close_tab`

Close specific task tab.

```json
{
  "task_id": "research-docs"
}
```

### 7. `browser_list_tabs`

List all active tabs.

```json
{}
```

Returns:
```json
{
  "active_tabs": ["research-docs", "check-prices", "scrape-data"],
  "count": 3
}
```

---

## 💡 Usage Patterns

### Pattern 1: Research Task

```
Task: "Research Python async/await"

1. browser_navigate(task_id="python-research", url="https://docs.python.org")
2. browser_click(task_id="python-research", selector="#search-button")
3. browser_type(task_id="python-research", selector="input", text="async")
4. browser_get_text(task_id="python-research", selector=".result")
5. browser_close_tab(task_id="python-research")
```

**Key**: All use same `task_id` → Same tab reused!

### Pattern 2: Multiple Parallel Tasks

```
Task A: Check prices on Site A
→ task_id="price-site-a"

Task B: Check prices on Site B
→ task_id="price-site-b"

Task C: Compare results
→ Uses data from both tabs
→ Closes both: browser_close_tab for each
```

**Key**: Different task_ids → Separate tabs!

### Pattern 3: Multi-Step Navigation

```
1. Navigate to login page
   → task_id="shopping"

2. Fill login form
   → Same task_id, same tab

3. Navigate to products
   → Same task_id, same tab (navigation within)

4. Screenshot for verification
   → Same task_id

5. Close when done
   → browser_close_tab("shopping")
```

---

## 🔐 Security & Best Practices

### Safe Usage

✅ **Use descriptive task_ids** - Easy to track
✅ **Close tabs when done** - Free resources
✅ **One tab per logical task** - Clean separation
✅ **Respect robots.txt** - Be a good citizen
✅ **Add delays for rate limiting** - Don't spam

### Task ID Naming

```
Good:
- "research-python-docs"
- "price-check-amazon"
- "scrape-news-headlines"
- "login-github"

Bad:
- "task1"
- "a"
- "temp"
```

### Automatic Cleanup

- All tabs closed on agent shutdown
- Manual close: `browser_close_tab(task_id)`
- Check active: `browser_list_tabs()`

---

## 📊 Example Conversations

### Example 1: Simple Research

**User**: "Check what's on the Playwright website"

**Agent**:
```
Using browser_navigate:
- task_id: "playwright-check"
- url: "https://playwright.dev"

Result: Navigated successfully
Title: "Fast and reliable end-to-end testing for modern web apps"
Content: 15,234 chars

Want me to extract specific information?
```

### Example 2: Multi-Step

**User**: "Go to Python.org, search for 'asyncio', and get the first result title"

**Agent**:
```
Step 1: Navigate to Python.org
→ browser_navigate(task_id="python-search", url="https://python.org")

Step 2: Click search button
→ browser_click(task_id="python-search", selector="#search-btn")

Step 3: Type 'asyncio'
→ browser_type(task_id="python-search", selector="#q", text="asyncio")

Step 4: Get first result
→ browser_get_text(task_id="python-search", selector=".result:first h3")

Result: "asyncio — Asynchronous I/O"

Closing tab...
→ browser_close_tab(task_id="python-search")
```

### Example 3: Parallel Tasks

**User**: "Check prices on Amazon and eBay for 'laptop', compare"

**Agent**:
```
Task 1: Amazon
→ browser_navigate(task_id="amazon-laptop", url="https://amazon.com")
→ [search and extract price]

Task 2: eBay
→ browser_navigate(task_id="ebay-laptop", url="https://ebay.com")
→ [search and extract price]

Comparison:
- Amazon: $899
- eBay: $849
→ eBay is $50 cheaper

Closing tabs...
→ browser_close_tab("amazon-laptop")
→ browser_close_tab("ebay-laptop")
```

---

## 🔧 Configuration

Browser runs with sensible defaults:

```go
Headless: true (runs in background)
Timeout: 30 seconds
User Agent: Chrome (modern)
Viewport: 1280x720
```

To change, edit `pkg/browser/browser.go`:

```go
func DefaultOptions() *Options {
	return &Options{
		Headless:       false,  // Show browser window
		DefaultTimeout: 60 * time.Second,  // 1 minute
		ViewportWidth:  1920,  // Full HD
		ViewportHeight: 1080,
	}
}
```

---

## 🐛 Troubleshooting

### Issue: "Failed to install playwright"

**Solution**:
```bash
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install
```

### Issue: Tab not found

**Solution**: Check task_id spelling, or list active tabs:
```
browser_list_tabs() → See all active tabs
```

### Issue: Element not found

**Solution**:
1. Check CSS selector is correct
2. Wait for element: Use `browser_get_text` after delay
3. Take screenshot to debug: `browser_screenshot`

### Issue: Too many tabs open

**Solution**:
```
browser_list_tabs() → See all
browser_close_tab(task_id) → Close each
```

Or restart agent (closes all automatically).

---

## 📚 Commands

### Check Active Tabs

```
/model                  # In case browser is slow
browser_list_tabs()     # Via agent
```

### Clean Up

```
Ctrl+C                  # Stops agent, closes all tabs
browser_close_tab(id)   # Close specific tab
```

### Reload Config

```
/config reload          # Reload settings without restart
```

---

## ✅ Summary

**Features**:
- ✅ Task-specific tab management
- ✅ No duplicate tabs per request
- ✅ Full browser automation
- ✅ Screenshot support
- ✅ Clean and automatic cleanup

**7 Tools Available**:
1. `browser_navigate` - Go to URL
2. `browser_click` - Click element
3. `browser_type` - Type text
4. `browser_get_text` - Extract text
5. `browser_screenshot` - Take screenshot
6. `browser_close_tab` - Close tab
7. `browser_list_tabs` - List active tabs

**Status**: ✅ **PRODUCTION READY**
**Build**: 13MB (includes Playwright)

*Navigate the web intelligently!* 🌐
