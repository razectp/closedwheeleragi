// Package browser provides web navigation capabilities using Playwright
package browser

import (
	"fmt"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
)

// Manager handles browser instances and tabs
type Manager struct {
	pw       *playwright.Playwright
	browser  playwright.Browser
	pages    map[string]playwright.Page // taskID -> page
	pagesMux sync.RWMutex
	options  *Options
	context  playwright.BrowserContext // For persistent contexts
	initMux  sync.Mutex                // For lazy initialization
}

// Options configures the browser manager
type Options struct {
	Headless       bool
	DefaultTimeout time.Duration
	UserAgent      string
	ViewportWidth  int
	ViewportHeight int
	CachePath      string // Path for persistent browser data (cache, cookies, etc.)
	Stealth        bool   // Enable stealth mode to avoid bot detection
	SlowMo         int    // Milliseconds to slow down operations (helps with detection)
}

// DefaultOptions returns sensible defaults with anti-detection
func DefaultOptions() *Options {
	return &Options{
		Headless:       false,
		DefaultTimeout: 60 * time.Second,
		// Real Chrome user agent
		UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		ViewportWidth:  1920, // More realistic viewport
		ViewportHeight: 1080,
		Stealth:        true,  // Enable stealth mode by default
		SlowMo:         100,   // Add slight delay to appear more human
	}
}

// NewManager creates a new browser manager
func NewManager(opts *Options) (*Manager, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	return &Manager{
		pages:   make(map[string]playwright.Page),
		options: opts,
	}, nil
}

// start launches the playwright driver and browser instance
func (m *Manager) start() error {
	// Install Playwright browsers if needed
	err := playwright.Install(&playwright.RunOptions{
		Verbose: false,
	})
	if err != nil {
		return fmt.Errorf("failed to install playwright: %w", err)
	}

	// Start Playwright
	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("failed to start playwright: %w", err)
	}

	// Launch browser (persistent or regular)
	var browserInstance playwright.Browser
	var context playwright.BrowserContext
	var errLaunch error

	// Anti-detection arguments
	args := []string{
		"--disable-blink-features=AutomationControlled", // Hide automation
		"--disable-dev-shm-usage",
		"--no-sandbox",
		"--disable-setuid-sandbox",
		"--disable-web-security",
		"--disable-features=IsolateOrigins,site-per-process",
		"--disable-infobars",
		"--window-size=1920,1080",
		"--start-maximized",
	}

	if m.options.CachePath != "" {
		context, errLaunch = pw.Chromium.LaunchPersistentContext(m.options.CachePath, playwright.BrowserTypeLaunchPersistentContextOptions{
			Headless:  playwright.Bool(m.options.Headless),
			UserAgent: playwright.String(m.options.UserAgent),
			Viewport: &playwright.Size{
				Width:  m.options.ViewportWidth,
				Height: m.options.ViewportHeight,
			},
			Args:              args,
			SlowMo:            playwright.Float(float64(m.options.SlowMo)),
			JavaScriptEnabled: playwright.Bool(true),
			AcceptDownloads:   playwright.Bool(true),
			IgnoreHttpsErrors: playwright.Bool(true),
			Locale:            playwright.String("en-US"),
			TimezoneId:        playwright.String("America/New_York"),
		})
	} else {
		browserInstance, errLaunch = pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
			Headless: playwright.Bool(m.options.Headless),
			Args:     args,
			SlowMo:   playwright.Float(float64(m.options.SlowMo)),
		})
	}

	if errLaunch != nil {
		pw.Stop()
		return fmt.Errorf("failed to launch browser: %w", errLaunch)
	}

	m.pw = pw
	m.browser = browserInstance
	m.context = context
	return nil
}

// EnsureStarted ensures the browser is running and ready
func (m *Manager) EnsureStarted() error {
	m.initMux.Lock()
	defer m.initMux.Unlock()

	// If already started and connected, nothing to do
	if m.pw != nil && m.isConnected() {
		return nil
	}

	// If it was started but is now disconnected, clean up before restarting
	if m.pw != nil {
		m.cleanup()
	}

	return m.start()
}

// isConnected checks if the browser instance is still active
func (m *Manager) isConnected() bool {
	if m.pw == nil {
		return false
	}

	// Playwright can sometimes panic internally if the browser process was killed
	// but the Go objects still exist. We wrap this in a recovery block.
	defer func() {
		if r := recover(); r != nil {
			// If we panic during check, assume it's disconnected
			return
		}
	}()

	if m.context != nil {
		// For persistent context, context.Browser() is available
		b := m.context.Browser()
		if b == nil {
			return false
		}
		return b.IsConnected()
	}

	if m.browser != nil {
		return m.browser.IsConnected()
	}

	return false
}

// cleanup resets the manager state for a fresh start
func (m *Manager) cleanup() {
	m.pagesMux.Lock()
	m.pages = make(map[string]playwright.Page)
	m.pagesMux.Unlock()

	if m.browser != nil {
		m.browser.Close()
		m.browser = nil
	}
	if m.context != nil {
		m.context.Close()
		m.context = nil
	}
	if m.pw != nil {
		m.pw.Stop()
		m.pw = nil
	}
}

