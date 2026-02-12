package trpcbridge

import (
	"ClosedWheeler/pkg/llm"
	"ClosedWheeler/pkg/logger"
	"ClosedWheeler/pkg/tools"

	trpcagent "trpc.group/trpc-go/trpc-agent-go/agent"
	"trpc.group/trpc-go/trpc-agent-go/agent/chainagent"
	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/runner"
	trpctool "trpc.group/trpc-go/trpc-agent-go/tool"
)

// Bridge is the top-level orchestrator that adapts ClosedWheeler components
// for use with trpc-agent-go agents.
type Bridge struct {
	modelAdapter   *ModelAdapter
	allTools       []trpctool.Tool
	readOnlyTools  []trpctool.Tool
	sessionAdapter *SessionAdapter
	registry       *tools.Registry
	executor       *tools.Executor
	logger         *logger.Logger
}

// NewBridge creates a Bridge from existing ClosedWheeler components.
func NewBridge(
	llmClient *llm.Client,
	registry *tools.Registry,
	executor *tools.Executor,
	log *logger.Logger,
) *Bridge {
	modelName := "default"
	if llmClient != nil && llmClient.ProviderName() != "" {
		modelName = llmClient.ProviderName()
	}

	return &Bridge{
		modelAdapter:   NewModelAdapter(llmClient, modelName, log),
		allTools:       AdaptAllTools(registry, executor),
		readOnlyTools:  AdaptToolsFiltered(registry, executor, IsReadOnlyTool),
		sessionAdapter: NewSessionAdapter(),
		registry:       registry,
		executor:       executor,
		logger:         log,
	}
}

// NewLLMAgent creates a trpc-agent-go LLMAgent with the given instruction and
// optional tool names. Pass nil for toolNames to get no tools.
func (b *Bridge) NewLLMAgent(name, instruction string, toolSet []trpctool.Tool) *llmagent.LLMAgent {
	opts := []llmagent.Option{
		llmagent.WithModel(b.modelAdapter),
		llmagent.WithInstruction(instruction),
		llmagent.WithGenerationConfig(model.GenerationConfig{
			Stream: false, // Pipeline roles don't need streaming; final output is consumed as text.
		}),
	}

	if len(toolSet) > 0 {
		opts = append(opts, llmagent.WithTools(toolSet))
	}

	return llmagent.New(name, opts...)
}

// RolePromptFunc is the signature for the function that returns role-specific
// system prompts. This avoids importing the agent package directly.
type RolePromptFunc func(role string) string

// NewChainPipeline creates a 4-role ChainAgent:
// Planner → Researcher → Executor → Critic.
// rolePrompt is a function that returns the system prompt for a given role name
// (e.g., "planner", "researcher", "executor", "critic").
func (b *Bridge) NewChainPipeline(rolePrompt RolePromptFunc) *chainagent.ChainAgent {
	plannerAgent := b.NewLLMAgent("planner", rolePrompt("planner"), nil)
	researcherAgent := b.NewLLMAgent("researcher", rolePrompt("researcher"), b.readOnlyTools)
	executorAgent := b.NewLLMAgent("executor", rolePrompt("executor"), b.allTools)
	criticAgent := b.NewLLMAgent("critic", rolePrompt("critic"), nil)

	return chainagent.New("pipeline",
		chainagent.WithSubAgents([]trpcagent.Agent{
			plannerAgent,
			researcherAgent,
			executorAgent,
			criticAgent,
		}),
		chainagent.WithChannelBufferSize(256),
	)
}

// NewRunner wraps an agent with session management and returns a Runner.
func (b *Bridge) NewRunner(appName string, ag trpcagent.Agent) runner.Runner {
	return runner.NewRunner(appName, ag,
		runner.WithSessionService(b.sessionAdapter),
	)
}

// ModelAdapter returns the underlying model adapter.
func (b *Bridge) ModelAdapter() *ModelAdapter {
	return b.modelAdapter
}

// SessionAdapter returns the underlying session adapter.
func (b *Bridge) SessionAdapter() *SessionAdapter {
	return b.sessionAdapter
}

// Close releases all bridge resources.
func (b *Bridge) Close() error {
	return b.sessionAdapter.Close()
}
