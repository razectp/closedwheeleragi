package tui

import (
	"fmt"
	"strings"

	"ClosedWheeler/pkg/config"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SetupModel handles the initial configuration wizard
type SetupModel struct {
	step       int
	config     *config.Config
	textInput  textinput.Model
	items      []string
	cursor     int
	choice     string
	err        error
	width      int
	height     int
	configPath string
}

const (
	StepWelcome = iota
	StepProvider
	StepAPIKey
	StepModel
	StepConfirm
	StepDone
)

var (
	// Wizard Styles
	wizardTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Bold(true).
				MarginBottom(1)

	wizardStepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			MarginBottom(1)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#10B981")).
				Bold(true)

	unselectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F9FAFB"))

	wizardBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			Padding(1, 2).
			Margin(1, 1)
)

// NewSetupModel creates a new setup wizard model
func NewSetupModel(cfg *config.Config, path string) SetupModel {
	ti := textinput.New()
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))
	ti.Focus()

	return SetupModel{
		step:       StepWelcome,
		config:     config.DefaultConfig(),
		textInput:  ti,
		configPath: path,
		items:      []string{"OpenAI", "Anthropic", "Gemini", "DeepSeek", "Local (Ollama)", "Custom"},
	}
}

// Init initializes the setup model
func (m SetupModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles setup events
func (m SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}

		switch m.step {
		case StepWelcome:
			if msg.Type == tea.KeyEnter {
				m.step++
				return m, nil
			}

		case StepProvider:
			switch msg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.items)-1 {
					m.cursor++
				}
			case "enter":
				m.choice = m.items[m.cursor]
				m.configureProviderDefaults(m.choice)
				m.step++
				m.textInput.Placeholder = "sk-..."
				m.textInput.EchoMode = textinput.EchoPassword
				m.textInput.SetValue("")
				m.textInput.Focus()
				return m, nil
			}

		case StepAPIKey:
			switch msg.Type {
			case tea.KeyEnter:
				m.config.APIKey = m.textInput.Value()
				m.step++
				m.textInput.Reset()
				m.textInput.EchoMode = textinput.EchoNormal
				m.textInput.SetValue(m.config.Model)
				return m, nil
			}
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd

		case StepModel:
			switch msg.Type {
			case tea.KeyEnter:
				m.config.Model = m.textInput.Value()
				m.step++
				return m, nil
			}
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd

		case StepConfirm:
			if msg.Type == tea.KeyEnter {
				if err := m.config.Save(m.configPath); err != nil {
					m.err = err
					return m, tea.Quit
				}
				m.step++
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m *SetupModel) configureProviderDefaults(provider string) {
	switch provider {
	case "OpenAI":
		m.config.APIBaseURL = "https://api.openai.com/v1"
		m.config.Model = "gpt-4o"
		m.config.Provider = "openai"
	case "Anthropic":
		m.config.APIBaseURL = "https://api.anthropic.com/v1"
		m.config.Model = "claude-sonnet-4-5-20250929"
		m.config.Provider = "anthropic"
	case "Gemini":
		m.config.APIBaseURL = "https://generativelanguage.googleapis.com/v1beta"
		m.config.Model = "gemini-1.5-pro"
		m.config.Provider = "openai"
	case "DeepSeek":
		m.config.APIBaseURL = "https://api.deepseek.com"
		m.config.Model = "deepseek-coder"
		m.config.Provider = "openai"
	case "Local (Ollama)":
		m.config.APIBaseURL = "http://localhost:11434/v1"
		m.config.Model = "llama3"
		m.config.Provider = "openai"
	}
}

// View renders the setup wizard
func (m SetupModel) View() string {
	var s strings.Builder

	// Title
	s.WriteString(wizardTitleStyle.Render("✨ Coder AGI Setup"))
	s.WriteString("\n\n")

	switch m.step {
	case StepWelcome:
		s.WriteString("Welcome! Let's get you set up in seconds.\n")
		s.WriteString("We need to configure your AI provider to get started.\n\n")
		s.WriteString(selectedItemStyle.Render("Press [Enter] to continue"))

	case StepProvider:
		s.WriteString("Select your AI Provider:\n\n")
		for i, item := range m.items {
			cursor := "  "
			style := unselectedItemStyle
			if m.cursor == i {
				cursor = "👉 "
				style = selectedItemStyle
			}
			s.WriteString(style.Render(fmt.Sprintf("%s%s\n", cursor, item)))
		}

	case StepAPIKey:
		s.WriteString(fmt.Sprintf("Enter your API Key for %s:\n", m.choice))
		s.WriteString("(Input will be hidden)\n\n")
		s.WriteString(m.textInput.View())

	case StepModel:
		s.WriteString("Confirm the Model to use:\n")
		s.WriteString("(You can change this later in config.json)\n\n")
		s.WriteString(m.textInput.View())

	case StepConfirm:
		s.WriteString("Does everything look correct?\n\n")
		s.WriteString(fmt.Sprintf("Provider: %s\n", m.choice))
		s.WriteString(fmt.Sprintf("Base URL: %s\n", m.config.APIBaseURL))
		s.WriteString(fmt.Sprintf("Model:    %s\n", m.config.Model))

		maskedKey := "Not set"
		if len(m.config.APIKey) > 8 {
			maskedKey = fmt.Sprintf("%s...%s", m.config.APIKey[:4], m.config.APIKey[len(m.config.APIKey)-4:])
		} else if len(m.config.APIKey) > 0 {
			maskedKey = "****"
		}
		s.WriteString(fmt.Sprintf("API Key:  %s\n\n", maskedKey))

		s.WriteString(selectedItemStyle.Render("Press [Enter] to Save and Start 🚀"))

	case StepDone:
		s.WriteString("Configuration saved successfully! Starting agent...\n")
	}

	return wizardBoxStyle.Render(s.String())
}

// RunSetup runs the setup wizard
func RunSetup(cfg *config.Config, path string) error {
	p := tea.NewProgram(NewSetupModel(cfg, path))
	_, err := p.Run()
	return err
}
