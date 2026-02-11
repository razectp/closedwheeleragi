package tui

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ClosedWheeler/pkg/llm"
	"ClosedWheeler/pkg/utils"
)

// InteractiveSetup runs enhanced interactive CLI setup
func InteractiveSetup(appRoot string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println(SetupHeaderStyle.Render("🚀 ClosedWheelerAGI - Setup"))
	fmt.Println(SetupInfoStyle.Render("Let's get you up and running in under 2 minutes."))
	fmt.Println()

	// Step 1: Agent Name
	agentName := promptString(reader, "Give your agent a name", "ClosedWheeler")
	fmt.Println(SetupSuccessStyle.Render(fmt.Sprintf("✅ Agent name: %s", agentName)))

	// Step 2: API Configuration
	fmt.Println()
	fmt.Println(SetupHeaderStyle.Render("🔑 API Configuration"))
	fmt.Println()
	fmt.Println(SetupInfoStyle.Render("Enter your API key to authenticate with your LLM provider."))
	fmt.Println(SetupInfoStyle.Render("Get yours at: console.anthropic.com / platform.openai.com"))
	fmt.Println()

	baseURL, apiKey, detectedProvider := promptAPI(reader)

	// Step 3: Fetch and select models
	fmt.Println()
	fmt.Println(SetupInfoStyle.Render("🔍 Fetching available models..."))

	models, err := llm.ListModelsWithProvider(baseURL, apiKey, detectedProvider)
	primaryModel, fallbackModels := selectModels(reader, models, err)

	// Step 3.5: Model parameter configuration via model interview
	var primaryConfig *llm.ModelSelfConfig

	fmt.Println()
	fmt.Println(SetupHeaderStyle.Render("🎤 Model Self-Configuration"))
	fmt.Println(SetupInfoStyle.Render(fmt.Sprintf("Asking '%s' to configure itself for agent work...", primaryModel)))
	fmt.Println()

	testClient := llm.NewClientWithProvider(baseURL, apiKey, primaryModel, detectedProvider)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	config, err := testClient.InterviewModel(ctx)
	cancel()

	if err != nil {
		fmt.Println()
		fmt.Println(SetupErrorStyle.Render(fmt.Sprintf("⚠️  Model self-configuration failed: %v", err)))
		fmt.Println()
		fmt.Print(SetupPromptStyle.Render(fmt.Sprintf("Use model '%s' anyway with fallback config? (Y/n): ", primaryModel)))
		useAnywayChoice, _ := reader.ReadString('\n')
		useAnywayChoice = strings.TrimSpace(strings.ToLower(useAnywayChoice))

		if useAnywayChoice == "n" || useAnywayChoice == "no" {
			return fmt.Errorf("model self-configuration failed and user declined to continue")
		}

		temp, topP, maxTok := llm.ApplyProfileToConfig(primaryModel)
		pTemp, pTopP, pMaxTok := 0.7, 1.0, 4096
		if temp != nil {
			pTemp = *temp
		}
		if topP != nil {
			pTopP = *topP
		}
		if maxTok != nil {
			pMaxTok = *maxTok
		}

		primaryConfig = &llm.ModelSelfConfig{
			ModelName:         primaryModel,
			RecommendedTemp:   pTemp,
			RecommendedTopP:   pTopP,
			RecommendedMaxTok: pMaxTok,
			ContextWindow:     128000,
		}
	} else {
		primaryConfig = config
		fmt.Println(SetupSuccessStyle.Render("✅ Model configured itself successfully!"))
		fmt.Println(SetupInfoStyle.Render(fmt.Sprintf("   Temperature: %.2f", config.RecommendedTemp)))
		fmt.Println(SetupInfoStyle.Render(fmt.Sprintf("   Top P:       %.2f", config.RecommendedTopP)))
		fmt.Println(SetupInfoStyle.Render(fmt.Sprintf("   Max Tokens:  %d", config.RecommendedMaxTok)))
	}

	// Step 4: Permissions Preset
	fmt.Println()
	fmt.Println(SetupHeaderStyle.Render("🛡️  Permissions Configuration"))
	permissionsPreset := selectPermissionsPreset(reader)
	fmt.Println(SetupSuccessStyle.Render(fmt.Sprintf("✅ Permissions: %s", permissionsPreset)))

	// Step 5: Rules Preset
	fmt.Println()
	fmt.Println(SetupHeaderStyle.Render("📜 Rules & Personality Preset"))
	rulesPreset := selectRulesPreset(reader)
	fmt.Println(SetupSuccessStyle.Render(fmt.Sprintf("✅ Rules: %s", rulesPreset)))

	// Step 6: Memory Preset
	fmt.Println()
	fmt.Println(SetupHeaderStyle.Render("🧠 Memory Storage Configuration"))
	memoryPreset := selectMemoryPreset(reader)
	fmt.Println(SetupSuccessStyle.Render(fmt.Sprintf("✅ Memory: %s", memoryPreset)))

	// Step 7: Telegram Integration (Optional)
	fmt.Println()
	fmt.Println(SetupHeaderStyle.Render("🤖 Telegram Integration (Optional)"))
	telegramToken, telegramEnabled := configureTelegram(reader)

	// Step 8: Save everything
	fmt.Println()
	fmt.Println(SetupInfoStyle.Render("💾 Saving configuration..."))

	if err := saveConfiguration(agentName, baseURL, apiKey, primaryModel, detectedProvider, fallbackModels, permissionsPreset, memoryPreset, telegramToken, telegramEnabled, primaryConfig); err != nil {
		return err
	}

	if err := saveRulesPreset(appRoot, rulesPreset); err != nil {
		return err
	}

	// Success summary
	fmt.Println()
	fmt.Println(SetupSuccessStyle.Render("🎉 Setup Complete!"))
	fmt.Println()
	fmt.Println(SetupInfoStyle.Render("Configuration Summary:"))
	fmt.Println(SetupInfoStyle.Render(fmt.Sprintf("  Agent:       %s", agentName)))
	fmt.Println(SetupInfoStyle.Render(fmt.Sprintf("  Model:       %s", primaryModel)))
	if len(fallbackModels) > 0 {
		fmt.Println(SetupInfoStyle.Render(fmt.Sprintf("  Fallbacks:   %s", strings.Join(fallbackModels, ", "))))
	}
	fmt.Println(SetupInfoStyle.Render(fmt.Sprintf("  Permissions: %s", permissionsPreset)))
	fmt.Println(SetupInfoStyle.Render(fmt.Sprintf("  Rules:       %s", rulesPreset)))
	fmt.Println(SetupInfoStyle.Render(fmt.Sprintf("  Memory:      %s", memoryPreset)))
	fmt.Println(SetupInfoStyle.Render(fmt.Sprintf("  Telegram:    %v", telegramEnabled)))
	fmt.Println()

	// Telegram pairing instructions
	if telegramEnabled {
		fmt.Println(SetupHeaderStyle.Render("📱 Telegram Pairing Instructions"))
		fmt.Println()
		fmt.Println(SetupInfoStyle.Render("To complete Telegram setup:"))
		fmt.Println(SetupInfoStyle.Render("  1. Start the ClosedWheeler agent"))
		fmt.Println(SetupInfoStyle.Render("  2. Open Telegram and find your bot"))
		fmt.Println(SetupInfoStyle.Render("  3. Send: /start"))
		fmt.Println(SetupInfoStyle.Render("  4. Copy your Chat ID from the bot's response"))
		fmt.Println(SetupInfoStyle.Render("  5. Edit .agi/config.json and set 'chat_id' field"))
		fmt.Println(SetupInfoStyle.Render("  6. Restart the agent"))
		fmt.Println()
		fmt.Println(SetupSuccessStyle.Render("💡 Tip: You can also configure Telegram later by editing .agi/config.json"))
		fmt.Println()
	}

	return nil
}

