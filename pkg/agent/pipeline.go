package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"ClosedWheeler/pkg/trpcbridge"

	"trpc.group/trpc-go/trpc-agent-go/agent/chainagent"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/runner"
)

// pipelineRoleDelay is the minimum wait between sequential role API calls to
// avoid hitting rate limits when multiple agents fire requests back-to-back.
const pipelineRoleDelay = 1500 * time.Millisecond

// maxRoleInputLen is the max characters allowed in a role input payload.
// Inputs exceeding this are tail-truncated to avoid context-length errors.
const maxRoleInputLen = 12000

// AgentRole identifies a role in the multi-agent pipeline.
type AgentRole string

const (
	RolePlanner    AgentRole = "planner"
	RoleResearcher AgentRole = "researcher"
	RoleExecutor   AgentRole = "executor"
	RoleCritic     AgentRole = "critic"
)

// criticJSON is the expected JSON structure from the Critic agent.
type criticJSON struct {
	Approved bool   `json:"approved"`
	Feedback string `json:"feedback"`
	Response string `json:"response"`
}

// MultiAgentPipeline orchestrates Planner → Researcher → Executor → Critic
// using trpc-agent-go's ChainAgent for sequential multi-agent execution.
type MultiAgentPipeline struct {
	base       *Agent
	enabled    bool
	maxRetries int
	mu         sync.Mutex
	statusCb   func(AgentRole, string) // callback for TUI status updates

	bridge *trpcbridge.Bridge           // lazy-initialized trpc-agent-go bridge
	chain  *chainagent.ChainAgent       // the 4-role sequential chain
	runner runner.Runner                // runner with session management
}

// NewMultiAgentPipeline creates a new pipeline (disabled by default).
func NewMultiAgentPipeline(base *Agent) *MultiAgentPipeline {
	return &MultiAgentPipeline{
		base:       base,
		enabled:    false,
		maxRetries: 2,
	}
}

// Enable activates or deactivates the pipeline.
func (p *MultiAgentPipeline) Enable(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = enabled
}

// IsEnabled returns whether the pipeline is active.
func (p *MultiAgentPipeline) IsEnabled() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.enabled
}

// SetStatusCallback sets a callback invoked when a role changes status.
// Status values: "thinking", "done", "error".
func (p *MultiAgentPipeline) SetStatusCallback(cb func(AgentRole, string)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.statusCb = cb
}

func (p *MultiAgentPipeline) notify(role AgentRole, status string) {
	p.mu.Lock()
	cb := p.statusCb
	p.mu.Unlock()
	if cb != nil {
		cb(role, status)
	}
}

// initBridge lazily initializes the trpc-agent-go bridge, chain, and runner.
func (p *MultiAgentPipeline) initBridge() {
	if p.bridge != nil {
		return
	}

	p.bridge = trpcbridge.NewBridge(
		p.base.llm,
		p.base.tools,
		p.base.executor,
		p.base.logger,
	)

	// Build the role prompt function that adapts our roleSystemPrompt.
	rolePrompt := func(role string) string {
		return roleSystemPrompt(AgentRole(role))
	}

	p.chain = p.bridge.NewChainPipeline(rolePrompt)
	p.runner = p.bridge.NewRunner("pipeline", p.chain)
}

