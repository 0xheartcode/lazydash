package board

import (
	"fmt"
	"strings"

	"github.com/0xheartcode/lazydash/internal/core"
	"github.com/0xheartcode/lazydash/internal/tui/theme"
	"github.com/0xheartcode/lazydash/internal/utils"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	// shared
	width   int
	height  int
	active  bool
	views   []core.ProjectView
	viewIdx int
	layout  string // BOARD_LAYOUT | TABLE_LAYOUT

	// board mode
	columns []core.Column
	colIdx  int
	cardIdx int

	// table mode
	tableItems    []core.Card
	tableRow      int
	tableScroll   int
	visibleFields []core.VisibleField
	optionColors  map[string]map[string]string // fieldName → optionName → terminal color
}

func New() Model { return Model{layout: "BOARD_LAYOUT"} }

// --- Setters ---

func (m *Model) SetLayout(layout string)                    { m.layout = layout }
func (m *Model) SetSize(w, h int)                           { m.width = w; m.height = h }
func (m *Model) SetActive(a bool)                           { m.active = a }
func (m *Model) SetViews(views []core.ProjectView, idx int) { m.views = views; m.viewIdx = idx }

func (m *Model) SetColumns(cols []core.Column) {
	m.columns = cols
	m.colIdx = 0
	m.cardIdx = 0
}

func (m *Model) SetTableItems(items []core.Card) {
	m.tableItems = items
	m.tableRow = 0
	m.tableScroll = 0
}

func (m *Model) SetVisibleFields(fields []core.VisibleField)         { m.visibleFields = fields }
func (m *Model) SetOptionColors(colors map[string]map[string]string) { m.optionColors = colors }

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

