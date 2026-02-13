package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConversationPage represents a page of conversation content
type ConversationPage struct {
	Content    string
	StartIndex int
	EndIndex   int
	PageNum    int
}

// ConversationPaginator handles pagination of long conversations
type ConversationPaginator struct {
	paginator  paginator.Model
	active     bool
	pages      []ConversationPage
	totalPages int
	keys       paginatorKeyMap
	helpShown  bool
}

type paginatorKeyMap struct {
	close key.Binding
	next  key.Binding
	prev  key.Binding
	first key.Binding
	last  key.Binding
	help  key.Binding
}

func (k paginatorKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.close, k.prev, k.next, k.help}
}

func (k paginatorKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.close, k.prev, k.next},
		{k.first, k.last, k.help},
	}
}

func newPaginatorKeyMap() paginatorKeyMap {
	return paginatorKeyMap{
		close: key.NewBinding(key.WithKeys("esc", "q"), key.WithHelp("esc/q", "close")),
		next:  key.NewBinding(key.WithKeys("right", "l", "n"), key.WithHelp("→/l/n", "next")),
		prev:  key.NewBinding(key.WithKeys("left", "h", "p"), key.WithHelp("←/h/p", "prev")),
		first: key.NewBinding(key.WithKeys("home", "g"), key.WithHelp("home/g", "first")),
		last:  key.NewBinding(key.WithKeys("end", "G"), key.WithHelp("end/G", "last")),
		help:  key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
	}
}

// NewConversationPaginator creates a new conversation paginator
func NewConversationPaginator() *ConversationPaginator {
	p := paginator.New()
	p.Type = paginator.Dots
	p.ActiveDot = lipgloss.NewStyle().Foreground(PrimaryColor).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(MutedColor).Render("•")

	return &ConversationPaginator{
		paginator:  p,
		active:     false,
		pages:      make([]ConversationPage, 0),
		totalPages: 0,
		keys:       newPaginatorKeyMap(),
	}
}

// Show shows the paginator
func (cp *ConversationPaginator) Show() {
	cp.active = true
}

// Hide hides the paginator
func (cp *ConversationPaginator) Hide() {
	cp.active = false
}

// IsActive returns whether the paginator is active
func (cp *ConversationPaginator) IsActive() bool {
	return cp.active
}

// SetContent paginates the given content
func (cp *ConversationPaginator) SetContent(content string, maxLinesPerPage int) {
	lines := splitIntoLines(content)
	cp.pages = make([]ConversationPage, 0)

	for i := 0; i < len(lines); i += maxLinesPerPage {
		end := i + maxLinesPerPage
		if end > len(lines) {
			end = len(lines)
		}

		page := ConversationPage{
			Content:    strings.Join(lines[i:end], "\n"),
			StartIndex: i,
			EndIndex:   end - 1,
			PageNum:    len(cp.pages) + 1,
		}

		cp.pages = append(cp.pages, page)
	}

	cp.totalPages = len(cp.pages)
	cp.paginator.SetTotalPages(cp.totalPages)

	if cp.totalPages > 0 {
		cp.paginator.Page = 1 // Reset to first page
	}
}

// GetCurrentPage returns the current page content
func (cp *ConversationPaginator) GetCurrentPage() *ConversationPage {
	if cp.totalPages == 0 {
		return nil
	}

	currentPage := cp.paginator.Page - 1 // Convert to 0-based
	if currentPage >= 0 && currentPage < len(cp.pages) {
		return &cp.pages[currentPage]
	}

	return nil
}

// GetPageInfo returns information about current page
func (cp *ConversationPaginator) GetPageInfo() (current, total int, content string) {
	if cp.totalPages == 0 {
		return 0, 0, ""
	}

	current = cp.paginator.Page
	total = cp.totalPages

	if page := cp.GetCurrentPage(); page != nil {
		content = page.Content
	}

	return current, total, content
}

// Update handles paginator updates
func (cp *ConversationPaginator) Update(msg tea.Msg) tea.Cmd {
	if !cp.active {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, cp.keys.close):
			cp.Hide()
			return nil
		case key.Matches(msg, cp.keys.first):
			cp.paginator.Page = 1
			return nil
		case key.Matches(msg, cp.keys.last):
			cp.paginator.Page = cp.totalPages
			return nil
		case key.Matches(msg, cp.keys.help):
			cp.helpShown = !cp.helpShown
			return nil
		}
	}

	var cmd tea.Cmd
	cp.paginator, cmd = cp.paginator.Update(msg)
	return cmd
}

// View renders the paginator
func (cp *ConversationPaginator) View() string {
	if !cp.active || cp.totalPages == 0 {
		return ""
	}

	currentPage := cp.GetCurrentPage()
	if currentPage == nil {
		return ""
	}

	title := fmt.Sprintf("📖 Conversation History (Page %d/%d)",
		cp.paginator.Page, cp.totalPages)
	titleStyle := lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(true).
		MarginBottom(1)

	contentStyle := lipgloss.NewStyle().
		Foreground(TextPrimary).
		MaxHeight(20).
		Width(80)

	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Padding(1, 2).
		MaxWidth(90)

	helpStyle := lipgloss.NewStyle().
		Foreground(TextMuted).
		Faint(true).
		MarginTop(1)

	content := titleStyle.Render(title) + "\n"
	content += contentStyle.Render(currentPage.Content) + "\n"
	content += cp.paginator.View() + "\n"

	if cp.helpShown {
		helpText := ""
		for _, binding := range cp.keys.ShortHelp() {
			helpText += binding.Help().Key + " - " + binding.Help().Desc + " "
		}
		content += helpStyle.Render(helpText)
	}

	return containerStyle.Render(content)
}

// splitIntoLines splits content into lines preserving structure
func splitIntoLines(content string) []string {
	if content == "" {
		return []string{}
	}

	lines := strings.Split(content, "\n")

	// Filter out empty lines at the end
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	return lines
}

// CanGoNext returns if there's a next page
func (cp *ConversationPaginator) CanGoNext() bool {
	return cp.paginator.Page < cp.totalPages
}

// CanGoPrev returns if there's a previous page
func (cp *ConversationPaginator) CanGoPrev() bool {
	return cp.paginator.Page > 1
}

// GoToPage goes to a specific page
func (cp *ConversationPaginator) GoToPage(page int) {
	if page >= 1 && page <= cp.totalPages {
		cp.paginator.Page = page
	}
}

// GetTotalPages returns the total number of pages
func (cp *ConversationPaginator) GetTotalPages() int {
	return cp.totalPages
}
