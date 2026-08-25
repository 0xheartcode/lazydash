package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/0xheartcode/lazydash/internal/cache"
	"github.com/0xheartcode/lazydash/internal/config"
	"github.com/0xheartcode/lazydash/internal/core"
	"github.com/0xheartcode/lazydash/internal/source"
	"github.com/0xheartcode/lazydash/internal/source/github"
	"github.com/0xheartcode/lazydash/internal/source/local"
	"github.com/0xheartcode/lazydash/internal/tui/components/board"
	"github.com/0xheartcode/lazydash/internal/tui/components/footer"
	"github.com/0xheartcode/lazydash/internal/tui/components/picker"
	"github.com/0xheartcode/lazydash/internal/tui/components/projectlist"
	"github.com/0xheartcode/lazydash/internal/tui/components/prompt"
	"github.com/0xheartcode/lazydash/internal/tui/components/sidebar"
	"github.com/0xheartcode/lazydash/internal/tui/keys"
	"github.com/0xheartcode/lazydash/internal/tui/theme"
	"github.com/0xheartcode/lazydash/internal/utils"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type pane int

const (
	paneProjects pane = iota
	paneBoard
	paneCount // always equals the number of panes; no magic number
)

// --- Messages ---

type projectsLoadedMsg struct{ projects []core.Project }
type boardLoadedMsg struct {
	data  *core.BoardData
	stale bool // served from the offline cache because the source was unreachable
}
type errMsg struct{ err error }
type tickMsg struct{}
type sourcesReadyMsg struct{ registry *source.Registry }

// mutationDoneMsg reports the result of a write. On success the owning project
// is reloaded so the board reflects the change.
type mutationDoneMsg struct {
	err    error
	reload core.Project
}

// modalMode is the kind of overlay currently capturing input.
type modalMode int

const (
	modeNormal modalMode = iota
	modeInput
	modePicker
	modeConfirm
)

// modalAction is what the open modal will do on submit.
type modalAction int

const (
	actNone modalAction = iota
	actCreate
	actComment
	actMove
	actLabels
	actAssign
	actClose
	actReopen
)

// --- Model ---

type Model struct {
	cfg  *config.Config
	keys keys.Bindings

	registry *source.Registry

	width  int
	height int

	active  pane
	loading bool
	spinner spinner.Model
	err     error
	status  string

	projects projectlist.Model
	board    board.Model
	sidebar  sidebar.Model
	prompt   prompt.Model
	picker   picker.Model
	footer   footer.Model

	boardData     *core.BoardData
	viewIdx       int
	activeProject core.Project // the project whose board is loaded

	showHelp    bool
	showDetails bool
	mode        modalMode
	action      modalAction
	confirm     string // prompt text shown in confirm mode
}

func (m *Model) setStatus(s string) {
	m.status = s
	m.footer.SetStatus(s)
}

func newModel(cfg *config.Config) Model {
	k := keys.FromConfig(cfg)
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.ColorPrimary)

	return Model{
		cfg:      cfg,
		keys:     k,
		spinner:  s,
		loading:  true,
		active:   paneProjects,
		projects: projectlist.New(),
		board:    board.New(),
		sidebar:  sidebar.New(),
		prompt:   prompt.New(),
		picker:   picker.New(),
		footer:   footer.New(k),
	}
}

func Start(cfg *config.Config) error {
	m := newModel(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, initSourcesCmd(m.cfg))
}

