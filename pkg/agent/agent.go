// Package agent provides the core AGI agent implementation.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ClosedWheeler/pkg/brain"
	"ClosedWheeler/pkg/browser"
	"ClosedWheeler/pkg/config"
	projectcontext "ClosedWheeler/pkg/context"
	"ClosedWheeler/pkg/editor"
	"ClosedWheeler/pkg/health"
	"ClosedWheeler/pkg/llm"
	"ClosedWheeler/pkg/logger"
	"ClosedWheeler/pkg/memory"
	"ClosedWheeler/pkg/permissions"
	"ClosedWheeler/pkg/prompts"
	"ClosedWheeler/pkg/roadmap"
	"ClosedWheeler/pkg/security"
	"ClosedWheeler/pkg/skills"
	"ClosedWheeler/pkg/telegram"
	"ClosedWheeler/pkg/tools"
	"ClosedWheeler/pkg/tools/builtin"
	"ClosedWheeler/pkg/utils"
)

// Agent represents the AGI agent
type Agent struct {
	config         *config.Config
	llm            *llm.Client
	memory         *memory.Manager
	project        *projectcontext.ProjectContext
	tools          *tools.Registry
	executor       *tools.Executor
	editManager    *editor.Manager
	logger         *logger.Logger
	statusCallback func(string)
	projectPath    string
	tgBot          *telegram.Bot
	rules          *prompts.RulesManager
	auditor        *security.Auditor
	skillManager   *skills.Manager
	permManager    *permissions.Manager
	totalUsage     llm.Usage
	lastRateLimits llm.RateLimits
	approvalChan   chan bool          // Channel for Telegram approvals
	ctx            context.Context    // Context for graceful shutdown
	cancel         context.CancelFunc // Cancel function for shutdown
	sessionMgr     *SessionManager    // Session manager for context optimization
	brain          *brain.Brain       // Knowledge base for learning
	roadmap        *roadmap.Roadmap   // Strategic planning
	healthChecker  *health.Checker    // Health monitoring
	mu             sync.Mutex         // Mutex for thread safety (Heartbeat vs User)
}

