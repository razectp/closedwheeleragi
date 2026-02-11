// Package browser provides web navigation and automation using chromedp (Chrome DevTools Protocol).
// It connects to the system Chrome/Chromium directly — no Node.js, no external binaries needed.
package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// actionTimeout is the per-operation timeout for individual browser actions
// (evaluate, click, type, screenshot, etc.).
// These should be near-instant on a loaded page — 15s is generous.
const actionTimeout = 15 * time.Second

// Manager handles browser instances and tabs using chromedp.
// Each task gets its own isolated browser context (separate cookies/sessions).
type Manager struct {
	opts     *Options
	allocCtx context.Context    // chromedp allocator context (one shared Chrome process)
	allocCnl context.CancelFunc // cancels the allocator / closes Chrome
	tabs     map[string]*tab    // taskID → tab
	tabsMu   sync.RWMutex
	initMu   sync.Mutex
	started  bool
}

// tab wraps a chromedp browser context + cancel for a single task.
type tab struct {
	ctx        context.Context
	cancel     context.CancelFunc
	url        string
	navigated  bool  // true once Navigate succeeds at least once
	statusCode int64 // last HTTP response status for Document requests
}

// Options configures the browser manager.
type Options struct {
	Headless      bool
	PageTimeout   time.Duration // timeout for page navigation (default 30s)
	UserAgent     string
	ViewportWidth  int
	ViewportHeight int
	CachePath     string
	Stealth       bool
}

// DefaultOptions returns sensible defaults.
func DefaultOptions() *Options {
	return &Options{
		Headless:      true,
		PageTimeout:   30 * time.Second,
		UserAgent:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
		ViewportWidth:  1920,
		ViewportHeight: 1080,
		Stealth:       true,
	}
}

// NewManager creates a new browser manager (lazy — Chrome not launched until first use).
func NewManager(opts *Options) (*Manager, error) {
	if opts == nil {
		opts = DefaultOptions()
	}
	return &Manager{
		opts: opts,
		tabs: make(map[string]*tab),
	}, nil
}

// start launches Chrome via chromedp allocator.
//
// Strategy: start from chromedp.DefaultExecAllocatorOptions (battle-tested baseline)
// and append overrides at the end — later entries win for duplicate flag keys.
//
// Headless mode:  override with Flag("headless","new") — required for Chrome 112+.
//                 The old Headless option (headless=true) was removed in Chrome 132+.
// Visible mode:   override with Flag("headless",false) — chromedp omits flags whose
//                 value is false, effectively removing --headless from the command line.
func (m *Manager) start() error {
	allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		// Required for CDP automation on Linux; safe no-op on Windows/Mac
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		// Hide the automation banner ("Chrome is being controlled by automated software")
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		// Viewport size
		chromedp.Flag("window-size", fmt.Sprintf("%d,%d", m.opts.ViewportWidth, m.opts.ViewportHeight)),
		// Custom User-Agent
		chromedp.UserAgent(m.opts.UserAgent),
	)

	if m.opts.Headless {
		// Override the DefaultExecAllocatorOptions "Headless" (= headless=true, old mode)
		// with headless=new, which works correctly in Chrome 112+ and is required for 132+.
		// Also add the companion flags that the old chromedp.Headless() added.
		allocOpts = append(allocOpts,
			chromedp.Flag("headless", "new"),
			chromedp.Flag("hide-scrollbars", true),
			chromedp.Flag("mute-audio", true),
		)
	} else {
		// Visible mode: remove the headless flag entirely.
		// chromedp.Flag(key, false) means "don't include this flag on the command line".
		allocOpts = append(allocOpts,
			chromedp.Flag("headless", false),
			// Ensure the window appears maximized and focused on screen.
			chromedp.Flag("start-maximized", true),
		)
	}

	if m.opts.CachePath != "" {
		allocOpts = append(allocOpts, chromedp.UserDataDir(m.opts.CachePath))
	}

	allocCtx, allocCnl := chromedp.NewExecAllocator(context.Background(), allocOpts...)
	m.allocCtx = allocCtx
	m.allocCnl = allocCnl
	m.started = true
	return nil
}

