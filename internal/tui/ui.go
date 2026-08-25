package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/0xheartcode/lazydash/internal/config"
	"github.com/0xheartcode/lazydash/internal/core"
	"github.com/0xheartcode/lazydash/internal/source"
	"github.com/0xheartcode/lazydash/internal/source/github"
	"github.com/0xheartcode/lazydash/internal/tui/components/board"
	"github.com/0xheartcode/lazydash/internal/tui/components/footer"
	"github.com/0xheartcode/lazydash/internal/tui/components/projectlist"
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
type boardLoadedMsg struct{ data *core.BoardData }
type errMsg struct{ err error }
type tickMsg struct{}
type sourcesReadyMsg struct{ registry *source.Registry }

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
	footer   footer.Model

	boardData *core.BoardData
	viewIdx   int

	showHelp bool
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

	case tea.KeyMsg:
		return m.handleKey(msg)
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
			m.setStatus(fmt.Sprintf("Loading %s…", p.Title))
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
	case m.keys.OpenInBrowser:
		m.openURL(m.board.SelectedCard())
	case m.keys.OpenInGhDash:
		m.openGhDash(m.board.SelectedCard())
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

// --- View ---

func (m Model) View() string {
	if m.width == 0 {
		return "Initializing…"
	}
	if m.showHelp {
		return m.helpView()
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

func (m Model) helpView() string {
	rows := []string{
		theme.Title.Render("lazydash — keyboard reference"),
		"",
		theme.HelpKey.Render("Navigation"),
		theme.HelpKey.Render("  tab / shift+tab") + "  " + theme.HelpDesc.Render("cycle panes (PROJECTS ↔ BOARD)"),
		theme.HelpKey.Render("  j / k          ") + "  " + theme.HelpDesc.Render("move up / down"),
		theme.HelpKey.Render("  h / l          ") + "  " + theme.HelpDesc.Render("move left / right (board columns only)"),
		theme.HelpKey.Render("  [ / ]          ") + "  " + theme.HelpDesc.Render("switch project views"),
		theme.HelpKey.Render("  enter          ") + "  " + theme.HelpDesc.Render("load selected project"),
		"",
		theme.HelpKey.Render("Actions"),
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
		gh, err := github.New(github.Options{
			Orgs:           cfg.Defaults.Orgs,
			IgnoreOrgs:     cfg.Defaults.IgnoreOrgs,
			IgnoreProjects: cfg.Defaults.IgnoreProjects,
			OnlyOrgs:       cfg.Defaults.OnlyOrgs,
			OnlyProjects:   cfg.Defaults.OnlyProjects,
		})
		if err != nil {
			return errMsg{err}
		}
		return sourcesReadyMsg{registry: source.NewRegistry(gh)}
	}
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
		data, err := reg.GetBoard(p)
		if err != nil {
			return errMsg{err}
		}
		return boardLoadedMsg{data}
	}
}
