package tui

import (
	"fmt"

	"github.com/0xheartcode/lazydash/internal/api"
	"github.com/0xheartcode/lazydash/internal/config"
	"github.com/0xheartcode/lazydash/internal/tui/components/board"
	"github.com/0xheartcode/lazydash/internal/tui/components/footer"
	"github.com/0xheartcode/lazydash/internal/tui/components/projectlist"
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
	paneSidebar
)

// --- Messages ---

type projectsLoadedMsg struct{ projects []api.Project }
type boardLoadedMsg struct{ columns []api.Column }
type errMsg struct{ err error }
type clientReadyMsg struct {
	login  string
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
	projects projectlist.Model
	board    board.Model
	sidebar  sidebar.Model
	footer   footer.Model

	showHelp bool
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
		footer:   footer.New(k),
	}
}

// Start creates and runs the bubbletea program.
func Start(cfg *config.Config) error {
	m := newModel(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// --- Init ---

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
		m.status = fmt.Sprintf("Loading projects for %s…", m.login)
		return m, m.fetchProjects()

	case projectsLoadedMsg:
		m.loading = false
		m.status = ""
		m.projects.SetProjects(msg.projects)
		m.syncActivePane()
		return m, nil

	case boardLoadedMsg:
		m.loading = false
		m.status = ""
		m.board.SetColumns(msg.columns)
		m.sidebar.SetCard(nil)
		m.active = paneBoard
		m.syncActivePane()
		return m, nil

	case errMsg:
		m.loading = false
		m.err = msg.err
		if utils.IsAuthError(msg.err) {
			m.err = fmt.Errorf("not authenticated — run: gh auth login")
		}
		m.status = "error: " + m.err.Error()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *Model) layout() {
	projectW := m.width * 20 / 100
	if projectW < 18 {
		projectW = 18
	}
	sidebarW := m.width * 26 / 100
	if sidebarW < 22 {
		sidebarW = 22
	}
	boardW := m.width - projectW - sidebarW

	contentH := m.height - 2 // 2 for footer line + padding

	m.projects.SetSize(projectW, contentH)
	m.board.SetSize(boardW, contentH)
	m.sidebar.SetSize(sidebarW, contentH)
	m.footer.SetWidth(m.width)
}

func (m *Model) syncActivePane() {
	m.projects.SetActive(m.active == paneProjects)
	m.board.SetActive(m.active == paneBoard)
	m.sidebar.SetActive(m.active == paneSidebar)

	switch m.active {
	case paneProjects:
		m.footer.SetPane(footer.PaneProjects)
	case paneBoard:
		m.footer.SetPane(footer.PaneBoard)
	case paneSidebar:
		m.footer.SetPane(footer.PaneSidebar)
	}
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()

	// Global keys
	switch k {
	case m.keys.Quit:
		return m, tea.Quit
	case m.keys.Help:
		m.showHelp = !m.showHelp
		return m, nil
	case m.keys.Refresh:
		if !m.loading {
			m.loading = true
			m.status = "Refreshing…"
			return m, tea.Batch(m.spinner.Tick, m.fetchProjects())
		}
		return m, nil
	case m.keys.NextPane:
		m.active = (m.active + 1) % 3
		m.syncActivePane()
		return m, nil
	case m.keys.PrevPane:
		m.active = (m.active + 2) % 3
		m.syncActivePane()
		return m, nil
	}

	// Pane-local keys
	switch m.active {
	case paneProjects:
		return m.handleProjectsKey(k)
	case paneBoard:
		return m.handleBoardKey(k)
	case paneSidebar:
		return m.handleSidebarKey(k)
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
			m.status = fmt.Sprintf("Loading %s…", p.Title)
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
	case m.keys.Enter:
		if card := m.board.SelectedCard(); card != nil {
			m.sidebar.SetCard(card)
			m.active = paneSidebar
			m.syncActivePane()
		}
	case m.keys.OpenInBrowser:
		m.openURL(m.board.SelectedCard())
	case m.keys.OpenInGhDash:
		m.openGhDash(m.board.SelectedCard())
	}
	return m, nil
}

func (m Model) handleSidebarKey(k string) (tea.Model, tea.Cmd) {
	switch k {
	case m.keys.Up:
		m.sidebar.ScrollUp()
	case m.keys.Down:
		m.sidebar.ScrollDown()
	case m.keys.OpenInBrowser:
		m.openURL(m.board.SelectedCard())
	case m.keys.OpenInGhDash:
		m.openGhDash(m.board.SelectedCard())
	}
	return m, nil
}

func (m *Model) openURL(card *api.Card) {
	if card == nil {
		return
	}
	if card.URL == "" {
		m.status = "no URL — draft issues cannot be opened in browser"
		return
	}
	_ = utils.OpenInBrowser(card.URL)
}

func (m *Model) openGhDash(card *api.Card) {
	url := ""
	if card != nil {
		url = card.URL
	}
	launched, err := utils.OpenInGhDash(url)
	if err != nil {
		m.status = "error opening: " + err.Error()
		return
	}
	if !launched {
		m.status = "gh-dash not found — opening in browser. Install: " + utils.GhDashInstallHint()
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
		center := lipgloss.Place(
			m.width, m.height-2,
			lipgloss.Center, lipgloss.Center,
			m.spinner.View()+" "+theme.Muted.Render(m.status),
		)
		body = center
	} else if m.err != nil {
		msg := theme.Muted.Render("Error: " + m.err.Error() + "\n\nPress r to retry or q to quit.")
		body = lipgloss.Place(m.width, m.height-2, lipgloss.Center, lipgloss.Center, msg)
	} else {
		body = lipgloss.JoinHorizontal(lipgloss.Top,
			m.projects.View(),
			m.board.View(),
			m.sidebar.View(),
		)
	}

	m.footer.SetStatus(m.status)
	return lipgloss.JoinVertical(lipgloss.Left, body, m.footer.View())
}

func (m Model) helpView() string {
	rows := []string{
		theme.Title.Render("lazydash — keyboard reference"),
		"",
		theme.HelpKey.Render("Navigation"),
		theme.HelpKey.Render("  tab / shift+tab") + "  " + theme.HelpDesc.Render("cycle panes"),
		theme.HelpKey.Render("  j / k          ") + "  " + theme.HelpDesc.Render("move up/down"),
		theme.HelpKey.Render("  h / l          ") + "  " + theme.HelpDesc.Render("move left/right (board columns)"),
		theme.HelpKey.Render("  enter          ") + "  " + theme.HelpDesc.Render("select / open"),
		"",
		theme.HelpKey.Render("Actions"),
		theme.HelpKey.Render("  "+m.keys.OpenInBrowser+"               ") + "  " + theme.HelpDesc.Render("open in browser"),
		theme.HelpKey.Render("  "+m.keys.OpenInGhDash+"               ") + "  " + theme.HelpDesc.Render("open in gh-dash (install: gh extension install dlvhdr/gh-dash)"),
		theme.HelpKey.Render("  "+m.keys.Refresh+"               ") + "  " + theme.HelpDesc.Render("refresh"),
		theme.HelpKey.Render("  "+m.keys.Help+"               ") + "  " + theme.HelpDesc.Render("toggle this help"),
		theme.HelpKey.Render("  "+m.keys.Quit+"               ") + "  " + theme.HelpDesc.Render("quit"),
	}

	content := ""
	for _, r := range rows {
		content += r + "\n"
	}

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
		return clientReadyMsg{login: login, client: client}
	}
}

func (m Model) fetchProjects() tea.Cmd {
	client := m.client
	login := m.login
	cfg := m.cfg
	return func() tea.Msg {
		if client == nil {
			return errMsg{fmt.Errorf("client not initialized")}
		}
		projects, err := client.ListUserProjects(login)
		if err != nil {
			return errMsg{err}
		}
		for _, org := range cfg.Defaults.Orgs {
			orgProjects, err := client.ListOrgProjects(org)
			if err == nil {
				projects = append(projects, orgProjects...)
			}
		}
		return projectsLoadedMsg{projects}
	}
}

func fetchBoardCmd(projectID string, client *api.Client) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return errMsg{fmt.Errorf("client not initialized")}
		}
		columns, err := client.GetProjectBoard(projectID)
		if err != nil {
			return errMsg{err}
		}
		return boardLoadedMsg{columns}
	}
}

