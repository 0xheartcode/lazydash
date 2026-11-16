package repostore

import (
	"path/filepath"
	"testing"
)

func TestAddDuplicateLoadRemove(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	if repos, err := Load(); err != nil || len(repos) != 0 {
		t.Fatalf("Load on fresh store = %v, %v, want empty", repos, err)
	}

	a := filepath.Join(dir, "repoA")
	added, stored, err := Add(a)
	if err != nil || !added {
		t.Fatalf("Add = added %v, %q, %v", added, stored, err)
	}
	if stored != a {
		t.Errorf("stored = %q, want normalized %q", stored, a)
	}

	// The registry lives next to the YAML config.
	if p, _ := Path(); p != filepath.Join(dir, "lazydash", "repos.json") {
		t.Errorf("Path = %q, want XDG/lazydash/repos.json", p)
	}

	// A second Add of the same path is a no-op.
	if added, _, err := Add(a); err != nil || added {
		t.Errorf("duplicate Add = added %v, %v, want added=false", added, err)
	}
	if repos, _ := Load(); len(repos) != 1 || repos[0] != a {
		t.Fatalf("Load = %v, want [%s]", repos, a)
	}

	// Remove reports success once, then false.
	if removed, _, err := Remove(a); err != nil || !removed {
		t.Fatalf("Remove = removed %v, %v", removed, err)
	}
	if removed, _, _ := Remove(a); removed {
		t.Error("second Remove should report removed=false")
	}
	if repos, _ := Load(); len(repos) != 0 {
		t.Fatalf("Load after remove = %v, want empty", repos)
	}
}

func TestAddNormalizesRelativePath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	_, stored, err := Add(".")
	if err != nil {
		t.Fatalf("Add(.): %v", err)
	}
	if !filepath.IsAbs(stored) {
		t.Errorf("stored = %q, want an absolute path", stored)
	}
}
