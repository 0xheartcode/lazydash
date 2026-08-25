package local

import (
	"strings"
	"testing"

	"github.com/0xheartcode/lazydash/internal/source"
)

func TestCreateCommentCloseRoundTrip(t *testing.T) {
	dir, _ := newRepo(t)
	s := &Source{root: dir, name: "x"}

	card, err := s.CreateIssue("", source.Draft{
		Title:    "Offline works",
		Body:     "No network needed.",
		Labels:   []string{"feature"},
		Priority: "high",
	})
	if err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	if card.ID == "" {
		t.Fatal("CreateIssue returned empty id")
	}
	if err := s.Comment(card.ID, "Confirmed on a plane."); err != nil {
		t.Fatalf("Comment: %v", err)
	}
	if err := s.SetState(card.ID, "closed"); err != nil {
		t.Fatalf("SetState: %v", err)
	}

	issues, err := ListIssues(dir)
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("issues = %d, want 1", len(issues))
	}
	got := issues[0]
	if got.Title != "Offline works" {
		t.Errorf("Title = %q", got.Title)
	}
	if got.State != "closed" {
		t.Errorf("State = %q, want closed", got.State)
	}
	if got.Priority != "high" {
		t.Errorf("Priority = %q, want high", got.Priority)
	}
	if strings.Join(got.Labels, ",") != "feature" {
		t.Errorf("Labels = %v, want [feature]", got.Labels)
	}
	if len(got.Comments) != 1 || !strings.Contains(got.Comments[0].Body, "Confirmed") {
		t.Errorf("Comments = %v", got.Comments)
	}
}

func TestSetFieldPriorityAndReopen(t *testing.T) {
	dir, _ := newRepo(t)
	s := &Source{root: dir, name: "x"}

	card, err := s.CreateIssue("", source.Draft{Title: "P", Priority: "low"})
	if err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	if err := s.SetField(card.ID, "Priority", "critical"); err != nil {
		t.Fatalf("SetField: %v", err)
	}
	if err := s.SetState(card.ID, "closed"); err != nil {
		t.Fatalf("SetState closed: %v", err)
	}
	if err := s.SetState(card.ID, "open"); err != nil {
		t.Fatalf("SetState open: %v", err)
	}

	issues, _ := ListIssues(dir)
	if len(issues) != 1 {
		t.Fatalf("issues = %d, want 1", len(issues))
	}
	if issues[0].Priority != "critical" {
		t.Errorf("Priority = %q, want critical (nearest-to-tip trailer wins)", issues[0].Priority)
	}
	if issues[0].State != "open" {
		t.Errorf("State = %q, want open (reopened last)", issues[0].State)
	}
}

func TestAppendEventUnknownIssue(t *testing.T) {
	dir, _ := newRepo(t)
	s := &Source{root: dir, name: "x"}
	if err := s.Comment("does-not-exist", "hi"); err == nil {
		t.Error("Comment on a missing issue should error")
	}
}