func promptString(reader *bufio.Reader, prompt, defaultValue string) string {
	fmt.Print(SetupPromptStyle.Render(fmt.Sprintf("%s [%s]: ", prompt, defaultValue)))
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}

func promptAPI(reader *bufio.Reader) (string, string, string) {
	fmt.Println()
	fmt.Println(SetupInfoStyle.Render("Examples:"))
	fmt.Println(SetupInfoStyle.Render("  1. OpenAI     - https://api.openai.com/v1"))
	fmt.Println(SetupInfoStyle.Render("  2. NVIDIA     - https://integrate.api.nvidia.com/v1"))
	fmt.Println(SetupInfoStyle.Render("  3. Anthropic  - https://api.anthropic.com/v1"))
	fmt.Println(SetupInfoStyle.Render("  4. Local      - http://localhost:11434/v1"))
	fmt.Println()

	baseURL := promptString(reader, "API Base URL", "https://api.openai.com/v1")
	apiKey := promptString(reader, "API Key", "")

	// Auto-detect provider from URL
	provider := ""
	if strings.Contains(baseURL, "anthropic.com") {
		provider = "anthropic"
		fmt.Println(SetupSuccessStyle.Render("  Detected provider: Anthropic"))
	} else if strings.Contains(baseURL, "openai.com") {
		provider = "openai"
		fmt.Println(SetupSuccessStyle.Render("  Detected provider: OpenAI"))
	} else if strings.HasPrefix(apiKey, "sk-ant-") {
		provider = "anthropic"
		fmt.Println(SetupSuccessStyle.Render("  Detected provider: Anthropic (from API key)"))
	}

	return baseURL, apiKey, provider
}

