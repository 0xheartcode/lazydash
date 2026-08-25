// Package cache persists the last successfully-loaded board per project so a
// board stays browsable when its source is unreachable — a GitHub project you
// synced earlier can still be read offline. It is a best-effort read cache:
// failures to save or load are non-fatal and simply mean no offline copy.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/0xheartcode/lazydash/internal/core"
)

// Dir returns the directory boards are cached in, honouring XDG_CACHE_HOME.
func Dir() string {
	if x := os.Getenv("XDG_CACHE_HOME"); x != "" {
		return filepath.Join(x, "lazydash", "boards")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cache", "lazydash", "boards")
}

// Save writes a board to the cache under key. It is best-effort: any error
// (no home dir, read-only fs) is returned but callers may ignore it.
func Save(key string, bd *core.BoardData) error {
	dir := Dir()
	if dir == "" {
		return os.ErrNotExist
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(bd)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath(dir, key), data, 0o644)
}

// Load returns the cached board for key, and whether one was found.
func Load(key string) (*core.BoardData, bool) {
	dir := Dir()
	if dir == "" {
		return nil, false
	}
	data, err := os.ReadFile(filePath(dir, key))
	if err != nil {
		return nil, false
	}
	var bd core.BoardData
	if err := json.Unmarshal(data, &bd); err != nil {
		return nil, false
	}
	return &bd, true
}

// filePath maps a cache key to a stable filename via its SHA-256 digest.
func filePath(dir, key string) string {
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(dir, hex.EncodeToString(sum[:])+".json")
}
