// Package prompt is a small modal text field used to collect a single- or
// multi-line value (an issue title, a comment, a label list). It is a dumb
// component: the parent decides which keys submit or cancel and reads Value.
package prompt

import (
	"github.com/0xheartcode/lazydash/internal/tui/theme"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	title string
	hint  string
	input textinput.Model
	area  textarea.Model
	multi bool
	width int
}

func New() Model {
	ti := textinput.New()
	ti.Prompt = "› "
	ta := textarea.New()
	ta.ShowLineNumbers = false
	return Model{input: ti, area: ta}
}

// Open configures the prompt for a field and focuses it, returning the focus
// command so the caller can start the cursor. multiline chooses a textarea
// (submit on ctrl+d) over a single-line input (submit on enter).
func (m *Model) Open(title, placeholder string, multiline bool) tea.Cmd {
	m.title = title
	m.multi = multiline
	if multiline {
		m.hint = "ctrl+d submit · esc cancel"
		m.area.Reset()
		m.area.Placeholder = placeholder
		return m.area.Focus()
	}
	m.hint = "enter submit · esc cancel"
	m.input.Reset()
	m.input.Placeholder = placeholder
	return m.input.Focus()
}

// Multiline reports whether the open prompt is a multi-line field.
func (m Model) Multiline() bool { return m.multi }

// SetWidth sizes the modal and its field.
func (m *Model) SetWidth(w int) {
	m.width = w
	inner := w - 6
	if inner < 4 {
		inner = 4
	}
	m.input.Width = inner
	m.area.SetWidth(inner)
	m.area.SetHeight(5)
}

// Value returns the current field contents.
func (m Model) Value() string {
	if m.multi {
		return m.area.Value()
	}
	return m.input.Value()
}

// Update forwards a message (key, cursor blink) to the focused field.
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	if m.multi {
		m.area, cmd = m.area.Update(msg)
	} else {
		m.input, cmd = m.input.Update(msg)
	}
	return cmd
}

func (m Model) View() string {
	field := m.input.View()
	if m.multi {
		field = m.area.View()
	}
	content := theme.Title.Render(m.title) + "\n\n" + field + "\n\n" + theme.Muted.Render(m.hint)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorPrimary).
		Padding(1, 3).
		Width(m.width).
		Render(content)
}