func selectModels(reader *bufio.Reader, models []llm.ModelInfo, err error) (string, []string) {
	if err != nil || len(models) == 0 {
		fmt.Println(SetupErrorStyle.Render("⚠️  Could not fetch models"))
		primary := promptString(reader, "Enter primary model", "gpt-4o-mini")
		return primary, []string{}
	}

	fmt.Println()
	fmt.Println(SetupSuccessStyle.Render(fmt.Sprintf("✅ Found %d models", len(models))))
	fmt.Println()

	// Pagination: show 10 models at a time
	page := 0
	pageSize := 10
	totalPages := (len(models) + pageSize - 1) / pageSize

	for {
		start := page * pageSize
		end := min(start+pageSize, len(models))

		fmt.Println(SetupInfoStyle.Render(fmt.Sprintf("📄 Page %d/%d (showing %d-%d of %d models)",
			page+1, totalPages, start+1, end, len(models))))
		fmt.Println()

		// Display current page
		for i := start; i < end; i++ {
			fmt.Printf("  %d. %s\n", i+1, models[i].ID)
		}
		fmt.Println()

		// Navigation instructions
		if totalPages > 1 {
			fmt.Println(SetupInfoStyle.Render("💡 Navigation:"))
			fmt.Println(SetupInfoStyle.Render("  • Type 'n' for next page, 'p' for previous page"))
			fmt.Println(SetupInfoStyle.Render(fmt.Sprintf("  • Type model number (1-%d) or model name to select", len(models))))
			fmt.Println()
		}

		fmt.Print(SetupPromptStyle.Render("Select model (number/name/n/p): "))
		selection, _ := reader.ReadString('\n')
		selection = strings.TrimSpace(selection)

		// Handle navigation
		if selection == "n" && page < totalPages-1 {
			page++
			fmt.Println()
			continue
		} else if selection == "p" && page > 0 {
			page--
			fmt.Println()
			continue
		} else if selection == "n" || selection == "p" {
			fmt.Println(SetupErrorStyle.Render("⚠️  No more pages in that direction"))
			fmt.Println()
			continue
		}

		// Handle model selection
		if idx, err := strconv.Atoi(selection); err == nil && idx > 0 && idx <= len(models) {
			primary := models[idx-1].ID
			fmt.Println(SetupSuccessStyle.Render(fmt.Sprintf("✅ Selected: %s", primary)))

			// Fallback models
			fallbacks := selectFallbackModels(reader, models, primary)
			return primary, fallbacks
		} else if selection != "" {
			primary := selection
			fmt.Println(SetupSuccessStyle.Render(fmt.Sprintf("✅ Selected: %s", primary)))

			// Fallback models
			fallbacks := selectFallbackModels(reader, models, primary)
			return primary, fallbacks
		} else {
			fmt.Println(SetupErrorStyle.Render("⚠️  Invalid selection, try again"))
			fmt.Println()
		}
	}
}

func selectFallbackModels(reader *bufio.Reader, models []llm.ModelInfo, _ string) []string {
	// Fallback models
	fmt.Println()
	fmt.Print(SetupPromptStyle.Render("Add fallback models? (y/N): "))
	addFallback, _ := reader.ReadString('\n')
	addFallback = strings.TrimSpace(strings.ToLower(addFallback))

	var fallbacks []string
	if addFallback == "y" || addFallback == "yes" {
		fmt.Println()
		fmt.Println(SetupInfoStyle.Render("Enter model numbers or names (comma-separated)"))
		fmt.Println(SetupInfoStyle.Render("Example: 2,5 or gpt-4,claude-3"))
		fmt.Print(SetupPromptStyle.Render("Fallbacks: "))

		fallbackInput, _ := reader.ReadString('\n')
		fallbackInput = strings.TrimSpace(fallbackInput)

		if fallbackInput != "" {
			parts := strings.Split(fallbackInput, ",")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if idx, err := strconv.Atoi(part); err == nil && idx > 0 && idx <= len(models) {
					fallbacks = append(fallbacks, models[idx-1].ID)
				} else if part != "" {
					fallbacks = append(fallbacks, part)
				}
			}
			fmt.Println(SetupSuccessStyle.Render(fmt.Sprintf("✅ Fallback models: %s", strings.Join(fallbacks, ", "))))
		}
	}

	return fallbacks
}