func (m Model) SelectedCard() *core.Card {
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

// tableColDef describes a dynamic column in table mode.
type tableColDef struct {
	name     string
	dataType string
}

func (m Model) tableColumns() (cols []tableColDef, showAssignees bool) {
	if len(m.visibleFields) == 0 {
		// Default fallback: Status + Assignees.
		return []tableColDef{{name: "Status", dataType: "SINGLE_SELECT"}}, true
	}
	for _, vf := range m.visibleFields {
		switch vf.DataType {
		case "TITLE":
			// always rendered as first column
		case "ASSIGNEES", "REVIEWERS":
			showAssignees = true
		case "REPOSITORY":
			cols = append(cols, tableColDef{name: "Repo", dataType: vf.DataType})
		default:
			cols = append(cols, tableColDef{name: vf.Name, dataType: vf.DataType})
		}
	}
	return cols, showAssignees
}

func (m Model) renderTable(innerW, innerH int) string {
	tabBar := m.renderTabBar(innerW)

	const cursorW, iconW, numW, colW = 2, 2, 5, 12

	midCols, showAssignees := m.tableColumns()

	assignW := 0
	if showAssignees {
		assignW = 14
	}
	fixed := cursorW + iconW + (numW + 1)
	extras := len(midCols)*(colW+1) + assignW
	titleW := innerW - fixed - extras
	if titleW < 10 {
		titleW = 10
	}

	sep := theme.Muted.Render(strings.Repeat("─", innerW))
	headerParts := []string{
		strings.Repeat(" ", cursorW+iconW),
		theme.Subtitle.Width(numW + 1).Render("#"),
		theme.Subtitle.Width(titleW).Render("Title"),
	}
	for _, c := range midCols {
		headerParts = append(headerParts, theme.Subtitle.Width(colW+1).Render(c.name))
	}
	if showAssignees {
		headerParts = append(headerParts, theme.Subtitle.Width(assignW).Render("Assigned"))
	}
	header := lipgloss.JoinHorizontal(lipgloss.Top, headerParts...)

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

		title := utils.Truncate(card.Title, titleW)
		var titleStr string
		if i == m.tableRow {
			titleStr = theme.CardSelected.Width(titleW).Render(title)
		} else {
			titleStr = theme.CardTitle.Width(titleW).Render(title)
		}

		rowParts := []string{cursor, icon, num, titleStr}
		for _, c := range midCols {
			var cellStr string
			switch c.dataType {
			case "REPOSITORY":
				cellStr = lipgloss.NewStyle().Width(colW + 1).Render(
					renderCell(card.Repo, c.dataType, "", colW),
				)
			case "LABELS":
				cellStr = lipgloss.NewStyle().Width(colW + 1).Render(
					renderLabels(card.Labels, colW),
				)
			default:
				var val string
				if card.FieldValues != nil {
					val = card.FieldValues[c.name]
				}
				optColor := ""
				if m.optionColors != nil {
					optColor = m.optionColors[c.name][val]
				}
				cellStr = lipgloss.NewStyle().Width(colW + 1).Render(
					renderCell(val, c.dataType, optColor, colW),
				)
			}
			rowParts = append(rowParts, cellStr)
		}
		if showAssignees {
			assignStr := ""
			if len(card.Assignees) > 0 {
				assignStr = utils.Truncate(strings.Join(card.Assignees, " "), assignW)
			}
			rowParts = append(rowParts, theme.Muted.Width(assignW).Render(assignStr))
		}
		lines = append(lines, strings.Join(rowParts, ""))
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

func renderLabels(labels []core.Label, width int) string {
	if len(labels) == 0 {
		return strings.Repeat(" ", width)
	}
	var parts []string
	remaining := width
	for _, l := range labels {
		if remaining <= 0 {
			break
		}
		name := l.Name
		if len(name) > remaining {
			name = name[:remaining]
		}
		color := "#" + l.Color
		badge := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(name)
		parts = append(parts, badge)
		remaining -= len(name) + 1 // +1 for the space separator
	}
	result := strings.Join(parts, " ")
	// Pad to width using visible length
	visLen := lipgloss.Width(result)
	if visLen < width {
		result += strings.Repeat(" ", width-visLen)
	}
	return result
}

func renderCell(val, dataType, optColor string, width int) string {
	if val == "" {
		return strings.Repeat(" ", width)
	}
	switch dataType {
	case "SINGLE_SELECT":
		if optColor != "" {
			return lipgloss.NewStyle().Foreground(lipgloss.Color(optColor)).Render(utils.Truncate(val, width))
		}
		return statusBadge(val, width)
	default:
		return theme.Muted.Width(width).Render(utils.Truncate(val, width))
	}
}

func statusBadge(status string, width int) string {
	if status == "" {
		return strings.Repeat(" ", width)
	}
	s := utils.Truncate(status, width)
	lower := strings.ToLower(status)
	switch {
	// Status values
	case strings.Contains(lower, "done") || strings.Contains(lower, "complete") || strings.Contains(lower, "closed"):
		return theme.StatusClosed.Render(s)
	case strings.Contains(lower, "progress") || strings.Contains(lower, "active") || strings.Contains(lower, "started"):
		return theme.StatusOpen.Render(s)
	case strings.Contains(lower, "review") || strings.Contains(lower, "planning"):
		return theme.StatusMerged.Render(s)
	// Type values
	case lower == "bug":
		return lipgloss.NewStyle().Foreground(theme.ColorDanger).Render(s)
	case lower == "feature":
		return lipgloss.NewStyle().Foreground(theme.ColorPrimary).Render(s)
	case lower == "chore":
		return lipgloss.NewStyle().Foreground(theme.ColorMuted).Render(s)
	case lower == "ci":
		return lipgloss.NewStyle().Foreground(theme.ColorSecondary).Render(s)
	// Effort values
	case strings.Contains(lower, "quick"):
		return lipgloss.NewStyle().Foreground(theme.ColorSuccess).Render(s)
	case lower == "medium":
		return lipgloss.NewStyle().Foreground(theme.ColorWarning).Render(s)
	case lower == "major":
		return lipgloss.NewStyle().Foreground(theme.ColorDanger).Bold(true).Render(s)
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

func renderColumn(col core.Column, active bool, selectedCard, width, height int) string {
	cardW := width - 2
	header := theme.ColumnHeader.Width(cardW).Render(
		utils.Truncate(col.Name, cardW) + fmt.Sprintf(" (%d)", len(col.Cards)),
	)
	lines := []string{header}
	maxCards := height - 3

	for i, card := range col.Cards {
		if len(lines)-1 >= maxCards {
			break
		}
		name := utils.Truncate(card.Title, cardW-3)
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
