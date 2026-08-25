package local

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// runGit runs git in dir with the given extra env and stdin, failing the test
// on error. It returns trimmed combined output (stdout for the plumbing we use).
func runGit(t *testing.T, dir string, env []string, stdin string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// commitTree writes an empty-tree commit with msg, an optional parent, and a
// fixed date — mirroring how git-native-issue records an issue event.
func commitTree(t *testing.T, dir, tree, parent, msg, date string) string {
	t.Helper()
	env := []string{
		"GIT_AUTHOR_DATE=" + date, "GIT_COMMITTER_DATE=" + date,
		"GIT_AUTHOR_NAME=Tester", "GIT_AUTHOR_EMAIL=tester@example.com",
		"GIT_COMMITTER_NAME=Tester", "GIT_COMMITTER_EMAIL=tester@example.com",
	}
	args := []string{"commit-tree", tree}
	if parent != "" {
		args = append(args, "-p", parent)
	}
	return runGit(t, dir, env, msg, args...)
}

// newRepo initialises a throwaway repo and returns its dir plus the empty-tree hash.
func newRepo(t *testing.T) (dir, tree string) {
	t.Helper()
	dir = t.TempDir()
	runGit(t, dir, nil, "", "init", "-q")
	tree = runGit(t, dir, nil, "", "mktree") // empty input -> empty tree object
	return dir, tree
}

func TestListIssuesFoldsChain(t *testing.T) {
	dir, tree := newRepo(t)

	rootMsg := "Fix login crash\n\nUsers hit a crash on special chars.\n\nState: open\nLabels: bug, auth\nPriority: critical\nFormat-Version: 1\n"
	r := commitTree(t, dir, tree, "", rootMsg, "2026-01-01T10:00:00")
	commentMsg := "Reproduced on Firefox\n\nHappens on 120 and 121.\n"
	c := commitTree(t, dir, tree, r, commentMsg, "2026-01-02T10:00:00")
	closeMsg := "Close issue\n\nState: closed\nFixed-By: abc1234\n"
	x := commitTree(t, dir, tree, c, closeMsg, "2026-01-03T10:00:00")

	const id = "a7f3b2c1-1111-2222-3333-444455556666"
	runGit(t, dir, nil, "", "update-ref", issueRefPrefix+id, x)

	if !HasIssues(dir) {
		t.Fatal("HasIssues = false, want true")
	}
	issues, err := ListIssues(dir)
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}
	got := issues[0]

	checks := []struct {
		name, got, want string
	}{
		{"ID", got.ID, id},
		{"Title", got.Title, "Fix login crash"},
		{"State", got.State, "closed"},
		{"Priority", got.Priority, "critical"},
		{"Labels", strings.Join(got.Labels, ","), "bug,auth"},
		{"Body", got.Body, "Users hit a crash on special chars."},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
	if len(got.Comments) != 1 {
		t.Fatalf("got %d comments, want 1 (state change must not be a comment)", len(got.Comments))
	}
	if !strings.Contains(got.Comments[0].Body, "Reproduced on Firefox") {
		t.Errorf("comment body = %q", got.Comments[0].Body)
	}
	if got.CreatedAt == "" || got.UpdatedAt == "" || got.CreatedAt == got.UpdatedAt {
		t.Errorf("Created/Updated = %q / %q, want distinct non-empty", got.CreatedAt, got.UpdatedAt)
	}
}

func TestReadIssueSingleCommitDefaultsOpen(t *testing.T) {
	dir, tree := newRepo(t)
	msg := "Add dark mode\n\nWould be nice to have.\n\nState: open\nPriority: low\n"
	r := commitTree(t, dir, tree, "", msg, "2026-02-01T09:00:00")
	runGit(t, dir, nil, "", "update-ref", issueRefPrefix+"bbbb0000", r)

	issues, err := ListIssues(dir)
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}
	got := issues[0]
	if got.State != "open" {
		t.Errorf("State = %q, want open", got.State)
	}
	if got.Title != "Add dark mode" {
		t.Errorf("Title = %q", got.Title)
	}
	if got.Priority != "low" {
		t.Errorf("Priority = %q, want low", got.Priority)
	}
	if len(got.Comments) != 0 {
		t.Errorf("Comments = %v, want none", got.Comments)
	}
}

func TestHasIssuesFalseWhenNone(t *testing.T) {
	dir, _ := newRepo(t)
	if HasIssues(dir) {
		t.Error("HasIssues = true on a repo with no refs/issues")
	}
	if _, err := RepoRoot(dir); err != nil {
		t.Errorf("RepoRoot: %v", err)
	}
}

func TestParseMessageSplitsDescriptionAndTrailers(t *testing.T) {
	raw := "Subject line\n\nSome description.\nMore prose.\n\nState: open\nLabels: a, b\n"
	desc, tr := parseMessage(raw)
	if desc != "Some description.\nMore prose." {
		t.Errorf("description = %q", desc)
	}
	if tr["State"] != "open" || tr["Labels"] != "a, b" {
		t.Errorf("trailers = %v", tr)
	}
}
