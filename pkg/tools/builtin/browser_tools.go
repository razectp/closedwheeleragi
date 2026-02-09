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

// SetBrowserOptions sets custom browser options (call before registering tools)
func SetBrowserOptions(opts *browser.Options) {
	browserConfig = opts
}

// RegisterBrowserTools registers web navigation tools
func RegisterBrowserTools(registry *tools.Registry, projectRoot string) error {
	// Initialize browser manager lazily
	if browserManager == nil {
		var err error
		opts := browserConfig
		if opts == nil {
			opts = browser.DefaultOptions()
		}
		if projectRoot != "" {
			opts.CachePath = filepath.Join(projectRoot, "browsers")
		}
		browserManager, err = browser.NewManager(opts)
		if err != nil {
			return fmt.Errorf("failed to initialize browser: %w", err)
		}
	}

	// Navigate to URL
	registry.Register(&tools.Tool{
		Name:        "browser_navigate",
		Description: "Navigate to a URL in a browser. Creates a dedicated tab for the task if needed.",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"task_id": {
					Type:        "string",
					Description: "Unique identifier for this browsing task (e.g., 'research-ai', 'check-docs')",
				},
				"url": {
					Type:        "string",
					Description: "URL to navigate to",
				},
			},
			Required: []string{"task_id", "url"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			taskID, _ := args["task_id"].(string)
			url, _ := args["url"].(string)

			if taskID == "" || url == "" {
				return tools.ToolResult{
					Success: false,
					Error:   "task_id and url are required",
				}, nil
			}

			result, err := browserManager.Navigate(taskID, url)
			if err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   err.Error(),
				}, nil
			}

			output := fmt.Sprintf("Navigated to: %s\nTitle: %s\nStatus: %d\nContent length: %d chars",
				result.URL, result.Title, result.StatusCode, len(result.Content))

			return tools.ToolResult{
				Success: true,
				Output:  output,
			}, nil
		},
	})

	// Click element
	registry.Register(&tools.Tool{
		Name:        "browser_click",
		Description: "Click an element on the page using a CSS selector.",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"task_id": {
					Type:        "string",
					Description: "Task identifier for the browser tab",
				},
				"selector": {
					Type:        "string",
					Description: "CSS selector for the element to click (e.g., 'button.submit', '#login-btn')",
				},
			},
			Required: []string{"task_id", "selector"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			taskID, _ := args["task_id"].(string)
			selector, _ := args["selector"].(string)

			if err := browserManager.Click(taskID, selector); err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   err.Error(),
				}, nil
			}

			return tools.ToolResult{
				Success: true,
				Output:  fmt.Sprintf("Clicked element: %s", selector),
			}, nil
		},
	})

	// Type text
	registry.Register(&tools.Tool{
		Name:        "browser_type",
		Description: "Type text into an input field using a CSS selector.",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"task_id": {
					Type:        "string",
					Description: "Task identifier for the browser tab",
				},
				"selector": {
					Type:        "string",
					Description: "CSS selector for the input element",
				},
				"text": {
					Type:        "string",
					Description: "Text to type",
				},
			},
			Required: []string{"task_id", "selector", "text"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			taskID, _ := args["task_id"].(string)
			selector, _ := args["selector"].(string)
			text, _ := args["text"].(string)

			if err := browserManager.Type(taskID, selector, text); err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   err.Error(),
				}, nil
			}

			return tools.ToolResult{
				Success: true,
				Output:  fmt.Sprintf("Typed text into: %s", selector),
			}, nil
		},
	})

	// Get text from element
	registry.Register(&tools.Tool{
		Name:        "browser_get_text",
		Description: "Extract text content from an element using a CSS selector.",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"task_id": {
					Type:        "string",
					Description: "Task identifier for the browser tab",
				},
				"selector": {
					Type:        "string",
					Description: "CSS selector for the element",
				},
			},
			Required: []string{"task_id", "selector"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			taskID, _ := args["task_id"].(string)
			selector, _ := args["selector"].(string)

			text, err := browserManager.GetText(taskID, selector)
			if err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   err.Error(),
				}, nil
			}

			return tools.ToolResult{
				Success: true,
				Output:  text,
			}, nil
		},
	})

	// Take screenshot
	registry.Register(&tools.Tool{
		Name:        "browser_screenshot",
		Description: "Take a screenshot of the current page. Use 'optimized=true' for AI-readable lower resolution (800x600, compressed). Default is full quality.",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"task_id": {
					Type:        "string",
					Description: "Task identifier for the browser tab",
				},
				"path": {
					Type:        "string",
					Description: "File path to save the screenshot (e.g., 'screenshot.png' or 'page.jpg')",
				},
				"optimized": {
					Type:        "boolean",
					Description: "If true, creates AI-optimized screenshot (800x600, compressed). Default: false",
				},
			},
			Required: []string{"task_id", "path"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			taskID, _ := args["task_id"].(string)
			path, _ := args["path"].(string)
			optimized, _ := args["optimized"].(bool)

			var err error
			if optimized {
				err = browserManager.ScreenshotOptimized(taskID, path)
			} else {
				err = browserManager.Screenshot(taskID, path)
			}

			if err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   err.Error(),
				}, nil
			}

			mode := "full quality"
			if optimized {
				mode = "AI-optimized (800x600, compressed)"
			}

			return tools.ToolResult{
				Success: true,
				Output:  fmt.Sprintf("Screenshot saved to: %s (%s)", path, mode),
			}, nil
		},
	})

	// Close task tab
	registry.Register(&tools.Tool{
		Name:        "browser_close_tab",
		Description: "Close the browser tab for a specific task.",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"task_id": {
					Type:        "string",
					Description: "Task identifier for the browser tab to close",
				},
			},
			Required: []string{"task_id"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			taskID, _ := args["task_id"].(string)

			if err := browserManager.ClosePage(taskID); err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   err.Error(),
				}, nil
			}

			return tools.ToolResult{
				Success: true,
				Output:  fmt.Sprintf("Closed browser tab for task: %s", taskID),
			}, nil
		},
	})

	// List active tasks
	registry.Register(&tools.Tool{
		Name:        "browser_list_tabs",
		Description: "List all active browser tabs and their task IDs.",
		Parameters: &tools.JSONSchema{
			Type:       "object",
			Properties: map[string]tools.Property{},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			tasks := browserManager.GetActiveTasks()

			if len(tasks) == 0 {
				return tools.ToolResult{
					Success: true,
					Output:  "No active browser tabs",
				}, nil
			}

			result, _ := json.MarshalIndent(map[string]any{
				"active_tabs": tasks,
				"count":       len(tasks),
			}, "", "  ")

			return tools.ToolResult{
				Success: true,
				Output:  string(result),
			}, nil
		},
	})

	// Get page elements map
	registry.Register(&tools.Tool{
		Name:        "browser_get_elements",
		Description: "Get a map of all interactive elements on the page with their positions and info. Use this to understand page structure before interacting.",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"task_id": {
					Type:        "string",
					Description: "Task identifier for the browser tab",
				},
			},
			Required: []string{"task_id"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			taskID, _ := args["task_id"].(string)

			elements, err := browserManager.GetPageElements(taskID)
			if err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   err.Error(),
				}, nil
			}

			result, _ := json.MarshalIndent(map[string]any{
				"elements": elements,
				"count":    len(elements),
			}, "", "  ")

			return tools.ToolResult{
				Success: true,
				Output:  string(result),
			}, nil
		},
	})

	// Click by coordinates
	registry.Register(&tools.Tool{
		Name:        "browser_click_coords",
		Description: "Click at specific X,Y coordinates on the page. Use browser_get_elements first to get coordinates.",
		Parameters: &tools.JSONSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"task_id": {
					Type:        "string",
					Description: "Task identifier for the browser tab",
				},
				"x": {
					Type:        "number",
					Description: "X coordinate to click",
				},
				"y": {
					Type:        "number",
					Description: "Y coordinate to click",
				},
			},
			Required: []string{"task_id", "x", "y"},
		},
		Handler: func(args map[string]any) (tools.ToolResult, error) {
			taskID, _ := args["task_id"].(string)
			x, _ := args["x"].(float64)
			y, _ := args["y"].(float64)

			if err := browserManager.ClickCoordinates(taskID, x, y); err != nil {
				return tools.ToolResult{
					Success: false,
					Error:   err.Error(),
				}, nil
			}

			return tools.ToolResult{
				Success: true,
				Output:  fmt.Sprintf("Clicked at coordinates (%d, %d)", int(x), int(y)),
			}, nil
		},
	})

	return nil
}

// CloseBrowserManager closes the browser manager (call on shutdown)
func CloseBrowserManager() error {
	if browserManager != nil {
		return browserManager.Close()
	}
	return nil
}
