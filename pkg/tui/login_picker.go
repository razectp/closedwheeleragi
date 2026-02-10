package tui

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"ClosedWheeler/pkg/llm"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Login flow steps
const (
	loginStepPickProvider = iota
	loginStepAnthropicPaste   // Anthropic: paste code#state
	loginStepOpenAIWaiting    // OpenAI: waiting for localhost callback
	loginStepOpenAIPaste      // OpenAI fallback: paste redirect URL
	loginStepGoogleWaiting    // Google: waiting for localhost callback
	loginStepGooglePaste      // Google fallback: paste redirect URL
)

// LoginProviderOption represents a provider available for OAuth login.
type LoginProviderOption struct {
	Label    string
	Provider string // "anthropic", "openai"
	Hint     string
}

// Available OAuth login providers
var loginProviders = []LoginProviderOption{
	{Label: "Anthropic", Provider: "anthropic", Hint: "Claude Pro / Max / Team"},
	{Label: "OpenAI", Provider: "openai", Hint: "ChatGPT Plus / Pro / Team"},
	{Label: "Google", Provider: "google", Hint: "Gemini Pro / Ultra (Cloud Code Assist)"},
	{Label: "Moonshot", Provider: "moonshot", Hint: "Kimi · Somente API key"},
	{Label: "DeepSeek", Provider: "deepseek", Hint: "Somente API key"},
}

// --- Tea messages for async OAuth operations ---

// openaiCallbackMsg is sent when the OpenAI localhost callback server receives a response.
type openaiCallbackMsg struct {
	code  string
	state string
	err   error
}

// oauthExchangeMsg is sent when an OAuth token exchange completes.
type oauthExchangeMsg struct {
	provider string
	err      error
}

// initLogin opens the login provider picker.
func (m *Model) initLogin() {
	m.loginActive = true
	m.loginStep = loginStepPickProvider
	m.loginCursor = 0
	m.loginProvider = ""
	m.loginVerifier = ""
	m.loginAuthURL = ""
}

// closeLogin exits login mode and resets state.
func (m *Model) closeLogin() {
	m.loginActive = false
	m.loginStep = loginStepPickProvider
	m.loginCursor = 0
	m.loginProvider = ""
	m.loginVerifier = ""
	m.loginAuthURL = ""
	m.loginClipboard = false
	// Clean up login URL file (contains PKCE verifier, no longer needed)
	_ = os.Remove(".agi/login-url.txt")
	if m.loginCancel != nil {
		m.loginCancel()
		m.loginCancel = nil
	}
}

// loginUpdate handles key events during the login flow.
func (m Model) loginUpdate(msg tea.KeyMsg) (Model, tea.Cmd) {
	if msg.Type == tea.KeyEsc {
		m.closeLogin()
		m.messages = append(m.messages, Message{
			Role:      "system",
			Content:   "OAuth login cancelled.",
			Timestamp: time.Now(),
		})
		m.updateViewport()
		return m, nil
	}

	switch m.loginStep {
	case loginStepPickProvider:
		return m.loginUpdatePickProvider(msg)
	case loginStepAnthropicPaste:
		return m.loginUpdateAnthropicPaste(msg)
	case loginStepOpenAIWaiting:
		// While waiting for callback, allow Esc (handled above) or 'p' to switch to paste mode
		if msg.String() == "p" || msg.String() == "P" {
			m.loginStep = loginStepOpenAIPaste
			ti := textinput.New()
			ti.Placeholder = "Paste the redirect URL from your browser..."
			ti.CharLimit = 1024
			ti.Width = 60
			ti.Focus()
			m.loginInput = ti
			return m, textinput.Blink
		}
		return m, nil
	case loginStepOpenAIPaste:
		return m.loginUpdateOpenAIPaste(msg)
	case loginStepGoogleWaiting:
		if msg.String() == "p" || msg.String() == "P" {
			m.loginStep = loginStepGooglePaste
			ti := textinput.New()
			ti.Placeholder = "Paste the redirect URL from your browser..."
			ti.CharLimit = 1024
			ti.Width = 60
			ti.Focus()
			m.loginInput = ti
			return m, textinput.Blink
		}
		return m, nil
	case loginStepGooglePaste:
		return m.loginUpdateGooglePaste(msg)
	}

	return m, nil
}

