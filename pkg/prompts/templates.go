// Package prompts provides system prompt templates for different contexts.
package prompts

import (
	"fmt"
	"strings"
	"time"
)

// Context represents the current interaction context
type Context string

const (
	ContextAnalysis    Context = "analysis"
	ContextGeneration  Context = "generation"
	ContextRefactoring Context = "refactoring"
	ContextDebugging   Context = "debugging"
	ContextPlanning    Context = "planning"
	ContextGeneral     Context = "general"
)

// Template represents a system prompt template
type Template struct {
	Name         string
	Context      Context
	BasePrompt   string
	Constraints  []string
	OutputFormat string
}

// Templates holds all available prompt templates
var Templates = map[Context]*Template{
	ContextAnalysis: {
		Name:    "Code Analysis",
		Context: ContextAnalysis,
		BasePrompt: `You are an expert code analyst. Your job is to analyze code for:
- Bugs and potential issues
- Security vulnerabilities
- Performance problems
- Code quality and maintainability
- Best practices violations

Be specific and actionable. Reference exact line numbers when possible.`,
		Constraints: []string{
			"Focus only on real issues, not style preferences",
			"Prioritize security and bugs over style",
			"Provide severity ratings: HIGH, MEDIUM, LOW",
		},
		OutputFormat: `Format your response as:
## Issues Found

### [SEVERITY] Issue Title
- **Location**: file:line
- **Problem**: Description
- **Fix**: Suggested solution

### Summary
- Total issues: N
- Score: X/100`,
	},

	ContextGeneration: {
		Name:    "Code Generation",
		Context: ContextGeneration,
		BasePrompt: `You are an expert programmer. Generate clean, well-documented code that:
- Follows language idioms and best practices
- Includes proper error handling
- Is testable and maintainable
- Has clear variable and function names`,
		Constraints: []string{
			"Include necessary imports",
			"Add comments for complex logic",
			"Handle edge cases",
			"Follow the existing code style in the project",
		},
		OutputFormat: `Provide the complete code without explanations unless asked.
Use proper code blocks with language specification.`,
	},

	ContextRefactoring: {
		Name:    "Code Refactoring",
		Context: ContextRefactoring,
		BasePrompt: `You are an expert at code refactoring. Your goal is to improve code structure while:
- Preserving exact behavior (no functional changes)
- Improving readability
- Reducing complexity
- Following SOLID principles`,
		Constraints: []string{
			"Never change functionality",
			"Keep changes minimal but impactful",
			"Explain each refactoring step",
			"Preserve all existing tests",
		},
		OutputFormat: `For each change:
1. BEFORE: original code
2. AFTER: refactored code
3. WHY: reason for change`,
	},

	ContextDebugging: {
		Name:    "Debugging",
		Context: ContextDebugging,
		BasePrompt: `You are an expert debugger. Analyze the problem systematically:
- Understand the expected vs actual behavior
- Identify root cause, not just symptoms
- Consider edge cases and race conditions
- Check for common pitfalls in the language`,
		Constraints: []string{
			"Ask for error messages and stack traces",
			"Consider the execution environment",
			"Check dependencies and versions",
			"Verify assumptions about input/output",
		},
		OutputFormat: `## Analysis
1. **Symptom**: What's happening
2. **Root Cause**: Why it's happening
3. **Solution**: How to fix it
4. **Prevention**: How to avoid it in future`,
	},

	ContextPlanning: {
		Name:    "Architecture Planning",
		Context: ContextPlanning,
		BasePrompt: `You are a software architect. Help plan and design solutions that:
- Are scalable and maintainable
- Follow established patterns
- Consider trade-offs explicitly
- Plan for testing and deployment`,
		Constraints: []string{
			"Consider existing codebase constraints",
			"Think about backward compatibility",
			"Plan for incremental implementation",
			"Document assumptions and risks",
		},
		OutputFormat: `## Proposed Design
### Overview
Brief description

### Components
- Component A: responsibility
- Component B: responsibility

### Trade-offs
| Approach | Pros | Cons |
|----------|------|------|

### Implementation Steps
1. Step one
2. Step two`,
	},

	ContextGeneral: {
		Name:    "General Assistant",
		Context: ContextGeneral,
		BasePrompt: `You are an expert programming assistant. Help the user with their coding tasks.
Be concise, accurate, and practical.`,
		Constraints: []string{
			"Be direct and helpful",
			"Provide examples when useful",
			"Admit when unsure",
		},
		OutputFormat: "",
	},
}