// NewAgent creates a new agent instance
func NewAgent(cfg *config.Config, projectPath string, appPath string) (*Agent, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Initialize LLM client
	llmClient := llm.NewClient(cfg.APIBaseURL, cfg.APIKey, cfg.Model)

	// Configure fallback models if specified
	if len(cfg.FallbackModels) > 0 {
		llmClient.SetFallbackModels(cfg.FallbackModels, cfg.FallbackTimeout)
	}

	// Initialize memory manager
	memConfig := &memory.Config{
		MaxShortTermItems:  cfg.Memory.MaxShortTermItems,
		MaxWorkingItems:    cfg.Memory.MaxWorkingItems,
		MaxLongTermItems:   cfg.Memory.MaxLongTermItems,
		CompressionTrigger: cfg.Memory.CompressionTrigger,
	}
	memManager := memory.NewManager(cfg.Memory.StoragePath, memConfig)
	memManager.Load() // Load existing long-term memory

	// Initialize project context
	project := projectcontext.NewProjectContext(projectPath)
	if err := project.Load(cfg.IgnorePatterns); err != nil {
		return nil, fmt.Errorf("failed to load project: %w", err)
	}

	// Initialize security auditor
	auditor := security.NewAuditor(projectPath)

	// Initialize tool registry
	registry := tools.NewRegistry()

	// Configure browser options from config
	builtin.SetBrowserOptions(&browser.Options{
		Headless: cfg.Browser.Headless,
		Stealth:  cfg.Browser.Stealth,
		SlowMo:   cfg.Browser.SlowMo,
	})

	builtin.RegisterBuiltinTools(registry, projectPath, appPath, auditor)

	// Set debug level for tools if enabled
	if cfg.DebugTools {
		tools.SetGlobalDebugLevel(tools.DebugVerbose)
	}

	// Initialize logger
	l, _ := logger.New(filepath.Join(projectPath, ".agi"))

	// Initialize skill manager
	skillManager := skills.NewManager(projectPath, auditor, registry)
	if err := skillManager.LoadSkills(); err != nil {
		l.Error("Failed to load skills: %v", err)
	}

	// Initialize edit manager
	editManager := editor.NewManager(projectPath, ".agi")

	// Initialize permissions manager
	permManager, err := permissions.NewManager(&cfg.Permissions)
	if err != nil {
		return nil, fmt.Errorf("failed to create permissions manager: %w", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize brain, roadmap, and health checker
	// Brain and Roadmap should point to workplace directory
	workplacePath := projectPath
	if filepath.Base(projectPath) != "workplace" {
		workplacePath = filepath.Join(projectPath, "workplace")
	}
	brainMgr := brain.NewBrain(workplacePath)
	roadmapMgr := roadmap.NewRoadmap(workplacePath)
	healthChecker := health.NewChecker(projectPath, cfg.TestCommand)

	ag := &Agent{
		config:         cfg,
		llm:            llmClient,
		memory:         memManager,
		project:        project,
		tools:          registry,
		executor:       tools.NewExecutor(registry),
		editManager:    editManager,
		logger:         l,
		statusCallback: func(s string) {}, // Default no-op
		projectPath:    projectPath,
		tgBot:          telegram.NewBot(cfg.Telegram.BotToken, cfg.Telegram.ChatID),
		rules:          prompts.NewRulesManager(projectPath),
		auditor:        auditor,
		skillManager:   skillManager,
		permManager:    permManager,
		approvalChan:   make(chan bool),
		ctx:            ctx,
		cancel:         cancel,
		sessionMgr:     NewSessionManager(), // Initialize session manager
		brain:          brainMgr,            // Initialize brain
		roadmap:        roadmapMgr,          // Initialize roadmap
		healthChecker:  healthChecker,       // Initialize health checker
		mu:             sync.Mutex{},        // Initialize mutex
	}

	// Initialize brain and roadmap files
	if err := ag.brain.Initialize(); err != nil {
		l.Error("Failed to initialize brain: %v", err)
	}
	if err := ag.roadmap.Initialize(); err != nil {
		l.Error("Failed to initialize roadmap: %v", err)
	}

	// Load project rules
	if err := ag.rules.LoadRules(); err != nil {
		l.Error("Failed to load project rules: %v", err)
	}

	return ag, nil
}

// SetStatusCallback sets the callback for status updates
func (a *Agent) SetStatusCallback(cb func(string)) {
	a.statusCallback = func(s string) {
		if cb != nil {
			cb(s)
		}
		if a.config.Telegram.Enabled && a.config.Telegram.ChatID != 0 {
			go a.tgBot.SendMessage("📢 " + s)
		}
	}
}

// GetLogger returns the agent's logger
func (a *Agent) GetLogger() *logger.Logger {
	return a.logger
}

// Chat processes a user message and returns the response
func (a *Agent) Chat(userMessage string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Add recovery to prevent panics from killing the whole agent
	defer func() {
		if r := recover(); r != nil {
			a.logger.Error("PANIC in Agent.Chat: %v", r)
		}
	}()

	stats := a.sessionMgr.GetContextStats()
	a.logger.Info("Chat started (Current Context: %d msgs)", stats.MessageCount)


	// Age working memory at the start of each chat
	a.memory.AgeWorkingMemory(0.05) // 5% decay per hour/interaction context

	// Add user message to memory
	a.memory.AddMessage("user", userMessage)

	// Detect context and build components
	ctx := prompts.DetectContext(userMessage)
	rulesContent := a.rules.GetFormattedRules()
	projectInfo := a.project.GetSummary()
	historyInfo := a.getContextSummary()
	toolsSummary := a.getToolsSummary()

	systemPrompt := prompts.NewBuilder(ctx).
		WithToolsSummary(toolsSummary).
		WithProjectInfo(projectInfo).
		WithHistory(historyInfo).
		WithCustomInstructions(rulesContent).
		Build()

	// Build messages - only include system prompt if context needs refresh
	var messages []llm.Message
	needsContext := a.sessionMgr.NeedsContextRefresh(systemPrompt, rulesContent, projectInfo)

	if needsContext {
		// First message or context changed - send full context
		messages = append(messages, llm.Message{
			Role:    "system",
			Content: systemPrompt,
		})
		a.sessionMgr.MarkContextSent(systemPrompt, rulesContent, projectInfo)
		a.statusCallback("🔄 Refreshing context...")
	}

	// Add conversation history
	for _, msg := range a.memory.GetMessages() {
		messages = append(messages, llm.Message{
			Role:    msg["role"],
			Content: msg["content"],
		})
	}

	// Get tool definitions
	toolDefs := a.getToolDefinitions()

	// Send to LLM
	resp, err := a.llm.ChatWithTools(messages, toolDefs, a.config.Temperature, a.config.TopP, a.config.MaxTokens)
	if err != nil {
		return "", fmt.Errorf("LLM error: %w", err)
	}

	// Update usage and rate limits
	a.totalUsage.PromptTokens += resp.Usage.PromptTokens
	a.totalUsage.CompletionTokens += resp.Usage.CompletionTokens
	a.totalUsage.TotalTokens += resp.Usage.TotalTokens
	a.lastRateLimits = resp.RateLimits

	// Update session stats
	a.sessionMgr.UpdateTokenUsage(resp.Usage.PromptTokens)

	var finalResponse string
	// Handle tool calls if present
	if a.llm.HasToolCalls(resp) {
		finalResponse, err = a.handleToolCalls(resp, messages, 0)
	} else {
		finalResponse = a.llm.GetContent(resp)
		// Check for truncation
		if a.llm.GetFinishReason(resp) == "length" {
			a.logger.Info("Chat response truncated (length), requesting continuation...")
			continuation, contErr := a.continueResponse(messages, finalResponse)
			if contErr == nil {
				finalResponse += continuation
			} else {
				a.logger.Error("Continuation failed: %v", contErr)
			}
		}
		a.memory.AddMessage("assistant", finalResponse)
	}

	if err != nil {
		return "", err
	}

	// Check for context compression based on session stats
	stats = a.sessionMgr.GetContextStats()
	if stats.ShouldCompress(a.config.Memory.CompressionTrigger) {
		a.statusCallback("🗜️ Compressing context...")

		// Compress memory
		if items := a.memory.GetItemsToCompress(); len(items) > 0 {
			a.compressContext(items)
		}

		// Reset session to force context refresh on next interaction
		a.sessionMgr.ResetSession()
		a.statusCallback("✅ Context compressed and session reset")
	}

	// Proactive Insight Extraction
	if len(a.memory.GetMessages())%6 == 0 {
		go a.extractInsights()
	}

	// Sync project tasks
	a.syncProjectTasks()

	return finalResponse, nil
}

// handleToolCalls executes tool calls and continues the conversation
func (a *Agent) handleToolCalls(resp *llm.ChatResponse, messages []llm.Message, depth int) (string, error) {
	// Add recovery to prevent tool panics from killing the agent
	defer func() {
		if r := recover(); r != nil {
			a.logger.Error("PANIC in handleToolCalls (depth %d): %v", depth, r)
		}
	}()

	if depth > 50 {
		return "", fmt.Errorf("maximum tool execution depth exceeded")
	}

	if depth > 10 {
		a.logger.Info("Deep tool execution detected (depth %d), continuing task...", depth)
	}

	toolCalls := a.llm.GetToolCalls(resp)
	a.logger.Info("Executing %d tool calls at depth %d", len(toolCalls), depth)

	// Add assistant message with tool calls
	messages = append(messages, resp.Choices[0].Message)

	// Execute tools in parallel where possible
	type toolExecutionResult struct {
		tc     llm.ToolCall
		args   map[string]any
		result tools.ToolResult
		err    error
		index  int
	}

	results := make([]toolExecutionResult, len(toolCalls))

	// Separate sensitive tools (require sequential approval) from non-sensitive
	var sensitiveCalls []int
	var nonSensitiveCalls []int

	for i, tc := range toolCalls {
		// Parse arguments first
		var args map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			a.logger.Error("Failed to unmarshal arguments for tool %s: %v", tc.Function.Name, err)
			results[i] = toolExecutionResult{
				tc:    tc,
				args:  args,
				result: tools.ToolResult{Success: false, Output: fmt.Sprintf("Error: %v", err)},
				err:   err,
				index: i,
			}
			continue
		}

		results[i].tc = tc
		results[i].args = args
		results[i].index = i

		// Check if tool requires approval
		if a.permManager.RequiresApproval(tc.Function.Name) {
			sensitiveCalls = append(sensitiveCalls, i)
		} else {
			nonSensitiveCalls = append(nonSensitiveCalls, i)
		}
	}

	// Execute non-sensitive tools in parallel
	if len(nonSensitiveCalls) > 0 {
		a.logger.Info("Executing %d non-sensitive tools in parallel", len(nonSensitiveCalls))

		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, idx := range nonSensitiveCalls {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				// Add recovery for each parallel tool execution
				defer func() {
					if r := recover(); r != nil {
						a.logger.Error("PANIC in parallel tool execution: %v", r)
					}
				}()

				tc := results[i].tc
				args := results[i].args

				a.logger.Info("Tool call (parallel): %s(%v)", tc.Function.Name, tc.Function.Arguments)
				a.statusCallback(fmt.Sprintf("🔧 Executing %s...", tc.Function.Name))

				result, err := a.executor.Execute(tools.ToolCall{
					Name:      tc.Function.Name,
					Arguments: args,
				})

				// Enhance errors with detailed feedback for LLM
				if !result.Success && result.Error != "" {
					result = tools.EnhanceToolError(tc.Function.Name, args, result)
				}

				mu.Lock()
				results[i].result = result
				results[i].err = err
				mu.Unlock()

				if err != nil {
					a.logger.Error("Tool %s execution error: %v", tc.Function.Name, err)
				} else if !result.Success {
					a.logger.Error("Tool %s failed: %s", tc.Function.Name, result.Error)
				}
			}(idx)
		}

		wg.Wait()
	}

	// Execute sensitive tools sequentially (require approval)
	for _, idx := range sensitiveCalls {
		tc := results[idx].tc
		args := results[idx].args

		a.logger.Info("Tool call (sequential): %s(%v)", tc.Function.Name, tc.Function.Arguments)
		a.statusCallback(fmt.Sprintf("🔧 Executing %s...", tc.Function.Name))

		// Request approval if Telegram enabled
		if a.config.Telegram.Enabled {
			if err := a.requestTelegramApproval(tc.Function.Name, tc.Function.Arguments); err != nil {
				a.logger.Error("Telegram approval failed or denied: %v", err)
				results[idx].result = tools.ToolResult{
					Success: false,
					Output:  "Error: Operation denied by user via Telegram.",
				}
				results[idx].err = err
				continue
			}
		}

		result, err := a.executor.Execute(tools.ToolCall{
			Name:      tc.Function.Name,
			Arguments: args,
		})

		// Enhance errors with detailed feedback for LLM
		if !result.Success && result.Error != "" {
			result = tools.EnhanceToolError(tc.Function.Name, args, result)
		}

		results[idx].result = result
		results[idx].err = err

		if err != nil {
			a.logger.Error("Tool %s execution error: %v", tc.Function.Name, err)
		} else if !result.Success {
			a.logger.Error("Tool %s failed: %s", tc.Function.Name, result.Error)
		}
	}

	// Process results in original order and add to messages
	for i, res := range results {
		result := res.result

		// Ensure result has error info if err is set
		if res.err != nil && result.Output == "" {
			result.Output = fmt.Sprintf("Error: %v", res.err)
			result.Success = false
		}

		// Add tool result to messages
		messages = append(messages, llm.Message{
			Role:       "tool",
			Content:    result.Output,
			ToolCallID: toolCalls[i].ID,
		})

		// Add relevant files to working memory
		if result.Success {
			if path, ok := res.args["path"].(string); ok {
				// High initial relevance for manual reads
				if res.tc.Function.Name == "read_file" || res.tc.Function.Name == "view_file" {
					a.memory.AddFile(path, result.Output, 1.0)
				}
			}
		}
	}

	// Get tool definitions for follow-up
	toolDefs := a.getToolDefinitions()

	// Continue conversation with tool results
	resp, err := a.llm.ChatWithTools(messages, toolDefs, a.config.Temperature, a.config.TopP, a.config.MaxTokens)
	if err != nil {
		a.logger.Error("LLM follow-up error: %v", err)
		return "", err
	}

	// Handle nested tool calls (recursive)
	if a.llm.HasToolCalls(resp) {
		return a.handleToolCalls(resp, messages, depth+1)
	}

	content := a.llm.GetContent(resp)
	// Check for truncation in follow-up
	if a.llm.GetFinishReason(resp) == "length" {
		a.logger.Info("Tool follow-up truncated (length), requesting continuation...")
		continuation, contErr := a.continueResponse(messages, content)
		if contErr == nil {
			content += continuation
		}
	}
	a.memory.AddMessage("assistant", content)

	return content, nil
}