func (m Model) loginUpdatePickProvider(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.loginCursor > 0 {
			m.loginCursor--
		}
	case "down", "j":
		if m.loginCursor < len(loginProviders)-1 {
			m.loginCursor++
		}
	case "enter":
		selected := loginProviders[m.loginCursor]
		m.loginProvider = selected.Provider

		switch selected.Provider {
		case "anthropic":
			return m.startAnthropicLogin()
		case "openai":
			return m.startOpenAILogin()
		case "google":
			return m.startGoogleLogin()
		case "deepseek", "moonshot":
			m.closeLogin()
			m.messages = append(m.messages, Message{
				Role:      "system",
				Content:   fmt.Sprintf("%s uses API key only. Use /model to configure.", selected.Label),
				Timestamp: time.Now(),
			})
			m.updateViewport()
			return m, nil
		default:
			m.closeLogin()
			m.messages = append(m.messages, Message{
				Role:      "system",
				Content:   fmt.Sprintf("%s OAuth not yet implemented. Use /model to configure with API key.", selected.Label),
				Timestamp: time.Now(),
			})
			m.updateViewport()
			return m, nil
		}
	}
	return m, nil
}

// startAnthropicLogin initiates the Anthropic OAuth PKCE flow.
func (m Model) startAnthropicLogin() (Model, tea.Cmd) {
	verifier, challenge, err := llm.GeneratePKCE()
	if err != nil {
		m.closeLogin()
		m.messages = append(m.messages, Message{
			Role:      "error",
			Content:   fmt.Sprintf("Failed to generate PKCE: %v", err),
			Timestamp: time.Now(),
		})
		m.updateViewport()
		return m, nil
	}

	authURL := llm.BuildAuthURL(challenge, verifier)

	writeLoginURL(authURL)
	openBrowser(authURL)
	copied := copyToClipboard(authURL)

	m.loginStep = loginStepAnthropicPaste
	m.loginVerifier = verifier
	m.loginAuthURL = authURL
	m.loginClipboard = copied

	ti := textinput.New()
	ti.Placeholder = "Paste the code#state here..."
	ti.CharLimit = 512
	ti.Width = 60
	ti.Focus()
	m.loginInput = ti
	m.updateViewport()
	return m, textinput.Blink
}

func (m Model) loginUpdateAnthropicPaste(msg tea.KeyMsg) (Model, tea.Cmd) {
	if msg.Type == tea.KeyEnter {
		code := strings.TrimSpace(m.loginInput.Value())
		if code == "" {
			return m, nil
		}

		// Save verifier BEFORE closeLogin() clears it
		verifier := m.loginVerifier
		m.closeLogin()
		err := m.agent.LoginOAuth("anthropic", code, verifier)
		if err != nil {
			m.messages = append(m.messages, Message{
				Role:      "error",
				Content:   fmt.Sprintf("Anthropic OAuth login failed: %v", err),
				Timestamp: time.Now(),
			})
		} else {
			m.messages = append(m.messages, Message{
				Role:      "system",
				Content:   fmt.Sprintf("Anthropic OAuth login successful! Token %s.\nYou can now use Claude models.", m.agent.GetOAuthExpiry()),
				Timestamp: time.Now(),
			})
		}
		m.updateViewport()
		return m, nil
	}

	var cmd tea.Cmd
	m.loginInput, cmd = m.loginInput.Update(msg)
	return m, cmd
}

// startOpenAILogin initiates the OpenAI OAuth PKCE flow with localhost callback server.
func (m Model) startOpenAILogin() (Model, tea.Cmd) {
	verifier, challenge, err := llm.GeneratePKCE()
	if err != nil {
		m.closeLogin()
		m.messages = append(m.messages, Message{
			Role:      "error",
			Content:   fmt.Sprintf("Failed to generate PKCE: %v", err),
			Timestamp: time.Now(),
		})
		m.updateViewport()
		return m, nil
	}

	authURL := llm.BuildOpenAIAuthURL(challenge, verifier)
	m.loginStep = loginStepOpenAIWaiting
	m.loginVerifier = verifier
	m.loginAuthURL = authURL

	// Start callback server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	m.loginCancel = cancel

	resultCh, err := llm.StartOpenAICallbackServer(ctx)
	if err != nil {
		cancel()
		m.closeLogin()
		m.messages = append(m.messages, Message{
			Role:      "error",
			Content:   fmt.Sprintf("Failed to start callback server: %v\nTry closing other apps using port %d.", err, llm.OpenAIOAuthCallbackPort),
			Timestamp: time.Now(),
		})
		m.updateViewport()
		return m, nil
	}

	writeLoginURL(authURL)
	openBrowser(authURL)
	copied := copyToClipboard(authURL)
	m.loginClipboard = copied
	m.updateViewport()

	// Start async wait for callback
	agent := m.agent
	waitCmd := func() tea.Msg {
		select {
		case result := <-resultCh:
			if result.Err != nil {
				return oauthExchangeMsg{provider: "openai", err: result.Err}
			}
			// Exchange the code for tokens
			err := agent.LoginOAuth("openai", result.Code, verifier)
			return oauthExchangeMsg{provider: "openai", err: err}
		case <-ctx.Done():
			return oauthExchangeMsg{provider: "openai", err: fmt.Errorf("login timed out (5 minutes)")}
		}
	}

	return m, tea.Cmd(waitCmd)
}

