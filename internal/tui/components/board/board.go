package board

import (
	"fmt"
	"strings"

	"github.com/0xheartcode/lazydash/internal/api"
	"github.com/0xheartcode/lazydash/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	// shared
	width   int
	height  int
	active  bool
	views   []api.ProjectView
	viewIdx int
	layout  string // BOARD_LAYOUT | TABLE_LAYOUT

	// board mode
	columns []api.Column
	colIdx  int
	cardIdx int

	// table mode
	tableItems  []api.Card
	tableRow    int
	tableScroll int
}

func New() Model { return Model{layout: "BOARD_LAYOUT"} }

// --- Setters ---

func (m *Model) SetLayout(layout string)               { m.layout = layout }
func (m *Model) SetSize(w, h int)                      { m.width = w; m.height = h }
func (m *Model) SetActive(a bool)                      { m.active = a }
func (m *Model) SetViews(views []api.ProjectView, idx int) { m.views = views; m.viewIdx = idx }

func (m *Model) SetColumns(cols []api.Column) {
	m.columns = cols
	m.colIdx = 0
	m.cardIdx = 0
}

func (m *Model) SetTableItems(items []api.Card) {
	m.tableItems = items
	m.tableRow = 0
	m.tableScroll = 0
}

// --- Navigation ---

func (m *Model) MoveUp() {
	if m.layout == "TABLE_LAYOUT" {
		if m.tableRow > 0 {
			m.tableRow--
			if m.tableRow < m.tableScroll {
				m.tableScroll = m.tableRow
			}
		}
		return
	}
	if m.cardIdx > 0 {
		m.cardIdx--
	}
}

func (m *Model) MoveDown() {
	if m.layout == "TABLE_LAYOUT" {
		if m.tableRow < len(m.tableItems)-1 {
			m.tableRow++
			visible := m.tableVisibleRows()
			if m.tableRow >= m.tableScroll+visible {
				m.tableScroll = m.tableRow - visible + 1
			}
		}
		return
	}
	if len(m.columns) == 0 {
		return
	}
	col := m.columns[m.colIdx]
	if m.cardIdx < len(col.Cards)-1 {
		m.cardIdx++
	}
}

func (m *Model) MoveLeft() {
	if m.layout == "TABLE_LAYOUT" {
		return
	}
	if m.colIdx > 0 {
		m.colIdx--
		m.cardIdx = 0
	}
}

func (m *Model) MoveRight() {
	if m.layout == "TABLE_LAYOUT" {
		return
	}
	if m.colIdx < len(m.columns)-1 {
		m.colIdx++
		m.cardIdx = 0
	}
}

