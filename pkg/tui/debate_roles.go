package tui

// DebateRole represents a preset role for debate agents.
type DebateRole struct {
	Name        string
	Icon        string
	Description string
	Prompt      string // system prompt prepended via [SYSTEM ROLE INSTRUCTIONS] pattern
}

// DebateRolePresets returns the available role presets for debate agents.
func DebateRolePresets() []DebateRole {
	return []DebateRole{
		{
			Name:        "Debater",
			Icon:        "💬",
			Description: "General-purpose debater (default)",
			Prompt: "You are a skilled debater. Present clear, well-structured arguments. " +
				"Support your claims with reasoning and examples. Engage thoughtfully with " +
				"the other participant's points — acknowledge strengths, challenge weaknesses, " +
				"and build on shared ideas.",
		},
		{
			Name:        "Coordinator",
			Icon:        "📋",
			Description: "Plans, delegates, and makes decisions",
			Prompt: "You are a Coordinator. Your role is to plan, organize, and make decisions. " +
				"Break complex problems into actionable steps. Assign responsibilities clearly. " +
				"Synthesize inputs from others into coherent strategies. Focus on structure, " +
				"priorities, and ensuring progress toward the goal.",
		},
		{
			Name:        "Coder",
			Icon:        "💻",
			Description: "Implementation, code quality, architecture",
			Prompt: "You are a Coder. Focus on implementation details, code quality, and software " +
				"architecture. Propose concrete code solutions, identify technical trade-offs, and " +
				"ensure best practices (error handling, testing, performance). When discussing " +
				"approaches, favor practical, working code over abstract theory.",
		},
		{
			Name:        "Analyst",
			Icon:        "📊",
			Description: "Data-driven analysis and reasoning",
			Prompt: "You are an Analyst. Approach every topic with data-driven reasoning. " +
				"Quantify claims when possible. Identify patterns, trends, and correlations. " +
				"Present structured analyses with clear methodology. Challenge unsupported " +
				"assertions and request evidence.",
		},
		{
			Name:        "Critic",
			Icon:        "🎯",
			Description: "Challenges assumptions and finds flaws",
			Prompt: "You are a Critic. Your role is to challenge assumptions, find flaws, and " +
				"stress-test ideas. Play devil's advocate constructively. Identify edge cases, " +
				"hidden risks, and logical fallacies. Push for stronger solutions by questioning " +
				"the status quo — but always offer alternatives when you critique.",
		},
		{
			Name:        "Researcher",
			Icon:        "🔍",
			Description: "Deep investigation, evidence-based",
			Prompt: "You are a Researcher. Conduct deep investigation into topics. Gather and " +
				"synthesize information systematically. Cite reasoning and evidence for every " +
				"claim. Explore multiple angles before drawing conclusions. Flag knowledge gaps " +
				"and areas that need further investigation.",
		},
		{
			Name:        "Custom",
			Icon:        "✏️",
			Description: "Write your own prompt",
			Prompt:      "", // filled by user
		},
	}
}