func (m Model) loginUpdateOpenAIPaste(msg tea.KeyMsg) (Model, tea.Cmd) {
	if msg.Type == tea.KeyEnter {
		rawURL := strings.TrimSpace(m.loginInput.Value())
		if rawURL == "" {
			return m, nil
		}

		// Extract code from redirect URL: http://localhost:1455/auth/callback?code=X&state=Y
		code, err := extractCodeFromURL(rawURL)
		if err != nil {
			m.messages = append(m.messages, Message{
				Role:      "error",
				Content:   fmt.Sprintf("Invalid URL: %v", err),
				Timestamp: time.Now(),
			})
			m.updateViewport()
			return m, nil
		}

		// Save verifier BEFORE closeLogin() clears it
		verifier := m.loginVerifier
		m.closeLogin()
		err = m.agent.LoginOAuth("openai", code, verifier)
		if err != nil {
			m.messages = append(m.messages, Message{
				Role:      "error",
				Content:   fmt.Sprintf("OpenAI OAuth login failed: %v", err),
				Timestamp: time.Now(),
			})
		} else {
			m.messages = append(m.messages, Message{
				Role:      "system",
				Content:   fmt.Sprintf("OpenAI OAuth login successful! Token %s.\nYou can now use OpenAI models.", m.agent.GetOAuthExpiry()),
				Timestamp: time.Now(),
			})
		}
		m.updateViewport()
		return m, nil
	}

	var cmd tea.Cmd
	m.loginInput, cmd = m.loginInput.Update(msg)
	return m, cmd
}

// startGoogleLogin initiates the Google OAuth PKCE flow with localhost callback server.
func (m Model) startGoogleLogin() (Model, tea.Cmd) {
	verifier, challenge, err := llm.GeneratePKCE()
	if err != nil {
		m.closeLogin()
		m.messages = append(m.messages, Message{
			Role:      "error",
			Content:   fmt.Sprintf("Failed to generate PKCE: %v", err),
			Timestamp: time.Now(),
		})
		m.updateViewport()
		return m, nil
	}

	authURL := llm.BuildGoogleAuthURL(challenge, verifier)
	m.loginStep = loginStepGoogleWaiting
	m.loginVerifier = verifier
	m.loginAuthURL = authURL

	// Start callback server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	m.loginCancel = cancel

	resultCh, err := llm.StartGoogleCallbackServer(ctx)
	if err != nil {
		cancel()
		m.closeLogin()
		m.messages = append(m.messages, Message{
			Role:      "error",
			Content:   fmt.Sprintf("Failed to start callback server: %v\nTry closing other apps using port %d.", err, llm.GoogleOAuthCallbackPort),
			Timestamp: time.Now(),
		})
		m.updateViewport()
		return m, nil
	}

	writeLoginURL(authURL)
	openBrowser(authURL)
	copied := copyToClipboard(authURL)
	m.loginClipboard = copied
	m.updateViewport()

	// Start async wait for callback
	agent := m.agent
	waitCmd := func() tea.Msg {
		select {
		case result := <-resultCh:
			if result.Err != nil {
				return oauthExchangeMsg{provider: "google", err: result.Err}
			}
			err := agent.LoginOAuth("google", result.Code, verifier)
			return oauthExchangeMsg{provider: "google", err: err}
		case <-ctx.Done():
			return oauthExchangeMsg{provider: "google", err: fmt.Errorf("login timed out (5 minutes)")}
		}
	}

	return m, tea.Cmd(waitCmd)
}

