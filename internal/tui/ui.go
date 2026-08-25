package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/0xheartcode/lazydash/internal/api"
	"github.com/0xheartcode/lazydash/internal/config"
	"github.com/0xheartcode/lazydash/internal/core"
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
type clientReadyMsg struct {
	login  string
	orgs   []string
	client *api.Client
}

// --- Model ---

type Model struct {
	cfg    *config.Config
	keys   keys.Bindings
	client *api.Client

	width  int
	height int

	active  pane
	loading bool
	spinner spinner.Model
	err     error
	status  string

	login    string
	orgs     []string
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
	return tea.Batch(m.spinner.Tick, fetchLoginCmd())
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

	case clientReadyMsg:
		m.client = msg.client
		m.login = msg.login
		m.orgs = mergeOrgs(msg.orgs, m.cfg.Defaults.Orgs)
		m.setStatus(fmt.Sprintf("Loading projects for %s…", m.login))
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

func mergeOrgs(discovered, configured []string) []string {
	seen := make(map[string]bool)
	var merged []string
	for _, o := range append(append([]string(nil), discovered...), configured...) {
		lo := strings.ToLower(o)
		if !seen[lo] {
			seen[lo] = true
			merged = append(merged, o)
		}
	}
	return merged
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
			return m, tea.Batch(m.spinner.Tick, fetchBoardCmd(p.ID, m.client))
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

func fetchLoginCmd() tea.Cmd {
	return func() tea.Msg {
		client, err := api.NewClient()
		if err != nil {
			return errMsg{err}
		}
		login, err := client.ViewerLogin()
		if err != nil {
			return errMsg{err}
		}
		// Auto-discover org memberships (best-effort — no error on failure).
		orgs, _ := client.ListViewerOrgs()
		return clientReadyMsg{login: login, orgs: orgs, client: client}
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
	client := m.client
	login := m.login
	orgs := m.orgs
	ignoreOrgs := m.cfg.Defaults.IgnoreOrgs
	ignoreProjects := m.cfg.Defaults.IgnoreProjects
	onlyOrgs := m.cfg.Defaults.OnlyOrgs
	onlyProjects := m.cfg.Defaults.OnlyProjects
	return func() tea.Msg {
		if client == nil {
			return errMsg{fmt.Errorf("client not initialized")}
		}
		projects, err := client.ListUserProjects(login)
		if err != nil {
			return errMsg{err}
		}

		// Step 1 — determine effective org list.
		var effectiveOrgs []string
		if len(onlyOrgs) > 0 {
			effectiveOrgs = onlyOrgs
		} else {
			for _, org := range orgs {
				if !isIgnored(org, ignoreOrgs) {
					effectiveOrgs = append(effectiveOrgs, org)
				}
			}
		}

		// Step 2 — fetch all effective org projects.
		for _, org := range effectiveOrgs {
			orgProjects, err := client.ListOrgProjects(org)
			if err == nil {
				projects = append(projects, orgProjects...)
			}
		}

		// Step 3 — filter final project list.
		if len(onlyProjects) > 0 {
			projects = filterAllowedProjects(projects, onlyProjects)
		} else {
			projects = filterIgnoredProjects(projects, ignoreProjects)
		}
		return projectsLoadedMsg{projects}
	}
}

func isIgnored(name string, list []string) bool {
	lower := strings.ToLower(name)
	for _, item := range list {
		if strings.ToLower(item) == lower {
			return true
		}
	}
	return false
}

func filterIgnoredProjects(projects []core.Project, ignore []string) []core.Project {
	if len(ignore) == 0 {
		return projects
	}
	lower := make([]string, len(ignore))
	for i, p := range ignore {
		lower[i] = strings.ToLower(p)
	}
	var out []core.Project
	for _, p := range projects {
		ownerTitle := strings.ToLower(p.Owner + "/" + p.Title)
		bare := strings.ToLower(p.Title)
		skip := false
		for _, lp := range lower {
			if lp == ownerTitle || lp == bare {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, p)
		}
	}
	return out
}

func filterAllowedProjects(projects []core.Project, only []string) []core.Project {
	if len(only) == 0 {
		return projects
	}
	lower := make([]string, len(only))
	for i, p := range only {
		lower[i] = strings.ToLower(p)
	}
	var out []core.Project
	for _, p := range projects {
		ownerTitle := strings.ToLower(p.Owner + "/" + p.Title)
		bare := strings.ToLower(p.Title)
		for _, lp := range lower {
			if lp == ownerTitle || lp == bare {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

func fetchBoardCmd(projectID string, client *api.Client) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return errMsg{fmt.Errorf("client not initialized")}
		}
		data, err := client.GetProjectBoard(projectID)
		if err != nil {
			return errMsg{err}
		}
		return boardLoadedMsg{data}
	}
}
