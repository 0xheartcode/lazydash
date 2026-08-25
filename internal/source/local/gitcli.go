// Package local is the offline backend: it reads (and later writes)
// git-native-issue issues stored as commit chains under refs/issues/ in the
// repository lazydash is launched in. Reads use only git plumbing, so viewing
// issues never depends on the `git issue` binary being installed.
package local

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// git runs a git command in dir and returns raw stdout. Stderr is folded into
// the error so callers get an actionable message. dir must be inside the repo.
func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}
