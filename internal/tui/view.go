package tui

import (
	"strings"

	"github.com/0xheartcode/lazydash/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// --- View ---

func (m Model) View() string {
	writeSnapshot(m) // optional test state seam; no-op unless --dump-state is set
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
		theme.HelpKey.Render("  "+m.keys.Pager+"               ") + "  " + theme.HelpDesc.Render("open full issue in pager (bat / $PAGER)"),
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
