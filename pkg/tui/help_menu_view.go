package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// helpMenuView renders the help menu overlay.
func (m EnhancedModel) helpMenuView() string {
	if m.helpSearchMode {
		return m.helpSearchView()
	}

	categories := GetAllCommands()
	boxWidth := m.width - 6
	if boxWidth < 40 {
		boxWidth = 40
	}

	var s strings.Builder

	// Title
	s.WriteString(HelpTitleStyle.Render("Command Reference"))
	s.WriteString("\n\n")

	// Two-column layout: categories on left, commands on right
	catWidth := 22
	cmdWidth := boxWidth - catWidth - 10
	if cmdWidth < 20 {
		cmdWidth = 20
	}

	// Build categories column
	var catCol strings.Builder
	catCol.WriteString(HelpDetailLabelStyle.Render("CATEGORIES"))
	catCol.WriteString("\n")
	for i, cat := range categories {
		cursor := "  "
		style := HelpCategoryInactiveStyle
		if i == m.helpCategoryCursor {
			cursor = "▸ "
			style = HelpCategoryActiveStyle
		}
		label := fmt.Sprintf("%s %s", cat.Icon, cat.Name)
		catCol.WriteString(style.Render(cursor + label))
		catCol.WriteString("\n")
	}

	// Build commands column
	var cmdCol strings.Builder
	if m.helpCategoryCursor < len(categories) {
		cat := categories[m.helpCategoryCursor]
		cmdCol.WriteString(HelpDetailLabelStyle.Render("COMMANDS"))
		cmdCol.WriteString("\n")
		for i, cmd := range cat.Commands {
			cursor := "  "
			style := HelpCommandNormalStyle
			if i == m.helpCommandCursor {
				cursor = "▸ "
				style = HelpCommandSelectedStyle
			}
			aliases := ""
			if len(cmd.Aliases) > 0 {
				aliases = fmt.Sprintf(" (/%s)", strings.Join(cmd.Aliases, ", /"))
			}
			line := fmt.Sprintf("/%s%s", cmd.Name, aliases)
			cmdCol.WriteString(style.Render(cursor + line))
			cmdCol.WriteString("\n")
			// Description under each command
			desc := "    " + cmd.Description
			if len(desc) > cmdWidth {
				desc = desc[:cmdWidth-3] + "..."
			}
			cmdCol.WriteString(HelpFooterStyle.Render(desc))
			cmdCol.WriteString("\n")
		}
	}

	// Join columns side by side
	catLines := strings.Split(catCol.String(), "\n")
	cmdLines := strings.Split(cmdCol.String(), "\n")

	maxLines := len(catLines)
	if len(cmdLines) > maxLines {
		maxLines = len(cmdLines)
	}

	for i := 0; i < maxLines; i++ {
		cat := ""
		cmd := ""
		if i < len(catLines) {
			cat = catLines[i]
		}
		if i < len(cmdLines) {
			cmd = cmdLines[i]
		}

		// Pad category column to fixed width
		catRendered := cat
		catVisualWidth := lipgloss.Width(catRendered)
		if catVisualWidth < catWidth {
			catRendered += strings.Repeat(" ", catWidth-catVisualWidth)
		}

		s.WriteString(catRendered)
		s.WriteString("  ")
		s.WriteString(cmd)
		s.WriteString("\n")
	}

	// Detail panel for selected command
	selectedCmd := m.helpGetSelectedCommand()
	if selectedCmd != nil {
		s.WriteString("\n")
		s.WriteString(DividerStyle.Render(strings.Repeat("─", boxWidth-6)))
		s.WriteString("\n\n")
		s.WriteString(HelpDetailLabelStyle.Render("/" + selectedCmd.Name))
		s.WriteString("\n")
		s.WriteString(HelpDetailValueStyle.Render(selectedCmd.Description))
		s.WriteString("\n")
		s.WriteString(HelpDetailLabelStyle.Render("Usage: "))
		s.WriteString(HelpDetailValueStyle.Render(selectedCmd.Usage))
		s.WriteString("\n")
		if len(selectedCmd.Aliases) > 0 {
			s.WriteString(HelpDetailLabelStyle.Render("Aliases: "))
			s.WriteString(HelpDetailValueStyle.Render(strings.Join(selectedCmd.Aliases, ", ")))
			s.WriteString("\n")
		}
	}

	// Footer
	s.WriteString("\n")
	s.WriteString(HelpFooterStyle.Render("Tab/Shift+Tab Category | ↑/↓ Command | / Search | Enter Run | Esc Close"))

	return HelpBoxStyle.Width(boxWidth).Render(s.String())
}

// helpSearchView renders the search mode overlay.
func (m EnhancedModel) helpSearchView() string {
	boxWidth := m.width - 6
	if boxWidth < 40 {
		boxWidth = 40
	}

	var s strings.Builder

	// Title
	s.WriteString(HelpTitleStyle.Render("Command Search"))
	s.WriteString("\n\n")

	// Search input
	s.WriteString(HelpSearchStyle.Render("/ "))
	s.WriteString(m.helpSearchInput.View())
	s.WriteString("\n\n")

	// Results
	if len(m.helpSearchResults) == 0 {
		s.WriteString(HelpFooterStyle.Render("  No matching commands"))
		s.WriteString("\n")
	} else {
		maxShow := 15
		if maxShow > len(m.helpSearchResults) {
			maxShow = len(m.helpSearchResults)
		}
		for i := 0; i < maxShow; i++ {
			result := m.helpSearchResults[i]
			cursor := "  "
			style := HelpCommandNormalStyle
			if i == m.helpCommandCursor {
				cursor = "▸ "
				style = HelpCommandSelectedStyle
			}
			aliases := ""
			if len(result.Aliases) > 0 {
				aliases = fmt.Sprintf(" (/%s)", strings.Join(result.Aliases, ", /"))
			}
			line := fmt.Sprintf("/%s%s", result.Name, aliases)
			s.WriteString(style.Render(cursor + line))
			s.WriteString("  ")
			s.WriteString(HelpFooterStyle.Render(result.Description))
			s.WriteString("\n")
		}
		if len(m.helpSearchResults) > maxShow {
			s.WriteString(HelpFooterStyle.Render(fmt.Sprintf("  ... %d more results", len(m.helpSearchResults)-maxShow)))
			s.WriteString("\n")
		}
	}

	// Footer
	s.WriteString("\n")
	s.WriteString(HelpFooterStyle.Render("↑/↓ Navigate | Enter Run | Esc Back"))

	return HelpBoxStyle.Width(boxWidth).Render(s.String())
}
