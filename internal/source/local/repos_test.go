package local

import "testing"

func TestDiscoverCurrentFirstDedupAndSkip(t *testing.T) {
	// Current dir is a repo WITH an issue.
	cwd, tree := newRepo(t)
	r := commitTree(t, cwd, tree, "", "Title\n\nState: open\n", "2026-01-01T10:00:00")
	runGit(t, cwd, nil, "", "update-ref", issueRefPrefix+"aaaa1111", r)

	empty, _ := newRepo(t) // a repo with no issues
	notRepo := t.TempDir() // not a git repo at all

	// cwd is also passed as an extra to prove it is deduped, and listed last.
	refs := Discover(cwd, []string{empty, notRepo, cwd})

	if len(refs) != 2 {
		t.Fatalf("refs = %+v, want 2 (cwd + empty; non-repo skipped, dup folded)", refs)
	}
	cwdRoot, _ := RepoRoot(cwd)
	if refs[0].Root != cwdRoot {
		t.Errorf("refs[0].Root = %q, want current repo first %q", refs[0].Root, cwdRoot)
	}
	emptyRoot, _ := RepoRoot(empty)
	if refs[1].Root != emptyRoot {
		t.Errorf("refs[1].Root = %q, want %q", refs[1].Root, emptyRoot)
	}
}

func TestDiscoverCwdWithoutIssuesNeedsExplicitListing(t *testing.T) {
	cwd, _ := newRepo(t) // repo, but no issues

	if refs := Discover(cwd, nil); len(refs) != 0 {
		t.Fatalf("refs = %+v, want 0 (cwd has no issues and nothing was listed)", refs)
	}
	// Naming it explicitly still includes it, even with no issues yet.
	if refs := Discover(cwd, []string{cwd}); len(refs) != 1 {
		t.Fatalf("refs = %+v, want 1 (explicit extra includes a repo without issues)", refs)
	}
}

func TestMultiRepoListProjectsAndRouting(t *testing.T) {
	cwd, tree := newRepo(t)
	r := commitTree(t, cwd, tree, "", "A\n\nState: open\n", "2026-01-01T10:00:00")
	runGit(t, cwd, nil, "", "update-ref", issueRefPrefix+"aaaa1111", r)
	other, _ := newRepo(t)

	s, err := NewMulti(cwd, []string{other})
	if err != nil || s == nil {
		t.Fatalf("NewMulti = %v, %v", s, err)
	}
	projects, err := s.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("projects = %d, want 2 (one per repo)", len(projects))
	}
	for _, p := range projects {
		if s.rootFor(p.ID) != p.ID {
			t.Errorf("rootFor(%q) did not round-trip to the repo root", p.ID)
		}
	}
	if s.rootFor("/does/not/exist") != "" {
		t.Error("rootFor on an unknown id should be empty")
	}
}
