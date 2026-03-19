package footer

import (
	"fmt"
	"strings"

	"github.com/0xheartcode/lazydash/internal/tui/keys"
	"github.com/0xheartcode/lazydash/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

const version = "v0.1.0"

type Pane int

const (
	PaneProjects Pane = iota
	PaneBoard
)

type Model struct {
	keys       keys.Bindings
	width      int
	pane       Pane
	boardLayout string // "TABLE_LAYOUT" or "BOARD_LAYOUT"
	status     string
}

func New(k keys.Bindings) Model { return Model{keys: k} }

func (m *Model) SetWidth(w int)            { m.width = w }
func (m *Model) SetPane(p Pane)            { m.pane = p }
func (m *Model) SetBoardLayout(l string)   { m.boardLayout = l }
func (m *Model) SetStatus(s string)        { m.status = s }

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
		paneLabel = theme.Title.Render("BOARD")
		if m.boardLayout == "TABLE_LAYOUT" {
			paneHints = []string{
				hint("j/k", "move"),
				hint(m.keys.OpenInBrowser, "browser"),
				hint(m.keys.OpenInGhDash, "gh-dash"),
				hint(m.keys.PrevView+"/"+m.keys.NextView, "views"),
				hint("tab", "→ projects"),
			}
		} else {
			paneHints = []string{
				hint("j/k", "item"),
				hint("h/l", "column"),
				hint(m.keys.OpenInBrowser, "browser"),
				hint(m.keys.OpenInGhDash, "gh-dash"),
				hint(m.keys.PrevView+"/"+m.keys.NextView, "views"),
				hint("tab", "→ projects"),
			}
		}
	}

	all := append(paneHints, common...)
	return paneLabel + "  " + strings.Join(all, "  ")
}
