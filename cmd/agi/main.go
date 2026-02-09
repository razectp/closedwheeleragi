package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

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

	// Start Telegram Bridge
	ag.StartTelegram()

	// Start Heartbeat
	ag.StartHeartbeat()

	// Run Enhanced TUI
	if err := tui.RunEnhanced(ag); err != nil {
		fmt.Printf("\n❌ TUI error: %v\n", err)
		fmt.Println("Check .agi/debug.log for more details.")
		os.Exit(1)
	}

	// Graceful shutdown
	if err := ag.Close(); err != nil {
		log.Printf("⚠️  Error during shutdown: %v", err)
	}

	fmt.Println("\n👋 Goodbye!")
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
	fmt.Println("  OPENAI_API_KEY    Your OpenAI API key (required)")
	fmt.Println("  OPENAI_BASE_URL   Custom API base URL (optional)")
	fmt.Println("  OPENAI_MODEL      Model to use (optional, default: gpt-4o-mini)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ClosedWheeler")
	fmt.Println("  ClosedWheeler -project /path/to/myproject")
	fmt.Println("  ClosedWheeler -config ~/.agi/config.json")
}
