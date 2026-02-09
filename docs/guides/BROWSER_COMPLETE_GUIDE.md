# 🌐 Browser Navigation - Complete & Enhanced Guide

**Status**: ✅ **ENHANCED**
**Features**: 9 Tools + AI-Optimized + Coordinates
**Instructions**: ✅ **Integrated in .agirules**

---

## 🎯 What's New

### ✅ Enhanced Features

1. **Built-in Instructions** - `.agirules` has detailed usage guide
2. **Element Mapping** - `browser_get_elements()` maps all interactive elements
3. **Coordinate Clicking** - `browser_click_coords(x, y)` for precise clicks
4. **AI-Optimized Screenshots** - Low-res (800x600), compressed for AI vision
5. **Error Prevention** - Detailed examples prevent common mistakes

---

## 🛠️ All 9 Tools

### Core Navigation

1. **`browser_navigate`** - Go to URL, create/reuse tab
2. **`browser_click`** - Click element by CSS selector
3. **`browser_type`** - Type text into input
4. **`browser_get_text`** - Extract text from element

### Advanced Features

5. **`browser_get_elements`** ✨ NEW - Map all interactive elements
6. **`browser_click_coords`** ✨ NEW - Click by coordinates
7. **`browser_screenshot`** - Full or AI-optimized screenshot
8. **`browser_close_tab`** - Close specific tab
9. **`browser_list_tabs`** - List all active tabs

---

## 🆕 Element Mapping Workflow

### Step 1: Navigate to Page

```json
{
  "name": "browser_navigate",
  "arguments": {
    "task_id": "research",
    "url": "https://example.com"
  }
}
```

### Step 2: Get Element Map

```json
{
  "name": "browser_get_elements",
  "arguments": {
    "task_id": "research"
  }
}
```

**Returns**:
```json
{
  "elements": [
    {
      "index": 0,
      "tag": "button",
      "text": "Search",
      "id": "search-btn",
      "class": "btn primary",
      "x": 450,
      "y": 120,
      "width": 100,
      "height": 40
    },
    {
      "index": 1,
      "tag": "input",
      "text": "",
      "id": "query",
      "class": "search-input",
      "x": 300,
      "y": 120,
      "width": 200,
      "height": 30
    }
  ],
  "count": 2
}
```

### Step 3: Click by Selector or Coordinates

**Option A: CSS Selector (recommended)**
```json
{
  "name": "browser_click",
  "arguments": {
    "task_id": "research",
    "selector": "#search-btn"
  }
}
```

**Option B: Coordinates (if selector fails)**
```json
{
  "name": "browser_click_coords",
  "arguments": {
    "task_id": "research",
    "x": 450,
    "y": 120
  }
}
```

---

## 📸 AI-Optimized Screenshots

### Full Quality (Default)

```json
{
  "name": "browser_screenshot",
  "arguments": {
    "task_id": "research",
    "path": "page-full.png"
  }
}
```

**Result**: Full resolution (1280x720), PNG, ~500KB

### AI-Optimized (For Vision Models)

```json
{
  "name": "browser_screenshot",
  "arguments": {
    "task_id": "research",
    "path": "page-ai.jpg",
    "optimized": true
  }
}
```

**Result**: Low-res (800x600), JPEG quality 60%, ~50KB

**Use AI-optimized when:**
- ✅ Sending to vision model (Claude, GPT-4V)
- ✅ Need fast processing
- ✅ Want to reduce token cost
- ✅ Mapping page layout

---

## 📋 Built-in Instructions

Instructions are **automatically loaded** from `workspace/.agirules`:

### What's Included

✅ **Exact JSON format** for each tool
✅ **CSS selector guide** with examples
✅ **Common mistakes** to avoid
✅ **Complete workflow** templates
✅ **Error prevention** tips
✅ **Debugging steps**

### Agent Reads This On Startup

The agent sees:
- Mandatory workflow (navigate → interact → close)
- Tool call examples in exact format
- CSS selector reference
- Error handling guide

**Result**: Reduces hallucinations and tool misuse!

---

## 🎓 Usage Patterns

### Pattern 1: Map Page Then Interact

```
1. Navigate
   browser_navigate(task_id="shop", url="https://shop.com")

2. Map elements
   browser_get_elements(task_id="shop")
   → See all buttons, inputs, links with positions

3. Take AI screenshot to verify
   browser_screenshot(task_id="shop", path="shop.jpg", optimized=true)

4. Click element
   browser_click(task_id="shop", selector="#add-to-cart")
   OR
   browser_click_coords(task_id="shop", x=500, y=300)

5. Clean up
   browser_close_tab(task_id="shop")
```

### Pattern 2: Visual Navigation

```
1. Navigate
   browser_navigate(task_id="visual", url="https://complex-site.com")

2. Screenshot for AI analysis
   browser_screenshot(task_id="visual", path="page.jpg", optimized=true)

3. AI analyzes image, identifies elements

4. Click by coordinates from visual analysis
   browser_click_coords(task_id="visual", x=250, y=400)

5. Extract result
   browser_get_text(task_id="visual", selector=".result")
```

### Pattern 3: Fallback Strategy

```
1. Try CSS selector first (fast)
   browser_click(task_id="task", selector="button.submit")

2. If fails, map elements
   browser_get_elements(task_id="task")

3. Find button in map, use coordinates
   browser_click_coords(task_id="task", x=450, y=200)
```

---

## 🔍 Element Map Details

### What's Included

**Every interactive element**:
- Links (`<a>`)
- Buttons (`<button>`)
- Inputs (`<input>`, `<textarea>`, `<select>`)
- Clickable elements (`[onclick]`, `[role="button"]`)

