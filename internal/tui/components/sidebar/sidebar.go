package sidebar

import (
	"fmt"
	"strings"
	"time"

	"github.com/0xheartcode/lazydash/internal/core"
	"github.com/0xheartcode/lazydash/internal/tui/theme"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	card     *core.Card
	history  []core.Event // the selected issue's op-log, newest-first
	viewport viewport.Model
	width    int
	height   int
	active   bool
	ready    bool
}

func New() Model {
	return Model{}
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	innerW := w - 4
	innerH := h - 4
	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}
	if !m.ready {
		m.viewport = viewport.New(innerW, innerH)
		m.ready = true
	} else {
		m.viewport.Width = innerW
		m.viewport.Height = innerH
	}
	if m.card != nil {
		m.viewport.SetContent(m.buildContent(innerW))
	}
}

func (m *Model) SetCard(card *core.Card) {
	m.card = card
	if m.ready && card != nil {
		m.viewport.SetContent(m.buildContent(m.width - 4))
		m.viewport.GotoTop()
	}
}

// SetHistory loads the selected issue's events, shown as a timeline under the
// body. Pass them newest-first.
func (m *Model) SetHistory(events []core.Event) {
	m.history = events
	if m.ready && m.card != nil {
		m.viewport.SetContent(m.buildContent(m.width - 4))
	}
}

func (m *Model) SetActive(a bool) { m.active = a }

func (m *Model) ScrollUp()   { m.viewport.ScrollUp(1) }
func (m *Model) ScrollDown() { m.viewport.ScrollDown(1) }

func (m Model) View() string {
	border := theme.InactiveBorder
	if m.active {
		border = theme.ActiveBorder
	}

	var content string
	if m.card == nil {
		msg := theme.Muted.Render("Select a card to view details")
		content = lipgloss.Place(m.width-4, m.height-4, lipgloss.Center, lipgloss.Center, msg)
	} else {
		content = m.viewport.View()
	}

	return border.Width(m.width - 2).Height(m.height - 2).Render(content)
}

func (m Model) buildContent(width int) string {
	if m.card == nil {
		return ""
	}
	c := m.card

	label := func(k, v string) string {
		return theme.SidebarLabel.Render(k+":") + " " + theme.SidebarValue.Render(v)
	}

	lines := []string{
		theme.Title.Width(width).Render(wordWrap(c.Title, width)),
		strings.Repeat("─", width),
	}

	// Type + number
	typeStr := c.Type
	if c.Number > 0 {
		typeStr = fmt.Sprintf("%s #%d", humanType(c.Type), c.Number)
	}
	lines = append(lines, label("Type", typeStr))

	if c.Repo != "" {
		lines = append(lines, label("Repo", c.Repo))
	}

	lines = append(lines, label("Status", c.Status))

	if c.State != "" {
		stateStyle := theme.StateColor(c.State)
		lines = append(lines, label("State", stateStyle.Render(strings.ToLower(c.State))))
	}

	// Notable field values: the local backend's Priority/Milestone trailers and
	// GitHub custom single-select fields both land in FieldValues.
	for _, k := range []string{"Priority", "Milestone", "Iteration"} {
		if v := c.FieldValues[k]; v != "" {
			lines = append(lines, label(k, v))
		}
	}

	if len(c.Assignees) > 0 {
		lines = append(lines, label("Assigned", strings.Join(c.Assignees, ", ")))
	}

	if len(c.Labels) > 0 {
		var labelParts []string
		for _, l := range c.Labels {
			colored := lipgloss.NewStyle().Foreground(lipgloss.Color("#" + l.Color)).Render(l.Name)
			labelParts = append(labelParts, colored)
		}
		lines = append(lines, theme.SidebarLabel.Render("Labels:")+" "+strings.Join(labelParts, " "))
	}

	if c.URL != "" {
		lines = append(lines, "")
		lines = append(lines, theme.Muted.Render("o  open in browser"))
	}

	if c.Body != "" {
		lines = append(lines, "")
		lines = append(lines, strings.Repeat("─", width))
		lines = append(lines, renderMarkdown(c.Body, width))
	}

	if len(m.history) > 0 {
		lines = append(lines, "")
		lines = append(lines, strings.Repeat("─", width))
		lines = append(lines, theme.SidebarLabel.Render("History"))
		for _, e := range m.history {
			when := e.Date
			if t, err := time.Parse(time.RFC3339, e.Date); err == nil {
				when = t.Format("2006-01-02 15:04")
			}
			line := theme.Muted.Render(when) + "  " + theme.SidebarValue.Render(e.Kind)
			if e.Actor != "" {
				line += " " + theme.Muted.Render("· "+e.Actor)
			}
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}

// renderMarkdown renders an issue body as styled terminal markdown (headings,
// lists, tables, and syntax-highlighted code fences). It falls back to a plain
// word wrap if the renderer cannot be built or the body fails to render, so a
// detail panel never goes blank on malformed input.
func renderMarkdown(s string, width int) string {
	if width <= 0 {
		return s
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return wordWrap(s, width)
	}
	out, err := r.Render(s)
	if err != nil {
		return wordWrap(s, width)
	}
	// glamour frames the document with blank lines; trim them so the body sits
	// flush against the separator above it.
	return strings.Trim(out, "\n")
}

func humanType(t string) string {
	switch t {
	case "ISSUE":
		return "Issue"
	case "PULL_REQUEST":
		return "PR"
	case "DRAFT_ISSUE":
		return "Draft"
	default:
		return t
	}
}

func wordWrap(s string, width int) string {
	if width <= 0 {
		return s
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return s
	}
	var lines []string
	line := ""
	for _, w := range words {
		if len(line)+len(w)+1 > width {
			if line != "" {
				lines = append(lines, line)
				line = w
				continue
			}
		}
		if line == "" {
			line = w
		} else {
			line += " " + w
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
