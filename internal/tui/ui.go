package tui

import (
	"fmt"

	"github.com/0xheartcode/lazydash/internal/config"
	"github.com/0xheartcode/lazydash/internal/core"
	"github.com/0xheartcode/lazydash/internal/source"
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

	case pagerClosedMsg:
		if msg.err != nil {
			m.setStatus("pager error: " + msg.err.Error())
		}
		return m, nil

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
