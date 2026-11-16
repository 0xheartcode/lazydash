package board

import (
	"fmt"
	"strings"
	"time"

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

	// activity + calendar modes
	events      []core.Event
	eventCursor int
	calCursor   time.Time // selected day in the calendar grid

	// roadmap mode
	roadmapCards      []core.Card // dated items first, then undated, in source order within each group
	roadmapStart      string      // name of the start date field ("" if none)
	roadmapTarget     string      // name of the target/end date field ("" if none)
	roadmapDatedCount int         // how many leading roadmapCards carry a usable date
	roadmapRow        int         // selected row across the whole item list
	roadmapScroll     int         // first visible row
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

// SetEvents loads the op-log for the Activity and Calendar views. The feed
// cursor resets to the newest event, and the calendar's selected day snaps to
// the newest event's date (or today when the log is empty).
func (m *Model) SetEvents(events []core.Event) {
	m.events = events
	m.eventCursor = 0
	m.calCursor = time.Now()
	if len(events) > 0 {
		if t, err := time.Parse(time.RFC3339, events[0].Date); err == nil {
			m.calCursor = t
		}
	}
}

// SetRoadmap loads the roadmap view with its flat item list and the names of the
// view's date fields. When two date fields are present, startField is the start
// date and targetField is the target/end date; when one is present it is a single
// marker date. Items are ordered dated-first (in source order), then the undated
// items, so the row cursor and the No-date section stay in step.
func (m *Model) SetRoadmap(cards []core.Card, startField, targetField string) {
	m.roadmapStart = startField
	m.roadmapTarget = targetField
	var dated, undated []core.Card
	for _, c := range cards {
		if m.hasRoadmapDate(c) {
			dated = append(dated, c)
		} else {
			undated = append(undated, c)
		}
	}
	m.roadmapCards = append(dated, undated...)
	m.roadmapDatedCount = len(dated)
	m.roadmapRow = 0
	m.roadmapScroll = 0
}

// --- Navigation ---

func (m *Model) MoveUp() {
	switch m.layout {
	case "TABLE_LAYOUT":
		if m.tableRow > 0 {
			m.tableRow--
			if m.tableRow < m.tableScroll {
				m.tableScroll = m.tableRow
			}
		}
		return
	case "ACTIVITY_LAYOUT":
		if m.eventCursor > 0 {
			m.eventCursor--
		}
		return
	case "CALENDAR_LAYOUT":
		m.calCursor = m.calCursor.AddDate(0, 0, -7)
		return
	case "ROADMAP_LAYOUT":
		if m.roadmapRow > 0 {
			m.roadmapRow--
			if m.roadmapRow < m.roadmapScroll {
				m.roadmapScroll = m.roadmapRow
			}
		}
		return
	}
	if m.cardIdx > 0 {
		m.cardIdx--
	}
}

func (m *Model) MoveDown() {
	switch m.layout {
	case "TABLE_LAYOUT":
		if m.tableRow < len(m.tableItems)-1 {
			m.tableRow++
			visible := m.tableVisibleRows()
			if m.tableRow >= m.tableScroll+visible {
				m.tableScroll = m.tableRow - visible + 1
			}
		}
		return
	case "ACTIVITY_LAYOUT":
		if m.eventCursor < len(m.events)-1 {
			m.eventCursor++
		}
		return
	case "CALENDAR_LAYOUT":
		m.calCursor = m.calCursor.AddDate(0, 0, 7)
		return
	case "ROADMAP_LAYOUT":
		if m.roadmapRow < len(m.roadmapCards)-1 {
			m.roadmapRow++
			visible := m.roadmapVisibleRows()
			if m.roadmapRow >= m.roadmapScroll+visible {
				m.roadmapScroll = m.roadmapRow - visible + 1
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
	switch m.layout {
	case "TABLE_LAYOUT", "ACTIVITY_LAYOUT", "ROADMAP_LAYOUT":
		// Roadmap auto-fits the whole date range to the width, so there is
		// nothing to pan horizontally.
		return
	case "CALENDAR_LAYOUT":
		m.calCursor = m.calCursor.AddDate(0, 0, -1)
		return
	}
	if m.colIdx > 0 {
		m.colIdx--
		m.cardIdx = 0
	}
}

func (m *Model) MoveRight() {
	switch m.layout {
	case "TABLE_LAYOUT", "ACTIVITY_LAYOUT", "ROADMAP_LAYOUT":
		return
	case "CALENDAR_LAYOUT":
		m.calCursor = m.calCursor.AddDate(0, 0, 1)
		return
	}
	if m.colIdx < len(m.columns)-1 {
		m.colIdx++
		m.cardIdx = 0
	}
}

// Layout reports the current layout ("BOARD_LAYOUT" or "TABLE_LAYOUT").
func (m Model) Layout() string { return m.layout }

// Counts returns the number of columns and the total number of cards. In table
// layout columns is 0 and cards is the row count.
func (m Model) Counts() (columns, cards int) {
	if m.layout == "TABLE_LAYOUT" {
		return 0, len(m.tableItems)
	}
	if m.layout == "ROADMAP_LAYOUT" {
		return 0, len(m.roadmapCards)
	}
	for _, c := range m.columns {
		cards += len(c.Cards)
	}
	return len(m.columns), cards
}

// EventCount reports how many op-log events are loaded, for the snapshot seam.
func (m Model) EventCount() int { return len(m.events) }

// SelectedEvent returns the event under the feed cursor in the Activity view, or
// nil in any other layout.
func (m Model) SelectedEvent() *core.Event {
	if m.layout != "ACTIVITY_LAYOUT" {
		return nil
	}
	if m.eventCursor < 0 || m.eventCursor >= len(m.events) {
		return nil
	}
	e := m.events[m.eventCursor]
	return &e
}

// SelectedDay returns the selected calendar day as YYYY-MM-DD in the Calendar
// view, or "" in any other layout.
func (m Model) SelectedDay() string {
	if m.layout != "CALENDAR_LAYOUT" || m.calCursor.IsZero() {
		return ""
	}
	return m.calCursor.Format("2006-01-02")
}

// RoadmapCount reports how many items are on the roadmap, for the snapshot seam.
func (m Model) RoadmapCount() int { return len(m.roadmapCards) }

// RoadmapUndatedCount reports how many roadmap items have no usable date and so
// land in the No-date section, for the snapshot seam.
func (m Model) RoadmapUndatedCount() int { return len(m.roadmapCards) - m.roadmapDatedCount }

// RoadmapWindow returns the roadmap's time axis as start and end months
// (YYYY-MM), or two empty strings when no item carries a usable date.
func (m Model) RoadmapWindow() (start, end string) {
	min, max, ok := m.roadmapRange()
	if !ok {
		return "", ""
	}
	return min.Format("2006-01"), max.Format("2006-01")
}

func (m Model) SelectedCard() *core.Card {
	if m.layout == "TABLE_LAYOUT" {
		if len(m.tableItems) == 0 || m.tableRow >= len(m.tableItems) {
			return nil
		}
		c := m.tableItems[m.tableRow]
		return &c
	}
	if m.layout == "ROADMAP_LAYOUT" {
		if len(m.roadmapCards) == 0 || m.roadmapRow < 0 || m.roadmapRow >= len(m.roadmapCards) {
			return nil
		}
		c := m.roadmapCards[m.roadmapRow]
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
	switch m.layout {
	case "TABLE_LAYOUT":
		content = m.renderTable(innerW, innerH)
	case "ACTIVITY_LAYOUT":
		content = m.renderActivity(innerW, innerH)
	case "CALENDAR_LAYOUT":
		content = m.renderCalendar(innerW, innerH)
	case "ROADMAP_LAYOUT":
		content = m.renderRoadmap(innerW, innerH)
	default:
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

// --- Activity rendering ---

// renderActivity draws the op-log as a reverse-chronological, scrollable list,
// one line per event, with a cursor driven by j/k.
func (m Model) renderActivity(innerW, innerH int) string {
	tabBar := m.renderTabBar(innerW)
	lines := []string{tabBar}

	if len(m.events) == 0 {
		lines = append(lines, "", theme.Muted.Render("  No activity recorded yet"))
		return strings.Join(lines, "\n")
	}

	visible := innerH - 1
	if visible < 1 {
		visible = 1
	}
	scroll := 0
	if m.eventCursor >= visible {
		scroll = m.eventCursor - visible + 1
	}
	end := scroll + visible
	if end > len(m.events) {
		end = len(m.events)
	}

	const dateW, kindW, actorW = 17, 12, 14
	titleW := innerW - 2 - dateW - kindW - actorW - 3
	if titleW < 10 {
		titleW = 10
	}

	for i := scroll; i < end; i++ {
		e := m.events[i]
		cursor := "  "
		title := theme.CardTitle.Render(utils.Truncate(e.IssueTitle, titleW))
		if i == m.eventCursor {
			cursor = theme.CardCursor.Render("> ")
			title = theme.CardSelected.Render(utils.Truncate(e.IssueTitle, titleW))
		}
		date := theme.Muted.Width(dateW).Render(utils.Truncate(shortDateTime(e.Date), dateW))
		kind := eventKindStyle(e.Kind).Width(kindW).Render(utils.Truncate(e.Kind, kindW))
		actor := theme.Muted.Width(actorW).Render(utils.Truncate(e.Actor, actorW))
		lines = append(lines, cursor+date+kind+lipgloss.NewStyle().Width(titleW+1).Render(title)+actor)
	}

	if len(m.events) > visible {
		pct := 0
		if len(m.events) > 1 {
			pct = m.eventCursor * 100 / (len(m.events) - 1)
		}
		lines = append(lines, theme.Muted.Render(fmt.Sprintf("  %d/%d (%d%%)", m.eventCursor+1, len(m.events), pct)))
	}

	return strings.Join(lines, "\n")
}

// --- Calendar rendering ---

// renderCalendar draws the selected day's month as a grid (weeks as rows, days
// as cells with a per-day activity marker), with the selected day highlighted,
// and lists that day's events below the grid. h/l/j/k move the selected day and
// roll across months at the edges.
func (m Model) renderCalendar(innerW, innerH int) string {
	tabBar := m.renderTabBar(innerW)

	cur := m.calCursor
	if cur.IsZero() {
		cur = time.Now()
	}

	counts := map[string]int{}
	for _, e := range m.events {
		if t, err := time.Parse(time.RFC3339, e.Date); err == nil {
			counts[t.Format("2006-01-02")]++
		}
	}

	year, month := cur.Year(), cur.Month()
	loc := cur.Location()
	first := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	startOffset := int(first.Weekday()) // Sunday = 0
	daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()

	header := []string{
		theme.Title.Render(cur.Format("January 2006")),
		theme.Subtitle.Render(" Su  Mo  Tu  We  Th  Fr  Sa"),
	}

	var rows []string
	var week []string
	for i := 0; i < startOffset; i++ {
		week = append(week, "    ")
	}
	for day := 1; day <= daysInMonth; day++ {
		d := time.Date(year, month, day, 0, 0, 0, 0, loc)
		week = append(week, dayCell(day, counts[d.Format("2006-01-02")], sameDay(d, cur)))
		if len(week) == 7 {
			rows = append(rows, strings.Join(week, ""))
			week = nil
		}
	}
	if len(week) > 0 {
		rows = append(rows, strings.Join(week, ""))
	}

	grid := lipgloss.JoinVertical(lipgloss.Left, append(header, rows...)...)

	// Selected-day panel.
	selKey := cur.Format("2006-01-02")
	panel := []string{
		theme.Subtitle.Render(cur.Format("Mon, Jan 2 2006")),
		theme.Muted.Render(strings.Repeat("─", 28)),
	}
	shown := 0
	for _, e := range m.events {
		t, err := time.Parse(time.RFC3339, e.Date)
		if err != nil || t.Format("2006-01-02") != selKey {
			continue
		}
		kind := eventKindStyle(e.Kind).Render(utils.Truncate(e.Kind, 12))
		panel = append(panel, "  "+kind+" "+theme.CardTitle.Render(utils.Truncate(e.IssueTitle, 28)))
		shown++
	}
	if shown == 0 {
		panel = append(panel, theme.Muted.Render("  No activity this day"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, tabBar, grid, "", strings.Join(panel, "\n"))
}

// dayCell renders one calendar day as a 4-cell: a leading marker column, the
// day number (two columns), and an activity dot. The selected day shows a cursor
// marker and is styled, keeping the same visible width.
func dayCell(day, count int, selected bool) string {
	dot := " "
	if count > 0 {
		dot = "*"
	}
	body := fmt.Sprintf("%2d%s", day, dot) // three columns
	if selected {
		return theme.CardCursor.Render(">") + theme.CardSelected.Render(body)
	}
	return " " + body
}

// sameDay reports whether a and b fall on the same calendar day.
func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// shortDateTime renders an ISO-8601 timestamp as "2006-01-02 15:04", falling
// back to the raw string (or its date prefix) when it does not parse.
func shortDateTime(iso string) string {
	if t, err := time.Parse(time.RFC3339, iso); err == nil {
		return t.Format("2006-01-02 15:04")
	}
	if len(iso) >= 10 {
		return iso[:10]
	}
	return iso
}

// eventKindStyle colors an event kind label by its sentiment, reusing the shared
// status palette so the feed stays consistent with the rest of the board.
func eventKindStyle(kind string) lipgloss.Style {
	switch kind {
	case "Created", "Reopened":
		return theme.StatusOpen
	case "Closed":
		return theme.StatusClosed
	case "Commented":
		return theme.StatusMerged
	default:
		return theme.Muted
	}
}

// --- Roadmap rendering ---

// roadmapDate parses the ISO date ("2006-01-02") held in a card's named field,
// reporting whether the value was present and parseable.
func (m Model) roadmapDate(c core.Card, field string) (time.Time, bool) {
	if field == "" || c.FieldValues == nil {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", c.FieldValues[field])
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// hasRoadmapDate reports whether a card carries any usable roadmap date, so it
// can be placed on the timeline rather than listed in the No-date section.
func (m Model) hasRoadmapDate(c core.Card) bool {
	_, s := m.roadmapDate(c, m.roadmapStart)
	_, e := m.roadmapDate(c, m.roadmapTarget)
	return s || e
}

// roadmapRange returns the earliest start and latest target across all dated
// items, reporting false when nothing is dated. A zero-width span is widened by
// one day so the axis never collapses.
func (m Model) roadmapRange() (min, max time.Time, ok bool) {
	for i := 0; i < m.roadmapDatedCount && i < len(m.roadmapCards); i++ {
		c := m.roadmapCards[i]
		for _, field := range []string{m.roadmapStart, m.roadmapTarget} {
			t, has := m.roadmapDate(c, field)
			if !has {
				continue
			}
			if !ok {
				min, max, ok = t, t, true
				continue
			}
			if t.Before(min) {
				min = t
			}
			if t.After(max) {
				max = t
			}
		}
	}
	if ok && !max.After(min) {
		max = min.AddDate(0, 0, 1)
	}
	return min, max, ok
}

// roadmapVisibleRows is the number of item rows that fit under the roadmap's
// tab bar, month header and axis separator.
func (m Model) roadmapVisibleRows() int {
	h := m.height - 2 - 3 // border + tab bar + month header + separator
	if h < 1 {
		h = 1
	}
	return h
}

// scaleCol maps a time onto a zero-based column in a bar of the given width,
// clamped to the bar's bounds.
func scaleCol(t, min, max time.Time, width int) int {
	if width <= 1 {
		return 0
	}
	span := max.Sub(min)
	if span <= 0 {
		return 0
	}
	c := int(float64(t.Sub(min)) / float64(span) * float64(width-1))
	if c < 0 {
		return 0
	}
	if c > width-1 {
		return width - 1
	}
	return c
}

// placeInto writes s into row starting at col, ignoring anything out of bounds.
func placeInto(row []rune, col int, s string) {
	for i, r := range []rune(s) {
		p := col + i
		if p < 0 || p >= len(row) {
			continue
		}
		row[p] = r
	}
}

// renderRoadmap draws a horizontal timeline: a left title column per item and a
// bar spanning its start to its target date, with a month-labelled axis scaled
// to fit the whole range into the available width. Items with no usable date are
// listed in a No-date section at the bottom.
func (m Model) renderRoadmap(innerW, innerH int) string {
	tabBar := m.renderTabBar(innerW)

	if len(m.roadmapCards) == 0 {
		return strings.Join([]string{tabBar, "", theme.Muted.Render("  No items")}, "\n")
	}

	const cursorW, iconW = 2, 2
	titleW := innerW / 3
	if titleW < 8 {
		titleW = 8
	}
	if titleW > 30 {
		titleW = 30
	}
	barWidth := innerW - cursorW - iconW - titleW - 1
	if barWidth < 1 {
		barWidth = 1
	}

	min, max, dated := m.roadmapRange()
	lines := []string{tabBar}

	// Month-labelled header and axis separator, aligned over the bar area.
	leftPad := strings.Repeat(" ", cursorW+iconW+titleW+1)
	if dated {
		header := make([]rune, barWidth)
		for i := range header {
			header[i] = ' '
		}
		mt := time.Date(min.Year(), min.Month(), 1, 0, 0, 0, 0, min.Location())
		for !mt.After(max) {
			label := mt.Format("Jan")
			if mt.Month() == time.January {
				label = mt.Format("Jan 06")
			}
			placeInto(header, scaleCol(mt, min, max, barWidth), label)
			mt = mt.AddDate(0, 1, 0)
		}
		lines = append(lines, leftPad+theme.Subtitle.Render(string(header)))
	} else {
		lines = append(lines, theme.Muted.Render("  No dated items to place on the timeline"))
	}
	lines = append(lines, theme.Muted.Render(strings.Repeat("─", innerW)))

	headerLines := len(lines)
	visible := innerH - headerLines
	if visible < 1 {
		visible = 1
	}
	if m.roadmapScroll > len(m.roadmapCards)-1 {
		m.roadmapScroll = 0
	}
	end := m.roadmapScroll + visible
	if end > len(m.roadmapCards) {
		end = len(m.roadmapCards)
	}

	for i := m.roadmapScroll; i < end; i++ {
		card := m.roadmapCards[i]
		if i == m.roadmapDatedCount {
			lines = append(lines, theme.Subtitle.Render("No date"))
		}

		cursor := "  "
		title := utils.Truncate(card.Title, titleW)
		titleStr := theme.CardTitle.Width(titleW).Render(title)
		if i == m.roadmapRow {
			cursor = theme.CardCursor.Render("> ")
			titleStr = theme.CardSelected.Width(titleW).Render(title)
		}
		icon := itemIcon(card.Type, card.State)

		bar := m.roadmapBar(card, min, max, barWidth, i == m.roadmapRow, dated)
		lines = append(lines, cursor+icon+titleStr+" "+bar)
	}

	if len(m.roadmapCards) > visible {
		pct := 0
		if len(m.roadmapCards) > 1 {
			pct = m.roadmapRow * 100 / (len(m.roadmapCards) - 1)
		}
		lines = append(lines, theme.Muted.Render(fmt.Sprintf("  %d/%d (%d%%)", m.roadmapRow+1, len(m.roadmapCards), pct)))
	}

	return strings.Join(lines, "\n")
}

// roadmapBar renders one item's bar over the shared axis: a span from start to
// target, a single marker glyph when only one date is present, or blanks when
// the item is undated.
func (m Model) roadmapBar(card core.Card, min, max time.Time, width int, selected, dated bool) string {
	row := make([]rune, width)
	for i := range row {
		row[i] = ' '
	}
	if !dated {
		return string(row)
	}

	start, hasStart := m.roadmapDate(card, m.roadmapStart)
	endT, hasEnd := m.roadmapDate(card, m.roadmapTarget)

	switch {
	case hasStart && hasEnd:
		sc := scaleCol(start, min, max, width)
		ec := scaleCol(endT, min, max, width)
		if ec < sc {
			sc, ec = ec, sc
		}
		for c := sc; c <= ec && c < width; c++ {
			row[c] = '━'
		}
		row[sc] = '┝'
		if ec < width {
			row[ec] = '┥'
		}
	case hasStart:
		row[scaleCol(start, min, max, width)] = '◆'
	case hasEnd:
		row[scaleCol(endT, min, max, width)] = '◆'
	}

	style := lipgloss.NewStyle().Foreground(theme.ColorPrimary)
	if selected {
		style = lipgloss.NewStyle().Foreground(theme.ColorSecondary)
	}
	return style.Render(string(row))
}