// getToolDefinitions returns tool definitions for the LLM
func (a *Agent) getToolDefinitions() []llm.ToolDefinition {
	defs := make([]llm.ToolDefinition, 0)
	for _, tool := range a.tools.List() {
		defs = append(defs, llm.ToolDefinition{
			Type: "function",
			Function: llm.FunctionSchema{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		})
	}
	return defs
}

// getToolsSummary generates a concise summary of available tools
func (a *Agent) getToolsSummary() string {
	var sb strings.Builder
	sb.WriteString("You have access to the following tools (use them via function calls):\n\n")

	toolsList := a.tools.List()

	// Group tools by category
	fileTools := []string{}
	browserTools := []string{}
	gitTools := []string{}
	otherTools := []string{}

	for _, tool := range toolsList {
		name := tool.Name
		desc := tool.Description
		if len(desc) > 80 {
			desc = desc[:77] + "..."
		}
		toolStr := fmt.Sprintf("- **%s**: %s", name, desc)

		// Categorize
		lowerName := strings.ToLower(name)
		if strings.Contains(lowerName, "file") || strings.Contains(lowerName, "read") ||
		   strings.Contains(lowerName, "write") || strings.Contains(lowerName, "edit") {
			fileTools = append(fileTools, toolStr)
		} else if strings.Contains(lowerName, "browser") || strings.Contains(lowerName, "navigate") {
			browserTools = append(browserTools, toolStr)
		} else if strings.Contains(lowerName, "git") {
			gitTools = append(gitTools, toolStr)
		} else {
			otherTools = append(otherTools, toolStr)
		}
	}

	// Write categorized tools
	if len(fileTools) > 0 {
		sb.WriteString("### File Operations\n")
		for _, t := range fileTools {
			sb.WriteString(t + "\n")
		}
		sb.WriteString("\n")
	}

	if len(browserTools) > 0 {
		sb.WriteString("### Browser Automation\n")
		for _, t := range browserTools {
			sb.WriteString(t + "\n")
		}
		sb.WriteString("\n")
	}

	if len(gitTools) > 0 {
		sb.WriteString("### Version Control\n")
		for _, t := range gitTools {
			sb.WriteString(t + "\n")
		}
		sb.WriteString("\n")
	}

	if len(otherTools) > 0 {
		sb.WriteString("### Other Tools\n")
		for _, t := range otherTools {
			sb.WriteString(t + "\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("**Total**: %d tools available. Use them to accomplish tasks efficiently.", len(toolsList)))

	return sb.String()
}

// getContextSummary returns a summary of long-term context
func (a *Agent) getContextSummary() string {
	return a.memory.GetContext()
}

// compressContext uses LLM to compress old context
func (a *Agent) compressContext(items []*memory.MemoryItem) {
	var conversation strings.Builder
	for _, item := range items {
		conversation.WriteString(fmt.Sprintf("%s: %s\n", item.Metadata.Role, item.Content))
	}

	prompt := fmt.Sprintf(`### Context Compression Task
Summarize the following conversation segment in 2-3 concise bullet points. 
Focus strictly on:
1. Technical decisions reached.
2. Patterns discovered in the codebase.
3. Errors or obstacles encountered and how they were solved.

Conversation Segment:
%s

Summary:`, conversation.String())

	summary, err := a.llm.SimpleQuery(prompt, utils.FloatPtr(0.3), nil, utils.IntPtr(300))
	if err != nil {
		a.logger.Error("Context compression failed: %v", err)
		return
	}

	a.memory.CompressItems(summary)
	a.logger.Info("Context compressed successfully.")
}

// extractInsights identifies patterns and decisions from recent memory
func (a *Agent) extractInsights() {
	messages := a.memory.GetMessages()
	if len(messages) < 4 {
		return
	}

	var chat strings.Builder
	// Get last 4 messages
	for i := len(messages) - 4; i < len(messages); i++ {
		chat.WriteString(fmt.Sprintf("%s: %s\n", messages[i]["role"], messages[i]["content"]))
	}

	prompt := fmt.Sprintf(`### Insight Extraction Task
Based on the recent interaction below, identify if any permanent technical decisions or recurring project patterns were established.
If yes, provide a single sentence starting with "Decision:" or "Pattern:". 
If nothing significant was established, reply with "NONE".

Recent Interaction:
%s

Insight:`, chat.String())

	insight, err := a.llm.SimpleQuery(prompt, utils.FloatPtr(0.2), nil, utils.IntPtr(150))
	if err != nil || strings.ToUpper(insight) == "NONE" || insight == "" {
		return
	}

	a.memory.AddDecision(insight, []string{"proactive-insight"})
	a.logger.Info("Proactive insight extracted: %s", insight)
}

// GetProjectInfo returns project information
func (a *Agent) GetProjectInfo() string {
	return a.project.GetSummary()
}

// GetMemoryStats returns memory statistics
func (a *Agent) GetMemoryStats() map[string]int {
	return a.memory.Stats()
}

// Save saves agent state
func (a *Agent) Save() error {
	return a.memory.Save()
}

// Close performs a graceful shutdown of the agent and its resources
func (a *Agent) Close() error {
	a.logger.Info("Stopping Heartbeat...")
	a.cancel() // Stop background routines

	a.logger.Info("Saving state...")
	if err := a.Save(); err != nil {
		a.logger.Error("Failed to save state: %v", err)
	}

	a.logger.Info("Closing browser...")
	if err := builtin.CloseBrowserManager(); err != nil {
		a.logger.Error("Failed to close browser: %v", err)
	}

	return nil
}

// Shutdown gracefully shuts down the agent
func (a *Agent) Shutdown() error {
	// Cancel context to stop goroutines
	if a.cancel != nil {
		a.cancel()
	}

	// Close browser manager
	if err := builtin.CloseBrowserManager(); err != nil {
		a.logger.Info("Failed to close browser manager: %v", err)
	}

	// Close permissions manager (closes audit log)
	if a.permManager != nil {
		if err := a.permManager.Close(); err != nil {
			return fmt.Errorf("failed to close permissions manager: %w", err)
		}
	}

	// Save memory state
	return a.Save()
}

// Config returns the agent configuration
func (a *Agent) Config() *config.Config {
	return a.config
}

// GetRulesSummary returns a summary of loaded rules
func (a *Agent) GetRulesSummary() string {
	return a.rules.GetRulesSummary()
}

// GetFormattedRules returns all active rules
func (a *Agent) GetFormattedRules() string {
	return a.rules.GetFormattedRules()
}

// GetUsageStats returns current token usage and rate limit information
func (a *Agent) GetUsageStats() map[string]any {
	return map[string]any{
		"prompt_tokens":      a.totalUsage.PromptTokens,
		"completion_tokens":  a.totalUsage.CompletionTokens,
		"total_tokens":       a.totalUsage.TotalTokens,
		"remaining_requests": a.lastRateLimits.RemainingRequests,
		"remaining_tokens":   a.lastRateLimits.RemainingTokens,
		"reset_requests":     a.lastRateLimits.ResetRequests,
		"reset_tokens":       a.lastRateLimits.ResetTokens,
	}
}

// GetContextStats returns current context session statistics
func (a *Agent) GetContextStats() ContextStats {
	return a.sessionMgr.GetContextStats()
}

// SaveConfig saves current configuration
func (a *Agent) SaveConfig() error {
	configPath := filepath.Join(a.projectPath, ".agi", "config.json")
	return a.config.Save(configPath)
}

// ReloadProject reloads the project context, rules, and skills
func (a *Agent) ReloadProject() error {
	a.rules.LoadRules()
	if err := a.skillManager.LoadSkills(); err != nil {
		a.logger.Error("Failed to reload skills: %v", err)
	}
	return a.project.Load(a.config.IgnorePatterns)
}

// AddDecision adds an important decision to long-term memory
func (a *Agent) AddDecision(decision string, tags []string) {
	a.memory.AddDecision(decision, tags)
}

// StartTelegram starts the Telegram background polling loop
func (a *Agent) StartTelegram() {
	if !a.config.Telegram.Enabled || a.config.Telegram.BotToken == "" {
		return
	}

	go func() {
		var offset int64
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-a.ctx.Done():
				// Context cancelled, shutdown gracefully
				a.logger.Info("Telegram polling stopped")
				return

			case <-ticker.C:
				updates, err := a.tgBot.GetUpdates(offset)
				if err != nil {
					a.logger.Error("Telegram update error: %v", err)
					continue
				}

				for _, u := range updates {
					if u.UpdateID >= offset {
						offset = u.UpdateID + 1
					}

					// Handle callback queries (approval buttons)
					if u.CallbackQuery != nil {
						// Null pointer guards - check all nested structures
						if u.CallbackQuery.Message == nil {
							a.logger.Error("Received callback query with nil message")
							continue
						}
						// Chat is a struct, not a pointer, so we check Message only

						if u.CallbackQuery.Message.Chat.ID == a.config.Telegram.ChatID {
							switch u.CallbackQuery.Data {
							case "approve":
								select {
								case a.approvalChan <- true:
									if err := a.tgBot.AnswerCallbackQuery(u.CallbackQuery.ID, "Aprovado!"); err != nil {
										a.logger.Error("Failed to answer callback query: %v", err)
									}
								default:
									a.logger.Error("Approval channel full, discarding approval")
								}
							case "deny":
								select {
								case a.approvalChan <- false:
									if err := a.tgBot.AnswerCallbackQuery(u.CallbackQuery.ID, "Negado."); err != nil {
										a.logger.Error("Failed to answer callback query: %v", err)
									}
								default:
									a.logger.Error("Approval channel full, discarding denial")
								}
							}
						}
						continue
					}

					if u.Message == nil {
						continue
					}

					// Check if command is allowed
					command := strings.ToLower(u.Message.Text)
					if !a.permManager.IsCommandAllowed(command) {
						a.tgBot.SendMessageToChat(u.Message.Chat.ID, fmt.Sprintf("🔒 *Comando não permitido:* `%s`", command))
						continue
					}

					// Handle Commands
					switch command {
					case "/start":
						msg := fmt.Sprintf("👋 *Hello! Welcome to ClosedWheelerAGI*\n\nYour Chat ID: `%d`\n\nConfigure this ID in config.json (`telegram.chat_id` field) to enable remote control.\n\nUse /help to see available commands.", u.Message.Chat.ID)
						a.tgBot.SendMessageToChat(u.Message.Chat.ID, msg)
						a.logger.Info("Telegram pairing requested by Chat ID: %d", u.Message.Chat.ID)

					case "/help":
						if u.Message.Chat.ID == a.config.Telegram.ChatID {
							helpMsg := `🤖 *ClosedWheelerAGI - Telegram Commands*

*Available Commands:*

/start - Initial information and your Chat ID
/help - This help message
/status - Memory and project status
/logs - Last system logs
/diff - Git repository differences
/model - View or change current model
  • /model - View current model and fallbacks
  • /model <name> - Switch to another model
/config reload - Reload configuration from file

*Conversation:*
Send any message without "/" to chat with the AGI!

Examples:
• _"Analyze the code in main.go"_
• _"Create a function to calculate fibonacci"_
• _"Explain what the User class does"_
• _"Refactor the getUsers() method"_

The AGI has full access to the project and can execute tools as configured in permissions.`
							a.tgBot.SendMessage(helpMsg)
						} else {
							a.tgBot.SendMessageToChat(u.Message.Chat.ID, fmt.Sprintf("🔒 *Access denied.*\nYour Chat ID (`%d`) is not authorized.", u.Message.Chat.ID))
						}

					case "/status":
						if u.Message.Chat.ID == a.config.Telegram.ChatID {
							stats := a.memory.Stats()
							msg := fmt.Sprintf("📊 *System Status*\n\n🧠 *Memory:*\nShort Term: %d/%d\nLong Term: %d/%d\n\n📂 *Project:* %s\n💓 *Heartbeat:* %ds",
								stats["short_term"], a.config.Memory.MaxShortTermItems,
								stats["long_term"], a.config.Memory.MaxLongTermItems,
								a.projectPath,
								a.config.HeartbeatInterval)
							a.tgBot.SendMessage(msg)
						} else {
							a.tgBot.SendMessageToChat(u.Message.Chat.ID, fmt.Sprintf("🔒 *Access denied.*\nYour Chat ID (`%d`) is not authorized in config.json.", u.Message.Chat.ID))
						}
					case "/logs":
						if u.Message.Chat.ID == a.config.Telegram.ChatID {
							// Simple way to get last logs
							logPath := filepath.Join(a.projectPath, ".agi", "agent.log")
							content, err := os.ReadFile(logPath)
							if err != nil {
								a.logger.Error("Failed to read log file: %v", err)
								a.tgBot.SendMessage("❌ *Error reading logs*")
								continue
							}
							lines := strings.Split(string(content), "\n")
							start := len(lines) - 20
							if start < 0 {
								start = 0
							}
							a.tgBot.SendMessage(fmt.Sprintf("📜 *Latest Logs:*\n```\n%s\n```", strings.Join(lines[start:], "\n")))
						}
					case "/diff":
						if u.Message.Chat.ID == a.config.Telegram.ChatID {
							res, err := a.executor.Execute(tools.ToolCall{Name: "git_diff", Arguments: map[string]any{}})
							if err != nil {
								a.logger.Error("Failed to execute git_diff: %v", err)
								a.tgBot.SendMessage("❌ *Error executing git diff*")
								continue
							}
							a.tgBot.SendMessage(fmt.Sprintf("🔍 *Git Diff:*\n```diff\n%s\n```", truncateAgentContent(res.Output, 3500)))
						}

					case "/model":
						if u.Message.Chat.ID == a.config.Telegram.ChatID {
							parts := strings.Fields(command)
							if len(parts) == 1 {
								// Show current model
								msg := fmt.Sprintf("🤖 *Current Model*\n\n*Primary:* `%s`", a.config.Model)
								if len(a.config.FallbackModels) > 0 {
									msg += fmt.Sprintf("\n*Fallbacks:* `%s`", strings.Join(a.config.FallbackModels, "`, `"))
								}
								a.tgBot.SendMessage(msg)
							} else if len(parts) == 2 {
								// Change model
								newModel := parts[1]
								a.config.Model = newModel
								a.llm = llm.NewClient(a.config.APIBaseURL, a.config.APIKey, newModel)
								if len(a.config.FallbackModels) > 0 {
									a.llm.SetFallbackModels(a.config.FallbackModels, a.config.FallbackTimeout)
								}
								if err := a.config.Save(filepath.Join(a.projectPath, ".agi", "config.json")); err != nil {
									a.logger.Error("Failed to save config: %v", err)
								}
								a.tgBot.SendMessage(fmt.Sprintf("✅ *Model changed to:* `%s`", newModel))
							} else {
								a.tgBot.SendMessage("❌ *Usage:* `/model` or `/model <model-name>`")
							}
						}

					case "/config":
						if u.Message.Chat.ID == a.config.Telegram.ChatID {
							parts := strings.Fields(command)
							if len(parts) == 2 && parts[1] == "reload" {
								// Reload configuration
								a.tgBot.SendMessage("🔄 *Reloading configuration...*")

								newConfig, _, err := config.Load(filepath.Join(a.projectPath, ".agi", "config.json"))
								if err != nil {
									a.logger.Error("Failed to reload config: %v", err)
									a.tgBot.SendMessage(fmt.Sprintf("❌ *Error:* %v", err))
									continue
								}

								// Update agent configuration
								a.config = newConfig

								// Recreate LLM client with new settings
								a.llm = llm.NewClient(a.config.APIBaseURL, a.config.APIKey, a.config.Model)
								if len(a.config.FallbackModels) > 0 {
									a.llm.SetFallbackModels(a.config.FallbackModels, a.config.FallbackTimeout)
								}

								// Update permissions manager
								if a.permManager != nil {
									a.permManager.Close()
								}
								a.permManager, err = permissions.NewManager(&a.config.Permissions)
								if err != nil {
									a.logger.Error("Failed to reload permissions: %v", err)
								}

								a.logger.Info("Configuration reloaded successfully")
								a.tgBot.SendMessage("✅ *Configuration reloaded!*\n\n*Model:* `" + a.config.Model + "`")
							} else {
								a.tgBot.SendMessage("❌ *Usage:* `/config reload`")
							}
						}

					default:
						// Handle normal conversation (non-commands)
						if u.Message.Chat.ID == a.config.Telegram.ChatID && !strings.HasPrefix(command, "/") {
							// Process message with agent
							go a.handleTelegramChat(u.Message.Text, u.Message.Chat.ID)
						}
					}
				}
			}
		}
	}()
}

// handleTelegramChat processes a chat message from Telegram
func (a *Agent) handleTelegramChat(userMessage string, chatID int64) {
	a.logger.Info("Telegram chat from %d: %s", chatID, userMessage)

	// Send typing indicator
	a.tgBot.SendMessage("💭 _Thinking..._")

	// Process message with agent
	response, err := a.Chat(userMessage)
	if err != nil {
		a.logger.Error("Telegram chat error: %v", err)
		a.tgBot.SendMessage(fmt.Sprintf("❌ *Error:* %v", err))
		return
	}

	// Split response if too long (Telegram limit: 4096 chars)
	maxLen := 4000
	if len(response) <= maxLen {
		a.tgBot.SendMessage(response)
	} else {
		// Split into chunks
		parts := splitMessage(response, maxLen)
		for i, part := range parts {
			header := ""
			if i == 0 {
				header = fmt.Sprintf("📝 *Response (part %d/%d):*\n\n", i+1, len(parts))
			} else {
				header = fmt.Sprintf("_(Continued %d/%d)_\n\n", i+1, len(parts))
			}
			a.tgBot.SendMessage(header + part)
			time.Sleep(500 * time.Millisecond) // Avoid rate limit
		}
	}
}

// splitMessage splits a long message into chunks
func splitMessage(message string, maxLen int) []string {
	if len(message) <= maxLen {
		return []string{message}
	}

	var parts []string
	for len(message) > 0 {
		if len(message) <= maxLen {
			parts = append(parts, message)
			break
		}

		// Try to split at newline
		splitPos := maxLen
		lastNewline := strings.LastIndex(message[:maxLen], "\n")
		if lastNewline > maxLen/2 { // Only if newline is in second half
			splitPos = lastNewline + 1
		}

		parts = append(parts, message[:splitPos])
		message = message[splitPos:]
	}

	return parts
}

func truncateAgentContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "\n... (truncated)"
}

// isSensitiveTool returns true if the tool requires manual approval
func (a *Agent) isSensitiveTool(name string) bool {
	return a.permManager.IsSensitiveTool(name)
}

// requestTelegramApproval sends an approval request and waits for a response
func (a *Agent) requestTelegramApproval(toolName, args string) error {
	a.statusCallback("⏳ Waiting for remote approval via Telegram...")

	// Escape special markdown characters in arguments
	escapedArgs := strings.ReplaceAll(args, "`", "'")
	escapedArgs = strings.ReplaceAll(escapedArgs, "*", "")
	escapedArgs = strings.ReplaceAll(escapedArgs, "_", "")

	// Truncate if too long (Telegram has limits)
	if len(escapedArgs) > 500 {
		escapedArgs = escapedArgs[:500] + "..."
	}

	msg := fmt.Sprintf("⚠️ *Approval Request*\n\n*Tool:* `%s`\n*Arguments:*\n```\n%s\n```", toolName, escapedArgs)
	buttons := [][]telegram.InlineButton{
		{
			{Text: "✅ Approve", CallbackData: "approve"},
			{Text: "❌ Deny", CallbackData: "deny"},
		},
	}

	if err := a.tgBot.SendMessageWithButtons(a.config.Telegram.ChatID, msg, buttons); err != nil {
		return fmt.Errorf("failed to send approval request: %w", err)
	}

	// Wait for response with timeout
	timeout := a.permManager.GetApprovalTimeout()
	ctx, cancel := context.WithTimeout(a.ctx, timeout)
	defer cancel()

	select {
	case approved := <-a.approvalChan:
		// Log the approval decision
		a.permManager.LogApprovalDecision(toolName, approved, a.config.Telegram.ChatID)
		if !approved {
			return fmt.Errorf("user denied the operation")
		}
		return nil
	case <-ctx.Done():
		// Log timeout
		a.permManager.LogApprovalTimeout(toolName)
		return fmt.Errorf("approval request timed out after %v", timeout)
	}
}

// ClearMemory clears a memory tier
func (a *Agent) ClearMemory(tier memory.MemoryTier) {
	a.memory.Clear(tier)
}

// ChatWithStreaming processes a user message with streaming response
func (a *Agent) ChatWithStreaming(userMessage string, callback llm.StreamingCallback) (string, error) {
	// Age working memory
	a.memory.AgeWorkingMemory(0.05)

	// Add user message to memory
	a.memory.AddMessage("user", userMessage)

	// Detect context and build system prompt
	ctx := prompts.DetectContext(userMessage)
	systemPrompt := prompts.NewBuilder(ctx).
		WithToolsSummary(a.getToolsSummary()).
		WithProjectInfo(a.project.GetSummary()).
		WithHistory(a.getContextSummary()).
		WithCustomInstructions(a.rules.GetFormattedRules()).
		Build()

	// Build messages
	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
	}

	// Add conversation history
	for _, msg := range a.memory.GetMessages() {
		messages = append(messages, llm.Message{
			Role:    msg["role"],
			Content: msg["content"],
		})
	}

	// Send to LLM with streaming
	resp, err := a.llm.ChatWithStreaming(messages, a.getToolDefinitions(), a.config.Temperature, a.config.TopP, a.config.MaxTokens, callback)
	if err != nil {
		return "", fmt.Errorf("LLM error: %w", err)
	}

	var finalResponse string
	// Handle tool calls if present (no streaming for tool results)
	if a.llm.HasToolCalls(resp) {
		finalResponse, err = a.handleToolCalls(resp, messages, 0)
	} else {
		finalResponse = a.llm.GetContent(resp)
		a.memory.AddMessage("assistant", finalResponse)
	}

	if err != nil {
		return "", err
	}

	// Check for context compression based on session stats
	stats := a.sessionMgr.GetContextStats()
	if stats.ShouldCompress(a.config.Memory.CompressionTrigger) {
		a.statusCallback("🗜️ Compressing context...")

		// Compress memory
		if items := a.memory.GetItemsToCompress(); len(items) > 0 {
			a.compressContext(items)
		}

		// Reset session to force context refresh on next interaction
		a.sessionMgr.ResetSession()
		a.statusCallback("✅ Context compressed and session reset")
	}

	// Sync project tasks
	a.syncProjectTasks()

	return finalResponse, nil
}