// EnsureStarted initialises Chrome if not already running.
func (m *Manager) EnsureStarted() error {
	m.initMu.Lock()
	defer m.initMu.Unlock()
	if m.started {
		return nil
	}
	return m.start()
}

// requireNavigatedTab returns a tab for taskID, enforcing that it has been navigated.
func (m *Manager) requireNavigatedTab(taskID string) (*tab, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	m.tabsMu.RLock()
	t, ok := m.tabs[taskID]
	m.tabsMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no browser tab open for task_id=%q — call browser_navigate first", taskID)
	}
	if !t.navigated {
		return nil, fmt.Errorf("tab %q exists but browser_navigate has not completed — call browser_navigate first", taskID)
	}
	if ctxErr := t.ctx.Err(); ctxErr != nil {
		m.tabsMu.Lock()
		t.cancel()
		delete(m.tabs, taskID)
		m.tabsMu.Unlock()
		return nil, fmt.Errorf("browser tab %q context expired (%s) — call browser_navigate again", taskID, ctxErr.Error())
	}
	return t, nil
}

// getOrCreateTab returns an existing live tab or creates a new isolated browser context.
// The status-code listener is registered once here so repeated Navigate calls don't
// accumulate duplicate listeners.
func (m *Manager) getOrCreateTab(taskID string) (*tab, error) {
	if err := m.EnsureStarted(); err != nil {
		return nil, fmt.Errorf("Chrome failed to start: %w\nEnsure Google Chrome or Chromium is installed", err)
	}

	m.tabsMu.Lock()
	defer m.tabsMu.Unlock()

	// Reuse a live tab
	if t, ok := m.tabs[taskID]; ok {
		if t.ctx.Err() == nil {
			return t, nil
		}
		// Dead context — tear down and recreate
		t.cancel()
		delete(m.tabs, taskID)
	}

	ctx, cancel := chromedp.NewContext(m.allocCtx)
	t := &tab{ctx: ctx, cancel: cancel}

	// Register the HTTP status listener once per tab so Navigate can be called
	// multiple times without accumulating duplicate listeners.
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if resp, ok := ev.(*network.EventResponseReceived); ok {
			if resp.Type == "Document" {
				t.statusCode = resp.Response.Status
			}
		}
	})

	m.tabs[taskID] = t
	return t, nil
}

// runWithTimeout executes fn inside a context with the given timeout derived from tabCtx.
// The per-call timeout does NOT affect the parent tab context — the tab stays alive.
func (m *Manager) runWithTimeout(tabCtx context.Context, timeout time.Duration, fn func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(tabCtx, timeout)
	defer cancel()
	return fn(ctx)
}

// stealthScript removes common automation fingerprints.
const stealthScript = `(function(){
	Object.defineProperty(navigator,'webdriver',{get:()=>undefined});
	window.chrome={runtime:{}};
	Object.defineProperty(navigator,'plugins',{get:()=>[1,2,3]});
	Object.defineProperty(navigator,'languages',{get:()=>['en-US','en']});
})();`

// ──────────────────────────────────────────────────────────
// Core browser operations
// ──────────────────────────────────────────────────────────

