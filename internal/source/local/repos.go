package local

import (
	"os"
	"path/filepath"
	"strings"
)

// RepoRef identifies one local repository in the multi-repo set.
type RepoRef struct {
	Root string // working-tree root (absolute)
	Name string // display name (basename of root)
}

// Discover builds the ordered, deduplicated set of local repositories to show.
// The current directory comes first when it is a git repo that actually has
// issues, which preserves the single-repo default. After it come the extra paths
// (from config and the add-repo registry) that resolve to a git repo. An extra
// path is kept even with no issues yet, since listing it is a deliberate choice;
// a path that is not a git repo, or a duplicate root, is skipped. Only paths are
// ever read here, never anything about a repo's code.
func Discover(cwd string, extra []string) []RepoRef {
	var refs []RepoRef
	seen := map[string]bool{}

	add := func(root string) {
		if root == "" || seen[root] {
			return
		}
		seen[root] = true
		refs = append(refs, RepoRef{Root: root, Name: filepath.Base(root)})
	}

	if root, err := RepoRoot(cwd); err == nil && HasIssues(root) {
		add(root)
	}
	for _, p := range extra {
		p = expandHome(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		root, err := RepoRoot(p)
		if err != nil {
			continue
		}
		add(root)
	}
	return refs
}

// expandHome resolves a leading ~ to the user's home directory.
func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}
