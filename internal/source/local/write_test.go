package local

import (
	"os/exec"
	"sort"
	"strings"
	"testing"

	"github.com/0xheartcode/lazydash/internal/core"
	"github.com/0xheartcode/lazydash/internal/source"
)

// gliSource prepares a temp git repo with gli issue tracking initialised and
// returns a writable Source rooted at it. It skips the test when gli is absent so
// CI (which has no binary) stays green.
func gliSource(t *testing.T) *Source {
	t.Helper()
	if !gliAvailable() {
		t.Skip("gli binary not on PATH")
	}
	dir := t.TempDir()
	runGit(t, dir, nil, "", "init", "-q")
	runGit(t, dir, nil, "", "config", "user.email", "tester@example.com")
	runGit(t, dir, nil, "", "config", "user.name", "Tester")
	cmd := exec.Command("gli", "init")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gli init: %v\n%s", err, out)
	}
	return &Source{root: dir, name: "x", writable: true}
}

func TestGliCreateCommentClose(t *testing.T) {
	s := gliSource(t)

	card, err := s.CreateIssue("", source.Draft{
		Title:     "Offline works",
		Body:      "No network needed.",
		Labels:    []string{"feature"},
		Priority:  "high",
		Assignees: []string{"alice", "bob"},
	})
	if err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	if card.ID == "" {
		t.Fatal("CreateIssue returned empty id")
	}
	if card.Title != "Offline works" || card.State != "OPEN" {
		t.Errorf("card = %+v, want Title/State set", card)
	}
	if err := s.Comment(card, "Confirmed on a plane."); err != nil {
		t.Fatalf("Comment: %v", err)
	}
	if err := s.SetState(card, "closed"); err != nil {
		t.Fatalf("SetState: %v", err)
	}

	got := onlyIssue(t, s)
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
	if !equalSet(got.Assignees, []string{"alice", "bob"}) {
		t.Errorf("Assignees = %v, want alice+bob (create seeds one, the rest are added)", got.Assignees)
	}
	if len(got.Comments) != 1 || !strings.Contains(got.Comments[0].Body, "Confirmed") {
		t.Errorf("Comments = %v", got.Comments)
	}
}

func TestGliSetFieldPriorityAndReopen(t *testing.T) {
	s := gliSource(t)

	card, err := s.CreateIssue("", source.Draft{Title: "P", Priority: "low"})
	if err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	if err := s.SetField(card, "Priority", "critical"); err != nil {
		t.Fatalf("SetField priority: %v", err)
	}
	// A field gli cannot express is a no-op, not an error.
	if err := s.SetField(card, "Milestone", "v1"); err != nil {
		t.Fatalf("SetField milestone should be a no-op, got %v", err)
	}
	if err := s.SetState(card, "closed"); err != nil {
		t.Fatalf("SetState closed: %v", err)
	}
	if err := s.SetField(card, "State", "open"); err != nil {
		t.Fatalf("SetField state open: %v", err)
	}

	got := onlyIssue(t, s)
	if got.Priority != "critical" {
		t.Errorf("Priority = %q, want critical", got.Priority)
	}
	if got.State != "open" {
		t.Errorf("State = %q, want open (reopened last)", got.State)
	}
}

func TestGliSetLabelsAndAssigneesDiff(t *testing.T) {
	s := gliSource(t)

	card, err := s.CreateIssue("", source.Draft{
		Title:     "T",
		Labels:    []string{"bug"},
		Assignees: []string{"alice"},
	})
	if err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	// Reflect the created state back onto the card, as the TUI does from the board.
	card.Labels = []core.Label{{Name: "bug"}}
	card.Assignees = []string{"alice"}

	// Want {bug, ui}: adds ui, keeps bug.
	if err := s.SetLabels(card, []string{"bug", "ui"}); err != nil {
		t.Fatalf("SetLabels: %v", err)
	}
	// Want {carol}: adds carol, removes alice.
	if err := s.SetAssignees(card, []string{"carol"}); err != nil {
		t.Fatalf("SetAssignees: %v", err)
	}

	got := onlyIssue(t, s)
	if !equalSet(got.Labels, []string{"bug", "ui"}) {
		t.Errorf("Labels = %v, want bug+ui", got.Labels)
	}
	if !equalSet(got.Assignees, []string{"carol"}) {
		t.Errorf("Assignees = %v, want [carol]", got.Assignees)
	}

	// A no-op diff (same set, order-independent) issues no edit and no error.
	card.Labels = []core.Label{{Name: "ui"}, {Name: "bug"}}
	if err := s.SetLabels(card, []string{"bug", "ui"}); err != nil {
		t.Fatalf("SetLabels no-op: %v", err)
	}
}

func TestGliCommentMissingIssue(t *testing.T) {
	s := gliSource(t)
	if err := s.Comment(core.Card{ID: "ffffffffffffffffffffffffffffffff"}, "hi"); err == nil {
		t.Error("Comment on a missing issue should error")
	}
}

// onlyIssue reads the single issue in the repo through the gli reader.
func onlyIssue(t *testing.T, s *Source) Issue {
	t.Helper()
	issues, err := ListIssuesGli(s.root)
	if err != nil {
		t.Fatalf("ListIssuesGli: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("issues = %d, want 1", len(issues))
	}
	return issues[0]
}

// equalSet reports whether a and b contain the same values, ignoring order.
func equalSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	ac := append([]string(nil), a...)
	bc := append([]string(nil), b...)
	sort.Strings(ac)
	sort.Strings(bc)
	for i := range ac {
		if ac[i] != bc[i] {
			return false
		}
	}
	return true
}
