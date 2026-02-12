package tui

import (
	"fmt"
	"strings"

	"ClosedWheeler/pkg/llm"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Setup wizard steps
const (
	wizStepWelcome     = 0
	wizStepAPI         = 1
	wizStepModel       = 2
	wizStepSelfConfig  = 3
	wizStepPermissions = 4
	wizStepRules       = 5
	wizStepMemory      = 6
	wizStepTelegram    = 7
	wizStepBrowser     = 8
	wizStepSummary     = 9
	wizTotalSteps      = 10
)

// Step icons and titles
var wizStepInfo = [wizTotalSteps][2]string{
	{"🚀", "Welcome"},
	{"🔑", "API Configuration"},
	{"🤖", "Model Selection"},
	{"🎤", "Model Self-Config"},
	{"🛡️", "Permissions"},
	{"📜", "Rules & Personality"},
	{"🧠", "Memory"},
	{"📱", "Telegram"},
	{"🌐", "Browser Deps"},
	{"✅", "Summary"},
}

// API URL presets
var wizAPIURLs = []struct {
	Label string
	URL   string
}{
	{"OpenAI", "https://api.openai.com/v1"},
	{"NVIDIA", "https://integrate.api.nvidia.com/v1"},
	{"Anthropic", "https://api.anthropic.com/v1"},
	{"Local (Ollama)", "http://localhost:11434/v1"},
}

// Permissions presets
var wizPermOptions = []struct {
	Label string
	Desc  string
	Value string
}{
	{"Full Access", "All commands and tools (recommended for solo dev)", "full"},
	{"Restricted", "Only read, edit, write files (safe for teams)", "restricted"},
	{"Read-Only", "Only read operations (maximum safety)", "read-only"},
}

// Rules presets
var wizRulesOptions = []struct {
	Label string
	Desc  string
	Value string
}{
	{"Code Quality", "Clean, maintainable code (recommended)", "code-quality"},
	{"Security First", "Security best practices", "security"},
	{"Performance", "Speed and efficiency", "performance"},
	{"Personal Assistant", "Helpful and conversational", "personal-assistant"},
	{"Cybersecurity", "Penetration testing and auditing", "cybersecurity"},
	{"Data Science", "ML/AI and analytics", "data-science"},
	{"DevOps", "Infrastructure and automation", "devops"},
	{"None", "No predefined rules", "none"},
}

// Memory presets
var wizMemOptions = []struct {
	Label string
	Desc  string
	Value string
}{
	{"Balanced", "20/50/100 items (recommended)", "balanced"},
	{"Minimal", "10/25/50 items (lightweight)", "minimal"},
	{"Extended", "30/100/200 items (maximum context)", "extended"},
}

// Async messages
type modelsFetchedMsg struct {
	models []llm.ModelInfo
	err    error
}

type selfConfigDoneMsg struct {
	config *llm.ModelSelfConfig
	err    error
}

type telegramValidatedMsg struct {
	botName string
	err     error
}

type browserInstallDoneMsg struct {
	err error
}

type saveDoneMsg struct {
	err error
}

// SetupWizardModel is the bubbletea model for the interactive setup wizard.
type SetupWizardModel struct {
	step   int
	width  int
	height int

	// Step 0: Welcome
	nameInput textinput.Model

	// Step 1: API
	apiURLCursor     int
	apiKeyInput      textinput.Model
	customURLInput   textinput.Model
	apiSubStep       int // 0=URL list, 1=custom URL input, 2=key
	detectedProvider string
	selectedURL      string

	// Step 2: Model
	models       []llm.ModelInfo
	modelCursor  int
	modelPage    int
	modelSearch  textinput.Model
	modelLoading bool
	modelErr     error

	// Step 3: Self-config
	selfConfigSpinner spinner.Model
	selfConfigResult  *llm.ModelSelfConfig
	selfConfigDone    bool
	selfConfigErr     error

	// Step 4-6: List selections
	permCursor  int
	rulesCursor int
	memCursor   int

	// Step 7: Telegram (expanded)
	telegramEnabled    bool
	telegramInput      textinput.Model // bot token
	telegramChatInput  textinput.Model // chat ID
	telegramSubStep    int             // 0=yes/no, 1=token, 2=validating, 3=chatID+pairing
	telegramValidating bool
	telegramBotName    string // bot username from validation
	telegramValidErr   error

	// Step 8: Browser
	browserDepsOK     bool
	browserCursor     int // 0=yes, 1=no
	browserInstalling bool
	browserSpinner    spinner.Model
	browserDone       bool
	browserErr        error

	// Step 9: Summary / save
	saving   bool
	saveErr  error
	saveDone bool

	// Collected values
	agentName      string
	apiURL         string
	apiKey         string
	primaryModel   string
	fallbackModels []string
	permPreset     string
	rulesPreset    string
	memPreset      string
	telegramToken  string
	telegramChatID int64
	primaryConfig  *llm.ModelSelfConfig

	quitting bool
	appRoot  string
}

// NewSetupWizardModel creates a new setup wizard model.
func NewSetupWizardModel(appRoot string) SetupWizardModel {
	ni := textinput.New()
	ni.Placeholder = "ClosedWheeler"
	ni.CharLimit = 64
	ni.Width = 40
	ni.Focus()

	aki := textinput.New()
	aki.Placeholder = "sk-..."
	aki.CharLimit = 256
	aki.Width = 50
	aki.EchoMode = textinput.EchoPassword

	cui := textinput.New()
	cui.Placeholder = "https://your-api.example.com/v1"
	cui.CharLimit = 256
	cui.Width = 50

	ms := textinput.New()
	ms.Placeholder = "Filter models..."
	ms.CharLimit = 64
	ms.Width = 40

	ti := textinput.New()
	ti.Placeholder = "Bot token..."
	ti.CharLimit = 128
	ti.Width = 50

	tci := textinput.New()
	tci.Placeholder = "e.g. 123456789"
	tci.CharLimit = 20
	tci.Width = 30

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(PrimaryColor)

	bsp := spinner.New()
	bsp.Spinner = spinner.Dot
	bsp.Style = lipgloss.NewStyle().Foreground(PrimaryColor)

	return SetupWizardModel{
		step:              wizStepWelcome,
		nameInput:         ni,
		apiKeyInput:       aki,
		customURLInput:    cui,
		modelSearch:       ms,
		telegramInput:     ti,
		telegramChatInput: tci,
		selfConfigSpinner: sp,
		browserSpinner:    bsp,
		appRoot:           appRoot,
		agentName:         "ClosedWheeler",
		apiURL:            "https://api.openai.com/v1",
		permPreset:        "full",
		rulesPreset:       "code-quality",
		memPreset:         "balanced",
	}
}

// Init implements tea.Model.
func (m SetupWizardModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.selfConfigSpinner.Tick)
}

