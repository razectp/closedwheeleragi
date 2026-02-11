package tui

import "github.com/charmbracelet/lipgloss"

// Theme colors - Professional "Indigo & Slate" Palette
var (
	// Primary colors
	PrimaryColor   = lipgloss.Color("#6366F1") // Indigo 500
	SecondaryColor = lipgloss.Color("#0EA5E9") // Sky 500
	AccentColor    = lipgloss.Color("#F59E0B") // Amber 500
	SuccessColor   = lipgloss.Color("#10B981") // Emerald 500
	ErrorColor     = lipgloss.Color("#EF4444") // Red 500
	MutedColor     = lipgloss.Color("#64748B") // Slate 500

	// Backgrounds
	BgBase   = lipgloss.Color("#0F172A") // Slate 900
	BgDark   = lipgloss.Color("#1E293B") // Slate 800
	BgDarker = lipgloss.Color("#020617") // Slate 950

	// Text
	TextPrimary   = lipgloss.Color("#F8FAFC") // Slate 50
	TextSecondary = lipgloss.Color("#94A3B8") // Slate 400
	TextMuted     = lipgloss.Color("#475569") // Slate 600
)

// Shared Styles
var (
	// Header - Sleek Modern Look
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(TextPrimary).
			Padding(0, 1).
			MarginBottom(1)

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryColor).
			Padding(0, 1)

	// Status bar - Compact & Informative
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(TextSecondary).
			Background(BgDark).
			Padding(0, 1)

	StatusItemStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Bold(true)

	// Chat Messages
	UserLabelStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Bold(true)

	AssistantLabelStyle = lipgloss.NewStyle().
				Foreground(PrimaryColor).
				Bold(true)

	AssistantTextStyle = lipgloss.NewStyle().
				Foreground(TextPrimary)

	SystemMsgStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			Italic(true)

	ErrorMsgStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			Bold(true)

	SuccessMsgStyle = lipgloss.NewStyle().
			Foreground(SuccessColor)

	// Setup Wizard Styles
	SetupHeaderStyle = lipgloss.NewStyle().
				Foreground(PrimaryColor).
				Bold(true).
				Underline(true)

	SetupPromptStyle = lipgloss.NewStyle().
				Foreground(SecondaryColor)

	SetupErrorStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			Bold(true)

	SetupSuccessStyle = lipgloss.NewStyle().
				Foreground(SuccessColor).
				Bold(true)

	SetupInfoStyle = lipgloss.NewStyle().
			Foreground(TextSecondary)

	// Code blocks - Refined contrasting background
	CodeBlockStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E2E8F0")).
			Background(BgDarker).
			Padding(0, 1)

	// Input area - Focus indigo border
	InputBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(PrimaryColor).
				Padding(0, 1)

	// Help - Subdued and minimal
	HelpStyle = lipgloss.NewStyle().
			Foreground(TextMuted)

	// Divider - Very subtle
	DividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1E293B"))

	// Badges - Clean & Flat
	BadgeStyle = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1).
			MarginRight(1)

	IdleBadgeStyle = BadgeStyle.Copy().
			Background(TextMuted).
			Foreground(TextPrimary)

	ThinkingBadgeStyle = BadgeStyle.Copy().
				Background(AccentColor).
				Foreground(BgDarker)

	WorkingBadgeStyle = BadgeStyle.Copy().
				Background(PrimaryColor).
				Foreground(TextPrimary)

	MemStatsStyle = lipgloss.NewStyle().
			Foreground(TextSecondary).
			Faint(true)

	// Thinking/Reasoning styles
	ThinkingStyle = lipgloss.NewStyle().
			Foreground(TextSecondary).
			Italic(true)

	ThinkingHeaderStyle = lipgloss.NewStyle().
				Foreground(AccentColor).
				Bold(true)
)