func (m Model) loginUpdateGooglePaste(msg tea.KeyMsg) (Model, tea.Cmd) {
	if msg.Type == tea.KeyEnter {
		rawURL := strings.TrimSpace(m.loginInput.Value())
		if rawURL == "" {
			return m, nil
		}

		code, err := extractCodeFromURL(rawURL)
		if err != nil {
			m.messages = append(m.messages, Message{
				Role:      "error",
				Content:   fmt.Sprintf("Invalid URL: %v", err),
				Timestamp: time.Now(),
			})
			m.updateViewport()
			return m, nil
		}

		verifier := m.loginVerifier
		m.closeLogin()
		err = m.agent.LoginOAuth("google", code, verifier)
		if err != nil {
			m.messages = append(m.messages, Message{
				Role:      "error",
				Content:   fmt.Sprintf("Google OAuth login failed: %v", err),
				Timestamp: time.Now(),
			})
		} else {
			m.messages = append(m.messages, Message{
				Role:      "system",
				Content:   fmt.Sprintf("Google OAuth login successful! Token %s.\nYou can now use Gemini models.", m.agent.GetOAuthExpiry()),
				Timestamp: time.Now(),
			})
		}
		m.updateViewport()
		return m, nil
	}

	var cmd tea.Cmd
	m.loginInput, cmd = m.loginInput.Update(msg)
	return m, cmd
}