// Update implements tea.Model.
func (m SetupWizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.quitting = true
			return m, tea.Quit
		}
		return m.handleKeyMsg(msg)

	case modelsFetchedMsg:
		m.modelLoading = false
		m.models = msg.models
		m.modelErr = msg.err
		m.modelCursor = 0
		m.modelPage = 0
		return m, nil

	case selfConfigDoneMsg:
		m.selfConfigDone = true
		m.selfConfigResult = msg.config
		m.selfConfigErr = msg.err
		if msg.config != nil {
			m.primaryConfig = msg.config
		}
		return m, nil

	case telegramValidatedMsg:
		m.telegramValidating = false
		m.telegramBotName = msg.botName
		m.telegramValidErr = msg.err
		if msg.err == nil {
			m.telegramSubStep = 3
			m.telegramChatInput.Focus()
			return m, textinput.Blink
		}
		// Stay on token step so user can fix
		m.telegramSubStep = 1
		m.telegramInput.Focus()
		return m, textinput.Blink

	case browserInstallDoneMsg:
		m.browserInstalling = false
		m.browserDone = true
		m.browserErr = msg.err
		return m, nil

	case saveDoneMsg:
		m.saving = false
		m.saveDone = true
		m.saveErr = msg.err
		if msg.err == nil {
			return m, tea.Quit
		}
		return m, nil

	case spinner.TickMsg:
		var cmd1, cmd2 tea.Cmd
		m.selfConfigSpinner, cmd1 = m.selfConfigSpinner.Update(msg)
		m.browserSpinner, cmd2 = m.browserSpinner.Update(msg)
		return m, tea.Batch(cmd1, cmd2)
	}

	return m, nil
}

