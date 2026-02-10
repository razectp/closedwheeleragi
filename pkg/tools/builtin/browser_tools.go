package builtin

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"ClosedWheeler/pkg/browser"
	"ClosedWheeler/pkg/tools"
)

var browserManager *browser.Manager
var browserConfig *browser.Options

// SetBrowserOptions sets custom browser options (call before registering tools).
func SetBrowserOptions(opts *browser.Options) {
	browserConfig = opts
}

// RegisterBrowserTools registers all browser and fetch tools.
// Returns an error only if the browser manager itself cannot be created.
func RegisterBrowserTools(registry *tools.Registry, projectRoot string) error {
	if browserManager == nil {
		opts := browserConfig
		if opts == nil {
			opts = browser.DefaultOptions()
		}
		if projectRoot != "" {
			opts.CachePath = filepath.Join(projectRoot, "browsers")
		}
		var err error
		browserManager, err = browser.NewManager(opts)
		if err != nil {
			return fmt.Errorf("failed to initialize browser: %w", err)
		}
	}

	registerWebFetch(registry)
	registerBrowserNavigate(registry)
	registerBrowserGetPageText(registry)
	registerBrowserClick(registry)
	registerBrowserType(registry)
	registerBrowserGetText(registry)
	registerBrowserScreenshot(registry)
	registerBrowserCloseTab(registry)
	registerBrowserListTabs(registry)
	registerBrowserGetElements(registry)
	registerBrowserClickCoords(registry)
	registerBrowserEval(registry)

	return nil
}

// CloseBrowserManager closes the browser manager on shutdown.
func CloseBrowserManager() error {
	if browserManager != nil {
		return browserManager.Close()
	}
	return nil
}

// guard validates that required string args are present and non-empty.
// Returns a ToolResult error if any key is missing.
func guardStrings(args map[string]any, keys ...string) (map[string]string, *tools.ToolResult) {
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		v, _ := args[k].(string)
		if v == "" {
			r := tools.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("missing required parameter: %s", k),
			}
			return nil, &r
		}
		out[k] = v
	}
	return out, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// web_fetch — fast HTTP fetch, no browser needed
// ──────────────────────────────────────────────────────────────────────────────

