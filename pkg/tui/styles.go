package tui

// styles.go — shared theme colors and lipgloss styles for the entire tui package.

import "github.com/charmbracelet/lipgloss"

// Theme colors
var (
	// Primary colors
	primaryColor   = lipgloss.Color("#7C3AED") // Purple
	secondaryColor = lipgloss.Color("#06B6D4") // Cyan
	accentColor    = lipgloss.Color("#F59E0B") // Amber
	successColor   = lipgloss.Color("#10B981") // Green
	errorColor     = lipgloss.Color("#EF4444") // Red
	mutedColor     = lipgloss.Color("#6B7280") // Gray

	// Background
	bgDark   = lipgloss.Color("#1F2937")
	bgDarker = lipgloss.Color("#111827")

	// Text
	textPrimary   = lipgloss.Color("#F9FAFB")
	textSecondary = lipgloss.Color("#9CA3AF")
)

// Styles
var (
	// Header
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textPrimary).
			Background(primaryColor).
			Padding(0, 2).
			MarginBottom(1)

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
			Foreground(textSecondary).
			Background(bgDark).
			Padding(0, 1)

	statusItemStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	// Messages
	userBubbleStyle = lipgloss.NewStyle().
			Foreground(textPrimary).
			Background(lipgloss.Color("#374151")).
			Padding(0, 1).
			MarginBottom(1)

	userLabelStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	assistantLabelStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)

	assistantTextStyle = lipgloss.NewStyle().
				Foreground(textPrimary)

	systemMsgStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			MarginBottom(1)

	errorMsgStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	successMsgStyle = lipgloss.NewStyle().
			Foreground(successColor)

	// Code blocks
	codeBlockStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A5F3FC")).
			Background(bgDarker).
			Padding(0, 1)

	// Input area
	inputBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(0, 1)

	// Help
	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Divider
	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#374151"))

	// Badges
	badgeStyle = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1).
			MarginRight(1)

	idleBadgeStyle = badgeStyle.Copy().
			Background(mutedColor).
			Foreground(textPrimary)

	thinkingBadgeStyle = badgeStyle.Copy().
				Background(accentColor).
				Foreground(bgDarker)

	workingBadgeStyle = badgeStyle.Copy().
				Background(successColor).
				Foreground(bgDarker)

	memStatsStyle = lipgloss.NewStyle().
			Foreground(textSecondary).
			Faint(true)

	// Thinking/Reasoning styles
	thinkingStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			Faint(true)

	thinkingHeaderStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true).
				MarginBottom(1)
)
