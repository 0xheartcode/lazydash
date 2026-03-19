package projectlist

import (
	"fmt"
	"strings"

	"github.com/0xheartcode/lazydash/internal/api"
	"github.com/0xheartcode/lazydash/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	projects []api.Project
	cursor   int
	width    int
	height   int
	active   bool
}

func New() Model {
	return Model{}
}

func (m *Model) SetProjects(p []api.Project) {
	m.projects = p
	if m.cursor >= len(p) {
		m.cursor = 0
	}
}

func (m *Model) SetSize(w, h int) { m.width = w; m.height = h }
func (m *Model) SetActive(a bool) { m.active = a }

func (m *Model) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *Model) MoveDown() {
	if m.cursor < len(m.projects)-1 {
		m.cursor++
	}
}

func (m Model) Selected() *api.Project {
	if len(m.projects) == 0 || m.cursor >= len(m.projects) {
		return nil
	}
	p := m.projects[m.cursor]
	return &p
}

func (m Model) Cursor() int { return m.cursor }

func (m Model) View() string {
	border := theme.InactiveBorder
	if m.active {
		border = theme.ActiveBorder
	}

	innerW := m.width - 4 // border + padding
	innerH := m.height - 2

	title := theme.Title.Render("Projects")
	lines := []string{title, theme.Muted.Render(strings.Repeat("─", max(0, innerW)))}

	if len(m.projects) == 0 {
		lines = append(lines, theme.Muted.Render("No projects found"))
	}

	// Simple scroll window
	scrollOffset := 0
	listH := innerH - len(lines)
	if m.cursor >= listH+scrollOffset {
		scrollOffset = m.cursor - listH + 1
	}

	for i, p := range m.projects {
		if i < scrollOffset {
			continue
		}
		if len(lines) >= innerH {
			break
		}
		name := truncate(p.Title, innerW-3)
		var row string
		if i == m.cursor {
			row = theme.CardCursor.Render("> ") + theme.CardSelected.Render(name)
		} else {
			row = "  " + theme.CardTitle.Render(name)
		}
		lines = append(lines, row)
	}

	// Pad to fill height
	for len(lines) < innerH {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")
	return border.Width(m.width - 2).Height(m.height - 2).Render(content)
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// TabTitle returns the display string used in a tab bar.
func (m Model) TabTitle() string {
	if len(m.projects) == 0 {
		return "Projects"
	}
	return fmt.Sprintf("Projects (%d)", len(m.projects))
}

// Render wraps lipgloss width for external callers.
func Width(s string) int { return lipgloss.Width(s) }
