package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"ClosedWheeler/pkg/agent"
	"ClosedWheeler/pkg/config"
	"ClosedWheeler/pkg/llm"
	"ClosedWheeler/pkg/tui"

	"github.com/charmbracelet/lipgloss"
)

const version = "0.1.0"

var (
	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")).
			Bold(true)
)

func main() {
	// Flags
	configPath := flag.String("config", "", "Path to configuration file")
	projectPath := flag.String("project", ".", "Path to project to analyze")
	showVersion := flag.Bool("version", false, "Show version")
	showHelp := flag.Bool("help", false, "Show help")
	doLogin := flag.Bool("login", false, "Authenticate with Anthropic OAuth (no TUI)")
	flag.Parse()

	if *showHelp {
		printHelp()
		return
	}

	if *showVersion {
		fmt.Printf("Coder AGI v%s\n", version)
		return
	}

	if *doLogin {
		runOAuthLogin()
		return
	}

	// Load configuration
	cfg, _, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	// Check API key — also allow OAuth credentials as alternative
	oauthCreds, _ := config.LoadOAuth()

	// Auto-refresh OAuth if expired
	if oauthCreds != nil && oauthCreds.NeedsRefresh() && oauthCreds.RefreshToken != "" {
		fmt.Println("🔄 Refreshing OAuth token...")
		newCreds, err := llm.RefreshOAuthToken(oauthCreds.RefreshToken)
		if err != nil {
			fmt.Printf("⚠️  OAuth token refresh failed: %v\n", err)
			fmt.Println("   Use /login to re-authenticate.")
		} else {
			oauthCreds = newCreds
			if err := config.SaveOAuth(newCreds); err != nil {
				fmt.Printf("⚠️  Failed to persist refreshed token: %v\n", err)
			}
			fmt.Println("✅ OAuth token refreshed.")
		}
	}

	if cfg.APIKey == "" && oauthCreds == nil {
		fmt.Println("⚡ Welcome to ClosedWheelerAGI!")
		fmt.Println("   First time setup detected.")
		fmt.Println()

		// Get application root before setup
		appRoot, err := os.Getwd()
		if err != nil {
			appRoot = "."
		}

		// Run interactive setup (no wizard)
		if err := tui.InteractiveSetup(appRoot); err != nil {
			log.Fatalf("❌ Setup failed: %v", err)
		}

		// Reload config after setup
		cfg, _, err = config.Load(*configPath)
		if err != nil {
			log.Fatalf("❌ Failed to reload config: %v", err)
		}

		// Re-verify after setup
		oauthCreds, _ = config.LoadOAuth()
		if cfg.APIKey == "" && oauthCreds == nil {
			fmt.Println("❌ Configuration incomplete. Exiting.")
			os.Exit(1)
		}
		fmt.Println(successStyle.Render("✅ Configuration complete! Starting agent..."))
		fmt.Println()
	}

	// Resolve project path
	absProjectPath, err := filepath.Abs(*projectPath)
	if err != nil {
		log.Fatalf("❌ Invalid project path: %v", err)
	}

	// Verify project exists
	if _, err := os.Stat(absProjectPath); os.IsNotExist(err) {
		log.Fatalf("❌ Project path does not exist: %s", absProjectPath)
	}

	// Print startup banner
	printBanner()
	fmt.Printf("📂 Project: %s\n", absProjectPath)
	fmt.Printf("🔧 Model: %s\n", cfg.Model)
	fmt.Println()

	// Get application root (current working directory)
	appRoot, err := os.Getwd()
	if err != nil {
		log.Printf("⚠️  Failed to get current directory: %v", err)
		appRoot = "."
	}

	// Create agent
	ag, err := agent.NewAgent(cfg, absProjectPath, appRoot)
	if err != nil {
		log.Fatalf("❌ Failed to create agent: %v", err)
	}

	// Context for graceful shutdown — cancelling it forces bubbletea to exit
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Signal handler: first SIGINT/SIGTERM cancels context, second force-exits
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
		select {
		case <-sigCh:
			os.Exit(1)
		case <-time.After(5 * time.Second):
			os.Exit(1)
		}
	}()

	// Start Telegram Bridge
	ag.StartTelegram()

	// Start Heartbeat
	ag.StartHeartbeat()

	// Run TUI (passes context so cancel() forces exit even if bubbletea hangs)
	if err := tui.Run(ag, ctx); err != nil {
		// Ignore context-cancelled errors — that's just our shutdown path
		if ctx.Err() == nil {
			log.Fatalf("❌ TUI error: %v", err)
		}
	}

	// Reset terminal to sane state (in case bubbletea didn't restore properly)
	fmt.Print("\033[?1000l\033[?1002l\033[?1003l\033[?1006l") // disable mouse modes
	fmt.Print("\033[?25h")                                      // show cursor
	fmt.Print("\033[?1049l")                                    // exit alt screen

	// Shutdown with timeout to guarantee exit
	done := make(chan struct{})
	go func() {
		ag.Shutdown()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		log.Println("Shutdown timed out, forcing exit")
	}

	fmt.Println("\n👋 Goodbye!")
	os.Exit(0)
}

