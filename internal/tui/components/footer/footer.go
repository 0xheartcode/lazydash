package footer

import (
	"fmt"
	"strings"

	"github.com/0xheartcode/lazydash/internal/tui/keys"
	"github.com/0xheartcode/lazydash/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

const version = "v0.1.0"

// Pane identifies which pane is active, for contextual hints.
type Pane int

const (
	PaneProjects Pane = iota
	PaneBoard
	PaneSidebar
)

type Model struct {
	keys   keys.Bindings
	width  int
	pane   Pane
	status string
}

func New(k keys.Bindings) Model {
	return Model{keys: k}
}

func (m *Model) SetWidth(w int) { m.width = w }
func (m *Model) SetPane(p Pane) { m.pane = p }
func (m *Model) SetStatus(s string) { m.status = s }

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
	line := left + strings.Repeat(" ", gap) + right
	return theme.FooterBar.Width(m.width).Render(line)
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

	var paneHints []string
	switch m.pane {
	case PaneProjects:
		paneHints = []string{
			hint("j/k", "move"),
			hint("enter", "open"),
			hint("tab", "→ board"),
		}
	case PaneBoard:
		paneHints = []string{
			hint("j/k", "card"),
			hint("h/l", "column"),
			hint(m.keys.OpenInBrowser, "browser"),
			hint("tab", "→ sidebar"),
		}
	case PaneSidebar:
		paneHints = []string{
			hint("↑/↓", "scroll"),
			hint(m.keys.OpenInBrowser, "browser"),
			hint("tab", "→ projects"),
		}
	}

	all := append(paneHints, common...)
	return strings.Join(all, "  ")
}