func selectPermissionsPreset(reader *bufio.Reader) string {
	fmt.Println()
	fmt.Println(SetupInfoStyle.Render("Presets:"))
	fmt.Println(SetupInfoStyle.Render("  1. Full Access    - All commands and tools (recommended for solo dev)"))
	fmt.Println(SetupInfoStyle.Render("  2. Restricted     - Only read, edit, write files (safe for teams)"))
	fmt.Println(SetupInfoStyle.Render("  3. Read-Only      - Only read operations (maximum safety)"))
	fmt.Println()

	choice := promptString(reader, "Select preset (1-3)", "1")

	switch choice {
	case "2":
		return "restricted"
	case "3":
		return "read-only"
	default:
		return "full"
	}
}

func selectRulesPreset(reader *bufio.Reader) string {
	fmt.Println()
	fmt.Println(SetupInfoStyle.Render("Presets:"))
	fmt.Println(SetupInfoStyle.Render("  1. Code Quality        - Clean, maintainable code (recommended)"))
	fmt.Println(SetupInfoStyle.Render("  2. Security First      - Security best practices"))
	fmt.Println(SetupInfoStyle.Render("  3. Performance         - Speed and efficiency"))
	fmt.Println(SetupInfoStyle.Render("  4. Personal Assistant  - Helpful and conversational"))
	fmt.Println(SetupInfoStyle.Render("  5. Cybersecurity       - Penetration testing and auditing"))
	fmt.Println(SetupInfoStyle.Render("  6. Data Science        - ML/AI and analytics"))
	fmt.Println(SetupInfoStyle.Render("  7. DevOps              - Infrastructure and automation"))
	fmt.Println(SetupInfoStyle.Render("  8. None                - No predefined rules"))
	fmt.Println()

	choice := promptString(reader, "Select preset (1-8)", "1")

	switch choice {
	case "1":
		return "code-quality"
	case "2":
		return "security"
	case "3":
		return "performance"
	case "4":
		return "personal-assistant"
	case "5":
		return "cybersecurity"
	case "6":
		return "data-science"
	case "7":
		return "devops"
	case "8":
		return "none"
	default:
		return "code-quality"
	}
}

func selectMemoryPreset(reader *bufio.Reader) string {
	fmt.Println()
	fmt.Println(SetupInfoStyle.Render("Presets:"))
	fmt.Println(SetupInfoStyle.Render("  1. Balanced  - 20/50/100 items (recommended)"))
	fmt.Println(SetupInfoStyle.Render("  2. Minimal   - 10/25/50 items (lightweight)"))
	fmt.Println(SetupInfoStyle.Render("  3. Extended  - 30/100/200 items (maximum context)"))
	fmt.Println()

	choice := promptString(reader, "Select preset (1-3)", "1")

	switch choice {
	case "2":
		return "minimal"
	case "3":
		return "extended"
	default:
		return "balanced"
	}
}

func configureTelegram(reader *bufio.Reader) (string, bool) {
	fmt.Println()
	fmt.Println(SetupInfoStyle.Render("Telegram allows you to control the agent remotely:"))
	fmt.Println(SetupInfoStyle.Render("  • Chat with the agent from anywhere"))
	fmt.Println(SetupInfoStyle.Render("  • Execute commands (/status, /logs, /model)"))
	fmt.Println(SetupInfoStyle.Render("  • Approve sensitive operations"))
	fmt.Println()

	fmt.Print(SetupPromptStyle.Render("Configure Telegram now? (y/N): "))
	configure, _ := reader.ReadString('\n')
	configure = strings.TrimSpace(strings.ToLower(configure))

	if configure != "y" && configure != "yes" {
		return "", false
	}

	fmt.Println()
	token := promptString(reader, "Enter Telegram Bot Token", "")

	if token == "" {
		return "", false
	}

	fmt.Println()
	fmt.Println(SetupSuccessStyle.Render("✅ Telegram token saved!"))

	return token, true
}

