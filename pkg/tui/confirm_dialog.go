package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmDialog represents a confirmation dialog
type ConfirmDialog struct {
	active    bool
	title     string
	message   string
	onConfirm func()
	onCancel  func()
	help      help.Model
	keys      confirmKeyMap
}

type confirmKeyMap struct {
	confirm key.Binding
	cancel  key.Binding
	help    key.Binding
}

func (k confirmKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.confirm, k.cancel, k.help}
}

func (k confirmKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.confirm, k.cancel},
		{k.help},
	}
}

func newConfirmKeyMap() confirmKeyMap {
	return confirmKeyMap{
		confirm: key.NewBinding(key.WithKeys("enter", "y"), key.WithHelp("enter/y", "confirm")),
		cancel:  key.NewBinding(key.WithKeys("esc", "n", "q"), key.WithHelp("esc/n/q", "cancel")),
		help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
	}
}

// NewConfirmDialog creates a new confirmation dialog
func NewConfirmDialog(title, message string, onConfirm, onCancel func()) *ConfirmDialog {
	return &ConfirmDialog{
		title:     title,
		message:   message,
		onConfirm: onConfirm,
		onCancel:  onCancel,
		help:      help.New(),
		keys:      newConfirmKeyMap(),
	}
}

// Show shows the confirmation dialog
func (d *ConfirmDialog) Show() {
	d.active = true
}

// Hide hides the confirmation dialog
func (d *ConfirmDialog) Hide() {
	d.active = false
}

// IsActive returns whether the dialog is active
func (d *ConfirmDialog) IsActive() bool {
	return d.active
}

// Update handles dialog updates
func (d *ConfirmDialog) Update(msg tea.Msg) tea.Cmd {
	if !d.active {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, d.keys.confirm):
			if d.onConfirm != nil {
				d.onConfirm()
			}
			d.Hide()
		case key.Matches(msg, d.keys.cancel):
			if d.onCancel != nil {
				d.onCancel()
			}
			d.Hide()
		case key.Matches(msg, d.keys.help):
			d.help.ShowAll = !d.help.ShowAll
		}
	}

	return nil
}

// View renders the confirmation dialog
func (d *ConfirmDialog) View() string {
	if !d.active {
		return ""
	}

	// Calculate dialog dimensions
	width := 60
	height := 8

	// Dialog styles
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Background(BgDark).
		Padding(1, 2).
		Width(width).
		Height(height)

	titleStyle := lipgloss.NewStyle().
		Foreground(ErrorColor).
		Bold(true).
		MarginBottom(1)

	messageStyle := lipgloss.NewStyle().
		Foreground(TextPrimary).
		MarginBottom(2)

	keysStyle := lipgloss.NewStyle().
		Foreground(TextSecondary).
		MarginTop(1)

	// Build dialog content
	content := titleStyle.Render(d.title) + "\n"
	content += messageStyle.Render(d.message) + "\n"
	content += keysStyle.Render(fmt.Sprintf("%s • %s",
		d.keys.confirm.Help().Key,
		d.keys.cancel.Help().Key))

	if d.help.ShowAll {
		content += "\n\n" + d.help.View(d.keys)
	}

	return dialogStyle.Render(content)
}