// --- Update ---

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case sourcesReadyMsg:
		m.registry = msg.registry
		m.setStatus("Loading projects…")
		return m, m.fetchProjects()

	case projectsLoadedMsg:
		m.loading = false
		m.setStatus(fmt.Sprintf("%d projects", len(msg.projects)))
		m.projects.SetProjects(msg.projects)
		m.syncActivePane()
		return m, m.scheduleRefresh()

	case tickMsg:
		if !m.loading {
			m.loading = true
			m.setStatus("Refreshing…")
			return m, tea.Batch(m.spinner.Tick, m.fetchProjects())
		}
		return m, m.scheduleRefresh()

	case boardLoadedMsg:
		m.loading = false
		m.boardData = msg.data
		m.viewIdx = 0
		m.applyView(0)
		m.active = paneBoard
		m.syncActivePane()
		status := fmt.Sprintf("%d items · %d views", len(msg.data.Items), len(msg.data.Views))
		if len(msg.data.Items) == 100 {
			status += "  ⚠ capped at 100 items"
		}
		if msg.stale {
			status = "⚠ offline — showing cached board · " + status
		}
		m.setStatus(status)
		return m, nil

	case errMsg:
		m.loading = false
		m.err = msg.err
		if utils.IsAuthError(msg.err) {
			m.err = fmt.Errorf("not authenticated — run: gh auth login")
		}
		m.setStatus("error: " + m.err.Error())
		return m, nil

	case mutationDoneMsg:
		if msg.err != nil {
			m.loading = false
			m.setStatus("error: " + msg.err.Error())
			return m, nil
		}
		m.loading = true
		m.setStatus("saved · reloading…")
		return m, tea.Batch(m.spinner.Tick, fetchBoardCmd(msg.reload, m.registry))

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// While an input modal is open, forward other messages (e.g. cursor blink)
	// to the field so it keeps updating.
	if m.mode == modeInput {
		cmd := m.prompt.Update(msg)
		return m, cmd
	}
	return m, nil
}

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

	if view.Layout == "TABLE_LAYOUT" || view.Layout == "ROADMAP_LAYOUT" {
		m.board.SetTableItems(core.FlatItems(m.boardData))
	} else {
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
			m.sidebar.SetActive(true)
			m.showDetails = true
		}
	case m.keys.OpenInBrowser:
		m.openURL(m.board.SelectedCard())
	case m.keys.OpenInGhDash:
		m.openGhDash(m.board.SelectedCard())
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
		m.setStatus("no URL — draft issues cannot be opened in browser")
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
		m.setStatus("gh-dash not found — opening in browser. Install: " + utils.GhDashInstallHint())
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

func (m *Model) openInput(action modalAction, title, placeholder string, multiline bool) tea.Cmd {
	m.action = action
	m.mode = modeInput
	return m.prompt.Open(title, placeholder, multiline)
}

func (m *Model) openPicker(action modalAction, title string, options []string) {
	m.action = action
	m.mode = modePicker
	m.picker.Open(title, options)
}

func (m *Model) openConfirm(action modalAction, text string) {
	m.action = action
	m.mode = modeConfirm
	m.confirm = text
}

func (m *Model) closeModal() {
	m.mode = modeNormal
	m.action = actNone
	m.confirm = ""
}

func (m Model) startCreate() (tea.Model, tea.Cmd) {
	if !m.activeCaps().Create {
		m.setStatus("create not supported by this source")
		return m, nil
	}
	return m, m.openInput(actCreate, "New issue title", "e.g. Fix login crash", false)
}

func (m Model) startComment() (tea.Model, tea.Cmd) {
	if !m.activeCaps().Comment || m.board.SelectedCard() == nil {
		return m, nil
	}
	return m, m.openInput(actComment, "Comment", "type a comment", true)
}

func (m *Model) startToggleState() {
	if !m.activeCaps().SetState {
		m.setStatus("state changes not supported by this source")
		return
	}
	card := m.board.SelectedCard()
	if card == nil {
		return
	}
	title := utils.Truncate(card.Title, 40)
	if strings.EqualFold(card.State, "closed") {
		m.openConfirm(actReopen, "Reopen \""+title+"\"?")
	} else {
		m.openConfirm(actClose, "Close \""+title+"\"?")
	}
}

func (m *Model) startMove() {
	if !m.activeCaps().SetField || m.board.SelectedCard() == nil {
		return
	}
	field := m.currentGroupField()
	opts := m.fieldOptions(field)
	if field == "" || len(opts) == 0 {
		m.setStatus("no field to move by in this view")
		return
	}
	m.openPicker(actMove, "Move to "+field, opts)
}

