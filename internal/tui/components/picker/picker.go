// Package picker is a modal single-choice list used to pick a field option
// (which column to move a card to, a priority, a state). Like prompt, it is a
// dumb component: the parent drives navigation keys and reads Selected.
package picker

import (
	"strings"

	"github.com/0xheartcode/lazydash/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	title   string
	options []string
	cursor  int
	width   int
}

func New() Model { return Model{} }

// Open configures the picker with a title and options and resets the cursor.
func (m *Model) Open(title string, options []string) {
	m.title = title
	m.options = options
	m.cursor = 0
}

func (m *Model) SetWidth(w int) { m.width = w }

func (m *Model) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *Model) MoveDown() {
	if m.cursor < len(m.options)-1 {
		m.cursor++
	}
}

// Selected returns the highlighted option, or "" when there are none.
func (m Model) Selected() string {
	if m.cursor < 0 || m.cursor >= len(m.options) {
		return ""
	}
	return m.options[m.cursor]
}

func (m Model) View() string {
	lines := []string{theme.Title.Render(m.title), ""}
	for i, o := range m.options {
		if i == m.cursor {
			lines = append(lines, theme.CardCursor.Render("> ")+theme.CardSelected.Render(o))
		} else {
			lines = append(lines, "  "+theme.CardTitle.Render(o))
		}
	}
	lines = append(lines, "", theme.Muted.Render("enter select · esc cancel"))
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorPrimary).
		Padding(1, 3).
		Width(m.width).
		Render(strings.Join(lines, "\n"))
}