// syncProjectTasks ensures the project's task.md is initialized
func (a *Agent) syncProjectTasks() {
	taskPath := filepath.Join(a.projectPath, "task.md")
	if _, err := os.Stat(taskPath); os.IsNotExist(err) {
		a.logger.Info("Initializing project task.md")
		initialContent := "# 📋 Project Tasks\n\n- [ ] Initial project audit and setup\n"
		os.WriteFile(taskPath, []byte(initialContent), 0644)
	}
}

// GetEditManager returns the edit manager for session tracking
func (a *Agent) GetEditManager() *editor.Manager {
	return a.editManager
}

// StartEditSession starts a new editing session
func (a *Agent) StartEditSession(description string) {
	a.editManager.StartSession(description)
}

// CompleteEditSession completes the current editing session
func (a *Agent) CompleteEditSession() error {
	return a.editManager.CompleteSession()
}

// RollbackEdits rolls back all edits in the current session
func (a *Agent) RollbackEdits() error {
	return a.editManager.RollbackAll()
}

// StartHeartbeat starts a background routine with reflection and health monitoring
func (a *Agent) StartHeartbeat() {
	if a.config.HeartbeatInterval <= 0 {
		a.logger.Info("Heartbeat disabled (interval <= 0)")
		return
	}

	a.logger.Info("Starting heartbeat with reflection (interval: %ds)", a.config.HeartbeatInterval)

	go func() {
		ticker := time.NewTicker(time.Duration(a.config.HeartbeatInterval) * time.Second)
		defer ticker.Stop()

		heartbeatCount := 0

		for {
			select {
			case <-a.ctx.Done():
				a.logger.Info("Heartbeat stopped")
				return
			case t := <-ticker.C:
				heartbeatCount++
				a.logger.Info("💓 Heartbeat #%d at %s", heartbeatCount, t.Format(time.RFC3339))

				// Perform health check
				healthStatus := a.healthChecker.Check()

				// Log health summary
				a.logger.Info("Health: Build=%s, Tests=%s, Git=%s, Tasks=%d",
					healthStatus.BuildStatus,
					healthStatus.TestStatus,
					healthStatus.GitStatus,
					healthStatus.PendingTasks)

				// Check for critical issues
				hasCriticalIssues := healthStatus.BuildStatus == "failing" ||
					healthStatus.TestStatus == "failing"

				// Read task.md directly
				taskPath := filepath.Join(a.projectPath, "workplace", "task.md")
				content, err := os.ReadFile(taskPath)
				if err != nil {
					// Fallback to root task.md
					taskPath = filepath.Join(a.projectPath, "task.md")
					content, err = os.ReadFile(taskPath)
					if err != nil {
						a.logger.Error("Heartbeat failed to read task.md: %v", err)
						continue
					}
				}

				// Simple heuristic to check for pending tasks
				taskStr := string(content)
				hasPending := strings.Contains(taskStr, "- [ ]") || strings.Contains(taskStr, "- [/]")

				// Decide if agent should wake up
				shouldAct := hasPending || hasCriticalIssues

				if shouldAct {
					a.logger.Info("Heartbeat: Waking up agent (pending=%v, critical=%v)",
						hasPending, hasCriticalIssues)

					// Build reflection prompt with health context
					prompt := a.buildHeartbeatPrompt(t, healthStatus, hasPending)

					// Execute Chat (will lock mutex)
					resp, err := a.Chat(prompt)
					if err != nil {
						a.logger.Error("Heartbeat chat error: %v", err)

						// Learn from the error
						a.brain.AddError(
							"Heartbeat Execution Failed",
							fmt.Sprintf("Error during heartbeat: %v", err),
							"Check logs and LLM configuration",
							[]string{"heartbeat", "error"},
						)
					} else {
						a.logger.Info("Heartbeat response: %s", resp)

						// If critical issues were resolved, record it
						if hasCriticalIssues {
							newStatus := a.healthChecker.Check()
							if newStatus.BuildStatus == "passing" && newStatus.TestStatus == "passing" {
								a.brain.AddInsight(
									"Heartbeat Resolved Critical Issues",
									"The agent successfully resolved build/test failures during heartbeat",
									[]string{"heartbeat", "success", "auto-fix"},
								)
							}
						}
					}
				} else {
					a.logger.Info("No action needed - project health is good and no pending tasks")
				}

				// Every 5th heartbeat, perform deeper reflection
				if heartbeatCount % 5 == 0 {
					a.performDeepReflection(healthStatus)
				}
			}
		}
	}()
}