func (m Model) startLabels() (tea.Model, tea.Cmd) {
	if !m.activeCaps().Labels || m.board.SelectedCard() == nil {
		return m, nil
	}
	return m, m.openInput(actLabels, "Labels (comma-separated)", "bug, auth", false)
}

func (m Model) startAssign() (tea.Model, tea.Cmd) {
	if !m.activeCaps().Assign || m.board.SelectedCard() == nil {
		return m, nil
	}
	return m, m.openInput(actAssign, "Assignee", "name or email", false)
}

func (m Model) handleModalKey(msg tea.KeyMsg, k string) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeInput:
		switch k {
		case "esc":
			m.closeModal()
			return m, nil
		case "enter":
			if !m.prompt.Multiline() {
				return m.submitModal()
			}
		case "ctrl+d":
			if m.prompt.Multiline() {
				return m.submitModal()
			}
		}
		return m, m.prompt.Update(msg)
	case modePicker:
		switch k {
		case "esc":
			m.closeModal()
		case m.keys.Up:
			m.picker.MoveUp()
		case m.keys.Down:
			m.picker.MoveDown()
		case "enter":
			return m.submitModal()
		}
		return m, nil
	case modeConfirm:
		switch k {
		case "y", "enter":
			return m.submitModal()
		case "n", "esc":
			m.closeModal()
		}
		return m, nil
	}
	return m, nil
}

// submitModal dispatches the open modal's action to the active source's Writer
// and returns a command that performs the mutation and reloads the board.
func (m Model) submitModal() (tea.Model, tea.Cmd) {
	action := m.action
	var value string
	switch m.mode {
	case modeInput:
		value = strings.TrimSpace(m.prompt.Value())
	case modePicker:
		value = m.picker.Selected()
	}
	card := m.board.SelectedCard()
	proj := m.activeProject
	m.closeModal()

	w, ok := m.activeWriter()
	if !ok {
		m.setStatus("this source is read-only")
		return m, nil
	}

	run := func(status string, fn func() error) (tea.Model, tea.Cmd) {
		m.loading = true
		m.setStatus(status)
		return m, tea.Batch(m.spinner.Tick, mutateCmd(proj, fn))
	}

	switch action {
	case actCreate:
		if value == "" {
			m.setStatus("create cancelled — empty title")
			return m, nil
		}
		return run("Creating issue…", func() error {
			_, err := w.CreateIssue(proj.ID, source.Draft{Title: value})
			return err
		})
	case actComment:
		if value == "" || card == nil {
			return m, nil
		}
		item := *card
		return run("Adding comment…", func() error { return w.Comment(item, value) })
	case actMove:
		if card == nil || value == "" {
			return m, nil
		}
		item, field := *card, m.currentGroupField()
		return run("Moving…", func() error { return w.SetField(item, field, value) })
	case actLabels:
		if card == nil {
			return m, nil
		}
		item, labels := *card, splitCSV(value)
		return run("Updating labels…", func() error { return w.SetLabels(item, labels) })
	case actAssign:
		if card == nil {
			return m, nil
		}
		item, who := *card, splitCSV(value)
		return run("Updating assignee…", func() error { return w.SetAssignees(item, who) })
	case actClose:
		if card == nil {
			return m, nil
		}
		item := *card
		return run("Closing…", func() error { return w.SetState(item, "closed") })
	case actReopen:
		if card == nil {
			return m, nil
		}
		item := *card
		return run("Reopening…", func() error { return w.SetState(item, "open") })
	}
	return m, nil
}

// currentGroupField is the field the current view groups its columns by.
func (m Model) currentGroupField() string {
	if m.boardData == nil || m.viewIdx < 0 || m.viewIdx >= len(m.boardData.Views) {
		return ""
	}
	return m.boardData.Views[m.viewIdx].GroupByField
}

