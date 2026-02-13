package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// helpFlatCommand is a flattened command entry used for search results.
type helpFlatCommand struct {
	CategoryIdx int
	CommandIdx  int
	Name        string
	Aliases     []string
	Description string
	Usage       string
	Category    string
}

// initHelpMenu activates the help menu overlay and resets state.
func (m *EnhancedModel) initHelpMenu() {
	m.helpActive = true
	m.helpCategoryCursor = 0
	m.helpCommandCursor = 0
	m.helpSearchMode = false
	m.helpSearchResults = nil

	ti := textinput.New()
	ti.Placeholder = "Search commands..."
	ti.CharLimit = 64
	ti.Width = 30
	m.helpSearchInput = ti
}

// helpMenuUpdate handles keyboard input while the help overlay is active.
func (m *EnhancedModel) helpMenuUpdate(msg tea.KeyMsg) (*EnhancedModel, tea.Cmd) {
	if m.helpSearchMode {
		return m.helpSearchModeUpdate(msg)
	}

	switch msg.String() {
	case "esc":
		m.helpActive = false
		return m, nil

	case "/":
		m.helpSearchMode = true
		m.helpSearchInput.SetValue("")
		m.helpSearchInput.Focus()
		m.helpSearchResults = m.buildSearchResults("")
		return m, textinput.Blink

	case "tab", "right", "l":
		categories := GetAllCommands()
		if m.helpCategoryCursor < len(categories)-1 {
			m.helpCategoryCursor++
		} else {
			m.helpCategoryCursor = 0
		}
		m.helpCommandCursor = 0
		return m, nil

	case "shift+tab", "left", "h":
		categories := GetAllCommands()
		if m.helpCategoryCursor > 0 {
			m.helpCategoryCursor--
		} else {
			m.helpCategoryCursor = len(categories) - 1
		}
		m.helpCommandCursor = 0
		return m, nil

	case "up", "k":
		if m.helpCommandCursor > 0 {
			m.helpCommandCursor--
		}
		return m, nil

	case "down", "j":
		categories := GetAllCommands()
		if m.helpCategoryCursor < len(categories) {
			cat := categories[m.helpCategoryCursor]
			if m.helpCommandCursor < len(cat.Commands)-1 {
				m.helpCommandCursor++
			}
		}
		return m, nil

	case "enter":
		cmd := m.helpGetSelectedCommand()
		if cmd != nil {
			m.helpActive = false
			m.helpSearchMode = false

			// Commands needing args: insert into textarea for user to complete
			if commandNeedsArgs(cmd) {
				m.textarea.SetValue("/" + cmd.Name + " ")
				m.textarea.Focus()
				return m, nil
			}

			result, c := cmd.Handler(m, nil)
			if em, ok := result.(*EnhancedModel); ok {
				return em, c
			}
			return m, c
		}
		return m, nil
	}

	return m, nil
}

// helpSearchModeUpdate handles keyboard input while in search mode.
func (m *EnhancedModel) helpSearchModeUpdate(msg tea.KeyMsg) (*EnhancedModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.helpSearchMode = false
		m.helpSearchResults = nil
		m.helpSearchInput.Blur()
		return m, nil

	case tea.KeyEnter:
		if len(m.helpSearchResults) > 0 && m.helpCommandCursor < len(m.helpSearchResults) {
			result := m.helpSearchResults[m.helpCommandCursor]
			categories := GetAllCommands()
			if result.CategoryIdx < len(categories) {
				cat := categories[result.CategoryIdx]
				if result.CommandIdx < len(cat.Commands) {
					cmd := cat.Commands[result.CommandIdx]
					m.helpActive = false
					m.helpSearchMode = false

					// Commands needing args: insert into textarea for user to complete
					if commandNeedsArgs(&cmd) {
						m.textarea.SetValue("/" + cmd.Name + " ")
						m.textarea.Focus()
						return m, nil
					}

					res, c := cmd.Handler(m, nil)
					if em, ok := res.(*EnhancedModel); ok {
						return em, c
					}
					return m, c
				}
			}
		}
		return m, nil

	case tea.KeyUp:
		if m.helpCommandCursor > 0 {
			m.helpCommandCursor--
		}
		return m, nil

	case tea.KeyDown:
		if m.helpCommandCursor < len(m.helpSearchResults)-1 {
			m.helpCommandCursor++
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.helpSearchInput, cmd = m.helpSearchInput.Update(msg)

	query := m.helpSearchInput.Value()
	m.helpSearchResults = m.buildSearchResults(query)
	m.helpCommandCursor = 0

	return m, cmd
}

// buildSearchResults returns flattened commands matching the query.
func (m EnhancedModel) buildSearchResults(query string) []helpFlatCommand {
	query = strings.ToLower(strings.TrimSpace(query))
	categories := GetAllCommands()
	var results []helpFlatCommand

	for ci, cat := range categories {
		for cmi, cmd := range cat.Commands {
			if query == "" || matchesQuery(cmd, query) {
				results = append(results, helpFlatCommand{
					CategoryIdx: ci,
					CommandIdx:  cmi,
					Name:        cmd.Name,
					Aliases:     cmd.Aliases,
					Description: cmd.Description,
					Usage:       cmd.Usage,
					Category:    cat.Name,
				})
			}
		}
	}
	return results
}

// matchesQuery returns true if a command matches the search query.
func matchesQuery(cmd Command, query string) bool {
	if strings.Contains(strings.ToLower(cmd.Name), query) {
		return true
	}
	if strings.Contains(strings.ToLower(cmd.Description), query) {
		return true
	}
	for _, alias := range cmd.Aliases {
		if strings.Contains(strings.ToLower(alias), query) {
			return true
		}
	}
	return false
}

// commandNeedsArgs returns true if the command usage contains a required argument placeholder.
func commandNeedsArgs(cmd *Command) bool {
	return strings.Contains(cmd.Usage, "<")
}

// helpGetSelectedCommand returns the currently selected command.
func (m *EnhancedModel) helpGetSelectedCommand() *Command {
	categories := GetAllCommands()
	if m.helpCategoryCursor >= len(categories) {
		return nil
	}
	cat := categories[m.helpCategoryCursor]
	if m.helpCommandCursor >= len(cat.Commands) {
		return nil
	}
	cmd := cat.Commands[m.helpCommandCursor]
	return &cmd
}