**Filters out**:
- Hidden elements
- Zero-size elements
- Elements outside viewport
- Invisible elements

**Limited to**: First 50 elements (most relevant)

### Element Info Structure

```json
{
  "index": 0,           // Position in list
  "tag": "button",      // HTML tag
  "text": "Click me",   // Visible text (first 50 chars)
  "id": "btn-submit",   // Element ID (if exists)
  "class": "btn",       // CSS classes
  "x": 450,             // Click X coordinate (center)
  "y": 200,             // Click Y coordinate (center)
  "width": 100,         // Element width
  "height": 40          // Element height
}
```

---

## 🎯 Coordinate System

### How It Works

```
Browser viewport: 1280x720 (default)

(0,0) ──────────────────── (1280,0)
  │                            │
  │    Element at (450, 200)   │
  │         ┌────┐             │
  │         │ BTN│             │
  │         └────┘             │
  │                            │
(0,720) ─────────────────── (1280,720)
```

**Coordinates are**:
- Center of element (x, y)
- Relative to viewport top-left
- In pixels

**AI-Optimized viewport**: 800x600
- Coordinates scale proportionally
- Still accurate for clicking

---

## 💡 Best Practices

### When to Use Each Approach

| Scenario | Method | Why |
|----------|--------|-----|
| Element has ID | CSS Selector `#id` | Fastest, most reliable |
| Element has class | CSS Selector `.class` | Fast, reliable |
| Complex page | Map elements first | Understand structure |
| Selector fails | Coordinates fallback | Works when CSS fails |
| AI vision task | AI-optimized screenshot | Lower cost, faster |
| Need proof | Full screenshot | Better quality |

### Optimization Tips

1. **Always map elements first** on complex pages
2. **Use AI-optimized screenshots** for vision models
3. **Prefer CSS selectors** over coordinates (faster)
4. **Close tabs immediately** after task done
5. **Reuse task_ids** for same logical task

---

## 🐛 Error Prevention

### Common Issues & Solutions

#### Issue: "Element not found"

**Solutions**:
```
1. Map elements:
   browser_get_elements(task_id="debug")

2. Take screenshot:
   browser_screenshot(task_id="debug", path="debug.jpg", optimized=true)

3. Try broader selector:
   "button" instead of "button.specific-class"

4. Use coordinates as fallback:
   browser_click_coords(task_id="debug", x=X, y=Y)
```

#### Issue: Selector confusion

**Wrong**:
```json
{"selector": "the submit button"}  // Natural language
{"selector": "click here"}          // Not a selector
```

**Right**:
```json
{"selector": "#submit-btn"}        // By ID
{"selector": "button.submit"}      // By class
{"selector": "button[type='submit']"}  // By attribute
```

#### Issue: Wrong coordinates

**Solution**: Always get from `browser_get_elements()`, don't guess!

---

## 📊 Tool Comparison

| Tool | Speed | Reliability | Use When |
|------|-------|-------------|----------|
| `browser_click` (selector) | Fast | High | Element has ID/class |
| `browser_click_coords` | Fast | Medium | Selector fails |
| `browser_get_elements` | Medium | High | Need page map |
| `browser_screenshot` (full) | Slow | High | Need quality |
| `browser_screenshot` (AI) | Fast | High | For AI vision |

---

## 🎮 Complete Example

### Task: "Research Python asyncio documentation"

```json
// Step 1: Navigate
{
  "name": "browser_navigate",
  "arguments": {
    "task_id": "python-docs",
    "url": "https://docs.python.org"
  }
}

// Step 2: Map page elements
{
  "name": "browser_get_elements",
  "arguments": {
    "task_id": "python-docs"
  }
}
// Returns: [{index: 0, tag: "input", id: "search", x: 300, y: 50}, ...]

// Step 3: Take AI screenshot for context
{
  "name": "browser_screenshot",
  "arguments": {
    "task_id": "python-docs",
    "path": "python-docs.jpg",
    "optimized": true
  }
}
// Saved: 800x600 JPEG, ~50KB

// Step 4: Type in search
{
  "name": "browser_type",
  "arguments": {
    "task_id": "python-docs",
    "selector": "#search",
    "text": "asyncio"
  }
}

// Step 5: Click search button
{
  "name": "browser_click",
  "arguments": {
    "task_id": "python-docs",
    "selector": "button.search"
  }
}
// OR if selector fails:
{
  "name": "browser_click_coords",
  "arguments": {
    "task_id": "python-docs",
    "x": 450,
    "y": 50
  }
}

// Step 6: Extract result
{
  "name": "browser_get_text",
  "arguments": {
    "task_id": "python-docs",
    "selector": ".result-title"
  }
}

// Step 7: Close tab
{
  "name": "browser_close_tab",
  "arguments": {
    "task_id": "python-docs"
  }
}
```

---

## ✅ Summary

### Features

✅ **9 tools** for complete browser control
✅ **Element mapping** - see all clickable elements
✅ **Coordinate clicking** - fallback when selectors fail
✅ **AI-optimized screenshots** - 800x600, compressed
✅ **Built-in instructions** - in `.agirules` file
✅ **Error prevention** - examples prevent hallucinations

### Best Workflow

1. Navigate (`browser_navigate`)
2. Map elements (`browser_get_elements`)
3. Screenshot if needed (`browser_screenshot` optimized)
4. Interact (`browser_click` or `browser_click_coords`)
5. Extract data (`browser_get_text`)
6. Clean up (`browser_close_tab`)

---

**Status**: ✅ **PRODUCTION READY**
**Build**: 13MB
**Instructions**: ✅ **In workspace/.agirules**
**AI-Friendly**: ✅ **Optimized screenshots + element maps**

*Intelligent web navigation with AI-optimized workflows!* 🌐🤖