// Run executes the full Planner → Researcher → Executor → Critic pipeline.
// It retries up to maxRetries times if the Critic rejects the result.
// NOTE: Each pipeline run makes 4 sequential LLM calls (Planner, Researcher,
// Executor, Critic) and may retry up to maxRetries times, so a single user
// message can trigger up to 4 * maxRetries LLM calls.
func (p *MultiAgentPipeline) Run(ctx context.Context, userMessage string) (string, error) {
	p.base.logger.Info("Pipeline started (ChainAgent, max %d attempts)", p.maxRetries)

	// Lazy-init the bridge and chain on first use.
	p.initBridge()

	var lastCriticFeedback string

	for attempt := 0; attempt < p.maxRetries; attempt++ {
		input := userMessage
		if attempt > 0 && lastCriticFeedback != "" {
			input = fmt.Sprintf("%s\n\n[Critic feedback from previous attempt]: %s\nPlease revise the plan accordingly.", userMessage, lastCriticFeedback)
		}

		input = truncatePipelineInput(input)

		// Notify all roles starting.
		p.notify(RolePlanner, "thinking")

		// Create event bridge with TUI callbacks.
		// Wrap the AgentRole-based statusCb into a string-based callback
		// to avoid an import cycle (trpcbridge cannot import agent).
		var roleCb trpcbridge.PipelineRoleCallback
		if p.statusCb != nil {
			roleCb = func(role string, status string) {
				p.statusCb(AgentRole(role), status)
			}
		}
		eb := trpcbridge.NewEventBridge(
			p.base.streamCallback,
			p.base.toolStartCb,
			p.base.toolCompleteCb,
			p.base.toolErrorCb,
			func(s string) {
				if p.base.statusCallback != nil {
					p.base.statusCallback(s)
				}
			},
			roleCb,
			p.base.logger,
		)

		// Run the chain via runner (manages sessions).
		sessionID := fmt.Sprintf("pipeline-%d-%d", time.Now().UnixMilli(), attempt)
		events, err := p.runner.Run(ctx, "agent", sessionID,
			model.NewUserMessage(input),
		)
		if err != nil {
			p.notify(RolePlanner, "error")
			return "", fmt.Errorf("pipeline chain error: %w", err)
		}

		// Consume events — fires TUI callbacks, returns final text.
		result, err := eb.ConsumeEvents(ctx, events)
		if err != nil {
			p.base.logger.Error("Pipeline event consumption error: %v", err)
			// If we got partial text, return it.
			if result != "" {
				return result, nil
			}
			return "", fmt.Errorf("pipeline error: %w", err)
		}

		// Parse critic output (the last agent in the chain).
		parsed, parseErr := parseCriticJSON(result)
		if parseErr != nil {
			// Not valid JSON — treat as approved with raw output.
			p.base.logger.Info("Critic returned non-JSON, treating as approved")
			p.savePipelineInsight(userMessage, result)
			return result, nil
		}

		if parsed.Approved {
			p.savePipelineInsight(userMessage, parsed.Feedback)
			return parsed.Response, nil
		}

		// Critic rejected — record feedback for next iteration.
		lastCriticFeedback = parsed.Feedback
		p.base.logger.Info("Critic rejected (attempt %d/%d): %s", attempt+1, p.maxRetries, parsed.Feedback)

		// Small pause before retry to avoid rate limits.
		time.Sleep(pipelineRoleDelay)
	}

	// Exhausted retries — return last feedback.
	p.base.logger.Info("Pipeline exhausted retries, returning best effort result")
	return fmt.Sprintf("[Pipeline: max retries reached]\n\n%s", lastCriticFeedback), nil
}

// parseCriticJSON attempts to extract the Critic's JSON from its response.
func parseCriticJSON(output string) (*criticJSON, error) {
	// Try to find a JSON block
	start := strings.Index(output, "{")
	end := strings.LastIndex(output, "}")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("no JSON found in critic output")
	}

	jsonStr := output[start : end+1]
	var parsed criticJSON
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}
	return &parsed, nil
}

// truncatePipelineInput ensures role inputs don't exceed the context limit.
// When content is too long we keep the beginning (user request + plan) and
// tail-trim the potentially-large research/execution sections.
func truncatePipelineInput(input string) string {
	if len(input) <= maxRoleInputLen {
		return input
	}
	// Keep first portion + indicator + last 2000 chars
	const tail = 2000
	head := maxRoleInputLen - tail - 40
	if head < 1000 {
		head = 1000
	}
	return input[:head] + "\n\n[...content truncated to fit context...]\n\n" + input[len(input)-tail:]
}

// savePipelineInsight persists a brain insight from the pipeline run.
func (p *MultiAgentPipeline) savePipelineInsight(userMessage, feedback string) {
	if p.base.brain == nil || feedback == "" {
		return
	}
	title := "Pipeline result"
	if len(userMessage) > 60 {
		title = "Pipeline: " + userMessage[:57] + "..."
	} else if userMessage != "" {
		title = "Pipeline: " + userMessage
	}
	_ = p.base.brain.AddInsight(title, feedback, []string{"pipeline", "auto"})
}

