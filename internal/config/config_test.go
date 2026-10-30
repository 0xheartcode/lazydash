package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile creates a file with the given contents and returns its path.
func writeFile(t *testing.T, dir, name, contents string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
	return path
}

// clearEnv removes the environment variables that influence path resolution so
// that each test starts from a known state.
func clearEnv(t *testing.T) {
	t.Helper()
	t.Setenv("LAZYDASH_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "")
}

func TestLoadReturnsDefaultsWhenPathEmpty(t *testing.T) {
	clearEnv(t)
	// An empty HOME makes UserHomeDir fail, which drives resolvePath to return
	// an empty path and Load to return the built-in defaults.
	t.Setenv("HOME", "")
	t.Chdir(t.TempDir())

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	want := defaults()
	if cfg.Keybindings != want.Keybindings {
		t.Errorf("keybindings = %+v, want %+v", cfg.Keybindings, want.Keybindings)
	}
	if cfg.Defaults.RefreshIntervalMinutes != 0 {
		t.Errorf("RefreshIntervalMinutes = %d, want 0", cfg.Defaults.RefreshIntervalMinutes)
	}
	if len(cfg.Defaults.Orgs) != 0 {
		t.Errorf("Orgs = %v, want empty", cfg.Defaults.Orgs)
	}
}

func TestLoadReturnsDefaultsWhenFileAbsent(t *testing.T) {
	clearEnv(t)
	missing := filepath.Join(t.TempDir(), "does-not-exist.yml")

	cfg, err := Load(missing)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Keybindings.Quit != "q" {
		t.Errorf("Quit = %q, want q", cfg.Keybindings.Quit)
	}
	if cfg.Keybindings.OpenInBrowser != "o" {
		t.Errorf("OpenInBrowser = %q, want o", cfg.Keybindings.OpenInBrowser)
	}
}

func TestLoadParsesFullConfig(t *testing.T) {
	clearEnv(t)
	const yaml = `defaults:
  orgs:
    - acme
    - globex
  refreshIntervalMinutes: 15
  ignoreProjects:
    - secret
  ignoreOrgs:
    - oldorg
  onlyOrgs:
    - acme
  onlyProjects:
    - roadmap
  sources:
    - github
    - local
keybindings:
  openInBrowser: O
  openInGhDash: D
  pager: P
  refresh: R
  help: h
  quit: Q
  prevView: "<"
  nextView: ">"
  create: C
  comment: m
  toggleState: t
  move: V
  labels: l
  assign: A
  undo: z
`
	path := writeFile(t, t.TempDir(), "config.yml", yaml)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	d := cfg.Defaults
	if len(d.Orgs) != 2 || d.Orgs[0] != "acme" || d.Orgs[1] != "globex" {
		t.Errorf("Orgs = %v, want [acme globex]", d.Orgs)
	}
	if d.RefreshIntervalMinutes != 15 {
		t.Errorf("RefreshIntervalMinutes = %d, want 15", d.RefreshIntervalMinutes)
	}
	if len(d.IgnoreProjects) != 1 || d.IgnoreProjects[0] != "secret" {
		t.Errorf("IgnoreProjects = %v, want [secret]", d.IgnoreProjects)
	}
	if len(d.IgnoreOrgs) != 1 || d.IgnoreOrgs[0] != "oldorg" {
		t.Errorf("IgnoreOrgs = %v, want [oldorg]", d.IgnoreOrgs)
	}
	if len(d.OnlyOrgs) != 1 || d.OnlyOrgs[0] != "acme" {
		t.Errorf("OnlyOrgs = %v, want [acme]", d.OnlyOrgs)
	}
	if len(d.OnlyProjects) != 1 || d.OnlyProjects[0] != "roadmap" {
		t.Errorf("OnlyProjects = %v, want [roadmap]", d.OnlyProjects)
	}
	if len(d.Sources) != 2 || d.Sources[0] != "github" || d.Sources[1] != "local" {
		t.Errorf("Sources = %v, want [github local]", d.Sources)
	}

	wantKB := Keybindings{
		OpenInBrowser: "O",
		OpenInGhDash:  "D",
		Pager:         "P",
		Refresh:       "R",
		Help:          "h",
		Quit:          "Q",
		PrevView:      "<",
		NextView:      ">",
		Create:        "C",
		Comment:       "m",
		ToggleState:   "t",
		Move:          "V",
		Labels:        "l",
		Assign:        "A",
		Undo:          "z",
	}
	if cfg.Keybindings != wantKB {
		t.Errorf("keybindings = %+v, want %+v", cfg.Keybindings, wantKB)
	}
}

func TestLoadFillsMissingKeybindings(t *testing.T) {
	clearEnv(t)
	// Only a couple of bindings are set; the rest must fall back to defaults.
	const yaml = `keybindings:
  quit: Q
  refresh: R
`
	path := writeFile(t, t.TempDir(), "partial.yml", yaml)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Keybindings.Quit != "Q" {
		t.Errorf("Quit = %q, want Q", cfg.Keybindings.Quit)
	}
	if cfg.Keybindings.Refresh != "R" {
		t.Errorf("Refresh = %q, want R", cfg.Keybindings.Refresh)
	}
	// Unset bindings keep their defaults.
	def := defaults().Keybindings
	if cfg.Keybindings.OpenInBrowser != def.OpenInBrowser {
		t.Errorf("OpenInBrowser = %q, want default %q", cfg.Keybindings.OpenInBrowser, def.OpenInBrowser)
	}
	if cfg.Keybindings.Help != def.Help {
		t.Errorf("Help = %q, want default %q", cfg.Keybindings.Help, def.Help)
	}
	if cfg.Keybindings.Undo != def.Undo {
		t.Errorf("Undo = %q, want default %q", cfg.Keybindings.Undo, def.Undo)
	}
}

func TestLoadResolvesPathSources(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, dir string) (flagPath string)
	}{
		{
			name: "flag path takes priority",
			setup: func(t *testing.T, dir string) string {
				clearEnv(t)
				return writeFile(t, dir, "flag.yml", "defaults:\n  refreshIntervalMinutes: 3\n")
			},
		},
		{
			name: "env var when no flag",
			setup: func(t *testing.T, dir string) string {
				clearEnv(t)
				path := writeFile(t, dir, "env.yml", "defaults:\n  refreshIntervalMinutes: 3\n")
				t.Setenv("LAZYDASH_CONFIG", path)
				return ""
			},
		},
		{
			name: "dot file in working directory",
			setup: func(t *testing.T, dir string) string {
				clearEnv(t)
				writeFile(t, dir, ".lazydash.yml", "defaults:\n  refreshIntervalMinutes: 3\n")
				t.Chdir(dir)
				return ""
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			flagPath := tc.setup(t, t.TempDir())
			cfg, err := Load(flagPath)
			if err != nil {
				t.Fatalf("Load returned error: %v", err)
			}
			if cfg.Defaults.RefreshIntervalMinutes != 3 {
				t.Errorf("RefreshIntervalMinutes = %d, want 3", cfg.Defaults.RefreshIntervalMinutes)
			}
		})
	}
}

func TestLoadRejectsInvalidYAML(t *testing.T) {
	clearEnv(t)
	path := writeFile(t, t.TempDir(), "bad.yml", "defaults: [this is not: valid\n")

	if _, err := Load(path); err == nil {
		t.Error("Load should return an error for malformed YAML")
	}
}
