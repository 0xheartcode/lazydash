package local

import (
	"testing"

	"github.com/0xheartcode/lazydash/internal/core"
)

func TestBuildBoardSynthesizesFieldsAndGroups(t *testing.T) {
	issues := []Issue{
		{ID: "1", Title: "A", State: "open", Priority: "high", Labels: []string{"bug"}, Assignee: "x@y.z"},
		{ID: "2", Title: "B", State: "closed", Priority: "low"},
	}
	bd := buildBoard(issues)

	if len(bd.Fields) != 2 {
		t.Fatalf("fields = %d, want 2 (State, Priority)", len(bd.Fields))
	}
	if len(bd.Views) != 3 {
		t.Fatalf("views = %d, want 3", len(bd.Views))
	}

	// Grouping by State must yield an open column with A and a closed column with B.
	cols := core.GroupByField(bd, "State")
	byName := map[string][]string{}
	for _, c := range cols {
		for _, card := range c.Cards {
			byName[c.Name] = append(byName[c.Name], card.Title)
		}
	}
	if got := byName["open"]; len(got) != 1 || got[0] != "A" {
		t.Errorf("open column = %v, want [A]", got)
	}
	if got := byName["closed"]; len(got) != 1 || got[0] != "B" {
		t.Errorf("closed column = %v, want [B]", got)
	}
}

func TestSourceNewAndGetBoard(t *testing.T) {
	dir, tree := newRepo(t)
	msg := "Wire up offline mode\n\nMake it work without a network.\n\nState: open\nPriority: high\n"
	r := commitTree(t, dir, tree, "", msg, "2026-03-01T12:00:00")
	runGit(t, dir, nil, "", "update-ref", issueRefPrefix+"cccc1111", r)

	s, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if s == nil {
		t.Fatal("New returned nil for a repo that has issues")
	}
	if !s.Caps().Offline {
		t.Error("local source should advertise Offline")
	}
	projects, err := s.ListProjects()
	if err != nil || len(projects) != 1 {
		t.Fatalf("ListProjects = %v, %v", projects, err)
	}
	bd, err := s.GetBoard(projects[0].ID)
	if err != nil {
		t.Fatalf("GetBoard: %v", err)
	}
	if len(bd.Items) != 1 || bd.Items[0].Title != "Wire up offline mode" {
		t.Fatalf("items = %+v", bd.Items)
	}
}

func TestNewReturnsNilWithoutIssues(t *testing.T) {
	dir, _ := newRepo(t)
	s, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if s != nil {
		t.Error("New should return nil for a repo with no local issues")
	}
}

func TestIssueToItemMapping(t *testing.T) {
	item := issueToItem(Issue{
		ID: "1", Title: "A", State: "open", Priority: "high",
		Labels: []string{"bug", "auth"}, Assignee: "x@y.z", Milestone: "v1",
	})

	if item.Type != "ISSUE" {
		t.Errorf("Type = %q, want ISSUE", item.Type)
	}
	if item.State != "OPEN" {
		t.Errorf("State = %q, want OPEN (drives the icon)", item.State)
	}
	if item.FieldValues["State"] != "open" {
		t.Errorf("FieldValues[State] = %q, want open (must match option name)", item.FieldValues["State"])
	}
	if item.FieldValues["Priority"] != "high" || item.FieldValues["Milestone"] != "v1" {
		t.Errorf("field values = %v", item.FieldValues)
	}
	if len(item.Assignees) != 1 || item.Assignees[0] != "x@y.z" {
		t.Errorf("Assignees = %v", item.Assignees)
	}
	if len(item.Labels) != 2 || item.Labels[0].Color == "" {
		t.Errorf("Labels = %v, want 2 colored labels", item.Labels)
	}
}