func openBrowserCLI(url string) bool {
	return utils.OpenBrowser(url) == nil
}

func saveConfiguration(agentName, baseURL, apiKey, primaryModel, provider string, fallbackModels []string, permPreset, memPreset, telegramToken string, telegramEnabled bool, primaryConfig *llm.ModelSelfConfig) error {
	// Build .env
	var env strings.Builder
	env.WriteString("# ClosedWheelerAGI Configuration\n")
	env.WriteString(fmt.Sprintf("API_BASE_URL=%s\n", baseURL))
	env.WriteString(fmt.Sprintf("API_KEY=%s\n", apiKey))
	env.WriteString(fmt.Sprintf("MODEL=%s\n", primaryModel))
	if provider != "" {
		env.WriteString(fmt.Sprintf("PROVIDER=%s\n", provider))
	}
	if telegramToken != "" {
		env.WriteString(fmt.Sprintf("TELEGRAM_BOT_TOKEN=%s\n", telegramToken))
	}

	if err := os.WriteFile(".env", []byte(env.String()), 0600); err != nil {
		return err
	}

	// Build config.json
	config := buildConfig(agentName, primaryModel, provider, fallbackModels, permPreset, memPreset, telegramEnabled, primaryConfig)
	data, _ := json.MarshalIndent(config, "", "  ")

	_ = os.MkdirAll(".agi", 0755)
	return os.WriteFile(".agi/config.json", data, 0644)
}

func buildConfig(agentName, primaryModel, provider string, fallbackModels []string, permPreset, memPreset string, telegramEnabled bool, primaryConfig *llm.ModelSelfConfig) map[string]interface{} {
	// Default memory
	mem := map[string]interface{}{
		"max_short_term_items": 20,
		"max_working_items":    50,
		"max_long_term_items":  100,
		"storage_path":         ".agi/memory.json",
	}

	switch memPreset {
	case "minimal":
		mem["max_short_term_items"] = 10
		mem["max_working_items"] = 25
	case "extended":
		mem["max_short_term_items"] = 30
		mem["max_working_items"] = 100
		mem["max_long_term_items"] = 200
	}

	// Default permissions
	perm := map[string]interface{}{
		"allowed_commands": []string{"*"},
		"allowed_tools":    []string{"*"},
		"sensitive_tools":  []string{"git_commit", "git_push", "exec_command", "write_file", "delete_file"},
	}

	switch permPreset {
	case "restricted":
		perm["allowed_tools"] = []string{"read_file", "list_files", "search_files", "edit_file", "write_file"}
	case "read-only":
		perm["allowed_tools"] = []string{"read_file", "list_files", "search_files"}
	}

	// Model params
	temp := 0.7
	topP := 1.0
	maxTok := 4096
	if primaryConfig != nil {
		temp = primaryConfig.RecommendedTemp
		topP = primaryConfig.RecommendedTopP
		maxTok = primaryConfig.RecommendedMaxTok
	}

	return map[string]interface{}{
		"agent_name":         agentName,
		"model":              primaryModel,
		"provider":           provider,
		"fallback_models":    fallbackModels,
		"temperature":        temp,
		"top_p":              topP,
		"max_tokens":         maxTok,
		"memory":             mem,
		"permissions":        perm,
		"heartbeat_interval": 0,
		"telegram": map[string]interface{}{
			"enabled": telegramEnabled,
			"chat_id": 0,
		},
		"ui": map[string]interface{}{
			"theme":   "dark",
			"verbose": false,
		},
	}
}

// saveRulesPreset saves personality and expertise templates based on the chosen preset.
func saveRulesPreset(appRoot, preset string) error {
	if preset == "none" {
		return nil
	}
	workplace := filepath.Join(appRoot, "workplace")
	if err := os.MkdirAll(workplace, 0755); err != nil {
		return fmt.Errorf("failed to create workplace directory: %w", err)
	}

	personality := "# " + preset + " Personality\n"
	if err := os.WriteFile(filepath.Join(workplace, "personality.md"), []byte(personality), 0644); err != nil {
		return fmt.Errorf("failed to write personality.md: %w", err)
	}

	expertise := "# " + preset + " Expertise\n"
	if err := os.WriteFile(filepath.Join(workplace, "expertise.md"), []byte(expertise), 0644); err != nil {
		return fmt.Errorf("failed to write expertise.md: %w", err)
	}

	return nil
}
