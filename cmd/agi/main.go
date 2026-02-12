package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"ClosedWheeler/pkg/agent"
	"ClosedWheeler/pkg/config"
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
	flag.Parse()

	if *showHelp {
		printHelp()
		return
	}

	if *showVersion {
		fmt.Printf("Coder AGI v%s\n", version)
		return
	}

	// Load configuration
	cfg, _, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	// Check API key
	if cfg.APIKey == "" {
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
		if cfg.APIKey == "" {
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

	// Redirect Go's standard log package to file BEFORE TUI starts.
	// Any log.Printf from llm, tools, browser, recovery, etc. will go to this file
	// instead of corrupting the Bubble Tea alternate screen.
	logDir := filepath.Join(appRoot, ".agi")
	_ = os.MkdirAll(logDir, 0755)
	if logFile, err := os.OpenFile(filepath.Join(logDir, "debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	// Run Enhanced TUI (passes context so cancel() forces exit even if bubbletea hangs)
	if err := tui.RunEnhanced(ag, ctx); err != nil {
		// Ignore context-cancelled errors — that's just our shutdown path
		if ctx.Err() == nil {
			log.Fatalf("❌ TUI error: %v", err)
		}
	}

	// TUI exited — restore log output to stderr for shutdown messages
	log.SetOutput(os.Stderr)

	// Reset terminal to sane state (in case bubbletea didn't restore properly)
	fmt.Print("\033[?1000l\033[?1002l\033[?1003l\033[?1006l") // disable mouse modes
	fmt.Print("\033[?25h")                                    // show cursor
	fmt.Print("\033[?1049l")                                  // exit alt screen

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
	fmt.Println("  -help")
	fmt.Println("        Show this help")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  API_KEY           Your LLM API key (required if not in config)")
	fmt.Println("  API_BASE_URL      Custom API base URL (optional)")
	fmt.Println("  MODEL             Model to use (optional)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ClosedWheeler")
	fmt.Println("  ClosedWheeler -project /path/to/myproject")
	fmt.Println("  ClosedWheeler -config ~/.agi/config.json")
}
