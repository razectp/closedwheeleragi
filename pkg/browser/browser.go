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
	ctx       context.Context
	cancel    context.CancelFunc
	url       string
	navigated bool // true once Navigate succeeds at least once
}

// Options configures the browser manager.
type Options struct {
	Headless       bool
	DefaultTimeout time.Duration
	UserAgent      string
	ViewportWidth  int
	ViewportHeight int
	CachePath      string
	Stealth        bool
	SlowMo         int // kept for API compat; unused
}

// DefaultOptions returns sensible defaults.
func DefaultOptions() *Options {
	return &Options{
		Headless:       true,
		DefaultTimeout: 60 * time.Second,
		UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
		ViewportWidth:  1920,
		ViewportHeight: 1080,
		Stealth:        true,
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
func (m *Manager) start() error {
	allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", m.opts.Headless),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("disable-infobars", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("window-size", fmt.Sprintf("%d,%d", m.opts.ViewportWidth, m.opts.ViewportHeight)),
		chromedp.UserAgent(m.opts.UserAgent),
	)
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
// Returns a clear error if the tab doesn't exist or hasn't been navigated yet.
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
	// Detect dead context (cancelled due to timeout or prior error)
	if ctxErr := t.ctx.Err(); ctxErr != nil {
		// Auto-recover: remove the dead tab so the next browser_navigate creates a fresh one
		m.tabsMu.Lock()
		t.cancel()
		delete(m.tabs, taskID)
		m.tabsMu.Unlock()
		reason := ctxErr.Error()
		return nil, fmt.Errorf("browser tab %q context expired (%s) — call browser_navigate again with the same task_id to reopen", taskID, reason)
	}
	return t, nil
}

// getOrCreateTab returns an existing tab or creates a fresh isolated browser context.
// Used only by Navigate — other operations use requireNavigatedTab.
func (m *Manager) getOrCreateTab(taskID string) (*tab, error) {
	if err := m.EnsureStarted(); err != nil {
		return nil, fmt.Errorf("Chrome failed to start: %w\nEnsure Google Chrome or Chromium is installed on this machine", err)
	}

	m.tabsMu.Lock()
	defer m.tabsMu.Unlock()

	// Reuse live tab
	if t, ok := m.tabs[taskID]; ok {
		if t.ctx.Err() == nil {
			return t, nil
		}
		// Dead context — recreate
		t.cancel()
		delete(m.tabs, taskID)
	}

	ctx, cancel := chromedp.NewContext(m.allocCtx)
	t := &tab{ctx: ctx, cancel: cancel}
	m.tabs[taskID] = t
	return t, nil
}

// withTimeout runs fn with a per-call timeout derived from the tab context.
func (m *Manager) withTimeout(tabCtx context.Context, fn func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(tabCtx, m.opts.DefaultTimeout)
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

	var pageTitle, pageURL, bodyText string
	var statusCode int64

	chromedp.ListenTarget(t.ctx, func(ev interface{}) {
		if resp, ok := ev.(*network.EventResponseReceived); ok {
			if resp.Type == "Document" {
				statusCode = resp.Response.Status
			}
		}
	})

	err = m.withTimeout(t.ctx, func(ctx context.Context) error {
		return chromedp.Run(ctx,
			emulation.SetDeviceMetricsOverride(
				int64(m.opts.ViewportWidth), int64(m.opts.ViewportHeight), 1.0, false,
			),
			chromedp.ActionFunc(func(ctx context.Context) error {
				if m.opts.Stealth {
					return chromedp.Evaluate(stealthScript, nil).Do(ctx)
				}
				return nil
			}),
			chromedp.Navigate(url),
			chromedp.WaitReady("body", chromedp.ByQuery),
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
	if err != nil {
		if isTimeout(err) {
			// Partial success — try to salvage title/url
			_ = m.withTimeout(t.ctx, func(ctx context.Context) error {
				return chromedp.Run(ctx, chromedp.Title(&pageTitle), chromedp.Location(&pageURL))
			})
			t.navigated = true // partial navigation is still navigation
			t.url = pageURL
		} else {
			return nil, fmt.Errorf("navigate %q: %w", url, err)
		}
	} else {
		t.navigated = true
		t.url = pageURL
	}

	if len(bodyText) > 8000 {
		bodyText = bodyText[:8000] + "\n[... content truncated ...]"
	}

	return &NavigationResult{
		URL:        pageURL,
		Title:      pageTitle,
		StatusCode: int(statusCode),
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
	err = m.withTimeout(t.ctx, func(ctx context.Context) error {
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
	return m.withTimeout(t.ctx, func(ctx context.Context) error {
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
	return m.withTimeout(t.ctx, func(ctx context.Context) error {
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
	return m.withTimeout(t.ctx, func(ctx context.Context) error {
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
	err = m.withTimeout(t.ctx, func(ctx context.Context) error {
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
	err = m.withTimeout(t.ctx, func(ctx context.Context) error {
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
	err = m.withTimeout(t.ctx, func(ctx context.Context) error {
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
	err = m.withTimeout(t.ctx, func(ctx context.Context) error {
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
	err = m.withTimeout(t.ctx, func(ctx context.Context) error {
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