// GetOrCreatePage gets an existing page for a task or creates a new one
func (m *Manager) GetOrCreatePage(taskID string) (playwright.Page, error) {
	if err := m.EnsureStarted(); err != nil {
		return nil, err
	}

	m.pagesMux.Lock()
	defer m.pagesMux.Unlock()

	// Check if page exists
	if page, exists := m.pages[taskID]; exists {
		return page, nil
	}

	// Create new page
	var page playwright.Page
	var err error

	if m.browser != nil {
		// Regular browser interaction
		page, err = m.browser.NewPage(playwright.BrowserNewPageOptions{
			UserAgent: playwright.String(m.options.UserAgent),
			Viewport: &playwright.Size{
				Width:  m.options.ViewportWidth,
				Height: m.options.ViewportHeight,
			},
		})
	} else {
		// Persistent context interaction (already has viewport/UA set)
		// We use the first page or create a new one in the context
		// Note: LaunchPersistentContext already creates one page
		pages := m.browserContext().Pages()
		if len(pages) > 0 {
			page = pages[0]
		} else {
			page, err = m.browserContext().NewPage()
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	// Set default timeout
	page.SetDefaultTimeout(float64(m.options.DefaultTimeout.Milliseconds()))

	// Apply stealth scripts if enabled
	if m.options.Stealth {
		m.applyStealth(page)
	}

	m.pages[taskID] = page
	return page, nil
}

// applyStealth injects JavaScript to hide automation detection
func (m *Manager) applyStealth(page playwright.Page) {
	// Script to hide webdriver property and other automation indicators
	stealthScript := `
		// Overwrite the navigator.webdriver property
		Object.defineProperty(navigator, 'webdriver', {
			get: () => undefined
		});

		// Mock chrome runtime
		window.chrome = {
			runtime: {}
		};

		// Mock plugins
		Object.defineProperty(navigator, 'plugins', {
			get: () => [1, 2, 3, 4, 5]
		});

		// Mock languages
		Object.defineProperty(navigator, 'languages', {
			get: () => ['en-US', 'en']
		});

		// Pass the Permissions Test
		const originalQuery = window.navigator.permissions.query;
		window.navigator.permissions.query = (parameters) => (
			parameters.name === 'notifications' ?
				Promise.resolve({ state: Notification.permission }) :
				originalQuery(parameters)
		);

		// Hide automation in iframes
		Object.defineProperty(HTMLIFrameElement.prototype, 'contentWindow', {
			get: function() {
				return window;
			}
		});
	`

	page.AddInitScript(playwright.Script{Content: &stealthScript})
}

// browserContext returns the underlying browser context
func (m *Manager) browserContext() playwright.BrowserContext {
	if m.context != nil {
		return m.context
	}
	// For non-persistent, pages technically have individual contexts.
	// This implementation doesn't manage a shared context for regular mode.
	return nil
}

// Navigate navigates to a URL in a task-specific page
func (m *Manager) Navigate(taskID, url string) (*NavigationResult, error) {
	page, err := m.GetOrCreatePage(taskID)
	if err != nil {
		return nil, err
	}

	// Navigate
	response, err := page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	})
	if err != nil {
		return nil, fmt.Errorf("navigation failed: %w", err)
	}

	// Extract page info
	title, _ := page.Title()
	content, _ := page.Content()
	currentURL := page.URL()

	return &NavigationResult{
		URL:        currentURL,
		Title:      title,
		StatusCode: response.Status(),
		Content:    content,
	}, nil
}

// Click clicks an element by selector
func (m *Manager) Click(taskID, selector string) error {
	page, err := m.GetOrCreatePage(taskID)
	if err != nil {
		return err
	}

	return page.Click(selector)
}

// ClickCoordinates clicks at specific x,y coordinates
func (m *Manager) ClickCoordinates(taskID string, x, y float64) error {
	page, err := m.GetOrCreatePage(taskID)
	if err != nil {
		return err
	}

	return page.Mouse().Click(x, y)
}

