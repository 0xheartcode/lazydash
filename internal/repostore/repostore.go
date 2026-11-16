// Package repostore manages the tool-owned list of local repositories that
// lazydash shows beyond the current directory. It is deliberately separate from
// the user's YAML config: `lazydash add-repo` writes here, never rewriting
// config.yml, so a hand-edited config keeps its comments and formatting. The
// file stores absolute repository paths only, never anything about their code.
package repostore

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type file struct {
	Repos []string `json:"repos"`
}

// Path returns the registry file location,
// $XDG_CONFIG_HOME/lazydash/repos.json (XDG default ~/.config), matching where
// the YAML config lives.
func Path() (string, error) {
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		xdg = filepath.Join(home, ".config")
	}
	return filepath.Join(xdg, "lazydash", "repos.json"), nil
}

// Load returns the registered repository paths, or an empty slice when the
// registry file does not exist yet.
func Load() ([]string, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return f.Repos, nil
}

// Add registers path (normalized to an absolute, cleaned path). It reports
// whether the entry was newly added and the normalized path that was stored.
func Add(path string) (added bool, stored string, err error) {
	norm, err := normalize(path)
	if err != nil {
		return false, "", err
	}
	repos, err := Load()
	if err != nil {
		return false, "", err
	}
	for _, r := range repos {
		if r == norm {
			return false, norm, nil
		}
	}
	repos = append(repos, norm)
	sort.Strings(repos)
	if err := save(repos); err != nil {
		return false, "", err
	}
	return true, norm, nil
}

// Remove unregisters path. It reports whether a matching entry was removed.
func Remove(path string) (removed bool, stored string, err error) {
	norm, err := normalize(path)
	if err != nil {
		return false, "", err
	}
	repos, err := Load()
	if err != nil {
		return false, "", err
	}
	kept := make([]string, 0, len(repos))
	for _, r := range repos {
		if r == norm {
			removed = true
			continue
		}
		kept = append(kept, r)
	}
	if !removed {
		return false, norm, nil
	}
	if err := save(kept); err != nil {
		return false, "", err
	}
	return true, norm, nil
}

// normalize resolves ~ and makes path absolute and clean. An empty path means
// the current directory.
func normalize(path string) (string, error) {
	if path == "" {
		path = "."
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

// save writes the registry atomically (temp file plus rename).
func save(repos []string) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(file{Repos: repos}, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
