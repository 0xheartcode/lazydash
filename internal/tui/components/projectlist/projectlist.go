package projectlist

import (
	"fmt"
	"strings"

	"github.com/0xheartcode/lazydash/internal/core"
	"github.com/0xheartcode/lazydash/internal/tui/theme"
	"github.com/0xheartcode/lazydash/internal/utils"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	projects []core.Project
	cursor   int
	width    int
	height   int
	active   bool
}

func New() Model {
	return Model{}
}

func (m *Model) SetProjects(p []core.Project) {
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

func (m Model) Selected() *core.Project {
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

	// Show a source badge only when projects come from more than one backend,
	// so a single-source list stays uncluttered.
	badges := hasMultipleSources(m.projects)

	for i, p := range m.projects {
		if i < scrollOffset {
			continue
		}
		if len(lines) >= innerH {
			break
		}
		label := p.Title
		if p.Owner != "" {
			label = p.Owner + "/" + p.Title
		}
		badge := ""
		avail := innerW - 3
		if badges {
			badge = sourceBadge(p.Source)
			avail -= lipgloss.Width(badge)
		}
		if avail < 1 {
			avail = 1
		}
		name := utils.Truncate(label, avail)
		var row string
		if i == m.cursor {
			row = theme.CardCursor.Render("> ") + badge + theme.CardSelected.Render(name)
		} else {
			row = "  " + badge + theme.CardTitle.Render(name)
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

// TabTitle returns the display string used in a tab bar.
func (m Model) TabTitle() string {
	if len(m.projects) == 0 {
		return "Projects"
	}
	return fmt.Sprintf("Projects (%d)", len(m.projects))
}

// Render wraps lipgloss width for external callers.
func Width(s string) int { return lipgloss.Width(s) }

// hasMultipleSources reports whether the projects span more than one backend.
func hasMultipleSources(projects []core.Project) bool {
	seen := ""
	for _, p := range projects {
		if p.Source == "" {
			continue
		}
		if seen == "" {
			seen = p.Source
		} else if p.Source != seen {
			return true
		}
	}
	return false
}

// sourceBadge renders a fixed-width, colored tag for a project's backend so
// names stay aligned regardless of which source a row came from.
func sourceBadge(src string) string {
	var text string
	var color lipgloss.Color
	switch src {
	case "local":
		text, color = "local", theme.ColorSuccess
	case "github":
		text, color = "gh", theme.ColorSecondary
	default:
		return strings.Repeat(" ", 6)
	}
	if len(text) < 5 {
		text += strings.Repeat(" ", 5-len(text))
	}
	return lipgloss.NewStyle().Foreground(color).Render(text) + " "
}
