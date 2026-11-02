package tui

import (
	"fmt"

	"github.com/0xheartcode/lazydash/internal/core"
	"github.com/0xheartcode/lazydash/internal/source"
	"github.com/0xheartcode/lazydash/internal/tui/components/footer"
	"github.com/0xheartcode/lazydash/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) applyView(idx int) {
	if m.boardData == nil || len(m.boardData.Views) == 0 {
		return
	}
	if idx < 0 || idx >= len(m.boardData.Views) {
		return
	}
	m.viewIdx = idx
	view := m.boardData.Views[idx]

	m.board.SetLayout(view.Layout)
	m.board.SetViews(m.boardData.Views, idx)
	m.board.SetVisibleFields(view.VisibleFields)
	m.board.SetOptionColors(core.OptionColors(m.boardData))
	m.footer.SetBoardLayout(view.Layout)

	switch view.Layout {
	case "TABLE_LAYOUT", "ROADMAP_LAYOUT":
		m.board.SetTableItems(core.FlatItems(m.boardData))
	case "ACTIVITY_LAYOUT", "CALENDAR_LAYOUT":
		m.board.SetEvents(m.boardData.Events)
	default:
		fieldName := view.GroupByField
		if fieldName == "" {
			fieldName = "Status"
		}
		m.board.SetColumns(core.GroupByField(m.boardData, fieldName))
	}
}

func (m *Model) layout() {
	projectW := m.width * 22 / 100
	if projectW < 20 {
		projectW = 20
	}
	contentW := m.width - projectW
	contentH := m.height - 2

	m.projects.SetSize(projectW, contentH)
	m.board.SetSize(contentW, contentH)
	m.footer.SetWidth(m.width)

	// The detail overlay is a centered panel sized from the viewport.
	sideW := m.width * 55 / 100
	if sideW < 30 {
		sideW = 30
	}
	if sideW > m.width-4 {
		sideW = m.width - 4
	}
	sideH := m.height - 4
	if sideH < 4 {
		sideH = 4
	}
	m.sidebar.SetSize(sideW, sideH)

	modalW := m.width * 60 / 100
	if modalW < 30 {
		modalW = 30
	}
	if modalW > 80 {
		modalW = 80
	}
	m.prompt.SetWidth(modalW)
	m.picker.SetWidth(modalW)
}

func (m *Model) syncActivePane() {
	m.projects.SetActive(m.active == paneProjects)
	m.board.SetActive(m.active == paneBoard)

	switch m.active {
	case paneProjects:
		m.footer.SetPane(footer.PaneProjects)
	case paneBoard:
		m.footer.SetPane(footer.PaneBoard)
	}
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()

	// A write modal captures all keys while open.
	if m.mode != modeNormal {
		return m.handleModalKey(msg, k)
	}

	// The detail overlay captures keys while open: scroll, or dismiss.
	if m.showDetails {
		switch k {
		case m.keys.Quit:
			return m, tea.Quit
		case "esc", m.keys.Enter:
			m.showDetails = false
		case m.keys.Up:
			m.sidebar.ScrollUp()
		case m.keys.Down:
			m.sidebar.ScrollDown()
		case m.keys.Pager:
			return m.openPager(m.board.SelectedCard())
		}
		return m, nil
	}

	switch k {
	case m.keys.Quit:
		return m, tea.Quit
	case m.keys.Help:
		m.showHelp = !m.showHelp
		return m, nil
	case m.keys.Refresh:
		if !m.loading {
			m.loading = true
			m.setStatus("Refreshing…")
			return m, tea.Batch(m.spinner.Tick, m.fetchProjects())
		}
		return m, nil
	case m.keys.NextPane:
		m.active = (m.active + 1) % paneCount
		m.syncActivePane()
		return m, nil
	case m.keys.PrevPane:
		m.active = (m.active + paneCount - 1) % paneCount
		m.syncActivePane()
		return m, nil
	}

	switch m.active {
	case paneProjects:
		return m.handleProjectsKey(k)
	case paneBoard:
		return m.handleBoardKey(k)
	}
	return m, nil
}

