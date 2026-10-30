package board

import (
	"strings"
	"testing"

	"github.com/0xheartcode/lazydash/internal/core"
	"github.com/charmbracelet/lipgloss"
)

func TestItemIcon(t *testing.T) {
	tests := []struct {
		name     string
		itemType string
		state    string
		wantRune string
	}{
		{"open issue", "ISSUE", "OPEN", "○"},
		{"closed issue", "ISSUE", "closed", "●"},
		{"merged pr", "PULL_REQUEST", "MERGED", "⎇"},
		{"closed pr", "PULL_REQUEST", "CLOSED", "⎇"},
		{"open pr", "PULL_REQUEST", "OPEN", "⎇"},
		{"draft issue", "DRAFT_ISSUE", "", "◌"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := itemIcon(tt.itemType, tt.state)
			if !strings.Contains(got, tt.wantRune) {
				t.Errorf("itemIcon(%q, %q) = %q, want it to contain %q", tt.itemType, tt.state, got, tt.wantRune)
			}
		})
	}
}

func TestItemIconUnknownTypeIsBlank(t *testing.T) {
	if got := itemIcon("SOMETHING_ELSE", "OPEN"); got != "  " {
		t.Errorf("unknown type icon = %q, want two spaces", got)
	}
}

func TestRenderLabelsEmptyIsPaddedBlank(t *testing.T) {
	got := renderLabels(nil, 8)
	if got != strings.Repeat(" ", 8) {
		t.Errorf("empty labels = %q, want 8 spaces", got)
	}
}

func TestRenderLabelsContainsEachName(t *testing.T) {
	labels := []core.Label{
		{Name: "bug", Color: "d73a4a"},
		{Name: "docs", Color: "0075ca"},
	}
	got := renderLabels(labels, 40)
	for _, l := range labels {
		if !strings.Contains(got, l.Name) {
			t.Errorf("renderLabels output %q missing label %q", got, l.Name)
		}
	}
	// Output is padded out to the requested visible width.
	if w := lipgloss.Width(got); w < 40 {
		t.Errorf("visible width = %d, want at least 40", w)
	}
}

func TestRenderCellEmptyIsPaddedBlank(t *testing.T) {
	got := renderCell("", "TEXT", "", 6)
	if got != strings.Repeat(" ", 6) {
		t.Errorf("empty cell = %q, want 6 spaces", got)
	}
}

func TestRenderCellSingleSelectUsesStatusBadge(t *testing.T) {
	// With no explicit option color, a SINGLE_SELECT value routes through
	// statusBadge, so the value text survives in the output.
	got := renderCell("In Progress", "SINGLE_SELECT", "", 20)
	if !strings.Contains(got, "In Progress") {
		t.Errorf("cell = %q, want it to contain %q", got, "In Progress")
	}
}

func TestRenderCellDefaultContainsValue(t *testing.T) {
	got := renderCell("acme/repo", "REPOSITORY", "", 20)
	if !strings.Contains(got, "acme/repo") {
		t.Errorf("cell = %q, want it to contain %q", got, "acme/repo")
	}
}

func TestStatusBadgeEmptyIsPaddedBlank(t *testing.T) {
	got := statusBadge("", 5)
	if got != strings.Repeat(" ", 5) {
		t.Errorf("empty status = %q, want 5 spaces", got)
	}
}

func TestStatusBadgeContainsStatusText(t *testing.T) {
	tests := []string{"Done", "In Progress", "In Review", "Bug", "Quick"}
	for _, status := range tests {
		t.Run(status, func(t *testing.T) {
			got := statusBadge(status, 30)
			if !strings.Contains(got, status) {
				t.Errorf("statusBadge(%q) = %q, want it to contain the status text", status, got)
			}
		})
	}
}

func TestStatusBadgeRespectsWidth(t *testing.T) {
	// A status longer than the width is truncated, so the visible width never
	// exceeds the requested column width.
	got := statusBadge("In Progress Review", 6)
	if w := lipgloss.Width(got); w > 6 {
		t.Errorf("visible width = %d, want at most 6", w)
	}
}

func TestLayoutReportsCurrentLayout(t *testing.T) {
	m := New()
	if got := m.Layout(); got != "BOARD_LAYOUT" {
		t.Errorf("default layout = %q, want BOARD_LAYOUT", got)
	}
	m.SetLayout("TABLE_LAYOUT")
	if got := m.Layout(); got != "TABLE_LAYOUT" {
		t.Errorf("layout after SetLayout = %q, want TABLE_LAYOUT", got)
	}
}

func TestCountsBoardLayout(t *testing.T) {
	m := New()
	m.SetColumns([]core.Column{
		{Name: "Todo", Cards: []core.Card{{Title: "A"}, {Title: "B"}}},
		{Name: "Done", Cards: []core.Card{{Title: "C"}}},
	})
	cols, cards := m.Counts()
	if cols != 2 {
		t.Errorf("columns = %d, want 2", cols)
	}
	if cards != 3 {
		t.Errorf("cards = %d, want 3", cards)
	}
}

func TestCountsTableLayout(t *testing.T) {
	m := New()
	m.SetLayout("TABLE_LAYOUT")
	m.SetTableItems([]core.Card{{Title: "A"}, {Title: "B"}, {Title: "C"}})
	cols, cards := m.Counts()
	if cols != 0 {
		t.Errorf("columns = %d, want 0 in table layout", cols)
	}
	if cards != 3 {
		t.Errorf("cards = %d, want 3", cards)
	}
}

func TestViewBoardLayoutRendersCardTitle(t *testing.T) {
	m := New()
	m.SetSize(80, 24)
	m.SetColumns([]core.Column{
		{Name: "Todo", Cards: []core.Card{{Title: "Fix login", Type: "ISSUE", State: "OPEN"}}},
		{Name: "Done", Cards: []core.Card{{Title: "Ship docs", Type: "ISSUE", State: "CLOSED"}}},
	})
	out := m.View()
	if out == "" {
		t.Fatal("board View() returned an empty string")
	}
	if !strings.Contains(out, "Fix login") {
		t.Errorf("board view missing card title %q", "Fix login")
	}
}

func TestViewTableLayoutRendersCardTitle(t *testing.T) {
	m := New()
	m.SetSize(80, 24)
	m.SetLayout("TABLE_LAYOUT")
	m.SetVisibleFields([]core.VisibleField{
		{Name: "Title", DataType: "TITLE"},
		{Name: "Status", DataType: "SINGLE_SELECT"},
		{Name: "Assignees", DataType: "ASSIGNEES"},
	})
	m.SetTableItems([]core.Card{
		{
			Title:       "Fix login",
			Type:        "ISSUE",
			State:       "OPEN",
			Number:      42,
			Assignees:   []string{"alice"},
			FieldValues: map[string]string{"Status": "In Progress"},
		},
	})
	out := m.View()
	if out == "" {
		t.Fatal("table View() returned an empty string")
	}
	if !strings.Contains(out, "Fix login") {
		t.Errorf("table view missing card title %q", "Fix login")
	}
}
