package board

import (
	"strings"
	"testing"

	"github.com/0xheartcode/lazydash/internal/core"
)

// roadmapCards builds three dated items spanning early to mid 2026 plus one
// undated item, for exercising the timeline renderer.
func roadmapCards() []core.Card {
	return []core.Card{
		{ID: "1", Type: "ISSUE", State: "OPEN", Number: 1, Title: "Design spec",
			FieldValues: map[string]string{"Start": "2026-01-05", "Target": "2026-02-10"}},
		{ID: "2", Type: "ISSUE", State: "OPEN", Number: 2, Title: "Build it",
			FieldValues: map[string]string{"Start": "2026-02-15", "Target": "2026-04-01"}},
		{ID: "3", Type: "ISSUE", State: "OPEN", Number: 3, Title: "Launch day",
			FieldValues: map[string]string{"Start": "2026-05-01"}},
		{ID: "4", Type: "ISSUE", State: "OPEN", Number: 4, Title: "Someday idea",
			FieldValues: map[string]string{}},
	}
}

func TestRenderRoadmapContainsTitlesAndMonths(t *testing.T) {
	m := New()
	m.SetSize(120, 24)
	m.SetLayout("ROADMAP_LAYOUT")
	m.SetRoadmap(roadmapCards(), "Start", "Target")

	out := m.View()
	if out == "" {
		t.Fatal("roadmap View() returned an empty string")
	}
	// Every dated title and the undated title appear, as does the No-date section.
	for _, want := range []string{"Design spec", "Build it", "Launch day", "Someday idea", "No date"} {
		if !strings.Contains(out, want) {
			t.Errorf("roadmap view missing %q", want)
		}
	}
	// The month-labelled axis shows at least the start and interior months.
	for _, want := range []string{"Jan 26", "Mar", "Apr"} {
		if !strings.Contains(out, want) {
			t.Errorf("roadmap axis missing month label %q", want)
		}
	}
}

func TestRoadmapSelectionAndUndatedCount(t *testing.T) {
	m := New()
	m.SetSize(120, 24)
	m.SetLayout("ROADMAP_LAYOUT")
	m.SetRoadmap(roadmapCards(), "Start", "Target")

	if m.RoadmapCount() != 4 {
		t.Fatalf("RoadmapCount = %d, want 4", m.RoadmapCount())
	}
	if m.RoadmapUndatedCount() != 1 {
		t.Errorf("RoadmapUndatedCount = %d, want 1", m.RoadmapUndatedCount())
	}
	if sel := m.SelectedCard(); sel == nil || sel.Title != "Design spec" {
		t.Fatalf("initial SelectedCard = %v, want Design spec", sel)
	}
	// The undated card is ordered last, so three downs lands on it.
	m.MoveDown()
	m.MoveDown()
	m.MoveDown()
	if sel := m.SelectedCard(); sel == nil || sel.Title != "Someday idea" {
		t.Errorf("after three downs SelectedCard = %v, want Someday idea", sel)
	}
	start, end := m.RoadmapWindow()
	if start != "2026-01" || end != "2026-05" {
		t.Errorf("RoadmapWindow = %q..%q, want 2026-01..2026-05", start, end)
	}
}

func TestRoadmapNarrowWidthDoesNotPanic(t *testing.T) {
	m := New()
	m.SetSize(12, 6)
	m.SetLayout("ROADMAP_LAYOUT")
	m.SetRoadmap(roadmapCards(), "Start", "Target")
	if out := m.View(); out == "" {
		t.Error("narrow roadmap View() returned an empty string")
	}
}

func TestRoadmapEmptyShowsNotice(t *testing.T) {
	m := New()
	m.SetSize(100, 24)
	m.SetLayout("ROADMAP_LAYOUT")
	m.SetRoadmap(nil, "Start", "Target")
	if out := m.View(); !strings.Contains(out, "No items") {
		t.Errorf("empty roadmap view = %q, want a no-items notice", out)
	}
}

func TestRoadmapSingleMarkerField(t *testing.T) {
	m := New()
	m.SetSize(100, 24)
	m.SetLayout("ROADMAP_LAYOUT")
	cards := []core.Card{
		{ID: "1", Type: "ISSUE", State: "OPEN", Title: "Milestone",
			FieldValues: map[string]string{"Due": "2026-03-01"}},
	}
	// One date field: it is the marker date, no target.
	m.SetRoadmap(cards, "Due", "")
	if m.RoadmapUndatedCount() != 0 {
		t.Errorf("RoadmapUndatedCount = %d, want 0", m.RoadmapUndatedCount())
	}
	if out := m.View(); !strings.Contains(out, "Milestone") {
		t.Errorf("single-marker roadmap view missing title, got %q", out)
	}
}