func (m Model) handleProjectsKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case m.keys.Up:
		m.projects.MoveUp()
	case m.keys.Down:
		m.projects.MoveDown()
	case m.keys.Enter, m.keys.Right:
		if p := m.projects.Selected(); p != nil {
			m.loading = true
			m.activeProject = *p
			m.setStatus(fmt.Sprintf("Loading %s…", p.Title))
			caps := m.capsFor(p.Source)
			m.footer.SetSource(p.Source, caps.Offline, canWrite(caps))
			return m, tea.Batch(m.spinner.Tick, fetchBoardCmd(*p, m.registry))
		}
	}
	return m, nil
}

func (m Model) handleBoardKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case m.keys.Up:
		m.board.MoveUp()
	case m.keys.Down:
		m.board.MoveDown()
	case m.keys.Left:
		m.board.MoveLeft()
	case m.keys.Right:
		m.board.MoveRight()
	case m.keys.PrevView:
		if m.boardData != nil && m.viewIdx > 0 {
			m.applyView(m.viewIdx - 1)
		}
	case m.keys.NextView:
		if m.boardData != nil && m.viewIdx < len(m.boardData.Views)-1 {
			m.applyView(m.viewIdx + 1)
		}
	case m.keys.Enter:
		if card := m.board.SelectedCard(); card != nil {
			m.sidebar.SetCard(card)
			m.sidebar.SetHistory(m.issueHistory(card.ID))
			m.sidebar.SetActive(true)
			m.showDetails = true
		}
	case m.keys.OpenInBrowser:
		m.openURL(m.board.SelectedCard())
	case m.keys.OpenInGhDash:
		m.openGhDash(m.board.SelectedCard())
	case m.keys.Pager:
		return m.openPager(m.board.SelectedCard())
	case m.keys.Create:
		return m.startCreate()
	case m.keys.Comment:
		return m.startComment()
	case m.keys.ToggleState:
		m.startToggleState()
	case m.keys.Move:
		m.startMove()
	case m.keys.Labels:
		return m.startLabels()
	case m.keys.Assign:
		return m.startAssign()
	case m.keys.Undo:
		return m.startUndo()
	}
	return m, nil
}

func (m *Model) openURL(card *core.Card) {
	if card == nil {
		return
	}
	if card.URL == "" {
		m.setStatus("no URL, draft issues cannot be opened in browser")
		return
	}
	_ = utils.OpenInBrowser(card.URL)
}

func (m *Model) openGhDash(card *core.Card) {
	url := ""
	if card != nil {
		url = card.URL
	}
	launched, err := utils.OpenInGhDash(url)
	if err != nil {
		m.setStatus("error opening: " + err.Error())
		return
	}
	if !launched {
		m.setStatus("gh-dash not found, opening in browser. Install: " + utils.GhDashInstallHint())
	}
}

// --- Write actions ---

// canWrite reports whether a source supports any mutation.
func canWrite(c source.Capabilities) bool {
	return c.Create || c.Comment || c.SetState || c.SetField || c.Labels || c.Assign
}

// capsFor returns the capabilities of the named source (zero value if unknown).
func (m Model) capsFor(name string) source.Capabilities {
	if m.registry == nil {
		return source.Capabilities{}
	}
	if s := m.registry.ByName(name); s != nil {
		return s.Caps()
	}
	return source.Capabilities{}
}

func (m Model) activeCaps() source.Capabilities { return m.capsFor(m.activeProject.Source) }

// activeWriter returns the active project's source as a Writer, if it is one.
func (m Model) activeWriter() (source.Writer, bool) {
	if m.registry == nil {
		return nil, false
	}
	s := m.registry.ByName(m.activeProject.Source)
	if s == nil {
		return nil, false
	}
	return source.AsWriter(s)
}

// activeUndoer returns the active project's source as an Undoer, if it is one.
func (m Model) activeUndoer() (source.Undoer, bool) {
	if m.registry == nil {
		return nil, false
	}
	s := m.registry.ByName(m.activeProject.Source)
	if s == nil {
		return nil, false
	}
	return source.AsUndoer(s)
}

// startUndo reverses the last change to the selected card, where supported.
func (m Model) startUndo() (tea.Model, tea.Cmd) {
	u, ok := m.activeUndoer()
	card := m.board.SelectedCard()
	if !ok || card == nil {
		return m, nil
	}
	item, proj := *card, m.activeProject
	m.loading = true
	m.setStatus("Undoing…")
	return m, tea.Batch(m.spinner.Tick, mutateCmd(proj, func() error { return u.Undo(item) }))
}
