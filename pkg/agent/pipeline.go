package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
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

// PipelineResult holds the output from one pipeline stage.
type PipelineResult struct {
	Role     AgentRole
	Output   string
	Approved bool // used by Critic
}

// criticJSON is the expected JSON structure from the Critic agent.
type criticJSON struct {
	Approved bool   `json:"approved"`
	Feedback string `json:"feedback"`
	Response string `json:"response"`
}

// MultiAgentPipeline orchestrates Planner → Researcher → Executor → Critic.
type MultiAgentPipeline struct {
	base       *Agent
	enabled    bool
	maxRetries int
	mu         sync.Mutex
	statusCb   func(AgentRole, string) // callback for TUI status updates
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

// Run executes the full Planner → Researcher → Executor → Critic pipeline.
// It retries up to maxRetries times if the Critic rejects the result.
// NOTE: Each pipeline run makes 4 sequential LLM calls (Planner, Researcher,
// Executor, Critic) and may retry up to maxRetries times, so a single user
// message can trigger up to 4 * maxRetries LLM calls.
func (p *MultiAgentPipeline) Run(ctx context.Context, userMessage string) (string, error) {
	p.base.logger.Info("Pipeline started (4 LLM calls per attempt, max %d attempts)", p.maxRetries)
	plannerInput := userMessage
	var lastCriticFeedback string

	for attempt := 0; attempt < p.maxRetries; attempt++ {
		// Build input for Planner (on retry, include Critic feedback)
		if attempt > 0 && lastCriticFeedback != "" {
			plannerInput = fmt.Sprintf("%s\n\n[Critic feedback from previous attempt]: %s\nPlease revise the plan accordingly.", userMessage, lastCriticFeedback)
		}

		// 1. Planner
		p.notify(RolePlanner, "thinking")
		planResult, err := p.runRole(ctx, RolePlanner, plannerInput)
		if err != nil {
			p.notify(RolePlanner, "error")
			return "", fmt.Errorf("planner error: %w", err)
		}
		p.notify(RolePlanner, "done")

		// Small pause between roles to avoid back-to-back API rate limits
		time.Sleep(pipelineRoleDelay)

		// 2. Researcher
		researcherInput := truncatePipelineInput(fmt.Sprintf("USER REQUEST:\n%s\n\nPLAN:\n%s\n\nNow gather the context needed to execute this plan.", userMessage, planResult))
		p.notify(RoleResearcher, "thinking")
		researchResult, err := p.runRole(ctx, RoleResearcher, researcherInput)
		if err != nil {
			p.notify(RoleResearcher, "error")
			return "", fmt.Errorf("researcher error: %w", err)
		}
		p.notify(RoleResearcher, "done")

		time.Sleep(pipelineRoleDelay)

		// 3. Executor
		executorInput := truncatePipelineInput(fmt.Sprintf("USER REQUEST:\n%s\n\nPLAN:\n%s\n\nRESEARCH CONTEXT:\n%s\n\nNow execute the plan.", userMessage, planResult, researchResult))
		p.notify(RoleExecutor, "thinking")
		execResult, err := p.runRole(ctx, RoleExecutor, executorInput)
		if err != nil {
			p.notify(RoleExecutor, "error")
			return "", fmt.Errorf("executor error: %w", err)
		}
		p.notify(RoleExecutor, "done")

		time.Sleep(pipelineRoleDelay)

		// 4. Critic
		criticInput := truncatePipelineInput(fmt.Sprintf("ORIGINAL USER REQUEST:\n%s\n\nPLAN:\n%s\n\nEXECUTION RESULT:\n%s\n\nReview and respond with JSON.", userMessage, planResult, execResult))
		p.notify(RoleCritic, "thinking")
		criticOutput, err := p.runRole(ctx, RoleCritic, criticInput)
		if err != nil {
			p.notify(RoleCritic, "error")
			// Fallback: treat execution result as approved
			p.base.logger.Error("Critic error (using executor result): %v", err)
			return execResult, nil
		}
		p.notify(RoleCritic, "done")

		// Parse Critic JSON
		parsed, parseErr := parseCriticJSON(criticOutput)
		if parseErr != nil {
			// Not valid JSON — treat as approved with raw output
			p.base.logger.Info("Critic returned non-JSON, treating as approved")
			p.savePipelineInsight(userMessage, criticOutput)
			return criticOutput, nil
		}

		if parsed.Approved {
			p.savePipelineInsight(userMessage, parsed.Feedback)
			return parsed.Response, nil
		}

		// Critic rejected — record feedback for next iteration
		lastCriticFeedback = parsed.Feedback
		p.base.logger.Info("Critic rejected (attempt %d/%d): %s", attempt+1, p.maxRetries, parsed.Feedback)
	}

	// Exhausted retries — return last executor result with a note
	p.base.logger.Info("Pipeline exhausted retries, returning best effort result")
	return fmt.Sprintf("[Pipeline: max retries reached]\n\n%s", lastCriticFeedback), nil
}

// runRole clones the base agent for a given role and calls Chat with the given input.
func (p *MultiAgentPipeline) runRole(ctx context.Context, role AgentRole, input string) (string, error) {
	clone := p.base.CloneForDebate(string(role))

	// Override the clone's system prompt by prepending role instructions to the message.
	// We use a special prefix that the clone will see as part of the conversation.
	rolePrompt := roleSystemPrompt(role)
	if rolePrompt != "" {
		input = fmt.Sprintf("[SYSTEM ROLE INSTRUCTIONS]\n%s\n[END ROLE INSTRUCTIONS]\n\n%s", rolePrompt, input)
	}

	// Per-role timeout: 10 minutes is generous for deep research/analysis tasks.
	// The outer ctx (from the user's request) still cancels this immediately if
	// StopCurrentRequest() is called.
	roleCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// Run in the clone's context — we need to respect cancellation
	type result struct {
		out string
		err error
	}
	ch := make(chan result, 1)

	go func() {
		out, err := clone.Chat(input)
		ch <- result{out, err}
	}()

	select {
	case <-roleCtx.Done():
		return "", fmt.Errorf("role %s timed out", role)
	case r := <-ch:
		return r.out, r.err
	}
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