func (m SetupWizardModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.step {
	case wizStepWelcome:
		return m.updateWelcome(msg)
	case wizStepAPI:
		return m.updateAPI(msg)
	case wizStepModel:
		return m.updateModel(msg)
	case wizStepSelfConfig:
		return m.updateSelfConfig(msg)
	case wizStepPermissions:
		return m.updatePermissions(msg)
	case wizStepRules:
		return m.updateRules(msg)
	case wizStepMemory:
		return m.updateMemory(msg)
	case wizStepTelegram:
		return m.updateTelegram(msg)
	case wizStepBrowser:
		return m.updateBrowser(msg)
	case wizStepSummary:
		return m.updateSummary(msg)
	}
	return m, nil
}

// View implements tea.Model.
func (m SetupWizardModel) View() string {
	boxWidth := m.width - 4
	if boxWidth < 50 {
		boxWidth = 50
	}

	var s strings.Builder

	s.WriteString(WizardTitleStyle.Render("ClosedWheelerAGI Setup"))
	s.WriteString("\n\n")
	s.WriteString(m.renderProgressBar(boxWidth - 8))
	s.WriteString("\n\n")

	switch m.step {
	case wizStepWelcome:
		s.WriteString(m.viewWelcome())
	case wizStepAPI:
		s.WriteString(m.viewAPI())
	case wizStepModel:
		s.WriteString(m.viewModel())
	case wizStepSelfConfig:
		s.WriteString(m.viewSelfConfig())
	case wizStepPermissions:
		s.WriteString(m.viewPermissions())
	case wizStepRules:
		s.WriteString(m.viewRules())
	case wizStepMemory:
		s.WriteString(m.viewMemory())
	case wizStepTelegram:
		s.WriteString(m.viewTelegram())
	case wizStepBrowser:
		s.WriteString(m.viewBrowser())
	case wizStepSummary:
		s.WriteString(m.viewSummary())
	}

	return WizardBoxStyle.Width(boxWidth).Render(s.String())
}

func (m SetupWizardModel) renderProgressBar(width int) string {
	icon := wizStepInfo[m.step][0]
	title := wizStepInfo[m.step][1]

	label := fmt.Sprintf("Step %d/%d — %s %s", m.step+1, wizTotalSteps, icon, title)

	barWidth := width - 4
	if barWidth < 10 {
		barWidth = 10
	}
	filled := barWidth * (m.step + 1) / wizTotalSteps
	empty := barWidth - filled

	bar := WizardProgressBarFillStyle.Render(strings.Repeat("█", filled)) +
		WizardProgressBarEmptyStyle.Render(strings.Repeat("░", empty))

	return label + "  " + bar
}

// RunSetupWizard runs the bubbletea setup wizard and returns an error if setup was
// cancelled or failed.
func RunSetupWizard(appRoot string) error {
	model := NewSetupWizardModel(appRoot)
	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("wizard failed: %w", err)
	}

	final, ok := finalModel.(SetupWizardModel)
	if !ok {
		return fmt.Errorf("unexpected model type")
	}

	if final.quitting && !final.saveDone {
		return fmt.Errorf("setup cancelled by user")
	}

	if final.saveErr != nil {
		return fmt.Errorf("save failed: %w", final.saveErr)
	}

	return nil
}