// Builder constructs prompts with context
type Builder struct {
	template           *Template
	projectInfo        string
	relevantCode       string
	history            string
	customInstructions string
	toolsSummary       string
}

// NewBuilder creates a new prompt builder for a context
func NewBuilder(ctx Context) *Builder {
	template, exists := Templates[ctx]
	if !exists {
		template = Templates[ContextGeneral]
	}

	return &Builder{
		template: template,
	}
}

// WithProjectInfo adds project context
func (b *Builder) WithProjectInfo(info string) *Builder {
	b.projectInfo = info
	return b
}

// WithRelevantCode adds relevant code snippets
func (b *Builder) WithRelevantCode(code string) *Builder {
	b.relevantCode = code
	return b
}

// WithHistory adds conversation history summary
func (b *Builder) WithHistory(history string) *Builder {
	b.history = history
	return b
}

// WithCustomInstructions adds custom instructions
func (b *Builder) WithCustomInstructions(instructions string) *Builder {
	b.customInstructions = instructions
	return b
}

// WithToolsSummary adds a summary of available tools
func (b *Builder) WithToolsSummary(summary string) *Builder {
	b.toolsSummary = summary
	return b
}

// Build constructs the final system prompt
func (b *Builder) Build() string {
	var sb strings.Builder

	// System Information (always first)
	now := time.Now()
	sb.WriteString("## System Information\n")
	sb.WriteString(fmt.Sprintf("- **Current Date**: %s\n", now.Format("Monday, January 2, 2006")))
	sb.WriteString(fmt.Sprintf("- **Current Time**: %s\n", now.Format("15:04:05 MST")))
	sb.WriteString(fmt.Sprintf("- **Timestamp**: %s\n", now.Format(time.RFC3339)))
	sb.WriteString("\n")

	// Base prompt
	sb.WriteString(b.template.BasePrompt)
	sb.WriteString("\n\n")

	// Available Tools
	if b.toolsSummary != "" {
		sb.WriteString("## Available Tools\n")
		sb.WriteString(b.toolsSummary)
		sb.WriteString("\n\n")
	}

	// Constraints
	if len(b.template.Constraints) > 0 {
		sb.WriteString("## Guidelines\n")
		for _, c := range b.template.Constraints {
			sb.WriteString(fmt.Sprintf("- %s\n", c))
		}
		sb.WriteString("\n")
	}

	// Project info
	if b.projectInfo != "" {
		sb.WriteString("## Project Context\n")
		sb.WriteString(b.projectInfo)
		sb.WriteString("\n\n")
	}

	// History summary
	if b.history != "" {
		sb.WriteString("## Previous Context\n")
		sb.WriteString(b.history)
		sb.WriteString("\n\n")
	}

	// Relevant code
	if b.relevantCode != "" {
		sb.WriteString("## Relevant Code\n")
		sb.WriteString(b.relevantCode)
		sb.WriteString("\n\n")
	}

	// Custom instructions
	if b.customInstructions != "" {
		sb.WriteString("## Additional Instructions\n")
		sb.WriteString(b.customInstructions)
		sb.WriteString("\n\n")
	}

	// Output format
	if b.template.OutputFormat != "" {
		sb.WriteString("## Expected Output Format\n")
		sb.WriteString(b.template.OutputFormat)
		sb.WriteString("\n")
	}

	return sb.String()
}

// QuickPrompt generates a simple system prompt for a context
func QuickPrompt(ctx Context) string {
	return NewBuilder(ctx).Build()
}

// DetectContext tries to detect the appropriate context from user input
func DetectContext(input string) Context {
	input = strings.ToLower(input)

	if strings.Contains(input, "bug") || strings.Contains(input, "error") ||
		strings.Contains(input, "fix") || strings.Contains(input, "debug") ||
		strings.Contains(input, "not working") {
		return ContextDebugging
	}

	if strings.Contains(input, "analyze") || strings.Contains(input, "review") ||
		strings.Contains(input, "check") || strings.Contains(input, "audit") {
		return ContextAnalysis
	}

	if strings.Contains(input, "refactor") || strings.Contains(input, "clean") ||
		strings.Contains(input, "improve") || strings.Contains(input, "simplify") {
		return ContextRefactoring
	}

	if strings.Contains(input, "create") || strings.Contains(input, "generate") ||
		strings.Contains(input, "write") || strings.Contains(input, "implement") ||
		strings.Contains(input, "add") {
		return ContextGeneration
	}

	if strings.Contains(input, "plan") || strings.Contains(input, "design") ||
		strings.Contains(input, "architect") || strings.Contains(input, "structure") {
		return ContextPlanning
	}

	return ContextGeneral
}