// Navigate navigates to a URL and returns page info + readable text content.
// This is the entry point for all browsing — always call this first.
func (m *Manager) Navigate(taskID, url string) (*NavigationResult, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	if url == "" {
		return nil, fmt.Errorf("url is required")
	}

	t, err := m.getOrCreateTab(taskID)
	if err != nil {
		return nil, err
	}

	// Phase 1 — page load: set viewport and navigate.
	// Uses PageTimeout (default 30s) because the network round-trip can be slow.
	// Stealth injection is intentionally done AFTER navigation so it runs in the
	// actual page context, not on about:blank.
	var pageTitle, pageURL, bodyText string

	err = m.runWithTimeout(t.ctx, m.opts.PageTimeout, func(ctx context.Context) error {
		return chromedp.Run(ctx,
			emulation.SetDeviceMetricsOverride(
				int64(m.opts.ViewportWidth), int64(m.opts.ViewportHeight), 1.0, false,
			),
			chromedp.Navigate(url),
			// WaitVisible only checks that <body> exists in the DOM.
			// WaitReady waits for document.readyState=complete which hangs on SPAs.
			chromedp.WaitVisible("body", chromedp.ByQuery),
		)
	})
	if err != nil && !isTimeout(err) {
		return nil, fmt.Errorf("navigate %q: %w", url, err)
	}
	// Timeout on page load is treated as partial success — the page may still be
	// useful (e.g. enough content loaded). Mark navigated so other tools can proceed.
	t.navigated = true
	t.url = url

	// Phase 2 — post-load actions: stealth injection + content extraction.
	// Uses actionTimeout (15s) — DOM access should be instant on a loaded page.
	_ = m.runWithTimeout(t.ctx, actionTimeout, func(ctx context.Context) error {
		return chromedp.Run(ctx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				if m.opts.Stealth {
					return chromedp.Evaluate(stealthScript, nil).Do(ctx)
				}
				return nil
			}),
			chromedp.Title(&pageTitle),
			chromedp.Location(&pageURL),
			chromedp.Evaluate(`(function(){
				var c=document.body.cloneNode(true);
				['script','style','noscript','nav','footer','aside'].forEach(function(tag){
					c.querySelectorAll(tag).forEach(function(el){el.remove();});
				});
				return (c.innerText||c.textContent||'').trim();
			})()`, &bodyText),
		)
	})

	if pageURL != "" {
		t.url = pageURL
	}

	if len(bodyText) > 8000 {
		bodyText = bodyText[:8000] + "\n[... content truncated ...]"
	}

	return &NavigationResult{
		URL:        t.url,
		Title:      pageTitle,
		StatusCode: int(t.statusCode),
		Content:    bodyText,
	}, nil
}

// GetPageText returns the full visible text of the current page.
func (m *Manager) GetPageText(taskID string) (string, error) {
	t, err := m.requireNavigatedTab(taskID)
	if err != nil {
		return "", err
	}
	var text string
	err = m.runWithTimeout(t.ctx, actionTimeout, func(ctx context.Context) error {
		return chromedp.Run(ctx, chromedp.Evaluate(`(function(){
			var c=document.body.cloneNode(true);
			c.querySelectorAll('script,style,noscript').forEach(function(el){el.remove();});
			return (c.innerText||c.textContent||'').trim();
		})()`, &text))
	})
	if err != nil {
		return "", fmt.Errorf("get_page_text: %w", err)
	}
	if len(text) > 10000 {
		text = text[:10000] + "\n[... content truncated ...]"
	}
	return text, nil
}

// Click clicks the first element matching selector.
func (m *Manager) Click(taskID, selector string) error {
	t, err := m.requireNavigatedTab(taskID)
	if err != nil {
		return err
	}
	if selector == "" {
		return fmt.Errorf("selector is required")
	}
	return m.runWithTimeout(t.ctx, actionTimeout, func(ctx context.Context) error {
		return chromedp.Run(ctx,
			chromedp.WaitVisible(selector, chromedp.ByQuery),
			chromedp.Click(selector, chromedp.ByQuery),
		)
	})
}

// ClickCoordinates dispatches a mouse click at X,Y.
func (m *Manager) ClickCoordinates(taskID string, x, y float64) error {
	t, err := m.requireNavigatedTab(taskID)
	if err != nil {
		return err
	}
	return m.runWithTimeout(t.ctx, actionTimeout, func(ctx context.Context) error {
		return chromedp.Run(ctx, chromedp.MouseClickXY(x, y))
	})
}

