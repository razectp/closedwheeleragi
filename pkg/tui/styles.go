package tui

import "github.com/charmbracelet/lipgloss"

// Theme colors - "Electric Nebula" Palette (vivid true-color)
var (
	// Primary colors — vivid and saturated
	PrimaryColor   = lipgloss.Color("#A855F7") // Vivid purple
	SecondaryColor = lipgloss.Color("#22D3EE") // Electric cyan
	AccentColor    = lipgloss.Color("#FBBF24") // Bright gold amber
	SuccessColor   = lipgloss.Color("#34D399") // Bright emerald
	ErrorColor     = lipgloss.Color("#FB7185") // Coral rose
	MutedColor     = lipgloss.Color("#94A3B8") // Silver slate
	HeadingColor   = lipgloss.Color("#C084FC") // Bright violet

	// Chat-specific vivid accents
	UserColor = lipgloss.Color("#38BDF8") // Bright sky blue — user bubbles
	GoldColor = lipgloss.Color("#FBBF24") // Amber gold — stats & highlights
	CodeColor = lipgloss.Color("#4ADE80") // Acid green — code blocks
	HotPink   = lipgloss.Color("#F472B6") // Hot pink — decorative

	// Backgrounds — deep cosmic dark
	BgBase   = lipgloss.Color("#0C0A1D") // Deep cosmic purple
	BgDark   = lipgloss.Color("#1A1333") // Dark purple tint
	BgDarker = lipgloss.Color("#050311") // Abyss

	// Text
	TextPrimary   = lipgloss.Color("#F1F5F9") // Near white
	TextSecondary = lipgloss.Color("#CBD5E1") // Silver
	TextMuted     = lipgloss.Color("#64748B") // Muted slate
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

	// Divider - Visible indigo glow
	DividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3730A3"))

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

	// Help Menu Overlay Styles
	HelpBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryColor).
			Padding(1, 2).
			Margin(1, 1)

	HelpTitleStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true).
			MarginBottom(1)

	HelpCategoryActiveStyle = lipgloss.NewStyle().
				Foreground(SuccessColor).
				Bold(true)

	HelpCategoryInactiveStyle = lipgloss.NewStyle().
					Foreground(TextSecondary)

	HelpCommandSelectedStyle = lipgloss.NewStyle().
				Foreground(SuccessColor).
				Bold(true)

	HelpCommandNormalStyle = lipgloss.NewStyle().
				Foreground(TextPrimary)

	HelpDetailLabelStyle = lipgloss.NewStyle().
				Foreground(SecondaryColor).
				Bold(true)

	HelpDetailValueStyle = lipgloss.NewStyle().
				Foreground(TextPrimary)

	HelpFooterStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			Faint(true)

	HelpSearchStyle = lipgloss.NewStyle().
			Foreground(AccentColor).
			Bold(true)

	// Setup Wizard Bubbletea Styles
	WizardBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryColor).
			Padding(1, 2).
			Margin(1, 1)

	WizardTitleStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true)

	WizardStepTitleStyle = lipgloss.NewStyle().
				Foreground(SecondaryColor).
				Bold(true).
				MarginBottom(1)

	WizardProgressBarFillStyle = lipgloss.NewStyle().
					Foreground(PrimaryColor)

	WizardProgressBarEmptyStyle = lipgloss.NewStyle().
					Foreground(MutedColor)

	WizardSelectedStyle = lipgloss.NewStyle().
				Foreground(SuccessColor).
				Bold(true)

	WizardUnselectedStyle = lipgloss.NewStyle().
				Foreground(TextPrimary)

	WizardDescStyle = lipgloss.NewStyle().
			Foreground(TextSecondary).
			MarginLeft(4)

	WizardFooterStyle = lipgloss.NewStyle().
				Foreground(MutedColor).
				Faint(true)

	// Panel Overlay Styles (read-only info panels)
	PanelBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(SecondaryColor).
			Padding(1, 2).
			Margin(1, 1)

	PanelTitleStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Bold(true)

	PanelScrollStyle = lipgloss.NewStyle().
				Foreground(MutedColor)

	PanelFooterStyle = lipgloss.NewStyle().
				Foreground(MutedColor).
				Faint(true)

	// Settings Overlay Styles (interactive toggle menu)
	SettingsBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(AccentColor).
				Padding(1, 2).
				Margin(1, 1)

	SettingsTitleStyle = lipgloss.NewStyle().
				Foreground(AccentColor).
				Bold(true)

	SettingsSelectedStyle = lipgloss.NewStyle().
				Foreground(SuccessColor).
				Bold(true)

	SettingsNormalStyle = lipgloss.NewStyle().
				Foreground(TextPrimary)

	SettingsOnStyle = lipgloss.NewStyle().
			Foreground(SuccessColor)

	SettingsOffStyle = lipgloss.NewStyle().
			Foreground(ErrorColor)

	SettingsValueStyle = lipgloss.NewStyle().
				Foreground(SecondaryColor)

	// Debate Viewer Styles
	DebateAgentAStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#3B82F6")).
				Bold(true)

	DebateAgentBStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#22C55E")).
				Bold(true)

	DebateSystemStyle = lipgloss.NewStyle().
				Foreground(MutedColor).
				Italic(true)

	DebateActiveStyle = lipgloss.NewStyle().
				Foreground(SuccessColor).
				Bold(true)

	DebateCompleteStyle = lipgloss.NewStyle().
				Foreground(MutedColor)

	// Debate Viewer Overlay Styles
	DebateViewBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(PrimaryColor).
				Padding(1, 2).
				Margin(1, 1)

	DebateViewTitleStyle = lipgloss.NewStyle().
				Foreground(PrimaryColor).
				Bold(true)

	DebateViewTurnStyle = lipgloss.NewStyle().
				Foreground(TextSecondary).
				Italic(true)

	DebateViewDividerStyle = lipgloss.NewStyle().
				Foreground(TextMuted)

	DebateViewScrollStyle = lipgloss.NewStyle().
				Foreground(MutedColor)

	DebateViewFooterStyle = lipgloss.NewStyle().
				Foreground(MutedColor).
				Faint(true)

	DebateViewThinkingStyle = lipgloss.NewStyle().
				Foreground(AccentColor).
				Italic(true)

	// Content rendering styles (used in tui.go renderContent / updateViewport)
	ContentHeadingStyle = lipgloss.NewStyle().
				Foreground(HeadingColor).
				Bold(true)

	StreamCursorStyle = lipgloss.NewStyle().
				Foreground(PrimaryColor)

	TimestampStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			Faint(true)

	// Tool status styles (renderActiveTools)
	ToolRunningStyle = lipgloss.NewStyle().
				Foreground(AccentColor)

	ToolSuccessStyle = lipgloss.NewStyle().
				Foreground(SuccessColor)

	ToolFailedStyle = lipgloss.NewStyle().
			Foreground(ErrorColor)

	ToolPendingStyle = lipgloss.NewStyle().
				Foreground(MutedColor)

	ToolDurationStyle = lipgloss.NewStyle().
				Foreground(MutedColor)

	ToolsSectionStyle = lipgloss.NewStyle().
				Foreground(TextSecondary).
				Background(BgDarker).
				Padding(0, 1)

	// Processing area style (renderProcessingArea)
	ProcessingStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(AccentColor).
			Padding(0, 1).
			MarginLeft(3).
			Height(2)

	// Pipeline role status styles (renderPipelineBar)
	PipelineRoleActiveStyle = lipgloss.NewStyle().
				Foreground(AccentColor).
				Bold(true)

	PipelineRoleDoneStyle = lipgloss.NewStyle().
				Foreground(SuccessColor)

	PipelineRoleErrorStyle = lipgloss.NewStyle().
				Foreground(ErrorColor)

	PipelineRoleWaitingStyle = lipgloss.NewStyle().
				Foreground(TextMuted)

	PipelineLabelStyle = lipgloss.NewStyle().
				Foreground(TextSecondary).
				Bold(true)

	PipelineSeparatorStyle = lipgloss.NewStyle().
				Foreground(MutedColor)

	PipelineErrorStyle = lipgloss.NewStyle().
				Foreground(ErrorColor).
				Bold(true)

	// Toggle display styles (commands.go toggle helpers)
	ToggleOnStyle = lipgloss.NewStyle().
			Foreground(SuccessColor)

	ToggleOffStyle = lipgloss.NewStyle().
			Foreground(ErrorColor)

	// Picker styles (picker_types.go)
	PickerTitleStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true).
			MarginBottom(1)

	PickerSubtitleStyle = lipgloss.NewStyle().
				Foreground(TextSecondary).
				MarginBottom(1)

	PickerSelectedStyle = lipgloss.NewStyle().
				Foreground(SuccessColor).
				Bold(true)

	PickerUnselectedStyle = lipgloss.NewStyle().
				Foreground(TextPrimary)

	PickerHintStyle = lipgloss.NewStyle().
			Foreground(TextMuted).
			Faint(true)

	PickerCurrentStyle = lipgloss.NewStyle().
				Foreground(AccentColor).
				Bold(true)

	PickerBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryColor).
			Padding(1, 2).
			Margin(1, 1)

	PickerFooterStyle = lipgloss.NewStyle().
				Foreground(TextMuted).
				MarginTop(1)

	// ── Chat Bubble Styles ──────────────────────────────────────

	// User messages — right-aligned rounded bubble (bright sky blue)
	UserBubbleStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(UserColor).
			Padding(0, 2).
			Foreground(TextPrimary)

	UserBadgeStyle = lipgloss.NewStyle().
			Background(UserColor).
			Foreground(BgDarker).
			Bold(true).
			Padding(0, 1)

	// Assistant messages — vivid purple left accent bar with subtle dark background
	AssistantBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(PrimaryColor).
				PaddingLeft(1).
				Background(BgDark)

	AssistantBadgeStyle = lipgloss.NewStyle().
				Background(PrimaryColor).
				Foreground(BgDarker).
				Bold(true).
				Padding(0, 1)

	// Narrow-layout assistant background tint (no border, just indent + bg)
	NarrowAssistantStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Background(BgDark)

	// Bordered code blocks — acid green accent
	CodeBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CodeColor).
			Background(BgDarker).
			Foreground(lipgloss.Color("#E2E8F0")).
			Padding(0, 1)

	CodeLangLabelStyle = lipgloss.NewStyle().
				Background(CodeColor).
				Foreground(BgDarker).
				Bold(true).
				Padding(0, 1)

	// Thinking accordion box — gold amber accent
	ThinkingBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(AccentColor).
				Foreground(TextSecondary).
				Italic(true).
				Padding(0, 1)

	ThinkingBoxHeaderStyle = lipgloss.NewStyle().
				Foreground(AccentColor).
				Bold(true)

	// Centered system messages
	SystemCenterStyle = lipgloss.NewStyle().
				Foreground(MutedColor).
				Italic(true).
				Align(lipgloss.Center)

	// Error box — coral rose thick left border
	ErrorBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder(), false, false, false, true).
			BorderForeground(ErrorColor).
			PaddingLeft(1)

	ErrorBoxHeaderStyle = lipgloss.NewStyle().
				Foreground(ErrorColor).
				Bold(true)

	// Stats pill — gold accent on dark
	StatsPillStyle = lipgloss.NewStyle().
			Foreground(GoldColor).
			Background(BgDark).
			Padding(0, 1)
)
