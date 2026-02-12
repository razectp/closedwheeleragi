// Package browser provides web navigation and automation using playwright-go.
package browser

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
)

// actionTimeout is the per-operation timeout for individual browser actions.
const actionTimeout = 15 * time.Second

// Manager handles browser instances and contexts using playwright.
type Manager struct {
	opts    *Options
	pw      *playwright.Playwright
	browser playwright.Browser
	tabs    map[string]*tab
	tabsMu  sync.RWMutex
	initMu  sync.Mutex
	started bool
}

// tab wraps a playwright BrowserContext + Page for a single task.
type tab struct {
	context    playwright.BrowserContext
	page       playwright.Page
	url        string
	navigated  bool
	statusCode int
	opMu       sync.Mutex
}

// Options configures the browser manager.
type Options struct {
	Headless            bool
	PageTimeout         time.Duration
	ViewportWidth       int
	ViewportHeight      int
	CachePath           string
	ExecPath            string
	RemoteDebuggingPort int
}

// DefaultOptions returns sensible defaults.
func DefaultOptions() *Options {
	return &Options{
		Headless:       true,
		PageTimeout:    30 * time.Second,
		ViewportWidth:  1920,
		ViewportHeight: 1080,
	}
}

// NewManager creates a new browser manager.
func NewManager(opts *Options) (*Manager, error) {
	if opts == nil {
		opts = DefaultOptions()
	}
	// Note: the original code forced Headless = true here but used headless: false in launch.
	// We will follow the same pattern for consistency if needed, but playwright-go's headless
	// default is usually what users want.
	return &Manager{
		opts: opts,
		tabs: make(map[string]*tab),
	}, nil
}

// start launches Playwright and the browser.
func (m *Manager) start() error {
	pw, err := playwright.Run()
	if err != nil {
		// Driver not installed — attempt auto-install
		log.Printf("[Browser] Playwright driver not found, installing automatically...")
		if installErr := InstallDeps(); installErr != nil {
			return fmt.Errorf("could not start playwright and auto-install failed: %w (original: %v)", installErr, err)
		}
		// Retry after install
		pw, err = playwright.Run()
		if err != nil {
			return fmt.Errorf("could not start playwright after install: %w", err)
		}
	}
	m.pw = pw

	launchOptions := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(m.opts.Headless),
	}

	if m.opts.ExecPath != "" {
		launchOptions.ExecutablePath = playwright.String(m.opts.ExecPath)
	} else {
		execPath := findChromePath()
		if execPath != "" {
			launchOptions.ExecutablePath = playwright.String(execPath)
			log.Printf("[Browser] Using system browser at: %s", execPath)
		}
	}

	// If RemoteDebuggingPort is set, we can pass it as an arg.
	if m.opts.RemoteDebuggingPort > 0 {
		launchOptions.Args = append(launchOptions.Args, fmt.Sprintf("--remote-debugging-port=%d", m.opts.RemoteDebuggingPort))
	}

	bro, err := pw.Chromium.Launch(launchOptions)
	if err != nil {
		// Browser binary missing — attempt auto-install of Chromium
		log.Printf("[Browser] Chromium launch failed, installing browser automatically...")
		if installErr := InstallDeps(); installErr != nil {
			return fmt.Errorf("could not launch chromium and auto-install failed: %w (original: %v)", installErr, err)
		}
		// Clear custom executable path so Playwright uses its own installed Chromium
		launchOptions.ExecutablePath = nil
		bro, err = pw.Chromium.Launch(launchOptions)
		if err != nil {
			return fmt.Errorf("could not launch chromium after install: %w", err)
		}
	}
	m.browser = bro
	m.started = true
	log.Printf("[Browser] Playwright Manager initialized (headless=%v)", m.opts.Headless)
	return nil
}

// EnsureStarted initializes Playwright if not already running.
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
	return t, nil
}

// getOrCreateTab returns an existing live tab or creates a new isolated browser context.
func (m *Manager) getOrCreateTab(taskID string) (*tab, error) {
	if err := m.EnsureStarted(); err != nil {
		return nil, err
	}

	m.tabsMu.Lock()
	defer m.tabsMu.Unlock()

	if t, ok := m.tabs[taskID]; ok {
		return t, nil
	}

	contextOptions := playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{
			Width:  m.opts.ViewportWidth,
			Height: m.opts.ViewportHeight,
		},
	}

	ctx, err := m.browser.NewContext(contextOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser context: %w", err)
	}

	page, err := ctx.NewPage()
	if err != nil {
		ctx.Close()
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	t := &tab{
		context: ctx,
		page:    page,
	}

	// Capture status code for main document responses
	page.OnResponse(func(res playwright.Response) {
		if res.Request().Frame() == page.MainFrame() {
			t.statusCode = res.Status()
		}
	})

	m.tabs[taskID] = t
	return t, nil
}

