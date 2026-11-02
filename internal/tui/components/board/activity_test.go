package board

import (
	"strings"
	"testing"

	"github.com/0xheartcode/lazydash/internal/core"
)

func sampleEvents() []core.Event {
	return []core.Event{
		{IssueID: "i1", IssueTitle: "Fix login crash", Actor: "alice", Kind: "Closed", Date: "2026-01-03T10:00:00Z"},
		{IssueID: "i1", IssueTitle: "Fix login crash", Actor: "bob", Kind: "Commented", Date: "2026-01-02T10:00:00Z"},
		{IssueID: "i1", IssueTitle: "Fix login crash", Actor: "alice", Kind: "Created", Date: "2026-01-01T10:00:00Z"},
	}
}

func TestRenderActivityContainsEvents(t *testing.T) {
	m := New()
	m.SetSize(100, 24)
	m.SetLayout("ACTIVITY_LAYOUT")
	m.SetEvents(sampleEvents())

	out := m.View()
	if out == "" {
		t.Fatal("activity View() returned empty string")
	}
	for _, want := range []string{"Fix login crash", "Closed", "Commented", "Created", "2026-01-03"} {
		if !strings.Contains(out, want) {
			t.Errorf("activity view missing %q", want)
		}
	}
}

func TestRenderActivityEmptyDoesNotPanic(t *testing.T) {
	m := New()
	m.SetSize(100, 24)
	m.SetLayout("ACTIVITY_LAYOUT")
	m.SetEvents(nil)
	if out := m.View(); !strings.Contains(out, "No activity") {
		t.Errorf("empty activity view = %q, want a no-activity notice", out)
	}
}

func TestRenderCalendarContainsMonthAndDay(t *testing.T) {
	m := New()
	m.SetSize(100, 24)
	m.SetLayout("CALENDAR_LAYOUT")
	m.SetEvents(sampleEvents())

	out := m.View()
	if out == "" {
		t.Fatal("calendar View() returned empty string")
	}
	// The newest event sets the selected month to January 2026.
	for _, want := range []string{"January 2026", "Su", "Fix login crash"} {
		if !strings.Contains(out, want) {
			t.Errorf("calendar view missing %q", want)
		}
	}
}

func TestCalendarNavigationRollsMonths(t *testing.T) {
	m := New()
	m.SetSize(100, 24)
	m.SetLayout("CALENDAR_LAYOUT")
	m.SetEvents(sampleEvents()) // selected day snaps to 2026-01-03

	if got := m.SelectedDay(); got != "2026-01-03" {
		t.Fatalf("initial SelectedDay = %q, want 2026-01-03", got)
	}
	m.MoveLeft()
	m.MoveLeft()
	m.MoveLeft() // back to 2025-12-31, crossing the month boundary
	if got := m.SelectedDay(); got != "2025-12-31" {
		t.Errorf("after three lefts SelectedDay = %q, want 2025-12-31", got)
	}
	m.MoveDown() // +7 days -> 2026-01-07
	if got := m.SelectedDay(); got != "2026-01-07" {
		t.Errorf("after down SelectedDay = %q, want 2026-01-07", got)
	}
}

func TestActivityCursorTracksSelectedEvent(t *testing.T) {
	m := New()
	m.SetSize(100, 24)
	m.SetLayout("ACTIVITY_LAYOUT")
	m.SetEvents(sampleEvents())

	if m.EventCount() != 3 {
		t.Fatalf("EventCount = %d, want 3", m.EventCount())
	}
	if ev := m.SelectedEvent(); ev == nil || ev.Kind != "Closed" {
		t.Fatalf("initial SelectedEvent = %v, want the Closed event", ev)
	}
	m.MoveDown()
	m.MoveDown()
	if ev := m.SelectedEvent(); ev == nil || ev.Kind != "Created" {
		t.Errorf("after two downs SelectedEvent = %v, want the Created event", ev)
	}
}