// fieldOptions returns the option names of a single-select field.
func (m Model) fieldOptions(field string) []string {
	if m.boardData == nil || field == "" {
		return nil
	}
	for _, f := range m.boardData.Fields {
		if f.Name == field {
			opts := make([]string, 0, len(f.Options))
			for _, o := range f.Options {
				opts = append(opts, o.Name)
			}
			return opts
		}
	}
	return nil
}

// mutateCmd runs a write off the UI goroutine and reports the outcome.
func mutateCmd(proj core.Project, fn func() error) tea.Cmd {
	return func() tea.Msg {
		if err := fn(); err != nil {
			return mutationDoneMsg{err: err}
		}
		return mutationDoneMsg{reload: proj}
	}
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// --- View ---

func (m Model) View() string {
	if m.width == 0 {
		return "Initializing…"
	}
	if m.showHelp {
		return m.helpView()
	}
	if m.showDetails {
		return m.detailsView()
	}
	switch m.mode {
	case modeInput:
		return m.overlay(m.prompt.View())
	case modePicker:
		return m.overlay(m.picker.View())
	case modeConfirm:
		return m.overlay(m.confirmView())
	}

	var body string
	if m.loading {
		body = lipgloss.Place(m.width, m.height-2, lipgloss.Center, lipgloss.Center,
			m.spinner.View()+" "+theme.Muted.Render(m.status))
	} else if m.err != nil {
		body = lipgloss.Place(m.width, m.height-2, lipgloss.Center, lipgloss.Center,
			theme.Muted.Render("Error: "+m.err.Error()+"\n\nPress r to retry or q to quit."))
	} else {
		body = lipgloss.JoinHorizontal(lipgloss.Top,
			m.projects.View(),
			m.board.View(),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left, body, m.footer.View())
}

// detailsView renders the selected card's detail panel as a centered overlay.
func (m Model) detailsView() string {
	return m.overlay(m.sidebar.View())
}

// overlay centers a box over the whole viewport.
func (m Model) overlay(box string) string {
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// confirmView renders the confirm modal.
func (m Model) confirmView() string {
	content := theme.Title.Render(m.confirm) + "\n\n" + theme.Muted.Render("y confirm · n cancel")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorWarning).
		Padding(1, 3).
		Render(content)
}

func (m Model) helpView() string {
	rows := []string{
		theme.Title.Render("lazydash — keyboard reference"),
		"",
		theme.HelpKey.Render("Navigation"),
		theme.HelpKey.Render("  tab / shift+tab") + "  " + theme.HelpDesc.Render("cycle panes (PROJECTS ↔ BOARD)"),
		theme.HelpKey.Render("  j / k          ") + "  " + theme.HelpDesc.Render("move up / down"),
		theme.HelpKey.Render("  h / l          ") + "  " + theme.HelpDesc.Render("move left / right (board columns only)"),
		theme.HelpKey.Render("  [ / ]          ") + "  " + theme.HelpDesc.Render("switch project views"),
		theme.HelpKey.Render("  enter          ") + "  " + theme.HelpDesc.Render("load project · open card details"),
		theme.HelpKey.Render("  esc            ") + "  " + theme.HelpDesc.Render("close details / overlay"),
		"",
		theme.HelpKey.Render("Actions"),
		theme.HelpKey.Render("  "+m.keys.Create+"               ") + "  " + theme.HelpDesc.Render("new issue  (writable sources only)"),
		theme.HelpKey.Render("  "+m.keys.Comment+"               ") + "  " + theme.HelpDesc.Render("comment on selected card"),
		theme.HelpKey.Render("  "+m.keys.ToggleState+"               ") + "  " + theme.HelpDesc.Render("close / reopen"),
		theme.HelpKey.Render("  "+m.keys.Move+"               ") + "  " + theme.HelpDesc.Render("move to another column"),
		theme.HelpKey.Render("  "+m.keys.Labels+" / "+m.keys.Assign+"           ") + "  " + theme.HelpDesc.Render("set labels / assignee"),
		theme.HelpKey.Render("  "+m.keys.Undo+"               ") + "  " + theme.HelpDesc.Render("undo last change (local issues)"),
		theme.HelpKey.Render("  "+m.keys.OpenInBrowser+"               ") + "  " + theme.HelpDesc.Render("open in browser"),
		theme.HelpKey.Render("  "+m.keys.OpenInGhDash+"               ") + "  " + theme.HelpDesc.Render("open in gh-dash  (install: gh extension install dlvhdr/gh-dash)"),
		theme.HelpKey.Render("  "+m.keys.Refresh+"               ") + "  " + theme.HelpDesc.Render("refresh"),
		theme.HelpKey.Render("  "+m.keys.Help+"               ") + "  " + theme.HelpDesc.Render("toggle this help"),
		theme.HelpKey.Render("  "+m.keys.Quit+"               ") + "  " + theme.HelpDesc.Render("quit"),
	}

	content := strings.Join(rows, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorPrimary).
		Padding(1, 3).
		Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// --- Commands ---

func initSourcesCmd(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		var sources []source.Source
		var firstErr error

		if sourceEnabled(cfg, "github") {
			gh, err := github.New(github.Options{
				Orgs:           cfg.Defaults.Orgs,
				IgnoreOrgs:     cfg.Defaults.IgnoreOrgs,
				IgnoreProjects: cfg.Defaults.IgnoreProjects,
				OnlyOrgs:       cfg.Defaults.OnlyOrgs,
				OnlyProjects:   cfg.Defaults.OnlyProjects,
			})
			if err != nil {
				firstErr = err // remembered, but a working local source can still carry the session
			} else {
				sources = append(sources, gh)
			}
		}

		if sourceEnabled(cfg, "local") {
			if loc, err := local.New("."); err == nil && loc != nil {
				sources = append(sources, loc)
			}
		}

		if len(sources) == 0 {
			if firstErr != nil {
				return errMsg{firstErr}
			}
			return errMsg{fmt.Errorf("no sources available — run `gh auth login`, or launch inside a repo with local issues")}
		}
		return sourcesReadyMsg{registry: source.NewRegistry(sources...)}
	}
}

// sourceEnabled reports whether a backend should be initialised. With no
// configured sources every backend is auto-detected; otherwise only the named
// ones are used.
func sourceEnabled(cfg *config.Config, name string) bool {
	if len(cfg.Defaults.Sources) == 0 {
		return true
	}
	for _, s := range cfg.Defaults.Sources {
		if strings.EqualFold(s, name) {
			return true
		}
	}
	return false
}

func (m Model) scheduleRefresh() tea.Cmd {
	if m.cfg.Defaults.RefreshIntervalMinutes <= 0 {
		return nil
	}
	d := time.Duration(m.cfg.Defaults.RefreshIntervalMinutes) * time.Minute
	return tea.Tick(d, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m Model) fetchProjects() tea.Cmd {
	reg := m.registry
	return func() tea.Msg {
		if reg == nil {
			return errMsg{fmt.Errorf("sources not initialized")}
		}
		projects, err := reg.ListProjects()
		if err != nil {
			return errMsg{err}
		}
		return projectsLoadedMsg{projects}
	}
}

func fetchBoardCmd(p core.Project, reg *source.Registry) tea.Cmd {
	return func() tea.Msg {
		if reg == nil {
			return errMsg{fmt.Errorf("sources not initialized")}
		}
		key := p.Source + "/" + p.ID
		data, err := reg.GetBoard(p)
		if err != nil {
			// Fall back to the last cached copy so the board stays browsable
			// offline; only error if we have never loaded it.
			if cached, ok := cache.Load(key); ok {
				return boardLoadedMsg{data: cached, stale: true}
			}
			return errMsg{err}
		}
		_ = cache.Save(key, data)
		return boardLoadedMsg{data: data}
	}
}