func printBanner() {
	banner := `
  ╔═══════════════════════════════════════════════════════════════╗
  ║                                                               ║
  ║          ClosedWheelerAGI - Intelligent Coding Agent          ║
  ║                                                               ║
  ║                        Version ` + version + `                              ║
  ║                                                               ║
  ╚═══════════════════════════════════════════════════════════════╝
`
	fmt.Println(banner)
}

func runOAuthLogin() {
	verifier, challenge, err := llm.GeneratePKCE()
	if err != nil {
		log.Fatalf("Failed to generate PKCE: %v", err)
	}

	authURL := llm.BuildAuthURL(challenge, verifier)

	fmt.Println("🔑 Anthropic OAuth Login")
	fmt.Println()
	// OSC 8 hyperlink: terminal renders "Login URL" as clickable link to the full authURL
	// Supported by iTerm2, Windows Terminal, GNOME Terminal, Alacritty, etc.
	fmt.Printf("\033]8;;%s\033\\>> Click here to open login page <<\033]8;;\033\\\n", authURL)
	fmt.Println()
	fmt.Println("(If not clickable, copy the URL below)")
	fmt.Println(authURL)
	fmt.Println()
	fmt.Println("After authorizing, paste the code (code#state) and press Enter:")
	fmt.Print("> ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		fmt.Println("Cancelled.")
		return
	}
	code := strings.TrimSpace(scanner.Text())
	if code == "" {
		fmt.Println("No code provided. Cancelled.")
		return
	}

	creds, err := llm.ExchangeCode(code, verifier)
	if err != nil {
		log.Fatalf("Token exchange failed: %v", err)
	}

	if err := config.SaveOAuth(creds); err != nil {
		log.Fatalf("Failed to save credentials: %v", err)
	}

	fmt.Println()
	fmt.Printf("✅ OAuth login successful! Token expires in %d minutes.\n", creds.ExpiresIn()/60)
	fmt.Println("   You can now run ./ClosedWheeler normally.")
}

func printHelp() {
	fmt.Printf("Coder AGI v%s - Intelligent coding assistant\n\n", version)
	fmt.Println("Usage: ClosedWheeler [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -project string")
	fmt.Println("        Path to project directory (default: current directory)")
	fmt.Println("  -config string")
	fmt.Println("        Path to configuration file")
	fmt.Println("  -version")
	fmt.Println("        Show version")
	fmt.Println("  -login")
	fmt.Println("        Authenticate with Anthropic OAuth (standalone, no TUI)")
	fmt.Println("  -help")
	fmt.Println("        Show this help")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  OPENAI_API_KEY    Your OpenAI API key (required)")
	fmt.Println("  OPENAI_BASE_URL   Custom API base URL (optional)")
	fmt.Println("  OPENAI_MODEL      Model to use (optional, default: gpt-4o-mini)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ClosedWheeler")
	fmt.Println("  ClosedWheeler -project /path/to/myproject")
	fmt.Println("  ClosedWheeler -config ~/.agi/config.json")
}
