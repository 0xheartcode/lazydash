package board

import (
	"fmt"
	"strings"

	"github.com/0xheartcode/lazydash/internal/api"
	"github.com/0xheartcode/lazydash/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	columns  []api.Column
	colIdx   int
	cardIdx  int
	width    int
	height   int
	active   bool
	views    []api.ProjectView
	viewIdx  int
}

func New() Model {
	return Model{}
}

func (m *Model) SetColumns(cols []api.Column) {
	m.columns = cols
	m.colIdx = 0
	m.cardIdx = 0
}

func (m *Model) SetViews(views []api.ProjectView, idx int) {
	m.views = views
	m.viewIdx = idx
}

func (m *Model) SetSize(w, h int) { m.width = w; m.height = h }
func (m *Model) SetActive(a bool) { m.active = a }

func (m *Model) MoveUp() {
	if m.cardIdx > 0 {
		m.cardIdx--
	}
}

func (m *Model) MoveDown() {
	if len(m.columns) == 0 {
		return
	}
	col := m.columns[m.colIdx]
	if m.cardIdx < len(col.Cards)-1 {
		m.cardIdx++
	}
}

func (m *Model) MoveLeft() {
	if m.colIdx > 0 {
		m.colIdx--
		m.cardIdx = 0
	}
}

func (m *Model) MoveRight() {
	if m.colIdx < len(m.columns)-1 {
		m.colIdx++
		m.cardIdx = 0
	}
}

func (m Model) SelectedCard() *api.Card {
	if len(m.columns) == 0 {
		return nil
	}
	col := m.columns[m.colIdx]
	if len(col.Cards) == 0 || m.cardIdx >= len(col.Cards) {
		return nil
	}
	c := col.Cards[m.cardIdx]
	return &c
}

func (m Model) View() string {
	border := theme.InactiveBorder
	if m.active {
		border = theme.ActiveBorder
	}

	innerW := m.width - 4
	innerH := m.height - 2

	if len(m.columns) == 0 {
		msg := theme.Muted.Render("Select a project to load the board")
		content := lipgloss.Place(innerW, innerH, lipgloss.Center, lipgloss.Center, msg)
		return border.Width(m.width - 2).Height(m.height - 2).Render(content)
	}

	// View tab bar (1 line) + separator (1 line) = 2 lines overhead.
	tabBar := m.renderTabBar(innerW)
	colsH := innerH - 2

	colWidth := innerW / len(m.columns)
	if colWidth < 14 {
		colWidth = 14
	}

	cols := make([]string, len(m.columns))
	for i, col := range m.columns {
		cols[i] = renderColumn(col, i == m.colIdx, m.cardIdx, colWidth, colsH)
	}

	columnsRow := lipgloss.JoinHorizontal(lipgloss.Top, cols...)
	content := lipgloss.JoinVertical(lipgloss.Left, tabBar, columnsRow)
	return border.Width(m.width - 2).Height(m.height - 2).Render(content)
}

func (m Model) renderTabBar(width int) string {
	if len(m.views) <= 1 {
		// Single view — just show its name as a plain header.
		name := ""
		if len(m.views) == 1 {
			name = m.views[0].Name
		}
		return theme.Subtitle.Width(width).Render(name)
	}

	var tabs []string
	for i, v := range m.views {
		label := " " + v.Name + " "
		if i == m.viewIdx {
			tabs = append(tabs, theme.CardSelected.
				Background(theme.ColorBg).
				Underline(true).
				Render(label))
		} else {
			tabs = append(tabs, theme.Muted.Render(label))
		}
	}

	bar := strings.Join(tabs, theme.Muted.Render("│"))
	hint := theme.Muted.Render("  [ / ] switch")
	gap := width - lipgloss.Width(bar) - lipgloss.Width(hint)
	if gap < 0 {
		gap = 0
	}
	return bar + strings.Repeat(" ", gap) + hint
}

func renderColumn(col api.Column, active bool, selectedCard, width, height int) string {
	cardW := width - 2

	header := theme.ColumnHeader.Width(cardW).Render(
		truncate(col.Name, cardW) + fmt.Sprintf(" (%d)", len(col.Cards)),
	)

	lines := []string{header}
	maxCards := height - 3

	for i, card := range col.Cards {
		if len(lines)-1 >= maxCards {
			break
		}
		name := truncate(card.Title, cardW-3)
		var line string
		if active && i == selectedCard {
			line = theme.CardCursor.Render("> ") + theme.CardSelected.Render(name)
		} else {
			line = "  " + theme.CardTitle.Render(name)
		}
		lines = append(lines, line)
	}

	if len(col.Cards) == 0 {
		lines = append(lines, theme.Muted.Render("  empty"))
	}

	for len(lines) < height-1 {
		lines = append(lines, "")
	}

	colStyle := lipgloss.NewStyle().Width(width).PaddingRight(1)
	if active {
		colStyle = colStyle.BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(theme.ColorBorder)
	}

	return colStyle.Render(strings.Join(lines, "\n"))
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
