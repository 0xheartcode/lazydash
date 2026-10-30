package tui

import (
	"os"
	"os/exec"
	"strings"

	"github.com/0xheartcode/lazydash/internal/core"
	tea "github.com/charmbracelet/bubbletea"
)

// pagerClosedMsg reports that the external pager exited; a non-nil err surfaces
// as a transient status line rather than the full error screen.
type pagerClosedMsg struct{ err error }

// openPager displays a card's full content in an external pager, suspending the
// TUI while the pager owns the terminal and restoring it on exit. The pager is
// $PAGER when set, otherwise bat (which syntax-highlights the markdown), and
// finally less. The card's markdown source is piped in unrendered so bat can
// highlight it and $PAGER users see the raw text.
func (m Model) openPager(card *core.Card) (tea.Model, tea.Cmd) {
	if card == nil {
		return m, nil
	}
	content := pagerContent(card)
	if strings.TrimSpace(content) == "" {
		m.setStatus("nothing to show in pager")
		return m, nil
	}
	name, args := resolvePager()
	c := exec.Command(name, args...)
	c.Stdin = strings.NewReader(content)
	return m, tea.ExecProcess(c, func(err error) tea.Msg {
		return pagerClosedMsg{err: err}
	})
}

// pagerContent assembles the markdown fed to the pager: the title as an H1
// heading followed by the body, so a markdown-aware pager renders both.
func pagerContent(c *core.Card) string {
	var b strings.Builder
	if c.Title != "" {
		b.WriteString("# " + c.Title + "\n\n")
	}
	if c.Body != "" {
		b.WriteString(c.Body)
		if !strings.HasSuffix(c.Body, "\n") {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// resolvePager picks the pager command and its arguments. $PAGER wins (split on
// whitespace so entries like "less -R" work), then bat with markdown syntax
// highlighting, and less as the universal fallback.
func resolvePager() (name string, args []string) {
	if p := strings.TrimSpace(os.Getenv("PAGER")); p != "" {
		fields := strings.Fields(p)
		return fields[0], fields[1:]
	}
	if path, err := exec.LookPath("bat"); err == nil {
		return path, []string{"--style=plain", "--paging=always", "--language=markdown"}
	}
	return "less", []string{"-R"}
}