// buildHeartbeatPrompt constructs a context-aware prompt for heartbeat
func (a *Agent) buildHeartbeatPrompt(timestamp time.Time, health *health.Status, hasPending bool) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("🔔 **Heartbeat Execution** - %s\n\n", timestamp.Format("2006-01-02 15:04:05")))
	sb.WriteString("## 🏥 Project Health Status\n\n")
	sb.WriteString(fmt.Sprintf("- **Build:** %s\n", health.BuildStatus))
	sb.WriteString(fmt.Sprintf("- **Tests:** %s\n", health.TestStatus))
	sb.WriteString(fmt.Sprintf("- **Git:** %s", health.GitStatus))
	if health.GitUncommitted > 0 {
		sb.WriteString(fmt.Sprintf(" (%d uncommitted files)", health.GitUncommitted))
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("- **Pending Tasks:** %d\n\n", health.PendingTasks))

	if len(health.Warnings) > 0 {
		sb.WriteString("⚠️ **Warnings:**\n")
		for _, warning := range health.Warnings {
			sb.WriteString(fmt.Sprintf("- %s\n", warning))
		}
		sb.WriteString("\n")
	}

	if len(health.Recommendations) > 0 {
		sb.WriteString("💡 **Recommendations:**\n")
		for _, rec := range health.Recommendations {
			sb.WriteString(fmt.Sprintf("- %s\n", rec))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## 📋 Your Actions\n\n")

	if health.BuildStatus == "failing" {
		sb.WriteString("🚨 **PRIORITY:** Build is failing. Please fix build errors immediately.\n\n")
	} else if health.TestStatus == "failing" {
		sb.WriteString("⚠️ **PRIORITY:** Tests are failing. Please address test failures.\n\n")
	} else if hasPending {
		sb.WriteString("Please read `workplace/task.md` and execute pending tasks.\n\n")
	}

	sb.WriteString("Respond with:\n")
	sb.WriteString("1. If you took action: Brief summary of what was done\n")
	sb.WriteString("2. If no action needed: Just say 'NO PENDING TASKS'\n")

	return sb.String()
}

// performDeepReflection performs strategic reflection on project state
func (a *Agent) performDeepReflection(health *health.Status) {
	a.logger.Info("🧠 Performing deep reflection...")

	// Read roadmap to check strategic goals
	_, err := a.roadmap.Read()
	if err != nil {
		a.logger.Error("Failed to read roadmap: %v", err)
		return
	}

	// Read brain to review learnings
	brainContent, err := a.brain.Read()
	if err != nil {
		a.logger.Error("Failed to read brain: %v", err)
		return
	}

	// Build reflection prompt
	reflectionPrompt := fmt.Sprintf(`🧠 **Strategic Reflection**

You are performing a deep reflection on the project state. Please analyze:

1. **Recent Learnings** (from brain.md)
2. **Strategic Goals** (from roadmap.md)
3. **Current Health Status**

Based on this analysis:
- Identify patterns or recurring issues
- Suggest strategic improvements
- Update roadmap if priorities have shifted
- Record important insights in brain.md

Keep response concise and actionable.

---

**Health Status:**
- Build: %s
- Tests: %s
- Git: %s (%d uncommitted)
- Tasks: %d pending

**Brain Summary:** %d recent entries
**Roadmap Summary:** Available for review

Please perform reflection and suggest next steps.`,
		health.BuildStatus,
		health.TestStatus,
		health.GitStatus,
		health.GitUncommitted,
		health.PendingTasks,
		strings.Count(brainContent, "###"))

	// Execute reflection (async, don't block heartbeat)
	go func() {
		resp, err := a.Chat(reflectionPrompt)
		if err != nil {
			a.logger.Error("Deep reflection failed: %v", err)
		} else {
			a.logger.Info("Deep reflection completed: %s", truncateAgentContent(resp, 200))
		}
	}()
}

// GetBrain returns the agent's brain (knowledge base)
func (a *Agent) GetBrain() *brain.Brain {
	return a.brain
}

// GetRoadmap returns the agent's roadmap (strategic planning)
func (a *Agent) GetRoadmap() *roadmap.Roadmap {
	return a.roadmap
}

// GetHealthChecker returns the health checker
func (a *Agent) GetHealthChecker() *health.Checker {
	return a.healthChecker
}

// PerformHealthCheck runs a health check and returns the status
func (a *Agent) PerformHealthCheck() *health.Status {
	return a.healthChecker.Check()
}

// GetToolExecutor returns the tool executor for intelligent retry wrapping
func (a *Agent) GetToolExecutor() *tools.Executor {
	return a.executor
}

// SetToolExecutor sets a new tool executor (for wrapping with retry logic)
func (a *Agent) SetToolExecutor(executor *tools.Executor) {
	a.executor = executor
}

// continueResponse requests a continuation from the LLM when a response is truncated
func (a *Agent) continueResponse(messages []llm.Message, currentContent string) (string, error) {
	var fullContinuation string
	// Limit continuations to prevent infinite loops
	for i := 0; i < 5; i++ {
		// Prepare messages for continuation
		contMessages := make([]llm.Message, len(messages))
		copy(contMessages, messages)

		contMessages = append(contMessages, llm.Message{
			Role:    "assistant",
			Content: currentContent,
		})
		contMessages = append(contMessages, llm.Message{
			Role:    "user",
			Content: "Continue exactly from where you were cut off.",
		})

		resp, err := a.llm.Chat(contMessages, a.config.Temperature, a.config.TopP, a.config.MaxTokens)
		if err != nil {
			return fullContinuation, err
		}

		newContent := a.llm.GetContent(resp)
		fullContinuation += newContent
		currentContent = newContent

		if a.llm.GetFinishReason(resp) != "length" {
			break
		}
		a.logger.Info("Continuation %d also truncated, requesting more...", i+1)
	}
	return fullContinuation, nil
}