func registerWebFetch(registry *tools.Registry) {
	registry.Register(&tools.Tool{
		Name: "web_fetch",
		Description: `Fetch a web page and return its readable content WITHOUT launching a browser.
Much faster than browser_navigate. Use this for: documentation, articles, APIs, GitHub, Wikipedia.
Use browser_navigate only when JavaScript rendering is required (SPAs, login flows, dynamic content).`,
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"url":     {Type: "string", Description: "URL to fetch (http/https)"},
				"format":  {Type: "string", Description: `Output format: "text" (default) or "markdown"`},
				"timeout": {Type: "number", Description: "Timeout in seconds (default 30, max 120)"},
			},
			Required: []string{"url"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			url, _ := args["url"].(string)
			if url == "" {
				return tools.ToolResult{Success: false, Error: "missing required parameter: url"}, nil
			}
			format, _ := args["format"].(string)
			timeoutF, _ := args["timeout"].(float64)
			timeoutSecs := int(timeoutF)
			if timeoutSecs <= 0 {
				timeoutSecs = 30
			}
			if timeoutSecs > 120 {
				timeoutSecs = 120
			}

			result, err := browser.FetchPage(url, timeoutSecs)
			if err != nil {
				return tools.ToolResult{Success: false, Error: fmt.Sprintf("web_fetch failed: %v", err)}, nil
			}

			content := result.Text
			if format == "markdown" {
				content = result.Markdown
			}

			header := fmt.Sprintf("URL: %s\nTitle: %s\nStatus: %d\n\n", result.URL, result.Title, result.StatusCode)
			return tools.ToolResult{Success: true, Output: header + content}, nil
		},
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// browser_navigate — open a URL in a real Chrome browser
// ──────────────────────────────────────────────────────────────────────────────

func registerBrowserNavigate(registry *tools.Registry) {
	registry.Register(&tools.Tool{
		Name: "browser_navigate",
		Description: `Navigate to a URL in a real Chrome browser (supports JS-rendered pages).
IMPORTANT: You MUST call this first before any other browser_* tool.
The task_id identifies the browser session — use the same task_id for all subsequent operations.
For static pages prefer web_fetch (faster, no Chrome required).`,
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"task_id": {Type: "string", Description: "Session identifier, e.g. 'research-1'. Reuse across calls to keep the same tab open."},
				"url":     {Type: "string", Description: "Full URL to navigate to (include https://)"},
			},
			Required: []string{"task_id", "url"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			params, bad := guardStrings(args, "task_id", "url")
			if bad != nil {
				return *bad, nil
			}
			result, err := browserManager.Navigate(params["task_id"], params["url"])
			if err != nil {
				return tools.ToolResult{Success: false, Error: fmt.Sprintf("browser_navigate: %v", err)}, nil
			}
			output := fmt.Sprintf("URL: %s\nTitle: %s\nStatus: %d\n\n%s",
				result.URL, result.Title, result.StatusCode, result.Content)
			return tools.ToolResult{Success: true, Output: output}, nil
		},
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// browser_get_page_text — get full text of the current page
// ──────────────────────────────────────────────────────────────────────────────

func registerBrowserGetPageText(registry *tools.Registry) {
	registry.Register(&tools.Tool{
		Name: "browser_get_page_text",
		Description: `Get the full visible text content of the currently loaded page.
Use this after browser_navigate to read page content without re-navigating.
Returns up to 10000 characters of clean text (scripts/styles stripped).`,
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"task_id": {Type: "string", Description: "Session identifier (must have called browser_navigate first)"},
			},
			Required: []string{"task_id"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			params, bad := guardStrings(args, "task_id")
			if bad != nil {
				return *bad, nil
			}
			text, err := browserManager.GetPageText(params["task_id"])
			if err != nil {
				return tools.ToolResult{Success: false, Error: fmt.Sprintf("browser_get_page_text: %v", err)}, nil
			}
			return tools.ToolResult{Success: true, Output: text}, nil
		},
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// browser_click
// ──────────────────────────────────────────────────────────────────────────────

func registerBrowserClick(registry *tools.Registry) {
	registry.Register(&tools.Tool{
		Name:        "browser_click",
		Description: "Click an element using a CSS selector. Use browser_get_elements first to discover valid selectors. Requires browser_navigate to have been called first.",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"task_id":  {Type: "string", Description: "Session identifier"},
				"selector": {Type: "string", Description: "CSS selector, e.g. 'button.submit', '#login-btn'"},
			},
			Required: []string{"task_id", "selector"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			params, bad := guardStrings(args, "task_id", "selector")
			if bad != nil {
				return *bad, nil
			}
			if err := browserManager.Click(params["task_id"], params["selector"]); err != nil {
				return tools.ToolResult{Success: false, Error: fmt.Sprintf("browser_click(%q): %v", params["selector"], err)}, nil
			}
			return tools.ToolResult{Success: true, Output: fmt.Sprintf("Clicked: %s", params["selector"])}, nil
		},
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// browser_type
// ──────────────────────────────────────────────────────────────────────────────

func registerBrowserType(registry *tools.Registry) {
	registry.Register(&tools.Tool{
		Name:        "browser_type",
		Description: "Type text into a form input. Clears the field first. Requires browser_navigate to have been called first.",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"task_id":  {Type: "string", Description: "Session identifier"},
				"selector": {Type: "string", Description: "CSS selector for the input element"},
				"text":     {Type: "string", Description: "Text to type into the field"},
			},
			Required: []string{"task_id", "selector", "text"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			params, bad := guardStrings(args, "task_id", "selector", "text")
			if bad != nil {
				return *bad, nil
			}
			if err := browserManager.Type(params["task_id"], params["selector"], params["text"]); err != nil {
				return tools.ToolResult{Success: false, Error: fmt.Sprintf("browser_type(%q): %v", params["selector"], err)}, nil
			}
			return tools.ToolResult{Success: true, Output: fmt.Sprintf("Typed into %s", params["selector"])}, nil
		},
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// browser_get_text
// ──────────────────────────────────────────────────────────────────────────────

func registerBrowserGetText(registry *tools.Registry) {
	registry.Register(&tools.Tool{
		Name:        "browser_get_text",
		Description: "Extract the visible text of a specific element by CSS selector. Use browser_get_page_text to get all page text.",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"task_id":  {Type: "string", Description: "Session identifier"},
				"selector": {Type: "string", Description: "CSS selector for the target element"},
			},
			Required: []string{"task_id", "selector"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			params, bad := guardStrings(args, "task_id", "selector")
			if bad != nil {
				return *bad, nil
			}
			text, err := browserManager.GetText(params["task_id"], params["selector"])
			if err != nil {
				return tools.ToolResult{Success: false, Error: fmt.Sprintf("browser_get_text(%q): %v", params["selector"], err)}, nil
			}
			return tools.ToolResult{Success: true, Output: text}, nil
		},
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// browser_screenshot
// ──────────────────────────────────────────────────────────────────────────────

func registerBrowserScreenshot(registry *tools.Registry) {
	registry.Register(&tools.Tool{
		Name: "browser_screenshot",
		Description: `Take a screenshot of the current page and save it to a file.
REQUIRES browser_navigate to have been called first.
Use optimized=true for a compact 800x600 image suitable for LLM vision.`,
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"task_id":   {Type: "string", Description: "Session identifier"},
				"path":      {Type: "string", Description: "File path to save screenshot, e.g. 'workplace/screenshot.png'"},
				"optimized": {Type: "boolean", Description: "If true, saves 800x600 image for LLM vision (default: false = full page)"},
			},
			Required: []string{"task_id", "path"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			params, bad := guardStrings(args, "task_id", "path")
			if bad != nil {
				return *bad, nil
			}
			optimized, _ := args["optimized"].(bool)
			var err error
			if optimized {
				err = browserManager.ScreenshotOptimized(params["task_id"], params["path"])
			} else {
				err = browserManager.Screenshot(params["task_id"], params["path"])
			}
			if err != nil {
				return tools.ToolResult{Success: false, Error: fmt.Sprintf("browser_screenshot: %v", err)}, nil
			}
			mode := "full-page PNG"
			if optimized {
				mode = "optimized 800x600 PNG"
			}
			return tools.ToolResult{Success: true, Output: fmt.Sprintf("Screenshot saved: %s (%s)", params["path"], mode)}, nil
		},
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// browser_close_tab
// ──────────────────────────────────────────────────────────────────────────────

func registerBrowserCloseTab(registry *tools.Registry) {
	registry.Register(&tools.Tool{
		Name:        "browser_close_tab",
		Description: "Close the browser tab for a task and free its resources. Call this when done browsing.",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"task_id": {Type: "string", Description: "Session identifier to close"},
			},
			Required: []string{"task_id"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			params, bad := guardStrings(args, "task_id")
			if bad != nil {
				return *bad, nil
			}
			if err := browserManager.ClosePage(params["task_id"]); err != nil {
				return tools.ToolResult{Success: false, Error: fmt.Sprintf("browser_close_tab: %v", err)}, nil
			}
			return tools.ToolResult{Success: true, Output: fmt.Sprintf("Closed browser tab: %s", params["task_id"])}, nil
		},
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// browser_list_tabs
// ──────────────────────────────────────────────────────────────────────────────

func registerBrowserListTabs(registry *tools.Registry) {
	registry.Register(&tools.Tool{
		Name:        "browser_list_tabs",
		Description: "List all currently open browser sessions (task IDs).",
		Parameters:  &tools.JSONSchema{Type: "object", Properties: map[string]tools.Property{}},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			tasks := browserManager.GetActiveTasks()
			if len(tasks) == 0 {
				return tools.ToolResult{Success: true, Output: "No active browser sessions."}, nil
			}
			out, _ := json.MarshalIndent(map[string]any{"active_sessions": tasks, "count": len(tasks)}, "", "  ")
			return tools.ToolResult{Success: true, Output: string(out)}, nil
		},
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// browser_get_elements
// ──────────────────────────────────────────────────────────────────────────────

func registerBrowserGetElements(registry *tools.Registry) {
	registry.Register(&tools.Tool{
		Name:        "browser_get_elements",
		Description: "Get visible interactive elements (buttons, links, inputs) with their CSS selectors and X,Y coordinates. Use this before browser_click to find the right target.",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"task_id": {Type: "string", Description: "Session identifier"},
			},
			Required: []string{"task_id"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			params, bad := guardStrings(args, "task_id")
			if bad != nil {
				return *bad, nil
			}
			elements, err := browserManager.GetPageElements(params["task_id"])
			if err != nil {
				return tools.ToolResult{Success: false, Error: fmt.Sprintf("browser_get_elements: %v", err)}, nil
			}
			out, _ := json.MarshalIndent(map[string]any{"elements": elements, "count": len(elements)}, "", "  ")
			return tools.ToolResult{Success: true, Output: string(out)}, nil
		},
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// browser_click_coords
// ──────────────────────────────────────────────────────────────────────────────

func registerBrowserClickCoords(registry *tools.Registry) {
	registry.Register(&tools.Tool{
		Name:        "browser_click_coords",
		Description: "Click at exact X,Y pixel coordinates. Use browser_get_elements to find element positions first.",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"task_id": {Type: "string", Description: "Session identifier"},
				"x":       {Type: "number", Description: "X coordinate in pixels"},
				"y":       {Type: "number", Description: "Y coordinate in pixels"},
			},
			Required: []string{"task_id", "x", "y"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			params, bad := guardStrings(args, "task_id")
			if bad != nil {
				return *bad, nil
			}
			x, _ := args["x"].(float64)
			y, _ := args["y"].(float64)
			if err := browserManager.ClickCoordinates(params["task_id"], x, y); err != nil {
				return tools.ToolResult{Success: false, Error: fmt.Sprintf("browser_click_coords(%d,%d): %v", int(x), int(y), err)}, nil
			}
			return tools.ToolResult{Success: true, Output: fmt.Sprintf("Clicked at (%d, %d)", int(x), int(y))}, nil
		},
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// browser_eval
// ──────────────────────────────────────────────────────────────────────────────

func registerBrowserEval(registry *tools.Registry) {
	registry.Register(&tools.Tool{
		Name:        "browser_eval",
		Description: "Execute JavaScript in the current browser page and return the result as JSON. Useful for extracting data or triggering actions. Requires browser_navigate first.",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"task_id": {Type: "string", Description: "Session identifier"},
				"script":  {Type: "string", Description: "JavaScript to evaluate (must return a JSON-serialisable value)"},
			},
			Required: []string{"task_id", "script"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			params, bad := guardStrings(args, "task_id", "script")
			if bad != nil {
				return *bad, nil
			}
			result, err := browserManager.EvaluateJS(params["task_id"], params["script"])
			if err != nil {
				return tools.ToolResult{Success: false, Error: fmt.Sprintf("browser_eval: %v", err)}, nil
			}
			return tools.ToolResult{Success: true, Output: result}, nil
		},
	})
}