// Navigate navigates to a URL and returns page info + readable text content.
func (m *Manager) Navigate(taskID, url string) (*NavigationResult, error) {
	t, err := m.getOrCreateTab(taskID)
	if err != nil {
		return nil, err
	}

	t.opMu.Lock()
	defer t.opMu.Unlock()

	resp, err := t.page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(float64(m.opts.PageTimeout.Milliseconds())),
	})
	if err != nil {
		return nil, fmt.Errorf("navigate %q: %w", url, err)
	}

	if resp != nil {
		t.statusCode = resp.Status()
	}
	t.url = t.page.URL()
	t.navigated = true

	// Extraction phase with retry for dynamic content
	var bodyText string
	for i := 0; i < 5; i++ {
		res, err := t.page.Evaluate(`() => {
            try {
                var c = document.body ? document.body.cloneNode(true) : document.documentElement.cloneNode(true);
                ['script','style','noscript','nav','footer','aside'].forEach(function(tag){
                    c.querySelectorAll(tag).forEach(function(el){el.remove();});
                });
                return (c.innerText || c.textContent || '').trim();
            } catch(e) { return 'ERR: '+e.message; }
        }`)
		if err == nil {
			bodyText = res.(string)
		}

		if len(bodyText) > 50 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	title, _ := t.page.Title()

	if len(bodyText) > 8000 {
		bodyText = bodyText[:8000] + "\n[... content truncated ...]"
	}

	return &NavigationResult{
		URL:        t.url,
		Title:      title,
		StatusCode: t.statusCode,
		Content:    bodyText,
	}, nil
}

// GetPageText returns the full visible text of the current page.
func (m *Manager) GetPageText(taskID string) (string, error) {
	t, err := m.requireNavigatedTab(taskID)
	if err != nil {
		return "", err
	}

	t.opMu.Lock()
	defer t.opMu.Unlock()

	res, err := t.page.Evaluate(`() => {
		var c = document.body.cloneNode(true);
		c.querySelectorAll('script,style,noscript').forEach(function(el){el.remove();});
		return (c.innerText || c.textContent || '').trim();
	}`)
	if err != nil {
		return "", fmt.Errorf("get_page_text: %w", err)
	}

	text := res.(string)
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

	t.opMu.Lock()
	defer t.opMu.Unlock()

	return t.page.Click(selector, playwright.PageClickOptions{
		Timeout: playwright.Float(float64(actionTimeout.Milliseconds())),
	})
}

// ClickCoordinates dispatches a mouse click at X,Y.
func (m *Manager) ClickCoordinates(taskID string, x, y float64) error {
	t, err := m.requireNavigatedTab(taskID)
	if err != nil {
		return err
	}

	t.opMu.Lock()
	defer t.opMu.Unlock()

	return t.page.Mouse().Click(x, y)
}

// Type fills a text input by selector.
func (m *Manager) Type(taskID, selector, text string) error {
	t, err := m.requireNavigatedTab(taskID)
	if err != nil {
		return err
	}

	t.opMu.Lock()
	defer t.opMu.Unlock()

	// Fill clears the input first, which matches the original behavior
	return t.page.Fill(selector, text, playwright.PageFillOptions{
		Timeout: playwright.Float(float64(actionTimeout.Milliseconds())),
	})
}

// GetText returns the inner text of the first element matching selector.
func (m *Manager) GetText(taskID, selector string) (string, error) {
	t, err := m.requireNavigatedTab(taskID)
	if err != nil {
		return "", err
	}

	t.opMu.Lock()
	defer t.opMu.Unlock()

	return t.page.Locator(selector).First().InnerText()
}

// EvaluateJS runs arbitrary JavaScript and JSON-encodes the result.
func (m *Manager) EvaluateJS(taskID, script string) (string, error) {
	t, err := m.requireNavigatedTab(taskID)
	if err != nil {
		return "", err
	}

	t.opMu.Lock()
	defer t.opMu.Unlock()

	res, err := t.page.Evaluate(script)
	if err != nil {
		return "", fmt.Errorf("eval_js: %w", err)
	}

	b, err := json.Marshal(res)
	if err != nil {
		return fmt.Sprintf("%v", res), nil
	}
	return string(b), nil
}

// GetPageElements returns visible interactive elements with position info.
func (m *Manager) GetPageElements(taskID string) ([]ElementInfo, error) {
	t, err := m.requireNavigatedTab(taskID)
	if err != nil {
		return nil, err
	}

	t.opMu.Lock()
	defer t.opMu.Unlock()

	script := `() => {
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
	}`

	raw, err := t.page.Evaluate(script)
	if err != nil {
		return nil, fmt.Errorf("get_elements: %w", err)
	}

	var elements []ElementInfo
	data, _ := json.Marshal(raw)
	_ = json.Unmarshal(data, &elements)

	return elements, nil
}