// Type fills a text input by selector.
func (m *Manager) Type(taskID, selector, text string) error {
	t, err := m.requireNavigatedTab(taskID)
	if err != nil {
		return err
	}
	if selector == "" {
		return fmt.Errorf("selector is required")
	}
	return m.runWithTimeout(t.ctx, actionTimeout, func(ctx context.Context) error {
		return chromedp.Run(ctx,
			chromedp.WaitVisible(selector, chromedp.ByQuery),
			chromedp.Clear(selector, chromedp.ByQuery),
			chromedp.SendKeys(selector, text, chromedp.ByQuery),
		)
	})
}

// GetText returns the inner text of the first element matching selector.
func (m *Manager) GetText(taskID, selector string) (string, error) {
	t, err := m.requireNavigatedTab(taskID)
	if err != nil {
		return "", err
	}
	if selector == "" {
		return "", fmt.Errorf("selector is required")
	}
	var text string
	err = m.runWithTimeout(t.ctx, actionTimeout, func(ctx context.Context) error {
		return chromedp.Run(ctx,
			chromedp.WaitVisible(selector, chromedp.ByQuery),
			chromedp.Text(selector, &text, chromedp.ByQuery),
		)
	})
	if err != nil {
		return "", fmt.Errorf("get_text(%q): %w", selector, err)
	}
	return text, nil
}

// EvaluateJS runs arbitrary JavaScript and JSON-encodes the result.
func (m *Manager) EvaluateJS(taskID, script string) (string, error) {
	t, err := m.requireNavigatedTab(taskID)
	if err != nil {
		return "", err
	}
	if script == "" {
		return "", fmt.Errorf("script is required")
	}
	var result interface{}
	err = m.runWithTimeout(t.ctx, actionTimeout, func(ctx context.Context) error {
		return chromedp.Run(ctx, chromedp.Evaluate(script, &result))
	})
	if err != nil {
		return "", fmt.Errorf("eval_js: %w", err)
	}
	b, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprintf("%v", result), nil
	}
	return string(b), nil
}

// GetPageElements returns visible interactive elements with position info.
func (m *Manager) GetPageElements(taskID string) ([]ElementInfo, error) {
	t, err := m.requireNavigatedTab(taskID)
	if err != nil {
		return nil, err
	}

	script := `(function(){
		return Array.from(document.querySelectorAll(
			'a,button,input,select,textarea,[onclick],[role="button"],[role="link"]'
		)).filter(function(el){
			var r=el.getBoundingClientRect(),s=window.getComputedStyle(el);
			return r.width>0&&r.height>0&&s.visibility!=='hidden'&&s.display!=='none'
			       &&r.top<window.innerHeight&&r.bottom>0;
		}).slice(0,60).map(function(el,idx){
			var r=el.getBoundingClientRect();
			var label=(el.getAttribute('aria-label')||el.getAttribute('placeholder')||
			           el.getAttribute('title')||el.innerText||'').substring(0,60).trim();
			return {index:idx,tag:el.tagName.toLowerCase(),text:label,
			        id:el.id||'',class:(el.className||'').substring(0,40),
			        x:Math.round(r.left+r.width/2),y:Math.round(r.top+r.height/2),
			        width:Math.round(r.width),height:Math.round(r.height)};
		});
	})()`

	var raw interface{}
	err = m.runWithTimeout(t.ctx, actionTimeout, func(ctx context.Context) error {
		return chromedp.Run(ctx, chromedp.Evaluate(script, &raw))
	})
	if err != nil {
		return nil, fmt.Errorf("get_elements: %w", err)
	}

	var elements []ElementInfo
	if arr, ok := raw.([]interface{}); ok {
		for _, item := range arr {
			if mp, ok := item.(map[string]interface{}); ok {
				elements = append(elements, ElementInfo{
					Index:  toInt(mp["index"]),
					Tag:    toString(mp["tag"]),
					Text:   toString(mp["text"]),
					ID:     toString(mp["id"]),
					Class:  toString(mp["class"]),
					X:      toInt(mp["x"]),
					Y:      toInt(mp["y"]),
					Width:  toInt(mp["width"]),
					Height: toInt(mp["height"]),
				})
			}
		}
	}
	return elements, nil
}