func (m Model) SelectedCard() *api.Card {
	if m.layout == "TABLE_LAYOUT" {
		if len(m.tableItems) == 0 || m.tableRow >= len(m.tableItems) {
			return nil
		}
		c := m.tableItems[m.tableRow]
		return &c
	}
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

// --- View ---

func (m Model) View() string {
	border := theme.InactiveBorder
	if m.active {
		border = theme.ActiveBorder
	}
	innerW := m.width - 4
	innerH := m.height - 2

	var content string
	if m.layout == "TABLE_LAYOUT" {
		content = m.renderTable(innerW, innerH)
	} else {
		content = m.renderBoard(innerW, innerH)
	}

	return border.Width(m.width - 2).Height(m.height - 2).Render(content)
}

// --- Table rendering ---

func (m Model) tableVisibleRows() int {
	h := m.height - 2 - 3 // border + tab bar + header row + separator
	if h < 1 {
		h = 1
	}
	return h
}

func (m Model) renderTable(innerW, innerH int) string {
	tabBar := m.renderTabBar(innerW)

	statusW := 14
	assignW := 14
	numW := 5
	iconW := 2
	cursorW := 2
	titleW := innerW - cursorW - iconW - numW - 1 - statusW - 1 - assignW
	if titleW < 10 {
		titleW = 10
	}

	sep := theme.Muted.Render(strings.Repeat("─", innerW))
	header := lipgloss.JoinHorizontal(lipgloss.Top,
		strings.Repeat(" ", cursorW+iconW),
		theme.Subtitle.Width(numW+1).Render("#"),
		theme.Subtitle.Width(titleW).Render("Title"),
		theme.Subtitle.Width(statusW+1).Render("Status"),
		theme.Subtitle.Width(assignW).Render("Assigned"),
	)

	lines := []string{tabBar, header, sep}

	if len(m.tableItems) == 0 {
		lines = append(lines, theme.Muted.Render("  No items"))
		return strings.Join(lines, "\n")
	}

	visibleRows := innerH - len(lines)
	if visibleRows < 1 {
		visibleRows = 1
	}

	end := m.tableScroll + visibleRows
	if end > len(m.tableItems) {
		end = len(m.tableItems)
	}

	for i := m.tableScroll; i < end; i++ {
		card := m.tableItems[i]
		cursor := "  "
		if i == m.tableRow {
			cursor = theme.CardCursor.Render("> ")
		}

		icon := itemIcon(card.Type, card.State)
		num := theme.Muted.Width(numW + 1).Render(fmt.Sprintf("#%-4d", card.Number))
		if card.Number == 0 {
			num = strings.Repeat(" ", numW+1)
		}

		title := truncate(card.Title, titleW)
		var titleStr string
		if i == m.tableRow {
			titleStr = theme.CardSelected.Width(titleW).Render(title)
		} else {
			titleStr = theme.CardTitle.Width(titleW).Render(title)
		}

		statusStr := lipgloss.NewStyle().Width(statusW + 1).Render(
			statusBadge(card.Status, statusW),
		)

		assignStr := ""
		if len(card.Assignees) > 0 {
			assignStr = truncate(strings.Join(card.Assignees, " "), assignW)
		}
		assignStr = theme.Muted.Width(assignW).Render(assignStr)

		row := cursor + icon + num + titleStr + statusStr + assignStr
		lines = append(lines, row)
	}

	// Scroll indicator
	if len(m.tableItems) > visibleRows {
		pct := 0
		if len(m.tableItems) > 1 {
			pct = m.tableRow * 100 / (len(m.tableItems) - 1)
		}
		scrollInfo := theme.Muted.Render(fmt.Sprintf("  %d/%d (%d%%)", m.tableRow+1, len(m.tableItems), pct))
		lines = append(lines, scrollInfo)
	}

	return strings.Join(lines, "\n")
}

func itemIcon(itemType, state string) string {
	switch itemType {
	case "ISSUE":
		if strings.ToUpper(state) == "CLOSED" {
			return theme.StatusClosed.Render("● ")
		}
		return theme.StatusOpen.Render("○ ")
	case "PULL_REQUEST":
		if strings.ToUpper(state) == "MERGED" {
			return theme.StatusMerged.Render("⎇ ")
		}
		if strings.ToUpper(state) == "CLOSED" {
			return theme.StatusClosed.Render("⎇ ")
		}
		return theme.StatusOpen.Render("⎇ ")
	case "DRAFT_ISSUE":
		return theme.StatusDraft.Render("◌ ")
	default:
		return "  "
	}
}

func statusBadge(status string, width int) string {
	if status == "" {
		return strings.Repeat(" ", width)
	}
	s := truncate(status, width)
	lower := strings.ToLower(status)
	switch {
	case strings.Contains(lower, "done") || strings.Contains(lower, "complete") || strings.Contains(lower, "closed"):
		return theme.StatusClosed.Render(s)
	case strings.Contains(lower, "progress") || strings.Contains(lower, "active") || strings.Contains(lower, "started"):
		return theme.StatusOpen.Render(s)
	case strings.Contains(lower, "review") || strings.Contains(lower, "planning"):
		return theme.StatusMerged.Render(s)
	default:
		return theme.Muted.Render(s)
	}
}

// --- Board rendering ---

func (m Model) renderBoard(innerW, innerH int) string {
	if len(m.columns) == 0 {
		msg := theme.Muted.Render("Select a project to load the board")
		return lipgloss.Place(innerW, innerH, lipgloss.Center, lipgloss.Center, msg)
	}

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

	return lipgloss.JoinVertical(lipgloss.Left,
		tabBar,
		lipgloss.JoinHorizontal(lipgloss.Top, cols...),
	)
}

func (m Model) renderTabBar(width int) string {
	if len(m.views) <= 1 {
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
			tabs = append(tabs, theme.CardSelected.Underline(true).Render(label))
		} else {
			tabs = append(tabs, theme.Muted.Render(label))
		}
	}

	bar := strings.Join(tabs, theme.Muted.Render("│"))
	hint := theme.Muted.Render("  [ / ]")
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