// Screenshot saves a full-page PNG to path.
func (m *Manager) Screenshot(taskID, path string) error {
	t, err := m.requireNavigatedTab(taskID)
	if err != nil {
		return err
	}

	t.opMu.Lock()
	defer t.opMu.Unlock()

	_, err = t.page.Screenshot(playwright.PageScreenshotOptions{
		Path:     playwright.String(path),
		FullPage: playwright.Bool(true),
	})
	return err
}

// ScreenshotOptimized saves a smaller viewport screenshot (suitable for LLM vision input).
func (m *Manager) ScreenshotOptimized(taskID, path string) error {
	t, err := m.requireNavigatedTab(taskID)
	if err != nil {
		return err
	}

	t.opMu.Lock()
	defer t.opMu.Unlock()

	// Store current viewport to restore it later
	viewport := t.page.ViewportSize()
	defer t.page.SetViewportSize(viewport.Width, viewport.Height)

	if err := t.page.SetViewportSize(800, 600); err != nil {
		return err
	}

	_, err = t.page.Screenshot(playwright.PageScreenshotOptions{
		Path: playwright.String(path),
	})
	return err
}

// WaitForSelector waits up to timeout for selector to become visible.
func (m *Manager) WaitForSelector(taskID, selector string, timeout time.Duration) error {
	t, err := m.requireNavigatedTab(taskID)
	if err != nil {
		return err
	}

	_, err = t.page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(float64(timeout.Milliseconds())),
	})
	return err
}

// ClosePage closes the browser tab/context for a task.
func (m *Manager) ClosePage(taskID string) error {
	m.tabsMu.Lock()
	defer m.tabsMu.Unlock()
	t, ok := m.tabs[taskID]
	if !ok {
		return nil
	}
	t.page.Close()
	t.context.Close()
	delete(m.tabs, taskID)
	return nil
}

// CloseAllPages closes all task tabs.
func (m *Manager) CloseAllPages() error {
	m.tabsMu.Lock()
	defer m.tabsMu.Unlock()
	for id, t := range m.tabs {
		t.page.Close()
		t.context.Close()
		delete(m.tabs, id)
	}
	return nil
}

// Close shuts down all tabs and the Playwright process.
func (m *Manager) Close() error {
	_ = m.CloseAllPages()

	if m.browser != nil {
		m.browser.Close()
	}
	if m.pw != nil {
		m.pw.Stop()
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

// findChromePath attempts to find a Chrome/Chromium/Edge executable on the current OS.
func findChromePath() string {
	var paths []string

	switch runtime.GOOS {
	case "windows":
		localAppData := os.Getenv("LocalAppData")
		programFiles := os.Getenv("ProgramFiles")
		programFilesX86 := os.Getenv("ProgramFiles(x86)")
		if programFiles == "" {
			programFiles = `C:\Program Files`
		}
		if programFilesX86 == "" {
			programFilesX86 = `C:\Program Files (x86)`
		}

		paths = []string{
			programFiles + `\Google\Chrome\Application\chrome.exe`,
			programFilesX86 + `\Google\Chrome\Application\chrome.exe`,
			localAppData + `\Google\Chrome\Application\chrome.exe`,
			programFiles + `\Chromium\Application\chrome.exe`,
			localAppData + `\Chromium\Application\chrome.exe`,
			programFilesX86 + `\Microsoft\Edge\Application\msedge.exe`,
			programFiles + `\Microsoft\Edge\Application\msedge.exe`,
		}

	case "darwin":
		paths = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
		}
		// Also check user-level Applications
		if home, err := os.UserHomeDir(); err == nil {
			paths = append(paths,
				home+"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
				home+"/Applications/Chromium.app/Contents/MacOS/Chromium",
				home+"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			)
		}

	case "linux":
		paths = []string{
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/usr/bin/microsoft-edge",
			"/usr/bin/microsoft-edge-stable",
			"/usr/bin/brave-browser",
			"/snap/bin/chromium",
			"/snap/bin/brave",
			"/usr/local/bin/chrome",
			"/usr/local/bin/chromium",
		}
	}

	// Check known paths first
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Fallback: search PATH using exec.LookPath
	candidates := []string{
		"google-chrome", "google-chrome-stable", "chrome",
		"chromium", "chromium-browser",
		"microsoft-edge", "microsoft-edge-stable", "msedge",
		"brave-browser",
	}
	for _, name := range candidates {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}

	return "" // Playwright will use its bundled browser
}