// Screenshot saves a full-page PNG to path.
func (m *Manager) Screenshot(taskID, path string) error {
	t, err := m.requireNavigatedTab(taskID)
	if err != nil {
		return err
	}
	if path == "" {
		return fmt.Errorf("path is required")
	}
	var buf []byte
	err = m.runWithTimeout(t.ctx, actionTimeout, func(ctx context.Context) error {
		return chromedp.Run(ctx, chromedp.FullScreenshot(&buf, 90))
	})
	if err != nil {
		return fmt.Errorf("screenshot: %w", err)
	}
	if len(buf) == 0 {
		return fmt.Errorf("screenshot produced empty image — page may not have loaded properly")
	}
	return os.WriteFile(path, buf, 0644)
}

// ScreenshotOptimized saves a smaller viewport screenshot (suitable for LLM vision input).
func (m *Manager) ScreenshotOptimized(taskID, path string) error {
	t, err := m.requireNavigatedTab(taskID)
	if err != nil {
		return err
	}
	if path == "" {
		return fmt.Errorf("path is required")
	}
	var buf []byte
	err = m.runWithTimeout(t.ctx, actionTimeout, func(ctx context.Context) error {
		return chromedp.Run(ctx,
			emulation.SetDeviceMetricsOverride(800, 600, 1.0, false),
			chromedp.CaptureScreenshot(&buf),
			emulation.SetDeviceMetricsOverride(
				int64(m.opts.ViewportWidth), int64(m.opts.ViewportHeight), 1.0, false,
			),
		)
	})
	if err != nil {
		return fmt.Errorf("screenshot_optimized: %w", err)
	}
	if len(buf) == 0 {
		return fmt.Errorf("screenshot produced empty image — page may not have loaded properly")
	}
	return os.WriteFile(path, buf, 0644)
}

// WaitForSelector waits up to timeout for selector to become visible.
func (m *Manager) WaitForSelector(taskID, selector string, timeout time.Duration) error {
	t, err := m.requireNavigatedTab(taskID)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(t.ctx, timeout)
	defer cancel()
	return chromedp.Run(ctx, chromedp.WaitVisible(selector, chromedp.ByQuery))
}

// ──────────────────────────────────────────────────────────
// Lifecycle
// ──────────────────────────────────────────────────────────

// ClosePage closes the browser context for a task.
func (m *Manager) ClosePage(taskID string) error {
	m.tabsMu.Lock()
	defer m.tabsMu.Unlock()
	t, ok := m.tabs[taskID]
	if !ok {
		return nil
	}
	t.cancel()
	delete(m.tabs, taskID)
	return nil
}

// CloseAllPages closes all task tabs.
func (m *Manager) CloseAllPages() error {
	m.tabsMu.Lock()
	defer m.tabsMu.Unlock()
	for id, t := range m.tabs {
		t.cancel()
		delete(m.tabs, id)
	}
	return nil
}

// Close shuts down all tabs and the Chrome process.
func (m *Manager) Close() error {
	_ = m.CloseAllPages()
	if m.allocCnl != nil {
		m.allocCnl()
	}
	return nil
}

// GetActiveTasks returns all task IDs with open tabs.
func (m *Manager) GetActiveTasks() []string {
	m.tabsMu.RLock()
	defer m.tabsMu.RUnlock()
	ids := make([]string, 0, len(m.tabs))
	for id := range m.tabs {
		ids = append(ids, id)
	}
	return ids
}

// ──────────────────────────────────────────────────────────
// Types
// ──────────────────────────────────────────────────────────

// NavigationResult contains the result of a page navigation.
type NavigationResult struct {
	URL        string
	Title      string
	StatusCode int
	Content    string
}

// ElementInfo describes an interactive element on the page.
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

// ──────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────

func isTimeout(err error) bool {
	s := err.Error()
	return strings.Contains(s, "deadline") || strings.Contains(s, "timeout")
}

func toInt(v interface{}) int {
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
	}
	return 0
}

func toString(v interface{}) string {
	s, _ := v.(string)
	return s
}
