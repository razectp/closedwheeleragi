package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LogEntry represents a log entry for the table
type LogEntry struct {
	Timestamp string
	Level     string
	Message   string
	Source    string
}

// LogTable represents a log viewer table
type LogTable struct {
	table     table.Model
	active    bool
	logs      []LogEntry
	keys      logTableKeyMap
	helpShown bool
}

type logTableKeyMap struct {
	close    key.Binding
	up       key.Binding
	down     key.Binding
	pageUp   key.Binding
	pageDown key.Binding
	help     key.Binding
}

func (k logTableKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.close, k.up, k.down, k.help}
}

func (k logTableKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.close, k.up, k.down},
		{k.pageUp, k.pageDown, k.help},
	}
}

func newLogTableKeyMap() logTableKeyMap {
	return logTableKeyMap{
		close:    key.NewBinding(key.WithKeys("esc", "q"), key.WithHelp("esc/q", "close")),
		up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		pageUp:   key.NewBinding(key.WithKeys("pgup", "u"), key.WithHelp("pgup/u", "page up")),
		pageDown: key.NewBinding(key.WithKeys("pgdown", "d"), key.WithHelp("pgdn/d", "page down")),
		help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
	}
}

// NewLogTable creates a new log table
func NewLogTable() *LogTable {
	columns := []table.Column{
		{Title: "Time", Width: 12},
		{Title: "Level", Width: 8},
		{Title: "Source", Width: 15},
		{Title: "Message", Width: 50},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(PrimaryColor).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(TextPrimary).
		Background(BgDark).
		Bold(false)

	t.SetStyles(s)
	t.SetRows([]table.Row{})

	return &LogTable{
		table:  t,
		active: false,
		logs:   make([]LogEntry, 0),
		keys:   newLogTableKeyMap(),
	}
}

// Show shows the log table
func (lt *LogTable) Show() {
	lt.active = true
}

// Hide hides the log table
func (lt *LogTable) Hide() {
	lt.active = false
}

// IsActive returns whether the table is active
func (lt *LogTable) IsActive() bool {
	return lt.active
}

// AddLog adds a new log entry
func (lt *LogTable) AddLog(level, message, source string) {
	entry := LogEntry{
		Timestamp: time.Now().Format("15:04:05"),
		Level:     level,
		Message:   message,
		Source:    source,
	}

	lt.logs = append(lt.logs, entry)
	lt.updateTableRows()
}

// AddLogWithTimestamp adds a log entry with custom timestamp
func (lt *LogTable) AddLogWithTimestamp(timestamp, level, message, source string) {
	entry := LogEntry{
		Timestamp: timestamp,
		Level:     level,
		Message:   message,
		Source:    source,
	}

	lt.logs = append(lt.logs, entry)
	lt.updateTableRows()
}

// ClearLogs clears all log entries
func (lt *LogTable) ClearLogs() {
	lt.logs = make([]LogEntry, 0)
	lt.updateTableRows()
}

// updateTableRows updates the table with current logs
func (lt *LogTable) updateTableRows() {
	rows := make([]table.Row, 0, len(lt.logs))

	for _, log := range lt.logs {
		// Truncate message if too long
		message := log.Message
		if len(message) > 47 {
			message = message[:44] + "..."
		}

		// Color code level
		levelStyle := lipgloss.NewStyle()
		switch strings.ToUpper(log.Level) {
		case "ERROR":
			levelStyle = levelStyle.Foreground(ErrorColor)
		case "WARN":
			levelStyle = levelStyle.Foreground(AccentColor)
		case "INFO":
			levelStyle = levelStyle.Foreground(SuccessColor)
		case "DEBUG":
			levelStyle = levelStyle.Foreground(MutedColor)
		}

		row := table.Row{
			log.Timestamp,
			levelStyle.Render(log.Level),
			log.Source,
			message,
		}
		rows = append(rows, row)
	}

	lt.table.SetRows(rows)
	lt.table.GotoBottom() // Auto-scroll to latest
}

// Update handles table updates
func (lt *LogTable) Update(msg tea.Msg) tea.Cmd {
	if !lt.active {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, lt.keys.close):
			lt.Hide()
			return nil
		case key.Matches(msg, lt.keys.help):
			// Toggle help - implementar manualmente
			lt.helpShown = !lt.helpShown
		}
	}

	var cmd tea.Cmd
	lt.table, cmd = lt.table.Update(msg)
	return cmd
}

// View renders the log table
func (lt *LogTable) View() string {
	if !lt.active {
		return ""
	}

	title := "📋 Log Viewer"
	titleStyle := lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(true).
		MarginBottom(1)

	tableStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Padding(1, 1).
		MaxWidth(100)

	helpStyle := lipgloss.NewStyle().
		Foreground(TextMuted).
		Faint(true).
		MarginTop(1)

	content := titleStyle.Render(title) + "\n"
	content += lt.table.View()

	if lt.helpShown {
		helpText := ""
		for _, binding := range lt.keys.ShortHelp() {
			helpText += binding.Help().Key + " - " + binding.Help().Desc + " "
		}
		content += "\n" + helpStyle.Render(helpText)
	}

	return tableStyle.Render(content)
}

// GetSelectedLog returns the currently selected log entry
func (lt *LogTable) GetSelectedLog() *LogEntry {
	if len(lt.logs) == 0 {
		return nil
	}

	selectedIndex := lt.table.Cursor()
	if selectedIndex >= 0 && selectedIndex < len(lt.logs) {
		return &lt.logs[selectedIndex]
	}

	return nil
}