// extractCodeFromURL extracts the "code" query parameter from a redirect URL.
func extractCodeFromURL(rawURL string) (string, error) {
	// Handle both full URL and just query string
	if !strings.Contains(rawURL, "?") {
		return rawURL, nil // Assume it's just the code
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	code := parsed.Query().Get("code")
	if code == "" {
		return "", fmt.Errorf("no 'code' parameter found in URL")
	}
	return code, nil
}

// --- Login view ---

func (m Model) loginView() string {
	var s strings.Builder
	boxWidth := m.width - 6
	if boxWidth < 20 {
		boxWidth = 20
	}

	switch m.loginStep {
	case loginStepPickProvider:
		s.WriteString(m.loginViewPickProvider())
	case loginStepAnthropicPaste:
		s.WriteString(m.loginViewAnthropicPaste())
	case loginStepOpenAIWaiting:
		s.WriteString(m.loginViewOpenAIWaiting())
	case loginStepOpenAIPaste:
		s.WriteString(m.loginViewOpenAIPaste())
	case loginStepGoogleWaiting:
		s.WriteString(m.loginViewGoogleWaiting())
	case loginStepGooglePaste:
		s.WriteString(m.loginViewGooglePaste())
	}

	return pickerBoxStyle.Width(boxWidth).Render(s.String())
}

func (m Model) loginViewPickProvider() string {
	var s strings.Builder

	s.WriteString(pickerTitleStyle.Render("OAuth Login"))
	s.WriteString("\n")
	s.WriteString(pickerSubtitleStyle.Render("Select provider to authenticate:"))
	s.WriteString("\n\n")

	for i, p := range loginProviders {
		cursor := "  "
		style := pickerUnselectedStyle
		if m.loginCursor == i {
			cursor = "> "
			style = pickerSelectedStyle
		}

		label := p.Label

		// Show OAuth status
		expiry := m.agent.GetOAuthExpiryFor(p.Provider)
		if expiry != "" {
			label += " [" + expiry + "]"
		}

		line := style.Render(cursor+label) + "  " + pickerHintStyle.Render(p.Hint)
		s.WriteString(line)
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(pickerFooterStyle.Render("Up/Down Navigate  |  Enter Select  |  Esc Cancel"))

	return s.String()
}

func (m Model) loginViewAnthropicPaste() string {
	var s strings.Builder

	s.WriteString(pickerTitleStyle.Render("Anthropic OAuth Login"))
	s.WriteString("\n\n")
	s.WriteString(pickerSubtitleStyle.Render("1. Open the login URL in a browser:"))
	s.WriteString("\n")
	if m.loginClipboard {
		s.WriteString(pickerSelectedStyle.Render("   URL copied to clipboard! Paste in browser."))
	} else {
		s.WriteString(pickerHintStyle.Render("   Run in another terminal:  cat .agi/login-url.txt"))
	}
	s.WriteString("\n\n")
	s.WriteString(pickerSubtitleStyle.Render("2. Authorize and copy the code (format: code#state)"))
	s.WriteString("\n\n")
	s.WriteString(pickerSubtitleStyle.Render("3. Paste below:"))
	s.WriteString("\n\n")
	s.WriteString(m.loginInput.View())
	s.WriteString("\n\n")
	s.WriteString(pickerHintStyle.Render("  Enter Submit  |  Esc Cancel"))

	return s.String()
}

func (m Model) loginViewOpenAIWaiting() string {
	var s strings.Builder

	s.WriteString(pickerTitleStyle.Render("OpenAI OAuth Login"))
	s.WriteString("\n\n")
	s.WriteString(pickerSubtitleStyle.Render("Waiting for authorization..."))
	s.WriteString("\n\n")
	if m.loginClipboard {
		s.WriteString(pickerSelectedStyle.Render("  URL copied to clipboard! Paste in browser."))
	} else {
		s.WriteString(pickerHintStyle.Render("  Run in another terminal:  cat .agi/login-url.txt"))
	}
	s.WriteString("\n\n")
	s.WriteString(pickerHintStyle.Render("  After authorizing, the login will complete automatically."))
	s.WriteString("\n\n")
	s.WriteString(pickerHintStyle.Render("  VPS? SSH tunnel: ssh -L 1455:localhost:1455 user@server"))
	s.WriteString("\n\n")
	s.WriteString(pickerFooterStyle.Render("P = Paste URL manually  |  Esc Cancel"))

	return s.String()
}

func (m Model) loginViewOpenAIPaste() string {
	var s strings.Builder

	s.WriteString(pickerTitleStyle.Render("OpenAI OAuth Login (Manual)"))
	s.WriteString("\n\n")
	s.WriteString(pickerSubtitleStyle.Render("Paste the redirect URL from your browser's address bar:"))
	s.WriteString("\n")
	s.WriteString(pickerHintStyle.Render("  (The page that says 'This site can't be reached')"))
	s.WriteString("\n\n")
	s.WriteString(m.loginInput.View())
	s.WriteString("\n\n")
	s.WriteString(pickerHintStyle.Render("  Enter Submit  |  Esc Cancel"))

	return s.String()
}

func (m Model) loginViewGoogleWaiting() string {
	var s strings.Builder

	s.WriteString(pickerTitleStyle.Render("Google OAuth Login"))
	s.WriteString("\n\n")
	s.WriteString(pickerSubtitleStyle.Render("Waiting for authorization..."))
	s.WriteString("\n\n")
	if m.loginClipboard {
		s.WriteString(pickerSelectedStyle.Render("  URL copied to clipboard! Paste in browser."))
	} else {
		s.WriteString(pickerHintStyle.Render("  Run in another terminal:  cat .agi/login-url.txt"))
	}
	s.WriteString("\n\n")
	s.WriteString(pickerHintStyle.Render("  After authorizing, the login will complete automatically."))
	s.WriteString("\n\n")
	s.WriteString(pickerHintStyle.Render("  VPS? SSH tunnel: ssh -L 8085:localhost:8085 user@server"))
	s.WriteString("\n\n")
	s.WriteString(pickerFooterStyle.Render("P = Paste URL manually  |  Esc Cancel"))

	return s.String()
}

func (m Model) loginViewGooglePaste() string {
	var s strings.Builder

	s.WriteString(pickerTitleStyle.Render("Google OAuth Login (Manual)"))
	s.WriteString("\n\n")
	s.WriteString(pickerSubtitleStyle.Render("Paste the redirect URL from your browser's address bar:"))
	s.WriteString("\n")
	s.WriteString(pickerHintStyle.Render("  (The page that says 'This site can't be reached')"))
	s.WriteString("\n\n")
	s.WriteString(m.loginInput.View())
	s.WriteString("\n\n")
	s.WriteString(pickerHintStyle.Render("  Enter Submit  |  Esc Cancel"))

	return s.String()
}

// writeLoginURL saves the auth URL to a file and copies to clipboard via OSC 52.
func writeLoginURL(authURL string) {
	_ = os.WriteFile(".agi/login-url.txt", []byte(authURL+"\n"), 0600)
}

// copyToClipboard copies text to the system clipboard using OSC 52 escape sequence.
// Works over SSH in modern terminals (iTerm2, Windows Terminal, kitty, alacritty, etc).
// Writes directly to /dev/tty to bypass bubbletea's rendering.
func copyToClipboard(text string) bool {
	f, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		return false
	}
	defer f.Close()
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	_, err = fmt.Fprintf(f, "\033]52;c;%s\a", encoded)
	return err == nil
}