// GetPageElements returns visible interactive elements with their info
func (m *Manager) GetPageElements(taskID string) ([]ElementInfo, error) {
	page, err := m.GetOrCreatePage(taskID)
	if err != nil {
		return nil, err
	}

	// JavaScript to get all interactive elements with their bounds
	script := `
		Array.from(document.querySelectorAll('a, button, input, select, textarea, [onclick], [role="button"]'))
			.filter(el => {
				const rect = el.getBoundingClientRect();
				const style = window.getComputedStyle(el);
				return rect.width > 0 && rect.height > 0 &&
				       style.visibility !== 'hidden' &&
				       style.display !== 'none' &&
				       rect.top < window.innerHeight &&
				       rect.bottom > 0;
			})
			.map((el, idx) => {
				const rect = el.getBoundingClientRect();
				return {
					index: idx,
					tag: el.tagName.toLowerCase(),
					text: el.innerText?.substring(0, 50) || '',
					id: el.id || '',
					class: el.className || '',
					x: Math.round(rect.left + rect.width / 2),
					y: Math.round(rect.top + rect.height / 2),
					width: Math.round(rect.width),
					height: Math.round(rect.height)
				};
			})
			.slice(0, 50);  // Limit to first 50 elements
	`

	result, err := page.Evaluate(script)
	if err != nil {
		return nil, err
	}

	// Convert to ElementInfo slice
	var elements []ElementInfo
	if arr, ok := result.([]interface{}); ok {
		for _, item := range arr {
			if elem, ok := item.(map[string]interface{}); ok {
				info := ElementInfo{
					Index:  toInt(elem["index"]),
					Tag:    elem["tag"].(string),
					Text:   elem["text"].(string),
					ID:     elem["id"].(string),
					Class:  elem["class"].(string),
					X:      toInt(elem["x"]),
					Y:      toInt(elem["y"]),
					Width:  toInt(elem["width"]),
					Height: toInt(elem["height"]),
				}
				elements = append(elements, info)
			}
		}
	}

	return elements, nil
}

// toInt safely converts interface{} to int
func toInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int:
		return val
	case int32:
		return int(val)
	case int64:
		return int(val)
	case float32:
		return int(val)
	case float64:
		return int(val)
	default:
		return 0
	}
}

// Type types text into an element
func (m *Manager) Type(taskID, selector, text string) error {
	page, err := m.GetOrCreatePage(taskID)
	if err != nil {
		return err
	}

	return page.Fill(selector, text)
}

// Screenshot takes a screenshot of the current page
func (m *Manager) Screenshot(taskID, path string) error {
	page, err := m.GetOrCreatePage(taskID)
	if err != nil {
		return err
	}

	_, err = page.Screenshot(playwright.PageScreenshotOptions{
		Path: playwright.String(path),
		Type: playwright.ScreenshotTypePng,
	})
	return err
}

// ScreenshotOptimized takes an AI-optimized screenshot (lower resolution, compressed)
func (m *Manager) ScreenshotOptimized(taskID, path string) error {
	page, err := m.GetOrCreatePage(taskID)
	if err != nil {
		return err
	}

	// Set viewport to lower resolution for AI processing
	if err := page.SetViewportSize(800, 600); err != nil {
		return err
	}

	// Take screenshot
	_, err = page.Screenshot(playwright.PageScreenshotOptions{
		Path:    playwright.String(path),
		Type:    playwright.ScreenshotTypeJpeg,
		Quality: playwright.Int(60), // Compressed quality
	})

	// Restore original viewport
	page.SetViewportSize(m.options.ViewportWidth, m.options.ViewportHeight)

	return err
}

// GetText extracts text from an element
func (m *Manager) GetText(taskID, selector string) (string, error) {
	page, err := m.GetOrCreatePage(taskID)
	if err != nil {
		return "", err
	}

	element, err := page.QuerySelector(selector)
	if err != nil || element == nil {
		return "", fmt.Errorf("element not found: %s", selector)
	}

	return element.TextContent()
}

// WaitForSelector waits for an element to appear
func (m *Manager) WaitForSelector(taskID, selector string, timeout time.Duration) error {
	page, err := m.GetOrCreatePage(taskID)
	if err != nil {
		return err
	}

	_, err = page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(float64(timeout.Milliseconds())),
	})
	return err
}

// ClosePage closes a specific task page
func (m *Manager) ClosePage(taskID string) error {
	m.pagesMux.Lock()
	defer m.pagesMux.Unlock()

	page, exists := m.pages[taskID]
	if !exists {
		return nil // Already closed
	}

	if err := page.Close(); err != nil {
		return err
	}

	delete(m.pages, taskID)
	return nil
}

// CloseAllPages closes all open pages
func (m *Manager) CloseAllPages() error {
	m.pagesMux.Lock()
	defer m.pagesMux.Unlock()

	for taskID, page := range m.pages {
		page.Close()
		delete(m.pages, taskID)
	}
	return nil
}

// Close shuts down the browser and playwright
func (m *Manager) Close() error {
	m.CloseAllPages()

	if m.browser != nil {
		m.browser.Close()
	}

	if m.context != nil {
		m.context.Close()
	}

	if m.pw != nil {
		return m.pw.Stop()
	}

	return nil
}

// GetActiveTasks returns list of active task IDs
func (m *Manager) GetActiveTasks() []string {
	m.pagesMux.RLock()
	defer m.pagesMux.RUnlock()

	tasks := make([]string, 0, len(m.pages))
	for taskID := range m.pages {
		tasks = append(tasks, taskID)
	}
	return tasks
}

// NavigationResult contains the result of a navigation
type NavigationResult struct {
	URL        string
	Title      string
	StatusCode int
	Content    string
}

// ElementInfo contains information about a page element
type ElementInfo struct {
	Index  int    `json:"index"`
	Tag    string `json:"tag"`
	Text   string `json:"text"`
	ID     string `json:"id"`
	Class  string `json:"class"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}
