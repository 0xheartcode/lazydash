package footer

import (
	"fmt"
	"strings"

	"github.com/0xheartcode/lazydash/internal/tui/keys"
	"github.com/0xheartcode/lazydash/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

const version = "v0.2.0"

type Pane int

const (
	PaneProjects Pane = iota
	PaneBoard
)

type Model struct {
	keys        keys.Bindings
	width       int
	pane        Pane
	boardLayout string // "TABLE_LAYOUT" or "BOARD_LAYOUT"
	status      string
	source      string // active source name, shown on the board pane
	offline     bool   // whether the active source works offline
	canWrite    bool   // whether the active source supports mutations
}

func New(k keys.Bindings) Model { return Model{keys: k} }

func (m *Model) SetWidth(w int)          { m.width = w }
func (m *Model) SetPane(p Pane)          { m.pane = p }
func (m *Model) SetBoardLayout(l string) { m.boardLayout = l }
func (m *Model) SetStatus(s string)      { m.status = s }

// SetSource records which backend the loaded board came from, whether it is an
// offline source, and whether it supports writes, so the board pane can label it
// and show the write hints only when they apply.
func (m *Model) SetSource(name string, offline, canWrite bool) {
	m.source = name
	m.offline = offline
	m.canWrite = canWrite
}

func (m Model) View() string {
	left := m.hintBar()
	right := theme.Muted.Render(fmt.Sprintf("lazydash %s", version))
	if m.status != "" {
		right = theme.Muted.Render(m.status)
	}

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return theme.FooterBar.Width(m.width).Render(left + strings.Repeat(" ", gap) + right)
}

func hint(key, desc string) string {
	return theme.FooterKey.Render(key) + " " + theme.FooterDesc.Render(desc)
}

func (m Model) hintBar() string {
	common := []string{
		hint(m.keys.Quit, "quit"),
		hint(m.keys.Refresh, "refresh"),
		hint(m.keys.Help, "help"),
	}

	paneLabel := ""
	var paneHints []string

	switch m.pane {
	case PaneProjects:
		paneLabel = theme.Title.Render("PROJECTS")
		paneHints = []string{
			hint("j/k", "move"),
			hint("enter", "load"),
			hint("tab", "→ board"),
		}
	case PaneBoard:
		paneLabel = theme.Title.Render("BOARD") + m.sourceTag()
		if m.boardLayout == "TABLE_LAYOUT" {
			paneHints = []string{
				hint("j/k", "move"),
				hint("enter", "details"),
				hint(m.keys.OpenInBrowser, "browser"),
				hint(m.keys.PrevView+"/"+m.keys.NextView, "views"),
				hint("tab", "→ projects"),
			}
		} else {
			paneHints = []string{
				hint("j/k", "item"),
				hint("h/l", "column"),
				hint("enter", "details"),
				hint(m.keys.OpenInBrowser, "browser"),
				hint(m.keys.PrevView+"/"+m.keys.NextView, "views"),
				hint("tab", "→ projects"),
			}
		}
		if m.canWrite {
			write := []string{
				hint(m.keys.Create, "new"),
				hint(m.keys.Comment, "comment"),
				hint(m.keys.ToggleState, "close"),
			}
			paneHints = append(write, paneHints...)
		}
	}

	all := append(paneHints, common...)
	return paneLabel + "  " + strings.Join(all, "  ")
}

// sourceTag renders the active source next to the BOARD label, with an offline
// marker for backends that need no network.
func (m Model) sourceTag() string {
	if m.source == "" {
		return ""
	}
	tag := " " + theme.Muted.Render("· "+m.source)
	if m.offline {
		tag += " " + lipgloss.NewStyle().Foreground(theme.ColorSuccess).Render("offline")
	}
	return tag
}
