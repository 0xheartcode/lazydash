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
	tea "github.com/charmbracelet/bubbletea"
)

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
