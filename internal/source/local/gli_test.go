package local

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/0xheartcode/lazydash/internal/core"
)

// cannedGliList is a real `gli ls --all --format json` payload (gli 0.1.2),
// captured from the binary. It carries an active issue with several labels,
// several assignees and comments, plus an archived, closed issue with a null
// legacy assignee, so the parser is exercised without needing the binary in CI.
const cannedGliList = `[
  {
    "uuid": "01a19d3894357d4196dabea88646802d",
    "handle": "#1",
    "number": 1,
    "nonce": "t-1",
    "short": "01a19d38943",
    "title": "Fix login crash",
    "description": "Users hit a crash on special chars.",
    "state": "open",
    "reason": null,
    "fixed_by": null,
    "assignee": "alice",
    "assignees": ["alice", "bob"],
    "priority": "critical",
    "labels": ["auth", "bug", "triaged"],
    "fields": [],
    "files": [],
    "archived": false,
    "creator": "t",
    "comment_count": 2,
    "comments": [
      {"id": "01a19d38945e75909a02ffe6a053e4de", "actor": "t", "author": "T <t@e.com>", "text": "Reproduced on Firefox", "edited": false, "hidden": false},
      {"id": "01a19d38946b7792ac3493f6978c5916", "actor": "t", "author": "T <t@e.com>", "text": "Also on Safari", "edited": false, "hidden": false}
    ]
  },
  {
    "uuid": "01a19d3894727781a33b65cdab77968e",
    "handle": "#2",
    "number": 2,
    "nonce": "t-2",
    "short": "01a19d38947",
    "title": "Add dark mode",
    "description": "Nice to have",
    "state": "closed",
    "reason": null,
    "fixed_by": null,
    "assignee": null,
    "assignees": [],
    "priority": "low",
    "labels": [],
    "fields": [],
    "files": [],
    "archived": true,
    "creator": "t",
    "comment_count": 0,
    "comments": []
  }
]`

func TestParseGliListFoldsFields(t *testing.T) {
	issues, err := parseGliList([]byte(cannedGliList))
	if err != nil {
		t.Fatalf("parseGliList: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("issues = %d, want 2", len(issues))
	}

	a := issues[0]
	if a.ID != "01a19d3894357d4196dabea88646802d" {
		t.Errorf("ID = %q (want the uuid, not a short id)", a.ID)
	}
	if a.Number != 1 {
		t.Errorf("Number = %d, want 1", a.Number)
	}
	if a.Title != "Fix login crash" {
		t.Errorf("Title = %q (must be the JSON title, not a commit subject)", a.Title)
	}
	if a.Body != "Users hit a crash on special chars." {
		t.Errorf("Body = %q", a.Body)
	}
	if a.State != "open" {
		t.Errorf("State = %q, want open", a.State)
	}
	if a.Priority != "critical" {
		t.Errorf("Priority = %q, want critical", a.Priority)
	}
	if strings.Join(a.Labels, ",") != "auth,bug,triaged" {
		t.Errorf("Labels = %v, want the full OR-set auth,bug,triaged", a.Labels)
	}
	if strings.Join(a.Assignees, ",") != "alice,bob" {
		t.Errorf("Assignees = %v, want the full OR-set alice,bob", a.Assignees)
	}
	if a.Archived {
		t.Error("issue #1 Archived = true, want false")
	}
	if a.CommentCount != 2 {
		t.Errorf("CommentCount = %d, want 2", a.CommentCount)
	}
	if len(a.Comments) != 2 {
		t.Fatalf("Comments = %d, want 2", len(a.Comments))
	}
	if a.Comments[0].Body != "Reproduced on Firefox" || a.Comments[1].Body != "Also on Safari" {
		t.Errorf("comment bodies = %q, %q", a.Comments[0].Body, a.Comments[1].Body)
	}
	if a.Comments[0].Author != "T <t@e.com>" {
		t.Errorf("comment author = %q", a.Comments[0].Author)
	}

	b := issues[1]
	if b.State != "closed" {
		t.Errorf("issue #2 State = %q, want closed", b.State)
	}
	if !b.Archived {
		t.Error("issue #2 Archived = false, want true")
	}
	if len(b.Assignees) != 0 {
		t.Errorf("issue #2 Assignees = %v, want none (null legacy assignee, empty OR-set)", b.Assignees)
	}
	if len(b.Labels) != 0 {
		t.Errorf("issue #2 Labels = %v, want none", b.Labels)
	}
}

func TestGliEndToEndGetBoard(t *testing.T) {
	if !gliAvailable() {
		t.Skip("gli binary not on PATH")
	}
	dir := t.TempDir()
	runGit(t, dir, nil, "", "init", "-q")
	runGit(t, dir, nil, "", "config", "user.email", "tester@example.com")
	runGit(t, dir, nil, "", "config", "user.name", "Tester")
	mustGli(t, dir, "init")

	id := strings.TrimSpace(mustGli(t, dir, "create", "Fix login crash",
		"-d", "Users hit a crash.", "-l", "bug", "-p", "critical", "-a", "alice", "-q"))
	mustGli(t, dir, "edit", id, "--add-label", "triaged", "--add-assignee", "bob")
	mustGli(t, dir, "comment", id, "Reproduced on Firefox")
	id2 := strings.TrimSpace(mustGli(t, dir, "create", "Add dark mode", "-p", "low", "-q"))
	mustGli(t, dir, "close", id2)

	s, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if s == nil {
		t.Fatal("New returned nil for a repo that has issues")
	}
	if !s.Caps().Create {
		t.Error("gli present: Caps().Create should be true")
	}

	bd, err := s.GetBoard("local")
	if err != nil {
		t.Fatalf("GetBoard: %v", err)
	}
	if len(bd.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(bd.Items))
	}

	byTitle := map[string]core.RawItem{}
	for _, it := range bd.Items {
		byTitle[it.Title] = it
	}
	fix, ok := byTitle["Fix login crash"]
	if !ok {
		t.Fatalf("titles = %v, want the created issue (title from JSON, not a commit subject)", keys(byTitle))
	}
	if _, ok := byTitle["Add dark mode"]; !ok {
		t.Fatalf("titles = %v, want the second issue too", keys(byTitle))
	}
	if fix.State != "OPEN" {
		t.Errorf("State = %q, want OPEN", fix.State)
	}
	var labels []string
	for _, l := range fix.Labels {
		labels = append(labels, l.Name)
	}
	if !containsAll(labels, "bug", "triaged") {
		t.Errorf("labels = %v, want full OR-set including bug and triaged", labels)
	}
	if !containsAll(fix.Assignees, "alice", "bob") {
		t.Errorf("assignees = %v, want full OR-set alice and bob", fix.Assignees)
	}
}

func keys(m map[string]core.RawItem) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

func mustGli(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("gli", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gli %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func containsAll(have []string, want ...string) bool {
	set := map[string]bool{}
	for _, h := range have {
		set[h] = true
	}
	for _, w := range want {
		if !set[w] {
			return false
		}
	}
	return true
}
